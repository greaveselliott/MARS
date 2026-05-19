/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// WriteSQLiteFixture creates a SQLite database from explicit legacy DDL/data.
//
// Store migration tests should create the historical base tables and rows here,
// then reopen through the production store constructor. Production init order
// must stay: create base tables, backfill missing columns, then create indexes,
// triggers, or views that depend on those columns.
func WriteSQLiteFixture(t *testing.T, path string, statements ...string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		_, err := db.Exec(statement)
		require.NoError(t, err, "sqlite fixture statement failed: %s", statement)
	}
	require.NoError(t, db.Close())
}

// AssertSQLiteColumns verifies that a table exposes the expected columns.
func AssertSQLiteColumns(t *testing.T, db *sql.DB, table string, columns ...string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	require.NoError(t, err)
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		seen[name] = true
	}
	require.NoError(t, rows.Err())
	for _, column := range columns {
		require.Truef(t, seen[column], "expected %s.%s column", table, column)
	}
}

// AssertSQLiteIndexes verifies that named SQLite indexes exist.
func AssertSQLiteIndexes(t *testing.T, db *sql.DB, indexes ...string) {
	t.Helper()
	for _, index := range indexes {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&count)
		require.NoError(t, err)
		require.Equalf(t, 1, count, "expected sqlite index %s", index)
	}
}
