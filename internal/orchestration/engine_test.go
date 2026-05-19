/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/design-docs/release-versioning.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-009-release-update-lifecycle.md
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

func TestDecide_completedEngineerRoutesDirectlyToQAWithOrchestratorPresent(t *testing.T) {
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
	require.Equal(t, "qa", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "without Orchestrator detour")
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

func TestDecide_noWorkWithNextNeedRoutesDirectly(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("ceo", "head-of-strategy", "orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:    "job-strategy",
			RepoID:   "repo-1",
			Role:     "ceo",
			Status:   "no_work",
			NextNeed: "strategy_advice",
			Reason:   "planning is ready; strategy advice is next",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "head-of-strategy", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "no-work")
}

func TestDecide_nonOrchestratorCompletedNextNeedSameRoleRoutesForward(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("coo", "cto-weekly", "orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:    "job-coo",
			RepoID:   "repo-1",
			Role:     "coo",
			Status:   "completed",
			NextNeed: "exec_plan",
			Reason:   "planning is still needed",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "cto-weekly", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "default forward owner")
	require.Empty(t, decision.StopReason)
}

func TestDecide_reviewNextNeedSameRoleRoutesForward(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("security", "dogfood", "release-manager", "orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:    "security-job",
			RepoID:   "repo-1",
			Role:     "security",
			Status:   "completed",
			NextNeed: "security_review",
			TicketID: "T-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "dogfood", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "next review owner")
	require.Empty(t, decision.StopReason)
}

func TestDecide_nonOrchestratorNoWorkNextNeedSameRoleStopsDirectDispatch(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("coo", "cto-weekly", "orchestrator"),
		Disposition: orgstate.Disposition{
			JobID:    "job-coo",
			RepoID:   "repo-1",
			Role:     "coo",
			Status:   "no_work",
			NextNeed: "feature_contract",
			Reason:   "contract still needs COO work",
		},
	})
	require.NoError(t, err)
	require.Empty(t, decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "no-work")
	require.Contains(t, decision.StopReason, "same-role")
}

func TestDecide_orchestratorSuggestedRoleCanonicalizesCase(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "engineer"),
		Disposition: orgstate.Disposition{
			JobID:         "job-2",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			SuggestedRole: "Engineer",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "engineer", decision.NextRole)
	require.Equal(t, "orchestrator", decision.DecisionKind)
}

func TestDecide_orchestratorSuggestedRoleAliasesCanonicalRoleKey(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:         "job-2",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			SuggestedRole: "cto",
			Reason:        "implementation tickets are needed",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "cto-weekly", decision.NextRole)
	require.Equal(t, "orchestrator", decision.DecisionKind)
}

func TestDecide_orchestratorHandoffTargetRoutesNext(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:  "job-2",
			RepoID: "repo-1",
			Role:   "orchestrator",
			Status: "completed",
			Handoff: orgstate.Handoff{
				TargetRole: "cto-weekly",
				Ask:        "shape implementation tickets",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "cto-weekly", decision.NextRole)
	require.Equal(t, "orchestrator", decision.DecisionKind)
	require.Contains(t, decision.Reason, "handoff.target_role")
}

func TestDecide_orchestratorFeedbackForRoleRoutesNext(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "coo"),
		Disposition: orgstate.Disposition{
			JobID:  "job-2",
			RepoID: "repo-1",
			Role:   "orchestrator",
			Status: "changes_requested",
			Reason: "plan needs correction",
			Feedback: orgstate.Feedback{
				ForRole:         "coo",
				RequestedChange: "clarify the current failing scenario",
				Severity:        "blocking",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "coo", decision.NextRole)
	require.Equal(t, "orchestrator", decision.DecisionKind)
	require.Contains(t, decision.Reason, "feedback.for_role")
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

func TestDecide_nonOrchestratorFeedbackReturnsToOrchestratorFirst(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "qa", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:  "job-1",
			RepoID: "repo-1",
			Role:   "qa",
			Status: "changes_requested",
			Reason: "ticket is under-specified",
			Feedback: orgstate.Feedback{
				ForRole:         "cto-weekly",
				RequestedChange: "split implementation scope",
				Severity:        "blocking",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "orchestrator", decision.NextRole)
	require.Equal(t, "orchestrator_review", decision.DecisionKind)
}

func TestDecide_qaTriggerContextBlockRetriesQAInspection(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "qa", "cto-weekly", "engineer"),
		Disposition: orgstate.Disposition{
			JobID:         "job-orchestrator",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			SuggestedRole: "cto-weekly",
			TicketID:      "T-001",
			Reason:        "CTO should reshape the ticket because QA lacked context",
		},
		SourceDisposition: &orgstate.Disposition{
			JobID:    "job-qa",
			RepoID:   "repo-1",
			Role:     "qa",
			Status:   "blocked",
			NextNeed: "liveness",
			TicketID: "T-001",
			Reason:   "Cannot perform quality review because the implementation source code was not provided in the trigger context.",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "qa", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "repository inspection")
}

func TestDecide_qaSubstantivePlanningBlockCanRouteCTO(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "qa", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:         "job-orchestrator",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			SuggestedRole: "cto-weekly",
			TicketID:      "T-001",
			Reason:        "CTO should reshape unclear acceptance criteria",
		},
		SourceDisposition: &orgstate.Disposition{
			JobID:    "job-qa",
			RepoID:   "repo-1",
			Role:     "qa",
			Status:   "blocked",
			NextNeed: "ticket_breakdown",
			TicketID: "T-001",
			Reason:   "The ticket lacks acceptance criteria and needs technical decomposition before QA can review.",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "cto-weekly", decision.NextRole)
	require.Equal(t, "orchestrator", decision.DecisionKind)
}

