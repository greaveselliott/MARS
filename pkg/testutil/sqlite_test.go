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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSQLiteFixtureHelpers(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "fixture.db")
	WriteSQLiteFixture(t, path, `
CREATE TABLE example (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL
);
`, `
CREATE INDEX idx_example_name ON example(name);
`, `
INSERT INTO example(id, name) VALUES('ex-1', 'fixture');
`)

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	AssertSQLiteColumns(t, db, "example", "id", "name")
	AssertSQLiteIndexes(t, db, "idx_example_name")
}
