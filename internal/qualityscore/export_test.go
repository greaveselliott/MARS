package qualityscore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func setupQualityRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{
		"docs/tickets/backlog",
		"docs/tickets/in-progress",
		"docs/tickets/done",
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, sub), 0o755))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs"), 0o755))
	return dir
}

func TestExportMissingDatabasePreservesManualNotes(t *testing.T) {
	t.Parallel()
	repo := setupQualityRepo(t)
	initial := "# Quality Score\n\n## Manual Notes\n\n" + manualStart + "\nKeep this operator note.\n" + manualEnd + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(repo, "docs", "QUALITY_SCORE.md"), []byte(initial), 0o644))

	report, err := Export(context.Background(), Options{
		RepoPath:              repo,
		DBPath:                filepath.Join(repo, "missing.db"),
		Now:                   time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC),
		DisableTicketCreation: true,
	})
	require.NoError(t, err)
	require.Equal(t, "Insufficient evidence", report.Grade)
	require.Len(t, report.Warnings, 1)

	data, err := os.ReadFile(filepath.Join(repo, "docs", "QUALITY_SCORE.md"))
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "**Status:** Generated")
	require.Contains(t, text, "Current overall grade: Insufficient evidence")
	require.Contains(t, text, "Keep this operator note.")
	require.Contains(t, text, "| Guardrail blocks | None recorded |")
	require.Contains(t, text, "| Human follow-up | None recorded |")
	require.Contains(t, text, "No SQLite database found")
}

func TestExportCreatesDedupedRegressionTicket(t *testing.T) {
	t.Parallel()
	repo := setupQualityRepo(t)
	dbPath := filepath.Join(repo, "mars.db")
	store, err := scoring.OpenStore(dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 6; i++ {
		require.NoError(t, store.RecordOutcome(ctx, scoring.Outcome{
			JobID:      fmt.Sprintf("failed-%d", i),
			RepoID:     "repo-1",
			Role:       "engineer",
			Type:       scoring.OutcomeFailed,
			RecordedAt: now.Add(-time.Hour),
		}))
	}
	require.NoError(t, store.RecordOutcome(ctx, scoring.Outcome{
		JobID:      "passed-1",
		RepoID:     "repo-1",
		Role:       "engineer",
		Type:       scoring.OutcomePassed,
		RecordedAt: now.Add(-time.Hour),
	}))
	require.NoError(t, store.Close())

	report, err := Export(ctx, Options{
		RepoPath:   repo,
		RepoID:     "repo-1",
		DBPath:     dbPath,
		Now:        now,
		WindowDays: 30,
	})
	require.NoError(t, err)
	require.Equal(t, "F", report.Grade)
	require.Len(t, report.TicketsChanged, 1)

	entries, err := os.ReadDir(filepath.Join(repo, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.True(t, strings.HasPrefix(entries[0].Name(), "T-001-"))

	ticketData, err := os.ReadFile(filepath.Join(repo, "docs", "tickets", "backlog", entries[0].Name()))
	require.NoError(t, err)
	ticketText := string(ticketData)
	require.Contains(t, ticketText, "kind: intervention-debt")
	require.Contains(t, ticketText, "Score snapshot: 0.14 over 7 samples in 30d")
	require.Contains(t, ticketText, "dedupe_key: \"intervention-debt:repo-1:engineer:process:score:30d\"")

	report, err = Export(ctx, Options{
		RepoPath:   repo,
		RepoID:     "repo-1",
		DBPath:     dbPath,
		Now:        now,
		WindowDays: 30,
	})
	require.NoError(t, err)
	require.Len(t, report.TicketsChanged, 1)
	require.Contains(t, report.TicketsChanged[0], "UNCHANGED")
	entries, err = os.ReadDir(filepath.Join(repo, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestExportRendersTelemetryAndOutcomeSignals(t *testing.T) {
	t.Parallel()
	repo := setupQualityRepo(t)
	dbPath := filepath.Join(repo, "mars.db")
	scoreStore, err := scoring.OpenStore(dbPath)
	require.NoError(t, err)
	telemStore, err := telemetry.OpenStore(dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)
	outcomes := []scoring.OutcomeType{
		scoring.OutcomePassed,
		scoring.OutcomeChecksPassed,
		scoring.OutcomeChecksFailed,
		scoring.OutcomeGuardrailBlocked,
		scoring.OutcomeNoop,
		scoring.OutcomeHumanFollowup,
		scoring.OutcomeFailed,
	}
	for i, typ := range outcomes {
		role := "engineer"
		if typ == scoring.OutcomeFailed {
			role = "dogfood"
		}
		require.NoError(t, scoreStore.RecordOutcome(ctx, scoring.Outcome{
			JobID:      fmt.Sprintf("job-%d", i),
			RepoID:     "repo-1",
			Role:       role,
			Type:       typ,
			RecordedAt: now.Add(-time.Hour),
		}))
	}
	for i := 0; i < telemetry.PatternThreshold; i++ {
		require.NoError(t, telemStore.Save(telemetry.Event{
			ID:        fmt.Sprintf("evt-%d", i),
			Timestamp: now.Add(-time.Minute),
			JobID:     fmt.Sprintf("job-%d", i),
			RepoID:    "repo-1",
			Role:      "engineer",
			Category:  telemetry.CategoryToolTimeout,
			Message:   "tool timed out",
		}))
	}
	require.NoError(t, scoreStore.Close())
	require.NoError(t, telemStore.Close())

	report, err := Export(ctx, Options{
		RepoPath:              repo,
		RepoID:                "repo-1",
		DBPath:                dbPath,
		Now:                   now,
		WindowDays:            30,
		DisableTicketCreation: true,
	})
	require.NoError(t, err)
	require.NotEqual(t, "Insufficient evidence", report.Grade)

	data, err := os.ReadFile(filepath.Join(repo, "docs", "QUALITY_SCORE.md"))
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "| Failed dogfood | 1 dogfood failures |")
	require.Contains(t, text, "| Guardrail blocks | 1 guardrail blocks |")
	require.Contains(t, text, "| Check results | 1 passed, 1 failed |")
	require.Contains(t, text, "| No-op runs | 1 no-op runs |")
	require.Contains(t, text, "| Human follow-up | 1 human follow-up outcomes |")
	require.Contains(t, text, "`repo-1/engineer` tool_timeout x3")
}
