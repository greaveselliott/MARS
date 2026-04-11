//go:build !linux

package sandbox

import "os/exec"

// applyNamespaces is a no-op on non-Linux platforms.
func applyNamespaces(_ *exec.Cmd) bool {
	return false
}
