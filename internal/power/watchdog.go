/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-006-queue-and-orchestration.md
- docs/product-specs/product-surface.md
*/
package power

import (
	"context"
	"log/slog"
	"time"
)

// WakeCallback is invoked when the watchdog detects a wake-from-sleep event.
// The gap parameter is how long the machine appeared to be asleep.
type WakeCallback func(gap time.Duration)

const (
	watchdogTick      = 5 * time.Second
	sleepGapThreshold = 30 * time.Second
)

// StartWatchdog starts a background goroutine that detects system sleep by
// monitoring for unexpected time gaps between ticks. When a gap exceeding
// the threshold is detected, onWake is called. Returns when ctx is cancelled.
func StartWatchdog(ctx context.Context, onWake WakeCallback) {
	go func() {
		ticker := time.NewTicker(watchdogTick)
		defer ticker.Stop()

		lastTick := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				elapsed := now.Sub(lastTick)
				lastTick = now

				if elapsed > sleepGapThreshold {
					gap := elapsed - watchdogTick
					slog.Warn("power: sleep/wake detected — machine was suspended",
						"gap", gap.Round(time.Second),
					)
					if onWake != nil {
						onWake(gap)
					}
				}
			}
		}
	}()
}
