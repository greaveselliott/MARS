package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func setupInterventionDebtRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, sub), 0o755))
	}
	return dir
}

func TestCreateInterventionDebtTicketFromTelemetryPattern(t *testing.T) {
	t.Parallel()
	repo := setupInterventionDebtRepo(t)
	proposal := telemetry.TriagePattern(telemetry.Pattern{
		RepoID:   "repo-1",
		Role:     "engineer",
		Category: telemetry.CategoryContextOverflow,
		Count:    telemetry.PatternThreshold,
		Window:   "24h",
	})
	origin := interventionDebtOrigin{
		Kind:           "telemetry_pattern",
		EvidenceWindow: "24h",
		Event: &telemetry.Event{
			ID:        "evt-1",
			Timestamp: time.Now().UTC(),
			JobID:     "job-1",
			RepoID:    "repo-1",
			Role:      "engineer",
			Category:  telemetry.CategoryContextOverflow,
			Message:   "request exceeds context size",
		},
	}

	ticket, err := createInterventionDebtTicket(repo, proposal, origin)
	require.NoError(t, err)
	require.Contains(t, ticket.Output, "created ticket")
	require.NotEmpty(t, ticket.Path)

	data, err := os.ReadFile(filepath.Join(repo, ticket.Path))
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "kind: intervention-debt")
	require.Contains(t, text, "Role: engineer")
	require.Contains(t, text, "Category: context_overflow")
	require.Contains(t, text, "Origin event: evt-1")
	require.Contains(t, text, "request exceeds context size")
}

func TestCreateInterventionDebtTicketUpdatesMatchingDedupeKey(t *testing.T) {
	t.Parallel()
	repo := setupInterventionDebtRepo(t)
	proposal := telemetry.TriagePattern(telemetry.Pattern{
		RepoID:   "repo-1",
		Role:     "engineer",
		Category: telemetry.CategoryMaxTurns,
		Count:    telemetry.PatternThreshold,
		Window:   "24h",
	})
	origin := interventionDebtOrigin{
		Kind:           "telemetry_pattern",
		EvidenceWindow: "24h",
		Event:          &telemetry.Event{ID: "evt-1", JobID: "job-1", RepoID: "repo-1", Role: "engineer", Category: telemetry.CategoryMaxTurns},
	}
	_, err := createInterventionDebtTicket(repo, proposal, origin)
	require.NoError(t, err)

	origin.Event = &telemetry.Event{ID: "evt-2", JobID: "job-2", RepoID: "repo-1", Role: "engineer", Category: telemetry.CategoryMaxTurns}
	ticket, err := createInterventionDebtTicket(repo, proposal, origin)
	require.NoError(t, err)
	require.Contains(t, ticket.Output, "UPDATED")

	entries, err := os.ReadDir(filepath.Join(repo, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(repo, "docs", "tickets", "backlog", entries[0].Name()))
	require.NoError(t, err)
	require.Contains(t, string(data), "source: telemetry:evt-2")
}

func TestCreateInterventionDebtTicketFromLowScoreSnapshot(t *testing.T) {
	t.Parallel()
	repo := setupInterventionDebtRepo(t)
	snapshot := telemetry.ScoreSnapshot{
		Role:       "dogfood",
		RepoID:     "repo-1",
		Value:      0.25,
		SampleSize: 8,
		WindowDays: 30,
	}
	proposal, ok := telemetry.TriageScore(snapshot)
	require.True(t, ok)

	ticket, err := createInterventionDebtTicket(repo, proposal, interventionDebtOrigin{
		Kind:           "score_snapshot",
		EvidenceWindow: "30d",
		Score:          &snapshot,
	})
	require.NoError(t, err)
	require.Contains(t, ticket.Output, "created ticket")

	data, err := os.ReadFile(filepath.Join(repo, ticket.Path))
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "priority: high")
	require.Contains(t, text, "Score snapshot: 0.25 over 8 samples in 30d")
	require.Contains(t, text, `source: score:repo-1:dogfood:30d`)
}

func TestInterventionDebtUnknownFailureStaysTicketed(t *testing.T) {
	t.Parallel()
	repo := setupInterventionDebtRepo(t)
	proposal := telemetry.TriagePattern(telemetry.Pattern{
		RepoID:   "repo-1",
		Role:     "engineer",
		Category: telemetry.CategoryUnknown,
		Count:    telemetry.PatternThreshold,
		Window:   "24h",
	})
	ticket, err := createInterventionDebtTicket(repo, proposal, interventionDebtOrigin{
		Kind:           "telemetry_pattern",
		EvidenceWindow: "24h",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repo, ticket.Path))
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "Classify unknown failure")
	require.Contains(t, text, "Unknown or unsafe fixes remain ticketed")
	require.False(t, strings.Contains(text, "directly edit arbitrary files"))
}

func TestRecordInterventionDebtFailureRecordsTelemetry(t *testing.T) {
	t.Parallel()
	srv := &Server{telemetry: telemetry.NewCollector(nil, nil)}
	proposal := telemetry.ImprovementProposal{
		RepoID:   "repo-1",
		Role:     "engineer",
		Target:   telemetry.TargetProcess,
		Category: telemetry.CategoryUnknown,
	}

	srv.recordInterventionDebtFailure(proposal, interventionDebtOrigin{
		Event: &telemetry.Event{ID: "evt-1", JobID: "job-1"},
	}, os.ErrPermission)

	events := srv.telemetry.Events()
	require.Len(t, events, 1)
	require.Equal(t, "job-1", events[0].JobID)
	require.Equal(t, "repo-1", events[0].RepoID)
	require.Equal(t, "engineer", events[0].Role)
	require.Contains(t, events[0].Message, "intervention debt ticket creation failed")
}
