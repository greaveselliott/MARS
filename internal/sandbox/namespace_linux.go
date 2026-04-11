//go:build linux

package sandbox

import (
	"os/exec"
	"syscall"
)

// applyNamespaces sets Linux namespace clone flags for mount/PID/net isolation.
// Requires CAP_SYS_ADMIN; caller should handle exec failures gracefully.
func applyNamespaces(cmd *exec.Cmd) bool {
	const nsFlags = syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Cloneflags = uintptr(nsFlags)
	return true
}
