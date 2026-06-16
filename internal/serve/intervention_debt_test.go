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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/queue"
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

func TestCreateInterventionDebtTicketFromConfiguredSignals(t *testing.T) {
	t.Parallel()
	repo := setupInterventionDebtRepo(t)
	signals := []interventionDebtSignal{
		{
			Kind:     "terminal_agent_result",
			RepoID:   "repo-1",
			Role:     "engineer",
			JobID:    "job-terminal",
			Category: telemetry.CategoryUnknown,
			Message:  "agent ended with unknown failure",
			TraceID:  "tr-terminal",
			Outcome:  "failed",
		},
		{
			Kind:     "guardrail_block",
			RepoID:   "repo-1",
			Role:     "engineer",
			JobID:    "job-guardrail",
			Category: telemetry.CategoryGuardrailBlock,
			Message:  "guardrails: blocked by rule no-secrets",
			ToolName: "file_write",
		},
		{
			Kind:     "repeated_tool_loop",
			RepoID:   "repo-1",
			Role:     "engineer",
			JobID:    "job-loop",
			Category: telemetry.CategoryCircleDetected,
			Message:  "circle_detected: same tool call 3x",
			TraceID:  "tr-loop",
		},
		{
			Kind:     "human_followup_commit",
			RepoID:   "repo-1",
			Role:     "engineer",
			Category: telemetry.CategoryHumanFollowup,
			Message:  "human follow-up commit fixed agent output",
			Commit:   "abc1234",
		},
		{
			Kind:     "reverted_agent_commit",
			RepoID:   "repo-1",
			Role:     "engineer",
			Category: telemetry.CategoryRevertedCommit,
			Message:  "reverted agent commit",
			Commit:   "def5678",
		},
		{
			Kind:     "stale_in_progress_ticket",
			RepoID:   "repo-1",
			Role:     "engineer",
			Category: telemetry.CategoryStaleTicket,
			Message:  "stale in-progress ticket T-001",
		},
		{
			Kind:     "manual_stop",
			RepoID:   "repo-1",
			Role:     "engineer",
			JobID:    "job-stop",
			Category: telemetry.CategoryManualStop,
			Message:  "manual stop requested by operator",
		},
		{
			Kind:     "timeout",
			RepoID:   "repo-1",
			Role:     "engineer",
			JobID:    "job-timeout",
			Category: telemetry.CategoryToolTimeout,
			Message:  "agent ended with timeout",
			Outcome:  "timeout",
		},
	}

	for _, signal := range signals {
		proposal := interventionDebtProposalFromSignal(signal)
		ticket, err := createInterventionDebtTicket(repo, proposal, interventionDebtOrigin{
			Kind:           interventionDebtSignalKind(signal),
			EvidenceWindow: interventionDebtSignalWindow(signal),
			TraceID:        signal.TraceID,
			Commit:         signal.Commit,
			Outcome:        signal.Outcome,
			ToolName:       signal.ToolName,
			Message:        signal.Message,
		})
		require.NoError(t, err, "signal %s", signal.Kind)
		data, err := os.ReadFile(filepath.Join(repo, ticket.Path))
		require.NoError(t, err)
		text := string(data)
		require.Contains(t, text, "kind: intervention-debt", "signal %s", signal.Kind)
		require.Contains(t, text, `origin_kind: "`+signal.Kind+`"`, "signal %s", signal.Kind)
		require.Contains(t, text, `category: "`+string(signal.Category)+`"`, "signal %s", signal.Kind)
	}
}

func TestHandleJobFailedKeepsFoundationOwnedFailuresOutOfTargetBacklog(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()
	require.NoError(t, srv.traceStore.Save(ctx, "job-1", "tr-job-1", "{}", "{}"))

	job := &queue.Job{
		ID:     "job-1",
		RepoID: repoID,
		Role:   "engineer",
	}

	srv.handleJobFailed(ctx, job, errTest("executor: agent ended with max_turns"))
	srv.handleJobFailed(ctx, job, errTest("executor: agent ended with max_turns"))

	rec, err := srv.repos.FindByID(ctx, repoID)
	require.NoError(t, err)
	entries, err := os.ReadDir(filepath.Join(rec.Path, "docs", "tickets", "backlog"))
	if os.IsNotExist(err) {
		entries = nil
	} else {
		require.NoError(t, err)
	}
	require.Empty(t, entries)
	require.GreaterOrEqual(t, len(srv.telemetry.Events()), 2)
}

