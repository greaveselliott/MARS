/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	ticketstate "github.com/greaveselliott/mars/internal/tickets"
)

func checkTicketCreatePolicy(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession {
		return nil
	}
	var args ticketCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	args = withInferredTicketCreateScenarios(ctx, args)
	role := strings.ToLower(strings.TrimSpace(session.Role))
	switch role {
	case "ceo", "coo", "head-of-strategy":
		return fmt.Errorf("policy: %s cannot create implementation tickets; CEO/strategy define goals, COO writes the feature contract, and CTO owns ticket_create before Engineer delivery", role)
	}
	if err := checkTicketCreatePlanningOrder(root, session, hasSession, args); err != nil {
		return err
	}
	if err := checkBrowserFrameworkTicketCreatePolicy(root, session, hasSession, args); err != nil {
		return err
	}
	switch role {
	case "engineer":
		return checkEngineerTicketCreatePolicy(root, args)
	case "dogfood":
		return checkDogfoodTicketCreatePolicy(session, args)
	default:
		return nil
	}
}

func checkEngineerTicketCreatePolicy(root Root, args ticketCreateArgs) error {
	eligible, err := ticketstate.EligibleInProgress(root.Abs())
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before ticket_create: %w", err)
	}
	if len(eligible) == 0 {
		return nil
	}
	if isInterventionDebtTicket(args) || isDependencyTicketForEligibleWork(args, eligible) {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer cannot create ordinary backlog tickets while eligible in-progress tickets remain: %s. Complete the ticket or create a linked dependency/intervention-debt ticket with metadata.blocks.",
		joinTicketNames(eligible),
	)
}

func isInterventionDebtTicket(args ticketCreateArgs) bool {
	return strings.TrimSpace(args.Kind) == "intervention-debt" || strings.TrimSpace(args.WorkType) == "intervention-debt"
}

func checkTicketCreatePlanningOrder(root Root, session Session, hasSession bool, args ticketCreateArgs) error {
	if isInterventionDebtTicket(args) {
		return nil
	}
	if !repoFileExists(root, filepath.Join("docs", "exec-plans", "active", "current-operating-plan.md")) {
		return fmt.Errorf("policy: ticket_create requires docs/exec-plans/active/current-operating-plan.md first; planning order is exec plan, feature contract, ticket, delivery")
	}

	workType := normalizeWorkType(args.Kind, args.WorkType)
	if workType != "feature" {
		return nil
	}
	featureIDs := featureIDsFromScenarios(args.BDDScenarios)
	if len(featureIDs) == 0 {
		if hasSession && (strings.EqualFold(strings.TrimSpace(session.Role), "cto") || strings.EqualFold(strings.TrimSpace(session.Role), "cto-weekly")) {
			if next := pendingCTOHandoffRequiredScenarios(session); len(next) > 0 {
				return fmt.Errorf("policy: feature ticket_create is missing bdd_scenarios. %s", ctoTicketCreateGuidance(next))
			}
		}
		return fmt.Errorf("policy: feature ticket_create requires bdd_scenarios from an existing docs/features contract; planning order is exec plan, feature contract, ticket, delivery")
	}
	for _, id := range featureIDs {
		if !featureContractExists(root, id) {
			return fmt.Errorf("policy: feature ticket_create references %s before a docs/features/%s*.md contract exists; planning order is exec plan, feature contract, ticket, delivery", id, id)
		}
		scenarios, covered := featureScenarioCoverage(root, id)
		var alreadyCovered []string
		for _, scenario := range args.BDDScenarios {
			scenario = strings.ToUpper(strings.TrimSpace(scenario))
			if scenario == "" || featureIDFromScenarioIDMust(scenario) != id {
				continue
			}
			if covered[scenario] {
				alreadyCovered = append(alreadyCovered, scenario)
			}
		}
		if len(alreadyCovered) > 0 {
			sort.Strings(alreadyCovered)
			firstMissing := firstUncoveredFeatureScenarioFromCoverage(scenarios, covered)
			if hasSession && (strings.EqualFold(strings.TrimSpace(session.Role), "cto") || strings.EqualFold(strings.TrimSpace(session.Role), "cto-weekly")) && !featureHasCompletedValidationTicket(root, id) {
				required := firstSliceCTOHandoffRequiredScenarios(root, id, scenarios)
				selected := featureScenariosForID(args.BDDScenarios, id)
				if len(required) > 0 && len(selected) == 1 && selected[0] == required[0] {
					if featureHasExactFirstSliceTicket(root, id, required[0]) {
						return fmt.Errorf("policy: cto already has an Engineer-ready first-slice ticket for %s before first build/smoke proof. Hand off that exact ticket to Engineer instead of creating a duplicate. %s", id, ctoFirstSliceTicketCreateGuidance(required[:1]))
					}
					if err := checkCTOFirstSliceImplementationTicket(args, required[:1]); err != nil {
						return err
					}
					continue
				}
				return fmt.Errorf("policy: cto cannot create additional or grouped feature tickets for %s before first build/smoke proof. Hand off the exact first-slice ticket to Engineer, or create it if missing. %s", id, ctoFirstSliceTicketCreateGuidance(required))
			}
			if firstMissing != "" {
				return fmt.Errorf("policy: feature ticket_create cannot include already-covered scenario(s) %s for %s. Create the next ticket for %s only, or group it with later uncovered adjacent scenarios", strings.Join(alreadyCovered, ", "), id, firstMissing)
			}
			return fmt.Errorf("policy: feature ticket_create cannot include already-covered scenario(s) %s for %s; all contract scenarios appear to be ticketed already", strings.Join(alreadyCovered, ", "), id)
		}
		if hasSession && (strings.EqualFold(strings.TrimSpace(session.Role), "cto") || strings.EqualFold(strings.TrimSpace(session.Role), "cto-weekly")) && !featureHasCompletedValidationTicket(root, id) {
			required := firstSliceCTOHandoffRequiredScenarios(root, id, scenarios)
			if len(required) > 0 {
				selected := featureScenariosForID(args.BDDScenarios, id)
				if len(selected) != 1 || selected[0] != required[0] {
					return fmt.Errorf("policy: cto fresh first-proof ticket_create must create exactly one first-slice scenario for %s before broader backlog expansion. %s", id, ctoFirstSliceTicketCreateGuidance(required[:1]))
				}
				if err := checkCTOFirstSliceImplementationTicket(args, required[:1]); err != nil {
					return err
				}
			}
		}
		firstMissing := firstUncoveredFeatureScenario(root, id)
		if firstMissing != "" && !scenarioListContains(args.BDDScenarios, firstMissing) {
			return fmt.Errorf("policy: feature ticket_create must start with the earliest uncovered scenario %s for %s. Create the next ticket from that scenario, or include it in this scenario group before later scenarios", firstMissing, id)
		}
	}
	return nil
}

