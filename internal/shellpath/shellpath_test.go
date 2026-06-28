/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-002-zero-config-shell-path.md
*/
package shellpath

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureFishWritesConfDFile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	installDir := filepath.Join(home, "go", "bin")

	result, err := Ensure(Config{
		InstallDir: installDir,
		ShellPath:  "/opt/homebrew/bin/fish",
		HomeDir:    home,
		EnvPath:    "",
	})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected profile change, got %+v", result)
	}
	path := filepath.Join(home, ".config", "fish", "conf.d", "mars.fish")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fish profile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "set -gx PATH") || !strings.Contains(text, installDir) {
		t.Fatalf("unexpected fish profile: %s", text)
	}
}

func TestEnsureZshIsIdempotent(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	installDir := filepath.Join(home, "bin")
	cfg := Config{
		InstallDir: installDir,
		ShellPath:  "/bin/zsh",
		HomeDir:    home,
		EnvPath:    "",
	}

	first, err := Ensure(cfg)
	if err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	second, err := Ensure(cfg)
	if err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if !first.Changed {
		t.Fatalf("first run should change profile")
	}
	if second.Changed {
		t.Fatalf("second run should be idempotent: %+v", second)
	}
	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("read zshrc: %v", err)
	}
	if strings.Count(string(data), startMarker) != 1 {
		t.Fatalf("expected one managed block, got:\n%s", string(data))
	}
}

func TestEnsureZshManagedBlockMovesInstallDirToFront(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	installDir := filepath.Join(home, "go", "bin")
	otherDir := filepath.Join(home, "other", "bin")
	cfg := Config{
		InstallDir: installDir,
		ShellPath:  "/bin/zsh",
		HomeDir:    home,
		EnvPath:    otherDir + string(os.PathListSeparator) + installDir + string(os.PathListSeparator) + "/usr/bin",
	}

	result, err := Ensure(cfg)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected profile change, got %+v", result)
	}
	profile := filepath.Join(home, ".zshrc")
	cmd := exec.Command("/bin/sh", "-c", ". "+posixQuote(profile)+"; printf '%s' \"$PATH\"")
	cmd.Env = []string{"PATH=" + cfg.EnvPath}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("source generated profile: %v", err)
	}
	got := string(out)
	parts := filepath.SplitList(got)
	if len(parts) == 0 || parts[0] != installDir {
		t.Fatalf("install dir should be first after sourcing profile, got %q", got)
	}
	if strings.Count(string(os.PathListSeparator)+got+string(os.PathListSeparator), string(os.PathListSeparator)+installDir+string(os.PathListSeparator)) != 1 {
		t.Fatalf("install dir should appear once after sourcing profile, got %q", got)
	}
}

func TestEnsureReplacesLegacyManagedBlock(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	installDir := filepath.Join(home, "bin")
	profile := filepath.Join(home, ".zshrc")
	legacy := legacyStartMarker + "\nexport PATH=/old:$PATH\n" + legacyEndMarker + "\n"
	if err := os.WriteFile(profile, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy profile: %v", err)
	}

	result, err := Ensure(Config{
		InstallDir: installDir,
		ShellPath:  "/bin/zsh",
		HomeDir:    home,
	})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !result.Changed {
		t.Fatalf("legacy marker should be replaced with the canonical block: %+v", result)
	}

	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if strings.Contains(string(data), legacyStartMarker) {
		t.Fatalf("legacy marker should not remain after migration: %s", string(data))
	}
	if strings.Count(string(data), startMarker) != 1 {
		t.Fatalf("expected one canonical marker after migration: %s", string(data))
	}
}

func TestEnsureFishRemovesLegacyConfDFile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	confDir := filepath.Join(home, ".config", "fish", "conf.d")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir conf dir: %v", err)
	}
	legacyPath := filepath.Join(confDir, "mars-harness.fish")
	if err := os.WriteFile(legacyPath, []byte("# legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy fish file: %v", err)
	}

	_, err := Ensure(Config{
		InstallDir: filepath.Join(home, "go", "bin"),
		ShellPath:  "/opt/homebrew/bin/fish",
		HomeDir:    home,
	})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy fish file should be removed, stat err=%v", err)
	}
}

func TestEnsureBashUsesExistingBashProfileOnMac(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bash_profile"), []byte("# user profile\n"), 0o644); err != nil {
		t.Fatalf("write bash profile: %v", err)
	}
	result, err := Ensure(Config{
		InstallDir: filepath.Join(home, "go", "bin"),
		ShellPath:  "/bin/bash",
		HomeDir:    home,
	})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if runtimeGOOS() == "darwin" && filepath.Base(result.ProfilePath) != ".bash_profile" {
		t.Fatalf("expected .bash_profile on darwin, got %s", result.ProfilePath)
	}
}

func TestEvaluateUnsupportedShellDoesNotWrite(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	result, err := Ensure(Config{
		InstallDir: filepath.Join(home, "bin"),
		ShellPath:  "/usr/local/bin/nu",
		HomeDir:    home,
	})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !result.UnsupportedShell {
		t.Fatalf("expected unsupported shell, got %+v", result)
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("read home: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("unsupported shell should not write files, got %d entries", len(entries))
	}
}

func TestEvaluateAlreadyInPathStillConfiguresProfile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	installDir := filepath.Join(home, "bin")
	result, err := Evaluate(Config{
		InstallDir: installDir,
		ShellPath:  "/bin/zsh",
		HomeDir:    home,
		EnvPath:    installDir,
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !result.AlreadyInPATH {
		t.Fatalf("expected already in PATH")
	}
	if result.ProfileAlreadyConfigured {
		t.Fatalf("profile should still need configuration")
	}
}

func runtimeGOOS() string {
	return runtime.GOOS
}
