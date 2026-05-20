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
	// SourceDisposition is populated for Orchestrator dispatch jobs so the
	// deterministic router can reject lifecycle regressions in the handoff the
	// Orchestrator just selected.
	SourceDisposition *orgstate.Disposition
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
	if err := orgstate.ValidateDisposition(d); err != nil {
		return orgstate.Decision{}, fmt.Errorf("orchestration: invalid disposition: %w", err)
	}

	nextRole, kind, reason, stop := route(in)
	if nextRole != "" {
		canonical, validated, vReason := validateRole(in.Manifest, nextRole)
		if !validated {
			nextRole = fallbackRole(in.Manifest)
			kind = "ambiguous"
			reason = fmt.Sprintf("suggested route rejected: %s; routing to Orchestrator", vReason)
			if nextRole == "" {
				stop = "no orchestrator role configured for ambiguous route"
			}
		} else {
			nextRole = canonical
		}
	}

	if nextRole != "" && repeatedRoute(in, nextRole) {
		kind = "ambiguous"
		reason = "loop guard detected repeated route without ticket-state change"
		if orch := fallbackRole(in.Manifest); orch != "" && orch != nextRole && d.Role != orch {
			reason += "; routing to Orchestrator"
			nextRole = orch
			stop = ""
		} else {
			nextRole = ""
			stop = "loop guard stopped repeated route without ticket-state change"
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
		if nextRole, kind, reason, stop, ok := directDeterministicRoute(in, status, nextNeed, suggested); ok {
			return nextRole, kind, reason, stop
		}
		if orch := orchestratorRole(in.Manifest); orch != "" {
			return orch, "orchestrator_review", "terminal disposition returned to Orchestrator for next-role selection", ""
		}
	}

	if suggested != "" {
		return enforceOrchestratorCandidate(in, suggested, "orchestrator", "using Orchestrator suggested_role")
	}
	if target := strings.TrimSpace(d.Handoff.TargetRole); target != "" {
		return enforceOrchestratorCandidate(in, target, "orchestrator", "using Orchestrator handoff.target_role")
	}
	if target := strings.TrimSpace(d.Feedback.ForRole); target != "" {
		return enforceOrchestratorCandidate(in, target, "orchestrator", "using Orchestrator feedback.for_role")
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
		if role := roleForNeedInManifest(in.Manifest, nextNeed); role != "" {
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
		if role := roleForNeedInManifest(in.Manifest, nextNeed); role != "" {
			return enforceReviewProgression(in, role, "deterministic", "routing approved work by next_need")
		}
		if role := defaultCompletionRoute(d.Role); role != "" {
			return role, "deterministic", "routing approved work by default role responsibility", ""
		}
		return "release-manager", "deterministic", "approved work is ready for release review", ""
	case "completed":
		if role := roleForNeedInManifest(in.Manifest, nextNeed); role != "" {
			return enforceReviewProgression(in, role, "deterministic", "routing completed work by next_need")
		}
		if role := defaultCompletionRoute(d.Role); role != "" {
			return role, "deterministic", "routing completed work by default role responsibility", ""
		}
		return "", "deterministic", "completed work has no required follow-up", "completed with no follow-up"
	default:
		return fallbackRole(in.Manifest), "ambiguous", fmt.Sprintf("unknown disposition status %q", d.Status), ""
	}
}

func directDeterministicRoute(in Input, status, nextNeed, suggested string) (nextRole, kind, reason, stop string, ok bool) {
	d := in.Disposition
	direct := func(candidateRole, candidateReason string) (string, string, string, string, bool) {
		candidateRole = strings.TrimSpace(candidateRole)
		if candidateRole == "" {
			return "", "", "", "", false
		}
		nextRole, nextKind, nextReason, nextStop := enforceDirectReviewProgression(in, candidateRole, "deterministic", candidateReason)
		return nextRole, nextKind, nextReason, nextStop, true
	}

	switch status {
	case "blocked":
		if releasePublicationBlocked(d) {
			return "", "deterministic", "release publication blocker is operator-visible; stopping dispatch without Orchestrator detour", "release publication blocked", true
		}
		return "", "", "", "", false
	case "completed", "approved":
		if suggested != "" {
			return direct(suggested, "using role suggested_role without Orchestrator detour")
		}
		if target := strings.TrimSpace(d.Handoff.TargetRole); target != "" {
			return direct(target, "using role handoff.target_role without Orchestrator detour")
		}
		if target := strings.TrimSpace(d.Feedback.ForRole); target != "" {
			return direct(target, "using role feedback.for_role without Orchestrator detour")
		}
		if role := roleForNeedInManifest(in.Manifest, nextNeed); role != "" {
			if sameRole(d.Role, role) {
				if forward := nextReviewLifecycleRoleForRole(in.Manifest, d.Role); forward != "" {
					return direct(forward, "current review next_need already belongs to source role; routing to next review owner without Orchestrator detour")
				}
				if forward := defaultCompletionRoute(d.Role); forward != "" {
					return direct(forward, "completed same-role next_need already belongs to source role; routing to default forward owner without Orchestrator detour")
				}
				return "", "deterministic", "next_need resolves to the current role; stopping direct dispatch to avoid a same-role loop", "same-role next_need has no forward owner", true
			}
			return direct(role, "routing completed work by next_need without Orchestrator detour")
		}
		if role := defaultCompletionRoute(d.Role); role != "" {
			return direct(role, "routing completed work by default role responsibility without Orchestrator detour")
		}
		return "", "deterministic", "completed work has no required follow-up", "completed with no follow-up", true
	case "in_review":
		if suggested != "" {
			return direct(suggested, "routing in-review ticket to requested reviewer without Orchestrator detour")
		}
		return direct("qa", "routing in-review ticket to QA without Orchestrator detour")
	case "no_work":
		if suggested != "" {
			return direct(suggested, "using no-work suggested_role without Orchestrator detour")
		}
		if target := strings.TrimSpace(d.Handoff.TargetRole); target != "" {
			return direct(target, "using no-work handoff.target_role without Orchestrator detour")
		}
		if target := strings.TrimSpace(d.Feedback.ForRole); target != "" {
			return direct(target, "using no-work feedback.for_role without Orchestrator detour")
		}
		if role := roleForNeedInManifest(in.Manifest, nextNeed); role != "" {
			if sameRole(d.Role, role) {
				return "", "deterministic", "next_need resolves to the current role; stopping direct no-work dispatch to avoid a same-role loop", "same-role next_need has no forward owner", true
			}
			return direct(role, "routing no-work disposition by next_need without Orchestrator detour")
		}
		if d.Role == "engineer" {
			return direct("ceo", "engineer found no actionable work; returning to planning without Orchestrator detour")
		}
		return "", "deterministic", "no work disposition stops dispatch", "no actionable work", true
	default:
		return "", "", "", "", false
	}
}

func sameRole(a, b string) bool {
	return normalize(a) == normalize(b)
}

func enforceDirectReviewProgression(in Input, candidateRole, kind, reason string) (nextRole, nextKind, nextReason, stop string) {
	candidateRole = strings.TrimSpace(candidateRole)
	if candidateRole == "" {
		return "", kind, reason, ""
	}
	source := in.Disposition
	sourceStatus := normalize(source.Status)
	if sourceStatus != "approved" && sourceStatus != "completed" {
		return candidateRole, kind, reason, ""
	}
	sourceRank, sourceOK := reviewLifecycleRank(source.Role)
	candidateRank, candidateOK := reviewLifecycleRank(candidateRole)
	if !sourceOK || !candidateOK || candidateRank > sourceRank {
		return candidateRole, kind, reason, ""
	}
	if forward := nextReviewLifecycleRole(in.Manifest, sourceRank); forward != "" {
		return forward,
			"deterministic",
			fmt.Sprintf("%s already completed review; routing forward to %s instead of repeating %s", source.Role, forward, candidateRole),
			""
	}
	return "",
		"deterministic",
		fmt.Sprintf("%s already completed review; no forward review role remains after %s", source.Role, candidateRole),
		"review chain complete"
}

func enforceOrchestratorCandidate(in Input, candidateRole, kind, reason string) (nextRole, nextKind, nextReason, stop string) {
	if sourceReleasePublicationBlocked(in) {
		return "",
			"deterministic",
			"source Release Manager disposition is release_blocked; stopping instead of routing back to product validation",
			"release publication blocked"
	}
	nextRole, nextKind, nextReason, stop = enforceReviewProgression(in, candidateRole, kind, reason)
	if stop != "" || strings.TrimSpace(nextRole) == "" {
		return nextRole, nextKind, nextReason, stop
	}
	return enforceQAInspectionBlock(in, nextRole, nextKind, nextReason)
}

func sourceReleasePublicationBlocked(in Input) bool {
	if in.SourceDisposition == nil || normalize(in.Disposition.Role) != "orchestrator" {
		return false
	}
	return releasePublicationBlocked(*in.SourceDisposition)
}

func releasePublicationBlocked(d orgstate.Disposition) bool {
	if normalize(d.Role) != "release_manager" {
		return false
	}
	if normalize(d.Status) != "blocked" {
		return false
	}
	nextNeed := normalize(d.NextNeed)
	if nextNeed == "release_blocked" || nextNeed == "release_publication" || nextNeed == "publication_blocked" {
		return true
	}
	text := normalize(strings.Join([]string{
		d.Reason,
		d.Handoff.Ask,
		d.Handoff.Context,
		d.Handoff.ExpectedOutput,
		d.Feedback.Summary,
		d.Feedback.RequestedChange,
	}, " "))
	return strings.Contains(text, "release_blocked") ||
		strings.Contains(text, "publication_blocked") ||
		strings.Contains(text, "no_remote") ||
		strings.Contains(text, "no remote") ||
		strings.Contains(text, "missing_remote") ||
		strings.Contains(text, "missing remote")
}

func enforceQAInspectionBlock(in Input, candidateRole, kind, reason string) (nextRole, nextKind, nextReason, stop string) {
	if in.SourceDisposition == nil || normalize(in.Disposition.Role) != "orchestrator" {
		return candidateRole, kind, reason, ""
	}
	source := *in.SourceDisposition
	if normalize(source.Role) != "qa" || normalize(source.Status) != "blocked" {
		return candidateRole, kind, reason, ""
	}
	if !qaBlockedOnRepoInspection(source) || !planningOrHygieneRole(candidateRole) {
		return candidateRole, kind, reason, ""
	}
	if _, ok := in.Manifest.Roles["qa"]; !ok {
		return candidateRole, kind, reason, ""
	}
	return "qa",
		"deterministic",
		"QA blocked on trigger-provided context instead of repository inspection; retrying QA with the existing repo-read tool contract before planning",
		""
}

func qaBlockedOnRepoInspection(d orgstate.Disposition) bool {
	text := normalize(strings.Join([]string{d.NextNeed, d.Reason, d.Handoff.Ask, d.Handoff.Context, d.Feedback.Summary}, " "))
	if strings.Contains(text, "source_code") ||
		strings.Contains(text, "implementation_source") ||
		strings.Contains(text, "trigger_context") ||
		strings.Contains(text, "not_provided") ||
		strings.Contains(text, "missing_context") ||
		strings.Contains(text, "cannot_inspect") {
		return true
	}
	return normalize(d.NextNeed) == "liveness"
}

func planningOrHygieneRole(role string) bool {
	switch normalize(role) {
	case "ceo", "coo", "cto_weekly", "cto", "architecture", "janitor":
		return true
	default:
		return false
	}
}

func enforceReviewProgression(in Input, candidateRole, kind, reason string) (nextRole, nextKind, nextReason, stop string) {
	candidateRole = strings.TrimSpace(candidateRole)
	if candidateRole == "" || in.SourceDisposition == nil || normalize(in.Disposition.Role) != "orchestrator" {
		return candidateRole, kind, reason, ""
	}
	source := *in.SourceDisposition
	sourceStatus := normalize(source.Status)
	if sourceStatus != "approved" && sourceStatus != "completed" {
		return candidateRole, kind, reason, ""
	}
	sourceRank, sourceOK := reviewLifecycleRank(source.Role)
	candidateRank, candidateOK := reviewLifecycleRank(candidateRole)
	if !sourceOK || !candidateOK || candidateRank > sourceRank {
		return candidateRole, kind, reason, ""
	}
	if forward := nextReviewLifecycleRole(in.Manifest, sourceRank); forward != "" {
		return forward,
			"deterministic",
			fmt.Sprintf("%s already completed review; routing forward to %s instead of repeating %s", source.Role, forward, candidateRole),
			""
	}
	return "",
		"deterministic",
		fmt.Sprintf("%s already completed review; no forward review role remains after %s", source.Role, candidateRole),
		"review chain complete"
}

func reviewLifecycleRank(role string) (int, bool) {
	switch normalize(role) {
	case "qa", "qa_review", "evidence_review", "review":
		return 0, true
	case "security", "security_review":
		return 1, true
	case "dogfood", "dogfood_validation", "e2e", "end_to_end":
		return 2, true
	case "dependency", "dependency_manager", "dependency_maintenance":
		return 3, true
	case "release", "release_manager", "release_review", "release_blocked":
		return 4, true
	default:
		return 0, false
	}
}

func nextReviewLifecycleRole(m *bundle.Manifest, sourceRank int) string {
	if m == nil {
		return ""
	}
	roles := []string{"qa", "security", "dogfood", "release-manager"}
	for i := sourceRank + 1; i < len(roles); i++ {
		if _, ok := m.Roles[roles[i]]; ok {
			return roles[i]
		}
	}
	return ""
}

func nextReviewLifecycleRoleForRole(m *bundle.Manifest, role string) string {
	rank, ok := reviewLifecycleRank(role)
	if !ok {
		return ""
	}
	return nextReviewLifecycleRole(m, rank)
}

func roleForNeedInManifest(m *bundle.Manifest, nextNeed string) string {
	role := roleForNeed(nextNeed)
	if role != "head-of-strategy" {
		return role
	}
	if m != nil {
		if _, ok := m.Roles["head-of-strategy"]; ok {
			return role
		}
	}
	return "ceo"
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
	case "architecture", "architecture_review", "architecture_blocker", "ticket", "ticket_shaping", "ticket_breakdown", "technical_ticket", "implementation_ticket":
		return "cto-weekly"
	case "exec_plan", "planning", "feature_contract", "scenario_schedule", "current_failing_scenario":
		return "coo"
	case "strategy_advice", "executive_narrative", "tradeoff_analysis", "goal_conflict":
		return "head-of-strategy"
	case "goal", "goals", "goal_decision", "vision", "scope_decision", "strategy", "strategy_review":
		return "ceo"
	case "dogfood", "dogfood_validation", "e2e", "end_to_end":
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
		return "coo"
	case "coo":
		return "cto-weekly"
	case "head-of-strategy":
		return "ceo"
	case "cto-weekly":
		return "engineer"
	case "engineer", "pipeline-fixer":
		return "qa"
	case "qa":
		return "security"
	case "security":
		return "dogfood"
	case "dogfood":
		return "release-manager"
	case "dependency-manager":
		return "release-manager"
	default:
		return ""
	}
}

func validateRole(m *bundle.Manifest, role string) (string, bool, string) {
	if strings.TrimSpace(role) == "" {
		return "", true, ""
	}
	if _, ok := m.Roles[role]; !ok {
		if alias, ok := manifestRoleAlias(m, role); ok {
			return alias, true, ""
		}
		normalized := normalize(role)
		for existing := range m.Roles {
			if normalize(existing) == normalized {
				return existing, true, ""
			}
		}
		return "", false, fmt.Sprintf("role %q is not defined in manifest", role)
	}
	return role, true, ""
}

func manifestRoleAlias(m *bundle.Manifest, role string) (string, bool) {
	if m == nil {
		return "", false
	}
	normalized := normalize(role)
	aliases := map[string][]string{
		"cto":          {"cto-weekly"},
		"architecture": {"cto-weekly"},
		"release":      {"release-manager"},
		"dependency":   {"dependency-manager"},
	}
	for _, candidate := range aliases[normalized] {
		if _, ok := m.Roles[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
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
			prev.SourceRole == d.Role &&
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