func ctoTicketCreateGuidance(next []string) string {
	nextText := strings.Join(next, ", ")
	if nextText == "" {
		nextText = "the next uncovered product scenario"
	}
	return fmt.Sprintf("Create the next valid implementation ticket with ticket_create using bdd_scenarios:%s (JSON array), covering %s, or group adjacent bounded product scenarios in one ticket when that is the clearer post-proof slice. Required follow-up sequence: git_status -> git_commit -> job_disposition_record with next_need implementation and suggested_role engineer.", quoteStringArray(next), nextText)
}

func ctoFirstSliceTicketCreateGuidance(next []string) string {
	nextText := strings.Join(next, ", ")
	if nextText == "" {
		nextText = "the current failing product scenario"
	}
	return fmt.Sprintf("Create exactly one first-slice implementation ticket with ticket_create using bdd_scenarios:%s (JSON array), covering %s. Do not retry later-scenario titles or grouped scenario lists before first proof; the ticket title/body must describe this first scenario only. Required follow-up sequence: git_status -> git_commit -> job_disposition_record with next_need implementation and suggested_role engineer.", quoteStringArray(next), nextText)
}

func checkCTOFirstSliceImplementationTicket(args ticketCreateArgs, next []string) error {
	if ticketCreateDescribesExecutableImplementation(args) {
		return nil
	}
	return fmt.Errorf("policy: cto first-slice ticket must describe executable product implementation, not only brief verification, planning, or understanding evidence. %s", ctoFirstSliceTicketCreateGuidance(next))
}

func ticketCreateDescribesExecutableImplementation(args ticketCreateArgs) bool {
	surface := normalizeCapabilitySurface(ticketCreatePolicySurface(args))
	for _, marker := range []string{
		"implement",
		"implementation",
		"build",
		"create",
		"render",
		"ship",
		"wire",
		"add",
		"deliver",
		"make",
		"code",
		"playable",
		"gameplay",
		"interactive",
		"browser",
		"html",
		"css",
		"javascript",
		"component",
		"view",
		"screen",
		"endpoint",
		"handler",
		"server",
		"api",
	} {
		if strings.Contains(surface, marker) {
			return true
		}
	}
	return false
}

