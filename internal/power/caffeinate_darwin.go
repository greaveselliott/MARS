//go:build darwin

package power

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// PreventSleep spawns caffeinate to keep the machine awake for the
// lifetime of this process. Returns a cancel function that kills it.
// Flags: -i (prevent idle sleep) -s (prevent system sleep, covers
// clamshell mode) -w PID (auto-exit when our process dies).
func PreventSleep() (cancel func(), err error) {
	pid := os.Getpid()
	cmd := exec.Command("caffeinate", "-i", "-s", "-w", fmt.Sprintf("%d", pid))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("power: failed to start caffeinate: %w", err)
	}
	slog.Info("power: sleep prevention active (caffeinate)", "caffeinate_pid", cmd.Process.Pid)
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			slog.Info("power: sleep prevention released")
		}
	}, nil
}
