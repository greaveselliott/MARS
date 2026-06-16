/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Cleanup kills stale processes from previous runs and lets SQLite recover
// any sidecar WAL/SHM files. Called automatically by `start` and `serve`.
// extraPorts are additional ports to check (e.g. the dashboard port).
func Cleanup(webhookPort int, dbPath string, extraPorts ...int) {
	killStalePort(webhookPort)
	for _, p := range extraPorts {
		killStalePort(p)
	}
	killStaleLlamaServers()
	cleanStaleSQLite(dbPath)
}

// CleanupScopedLifecycle recovers per-repo SQLite sidecars for `start` without
// killing shared control ports or llama-server processes owned by parallel runs.
func CleanupScopedLifecycle(dbPath string) {
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
	if _, err := os.Stat(dbPath); err != nil {
		if !os.IsNotExist(err) {
			slog.Debug("cleanup: inspect sqlite db failed", "path", dbPath, "err", err)
		}
		return
	}

	hasSidecar := false
	for _, suffix := range []string{"-shm", "-wal"} {
		path := dbPath + suffix
		if _, err := os.Stat(path); err == nil {
			hasSidecar = true
		} else if err != nil && !os.IsNotExist(err) {
			slog.Debug("cleanup: inspect sqlite sidecar failed", "path", path, "err", err)
		}
	}
	if !hasSidecar {
		return
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		slog.Warn("cleanup: sqlite sidecar recovery open failed", "path", dbPath, "err", err)
		return
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA wal_checkpoint(PASSIVE)"); err != nil {
		slog.Warn("cleanup: sqlite sidecar recovery checkpoint failed", "path", dbPath, "err", err)
		return
	}
	slog.Info("cleanup: sqlite sidecar recovery checked", "path", dbPath)
}