func featureIDsFromScenarios(scenarios []string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, scenario := range scenarios {
		id, ok := featureIDFromScenarioID(scenario)
		if !ok {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func featureIDFromScenarioIDMust(scenario string) string {
	id, _ := featureIDFromScenarioID(scenario)
	return id
}

func featureIDFromScenarioID(scenario string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(strings.ToUpper(scenario)), "-")
	if len(parts) < 2 || parts[0] != "F" || parts[1] == "" {
		return "", false
	}
	return "F-" + parts[1], true
}

func featureContractExists(root Root, featureID string) bool {
	return featureContractPath(root, featureID) != ""
}

func featureContractPath(root Root, featureID string) string {
	matches, err := root.RepoFS().Glob(filepath.Join("docs", "features", featureID+"*.md"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	return filepath.Join(root.Abs(), filepath.FromSlash(matches[0]))
}

func firstUncoveredFeatureScenario(root Root, featureID string) string {
	scenarios, covered := featureScenarioCoverage(root, featureID)
	return firstUncoveredFeatureScenarioFromCoverage(scenarios, covered)
}

func firstUncoveredFeatureScenarioFromCoverage(scenarios []string, covered map[string]bool) string {
	if len(scenarios) == 0 {
		return ""
	}
	for _, scenario := range scenarios {
		if !covered[scenario] {
			return scenario
		}
	}
	return ""
}

func featureScenarioCoverage(root Root, featureID string) ([]string, map[string]bool) {
	covered := map[string]bool{}
	featurePath := featureContractPath(root, featureID)
	if featurePath == "" {
		return nil, covered
	}
	data, err := root.RepoFS().ReadFile(relPathFromAbs(root, featurePath))
	if err != nil {
		return nil, covered
	}
	scenarios := orderedFeatureScenarioIDs(string(data))
	if len(scenarios) == 0 {
		return nil, covered
	}
	tickets, err := ticketstate.List(root.Abs())
	if err == nil {
		for _, t := range tickets {
			if t.Kind == "intervention-debt" {
				continue
			}
			if strings.TrimSpace(t.RelPath) == "" {
				continue
			}
			data, err := root.RepoFS().ReadFile(filepath.FromSlash(t.RelPath))
			if err != nil {
				continue
			}
			for _, scenario := range orderedFeatureScenarioIDs(string(data)) {
				covered[scenario] = true
			}
		}
	}
	return scenarios, covered
}

func featureContractIDs(root Root) []string {
	matches, err := root.RepoFS().Glob(filepath.Join("docs", "features", "F-*.md"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, match := range matches {
		if data, err := root.RepoFS().ReadFile(filepath.FromSlash(match)); err == nil && featureContractSuperseded(string(data)) {
			continue
		}
		id, ok := featureContractIDFromName(filepath.Base(match))
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func orderedFeatureScenarioIDs(content string) []string {
	matches := toolsFeatureScenarioIDPattern.FindAllString(strings.ToUpper(content), -1)
	seen := map[string]bool{}
	var out []string
	for _, match := range matches {
		if seen[match] {
			continue
		}
		seen[match] = true
		out = append(out, match)
	}
	return out
}

func scenarioListContains(values []string, want string) bool {
	want = strings.ToUpper(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToUpper(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func isDependencyTicketForEligibleWork(args ticketCreateArgs, eligible []ticketstate.Ticket) bool {
	if strings.TrimSpace(args.DedupeKey) == "" {
		return false
	}
	values := []string{
		args.Metadata["blocks"],
		args.Metadata["blocked_ticket"],
		args.Metadata["blocked_by_target"],
	}
	for _, value := range values {
		for _, t := range eligible {
			if ticketMetadataMentions(value, t) {
				return true
			}
		}
	}
	return false
}

func ticketMetadataMentions(value string, ticket ticketstate.Ticket) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, strings.ToLower(ticket.ID)) ||
		strings.Contains(value, strings.ToLower(ticket.Name))
}

func checkDogfoodTicketCreatePolicy(session Session, args ticketCreateArgs) error {
	if session.ToolCounts == nil {
		return nil
	}
	severity := strings.ToLower(strings.TrimSpace(args.Priority))
	if severity == "" {
		severity = "medium"
	}
	group := dogfoodTicketGroup(args)
	dedupe := strings.TrimSpace(args.DedupeKey)
	totalKey := "ticket_create:dogfood:total"
	severityKey := "ticket_create:dogfood:severity:" + severity
	groupKey := "ticket_create:dogfood:group:" + group
	dedupeKey := "ticket_create:dogfood:dedupe:" + dedupe
	if dedupe != "" && session.ToolCounts[dedupeKey] > 0 {
		return fmt.Errorf("policy: dogfood ticket_create repeated dedupe key %q in one run; update the existing ticket instead", dedupe)
	}
	if session.ToolCounts[totalKey] >= dogfoodTicketCreateLimitTotal {
		return fmt.Errorf("policy: dogfood ticket_create capped at %d tickets per run; group remaining findings behind the highest-severity dedupe keys", dogfoodTicketCreateLimitTotal)
	}
	if session.ToolCounts[severityKey] >= dogfoodTicketCreateLimitPerSeverity {
		return fmt.Errorf("policy: dogfood ticket_create capped at %d %s-severity tickets per run", dogfoodTicketCreateLimitPerSeverity, severity)
	}
	if session.ToolCounts[groupKey] >= dogfoodTicketCreateLimitPerGroup {
		return fmt.Errorf("policy: dogfood ticket_create capped at %d tickets for group %q per run", dogfoodTicketCreateLimitPerGroup, group)
	}
	session.ToolCounts[totalKey]++
	session.ToolCounts[severityKey]++
	session.ToolCounts[groupKey]++
	if dedupe != "" {
		session.ToolCounts[dedupeKey]++
	}
	return nil
}

func checkDogfoodFindingCommitPolicy(ctx context.Context, root Root, session Session, hasSession bool, toolName string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "dogfood" {
		return nil
	}
	if dogfoodToolAllowedWithUncommittedFindings(toolName) {
		return nil
	}
	if session.ToolCounts != nil && session.ToolCounts["ticket_create:dogfood:total"] > 0 {
		return fmt.Errorf(
			"policy: dogfood already created a target-owned finding ticket in this run. Commit the ticket, call git_push if a remote exists, and record job_disposition_record before continuing validation or creating another ticket",
		)
	}
	paths, err := uncommittedTicketMarkdownPaths(ctx, root)
	if err != nil || len(paths) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: dogfood has uncommitted target-owned finding ticket(s): %s. Run git_status, commit the ticket with git_commit, call git_push if a remote exists, then record job_disposition_record before continuing validation or creating another ticket",
		summarizeChangedFiles(paths),
	)
}

func dogfoodToolAllowedWithUncommittedFindings(toolName string) bool {
	switch toolName {
	case "file_read", "git_status", "git_diff", "git_commit", "git_push", "job_disposition_record":
		return true
	default:
		return false
	}
}

func uncommittedTicketMarkdownPaths(ctx context.Context, root Root) ([]string, error) {
	files, err := changedFiles(ctx, root)
	if err != nil {
		return nil, err
	}
	var tickets []string
	for _, file := range files {
		if ticketMarkdownLifecyclePath(file) {
			tickets = append(tickets, file)
		}
	}
	return tickets, nil
}

func ticketMarkdownLifecyclePath(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if rel == "." || !strings.HasSuffix(strings.ToLower(rel), ".md") {
		return false
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 4 || parts[0] != "docs" || parts[1] != "tickets" {
		return false
	}
	switch parts[2] {
	case "backlog", "in-progress", "in-review", "done":
		return true
	default:
		return false
	}
}

func dogfoodTicketGroup(args ticketCreateArgs) string {
	if dedupe := strings.TrimSpace(args.DedupeKey); dedupe != "" {
		parts := strings.Split(dedupe, ":")
		if len(parts) >= 5 {
			return strings.Join(parts[:5], ":")
		}
		return dedupe
	}
	if args.Metadata != nil {
		if category := strings.TrimSpace(args.Metadata["category"]); category != "" {
			return category
		}
	}
	if title := strings.TrimSpace(args.Title); title != "" {
		return slugify(title)
	}
	return "unknown"
}

func joinTicketNames(tickets []ticketstate.Ticket) string {
	names := make([]string, 0, len(tickets))
	for _, t := range tickets {
		names = append(names, t.Name)
	}
	return strings.Join(names, ", ")
}

func checkEngineerClaimBeforeProductMutation(ctx context.Context, root Root, session Session, hasSession bool, toolName string, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if !engineerToolRequiresClaim(toolName, raw) {
		return nil
	}
	if engineerToolIsTicketOnlyMutation(toolName, raw) {
		return nil
	}
	if toolName == "git_commit" && gitCommitTouchesOnlyWorkspaceNoise(ctx, root, raw) {
		return nil
	}
	if toolName == "git_commit" && worktreeHasInProgressToDoneTicketMove(ctx, root) {
		return nil
	}
	if engineerCompletedTicketThisRun(session) {
		return fmt.Errorf("policy: engineer already moved product ticket %s to docs/tickets/done in this run. Do not claim or mutate another ticket in the same job; commit any remaining lifecycle changes, git_push if a remote exists, then record job_disposition_record with ticket_id %s and next_need qa_review", engineerCompletedTicketID(session), engineerCompletedTicketID(session))
	}
	if err := checkEngineerPinnedReworkBeforeProductMutation(root, session, toolName); err != nil {
		return err
	}
	inProgress, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusInProgress)
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before %s: %w", toolName, err)
	}
	inProgress = ordinaryProductTickets(inProgress)
	if len(inProgress) > 0 {
		return nil
	}
	backlog, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusBacklog)
	if err != nil {
		return fmt.Errorf("policy: inspect backlog tickets before %s: %w", toolName, err)
	}
	backlog = ordinaryProductTickets(backlog)
	if len(backlog) == 0 {
		rework, err := engineerReworkTickets(root, session)
		if err != nil {
			return fmt.Errorf("policy: inspect review rework tickets before %s: %w", toolName, err)
		}
		if len(rework) == 0 {
			return nil
		}
		return fmt.Errorf(
			"policy: engineer must reopen product ticket %s before %s mutates product files; move %s from %s to docs/tickets/in-progress/ with git mv, commit the rework claim, then continue",
			rework[0].ID,
			toolName,
			rework[0].ID,
			rework[0].RelPath,
		)
	}
	return fmt.Errorf(
		"policy: engineer must claim a product ticket before %s mutates product files; move %s from %s to docs/tickets/in-progress/ with git mv, commit the claim, then continue",
		toolName,
		backlog[0].ID,
		backlog[0].RelPath,
	)
}

func checkEngineerShellExecBeforeTicketClaim(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage, generatedArtifactCleanup bool) error {
	if generatedArtifactCleanup || !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if err := checkEngineerPinnedReworkBeforeShellExec(root, session, raw); err != nil {
		return err
	}
	if shellExecMovesTicketToInProgress(raw) {
		if worktreeHasInProgressToDoneTicketMove(ctx, root) {
			return fmt.Errorf("policy: engineer has already moved a product ticket to docs/tickets/done but the lifecycle move is not committed. Run git_status, git_commit the ticket lifecycle move, git_push if a remote exists, then record job_disposition_record with next_need qa_review before claiming another ticket")
		}
		if engineerCompletedTicketThisRun(session) {
			return fmt.Errorf("policy: engineer already completed product ticket %s in this run. Do not claim another ticket in the same job; record job_disposition_record with ticket_id %s and next_need qa_review after the lifecycle commit", engineerCompletedTicketID(session), engineerCompletedTicketID(session))
		}
		return nil
	}
	inProgress, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusInProgress)
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before shell_exec: %w", err)
	}
	inProgress = ordinaryProductTickets(inProgress)
	if len(inProgress) > 0 {
		return nil
	}
	backlog, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusBacklog)
	if err != nil {
		return fmt.Errorf("policy: inspect backlog tickets before shell_exec: %w", err)
	}
	backlog = ordinaryProductTickets(backlog)
	if worktreeHasInProgressToDoneTicketMove(ctx, root) {
		return fmt.Errorf("policy: engineer has an uncommitted product ticket lifecycle move to docs/tickets/done. Commit that move and record job_disposition_record with next_need qa_review before running more shell commands or claiming another ticket")
	}
	if engineerCompletedTicketThisRun(session) {
		return fmt.Errorf("policy: engineer already completed product ticket %s in this run. Do not continue into another ticket; record job_disposition_record with ticket_id %s and next_need qa_review", engineerCompletedTicketID(session), engineerCompletedTicketID(session))
	}
	if len(backlog) == 0 {
		rework, err := engineerReworkTickets(root, session)
		if err != nil {
			return fmt.Errorf("policy: inspect review rework tickets before shell_exec: %w", err)
		}
		if len(rework) == 0 {
			return nil
		}
		return fmt.Errorf(
			"policy: engineer must reopen %s before running shell_exec for rework; run shell_exec with argv [\"git\", \"mv\", %q, \"docs/tickets/in-progress/\"] and then git_commit \"chore(tickets): reopen %s for rework\" before discovery, validation, or implementation shell commands",
			rework[0].ID,
			rework[0].RelPath,
			rework[0].ID,
		)
	}
	return fmt.Errorf(
		"policy: engineer must claim %s before running shell_exec; run shell_exec with argv [\"git\", \"mv\", %q, \"docs/tickets/in-progress/\"] and then git_commit \"chore(tickets): claim %s\" before discovery, validation, or implementation shell commands",
		backlog[0].ID,
		backlog[0].RelPath,
		backlog[0].ID,
	)
}

func checkEngineerPinnedReworkBeforeProductMutation(root Root, session Session, toolName string) error {
	targetID := engineerReworkTicketIDFromTrigger(session.Trigger)
	if targetID == "" {
		return nil
	}
	ticket, err := engineerPinnedReworkTicket(root, targetID)
	if err != nil {
		return err
	}
	if ticket.Status == ticketstate.StatusInProgress {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer is pinned to review rework ticket %s; reopen it before %s mutates product files by moving %s from %s to docs/tickets/in-progress/ with git mv, committing the rework claim, then continuing",
		targetID,
		toolName,
		targetID,
		ticket.RelPath,
	)
}

func checkEngineerPinnedReworkBeforeShellExec(root Root, session Session, raw json.RawMessage) error {
	targetID := engineerReworkTicketIDFromTrigger(session.Trigger)
	if targetID == "" {
		return nil
	}
	if moveID, ok := shellExecTicketMoveToInProgressID(raw); ok {
		if strings.EqualFold(moveID, targetID) {
			return nil
		}
		return fmt.Errorf("policy: engineer is pinned to review rework ticket %s and cannot claim or reopen unrelated ticket %s in this job", targetID, moveID)
	}
	ticket, err := engineerPinnedReworkTicket(root, targetID)
	if err != nil {
		return err
	}
	if ticket.Status == ticketstate.StatusInProgress {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer must reopen %s before running shell_exec for review rework; run shell_exec with argv [\"git\", \"mv\", %q, \"docs/tickets/in-progress/\"] and then git_commit \"chore(tickets): reopen %s for rework\" before discovery, validation, or implementation shell commands",
		targetID,
		ticket.RelPath,
		targetID,
	)
}

func engineerCompletedTicketThisRun(session Session) bool {
	return session.ToolCounts != nil && session.ToolCounts[ticketDoneMoveSuccessKey] > 0
}

func engineerCompletedTicketID(session Session) string {
	if session.ToolState != nil {
		if id := strings.TrimSpace(session.ToolState[ticketDoneMoveLastIDKey]); id != "" {
			return id
		}
	}
	return "the current ticket"
}

func engineerToolRequiresClaim(toolName string, raw json.RawMessage) bool {
	switch toolName {
	case "file_write", "dependency_sync", "mars_cli", "git_commit":
		return true
	case "shell_exec":
		return !shellExecReadOnly(raw)
	default:
		return false
	}
}

func engineerToolIsTicketOnlyMutation(toolName string, raw json.RawMessage) bool {
	switch toolName {
	case "file_write":
		args, err := decodeFileWriteArgs(raw)
		if err != nil {
			return false
		}
		rel := strings.ToLower(cleanRepoPath(args.Path))
		return rel == "docs/tickets/readme.md" || strings.HasPrefix(rel, "docs/tickets/")
	case "shell_exec":
		return shellExecMovesTicketToInProgress(raw)
	default:
		return false
	}
}

func gitCommitTouchesOnlyWorkspaceNoise(ctx context.Context, root Root, raw json.RawMessage) bool {
	var args gitCommitArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return false
	}
	if len(args.Paths) > 0 {
		for _, p := range args.Paths {
			if !IsWorkspaceNoisePath(p) {
				return false
			}
		}
		return true
	}
	files, err := changedFiles(ctx, root)
	if err != nil || len(dispositionBlockingFiles(files)) > 0 {
		return false
	}
	noise, err := dirtyWorkspaceNoisePaths(ctx, root)
	return err == nil && len(noise) > 0
}

func engineerReworkTickets(root Root, session Session) ([]ticketstate.Ticket, error) {
	if targetID := engineerReworkTicketIDFromTrigger(session.Trigger); targetID != "" {
		ticket, err := engineerPinnedReworkTicket(root, targetID)
		if err != nil {
			return nil, err
		}
		return []ticketstate.Ticket{ticket}, nil
	}
	var out []ticketstate.Ticket
	for _, status := range []string{ticketstate.StatusInReview, ticketstate.StatusDone} {
		tickets, err := ticketstate.ListStatus(root.Abs(), status)
		if err != nil {
			return nil, err
		}
		out = append(out, ordinaryProductTickets(tickets)...)
	}
	return out, nil
}

func engineerPinnedReworkTicket(root Root, targetID string) (ticketstate.Ticket, error) {
	targetID = strings.ToUpper(strings.TrimSpace(targetID))
	if targetID == "" {
		return ticketstate.Ticket{}, fmt.Errorf("dispatch-named rework ticket is empty")
	}
	tickets, err := ticketstate.List(root.Abs())
	if err != nil {
		return ticketstate.Ticket{}, fmt.Errorf("inspect dispatch-named rework ticket %s: %w", targetID, err)
	}
	for _, ticket := range ordinaryProductTickets(tickets) {
		if strings.EqualFold(strings.TrimSpace(ticket.ID), targetID) {
			return ticket, nil
		}
	}
	return ticketstate.Ticket{}, fmt.Errorf("dispatch-named rework ticket %s not found", targetID)
}

func engineerReworkTicketIDFromTrigger(raw string) string {
	var trigger struct {
		Type              string `json:"type"`
		TargetRole        string `json:"target_role"`
		SourceDisposition struct {
			Status   string `json:"status"`
			NextNeed string `json:"next_need"`
			TicketID string `json:"ticket_id"`
		} `json:"source_disposition"`
	}
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		return ""
	}
	if !strings.EqualFold(strings.TrimSpace(trigger.Type), "dispatch") {
		return ""
	}
	if target := strings.ToLower(strings.TrimSpace(trigger.TargetRole)); target != "" && target != "engineer" {
		return ""
	}
	source := trigger.SourceDisposition
	if !strings.EqualFold(strings.TrimSpace(source.Status), "changes_requested") {
		return ""
	}
	if next := strings.ToLower(strings.TrimSpace(source.NextNeed)); next != "" && next != "implementation_rework" && next != "implementation" && next != "fix" {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(source.TicketID))
}

func worktreeHasInProgressToDoneTicketMove(ctx context.Context, root Root) bool {
	tr, err := runGit(ctx, root, "status", "--porcelain=v1", "--", "docs/tickets")
	if err != nil || tr.ExitCode != 0 {
		return false
	}
	for _, line := range strings.Split(tr.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, " -> ") {
			continue
		}
		parts := strings.SplitN(line, " -> ", 2)
		if len(parts) != 2 {
			continue
		}
		source := strings.TrimSpace(parts[0])
		if len(source) > 3 {
			source = strings.TrimSpace(source[3:])
		}
		dest := strings.TrimSpace(parts[1])
		if ticketMoveTargetsDone(source, dest) {
			_, sourceState, sourceOK := ticketLifecyclePathIdentity(cleanRepoPath(cleanShellPathToken(source)))
			if sourceOK && sourceState == "in-progress" {
				return true
			}
		}
	}
	return false
}

func ordinaryProductTickets(tickets []ticketstate.Ticket) []ticketstate.Ticket {
	var out []ticketstate.Ticket
	for _, t := range tickets {
		if strings.EqualFold(strings.TrimSpace(t.Kind), "intervention-debt") || strings.EqualFold(strings.TrimSpace(t.WorkType), "intervention-debt") {
			continue
		}
		out = append(out, t)
	}
	return out
}

func checkTicketFileWritePolicy(root Root, rel string) error {
	rel = cleanRepoPath(rel)
	lowerRel := strings.ToLower(rel)
	if rel == "" || lowerRel == "docs/tickets/readme.md" {
		return nil
	}
	if !strings.HasPrefix(lowerRel, "docs/tickets/") || !strings.HasSuffix(lowerRel, ".md") {
		return nil
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 4 || !isTicketLifecycleDir(parts[2]) {
		return fmt.Errorf("policy: ticket markdown must live under docs/tickets/backlog, docs/tickets/in-progress, docs/tickets/in-review, or docs/tickets/done; use ticket_create for new tickets instead of file_write to %s", rel)
	}
	if _, err := root.RepoFS().Stat(rel); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("policy: new ticket files must be created with ticket_create so numbering, backlog placement, and dedupe are enforced; attempted file_write to %s", rel)
	}
	return nil
}

func checkTicketDoneContentPolicy(root Root, rel, content string) error {
	rel = cleanRepoPath(rel)
	_, state, ok := ticketLifecyclePathIdentity(rel)
	if !ok || state != "done" {
		return nil
	}
	missing := missingFeatureTicketEvidence(parseTicketPolicyFrontmatter(content))
	if len(missing) == 0 {
		if dupes := ticketLifecycleDuplicatesOutsideState(root, rel, "done"); len(dupes) > 0 {
			return fmt.Errorf(
				"policy: feature ticket %s cannot be copied into docs/tickets/done while the same ticket still exists at %s; update evidence in the current lifecycle file, then use git mv to move that exact file to done",
				filepath.Base(rel),
				strings.Join(dupes, ", "),
			)
		}
		return nil
	}
	return fmt.Errorf(
		"policy: feature ticket %s cannot be saved in docs/tickets/done without BDD scenario evidence: missing %s; update the ticket evidence before moving or saving it as done",
		filepath.Base(rel),
		strings.Join(missing, ", "),
	)
}

func checkEngineerTicketEvidenceWriteRequiresValidation(root Root, session Session, hasSession bool, rel, content string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	rel = cleanRepoPath(rel)
	_, state, ok := ticketLifecyclePathIdentity(rel)
	if !ok || state != "in-progress" {
		return nil
	}
	frontmatter := parseTicketPolicyFrontmatter(content)
	if !ticketEvidencePopulated(frontmatter) {
		return nil
	}
	if engineerInValidationFailedTestBuildLane(session) || engineerInValidationFailedRuntimeLane(session) {
		return nil
	}
	if engineerInValidatedBrowserCompletionPhase(root, session) {
		if blockers := engineerBrowserFrameworkCompletionBlockers(root, session); len(blockers) > 0 {
			return fmt.Errorf(
				"policy: engineer cannot populate ticket evidence for browser-framework work in %s yet: %s. Add or fix package build/browser validation, run it successfully in this job, then update evidence_links with the concrete commands",
				rel,
				strings.Join(blockers, "; "),
			)
		}
		return nil
	}
	if blockers := staticBrowserCompletionBlockers(root, session); len(blockers) > 0 {
		return fmt.Errorf(
			"policy: engineer cannot populate ticket evidence for static browser work in %s yet: %s. `node --check` is useful syntax evidence, but it does not prove the page can be served or loaded",
			rel,
			strings.Join(blockers, "; "),
		)
	}
	if engineerInValidatedPhase(session) {
		return nil
	}
	return fmt.Errorf("policy: engineer cannot populate ticket evidence_links or verified_by in %s before successful validation in this job; run go test, a build, or a runtime command that exercises the BDD scenario, then update the ticket with exact evidence", rel)
}

func isTicketLifecycleDir(dir string) bool {
	switch dir {
	case "backlog", "in-progress", "in-review", "done":
		return true
	default:
		return false
	}
}

func checkShellTicketDoneEvidencePolicy(ctx context.Context, root Root, raw json.RawMessage) error {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
	}
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellFieldsPreserveCase(args.ShellCommand)
	}
	if copies := ticketDoneCopySources(fields); len(copies) > 0 {
		return fmt.Errorf(
			"policy: feature ticket %s cannot be copied into docs/tickets/done; update evidence in the current lifecycle file, then use git mv so only one lifecycle copy exists",
			filepath.Base(copies[0]),
		)
	}
	moveSources := ticketDoneMoveSources(fields)
	if len(moveSources) > 0 {
		if err := checkTicketDoneMoveHasOnlyTicketChanges(ctx, root); err != nil {
			return err
		}
	}
	for _, source := range moveSources {
		data, err := root.RepoFS().ReadFile(source)
		if err != nil {
			continue
		}
		missing := missingFeatureTicketEvidence(parseTicketPolicyFrontmatter(string(data)))
		if len(missing) == 0 {
			continue
		}
		return fmt.Errorf(
			"policy: feature ticket %s cannot move to docs/tickets/done without BDD scenario evidence: missing %s; update evidence_links and verified_by before moving it",
			filepath.Base(source),
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func checkTicketDoneMoveHasOnlyTicketChanges(ctx context.Context, root Root) error {
	files, err := changedFiles(ctx, root)
	if err != nil {
		return nil
	}
	files = dispositionBlockingFiles(files)
	var nonTicket []string
	for _, file := range files {
		rel := cleanRepoPath(file)
		if rel == "" || strings.HasPrefix(rel, "docs/tickets/") {
			continue
		}
		nonTicket = append(nonTicket, rel)
	}
	if len(nonTicket) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: ticket lifecycle move to docs/tickets/done requires product/source changes to be committed first; uncommitted non-ticket files remain: %s. Commit implementation, tests, docs, and package files with git_commit before moving the ticket to done",
		summarizeChangedFiles(nonTicket),
	)
}

func ticketDoneCopySources(fields []string) []string {
	var sources []string
	for i, field := range fields {
		if strings.ToLower(filepathBase(field)) != "cp" {
			continue
		}
		if source, dest, ok := ticketMoveOperands(fields[i+1:]); ok && ticketMoveTargetsDone(source, dest) {
			sources = append(sources, cleanShellPathToken(source))
		}
	}
	return sources
}

func ticketDoneMoveSources(fields []string) []string {
	var sources []string
	for i, field := range fields {
		switch strings.ToLower(filepathBase(field)) {
		case "git":
			if i+1 >= len(fields) || strings.ToLower(strings.TrimSpace(fields[i+1])) != "mv" {
				continue
			}
			if source, dest, ok := ticketMoveOperands(fields[i+2:]); ok && ticketMoveTargetsDone(source, dest) {
				sources = append(sources, cleanShellPathToken(source))
			}
		case "mv":
			if i > 0 && strings.ToLower(filepathBase(fields[i-1])) == "git" {
				continue
			}
			if source, dest, ok := ticketMoveOperands(fields[i+1:]); ok && ticketMoveTargetsDone(source, dest) {
				sources = append(sources, cleanShellPathToken(source))
			}
		}
	}
	return sources
}

func shellExecMovesTicketToInProgress(raw json.RawMessage) bool {
	_, ok := shellExecTicketMoveToInProgressID(raw)
	return ok
}

func shellExecTicketMoveToInProgressID(raw json.RawMessage) (string, bool) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return "", false
	}
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellFields(args.ShellCommand)
	}
	for i, field := range fields {
		if filepathBase(field) != "git" {
			continue
		}
		if i+1 >= len(fields) || strings.ToLower(strings.TrimSpace(fields[i+1])) != "mv" {
			continue
		}
		source, dest, ok := ticketMoveOperands(fields[i+2:])
		if ok && ticketMoveTargetsInProgress(source, dest) {
			id, _, idOK := ticketLifecyclePathIdentity(cleanRepoPath(cleanShellPathToken(source)))
			return id, idOK
		}
	}
	return "", false
}

