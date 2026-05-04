//go:build !darwin

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-006-queue-and-orchestration.md
- docs/product-specs/product-surface.md
*/
package power

import "log/slog"

// PreventSleep is a no-op on non-macOS platforms where idle sleep
// is typically not a concern for headless/server machines.
func PreventSleep() (cancel func(), err error) {
	slog.Debug("power: sleep prevention not needed on this platform")
	return func() {}, nil
}
