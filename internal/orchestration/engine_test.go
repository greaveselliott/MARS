/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package orchestration

import (
	"testing"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/stretchr/testify/require"
)

func testManifest(roles ...string) *bundle.Manifest {
	m := &bundle.Manifest{Roles: map[string]bundle.RoleConfig{}}
	for _, role := range roles {
		m.Roles[role] = bundle.RoleConfig{Prompt: "roles/" + role + ".md"}
	}
	return m
}

func TestDecide_completedEngineerRoutesToQAMissingOrchestrator(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("engineer", "qa"),
		Disposition: orgstate.Disposition{
			JobID:  "job-1",
			RepoID: "repo-1",
			Role:   "engineer",
			Status: "completed",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "qa", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Empty(t, decision.StopReason)
}

func TestDecide_completedEngineerReturnsToOrchestrator(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("engineer", "qa", "orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:  "job-1",
			RepoID: "repo-1",
			Role:   "engineer",
			Status: "completed",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "orchestrator", decision.NextRole)
	require.Equal(t, "orchestrator_review", decision.DecisionKind)
	require.Contains(t, decision.Reason, "returned to Orchestrator")
}

func TestDecide_orchestratorSuggestedRoleRoutesNext(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:         "job-2",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			SuggestedRole: "cto-weekly",
			Reason:        "plan needs architecture review",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "cto-weekly", decision.NextRole)
	require.Equal(t, "orchestrator", decision.DecisionKind)
	require.Contains(t, decision.Reason, "suggested_role")
}

func TestDecide_invalidOrchestratorSuggestedRoleFallsBackToOrchestrator(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:         "job-1",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "blocked",
			SuggestedRole: "missing-role",
			Reason:        "needs routing",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "orchestrator", decision.NextRole)
	require.Equal(t, "ambiguous", decision.DecisionKind)
	require.Contains(t, decision.Reason, "suggested route rejected")
}

func TestDecide_repeatedOrchestratorRouteStopsDispatch(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("engineer", "qa", "orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:         "job-3",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			NextNeed:      "qa_review",
			SuggestedRole: "qa",
			TicketID:      "MH-001",
		},
		TicketStateHash: "same-state",
		RecentDecisions: []orgstate.Decision{
			{RepoID: "repo-1", SourceRole: "orchestrator", TicketID: "MH-001", NextNeed: "qa_review", NextRole: "qa", TicketStateHash: "same-state"},
			{RepoID: "repo-1", SourceRole: "orchestrator", TicketID: "MH-001", NextNeed: "qa_review", NextRole: "qa", TicketStateHash: "same-state"},
		},
	})
	require.NoError(t, err)
	require.Empty(t, decision.NextRole)
	require.Equal(t, "ambiguous", decision.DecisionKind)
	require.Contains(t, decision.Reason, "loop guard")
	require.Contains(t, decision.StopReason, "loop guard")
}

func TestDecide_noWorkStopsDispatch(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("qa"),
		Disposition: orgstate.Disposition{
			JobID:  "job-1",
			RepoID: "repo-1",
			Role:   "qa",
			Status: "no_work",
			Reason: "nothing to review",
		},
	})
	require.NoError(t, err)
	require.Empty(t, decision.NextRole)
	require.Equal(t, "no actionable work", decision.StopReason)
}

func TestDecide_approvedUsesDefaultResponsibility(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("qa", "security"),
		Disposition: orgstate.Disposition{
			JobID:  "job-1",
			RepoID: "repo-1",
			Role:   "qa",
			Status: "approved",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "security", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
}
