/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package orgstate

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestStoreRecordsDispositionAndDecision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "orgstate.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	disposition := Disposition{
		JobID:         "job-1",
		RepoID:        "repo-1",
		Role:          "engineer",
		Status:        "blocked",
		NextNeed:      "qa_review",
		TicketID:      "MH-001",
		Reason:        "needs evidence review",
		EvidenceLinks: []string{"go test ./..."},
		Handoff: Handoff{
			TargetRole:      "qa",
			Ask:             "review the evidence",
			Context:         "ticket MH-001",
			Constraints:     []string{"do not approve without test output"},
			ExpectedOutput:  "approval or changes requested",
			SuccessEvidence: []string{"review note"},
		},
		Feedback: Feedback{
			ForRole:         "qa",
			Summary:         "missing evidence",
			RequestedChange: "verify attached test output before approval",
			Severity:        "revision_requested",
			EvidenceLinks:   []string{"docs/tickets/in-progress/MH-001.md"},
		},
	}
	require.NoError(t, store.RecordDisposition(ctx, disposition))

	got, err := store.GetDisposition(ctx, "job-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "qa_review", got.NextNeed)
	require.Equal(t, []string{"go test ./..."}, got.EvidenceLinks)
	require.Equal(t, "qa", got.Handoff.TargetRole)
	require.Equal(t, "verify attached test output before approval", got.Feedback.RequestedChange)

	dispositions, err := store.RecentDispositions(ctx, "repo-1", 10)
	require.NoError(t, err)
	require.Len(t, dispositions, 1)

	decision, err := store.RecordDecision(ctx, Decision{
		JobID:        "job-1",
		RepoID:       "repo-1",
		SourceRole:   "engineer",
		TicketID:     "MH-001",
		NextNeed:     "qa_review",
		NextRole:     "qa",
		DecisionKind: "deterministic",
		Reason:       "routing by next need",
	})
	require.NoError(t, err)
	require.NotEmpty(t, decision.ID)

	decisions, err := store.RecentDecisions(ctx, "repo-1", 10)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, decision.ID, decisions[0].ID)
}

func TestDecodeDispositionNormalizesStringLists(t *testing.T) {
	t.Parallel()

	got, err := DecodeDisposition([]byte(`{
		"status":"completed",
		"next_need":"implementation",
		"suggested_role":"engineer",
		"ticket_id":"T-001",
		"evidence_links":"['docs/exec-plans/active/current-operating-plan.md', 'docs/features/F-001-product-walking-skeleton.md']",
		"work_product_ids":"[\"plan\", \"feature\"]",
		"blocked_by":"manual follow-up",
		"handoff":{
			"target_role":"engineer",
			"ask":"claim T-001",
			"constraints":"['one ticket only']",
			"success_evidence":"[\"T-001 in done\"]"
		},
		"feedback":{
			"for_role":"coo",
			"requested_change":"tighten plan",
			"severity":"revision_requested",
			"evidence_links":"docs/exec-plans/active/current-operating-plan.md"
		}
	}`))
	require.NoError(t, err)
	require.Equal(t, []string{
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/features/F-001-product-walking-skeleton.md",
	}, got.EvidenceLinks)
	require.Equal(t, []string{"plan", "feature"}, got.WorkProductIDs)
	require.Equal(t, []string{"manual follow-up"}, got.BlockedBy)
	require.Equal(t, []string{"one ticket only"}, got.Handoff.Constraints)
	require.Equal(t, []string{"T-001 in done"}, got.Handoff.SuccessEvidence)
	require.Equal(t, []string{"docs/exec-plans/active/current-operating-plan.md"}, got.Feedback.EvidenceLinks)
}

