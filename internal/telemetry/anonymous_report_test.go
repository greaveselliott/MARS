/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildAnonymousReportUsesAllowlistedAggregates(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(filepath.Join(t.TempDir(), "telemetry.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	start := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		require.NoError(t, store.Save(Event{
			ID:        fmt.Sprintf("evt-%d", i),
			Timestamp: start.Add(time.Duration(i) * time.Minute),
			JobID:     fmt.Sprintf("job-%d", i),
			RepoID:    "repo-1",
			Role:      "orchestrator",
			Category:  CategoryMaxTurns,
			Message:   "raw path <repo> and command output must not leave",
		}))
	}

	report, err := store.BuildAnonymousReport(AnonymousReportOptions{
		RepoID:            "repo-1",
		ReportKeySeed:     "seed",
		HarnessVersion:    "0.30.1",
		OS:                "darwin",
		Arch:              "arm64",
		HardwareTier:      "unified-memory",
		OrchestrationMode: "dispatch",
		WindowStart:       start.Add(-time.Minute),
		WindowEnd:         start.Add(time.Hour),
		Roles: map[string]RoleMetadata{
			"orchestrator": {Domain: "orchestrator", Mode: "routing"},
		},
	})
	require.NoError(t, err)
	require.Len(t, report.Patterns, 1)
	require.Equal(t, "orchestrator", report.Patterns[0].RoleDomain)
	require.Equal(t, "routing", report.Patterns[0].RoleMode)
	require.Equal(t, "max_turns", report.Patterns[0].Category)
	require.Equal(t, 3, report.Patterns[0].DistinctJobs)

	payload := fmt.Sprintf("%+v", report)
	require.NotContains(t, payload, "<repo>")
	require.NotContains(t, payload, "command output")
}

func TestBuildAnonymousReportSkipsTargetOwnedCategories(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(filepath.Join(t.TempDir(), "telemetry.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	now := time.Now().UTC()
	require.NoError(t, store.Save(Event{
		ID:        "evt-human",
		Timestamp: now,
		JobID:     "job-human",
		RepoID:    "repo-1",
		Role:      "engineer",
		Category:  CategoryHumanFollowup,
		Message:   "human follow-up commit",
	}))

	report, err := store.BuildAnonymousReport(AnonymousReportOptions{
		RepoID:            "repo-1",
		ReportKeySeed:     "seed",
		HarnessVersion:    "0.30.1",
		OS:                "darwin",
		Arch:              "arm64",
		HardwareTier:      "unified-memory",
		OrchestrationMode: "dispatch",
		WindowStart:       now.Add(-time.Hour),
		WindowEnd:         now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.Empty(t, report.Patterns)
}

func TestAnonymousReportOutboxDedupeAndRetryState(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(filepath.Join(t.TempDir(), "telemetry.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	now := time.Now().UTC()
	report, err := store.BuildAnonymousReport(AnonymousReportOptions{
		ReportKeySeed:     "seed",
		HarnessVersion:    "0.30.1",
		OS:                "darwin",
		Arch:              "arm64",
		HardwareTier:      "unified-memory",
		OrchestrationMode: "dispatch",
		WindowStart:       now.Add(-time.Hour),
		WindowEnd:         now,
	})
	require.NoError(t, err)

	first, err := store.EnqueueAnonymousReport(context.Background(), report)
	require.NoError(t, err)
	second, err := store.EnqueueAnonymousReport(context.Background(), report)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	pending, err := store.PendingReports(context.Background(), time.Now().UTC(), 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	require.NoError(t, store.MarkReportFailed(context.Background(), first.ID, time.Now().UTC().Add(time.Hour), fmt.Errorf("network down")))
	stats, err := store.OutboxStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, stats["pending"])
	require.True(t, strings.Contains(pending[0].PayloadJSON, "schema_version"))
}
