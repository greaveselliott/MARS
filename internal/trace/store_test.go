package trace

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