func TestValidateDispositionRequiresBlockedNextStep(t *testing.T) {
	t.Parallel()

	err := ValidateDisposition(Disposition{
		JobID:  "job-1",
		RepoID: "repo-1",
		Role:   "engineer",
		Status: "blocked",
		Reason: "needs help",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked disposition requires next_need, suggested_role, handoff.target_role, or feedback.for_role")
}

func TestValidateDispositionAllowsApprovedWithoutReason(t *testing.T) {
	t.Parallel()

	err := ValidateDisposition(Disposition{
		JobID:  "job-1",
		RepoID: "repo-1",
		Role:   "qa",
		Status: "approved",
	})
	require.NoError(t, err)
}

func TestValidateDispositionRejectsInvalidStructuredHandoffAndFeedback(t *testing.T) {
	t.Parallel()

	base := Disposition{
		JobID:    "job-1",
		RepoID:   "repo-1",
		Role:     "qa",
		Status:   "changes_requested",
		Reason:   "needs rework",
		NextNeed: "implementation_rework",
	}

	invalidHandoff := base
	invalidHandoff.Handoff = Handoff{TargetRole: "engineer"}
	require.ErrorContains(t, ValidateDisposition(invalidHandoff), "handoff.ask is required")

	invalidFeedback := base
	invalidFeedback.Feedback = Feedback{ForRole: "engineer"}
	require.ErrorContains(t, ValidateDisposition(invalidFeedback), "feedback.requested_change is required")

	invalidSeverity := base
	invalidSeverity.Feedback = Feedback{ForRole: "engineer", RequestedChange: "fix tests", Severity: "urgent"}
	require.ErrorContains(t, ValidateDisposition(invalidSeverity), "feedback.severity")

	conflictingSuggested := base
	conflictingSuggested.SuggestedRole = "engineer"
	conflictingSuggested.Handoff = Handoff{TargetRole: "cto-weekly", Ask: "reshape the ticket"}
	require.ErrorContains(t, ValidateDisposition(conflictingSuggested), "suggested_role")
	require.ErrorContains(t, ValidateDisposition(conflictingSuggested), "handoff.target_role")

	conflictingFeedback := base
	conflictingFeedback.Handoff = Handoff{TargetRole: "engineer", Ask: "fix the implementation"}
	conflictingFeedback.Feedback = Feedback{ForRole: "cto-weekly", RequestedChange: "reshape the ticket", Severity: "blocking"}
	require.ErrorContains(t, ValidateDisposition(conflictingFeedback), "handoff.target_role")
	require.ErrorContains(t, ValidateDisposition(conflictingFeedback), "feedback.for_role")

	ambiguousConflict := base
	ambiguousConflict.Status = "ambiguous"
	ambiguousConflict.Handoff = Handoff{TargetRole: "engineer", Ask: "fix the implementation"}
	ambiguousConflict.Feedback = Feedback{ForRole: "cto-weekly", RequestedChange: "reshape the ticket", Severity: "blocking"}
	require.NoError(t, ValidateDisposition(ambiguousConflict))
}

func TestOpenStoreMigratesExistingDispositionsWithoutStructuredFields(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "orgstate.db")
	testutil.WriteSQLiteFixture(t, dbPath, `
CREATE TABLE job_dispositions (
  job_id TEXT PRIMARY KEY,
  repo_id TEXT NOT NULL,
  role TEXT NOT NULL,
  status TEXT NOT NULL,
  next_need TEXT NOT NULL DEFAULT '',
  suggested_role TEXT NOT NULL DEFAULT '',
  ticket_id TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  evidence_json TEXT NOT NULL DEFAULT '[]',
  approval_id TEXT NOT NULL DEFAULT '',
  work_products_json TEXT NOT NULL DEFAULT '[]',
  blocked_by_json TEXT NOT NULL DEFAULT '[]',
  trace_id TEXT NOT NULL DEFAULT '',
  recorded_at INTEGER NOT NULL
);
INSERT INTO job_dispositions(job_id, repo_id, role, status, reason, recorded_at)
VALUES('job-legacy', 'repo-1', 'engineer', 'completed', '', 1);
`)

	store, err := OpenStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})
	testutil.AssertSQLiteColumns(t, store.db, "job_dispositions", "handoff_json", "feedback_json")
	testutil.AssertSQLiteIndexes(t, store.db, "idx_job_dispositions_repo_time", "idx_orchestration_decisions_loop")

	got, err := store.GetDisposition(context.Background(), "job-legacy")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got.Handoff)
	require.Empty(t, got.Feedback)
}
