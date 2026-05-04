/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package orchestration

import (
	"fmt"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
)

// Input contains the state needed to choose the next role after a job disposition.
type Input struct {
	Disposition     orgstate.Disposition
	Manifest        *bundle.Manifest
	RecentDecisions []orgstate.Decision
	TicketStateHash string
}

// Decide routes a disposition to the Orchestrator, the Orchestrator-selected
// next role, a deterministic fallback for manifests without an Orchestrator, or
// a stop reason.
func Decide(in Input) (orgstate.Decision, error) {
	d := in.Disposition
	if in.Manifest == nil {
		return orgstate.Decision{}, fmt.Errorf("orchestration: manifest is required")
	}
	if d.JobID == "" || d.RepoID == "" || d.Role == "" {
		return orgstate.Decision{}, fmt.Errorf("orchestration: disposition job_id, repo_id, and role are required")
	}

	nextRole, kind, reason, stop := route(in)
	if nextRole != "" {
		validated, vReason := validateRole(in.Manifest, nextRole)
		if !validated {
			nextRole = fallbackRole(in.Manifest)
			kind = "ambiguous"
			reason = fmt.Sprintf("suggested route rejected: %s; routing to Orchestrator", vReason)
			if nextRole == "" {
				stop = "no orchestrator role configured for ambiguous route"
			}
		}
	}

	if nextRole != "" && repeatedRoute(in, nextRole) {
		if orch := fallbackRole(in.Manifest); orch != "" && orch != nextRole {
			kind = "ambiguous"
			reason = "loop guard detected repeated route without ticket-state change; routing to Orchestrator"
			nextRole = orch
			stop = ""
		}
	}

	return orgstate.Decision{
		JobID:           d.JobID,
		RepoID:          d.RepoID,
		SourceRole:      d.Role,
		TicketID:        d.TicketID,
		NextNeed:        d.NextNeed,
		NextRole:        nextRole,
		DecisionKind:    kind,
		Reason:          reason,
		StopReason:      stop,
		TicketStateHash: in.TicketStateHash,
	}, nil
}

func route(in Input) (nextRole, kind, reason, stop string) {
	d := in.Disposition
	status := normalize(d.Status)
	nextNeed := normalize(d.NextNeed)
	suggested := strings.TrimSpace(d.SuggestedRole)

	if d.Role != "orchestrator" {
		if orch := orchestratorRole(in.Manifest); orch != "" {
			return orch, "orchestrator_review", "terminal disposition returned to Orchestrator for next-role selection", ""
		}
	}

	if suggested != "" {
		return suggested, "orchestrator", "using Orchestrator suggested_role", ""
	}

	switch status {
	case "changes_requested":
		return "engineer", "deterministic", "review requested implementation changes", ""
	case "in_review":
		if suggested != "" {
			return suggested, "deterministic", "routing in-review ticket to requested reviewer", ""
		}
		return "qa", "deterministic", "routing in-review ticket to QA", ""
	case "blocked":
		if role := roleForNeed(nextNeed); role != "" {
			return role, "deterministic", "routing blocker by next_need", ""
		}
		return fallbackRole(in.Manifest), "ambiguous", "blocked disposition lacks a deterministic next role", ""
	case "ambiguous":
		return fallbackRole(in.Manifest), "ambiguous", "role reported ambiguous state", ""
	case "failed":
		return fallbackRole(in.Manifest), "ambiguous", "failed job requires orchestration review", ""
	case "no_work":
		if d.Role == "engineer" {
			return "ceo", "deterministic", "engineer found no actionable work; returning to planning", ""
		}
		return "", "deterministic", "no work disposition stops dispatch", "no actionable work"
	case "approved":
		if role := roleForNeed(nextNeed); role != "" {
			return role, "deterministic", "routing approved work by next_need", ""
		}
		if role := defaultCompletionRoute(d.Role); role != "" {
			return role, "deterministic", "routing approved work by default role responsibility", ""
		}
		return "release-manager", "deterministic", "approved work is ready for release review", ""
	case "completed":
		if role := roleForNeed(nextNeed); role != "" {
			return role, "deterministic", "routing completed work by next_need", ""
		}
		if role := defaultCompletionRoute(d.Role); role != "" {
			return role, "deterministic", "routing completed work by default role responsibility", ""
		}
		return "", "deterministic", "completed work has no required follow-up", "completed with no follow-up"
	default:
		return fallbackRole(in.Manifest), "ambiguous", fmt.Sprintf("unknown disposition status %q", d.Status), ""
	}
}

func roleForNeed(nextNeed string) string {
	switch nextNeed {
	case "qa", "qa_review", "evidence_review", "review":
		return "qa"
	case "security", "security_review":
		return "security"
	case "dependency", "dependency_maintenance":
		return "dependency-manager"
	case "release", "release_review", "release_blocked":
		return "release-manager"
	case "implementation", "implementation_rework", "engineering", "fix":
		return "engineer"
	case "architecture", "architecture_review", "architecture_blocker":
		return "cto-weekly"
	case "ticket", "ticket_shaping", "ticket_breakdown":
		return "coo"
	case "goal", "goals", "goal_decision", "strategy", "strategy_review":
		return "ceo"
	case "dogfood", "e2e", "end_to_end":
		return "dogfood"
	case "janitor", "liveness", "stale_checkout":
		return "janitor"
	default:
		return ""
	}
}

func defaultCompletionRoute(role string) string {
	switch role {
	case "ceo":
		return "cto-weekly"
	case "cto-weekly":
		return "coo"
	case "coo":
		return "engineer"
	case "engineer", "pipeline-fixer":
		return "qa"
	case "qa":
		return "security"
	case "security":
		return "dependency-manager"
	default:
		return ""
	}
}

func validateRole(m *bundle.Manifest, role string) (bool, string) {
	if strings.TrimSpace(role) == "" {
		return true, ""
	}
	if _, ok := m.Roles[role]; !ok {
		return false, fmt.Sprintf("role %q is not defined in manifest", role)
	}
	return true, ""
}

func fallbackRole(m *bundle.Manifest) string {
	if m == nil {
		return ""
	}
	if role := orchestratorRole(m); role != "" {
		return role
	}
	if _, ok := m.Roles["janitor"]; ok {
		return "janitor"
	}
	return ""
}

func orchestratorRole(m *bundle.Manifest) string {
	if m == nil {
		return ""
	}
	if _, ok := m.Roles["orchestrator"]; ok {
		return "orchestrator"
	}
	return ""
}

func repeatedRoute(in Input, nextRole string) bool {
	if nextRole == "" || in.TicketStateHash == "" {
		return false
	}
	count := 0
	d := in.Disposition
	for _, prev := range in.RecentDecisions {
		if prev.RepoID == d.RepoID &&
			prev.TicketID == d.TicketID &&
			prev.NextNeed == d.NextNeed &&
			prev.NextRole == nextRole &&
			prev.TicketStateHash == in.TicketStateHash {
			count++
		}
	}
	return count >= 2
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}
