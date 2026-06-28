/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package trace

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/greaveselliott/mars/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpenStoreLegacyFixture(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "legacy-trace.db")
	testutil.WriteSQLiteFixture(t, path, `
CREATE TABLE traces (
  trace_id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL,
  turns_jsonl TEXT NOT NULL,
  summary_json TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
`, `
INSERT INTO traces(trace_id, job_id, turns_jsonl, summary_json, created_at)
VALUES('trace-legacy', 'job-legacy', '{"type":"turn"}', '{"ok":true}', 1);
`)

	st, err := OpenStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	testutil.AssertSQLiteIndexes(t, st.db, "idx_traces_job_id")

	got, err := st.GetLatestByJobID(context.Background(), "job-legacy")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "trace-legacy", got.TraceID)
}

func TestStore_roundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "db.sqlite")
	uri := "file:" + path + "?mode=rwc"
	st, err := OpenStore(uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	jobID := "job-42"
	traceID := "tr-99"
	jsonl := "{\"type\":\"header\"}\n{\"type\":\"turn\"}\n"
	summary := `{"outcome":"ok"}`

	require.NoError(t, st.Save(ctx, jobID, traceID, jsonl, summary))

	got, err := st.GetLatestByJobID(ctx, jobID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, traceID, got.TraceID)
	require.Equal(t, jobID, got.JobID)
	require.Equal(t, jsonl, got.TurnsJSONL)
	require.Equal(t, summary, got.SummaryJSON)
}

func TestStoreListSinceReturnsNewestFirst(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "db.sqlite")
	uri := "file:" + path + "?mode=rwc"
	st, err := OpenStore(uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	require.NoError(t, st.Save(ctx, "job-old", "trace-old", "old-jsonl", `{"old":true}`))
	_, err = st.db.ExecContext(ctx, "UPDATE traces SET created_at = ? WHERE trace_id = ?", time.Now().Add(-time.Hour).Unix(), "trace-old")
	require.NoError(t, err)
	since := time.Now().Add(-time.Minute)
	require.NoError(t, st.Save(ctx, "job-new", "trace-new", "new-jsonl", `{"new":true}`))

	records, err := st.ListSince(ctx, since)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "trace-new", records[0].TraceID)
	require.Equal(t, "job-new", records[0].JobID)
	require.Equal(t, "new-jsonl", records[0].TurnsJSONL)
}

func TestStore_missingJobReturnsNil(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "db2.sqlite")
	uri := "file:" + path + "?mode=rwc"
	st, err := OpenStore(uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	got, err := st.GetLatestByJobID(context.Background(), "does-not-exist")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestNilStoreReturnsActionableErrors(t *testing.T) {
	t.Parallel()
	var st *Store
	require.NoError(t, st.Close())
	require.ErrorContains(t, st.Save(context.Background(), "job", "trace", "", ""), "store is nil")
	_, err := st.GetLatestByJobID(context.Background(), "job")
	require.ErrorContains(t, err, "store is nil")
	_, err = st.ListSince(context.Background(), time.Now())
	require.ErrorContains(t, err, "store is nil")
}
