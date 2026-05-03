package orgstate

import (
	"context"
	"path/filepath"
	"testing"

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
	}
	require.NoError(t, store.RecordDisposition(ctx, disposition))

	got, err := store.GetDisposition(ctx, "job-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "qa_review", got.NextNeed)
	require.Equal(t, []string{"go test ./..."}, got.EvidenceLinks)

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
	require.Contains(t, err.Error(), "blocked disposition requires next_need or suggested_role")
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