func TestRecordInterventionDebtSignalQuarantinesRuntimeFailures(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()
	rec, err := srv.repos.FindByID(ctx, repoID)
	require.NoError(t, err)
	require.NotNil(t, rec)

	for _, signal := range []interventionDebtSignal{
		{RepoID: repoID, Role: "engineer", JobID: "job-guardrail", Category: telemetry.CategoryGuardrailBlock, Message: "blast radius exceeded"},
		{RepoID: repoID, Role: "engineer", JobID: "job-workspace", Category: telemetry.CategoryWorkspaceHygiene, Message: "workspace_hygiene_blocked"},
		{RepoID: repoID, Role: "engineer", JobID: "job-ticket", Category: telemetry.CategoryTicketGate, Message: "ticket gate failed"},
		{RepoID: repoID, Role: "engineer", JobID: "job-dispatch", Category: telemetry.CategoryDispatchProtocol, Message: "dispatch mode requires job_disposition_record"},
		{RepoID: repoID, Role: "engineer", JobID: "job-max", Category: telemetry.CategoryMaxTurns, Message: "agent ended with max_turns"},
		{RepoID: repoID, Role: "engineer", JobID: "job-context", Category: telemetry.CategoryContextOverflow, Message: "context size exceeded"},
		{RepoID: repoID, Role: "engineer", JobID: "job-timeout", Category: telemetry.CategoryToolTimeout, Message: "timed out"},
		{RepoID: repoID, Role: "engineer", JobID: "job-unknown", Category: telemetry.CategoryUnknown, Message: "agent failed unexpectedly"},
	} {
		srv.recordInterventionDebtSignal(ctx, signal)
	}

	entries, err := os.ReadDir(filepath.Join(rec.Path, "docs", "tickets", "backlog"))
	if os.IsNotExist(err) {
		entries = nil
	} else {
		require.NoError(t, err)
	}
	require.Empty(t, entries)
	require.Len(t, srv.telemetry.Events(), 8)
}

func TestRecordInterventionDebtSignalOffersEvolutionForGuardrailLoop(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()
	rec, err := srv.repos.FindByID(ctx, repoID)
	require.NoError(t, err)
	require.NotNil(t, rec)

	srv.recordInterventionDebtSignal(ctx, interventionDebtSignal{
		RepoID:         repoID,
		Role:           "coo",
		JobID:          "job-loop",
		Category:       telemetry.CategoryGuardrailLoop,
		Count:          telemetry.PatternThreshold,
		EvidenceWindow: "same-job",
		ToolName:       "job_disposition_record",
		Message:        "repeated policy block loop after 3 identical blocks: post tool policy blocked job_disposition_record: missing feature contract coverage",
	})

	entries, err := os.ReadDir(filepath.Join(rec.Path, "docs", "tickets", "backlog"))
	if os.IsNotExist(err) {
		entries = nil
	} else {
		require.NoError(t, err)
	}
	require.Empty(t, entries)

	events := srv.telemetry.Events()
	require.Len(t, events, 1)
	require.Equal(t, telemetry.CategoryGuardrailLoop, events[0].Category)

	evolutions, err := srv.evoStore.GetEvolutions(ctx, "coo", 10)
	require.NoError(t, err)
	require.Len(t, evolutions, 1)
	require.Contains(t, evolutions[0].Result, `"classification":"signal_guardrail_loop"`)
	require.Contains(t, evolutions[0].Result, "repeatedly hit the same guardrail")
	require.Contains(t, evolutions[0].Result, ".harness/roles/coo.md")
}

func TestRecordInterventionDebtSignalMarksGuardrailLoopRemediated(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	srv.recordInterventionDebtSignal(ctx, interventionDebtSignal{
		RepoID:         repoID,
		Role:           "coo",
		JobID:          "job-loop",
		Category:       telemetry.CategoryGuardrailLoop,
		Count:          telemetry.PatternThreshold,
		EvidenceWindow: "same-job",
		ToolName:       "job_disposition_record",
		Message:        "repeated policy block loop after 3 identical blocks: post tool policy blocked job_disposition_record: missing feature contract coverage",
	})

	events := srv.telemetry.Events()
	require.Len(t, events, 1)
	require.False(t, events[0].Remedied)

	srv.recordInterventionDebtSignal(ctx, interventionDebtSignal{
		Kind:     "guardrail_loop_remediated",
		RepoID:   repoID,
		Role:     "coo",
		JobID:    "job-loop",
		Category: telemetry.CategoryGuardrailLoop,
		Outcome:  "remedied",
		Message:  "guardrail loop remediated by later terminal job_disposition_record",
	})

	events = srv.telemetry.Events()
	require.Len(t, events, 1)
	require.True(t, events[0].Remedied)
	require.Equal(t, "same_job_disposition", events[0].Action)
}

