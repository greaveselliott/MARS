/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-002-zero-config-shell-path.md
*/
package shellpath

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	startMarker       = "# >>> mars PATH >>>"
	endMarker         = "# <<< mars PATH <<<"
	legacyStartMarker = "# >>> mars-harness PATH >>>"
	legacyEndMarker   = "# <<< mars-harness PATH <<<"
)

// Config controls shell PATH setup.
type Config struct {
	InstallDir string
	ShellPath  string
	HomeDir    string
	EnvPath    string
	DryRun     bool
}

// Result describes shell PATH state and any profile update.
type Result struct {
	InstallDir               string `json:"install_dir"`
	Shell                    string `json:"shell"`
	ProfilePath              string `json:"profile_path,omitempty"`
	AlreadyInPATH            bool   `json:"already_in_path"`
	ProfileAlreadyConfigured bool   `json:"profile_already_configured"`
	Changed                  bool   `json:"changed"`
	DryRun                   bool   `json:"dry_run"`
	UnsupportedShell         bool   `json:"unsupported_shell"`
	Message                  string `json:"message"`
	ReloadHint               string `json:"reload_hint,omitempty"`
}

// ResolveInstallDir finds the directory that should contain mars.
func ResolveInstallDir(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Abs(explicit)
	}
	if exe, err := os.Executable(); err == nil {
		if eval, evalErr := filepath.EvalSymlinks(exe); evalErr == nil {
			exe = eval
		}
		dir := filepath.Dir(exe)
		if !looksTemporary(dir) {
			return filepath.Abs(dir)
		}
	}
	if goBin := goBinDir(); goBin != "" {
		return filepath.Abs(goBin)
	}
	return "", fmt.Errorf("shell path: cannot resolve install dir — pass --install-dir or install Go so `go env GOPATH` works")
}

// Evaluate checks whether shell PATH setup is already present without writing.
func Evaluate(cfg Config) (Result, error) {
	return evaluate(cfg)
}

// Ensure writes an idempotent shell profile snippet when needed.
func Ensure(cfg Config) (Result, error) {
	result, err := evaluate(cfg)
	if err != nil {
		return Result{}, err
	}
	if result.UnsupportedShell || result.ProfileAlreadyConfigured {
		return result, nil
	}
	if result.DryRun {
		result.Message = "would configure shell PATH"
		return result, nil
	}
	content := profileContent(result.Shell, result.InstallDir)
	if content == "" {
		return result, nil
	}
	if err := os.MkdirAll(filepath.Dir(result.ProfilePath), 0o755); err != nil {
		return Result{}, fmt.Errorf("shell path: create profile directory %s: %w", filepath.Dir(result.ProfilePath), err)
	}
	if result.Shell == "fish" && filepath.Base(result.ProfilePath) == "mars.fish" {
		legacyFish := filepath.Join(filepath.Dir(result.ProfilePath), "mars-harness.fish")
		if _, err := os.Stat(legacyFish); err == nil {
			_ = os.Remove(legacyFish)
		}
		if err := os.WriteFile(result.ProfilePath, []byte(content), 0o644); err != nil {
			return Result{}, fmt.Errorf("shell path: write %s: %w", result.ProfilePath, err)
		}
	} else if err := upsertManagedBlock(result.ProfilePath, content); err != nil {
		return Result{}, err
	}
	result.Changed = true
	result.Message = "configured shell PATH"
	return result, nil
}

func evaluate(cfg Config) (Result, error) {
	installDir, err := ResolveInstallDir(cfg.InstallDir)
	if err != nil {
		return Result{}, err
	}
	installDir = filepath.Clean(installDir)
	shellName := detectShell(cfg.ShellPath)
	result := Result{
		InstallDir:    installDir,
		Shell:         shellName,
		AlreadyInPATH: inPath(installDir, envPath(cfg.EnvPath)),
		DryRun:        cfg.DryRun,
	}
	profilePath, reloadHint, ok, err := profileForShell(shellName, cfg.HomeDir)
	if err != nil {
		return Result{}, err
	}
	if !ok {
		result.UnsupportedShell = true
		result.Message = fmt.Sprintf("unsupported shell %q; add %s to PATH manually", shellName, installDir)
		return result, nil
	}
	result.ProfilePath = profilePath
	result.ReloadHint = reloadHint
	configured, err := profileHasPath(profilePath, installDir)
	if err != nil {
		return Result{}, err
	}
	result.ProfileAlreadyConfigured = configured
	if configured {
		result.Message = "shell PATH already configured"
	} else if result.AlreadyInPATH {
		result.Message = "current PATH already contains install dir; shell profile will be configured for future terminals"
	} else {
		result.Message = "shell PATH needs configuration"
	}
	return result, nil
}

func detectShell(shellPath string) string {
	shellPath = strings.TrimSpace(shellPath)
	if shellPath == "" {
		shellPath = os.Getenv("SHELL")
	}
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			return "powershell"
		}
		return "sh"
	}
	name := filepath.Base(shellPath)
	name = strings.TrimPrefix(name, "-")
	return strings.ToLower(name)
}

