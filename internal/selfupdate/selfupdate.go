package selfupdate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/shellpath"
)

const (
	DefaultPackage = "github.com/greaveselliott/mars-harness/cmd/mars-harness"
	DefaultVersion = "latest"
	DefaultBinary  = "mars-harness"
)

// Config controls an installed-tool update run.
type Config struct {
	Version    string
	InstallDir string
	BinaryName string
	DryRun     bool
}

// Plan is the resolved update action.
type Plan struct {
	Package    string           `json:"package"`
	Version    string           `json:"version"`
	InstallDir string           `json:"install_dir"`
	BinaryPath string           `json:"binary_path"`
	Command    []string         `json:"command"`
	ShellPath  shellpath.Result `json:"shell_path"`
	DryRun     bool             `json:"dry_run"`
}

// ResolvePlan computes the go-install update command without executing it.
func ResolvePlan(cfg Config) (Plan, error) {
	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = DefaultVersion
	}
	version = strings.TrimPrefix(version, "@")

	binary := strings.TrimSpace(cfg.BinaryName)
	if binary == "" {
		binary = DefaultBinary
	}

	installDir := strings.TrimSpace(cfg.InstallDir)
	if installDir == "" {
		exe, err := os.Executable()
		if err != nil {
			return Plan{}, fmt.Errorf("update tool: resolve current executable: %w", err)
		}
		if eval, err := filepath.EvalSymlinks(exe); err == nil {
			exe = eval
		}
		installDir = filepath.Dir(exe)
	}
	absInstallDir, err := filepath.Abs(installDir)
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: resolve install dir %q: %w", installDir, err)
	}

	pkg := DefaultPackage + "@" + version
	return Plan{
		Package:    DefaultPackage,
		Version:    version,
		InstallDir: absInstallDir,
		BinaryPath: filepath.Join(absInstallDir, binary),
		Command:    []string{"go", "install", pkg},
		DryRun:     cfg.DryRun,
	}, nil
}

// Run reinstalls mars-harness into the resolved install directory using go install.
func Run(ctx context.Context, cfg Config) (Plan, error) {
	plan, err := ResolvePlan(cfg)
	if err != nil {
		return Plan{}, err
	}
	if cfg.DryRun {
		pathResult, pathErr := shellpath.Ensure(shellpath.Config{InstallDir: plan.InstallDir, DryRun: true})
		if pathErr == nil {
			plan.ShellPath = pathResult
		}
		return plan, nil
	}

	if err := os.MkdirAll(plan.InstallDir, 0o755); err != nil {
		return Plan{}, fmt.Errorf("update tool: create install dir %s: %w", plan.InstallDir, err)
	}

	cmd := exec.CommandContext(ctx, plan.Command[0], plan.Command[1:]...)
	cmd.Env = append(os.Environ(), "GOBIN="+plan.InstallDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: %s failed with GOBIN=%s: %w\n%s\nIf this is a permission error, rerun with an install dir you own or install from the source checkout with `make install`",
			strings.Join(plan.Command, " "), plan.InstallDir, err, strings.TrimSpace(string(output)))
	}
	pathResult, err := shellpath.Ensure(shellpath.Config{InstallDir: plan.InstallDir})
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: installed %s but could not configure shell PATH: %w", plan.BinaryPath, err)
	}
	plan.ShellPath = pathResult
	return plan, nil
}
