package serve

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Cleanup kills stale processes from previous runs and removes corrupt
// SQLite WAL/SHM files. Called automatically by `start` and `serve`.
// extraPorts are additional ports to check (e.g. the dashboard port).
func Cleanup(webhookPort int, dbPath string, extraPorts ...int) {
	killStalePort(webhookPort)
	for _, p := range extraPorts {
		killStalePort(p)
	}
	killStaleLlamaServers()
	cleanStaleSQLite(dbPath)
}

func killStalePort(port int) {
	if port <= 0 {
		return
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return
	}
	conn.Close()

	slog.Info("cleanup: port in use from previous run, killing", "port", port)
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil || len(out) == 0 {
		return
	}
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		cmd := exec.Command("kill", "-9", pidStr)
		if err := cmd.Run(); err != nil {
			slog.Debug("cleanup: kill failed", "pid", pidStr, "err", err)
		} else {
			slog.Info("cleanup: killed stale process", "pid", pidStr, "port", port)
		}
	}

	for i := 0; i < 10; i++ {
		time.Sleep(200 * time.Millisecond)
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
		if err != nil {
			return
		}
		conn.Close()
	}
	slog.Warn("cleanup: port still in use after kill attempts", "port", port)
}

func killStaleLlamaServers() {
	out, err := exec.Command("pgrep", "-f", "llama-server").Output()
	if err != nil || len(out) == 0 {
		return
	}
	slog.Info("cleanup: killing stale llama-server processes")
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		cmd := exec.Command("kill", "-9", pidStr)
		if err := cmd.Run(); err != nil {
			slog.Debug("cleanup: kill llama-server failed", "pid", pidStr, "err", err)
		} else {
			slog.Info("cleanup: killed stale llama-server", "pid", pidStr)
		}
	}
}

func cleanStaleSQLite(dbPath string) {
	if dbPath == "" {
		return
	}
	for _, suffix := range []string{"-shm", "-wal"} {
		path := dbPath + suffix
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				slog.Debug("cleanup: remove stale file failed", "path", path, "err", err)
			} else {
				slog.Info("cleanup: removed stale file", "path", path)
			}
		}
	}
}
