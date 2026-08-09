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
	"log/slog"
	"os"

	_ "modernc.org/sqlite"
)

// CleanupScopedLifecycle recovers SQLite sidecars without killing listeners or
// processes. Runtime-owned tool children and inference servers are stopped by
// their owning job/session and Router lifecycle hooks.
func CleanupScopedLifecycle(dbPath string) {
	cleanStaleSQLite(dbPath)
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