func shellExecMovesInProgressTicketToDone(raw json.RawMessage) bool {
	_, ok := shellExecInProgressToDoneTicketID(raw)
	return ok
}

func shellExecInProgressToDoneTicketID(raw json.RawMessage) (string, bool) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return "", false
	}
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellFields(args.ShellCommand)
	}
	for i, field := range fields {
		if filepathBase(field) != "git" {
			continue
		}
		if i+1 >= len(fields) || strings.ToLower(strings.TrimSpace(fields[i+1])) != "mv" {
			continue
		}
		source, dest, ok := ticketMoveOperands(fields[i+2:])
		if !ok || !ticketMoveTargetsDone(source, dest) {
			continue
		}
		id, state, ok := ticketLifecyclePathIdentity(cleanRepoPath(cleanShellPathToken(source)))
		if ok && state == "in-progress" {
			return id, true
		}
	}
	return "", false
}

func ticketMoveOperands(fields []string) (source, dest string, ok bool) {
	operands := make([]string, 0, 2)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || strings.HasPrefix(field, "-") {
			continue
		}
		operands = append(operands, field)
		if len(operands) == 2 {
			break
		}
	}
	if len(operands) < 2 {
		return "", "", false
	}
	return operands[0], operands[1], true
}

func ticketMoveTargetsInProgress(source, dest string) bool {
	source = cleanRepoPath(cleanShellPathToken(source))
	dest = cleanRepoPath(cleanShellPathToken(dest))
	if dest == filepath.ToSlash(filepath.Join("docs", "tickets", "in-progress")) {
		_, sourceState, sourceOK := ticketLifecyclePathIdentity(source)
		return sourceOK && (sourceState == "backlog" || sourceState == "in-review" || sourceState == "done")
	}
	_, destState, destOK := ticketLifecyclePathIdentity(dest)
	_, sourceState, sourceOK := ticketLifecyclePathIdentity(source)
	return destOK && destState == "in-progress" && sourceOK && (sourceState == "backlog" || sourceState == "in-review" || sourceState == "done")
}

