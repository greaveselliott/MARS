package sandbox

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRun_setsWorkingDirectory(t *testing.T) {
	cfg := Config{
		WorkDir: "/tmp",
	}

	cmd, err := Run(context.Background(), cfg, "echo", "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if cmd.Dir != "/tmp" {
		t.Errorf("expected Dir=/tmp, got %q", cmd.Dir)
	}
}

func TestRun_timeoutEnforcement(t *testing.T) {
	cfg := Config{
		WorkDir: "/tmp",
		Timeout: 200 * time.Millisecond,
	}

	cmd, err := Run(context.Background(), cfg, "sleep", "10")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	err = cmd.Wait()
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestRun_ulimitWrapper_darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test")
	}

	cfg := Config{
		WorkDir:   "/tmp",
		MaxCPUSec: 60,
		MaxMemMB:  512,
		MaxFiles:  256,
	}

	cmd, err := Run(context.Background(), cfg, "echo", "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if cmd.Path == "" {
		t.Fatal("expected non-empty command path")
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "ulimit") {
		t.Errorf("expected ulimit in args, got %q", args)
	}
}

func TestRun_noLimitsSkipsWrapper(t *testing.T) {
	cfg := Config{
		WorkDir: "/tmp",
	}

	cmd, err := Run(context.Background(), cfg, "echo", "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.HasSuffix(cmd.Path, "echo") && !strings.Contains(cmd.Path, "echo") {
		// Without limits, the command should be "echo" directly, not bash wrapper.
		// On some systems cmd.Path is resolved to full path.
		args := strings.Join(cmd.Args, " ")
		if strings.Contains(args, "ulimit") {
			t.Errorf("expected no ulimit wrapper without limits, got %q", args)
		}
	}
}

func TestRun_processGroupIsolation(t *testing.T) {
	cfg := Config{
		WorkDir: "/tmp",
	}

	cmd, err := Run(context.Background(), cfg, "echo", "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be set")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Error("expected Setpgid to be true for process group isolation")
	}
}

func TestUlimitWrapper_allLimits(t *testing.T) {
	cfg := Config{
		MaxCPUSec: 30,
		MaxMemMB:  256,
		MaxFiles:  128,
	}

	wrapper, args := ulimitWrapper(cfg, "my-cmd", "--flag", "value")
	if wrapper != "bash" {
		t.Errorf("expected bash wrapper, got %q", wrapper)
	}
	script := strings.Join(args, " ")
	if !strings.Contains(script, "ulimit -t 30") {
		t.Errorf("expected CPU limit, got %q", script)
	}
	if !strings.Contains(script, "ulimit -v 262144") {
		t.Errorf("expected memory limit (256*1024=262144), got %q", script)
	}
	if !strings.Contains(script, "ulimit -n 128") {
		t.Errorf("expected file limit, got %q", script)
	}
	if !strings.Contains(script, "exec") {
		t.Errorf("expected exec in script, got %q", script)
	}
}

func TestUlimitWrapper_noLimits(t *testing.T) {
	cfg := Config{}

	wrapper, args := ulimitWrapper(cfg, "echo", "hello")
	if wrapper != "echo" {
		t.Errorf("expected passthrough command, got %q", wrapper)
	}
	if len(args) != 1 || args[0] != "hello" {
		t.Errorf("expected passthrough args, got %v", args)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"", "''"},
		{"it's", "'it'\"'\"'s'"},
		{"a b", "'a b'"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
