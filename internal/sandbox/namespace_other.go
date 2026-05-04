//go:build !linux

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/features/F-007-guardrails-and-safety.md
*/
package sandbox

import "os/exec"

// applyNamespaces is a no-op on non-Linux platforms.
func applyNamespaces(_ *exec.Cmd) bool {
	return false
}
