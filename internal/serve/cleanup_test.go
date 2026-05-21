/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCleanStaleSQLitePreservesRecoverableSidecars(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("create table events (id integer primary key, message text)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("insert into events (message) values ('seeded before retry')"); err != nil {
		t.Fatalf("insert row: %v", err)
	}
	requireFileExists(t, dbPath+"-wal")
	requireFileExists(t, dbPath+"-shm")

	cleanStaleSQLite(dbPath)

	requireFileExists(t, dbPath+"-wal")
	requireFileExists(t, dbPath+"-shm")

	var count int
	if err := db.QueryRow("select count(*) from events").Scan(&count); err != nil {
		t.Fatalf("query after cleanup: %v", err)
	}
	if count != 1 {
		t.Fatalf("count after cleanup = %d, want 1", count)
	}
}

func requireFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
