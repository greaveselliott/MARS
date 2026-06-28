//go:build darwin

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package hardware

import (
	"log/slog"

	"golang.org/x/sys/unix"
)

func memTotalDarwinSyscall() int {
	bytesTotal, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		slog.Debug("sysctlbyname hw.memsize failed", "err", err)
		return 0
	}
	return int(bytesTotal / (1024 * 1024))
}
