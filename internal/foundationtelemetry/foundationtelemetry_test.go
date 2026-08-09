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

	"github.com/greaveselliott/mars/pkg/testutil"
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

func TestSQLiteStoreLegacyFixture(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "legacy-intake.db")
	testutil.WriteSQLiteFixture(t, path, `
CREATE TABLE foundation_telemetry_reports (
  id TEXT PRIMARY KEY,
  received_at INTEGER NOT NULL,
  schema_version INTEGER NOT NULL,
  report_key TEXT NOT NULL,
  payload_hash TEXT NOT NULL UNIQUE,
  payload_json TEXT NOT NULL
);
`, `
CREATE TABLE foundation_telemetry_patterns (
  signature TEXT PRIMARY KEY,
  first_seen INTEGER NOT NULL,
  last_seen INTEGER NOT NULL,
  report_count INTEGER NOT NULL,
  install_window_count INTEGER NOT NULL,
  harness_versions TEXT NOT NULL,
  category TEXT NOT NULL,
  target TEXT NOT NULL,
  severity TEXT NOT NULL
);
`, `
CREATE TABLE foundation_telemetry_pattern_reports (
  signature TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  report_key TEXT NOT NULL,
  PRIMARY KEY(signature, payload_hash)
);
`, `
CREATE TABLE foundation_telemetry_pattern_report_keys (
  signature TEXT NOT NULL,
  report_key TEXT NOT NULL,
  PRIMARY KEY(signature, report_key)
);
`, `
INSERT INTO foundation_telemetry_patterns(signature, first_seen, last_seen, report_count, install_window_count, harness_versions, category, target, severity)
VALUES('sig-legacy', 1, 1779148800, 2, 1, '["0.41.0"]', 'tool_timeout', 'tools', 'medium');
`)

	store, err := OpenSQLiteStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	testutil.AssertSQLiteIndexes(t, store.db, "idx_foundation_telemetry_patterns_last_seen")

	patterns, err := store.PatternsSince(context.Background(), time.Unix(1, 0).UTC())
	require.NoError(t, err)
	require.Len(t, patterns, 1)
	require.Equal(t, "sig-legacy", patterns[0].Signature)
	require.Equal(t, []string{"0.41.0"}, patterns[0].HarnessVersions)
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
		req.Host = "127.0.0.1:9092"
		req.Header.Set("Content-Type", "application/json")
		Handler(store, "127.0.0.1:9092").ServeHTTP(rec, req)
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
	req.Host = "127.0.0.1:9092"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	Handler(store, "127.0.0.1:9092").ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	patterns, err := store.PatternsSince(context.Background(), time.Now().UTC().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Len(t, patterns, 1)
}