func TestDecide_rejectsConflictingStructuredTargets(t *testing.T) {
	t.Parallel()

	_, err := Decide(Input{
		Manifest: testManifest("orchestrator", "engineer", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:         "job-1",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			SuggestedRole: "engineer",
			Handoff: orgstate.Handoff{
				TargetRole: "cto-weekly",
				Ask:        "reshape ticket",
			},
		},
	})
	require.ErrorContains(t, err, "suggested_role")
	require.ErrorContains(t, err, "handoff.target_role")
}

func TestDecide_strategyAdviceRoutesToHeadOfStrategyWhenConfigured(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "ceo", "head-of-strategy"),
		Disposition: orgstate.Disposition{
			JobID:    "job-1",
			RepoID:   "repo-1",
			Role:     "orchestrator",
			Status:   "completed",
			NextNeed: "strategy_advice",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "head-of-strategy", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "next_need")
}

func TestDecide_strategyAdviceFallsBackToCEOWhenHeadOfStrategyMissing(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "ceo"),
		Disposition: orgstate.Disposition{
			JobID:    "job-1",
			RepoID:   "repo-1",
			Role:     "orchestrator",
			Status:   "completed",
			NextNeed: "strategy_advice",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "ceo", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
}

func TestDecide_ticketShapingRoutesToCTO(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "cto-weekly"),
		Disposition: orgstate.Disposition{
			JobID:    "job-1",
			RepoID:   "repo-1",
			Role:     "orchestrator",
			Status:   "blocked",
			NextNeed: "ticket_shaping",
			Reason:   "technical tickets are needed",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "cto-weekly", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
}

func TestDecide_execPlanRoutesToCOO(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "coo"),
		Disposition: orgstate.Disposition{
			JobID:    "job-1",
			RepoID:   "repo-1",
			Role:     "orchestrator",
			Status:   "blocked",
			NextNeed: "feature_contract",
			Reason:   "BDD contract is needed before tickets",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "coo", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
}

func TestDecide_defaultCompletionRouteMatchesOwnershipSpine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role string
		want string
	}{
		{"ceo", "coo"},
		{"head-of-strategy", "ceo"},
		{"coo", "cto-weekly"},
		{"cto-weekly", "engineer"},
		{"engineer", "qa"},
		{"qa", "security"},
		{"security", "dogfood"},
		{"dogfood", "release-manager"},
		{"dependency-manager", "release-manager"},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, defaultCompletionRoute(tt.role))
		})
	}
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

func TestDecide_orchestratorRoutesSecurityApprovalToDogfoodBeforeGovernance(t *testing.T) {
	t.Parallel()

	source := orgstate.Disposition{
		JobID:    "security-job",
		RepoID:   "repo-1",
		Role:     "security",
		Status:   "approved",
		TicketID: "T-001",
	}
	decision, err := Decide(Input{
		Manifest:          testManifest("orchestrator", "qa", "security", "dogfood", "dependency-manager", "release-manager"),
		SourceDisposition: &source,
		Disposition: orgstate.Disposition{
			JobID:         "orchestrator-job",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			NextNeed:      "qa_review",
			SuggestedRole: "qa",
			TicketID:      "T-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "dogfood", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "routing forward")
}

func TestDecide_orchestratorRoutesSecurityApprovalToReleaseWhenDogfoodAbsent(t *testing.T) {
	t.Parallel()

	source := orgstate.Disposition{
		JobID:    "security-job",
		RepoID:   "repo-1",
		Role:     "security",
		Status:   "completed",
		TicketID: "T-001",
	}
	decision, err := Decide(Input{
		Manifest:          testManifest("orchestrator", "qa", "security", "release-manager"),
		SourceDisposition: &source,
		Disposition: orgstate.Disposition{
			JobID:         "orchestrator-job",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			NextNeed:      "qa_review",
			SuggestedRole: "qa",
			TicketID:      "T-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "release-manager", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "routing forward")
}

func TestDecide_dogfoodApprovalRoutesDirectlyToReleaseManager(t *testing.T) {
	t.Parallel()

	decision, err := Decide(Input{
		Manifest: testManifest("orchestrator", "dogfood", "release-manager"),
		Disposition: orgstate.Disposition{
			JobID:    "dogfood-job",
			RepoID:   "repo-1",
			Role:     "dogfood",
			Status:   "approved",
			TicketID: "T-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "release-manager", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "without Orchestrator detour")
	require.Empty(t, decision.StopReason)
}

func TestDecide_orchestratorRoutesCompletedReviewChainToReleaseManager(t *testing.T) {
	t.Parallel()

	source := orgstate.Disposition{
		JobID:    "dogfood-job",
		RepoID:   "repo-1",
		Role:     "dogfood",
		Status:   "completed",
		TicketID: "T-001",
	}
	decision, err := Decide(Input{
		Manifest:          testManifest("orchestrator", "qa", "security", "dogfood", "release-manager"),
		SourceDisposition: &source,
		Disposition: orgstate.Disposition{
			JobID:         "orchestrator-job",
			RepoID:        "repo-1",
			Role:          "orchestrator",
			Status:        "completed",
			NextNeed:      "qa_review",
			SuggestedRole: "qa",
			TicketID:      "T-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "release-manager", decision.NextRole)
	require.Equal(t, "deterministic", decision.DecisionKind)
	require.Contains(t, decision.Reason, "routing forward")
	require.Empty(t, decision.StopReason)
}