func ticketMoveTargetsDone(source, dest string) bool {
	source = cleanRepoPath(cleanShellPathToken(source))
	dest = cleanRepoPath(cleanShellPathToken(dest))
	if dest == filepath.ToSlash(filepath.Join("docs", "tickets", "done")) {
		_, sourceState, sourceOK := ticketLifecyclePathIdentity(source)
		return sourceOK && sourceState != "done"
	}
	_, destState, destOK := ticketLifecyclePathIdentity(dest)
	_, sourceState, sourceOK := ticketLifecyclePathIdentity(source)
	return destOK && destState == "done" && sourceOK && sourceState != "done"
}

func parseTicketPolicyFrontmatter(text string) map[string]string {
	fields := make(map[string]string)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields
	}
	currentKey := ""
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			break
		}
		if strings.HasPrefix(trimmed, "- ") && currentKey != "" {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if item == "" {
				continue
			}
			if fields[currentKey] == "" {
				fields[currentKey] = item
			} else {
				fields[currentKey] += "\n" + item
			}
			continue
		}
		if trimmed == "" || !strings.Contains(line, ":") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		currentKey = strings.TrimSpace(key)
		fields[currentKey] = unquoteYAMLString(strings.TrimSpace(value))
	}
	return fields
}

func missingFeatureTicketEvidence(frontmatter map[string]string) []string {
	workType := strings.Trim(strings.ToLower(frontmatter["work_type"]), `"'`)
	endToEndEvidence := strings.Trim(strings.ToLower(frontmatter["end_to_end_evidence"]), `"'`)
	if workType != "feature" && endToEndEvidence != "required" {
		return nil
	}
	var missing []string
	if workType == "feature" && endToEndEvidence != "required" {
		missing = append(missing, "end_to_end_evidence: required")
	}
	if workType == "feature" && ticketEvidenceFieldEmpty(frontmatter["bdd_scenarios"]) {
		missing = append(missing, "bdd_scenarios")
	}
	if ticketEvidenceFieldEmpty(frontmatter["evidence_links"]) {
		missing = append(missing, "evidence_links")
	}
	if ticketEvidenceFieldEmpty(frontmatter["verified_by"]) {
		missing = append(missing, "verified_by")
	}
	return missing
}