func TestRecordInterventionDebtSignalMarksPriorGuardrailLoopRemediatedByLaterJob(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	srv.recordInterventionDebtSignal(ctx, interventionDebtSignal{
		RepoID:         repoID,
		Role:           "coo",
		JobID:          "job-loop",
		Category:       telemetry.CategoryGuardrailLoop,
		Count:          telemetry.PatternThreshold,
		EvidenceWindow: "same-job",
		ToolName:       "job_disposition_record",
		Message:        "repeated policy block loop after 3 identical blocks: post tool policy blocked job_disposition_record: missing feature contract coverage",
	})

	srv.recordInterventionDebtSignal(ctx, interventionDebtSignal{
		Kind:           "guardrail_loop_remediated",
		RepoID:         repoID,
		Role:           "coo",
		JobID:          "job-later",
		Category:       telemetry.CategoryGuardrailLoop,
		EvidenceWindow: "later-job",
		Outcome:        "completed",
		Message:        "prior guardrail loop remediated by later accepted job_disposition_record",
	})

	events := srv.telemetry.Events()
	require.Len(t, events, 1)
	require.True(t, events[0].Remedied)
	require.Equal(t, "later_successful_disposition", events[0].Action)
}

func TestRecordInterventionDebtSignalAllowsTargetOwnedFailures(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()
	rec, err := srv.repos.FindByID(ctx, repoID)
	require.NoError(t, err)
	require.NotNil(t, rec)

	for _, signal := range []interventionDebtSignal{
		{RepoID: repoID, Role: "engineer", JobID: "job-human", Category: telemetry.CategoryHumanFollowup, Message: "human follow-up commit fixed target behavior"},
		{RepoID: repoID, Role: "engineer", JobID: "job-revert", Category: telemetry.CategoryRevertedCommit, Message: "reverted agent commit"},
		{RepoID: repoID, Role: "engineer", JobID: "job-stale", Category: telemetry.CategoryStaleTicket, Message: "stale in-progress ticket"},
	} {
		srv.recordInterventionDebtSignal(ctx, signal)
	}

	entries, err := os.ReadDir(filepath.Join(rec.Path, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 3)
}

func TestHandleJobFailedSuppressesSecondaryTicketGateAfterPolicyBlock(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	rec, err := srv.repos.FindByID(ctx, repoID)
	require.NoError(t, err)
	require.NotNil(t, rec)
	require.NoError(t, os.MkdirAll(filepath.Join(rec.Path, "docs", "tickets", "backlog"), 0o755))

	job := &queue.Job{
		ID:     "job-1",
		RepoID: repoID,
		Role:   "engineer",
	}

	srv.telemetry.Record(job.ID, job.RepoID, job.Role, "post tool policy blocked shell_exec: blast radius exceeded: 127 files changed (limit 10)")
	srv.handleJobFailed(ctx, job, errTest("executor: ticket gate: engineer cannot hand off with newly claimed ticket(s) still in docs/tickets/in-progress: T-003.md"))

	entries, err := os.ReadDir(filepath.Join(rec.Path, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Empty(t, entries, "ticket-gate fallout should not create a second intervention-debt ticket when the same job already hit policy blocks")
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

func TestCheckEvolutionKeepsFoundationTelemetryOutOfTargetBacklog(t *testing.T) {
	t.Parallel()
	repo := setupInterventionDebtRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness", "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", "manifest.yaml"), []byte("name: test\nroles:\n  engineer:\n    prompt: roles/engineer.md\n    tools: [file_read]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", "roles", "engineer.md"), []byte("# Engineer\n"), 0o644))

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })

	repoID, err := srv.Repos().Register(context.Background(), repo, "", "main")
	require.NoError(t, err)

	if srv.evoStore != nil {
		require.NoError(t, srv.evoStore.Close())
		srv.evoStore = nil
	}

	for i := 0; i < telemetry.PatternThreshold; i++ {
		srv.telemetry.Record("job-"+string(rune('a'+i)), repoID, "engineer", "agent stopped: max_turns reached")
	}

	srv.checkEvolution(context.Background(), "engineer", repoID)

	entries, err := os.ReadDir(filepath.Join(repo, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Empty(t, entries)
}
