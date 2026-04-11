package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Config controls sandbox restrictions.
type Config struct {
	WorkDir   string        // forced working directory
	MaxCPUSec int           // CPU time limit (ulimit -t)
	MaxMemMB  int           // memory limit (ulimit -v)
	MaxFiles  int           // max open files (ulimit -n)
	Timeout   time.Duration // wall time limit
}

// Run executes a command within sandbox restrictions.
// On Linux, uses namespace isolation if available, falls back to ulimit.
// On macOS, uses cwd restriction + ulimit only (sandbox-exec deprecated).
func Run(ctx context.Context, cfg Config, name string, args ...string) (*exec.Cmd, error) {
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		_ = cancel // caller must manage context; we attach deadline only
	}

	if runtime.GOOS == "linux" {
		return runLinux(ctx, cfg, name, args...)
	}
	return runDarwin(ctx, cfg, name, args...)
}

func runLinux(ctx context.Context, cfg Config, name string, args ...string) (*exec.Cmd, error) {
	wrapper, wrapperArgs := ulimitWrapper(cfg, name, args...)
	cmd := exec.CommandContext(ctx, wrapper, wrapperArgs...)
	cmd.Dir = cfg.WorkDir

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if applyNamespaces(cmd) {
		slog.Info("sandbox: Linux namespace isolation enabled")
	} else {
		slog.Warn("sandbox: namespace isolation unavailable, using ulimit only")
	}

	return cmd, nil
}

func runDarwin(ctx context.Context, cfg Config, name string, args ...string) (*exec.Cmd, error) {
	wrapper, wrapperArgs := ulimitWrapper(cfg, name, args...)
	cmd := exec.CommandContext(ctx, wrapper, wrapperArgs...)
	cmd.Dir = cfg.WorkDir

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	slog.Info("sandbox: macOS — using cwd restriction + ulimit")
	return cmd, nil
}

// ulimitWrapper wraps a command in bash with ulimit restrictions.
func ulimitWrapper(cfg Config, name string, args ...string) (string, []string) {
	var parts []string

	if cfg.MaxCPUSec > 0 {
		parts = append(parts, fmt.Sprintf("ulimit -t %d", cfg.MaxCPUSec))
	}
	if cfg.MaxMemMB > 0 {
		kbytes := cfg.MaxMemMB * 1024
		parts = append(parts, fmt.Sprintf("ulimit -v %d", kbytes))
	}
	if cfg.MaxFiles > 0 {
		parts = append(parts, fmt.Sprintf("ulimit -n %d", cfg.MaxFiles))
	}

	if len(parts) == 0 {
		return name, args
	}

	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, name)
	for _, a := range args {
		quoted = append(quoted, shellQuote(a))
	}

	parts = append(parts, "exec "+strings.Join(quoted, " "))
	script := strings.Join(parts, " && ")

	return "bash", []string{"-c", script}
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