func profileForShell(shellName, homeDir string) (path, reloadHint string, ok bool, err error) {
	home := strings.TrimSpace(homeDir)
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return "", "", false, fmt.Errorf("shell path: cannot determine home directory: %w", err)
		}
	}
	switch shellName {
	case "fish":
		path = filepath.Join(home, ".config", "fish", "conf.d", "mars.fish")
		return path, "open a new terminal or run `exec fish`", true, nil
	case "zsh":
		path = filepath.Join(home, ".zshrc")
		return path, "open a new terminal or run `source ~/.zshrc`", true, nil
	case "bash":
		path = bashProfilePath(home)
		return path, fmt.Sprintf("open a new terminal or run `source %s`", shellDisplayPath(path, home)), true, nil
	case "sh", "dash", "ksh", "mksh":
		path = filepath.Join(home, ".profile")
		return path, "open a new terminal or run `. ~/.profile`", true, nil
	case "tcsh", "csh":
		path = filepath.Join(home, ".cshrc")
		return path, "open a new terminal or run `source ~/.cshrc`", true, nil
	default:
		return "", "", false, nil
	}
}

func bashProfilePath(home string) string {
	if runtime.GOOS == "darwin" {
		if fileExists(filepath.Join(home, ".bash_profile")) {
			return filepath.Join(home, ".bash_profile")
		}
	}
	return filepath.Join(home, ".bashrc")
}

func profileHasPath(profilePath, installDir string) (bool, error) {
	data, err := os.ReadFile(profilePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("shell path: read %s: %w", profilePath, err)
	}
	text := string(data)
	if strings.Contains(text, legacyStartMarker) {
		return false, nil
	}
	return strings.Contains(text, startMarker) && strings.Contains(text, installDir), nil
}

func profileContent(shellName, installDir string) string {
	switch shellName {
	case "fish":
		return "# Managed by mars; safe to remove this file.\n" +
			fmt.Sprintf("if not contains -- %s $PATH\n", fishQuote(installDir)) +
			fmt.Sprintf("    set -gx PATH %s $PATH\n", fishQuote(installDir)) +
			"end\n"
	case "tcsh", "csh":
		return startMarker + "\n" +
			"# Managed by mars; safe to remove this block.\n" +
			fmt.Sprintf("if ( \"$PATH\" !~ *%s* ) setenv PATH %s:$PATH\n", cshQuote(installDir), cshQuote(installDir)) +
			endMarker + "\n"
	default:
		return startMarker + "\n" +
			"# Managed by mars; safe to remove this block.\n" +
			fmt.Sprintf("_mars_install_dir=%s\n", posixQuote(installDir)) +
			"_mars_new_path=\"$_mars_install_dir\"\n" +
			"_mars_old_ifs=$IFS\n" +
			"IFS=:\n" +
			"for _mars_path_entry in $PATH; do\n" +
			"  if [ \"$_mars_path_entry\" != \"$_mars_install_dir\" ] && [ -n \"$_mars_path_entry\" ]; then\n" +
			"    _mars_new_path=\"$_mars_new_path:$_mars_path_entry\"\n" +
			"  fi\n" +
			"done\n" +
			"IFS=$_mars_old_ifs\n" +
			"export PATH=\"$_mars_new_path\"\n" +
			"unset _mars_install_dir _mars_new_path _mars_old_ifs _mars_path_entry\n" +
			endMarker + "\n"
	}
}

func upsertManagedBlock(profilePath, block string) error {
	data, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("shell path: read %s: %w", profilePath, err)
	}
	text := string(data)
	text = replaceLegacyManagedBlock(text)
	if start := strings.Index(text, startMarker); start >= 0 {
		if end := strings.Index(text[start:], endMarker); end >= 0 {
			end += start + len(endMarker)
			text = strings.TrimRight(text[:start], "\n") + "\n\n" + strings.TrimRight(block, "\n") + "\n" + strings.TrimLeft(text[end:], "\n")
			return os.WriteFile(profilePath, []byte(text), 0o644)
		}
	}
	if strings.TrimSpace(text) == "" {
		text = strings.TrimRight(block, "\n") + "\n"
	} else {
		text = strings.TrimRight(text, "\n") + "\n\n" + strings.TrimRight(block, "\n") + "\n"
	}
	if err := os.WriteFile(profilePath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("shell path: write %s: %w", profilePath, err)
	}
	return nil
}

func replaceLegacyManagedBlock(text string) string {
	start := strings.Index(text, legacyStartMarker)
	if start < 0 {
		return text
	}
	end := strings.Index(text[start:], legacyEndMarker)
	if end < 0 {
		return text
	}
	end += start + len(legacyEndMarker)
	return strings.TrimRight(text[:start], "\n") + "\n\n" + strings.TrimLeft(text[end:], "\n")
}

func inPath(dir, pathValue string) bool {
	dir = filepath.Clean(dir)
	for _, entry := range filepath.SplitList(pathValue) {
		if filepath.Clean(entry) == dir {
			return true
		}
	}
	return false
}

func envPath(value string) string {
	if value != "" {
		return value
	}
	return os.Getenv("PATH")
}

func looksTemporary(dir string) bool {
	dir = filepath.Clean(dir)
	tmp := filepath.Clean(os.TempDir())
	return strings.HasPrefix(dir, tmp) || strings.Contains(dir, string(filepath.Separator)+"go-build")
}

func goBinDir() string {
	if out, err := exec.Command("go", "env", "GOBIN").Output(); err == nil {
		if gobin := strings.TrimSpace(string(out)); gobin != "" {
			return gobin
		}
	}
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		if gopath := strings.TrimSpace(string(out)); gopath != "" {
			parts := filepath.SplitList(gopath)
			if len(parts) > 0 {
				return filepath.Join(parts[0], "bin")
			}
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func shellDisplayPath(path, home string) string {
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + filepath.ToSlash(rel)
	}
	return path
}

func posixQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func fishQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func cshQuote(value string) string {
	return "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
}
