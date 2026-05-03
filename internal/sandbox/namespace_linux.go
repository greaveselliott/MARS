//go:build linux

package sandbox

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const linuxNamespaceFlags = syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET

var (
	linuxNamespaceProbeOnce      sync.Once
	linuxNamespaceProbeAvailable bool
)

// applyNamespaces sets Linux namespace clone flags for mount/PID/net isolation.
// Restricted hosts can deny clone flags at process start, so availability is
// probed once before flags are applied.
func applyNamespaces(cmd *exec.Cmd) bool {
	if !linuxNamespacesAvailable() {
		return false
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Cloneflags = uintptr(linuxNamespaceFlags)
	return true
}

func linuxNamespacesAvailable() bool {
	if linuxNamespacesDisabledByEnv() {
		return false
	}

	linuxNamespaceProbeOnce.Do(func() {
		linuxNamespaceProbeAvailable = probeLinuxNamespaces()
	})
	return linuxNamespaceProbeAvailable
}

func linuxNamespacesDisabledByEnv() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("MARS_HARNESS_DISABLE_LINUX_NAMESPACES")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func probeLinuxNamespaces() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "true")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:    true,
		Cloneflags: uintptr(linuxNamespaceFlags),
	}

	if err := cmd.Start(); err != nil {
		slog.Warn("sandbox: Linux namespace probe failed, using ulimit only", "error", err)
		return false
	}
	if err := cmd.Wait(); err != nil {
		slog.Warn("sandbox: Linux namespace probe exited unsuccessfully, using ulimit only", "error", err)
		return false
	}
	return true
}
