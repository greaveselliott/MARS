/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package foundationtelemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testReport(t *testing.T) AnonymousReport {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	return AnonymousReport{
		SchemaVersion:     SchemaVersion,
		ReportKey:         "rk-test",
		HarnessVersion:    "0.30.1",
		OS:                "darwin",
		Arch:              "arm64",
		HardwareTier:      "unified-memory",
		OrchestrationMode: "dispatch",
		WindowStart:       now.Add(-24 * time.Hour),
		WindowEnd:         now,
		Patterns: []AnonymousPattern{
			{
				Signature:    "sig-test",
				RoleDomain:   "orchestrator",
				RoleMode:     "routing",
				Category:     "max_turns",
				Target:       "skill",
				Severity:     "medium",
				Count:        3,
				DistinctJobs: 3,
			},
		},
	}
}

func TestDecodeBatchRejectsUnknownRawFields(t *testing.T) {
	report := testReport(t)
	data, err := json.Marshal(map[string]any{
		"reports":      []AnonymousReport{report},
		"raw_message":  "secret path <repo>",
		"trace_output": "not allowed",
	})
	require.NoError(t, err)

	_, err = DecodeBatch(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field")
}

func TestValidateReportRejectsRawPathLeakageInAllowlistedFields(t *testing.T) {
	report := testReport(t)
	report.OS = "<repo>"

	err := ValidateReport(report)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsafe characters")
}

func TestSQLiteStoreSavesReportAndAggregatesPattern(t *testing.T) {
	store, err := OpenSQLiteStore(filepath.Join(t.TempDir(), "intake.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	report := testReport(t)
	require.NoError(t, store.SaveReport(context.Background(), report))
	for _, pattern := range AggregatesFromReport(report) {
		require.NoError(t, store.UpsertPattern(context.Background(), pattern))
	}

	patterns, err := store.PatternsSince(context.Background(), time.Now().UTC().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Len(t, patterns, 1)
	require.Equal(t, "sig-test", patterns[0].Signature)
	require.Equal(t, 1, patterns[0].ReportCount)
	require.Equal(t, 1, patterns[0].InstallWindowCount)
	require.Equal(t, []string{"0.30.1"}, patterns[0].HarnessVersions)
}

func TestSQLiteStoreDedupesRepeatedPayloadsAndCountsDistinctReportKeys(t *testing.T) {
	store, err := OpenSQLiteStore(filepath.Join(t.TempDir(), "intake.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	postReport := func(report AnonymousReport) {
		t.Helper()
		data, err := json.Marshal(ReportBatch{Reports: []AnonymousReport{report}})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, ReportsPath, bytes.NewReader(data))
		rec := httptest.NewRecorder()
		Handler(store).ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	report := testReport(t)
	postReport(report)
	postReport(report)

	patterns, err := store.PatternsSince(context.Background(), time.Now().UTC().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Len(t, patterns, 1)
	require.Equal(t, 1, patterns[0].ReportCount)
	require.Equal(t, 1, patterns[0].InstallWindowCount)

	second := report
	second.ReportKey = "rk-second"
	postReport(second)

	patterns, err = store.PatternsSince(context.Background(), time.Now().UTC().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Len(t, patterns, 1)
	require.Equal(t, 2, patterns[0].ReportCount)
	require.Equal(t, 2, patterns[0].InstallWindowCount)
}

func TestHandlerAcceptsValidBatch(t *testing.T) {
	store, err := OpenSQLiteStore(filepath.Join(t.TempDir(), "intake.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	batch := ReportBatch{Reports: []AnonymousReport{testReport(t)}}
	data, err := json.Marshal(batch)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, ReportsPath, bytes.NewReader(data))
	rec := httptest.NewRecorder()

	Handler(store).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	patterns, err := store.PatternsSince(context.Background(), time.Now().UTC().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Len(t, patterns, 1)
}