func ticketEvidencePopulated(frontmatter map[string]string) bool {
	return !ticketEvidenceFieldEmpty(frontmatter["evidence_links"]) ||
		!ticketEvidenceFieldEmpty(frontmatter["verified_by"])
}

func ticketEvidenceFieldEmpty(value string) bool {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	switch strings.ToLower(value) {
	case "", "[]", "none", "null", "nil", "tbd", "todo":
		return true
	default:
		return false
	}
}

func ticketLifecycleDuplicatesOutsideState(root Root, rel, state string) []string {
	id, _, ok := ticketLifecyclePathIdentity(rel)
	if !ok {
		return nil
	}
	var dupes []string
	for _, candidateState := range []string{"backlog", "in-progress", "in-review", "done"} {
		if candidateState == state {
			continue
		}
		dir := filepath.Join("docs", "tickets", candidateState)
		directory, err := root.RepoFS().OpenFile(dir)
		if err != nil {
			continue
		}
		entries, readErr := directory.ReadDir(-1)
		_ = directory.Close()
		if readErr != nil {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			candidateRel := filepath.ToSlash(filepath.Join("docs", "tickets", candidateState, entry.Name()))
			candidateID, _, candidateOK := ticketLifecyclePathIdentity(candidateRel)
			if candidateOK && strings.EqualFold(candidateID, id) {
				dupes = append(dupes, candidateRel)
			}
		}
	}
	return dupes
}

func ticketLifecyclePathIdentity(rel string) (id, state string, ok bool) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	parts := strings.Split(rel, "/")
	if len(parts) != 4 || parts[0] != "docs" || parts[1] != "tickets" {
		return "", "", false
	}
	state = parts[2]
	switch state {
	case "backlog", "in-progress", "in-review", "done":
	default:
		return "", "", false
	}
	base := parts[3]
	if !strings.HasSuffix(strings.ToLower(base), ".md") || strings.EqualFold(base, "README.md") {
		return "", "", false
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	idParts := strings.Split(name, "-")
	if len(idParts) < 2 || idParts[0] == "" || idParts[1] == "" {
		return "", "", false
	}
	return idParts[0] + "-" + idParts[1], state, true
}

func pendingCTOHandoffRequiredScenarios(session Session) []string {
	if session.ToolState == nil {
		return nil
	}
	return splitScenarioList(session.ToolState[ctoHandoffRequiredScenariosKey])
}

func splitScenarioList(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' '
	}) {
		part = strings.ToUpper(strings.TrimSpace(part))
		if toolsFeatureScenarioIDPattern.MatchString(part) {
			out = append(out, part)
		}
	}
	return out
}

func quoteStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
