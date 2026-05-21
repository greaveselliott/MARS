/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/release-versioning.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-007-guardrails-and-safety.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/docsync"
	"github.com/greaveselliott/mars-harness/internal/safety"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

var mutatingTools = map[string]bool{
	"file_write":          true,
	"dependency_sync":     true,
	"shell_exec":          true,
	"mars_harness_cli":    true,
	"git_commit":          true,
	"git_push":            true,
	"record_decision":     true,
	"ticket_create":       true,
	"tool_create":         true,
	"persona_create":      true,
	"release_orchestrate": true,
}

const (
	dogfoodTicketCreateLimitTotal             = 5
	dogfoodTicketCreateLimitPerSeverity       = 3
	dogfoodTicketCreateLimitPerGroup          = 2
	runtimeLearningsPath                      = ".harness/learnings.yaml"
	validationCommandAttemptKey               = "validation:command:attempt"
	validationCommandSuccessKey               = "validation:command:success"
	validationCommandFailureKey               = "validation:command:failure"
	validationProcedureFailureKey             = "validation:procedure_failure"
	validationProcedureFailureCommandKey      = "validation:procedure_failure:command"
	expectedRuntimeFailureSuccessKey          = "validation:runtime_expected_failure:success"
	unexpectedRuntimeValidationOutstandingKey = "validation:runtime_unexpected_failure:outstanding"
	testCommandSuccessKey                     = "validation:test:success"
	testCommandFailureKey                     = "validation:test:failure"
	buildCommandSuccessKey                    = "validation:build:success"
	buildCommandFailureKey                    = "validation:build:failure"
	testBuildValidationOutstandingKey         = "validation:test_build_failure:outstanding"
	testBuildValidationCommandKey             = "validation:test_build_failure:command"
	testBuildValidationOutputKey              = "validation:test_build_failure:output"
	testBuildValidationScopeKey               = "validation:test_build_failure:scope"
	testBuildValidationLastFailureEditKey     = "validation:test_build_failure:last_edit_watermark"
	testBuildRepairWritePathPrefix            = "validation:test_build_failure:repair_write:"
	shellNoopFailureKey                       = "shell:noop:failure"
	reviewTerminalDispositionRequiredKey      = "review:terminal_disposition:required"
	ticketCreationOutstandingFailureKey       = "ticket_create:failure:outstanding"
	unexpectedRuntimeValidationCommandKey     = "validation:runtime_unexpected_failure:command"
	unexpectedRuntimeValidationCorrectionKey  = "validation:runtime_unexpected_failure:correction"
	unexpectedRuntimeValidationMissingArgKey  = "validation:runtime_unexpected_failure:missing_argument"
	unexpectedRuntimeValidationAttemptedKey   = "validation:runtime_unexpected_failure:correction_attempted"
)

func preToolPolicy(ctx context.Context, root Root, name string, raw json.RawMessage) error {
	session, hasSession := SessionFromContext(ctx)
	if hasSession {
		if err := enforceTrust(session, name); err != nil {
			return err
		}
	}
	if err := checkDogfoodFindingCommitPolicy(ctx, root, session, hasSession, name); err != nil {
		return err
	}
	if err := checkEngineerMissingArgumentCorrectionOnly(session, hasSession, name); err != nil {
		return err
	}
	if err := checkReviewTerminalDispositionOnly(root, session, hasSession, name); err != nil {
		return err
	}

	switch name {
	case "file_write":
		if err := checkEngineerClaimBeforeProductMutation(ctx, root, session, hasSession, name, raw); err != nil {
			return err
		}
		if err := checkFileWritePolicy(root, session, hasSession, raw); err != nil {
			return err
		}
		return nil
	case "ticket_create":
		return checkTicketCreatePolicy(root, session, hasSession, raw)
	case "job_disposition_record":
		return checkJobDispositionRecordPolicy(ctx, root, session, hasSession, raw)
	case "dependency_sync", "mars_harness_cli":
		return checkEngineerClaimBeforeProductMutation(ctx, root, session, hasSession, name, raw)
	case "git_commit":
		if err := checkEngineerClaimBeforeProductMutation(ctx, root, session, hasSession, name, raw); err != nil {
			return err
		}
		if err := checkEngineerUnresolvedTestBuildValidationBeforeCommit(session, hasSession); err != nil {
			return err
		}
		if err := checkEngineerUnresolvedRuntimeValidationBeforeCommit(session, hasSession); err != nil {
			return err
		}
		var args gitCommitArgs
		if err := json.Unmarshal(raw, &args); err == nil {
			if err := checkGitCommitGeneratedWorkspacePolicy(ctx, root, args); err != nil {
				return err
			}
		}
		if err := checkEngineerUnresolvedRuntimeValidationBeforeCompletion(ctx, root, session, hasSession, name, raw); err != nil {
			return err
		}
		if err := checkEngineerUnresolvedTestBuildValidationBeforeCompletion(ctx, root, session, hasSession, name, raw); err != nil {
			return err
		}
		return validateRepoDiff(ctx, root, session)
	case "git_push":
		return checkGitPushPolicy(ctx, root, raw)
	case "shell_exec":
		args, err := validateShellExecPolicyArgs(raw)
		if err != nil {
			return err
		}
		if err := checkReviewValidationFailureShellPolicy(session, hasSession, args); err != nil {
			return err
		}
		if err := checkEngineerUnexpectedRuntimeValidationReworkPolicy(root, session, hasSession, args); err != nil {
			return err
		}
		if err := checkEngineerTestBuildValidationReworkPolicy(root, session, hasSession, args); err != nil {
			return err
		}
		if err := checkExternalValidationArtifactFreshness(root, session, hasSession, args); err != nil {
			return err
		}
		if err := checkReviewerShellExecValidationPolicy(root, session, hasSession, raw, args); err != nil {
			return err
		}
		if err := checkShellReleaseTagPolicy(ctx, root, args); err != nil {
			return err
		}
		if err := checkEngineerRepeatedNoopPolicy(ctx, root, session, hasSession, args); err != nil {
			return err
		}
		if err := checkEngineerUnresolvedRuntimeValidationBeforeCompletion(ctx, root, session, hasSession, name, raw); err != nil {
			return err
		}
		if err := checkEngineerUnresolvedTestBuildValidationBeforeCompletion(ctx, root, session, hasSession, name, raw); err != nil {
			return err
		}
		generatedArtifactCleanup, err := shellExecGeneratedArtifactCleanup(ctx, root, args)
		if err != nil {
			return err
		}
		sameJobTestBuildRepairCleanup := shellExecSameJobTestBuildRepairCleanup(root, session, hasSession, args)
		if !generatedArtifactCleanup {
			if err := checkEngineerPostValidationCompletionShellPolicy(ctx, root, session, hasSession, raw); err != nil {
				return err
			}
		}
		if err := checkEngineerShellExecBeforeTicketClaim(root, session, hasSession, raw, generatedArtifactCleanup); err != nil {
			return err
		}
		if !generatedArtifactCleanup && !sameJobTestBuildRepairCleanup {
			if err := checkShellPolicy(raw); err != nil {
				return err
			}
			if err := checkForegroundLongRunningShellPolicy(root, args); err != nil {
				return err
			}
			if err := checkShellBuildOutputPolicy(root, args); err != nil {
				return err
			}
		}
		if err := checkShellTicketDoneEvidencePolicy(ctx, root, raw); err != nil {
			return err
		}
		if generatedArtifactCleanup {
			return nil
		}
		if hasSession && strings.ToLower(strings.TrimSpace(session.Role)) == "coo" && !shellExecReadOnly(raw) {
			return fmt.Errorf("policy: coo cannot run mutating shell_exec; update planning docs with file_write and use git tools for commit/push, while implementation stays behind CTO tickets and Engineer delivery")
		}
		if !shellExecReadOnly(raw) {
			if err := checkEngineerClaimBeforeProductMutation(ctx, root, session, hasSession, name, raw); err != nil {
				return err
			}
			if err := validateRepoDiff(ctx, root, session); err != nil {
				return fmt.Errorf("policy: shell_exec command may mutate while repository is already outside blast-radius limits: %w", err)
			}
		}
		return nil
	default:
		return nil
	}
}

func postToolPolicy(ctx context.Context, root Root, name string, raw json.RawMessage) error {
	if !mutatingTools[name] {
		return nil
	}
	session, _ := SessionFromContext(ctx)
	switch name {
	case "git_commit", "git_push":
		return nil
	case "shell_exec":
		if shellExecReadOnly(raw) {
			return nil
		}
		return validateRepoDiff(ctx, root, session)
	default:
		return validateRepoDiff(ctx, root, session)
	}
}

func enforceTrust(session Session, name string) error {
	level := strings.TrimSpace(strings.ToLower(session.TrustLevel))
	if level == "" {
		return nil
	}
	if level == "observer" && mutatingTools[name] {
		return fmt.Errorf("policy: trust level observer cannot run mutating tool %q", name)
	}
	return nil
}

func checkTicketCreatePolicy(root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession {
		return nil
	}
	var args ticketCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	if err := checkTicketCreatePlanningOrder(root, args); err != nil {
		return err
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
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

func checkTicketCreatePlanningOrder(root Root, args ticketCreateArgs) error {
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
		return fmt.Errorf("policy: feature ticket_create requires bdd_scenarios from an existing docs/features contract; planning order is exec plan, feature contract, ticket, delivery")
	}
	for _, id := range featureIDs {
		if !featureContractExists(root, id) {
			return fmt.Errorf("policy: feature ticket_create references %s before a docs/features/%s*.md contract exists; planning order is exec plan, feature contract, ticket, delivery", id, id)
		}
	}
	return nil
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

func featureIDFromScenarioID(scenario string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(strings.ToUpper(scenario)), "-")
	if len(parts) < 2 || parts[0] != "F" || parts[1] == "" {
		return "", false
	}
	return "F-" + parts[1], true
}

func featureContractExists(root Root, featureID string) bool {
	featuresDir, err := root.ResolvePath(filepath.Join("docs", "features"))
	if err != nil {
		return false
	}
	matches, err := filepath.Glob(filepath.Join(featuresDir, featureID+"*.md"))
	return err == nil && len(matches) > 0
}

func repoFileExists(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
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
		if toolName == "git_commit" && worktreeHasInProgressToDoneTicketMove(ctx, root) {
			return nil
		}
		rework, err := engineerReworkTickets(root)
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

func checkEngineerShellExecBeforeTicketClaim(root Root, session Session, hasSession bool, raw json.RawMessage, generatedArtifactCleanup bool) error {
	if generatedArtifactCleanup || !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if shellExecMovesTicketToInProgress(raw) {
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
	if len(backlog) == 0 {
		rework, err := engineerReworkTickets(root)
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

func checkEngineerPostValidationCompletionShellPolicy(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[validationCommandSuccessKey] == 0 || counts["tool:git_commit:success"] == 0 {
		return nil
	}
	if shellExecMovesInProgressTicketToDone(raw) {
		return nil
	}
	args, err := decodeShellExecArgs(raw)
	if err == nil && shellExecRunsRecordedValidationArtifact(session, root, args) {
		return nil
	}
	tickets, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusInProgress)
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before post-validation shell_exec: %w", err)
	}
	tickets = ordinaryProductTickets(tickets)
	if len(tickets) == 0 {
		return nil
	}
	files, err := changedFiles(ctx, root)
	if err != nil {
		return nil
	}
	blockingFiles := dispositionBlockingFiles(files)
	if shellExecNoop(args) && len(blockingFiles) > 0 {
		return fmt.Errorf(
			"policy: engineer has successful validation and dirty implementation or ticket work for %s. Do not call shell_exec no-op placeholders or waits. %s",
			tickets[0].ID,
			engineerDirtyPostValidationGuidance(tickets[0], blockingFiles),
		)
	}
	if len(blockingFiles) > 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer already has successful validation and a clean implementation commit while product ticket %s remains in progress. Do not call shell_exec again except the exact lifecycle move. Next use file_read on %q, then file_write the same ticket with evidence_links and verified_by populated, then run shell_exec argv [\"git\", \"mv\", %q, \"docs/tickets/done/\"], commit the lifecycle move, and record job_disposition_record with ticket_id %s and next_need qa_review",
		tickets[0].ID,
		tickets[0].RelPath,
		tickets[0].RelPath,
		tickets[0].ID,
	)
}

func engineerDirtyPostValidationGuidance(ticket ticketstate.Ticket, files []string) string {
	var b strings.Builder
	if pids := trackedBackgroundPIDs(); len(pids) > 0 {
		b.WriteString("Stop tracked background validation first")
		for _, pid := range pids {
			b.WriteString(fmt.Sprintf(" with shell_exec argv [\"kill\",\"%d\"]", pid))
		}
		b.WriteString(". ")
	}
	b.WriteString(fmt.Sprintf(
		"Run git_status, git_commit the dirty files (%s), use file_read and file_write on %q to populate evidence_links and verified_by, move %q to docs/tickets/done/ with shell_exec argv [\"git\",\"mv\",%q,\"docs/tickets/done/\"], commit that lifecycle move, git_push if a remote exists, then record job_disposition_record with ticket_id %s and next_need qa_review",
		strings.Join(files, ", "),
		ticket.RelPath,
		ticket.RelPath,
		ticket.RelPath,
		ticket.ID,
	))
	return b.String()
}

func checkEngineerRepeatedNoopPolicy(ctx context.Context, root Root, session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !shellExecNoop(args) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[shellNoopFailureKey] == 0 {
		return nil
	}
	tickets, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusInProgress)
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before repeated shell_exec no-op: %w", err)
	}
	tickets = ordinaryProductTickets(tickets)
	if len(tickets) == 0 {
		return nil
	}
	if counts[validationCommandSuccessKey] == 0 {
		return fmt.Errorf(
			"policy: repeated shell_exec no-op before implementation is a loop. Do not call shell_exec again for placeholders or waits. Use file_read on %q and the linked feature contract, then file_write the product implementation or record job_disposition_record with status blocked if the ticket cannot be implemented",
			tickets[0].RelPath,
		)
	}
	files, err := changedFiles(ctx, root)
	if err != nil || len(dispositionBlockingFiles(files)) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: repeated shell_exec no-op after successful validation is a loop. Do not call shell_exec again for placeholders or waits. Run git_status, update evidence for %s if needed, git_commit the dirty implementation/ticket files, move %s to docs/tickets/done/ when acceptance evidence is present, commit that lifecycle move, then record job_disposition_record with next_need qa_review",
		tickets[0].ID,
		tickets[0].ID,
	)
}

func checkEngineerMissingArgumentCorrectionOnly(session Session, hasSession bool, name string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingRuntimeValidationFailures(session) == 0 {
		return nil
	}
	if session.ToolState == nil || session.ToolState[unexpectedRuntimeValidationMissingArgKey] != "true" {
		return nil
	}
	if name == "file_write" && missingArgumentCorrectionAttempted(session) {
		return nil
	}
	switch name {
	case "file_write", "git_commit", "git_push", "record_decision", "dependency_sync", "mars_harness_cli", "tool_create", "persona_create":
		return fmt.Errorf("policy: engineer has an unresolved no-argument or missing-required-input runtime probe. Do not continue edits, commits, decisions, pushes, or other work yet. Run the exact correction next: %s. If that correction still fails, inspect and edit the implementation before rerunning validation", runtimeValidationExactCorrection(session))
	default:
		return nil
	}
}

func missingArgumentCorrectionAttempted(session Session) bool {
	if session.ToolState == nil {
		return false
	}
	correction := strings.TrimSpace(session.ToolState[unexpectedRuntimeValidationCorrectionKey])
	attempted := strings.TrimSpace(session.ToolState[unexpectedRuntimeValidationAttemptedKey])
	return correction != "" && attempted == correction
}

func checkEngineerUnresolvedRuntimeValidationBeforeCompletion(ctx context.Context, root Root, session Session, hasSession bool, toolName string, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingRuntimeValidationFailures(session) == 0 {
		return nil
	}
	switch toolName {
	case "shell_exec":
		if shellExecMovesInProgressTicketToDone(raw) {
			return unresolvedRuntimeValidationCompletionError("move a product ticket to docs/tickets/done", session)
		}
	case "git_commit":
		if worktreeHasInProgressToDoneTicketMove(ctx, root) {
			return unresolvedRuntimeValidationCompletionError("commit a product ticket completion", session)
		}
	}
	return nil
}

func checkEngineerUnresolvedTestBuildValidationBeforeCompletion(ctx context.Context, root Root, session Session, hasSession bool, toolName string, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingTestBuildValidationFailures(session) == 0 {
		return nil
	}
	switch toolName {
	case "shell_exec":
		if shellExecMovesInProgressTicketToDone(raw) {
			return unresolvedTestBuildValidationCompletionError("move a product ticket to docs/tickets/done", session)
		}
	case "git_commit":
		if worktreeHasInProgressToDoneTicketMove(ctx, root) {
			return unresolvedTestBuildValidationCompletionError("commit a product ticket completion", session)
		}
	}
	return nil
}

func checkEngineerUnresolvedRuntimeValidationBeforeCommit(session Session, hasSession bool) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingRuntimeValidationFailures(session) == 0 {
		return nil
	}
	return unresolvedRuntimeValidationCommitError(session)
}

func checkEngineerUnresolvedTestBuildValidationBeforeCommit(session Session, hasSession bool) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingTestBuildValidationFailures(session) == 0 {
		return nil
	}
	return unresolvedTestBuildValidationCommitError(session)
}

func checkEngineerUnresolvedTestBuildValidationBeforeFileWrite(session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingTestBuildValidationFailures(session) == 0 {
		return nil
	}
	rel = cleanRepoPath(rel)
	if rel == "docs/tickets" || strings.HasPrefix(rel, "docs/tickets/") {
		return unresolvedTestBuildValidationCompletionError("update ticket evidence or lifecycle files", session)
	}
	if engineerTestBuildRepairWritePath(rel) && engineerTestBuildRepairWritePathInScope(session, rel) {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer cannot write %s while a failing test or build command is unresolved in this job. Repair the failing lane by editing the source, tests, fixtures, or package/build config for the failed test/build scope only, or remove duplicate/generated test files created or rewritten earlier in this job, then rerun a test/build command successfully before writing helper scripts, docs, ticket evidence, alternate entrypoints, or other files.%s",
		rel,
		testBuildValidationCorrectionGuidance(session),
	)
}

func engineerTestBuildRepairWritePathInScope(session Session, rel string) bool {
	rel = cleanRepoPath(rel)
	if rel == "" {
		return false
	}
	if !sourceFileRequiresDocSync(rel) && !pathLooksLikeFixtureOrTestdata(rel) {
		return true
	}
	if session.ToolState == nil {
		return true
	}
	raw := strings.TrimSpace(session.ToolState[testBuildValidationScopeKey])
	if raw == "" {
		return true
	}
	for _, scope := range strings.Fields(raw) {
		scope = filepath.ToSlash(strings.TrimSpace(scope))
		scopeDir := strings.HasSuffix(scope, "/")
		scope = cleanRepoPath(scope)
		if scope == "" || scope == "." || scope == "./..." {
			return true
		}
		if scopeDir {
			if strings.HasPrefix(rel, strings.TrimSuffix(scope, "/")+"/") {
				return true
			}
			continue
		}
		if rel == scope {
			return true
		}
	}
	return false
}

func engineerTestBuildRepairWritePath(rel string) bool {
	rel = cleanRepoPath(rel)
	if rel == "" {
		return false
	}
	if sourceFileRequiresDocSync(rel) {
		return true
	}
	lowerRel := strings.ToLower(filepath.ToSlash(rel))
	base := filepath.Base(lowerRel)
	switch base {
	case "go.mod", "go.sum",
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb",
		"cargo.toml", "cargo.lock",
		"pyproject.toml", "poetry.lock", "uv.lock", "requirements.txt", "requirements-dev.txt",
		"pom.xml", "build.gradle", "settings.gradle", "gradle.properties",
		"makefile", "justfile", "gemfile", "gemfile.lock",
		"tsconfig.json", "jsconfig.json", "vite.config.js", "vite.config.ts", "vitest.config.js", "jest.config.js", "webpack.config.js", "next.config.js", "eslint.config.js":
		return true
	}
	return pathLooksLikeFixtureOrTestdata(lowerRel)
}

func pathLooksLikeFixtureOrTestdata(rel string) bool {
	lowerRel := strings.ToLower(filepath.ToSlash(cleanRepoPath(rel)))
	wrapped := "/" + lowerRel + "/"
	return strings.Contains(wrapped, "/testdata/") || strings.Contains(wrapped, "/fixtures/")
}

func checkEngineerUnresolvedRuntimeValidationBeforeDoneFileWrite(session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingRuntimeValidationFailures(session) == 0 {
		return nil
	}
	_, state, ok := ticketLifecyclePathIdentity(cleanRepoPath(rel))
	if !ok || state != "done" {
		return nil
	}
	return unresolvedRuntimeValidationCompletionError("write a product ticket directly in docs/tickets/done", session)
}

func engineerOutstandingRuntimeValidationFailures(session Session) int {
	if session.ToolCounts == nil {
		return 0
	}
	if n := session.ToolCounts[unexpectedRuntimeValidationOutstandingKey]; n > 0 {
		return n
	}
	return 0
}

func engineerOutstandingTestBuildValidationFailures(session Session) int {
	if session.ToolCounts == nil {
		return 0
	}
	if n := session.ToolCounts[testBuildValidationOutstandingKey]; n > 0 {
		return n
	}
	return 0
}

func unresolvedRuntimeValidationCompletionError(action string, sessions ...Session) error {
	var session Session
	if len(sessions) > 0 {
		session = sessions[0]
	}
	return fmt.Errorf("policy: engineer cannot %s while an unexpected runtime validation failure is unresolved in this job. Fix the behavior and rerun the exact failing command successfully before updating evidence, moving the ticket to done, or recording qa_review.%s", action, runtimeValidationCorrectionGuidance(session))
}

func unresolvedRuntimeValidationCommitError(session Session) error {
	return fmt.Errorf("policy: engineer cannot commit product work while an unexpected runtime validation failure is unresolved in this job. Keep the failed implementation uncommitted, inspect and edit the source if needed, rebuild a stale validation artifact when required, then rerun the exact failing command successfully before committing.%s", runtimeValidationCorrectionGuidance(session))
}

func unresolvedTestBuildValidationCompletionError(action string, sessions ...Session) error {
	var session Session
	if len(sessions) > 0 {
		session = sessions[0]
	}
	return fmt.Errorf("policy: engineer cannot %s while a failing test or build command is unresolved in this job. Use file_read/file_write to repair source, tests, fixtures, or package/build config, or remove duplicate/generated test files created or rewritten earlier in this job, then rerun a test/build command successfully before updating evidence, moving the ticket to done, or recording qa_review.%s", action, testBuildValidationCorrectionGuidance(session))
}

func unresolvedTestBuildValidationCommitError(session Session) error {
	return fmt.Errorf("policy: engineer cannot commit product work while a failing test or build command is unresolved in this job. Keep the failed implementation uncommitted, use file_read/file_write to repair source, tests, fixtures, or package/build config, or remove duplicate/generated test files created or rewritten earlier in this job, then rerun a test/build command successfully before committing.%s", testBuildValidationCorrectionGuidance(session))
}

func testBuildValidationCorrectionGuidance(session Session) string {
	var parts []string
	if session.ToolState != nil {
		if command := strings.TrimSpace(session.ToolState[testBuildValidationCommandKey]); command != "" {
			parts = append(parts, "The exact unresolved command was: "+command+".")
		}
		if output := strings.TrimSpace(session.ToolState[testBuildValidationOutputKey]); output != "" {
			parts = append(parts, "Latest failing output: "+output)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "Rerun a same-lane go test, npm test, cargo test, go build, or equivalent build/test command successfully before continuing.")
	}
	parts = append(parts, "If the failing assertion matches the ticket, README, or BDD contract, edit the implementation rather than deleting or weakening the test.")
	return " " + strings.Join(parts, " ")
}

func runtimeValidationCorrectionGuidance(session Session) string {
	if session.ToolState == nil {
		return " If the unresolved failure was an intentional no-argument or missing-required-input negative-path probe, rerun that exact command once with expected_exit_code, usually 1, instead of continuing other probes or finishing."
	}
	if correction := strings.TrimSpace(session.ToolState[unexpectedRuntimeValidationCorrectionKey]); correction != "" {
		return " If the unresolved failure was an intentional no-argument or missing-required-input negative-path probe, run this exact correction next: " + correction + ". Do not run other probes, edits, commits, decisions, or completion first."
	}
	if command := strings.TrimSpace(session.ToolState[unexpectedRuntimeValidationCommandKey]); command != "" {
		return " The exact unresolved command was: " + command + ". Rerun that command successfully before continuing."
	}
	return " If the unresolved failure was an intentional no-argument or missing-required-input negative-path probe, rerun that exact command once with expected_exit_code, usually 1, instead of continuing other probes or finishing."
}

func runtimeValidationExactCorrection(session Session) string {
	if session.ToolState != nil {
		if correction := strings.TrimSpace(session.ToolState[unexpectedRuntimeValidationCorrectionKey]); correction != "" {
			return correction
		}
		if command := strings.TrimSpace(session.ToolState[unexpectedRuntimeValidationCommandKey]); command != "" {
			return command
		}
	}
	return "the exact failing shell_exec command with expected_exit_code set to the expected non-zero code"
}

func checkEngineerTestBuildValidationReworkPolicy(root Root, session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingTestBuildValidationFailures(session) == 0 {
		return nil
	}
	if shellExecSameJobTestBuildRepairCleanupNoRoot(session, args) {
		return nil
	}
	if shellExecRunsMissingPackageConfigBootstrap(root, session, args) {
		return nil
	}
	if !shellExecRunsTestCommand(args) && !shellExecRunsBuildCommand(args) {
		return fmt.Errorf("policy: engineer cannot run that shell_exec while a failing test or build command is unresolved in this job. Do not run runtime probes, shell wrappers, placeholders, discovery, ticket moves, or unrelated shell commands yet. Use file_read/file_write to repair source, tests, fixtures, or package/build config, or remove duplicate/generated test files created or rewritten earlier in this job, then rerun a test/build command successfully.%s", testBuildValidationCorrectionGuidance(session))
	}
	if shellExecRunsTestCommand(args) && session.ToolCounts[testCommandFailureKey] == 0 && session.ToolCounts[buildCommandFailureKey] > 0 {
		return fmt.Errorf("policy: engineer has an unresolved build failure from an earlier command. Do not replace it with a test command yet.%s Repair the build, then rerun a build command successfully before continuing validation", testBuildValidationCorrectionGuidance(session))
	}
	if shellExecRunsBuildCommand(args) && session.ToolCounts[buildCommandFailureKey] == 0 && session.ToolCounts[testCommandFailureKey] > 0 {
		return fmt.Errorf("policy: engineer has an unresolved test failure from an earlier command. Do not replace it with a build command yet.%s Repair the tests or implementation, then rerun a test command successfully before continuing validation", testBuildValidationCorrectionGuidance(session))
	}
	if session.ToolCounts[testBuildValidationEditAfterFailureKey] <= session.ToolCounts[testBuildValidationLastFailureEditKey] {
		return fmt.Errorf("policy: a test/build command already failed after the latest repair edit. Do not rerun validation unchanged or switch commands yet; use file_read/file_write to inspect and edit source, tests, fixtures, or package/build config, or remove duplicate/generated test files created or rewritten earlier in this job, then rerun a same-lane test/build command and make it pass")
	}
	return nil
}

func shellExecRunsMissingPackageConfigBootstrap(root Root, session Session, args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	if len(fields) < 4 {
		return false
	}
	if filepathBase(fields[0]) != "go" || fields[1] != "mod" || fields[2] != "init" {
		return false
	}
	if repoPathExists(root, "go.mod") {
		return false
	}
	return testBuildFailureLooksLikeMissingGoModule(session)
}

func testBuildFailureLooksLikeMissingGoModule(session Session) bool {
	if session.ToolState == nil {
		return false
	}
	output := strings.ToLower(strings.TrimSpace(session.ToolState[testBuildValidationOutputKey]))
	return strings.Contains(output, "cannot find main module") ||
		strings.Contains(output, "go.mod file not found") ||
		strings.Contains(output, "go: go.mod file not found")
}

func repoPathExists(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	_, err = os.Stat(abs)
	return err == nil
}

func shellExecSameJobTestBuildRepairCleanup(root Root, session Session, hasSession bool, args shellExecArgs) bool {
	if !hasSession || !shellExecSameJobTestBuildRepairCleanupNoRoot(session, args) {
		return false
	}
	paths, ok := shellRemovalPathOperands(args)
	if !ok {
		return false
	}
	for _, rel := range paths {
		inside, err := pathResolvesInsideRepo(root, rel)
		if err != nil || !inside {
			return false
		}
	}
	return true
}

func shellExecSameJobTestBuildRepairCleanupNoRoot(session Session, args shellExecArgs) bool {
	if strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingTestBuildValidationFailures(session) == 0 {
		return false
	}
	if session.ToolState == nil {
		return false
	}
	if !testBuildFailureAllowsSameJobTestCleanup(session) {
		return false
	}
	paths, ok := shellRemovalPathOperands(args)
	if !ok {
		return false
	}
	for _, rel := range paths {
		rel = cleanRepoPath(rel)
		if !engineerTestBuildRepairRemovalPath(rel) {
			return false
		}
		if session.ToolState[testBuildRepairWritePathKey(rel)] != "true" {
			return false
		}
	}
	return true
}

func testBuildFailureAllowsSameJobTestCleanup(session Session) bool {
	if session.ToolState == nil {
		return false
	}
	output := strings.ToLower(strings.TrimSpace(session.ToolState[testBuildValidationOutputKey]))
	if output == "" {
		return false
	}
	for _, marker := range []string{
		"redeclared",
		"already declared",
		"duplicate",
		"found packages",
		"expected declaration",
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func engineerTestBuildRepairRemovalPath(rel string) bool {
	rel = cleanRepoPath(rel)
	if rel == "" {
		return false
	}
	lowerRel := strings.ToLower(filepath.ToSlash(rel))
	base := filepath.Base(lowerRel)
	if sourceFileRequiresDocSync(rel) && strings.Contains(base, "test") {
		return true
	}
	wrapped := "/" + lowerRel + "/"
	return strings.Contains(wrapped, "/testdata/") || strings.Contains(wrapped, "/fixtures/")
}

func testBuildRepairWritePathKey(rel string) string {
	return testBuildRepairWritePathPrefix + cleanRepoPath(rel)
}

func checkEngineerUnexpectedRuntimeValidationReworkPolicy(root Root, session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || engineerOutstandingRuntimeValidationFailures(session) == 0 {
		return nil
	}
	if !shellExecRunsRuntimeOrArtifactValidationCommandForSession(&session, root, args) {
		if shellExecRebuildsStaleValidationArtifact(root, session, args) {
			return nil
		}
		return fmt.Errorf("policy: engineer cannot run that shell_exec while an unexpected runtime validation failure is unresolved in this job. Do not run shell_exec for other probes, shell wrappers, placeholders, tests, commits, or ticket moves yet. Use file_read/file_write to repair the implementation, rebuild the same stale validation artifact if required, then rerun the exact failing command successfully.%s", runtimeValidationCorrectionGuidance(session))
	}
	if args.ExpectedExitCode != nil && *args.ExpectedExitCode != 0 {
		if expectedExitCodeCorrectsUnexpectedValidationFailure(session, args) && shellExecLooksLikeMissingArgumentRuntimeProbe(args) {
			return nil
		}
		return fmt.Errorf("policy: engineer cannot use expected_exit_code to resolve an unexpected runtime validation failure unless this exact command is an obvious no-argument or missing-required-input negative-path probe. For positive acceptance failures, edit the implementation, then rerun the exact failing command without expected_exit_code and make it exit successfully.%s", runtimeValidationCorrectionGuidance(session))
	}
	failures := session.ToolCounts[unexpectedRuntimeValidationFailureFingerprintKey(args)]
	repairs := session.ToolCounts[runtimeValidationRepairKey(args)]
	if failures <= repairs {
		return fmt.Errorf("policy: engineer has an unresolved unexpected runtime validation failure from an earlier command. Do not run other runtime probes yet.%s Otherwise inspect and edit the implementation, then rerun the exact failing command successfully before continuing validation", runtimeValidationCorrectionGuidance(session))
	}
	watermark := session.ToolCounts[runtimeValidationFailureEditWatermarkKey(args)]
	if session.ToolCounts[runtimeValidationEditAfterFailureKey] <= watermark {
		return fmt.Errorf("policy: this runtime validation command already failed unexpectedly in this job. Do not rerun the same failing command unchanged; use file_read/file_write to inspect and edit the implementation, then rerun this exact command and make it pass")
	}
	return nil
}

func shellExecRebuildsStaleValidationArtifact(root Root, session Session, args shellExecArgs) bool {
	if !shellExecRunsBuildCommand(args) || session.ToolCounts == nil {
		return false
	}
	output, implicit, ok := goBuildOutputPath(root, args)
	if !ok || implicit {
		return false
	}
	path, ok := validationArtifactPath(root, output)
	if !ok {
		return false
	}
	return session.ToolCounts[validationArtifactSessionKey(path)] > 0 && validationArtifactStaleAfterRuntimeEdit(session, path)
}

func checkForegroundLongRunningShellPolicy(root Root, args shellExecArgs) error {
	if args.Background {
		return nil
	}
	cmd, ok := likelyForegroundLongRunningCommand(root, args)
	if !ok {
		return nil
	}
	return fmt.Errorf("policy: shell_exec command %q is likely a long-running server or watcher; rerun it with background:true, probe readiness with a separate curl or equivalent command, then stop the tracked PID after validation", cmd)
}

func likelyForegroundLongRunningCommand(root Root, args shellExecArgs) (string, bool) {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return "", false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "npm":
		if len(fields) >= 2 && fields[1] == "start" {
			return "npm start", true
		}
		if len(fields) >= 3 && fields[1] == "run" && serverScriptName(fields[2]) {
			return "npm run " + fields[2], true
		}
	case "pnpm", "yarn", "bun":
		if len(fields) >= 2 && serverScriptName(fields[1]) {
			return cmd + " " + fields[1], true
		}
		if len(fields) >= 3 && fields[1] == "run" && serverScriptName(fields[2]) {
			return cmd + " run " + fields[2], true
		}
	case "python", "python3":
		if len(fields) >= 3 && fields[1] == "-m" && fields[2] == "http.server" {
			return cmd + " -m http.server", true
		}
	case "uvicorn", "gunicorn", "hypercorn", "rails", "vite", "next":
		return cmd, true
	case "go":
		if len(fields) >= 2 && fields[1] == "run" && goRunLikelyStartsServer(root, fields[2:]) {
			return "go run", true
		}
	}
	return "", false
}

func normalizedShellExecFields(args shellExecArgs) []string {
	if len(args.Argv) > 0 {
		fields := make([]string, 0, len(args.Argv))
		for _, field := range args.Argv {
			field = strings.Trim(strings.TrimSpace(strings.ToLower(field)), `"'`)
			if field != "" {
				fields = append(fields, field)
			}
		}
		return fields
	}
	cmd := strings.TrimSpace(args.ShellCommand)
	if cmd == "" {
		return nil
	}
	if fields, ok := simpleCDShellCommandTrailingFields(cmd); ok {
		return fields
	}
	if shellCommandHasControlSyntax(cmd) {
		return nil
	}
	return shellFields(cmd)
}

func simpleCDShellCommandTrailingFields(cmd string) ([]string, bool) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || strings.Count(cmd, "&&") != 1 {
		return nil, false
	}
	parts := strings.SplitN(cmd, "&&", 2)
	left := shellFields(parts[0])
	if len(left) != 2 || left[0] != "cd" || strings.TrimSpace(left[1]) == "" {
		return nil, false
	}
	right := strings.TrimSpace(parts[1])
	if right == "" || shellCommandHasControlSyntax(right) {
		return nil, false
	}
	fields := shellFields(right)
	if !shellFieldsRunTestCommand(fields) && !shellFieldsRunBuildCommand(fields) {
		return nil, false
	}
	return fields, len(fields) > 0
}

func serverScriptName(name string) bool {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "dev", "start", "serve", "server", "preview", "watch":
		return true
	default:
		return false
	}
}

func goRunLikelyStartsServer(root Root, args []string) bool {
	targets := goRunTargets(args)
	if len(targets) == 0 {
		return false
	}
	for _, target := range targets {
		for _, rel := range goRunCandidateFiles(target) {
			if sourceContainsServerMarker(root, rel) {
				return true
			}
		}
	}
	return false
}

func goRunTargets(args []string) []string {
	var targets []string
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		targets = append(targets, arg)
	}
	return targets
}

func goRunCandidateFiles(target string) []string {
	target = strings.TrimPrefix(cleanRepoPath(cleanShellPathToken(target)), "./")
	switch {
	case target == "" || target == ".":
		return []string{"main.go"}
	case strings.HasSuffix(target, ".go"):
		return []string{target}
	default:
		return []string{filepath.ToSlash(filepath.Join(target, "main.go"))}
	}
}

func sourceContainsServerMarker(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	for _, marker := range []string{
		"listenandserve",
		"http.handle",
		"http.newservemux",
		"gin.default",
		"fiber.new",
		"chi.newrouter",
		"echo.new",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func engineerToolRequiresClaim(toolName string, raw json.RawMessage) bool {
	switch toolName {
	case "file_write", "dependency_sync", "mars_harness_cli", "git_commit":
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

func engineerReworkTickets(root Root) ([]ticketstate.Ticket, error) {
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

func checkFileWritePolicy(root Root, session Session, hasSession bool, raw json.RawMessage) error {
	args, err := decodeFileWriteArgs(raw)
	if err != nil {
		return nil
	}
	if err := checkTicketFileWritePolicy(root, args.Path); err != nil {
		return err
	}
	if err := checkTicketDoneContentPolicy(root, args.Path, args.Content); err != nil {
		return err
	}
	if err := checkEngineerTicketEvidenceWriteRequiresValidation(session, hasSession, args.Path, args.Content); err != nil {
		return err
	}
	if err := checkEngineerUnresolvedTestBuildValidationBeforeFileWrite(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkEngineerUnresolvedRuntimeValidationBeforeDoneFileWrite(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkFeatureFileWritePolicy(root, args.Path); err != nil {
		return err
	}
	if err := checkFeatureScenarioIDPolicy(args.Path, args.Content); err != nil {
		return err
	}
	if err := checkPlannerFileWritePolicy(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkDogfoodFileWritePolicy(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkSecurityFileWritePolicy(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkRootScratchValidationWritePolicy(root, args.Path); err != nil {
		return err
	}
	if err := checkSourceFileDocSyncWritePolicy(root, args.Path, args.Content); err != nil {
		return err
	}
	if !hasSession {
		return nil
	}
	if session.Guardrails != nil {
		if err := session.Guardrails.CheckFile(session.Role, args.Path, args.Content); err != nil {
			return err
		}
	}
	if hits := safety.ScanForSecrets(args.Path, args.Content); len(hits) > 0 {
		return fmt.Errorf("policy: secret scanner blocked %s:%d (%s)", hits[0].File, hits[0].Line, hits[0].Pattern)
	}
	return nil
}

func checkSourceFileDocSyncWritePolicy(root Root, rel, content string) error {
	rel = cleanRepoPath(rel)
	if !sourceFileRequiresDocSync(rel) {
		return nil
	}
	docs := docsync.MetadataDocs(content)
	if len(docs) == 0 {
		return fmt.Errorf("policy: source file %s must include top-of-file MarsDocSync docs metadata before it can be written; reference the canonical feature contract under docs/features/ (for example docs/features/F-001-product-walking-skeleton.md), not a scenario ID path", rel)
	}
	for _, doc := range docs {
		if !strings.HasPrefix(doc, "docs/") && doc != "AGENTS.md" && doc != "README.md" && doc != "ARCHITECTURE.md" && doc != "CONTRIBUTING.md" {
			return fmt.Errorf("policy: source file %s MarsDocSync metadata references non-documentation path %s", rel, doc)
		}
		if _, err := os.Stat(filepath.Join(root.Abs(), filepath.FromSlash(doc))); err != nil {
			return fmt.Errorf("policy: source file %s MarsDocSync metadata references missing doc %s; read docs/features/ and use the existing canonical feature contract, not a scenario ID path", rel, doc)
		}
	}
	return nil
}

func sourceFileRequiresDocSync(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	switch filepath.Ext(rel) {
	case ".go", ".html", ".css", ".js", ".yaml", ".yml":
	default:
		return false
	}
	if !strings.Contains(rel, "/") {
		return true
	}
	for _, prefix := range []string{
		"cmd/",
		"internal/",
		"pkg/",
		"examples/",
		"src/",
		"app/",
		"pages/",
		"public/",
		"web/",
		"static/",
		".github/workflows/",
	} {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func checkRootScratchValidationWritePolicy(root Root, rel string) error {
	rel = cleanRepoPath(rel)
	if rel == "" || strings.Contains(rel, "/") || !rootScratchValidationName(rel) {
		return nil
	}
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return nil
	}
	if _, err := os.Stat(abs); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("policy: new repo-root scratch validation file %s is blocked because scratch probes become committed product noise; use existing tests, direct shell_exec build/run/curl evidence, or create durable validation code under tests/ with ticket scope", rel)
}

func rootScratchValidationName(rel string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(strings.TrimSpace(rel))))
	ext := filepath.Ext(base)
	if ext == "" {
		return false
	}
	if !rootScratchValidationExt(ext) {
		return false
	}
	stem := strings.TrimSuffix(base, ext)
	switch stem {
	case "debug", "probe", "scratch", "tmp", "temp", "validate", "validation", "verify", "smoke", "smoke-test", "test-server":
		return true
	default:
		return strings.Contains(stem, "validation") || strings.Contains(stem, "scratch") || strings.Contains(stem, "verify")
	}
}

func rootScratchValidationExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".sh", ".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py", ".rb":
		return true
	default:
		return false
	}
}

func checkSecurityFileWritePolicy(session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "security" {
		return nil
	}
	rel = cleanRepoPath(rel)
	if securityReportWritePath(rel) {
		return nil
	}
	return fmt.Errorf("policy: security review cannot write product or ticket files such as %s; write docs/reports/security/security-audit-<date>.md and record changes_requested for Engineer when tests, code, docs, or evidence need remediation", rel)
}

func securityReportWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	return strings.HasPrefix(rel, "docs/reports/security/") && strings.HasSuffix(strings.ToLower(rel), ".md")
}

func checkDogfoodFileWritePolicy(session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "dogfood" {
		return nil
	}
	rel = cleanRepoPath(rel)
	if dogfoodEvidenceWritePath(rel) {
		return nil
	}
	return fmt.Errorf("policy: dogfood is observation-first and cannot write product or package files such as %s; record validation evidence under docs/reports/dogfood or create a target-owned finding with ticket_create", rel)
}

func dogfoodEvidenceWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	return strings.HasPrefix(rel, "docs/reports/dogfood/") && strings.HasSuffix(strings.ToLower(rel), ".md")
}

func checkPlannerFileWritePolicy(session Session, hasSession bool, rel string) error {
	if !hasSession {
		return nil
	}
	rel = cleanRepoPath(rel)
	switch strings.ToLower(strings.TrimSpace(session.Role)) {
	case "coo":
		if cooPlanningWritePath(rel) {
			return nil
		}
		return fmt.Errorf("policy: coo may only write planning artifacts under docs/exec-plans, docs/features, or docs/goals/observations.md; implementation path %s belongs behind CTO tickets and Engineer delivery", rel)
	case "cto", "cto-weekly":
		if ctoTechnicalPlanningWritePath(rel) {
			return nil
		}
		return fmt.Errorf("policy: cto may only write technical planning artifacts under docs/design-docs or docs/reports/strategy; implementation path %s belongs behind ticket_create and Engineer delivery", rel)
	case "ceo":
		if ceoStrategyWritePath(rel) {
			return nil
		}
		return fmt.Errorf("policy: ceo may only write strategy artifacts under docs/goals/active.md, docs/goals/observations.md, docs/product-specs/vision.md, or docs/reports/strategy/; planning path %s belongs to COO/CTO handoff", rel)
	default:
		return nil
	}
}

func cooPlanningWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == filepath.ToSlash(filepath.Join("docs", "goals", "observations.md")) {
		return true
	}
	if rel == filepath.ToSlash(filepath.Join("docs", "exec-plans", "active", "current-operating-plan.md")) {
		return true
	}
	if strings.HasPrefix(rel, "docs/exec-plans/backlog/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
		return true
	}
	if strings.HasPrefix(rel, "docs/features/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
		return true
	}
	return false
}

func ctoTechnicalPlanningWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == filepath.ToSlash(filepath.Join("docs", "goals", "observations.md")) {
		return true
	}
	if strings.HasPrefix(rel, "docs/design-docs/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
		return true
	}
	if strings.HasPrefix(rel, "docs/reports/strategy/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
		return true
	}
	return false
}

func ceoStrategyWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	switch rel {
	case filepath.ToSlash(filepath.Join("docs", "goals", "active.md")),
		filepath.ToSlash(filepath.Join("docs", "goals", "observations.md")),
		filepath.ToSlash(filepath.Join("docs", "product-specs", "vision.md")):
		return true
	default:
		return strings.HasPrefix(rel, "docs/reports/strategy/") && strings.HasSuffix(strings.ToLower(rel), ".md")
	}
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
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return nil
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
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

func checkEngineerTicketEvidenceWriteRequiresValidation(session Session, hasSession bool, rel, content string) error {
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
	if session.ToolCounts != nil && session.ToolCounts[validationCommandSuccessKey] > 0 {
		return nil
	}
	return fmt.Errorf("policy: engineer cannot populate ticket evidence_links or verified_by in %s before successful validation in this job; run go test, a build, or a runtime command that exercises the BDD scenario, then update the ticket with exact evidence", rel)
}

var featureScenarioHeadingRe = regexp.MustCompile(`(?m)^#{3,6}\s+(F-\d{3}-S\d{3})\b`)

type featureScenarioDuplicate struct {
	ID    string
	Lines []int
}

func checkFeatureScenarioIDPolicy(rel, content string) error {
	rel = cleanRepoPath(rel)
	lowerRel := strings.ToLower(rel)
	if !strings.HasPrefix(lowerRel, "docs/features/") || !strings.HasSuffix(lowerRel, ".md") {
		return nil
	}
	dupes := duplicateFeatureScenarioHeadings(content)
	if len(dupes) > 0 {
		return fmt.Errorf(
			"policy: feature contract %s has duplicate scenario ID heading(s): %s; each scenario heading such as `### F-001-S001` may appear once. Read the current file and replace the existing scenario section in one full-file write; do not append a second heading. Scenario Schedule list entries may repeat the ID and are not the duplicate.",
			rel,
			formatFeatureScenarioDuplicates(dupes),
		)
	}
	mismatches := featureScenarioContractMismatches(rel, content)
	if len(mismatches) > 0 {
		return fmt.Errorf(
			"policy: feature contract %s has scenario heading(s) whose feature ID does not match the file: %s. Scenario headings inside %s must use that contract's feature ID; rename the headings or create the matching docs/features/F-NNN*.md contract before ticket creation.",
			rel,
			formatFeatureScenarioMismatches(mismatches),
			rel,
		)
	}
	return nil
}

type featureScenarioMismatch struct {
	ID         string
	Line       int
	ExpectedID string
}

func featureScenarioContractMismatches(rel, content string) []featureScenarioMismatch {
	expectedID, ok := featureContractIDFromName(filepath.Base(rel))
	if !ok {
		return nil
	}
	var mismatches []featureScenarioMismatch
	for lineNumber, line := range strings.Split(content, "\n") {
		match := featureScenarioHeadingRe.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		scenarioID := strings.ToUpper(strings.TrimSpace(match[1]))
		featureID, ok := featureIDFromScenarioID(scenarioID)
		if !ok || featureID == expectedID {
			continue
		}
		mismatches = append(mismatches, featureScenarioMismatch{
			ID:         scenarioID,
			Line:       lineNumber + 1,
			ExpectedID: expectedID,
		})
	}
	return mismatches
}

func formatFeatureScenarioMismatches(mismatches []featureScenarioMismatch) string {
	formatted := make([]string, 0, len(mismatches))
	for _, mismatch := range mismatches {
		formatted = append(formatted, fmt.Sprintf("%s (heading line %d; expected %s-SNNN)", mismatch.ID, mismatch.Line, mismatch.ExpectedID))
	}
	return strings.Join(formatted, ", ")
}

func duplicateFeatureScenarioHeadings(content string) []featureScenarioDuplicate {
	seen := make(map[string][]int)
	var order []string
	for lineNumber, line := range strings.Split(content, "\n") {
		match := featureScenarioHeadingRe.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		id := strings.ToUpper(strings.TrimSpace(match[1]))
		if len(seen[id]) == 0 {
			order = append(order, id)
		}
		seen[id] = append(seen[id], lineNumber+1)
	}
	var dupes []featureScenarioDuplicate
	for _, id := range order {
		lines := seen[id]
		if len(lines) > 1 {
			dupes = append(dupes, featureScenarioDuplicate{ID: id, Lines: lines})
		}
	}
	return dupes
}

func formatFeatureScenarioDuplicates(dupes []featureScenarioDuplicate) string {
	formatted := make([]string, 0, len(dupes))
	for _, dupe := range dupes {
		lineParts := make([]string, 0, len(dupe.Lines))
		for _, line := range dupe.Lines {
			lineParts = append(lineParts, strconv.Itoa(line))
		}
		formatted = append(formatted, fmt.Sprintf("%s (heading lines %s)", dupe.ID, strings.Join(lineParts, ", ")))
	}
	return strings.Join(formatted, ", ")
}

func checkFeatureFileWritePolicy(root Root, rel string) error {
	rel = cleanRepoPath(rel)
	lowerRel := strings.ToLower(rel)
	if !strings.HasPrefix(lowerRel, "docs/features/") || !strings.HasSuffix(lowerRel, ".md") {
		return nil
	}
	base := filepath.Base(rel)
	featureID, ok := featureContractIDFromName(base)
	if !ok {
		return nil
	}
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return nil
	}
	if _, err := os.Stat(abs); err == nil {
		return nil
	}
	featuresDir, err := root.ResolvePath(filepath.Join("docs", "features"))
	if err != nil {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(featuresDir, featureID+"*.md"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	cleanAbs := filepath.Clean(abs)
	for _, match := range matches {
		if filepath.Clean(match) == cleanAbs {
			return nil
		}
	}
	existing := filepath.ToSlash(filepath.Join("docs", "features", filepath.Base(matches[0])))
	return fmt.Errorf("policy: feature contract %s already exists as %s; update the canonical contract instead of creating duplicate feature path %s", featureID, existing, rel)
}

func featureContractIDFromName(base string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(base))
	if !strings.HasPrefix(lower, "f-") || !strings.HasSuffix(lower, ".md") {
		return "", false
	}
	id := strings.TrimSuffix(lower, ".md")
	parts := strings.Split(id, "-")
	if len(parts) < 2 || len(parts[1]) != 3 {
		return "", false
	}
	for _, r := range parts[1] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return "F-" + strings.ToUpper(parts[1]), true
}

func checkJobDispositionRecordPolicy(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) == "orchestrator" {
		return nil
	}
	var args struct {
		Status        string `json:"status"`
		TicketID      string `json:"ticket_id"`
		NextNeed      string `json:"next_need"`
		SuggestedRole string `json:"suggested_role"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	if err := checkEngineerDispositionTicketState(root, session, args.Status, args.TicketID); err != nil {
		return err
	}
	if err := checkReviewDispositionValidationEvidence(root, session, args.Status, args.TicketID); err != nil {
		return err
	}
	if err := checkSuccessfulDispositionUnresolvedTicketCreation(session, args.Status, args.NextNeed, args.SuggestedRole); err != nil {
		return err
	}
	if !dispositionRequiresCleanTree(args.Status) {
		return nil
	}
	files, err := changedFiles(ctx, root)
	if err != nil {
		return nil
	}
	files = dispositionBlockingFiles(files)
	if len(files) == 0 {
		return checkSuccessfulDispositionDocSync(root, session, args.Status)
	}
	return fmt.Errorf("policy: job_disposition_record cannot complete while repository has uncommitted changes: %s. Run git_status, commit the changed work with git_commit, then record the disposition", summarizeChangedFiles(files))
}

func checkSuccessfulDispositionUnresolvedTicketCreation(session Session, status, nextNeed, suggestedRole string) error {
	if !successfulDispositionStatus(status) || session.ToolCounts == nil {
		return nil
	}
	if session.ToolCounts[ticketCreationOutstandingFailureKey] == 0 {
		return nil
	}
	if planningRoleCanHandOffTicketCreation(session.Role, nextNeed, suggestedRole) {
		return nil
	}
	return fmt.Errorf("policy: job_disposition_record cannot record a successful disposition while ticket creation failed earlier in this job and no successful ticket_create followed. Retry ticket_create with valid JSON, including bdd_scenarios as an array like [\"F-001-S002\"], or record status blocked with the exact ticket_create error as the blocker")
}

func planningRoleCanHandOffTicketCreation(role, nextNeed, suggestedRole string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	nextNeed = strings.ToLower(strings.TrimSpace(nextNeed))
	suggestedRole = strings.ToLower(strings.TrimSpace(suggestedRole))
	if role != "coo" && role != "head-of-strategy" {
		return false
	}
	if nextNeed != "ticket_breakdown" && nextNeed != "technical_ticket" && nextNeed != "implementation_ticket" {
		return false
	}
	return suggestedRole == "" || suggestedRole == "cto" || suggestedRole == "cto-weekly"
}

func checkReviewDispositionValidationEvidence(root Root, session Session, status, ticketID string) error {
	if !reviewRoleRequiresValidationEvidence(session.Role) || !successfulReviewDispositionStatus(status) || strings.TrimSpace(ticketID) == "" {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil {
		counts = map[string]int{}
	}
	if counts[testCommandFailureKey] > 0 || counts[buildCommandFailureKey] > 0 {
		return fmt.Errorf("policy: %s cannot approve %s after a failing build or test command in this job; record changes_requested with the exact failing command, output, and Engineer next action instead. Non-zero runtime probes for expected error paths should be documented in evidence and followed by passing tests or positive runtime validation", strings.ToLower(strings.TrimSpace(session.Role)), strings.ToUpper(strings.TrimSpace(ticketID)))
	}
	if counts[validationCommandFailureKey] > 0 {
		return fmt.Errorf("policy: %s cannot approve %s after an unexpected failing validation command in this job; record changes_requested with the exact failing command, output, and Engineer next action instead. If a non-zero runtime probe is intentionally testing an error path, rerun it with shell_exec expected_exit_code set to the expected non-zero code and follow it with passing tests or positive runtime validation", strings.ToLower(strings.TrimSpace(session.Role)), strings.ToUpper(strings.TrimSpace(ticketID)))
	}
	if repoHasTestFiles(root) && counts[testCommandSuccessKey] == 0 {
		return fmt.Errorf("policy: %s must run the repository's authoritative test command successfully before approving %s because test files are present; run the relevant test command such as go test ./... or record changes_requested with the blocker", strings.ToLower(strings.TrimSpace(session.Role)), strings.ToUpper(strings.TrimSpace(ticketID)))
	}
	if strings.ToLower(strings.TrimSpace(session.Role)) == "qa" && repoHasGoSourceFiles(root) && !repoHasTestFiles(root) {
		return fmt.Errorf("policy: qa cannot approve %s because Go source files exist but no _test.go files are present; request Engineer tests that assert the ticket and BDD expected behavior before approval", strings.ToUpper(strings.TrimSpace(ticketID)))
	}
	if counts[validationCommandSuccessKey] == 0 {
		return fmt.Errorf("policy: %s must run at least one authoritative validation command successfully before approving %s; use tests, build, or an end-to-end command that exercises the completed ticket", strings.ToLower(strings.TrimSpace(session.Role)), strings.ToUpper(strings.TrimSpace(ticketID)))
	}
	return nil
}

func checkReviewTerminalDispositionOnly(root Root, session Session, hasSession bool, name string) error {
	if !hasSession || !reviewRoleRequiresValidationEvidence(session.Role) || name == "job_disposition_record" {
		return nil
	}
	if session.ToolCounts == nil || session.ToolCounts[reviewTerminalDispositionRequiredKey] == 0 {
		return nil
	}
	return fmt.Errorf("policy: %s already received terminal disposition guidance after successful validation and a blocked shell_exec no-op. Do not call more tools except job_disposition_record. %s", strings.ToLower(strings.TrimSpace(session.Role)), reviewTerminalDispositionGuidance(root, session))
}

func reviewTerminalDispositionGuidance(root Root, session Session) string {
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role == "" {
		role = "reviewer"
	}
	counts := session.ToolCounts
	if counts == nil {
		counts = map[string]int{}
	}
	if repoHasGoSourceFiles(root) && !repoHasTestFiles(root) {
		return "Call job_disposition_record with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links explaining that Go source files exist but no _test.go files assert the completed behavior."
	}
	if repoHasTestFiles(root) && counts[testCommandSuccessKey] == 0 {
		return "Run the repository's authoritative test command such as go test ./... before approval, or record job_disposition_record with status changes_requested if tests cannot be run."
	}
	if roleRequiresDocSyncForSuccessfulDisposition(session.Role) && counts["tool:docsync_audit:success"] == 0 {
		return "Run docsync_audit before approval, or record job_disposition_record with status changes_requested if documentation sync cannot be verified."
	}
	return "Call job_disposition_record now with status approved, ticket_id, next_need security_review or no_need, and evidence_links naming the build/test/runtime commands that passed."
}

// ReviewTerminalEvidenceSatisfied reports whether a review role has gathered
// enough clean evidence that the agent loop should force the terminal
// disposition path instead of letting more inspection consume the job.
func ReviewTerminalEvidenceSatisfied(root Root, session *Session) bool {
	if session == nil || !reviewRoleRequiresValidationEvidence(session.Role) {
		return false
	}
	counts := session.ToolCounts
	if counts == nil {
		return false
	}
	if counts[reviewTerminalDispositionRequiredKey] > 0 {
		return true
	}
	if counts[validationCommandSuccessKey] == 0 {
		return false
	}
	if counts[testCommandFailureKey] > 0 || counts[buildCommandFailureKey] > 0 || counts[validationCommandFailureKey] > 0 {
		return false
	}
	if repoHasTestFiles(root) && counts[testCommandSuccessKey] == 0 {
		return false
	}
	if roleRequiresDocSyncForSuccessfulDisposition(session.Role) && counts["tool:docsync_audit:success"] == 0 {
		return false
	}
	if counts["tool:file_read:success"] == 0 {
		return false
	}
	return true
}

// MarkReviewTerminalDispositionRequired lets the agent loop turn sufficient
// review evidence into a hard terminal-only boundary before the next tool call.
func MarkReviewTerminalDispositionRequired(session *Session) {
	if session == nil {
		return
	}
	if session.ToolCounts == nil {
		session.ToolCounts = make(map[string]int)
	}
	session.ToolCounts[reviewTerminalDispositionRequiredKey]++
}

// ReviewTerminalDispositionGuidance returns the same terminal guidance used by
// tool policy errors, with nil-safe defaults for agent-loop reminders.
func ReviewTerminalDispositionGuidance(root Root, session *Session) string {
	if session == nil {
		return "Call job_disposition_record now with status approved or changes_requested, ticket_id when applicable, next_need, and evidence_links naming the review evidence."
	}
	return reviewTerminalDispositionGuidance(root, *session)
}

func checkReviewValidationFailureShellPolicy(session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession || !reviewRoleRequiresValidationEvidence(session.Role) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil {
		return nil
	}
	if counts[testCommandFailureKey] == 0 && counts[buildCommandFailureKey] == 0 && counts[validationCommandFailureKey] == 0 {
		return nil
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if counts[testCommandFailureKey] == 0 && counts[buildCommandFailureKey] == 0 && expectedExitCodeCorrectsUnexpectedValidationFailure(session, args) {
		return nil
	}
	return fmt.Errorf("policy: %s already observed a failing build, test, or unexpected runtime validation command in this job; stop validation and call job_disposition_record with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and the exact failing command/output. If the last runtime command was intentionally testing an error path and exited with the expected non-zero code, rerun that exact command once with shell_exec expected_exit_code before any other shell command; otherwise do not call shell_exec again", role)
}

func checkReviewerShellExecValidationPolicy(root Root, session Session, hasSession bool, raw json.RawMessage, args shellExecArgs) error {
	if !hasSession || !reviewRoleRequiresValidationEvidence(session.Role) {
		return nil
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if shellExecNoop(args) {
		counts := session.ToolCounts
		if counts != nil && counts[validationCommandSuccessKey] > 0 && counts[testCommandFailureKey] == 0 && counts[buildCommandFailureKey] == 0 && counts[validationCommandFailureKey] == 0 {
			if repoHasGoSourceFiles(root) && !repoHasTestFiles(root) {
				return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders during review after validation. Stop shell validation now. %s", role, reviewTerminalDispositionGuidance(root, session))
			}
			if repoHasTestFiles(root) && counts[testCommandSuccessKey] == 0 {
				return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders during review after build/runtime validation because test files are present and the authoritative test command has not passed. Run the relevant test command such as go test ./..., or record job_disposition_record with status changes_requested if tests cannot be run", role)
			}
			if roleRequiresDocSyncForSuccessfulDisposition(session.Role) && counts["tool:docsync_audit:success"] == 0 {
				return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders during review after validation because docsync_audit has not passed. Run docsync_audit before approval, or record job_disposition_record with status changes_requested if documentation sync cannot be verified", role)
			}
			return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders during review after successful validation. Stop shell validation now. %s", role, reviewTerminalDispositionGuidance(root, session))
		}
		if counts != nil && (counts[testCommandFailureKey] > 0 || counts[buildCommandFailureKey] > 0 || counts[validationCommandFailureKey] > 0) {
			return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders after failing validation. Stop shell validation and call job_disposition_record now with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links naming the failing command/output", role)
		}
		return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders during review; run a concrete read-only inspection or validation command, or record job_disposition_record with the quality decision", role)
	}
	if shellExecReadOnly(raw) {
		return nil
	}
	if shellExecRunsValidationCommand(args) || shellExecRunsRecordedValidationArtifact(session, root, args) || shellExecRunsHTTPProbe(args) {
		return nil
	}
	return fmt.Errorf("policy: %s shell_exec is validation-only; do not run mutating setup, package initialization, product edits, cleanup, or broad discovery such as %q. Use read-only tools, tests/builds/runtime probes, or record job_disposition_record with a blocker", role, shellExecCommandDisplay(args))
}

func expectedExitCodeCorrectsUnexpectedValidationFailure(session Session, args shellExecArgs) bool {
	if args.ExpectedExitCode == nil || *args.ExpectedExitCode == 0 {
		return false
	}
	if session.ToolCounts == nil {
		return false
	}
	failureKey := unexpectedRuntimeValidationFailureKey(args, *args.ExpectedExitCode)
	correctionKey := expectedRuntimeValidationCorrectionKey(args, *args.ExpectedExitCode)
	return session.ToolCounts[failureKey] > session.ToolCounts[correctionKey]
}

func checkExternalValidationArtifactFreshness(root Root, session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession {
		return nil
	}
	path, ok := shellExecValidationArtifactInvocation(root, args)
	if !ok {
		return nil
	}
	if !shellExecRunsRecordedValidationArtifact(session, root, args) {
		return fmt.Errorf("policy: external validation binary %q must be built in this role session before it can be trusted; next run %s, then execute that freshly built path", path, validationArtifactBuildCorrection(root, path))
	}
	if validationArtifactStaleAfterRuntimeEdit(session, path) {
		return fmt.Errorf("policy: external validation binary %q was built before a post-failure implementation edit. Rebuild it with %s before rerunning runtime validation so stale binary output cannot stand in for the current source", path, validationArtifactBuildCorrection(root, path))
	}
	return nil
}

func validationArtifactBuildCorrection(root Root, path string) string {
	target := validationArtifactBuildTarget(root)
	parts := []string{"go", "build", "-o", path, target}
	data, _ := json.Marshal(parts)
	return "shell_exec argv " + string(data)
}

func validationArtifactBuildTarget(root Root) string {
	if repoFileExists(root, "go.mod") {
		if repoFileExists(root, "main.go") {
			return "."
		}
		cmdMain, ok := firstCmdMain(root)
		if ok {
			return "./" + filepath.ToSlash(filepath.Dir(cmdMain))
		}
		return "."
	}
	if repoFileExists(root, "main.go") {
		return "main.go"
	}
	cmdMain, ok := firstCmdMain(root)
	if ok {
		return filepath.ToSlash(cmdMain)
	}
	return "."
}

func firstCmdMain(root Root) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(root.Abs(), "cmd", "*", "main.go"))
	if err != nil || len(matches) == 0 {
		return "", false
	}
	rel, err := filepath.Rel(root.Abs(), matches[0])
	if err != nil {
		return "", false
	}
	return rel, true
}

func reviewRoleRequiresValidationEvidence(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "qa", "security":
		return true
	default:
		return false
	}
}

func successfulReviewDispositionStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "approved", "in_review":
		return true
	default:
		return false
	}
}

func shellExecRunsValidationCommand(args shellExecArgs) bool {
	return shellExecRunsTestCommand(args) || shellExecRunsBuildCommand(args) || shellExecRunsRuntimeValidationCommand(args)
}

func shellExecRunsValidationCommandForSession(session *Session, root Root, args shellExecArgs) bool {
	if shellExecRunsValidationCommand(args) {
		return true
	}
	if session == nil {
		return false
	}
	return shellExecRunsRecordedValidationArtifact(*session, root, args)
}

func recordSuccessfulValidationArtifactBuild(session *Session, root Root, args shellExecArgs) {
	if session == nil {
		return
	}
	output, implicit, ok := goBuildOutputPath(root, args)
	if !ok || implicit {
		return
	}
	path, ok := validationArtifactPath(root, output)
	if !ok {
		return
	}
	session.ToolCounts[validationArtifactSessionKey(path)]++
	session.ToolCounts[validationArtifactBuildEditWatermarkKey(path)] = session.ToolCounts[runtimeValidationEditAfterFailureKey]
}

func shellExecRunsRecordedValidationArtifact(session Session, root Root, args shellExecArgs) bool {
	if session.ToolCounts == nil {
		return false
	}
	path, ok := shellExecValidationArtifactInvocation(root, args)
	if !ok {
		return false
	}
	return session.ToolCounts[validationArtifactSessionKey(path)] > 0
}

func shellExecValidationArtifactInvocation(root Root, args shellExecArgs) (string, bool) {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return "", false
	}
	return validationArtifactPath(root, fields[0])
}

func validationArtifactPath(root Root, rawPath string) (string, bool) {
	path := cleanShellPathToken(rawPath)
	if !filepath.IsAbs(path) {
		return "", false
	}
	path = filepath.Clean(path)
	inside, err := pathResolvesInsideRepo(root, path)
	if err == nil && inside {
		return "", false
	}
	base := filepath.Base(path)
	if !strings.Contains(base, "-validation") {
		return "", false
	}
	if !pathIsInTempDir(path) {
		return "", false
	}
	return path, true
}

func pathIsInTempDir(path string) bool {
	path = filepath.Clean(path)
	candidates := []string{"/tmp", os.TempDir()}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if candidate == "." || candidate == string(filepath.Separator) {
			continue
		}
		if path == candidate || strings.HasPrefix(path, candidate+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func validationArtifactSessionKey(path string) string {
	return "validation:artifact:" + filepath.Clean(path)
}

func validationArtifactBuildEditWatermarkKey(path string) string {
	return "validation:artifact_build_edit_watermark:" + filepath.Clean(path)
}

func validationArtifactStaleAfterRuntimeEdit(session Session, path string) bool {
	if session.ToolCounts == nil {
		return false
	}
	return session.ToolCounts[runtimeValidationEditAfterFailureKey] > session.ToolCounts[validationArtifactBuildEditWatermarkKey(path)]
}

func unexpectedRuntimeValidationFailureKey(args shellExecArgs, exitCode int) string {
	return fmt.Sprintf("validation:runtime_unexpected_failure:%d:%s", exitCode, shellExecCommandFingerprint(args))
}

func unexpectedRuntimeValidationFailureFingerprintKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:runtime_unexpected_failure:%s", shellExecCommandFingerprint(args))
}

func expectedRuntimeValidationCorrectionKey(args shellExecArgs, exitCode int) string {
	return fmt.Sprintf("validation:runtime_expected_correction:%d:%s", exitCode, shellExecCommandFingerprint(args))
}

func runtimeValidationRepairKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:runtime_repair:%s", shellExecCommandFingerprint(args))
}

const runtimeValidationEditAfterFailureKey = "validation:runtime_unexpected_failure:edit_after"

func runtimeValidationFailureEditWatermarkKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:runtime_unexpected_failure_edit_watermark:%s", shellExecCommandFingerprint(args))
}

func testBuildValidationFailureFingerprintKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:test_build_failure:%s", shellExecCommandFingerprint(args))
}

func testBuildValidationRepairKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:test_build_repair:%s", shellExecCommandFingerprint(args))
}

const testBuildValidationEditAfterFailureKey = "validation:test_build_failure:edit_after"

func testBuildValidationFailureEditWatermarkKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:test_build_failure_edit_watermark:%s", shellExecCommandFingerprint(args))
}

func testBuildValidationRepairScopes(args shellExecArgs) []string {
	fields := normalizedShellExecFields(args)
	if len(fields) < 2 || filepathBase(fields[0]) != "go" {
		return nil
	}
	switch fields[1] {
	case "test", "build":
	default:
		return nil
	}
	var scopes []string
	skipNext := false
	for _, field := range fields[2:] {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(field, "-") {
			if goFlagLikelyConsumesValue(field) && !strings.Contains(field, "=") {
				skipNext = true
			}
			continue
		}
		scope, ok := goValidationTargetRepairScope(field)
		if !ok {
			continue
		}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		scopes = append(scopes, ".")
	}
	return uniqueNonEmptyStrings(scopes)
}

func goFlagLikelyConsumesValue(flag string) bool {
	flag = strings.TrimSpace(strings.ToLower(flag))
	switch flag {
	case "-run", "-bench", "-count", "-timeout", "-tags", "-coverprofile", "-coverpkg", "-o":
		return true
	default:
		return false
	}
}

func goValidationTargetRepairScope(target string) (string, bool) {
	target = strings.Trim(strings.TrimSpace(strings.ToLower(target)), `"'`)
	if target == "" {
		return "", false
	}
	if target == "." || target == "./..." {
		return ".", true
	}
	if !strings.HasPrefix(target, "./") {
		return "", false
	}
	rel := cleanRepoPath(strings.TrimPrefix(target, "./"))
	if rel == "" || rel == "..." {
		return ".", true
	}
	rel = strings.TrimSuffix(rel, "/...")
	if rel == "" || rel == "." {
		return ".", true
	}
	if filepath.Ext(rel) != "" {
		return rel, true
	}
	return strings.TrimSuffix(rel, "/") + "/", true
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func shellExecLooksLikeMissingArgumentRuntimeProbe(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "go":
		return len(fields) == 3 && fields[1] == "run"
	case "cargo", "dotnet":
		return len(fields) == 2 && fields[1] == "run"
	case "python", "python3", "node", "deno", "ruby":
		return len(fields) == 2
	default:
		return len(fields) == 1 && !strings.HasPrefix(fields[0], "-")
	}
}

func shellExecCommandFingerprint(args shellExecArgs) string {
	var payload string
	if len(args.Argv) > 0 {
		payload = "argv\x00" + strings.Join(normalizedShellExecFields(args), "\x00")
	} else {
		payload = "shell\x00" + strings.TrimSpace(strings.ToLower(args.ShellCommand))
	}
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", sum[:8])
}

func shellExecRunsTestCommand(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	return shellFieldsRunTestCommand(fields)
}

func shellFieldsRunTestCommand(fields []string) bool {
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "go", "cargo":
		return len(fields) >= 2 && fields[1] == "test"
	case "npm", "pnpm", "yarn":
		return len(fields) >= 2 && (fields[1] == "test" || (fields[1] == "run" && len(fields) >= 3 && testScriptName(fields[2])))
	case "bun":
		return len(fields) >= 2 && (fields[1] == "test" || (fields[1] == "run" && len(fields) >= 3 && testScriptName(fields[2])))
	case "pytest", "go test":
		return true
	case "python", "python3":
		return len(fields) >= 3 && fields[1] == "-m" && fields[2] == "pytest"
	case "make":
		return len(fields) >= 2 && testScriptName(fields[1])
	default:
		return false
	}
}

func shellExecRunsBuildCommand(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	return shellFieldsRunBuildCommand(fields)
}

func shellFieldsRunBuildCommand(fields []string) bool {
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "go", "cargo":
		return len(fields) >= 2 && fields[1] == "build"
	case "npm", "pnpm", "yarn", "bun":
		return len(fields) >= 2 && ((fields[1] == "run" && len(fields) >= 3 && buildScriptName(fields[2])) || buildScriptName(fields[1]))
	case "make":
		return len(fields) >= 2 && buildScriptName(fields[1])
	default:
		return false
	}
}

func shellExecRunsRuntimeValidationCommand(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "go", "cargo", "dotnet":
		return len(fields) >= 2 && fields[1] == "run"
	case "python", "python3", "node", "deno", "ruby":
		return len(fields) >= 2
	case "java":
		return len(fields) >= 3 && fields[1] == "-jar"
	case "npm", "pnpm", "yarn", "bun":
		if len(fields) < 2 {
			return false
		}
		if fields[1] == "run" {
			return len(fields) >= 3 && runtimeScriptName(fields[2])
		}
		return runtimeScriptName(fields[1])
	default:
		return false
	}
}

func shellExecRunsHTTPProbe(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return false
	}
	switch filepathBase(fields[0]) {
	case "curl", "wget":
		return len(fields) >= 2
	default:
		return false
	}
}

func shellExecCommandDisplay(args shellExecArgs) string {
	if len(args.Argv) > 0 {
		return strings.Join(args.Argv, " ")
	}
	return strings.TrimSpace(args.ShellCommand)
}

func testScriptName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return name == "test" || strings.HasPrefix(name, "test:") || strings.Contains(name, "test")
}

func buildScriptName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return name == "build" || strings.HasPrefix(name, "build:") || strings.Contains(name, "build")
}

func runtimeScriptName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	switch name {
	case "start", "serve", "run", "dev", "preview":
		return true
	default:
		return strings.HasPrefix(name, "start:") || strings.HasPrefix(name, "serve:") || strings.HasPrefix(name, "run:") || strings.HasPrefix(name, "dev:") || strings.HasPrefix(name, "preview:")
	}
}

func repoHasTestFiles(root Root) bool {
	hasTests := false
	_ = filepath.WalkDir(root.Abs(), func(path string, d os.DirEntry, err error) error {
		if err != nil || hasTests {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		rel, err := filepath.Rel(root.Abs(), path)
		if err != nil {
			return nil
		}
		if testFilePath(filepath.ToSlash(rel)) {
			hasTests = true
		}
		return nil
	})
	return hasTests
}

func repoHasGoSourceFiles(root Root) bool {
	hasSource := false
	_ = filepath.WalkDir(root.Abs(), func(path string, d os.DirEntry, err error) error {
		if err != nil || hasSource {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		rel, err := filepath.Rel(root.Abs(), path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(strings.ToLower(strings.TrimSpace(rel)))
		if strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go") {
			hasSource = true
		}
		return nil
	})
	return hasSource
}

func testFilePath(rel string) bool {
	rel = filepath.ToSlash(strings.ToLower(strings.TrimSpace(rel)))
	base := filepath.Base(rel)
	if strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.rs") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".test.jsx") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".spec.jsx") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") ||
		strings.HasSuffix(base, "_test.py") {
		return true
	}
	return strings.HasPrefix(rel, "tests/") || strings.Contains(rel, "/tests/")
}

func checkSuccessfulDispositionDocSync(root Root, session Session, status string) error {
	if !docSyncDispositionStatus(status) || !roleRequiresDocSyncForSuccessfulDisposition(session.Role) {
		return nil
	}
	report, err := docsync.Audit(docsync.Config{RepoRoot: root.Abs()})
	if err != nil {
		return fmt.Errorf("policy: job_disposition_record could not run docsync_audit before a successful disposition: %w", err)
	}
	if report.OK() {
		return nil
	}
	return fmt.Errorf("policy: successful disposition blocked by docsync_audit findings: %s. Fix MarsDocSync metadata and rerun docsync_audit, or record changes_requested/blocked with feedback instead of approving", summarizeDocSyncFindings(report.Findings))
}

func docSyncDispositionStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "approved", "in_review":
		return true
	default:
		return false
	}
}

func roleRequiresDocSyncForSuccessfulDisposition(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "engineer", "pipeline-fixer", "qa", "security", "dogfood", "release-manager", "dependency-manager":
		return true
	default:
		return false
	}
}

func summarizeDocSyncFindings(findings []docsync.Finding) string {
	if len(findings) == 0 {
		return "no findings"
	}
	limit := len(findings)
	if limit > 4 {
		limit = 4
	}
	parts := make([]string, 0, limit)
	for _, finding := range findings[:limit] {
		parts = append(parts, finding.Path+": "+finding.Message)
	}
	if len(findings) > limit {
		parts = append(parts, fmt.Sprintf("and %d more", len(findings)-limit))
	}
	return strings.Join(parts, "; ")
}

func checkEngineerDispositionTicketState(root Root, session Session, status, ticketID string) error {
	if strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if successfulReviewDispositionStatus(status) && engineerOutstandingRuntimeValidationFailures(session) > 0 {
		return unresolvedRuntimeValidationCompletionError("record a successful product disposition", session)
	}
	if successfulReviewDispositionStatus(status) && engineerOutstandingTestBuildValidationFailures(session) > 0 {
		return unresolvedTestBuildValidationCompletionError("record a successful product disposition", session)
	}
	if strings.ToLower(strings.TrimSpace(status)) == "no_work" && strings.TrimSpace(ticketID) == "" {
		return nil
	}
	ticketID = strings.ToUpper(strings.TrimSpace(ticketID))
	if ticketID == "" {
		tickets, err := ticketstate.List(root.Abs())
		if err == nil && len(ordinaryProductTickets(tickets)) > 0 {
			return fmt.Errorf("policy: engineer successful disposition must name ticket_id for the completed product ticket; if no eligible ticket exists, use status no_work")
		}
		return nil
	}
	tickets, err := ticketstate.List(root.Abs())
	if err != nil {
		return nil
	}
	found := false
	for _, t := range tickets {
		if strings.ToUpper(strings.TrimSpace(t.ID)) != ticketID {
			continue
		}
		found = true
		if t.Status == ticketstate.StatusDone {
			continue
		}
		return fmt.Errorf("policy: engineer cannot record a successful disposition for %s while it remains in %s; update evidence, move the ticket to docs/tickets/done/ with git mv, commit the lifecycle move, then record qa_review", ticketID, t.RelPath)
	}
	if !found {
		return fmt.Errorf("policy: engineer cannot record a successful disposition for missing ticket_id %s; move the selected ticket to docs/tickets/done/ with evidence, commit it, then record qa_review", ticketID)
	}
	return nil
}

func dispositionBlockingFiles(files []string) []string {
	var blocking []string
	for _, file := range files {
		if filepath.ToSlash(strings.TrimSpace(file)) == runtimeLearningsPath {
			continue
		}
		blocking = append(blocking, file)
	}
	return blocking
}

func successfulDispositionStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "approved", "in_review", "no_work":
		return true
	default:
		return false
	}
}

func dispositionRequiresCleanTree(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "approved", "in_review", "no_work", "changes_requested", "blocked", "failed":
		return true
	default:
		return false
	}
}

func summarizeChangedFiles(files []string) string {
	if len(files) <= 6 {
		return strings.Join(files, ", ")
	}
	return strings.Join(files[:6], ", ") + fmt.Sprintf(", and %d more", len(files)-6)
}

func cleanRepoPath(rel string) string {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(rel))
}

func isTicketLifecycleDir(dir string) bool {
	switch dir {
	case "backlog", "in-progress", "in-review", "done":
		return true
	default:
		return false
	}
}

func checkGitPushPolicy(ctx context.Context, root Root, raw json.RawMessage) error {
	var args gitPushArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	branch := strings.TrimSpace(args.Branch)
	if branch == "" {
		out, err := runGit(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}
		if out.ExitCode != 0 {
			return fmt.Errorf("policy: determine branch before push: %s", strings.TrimSpace(out.Stderr))
		}
		branch = strings.TrimSpace(out.Output)
	}
	if branch != "main" {
		return fmt.Errorf("policy: strict trunk only allows pushing main, got %q", branch)
	}
	return nil
}

func checkShellReleaseTagPolicy(ctx context.Context, root Root, args shellExecArgs) error {
	tag, target, ok := shellExecReleaseTagMutation(args)
	if !ok {
		return nil
	}
	version := strings.TrimSpace(readOptional(root, "VERSION"))
	if version == "" {
		return fmt.Errorf("policy: release tag %s cannot be created before VERSION exists; generate release notes, commit them, then tag the release-note commit", tag)
	}
	expectedTag := "v" + version
	if tag != strings.ToLower(expectedTag) {
		return fmt.Errorf("policy: release tag %s does not match VERSION %s; create or update %s only after the release-note commit is HEAD", tag, version, expectedTag)
	}
	files, err := changedFiles(ctx, root)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		return fmt.Errorf("policy: release tag %s must be created after VERSION and CHANGELOG.md are committed; uncommitted changes remain: %s. Commit them with git_commit message %q, then tag that release-note commit", expectedTag, summarizeChangedFiles(files), "release: notes "+version)
	}
	headSubject := strings.TrimSpace(gitOutput(ctx, root, "log", "-1", "--pretty=%s"))
	if !strings.HasPrefix(strings.ToLower(headSubject), "release: notes ") {
		return fmt.Errorf("policy: release tag %s must point at a release-note commit, but HEAD subject is %q. Commit VERSION and CHANGELOG.md as %q before tagging", expectedTag, headSubject, "release: notes "+version)
	}
	if target == "" {
		return nil
	}
	head := strings.TrimSpace(gitOutput(ctx, root, "rev-parse", "HEAD"))
	resolved, err := runGit(ctx, root, "rev-parse", "--verify", target+"^{commit}")
	if err != nil {
		return err
	}
	if resolved.ExitCode != 0 {
		return fmt.Errorf("policy: release tag %s target %q is not a commit; tag the current release-note HEAD instead", expectedTag, target)
	}
	targetSHA := strings.TrimSpace(resolved.Output)
	if head == "" || targetSHA != head {
		return fmt.Errorf("policy: release tag %s target %q resolves to %s, not current release-note HEAD %s; create or update the tag at HEAD after committing release notes", expectedTag, target, targetSHA, head)
	}
	return nil
}

func shellExecReleaseTagMutation(args shellExecArgs) (tag string, target string, ok bool) {
	fields := normalizedShellExecFields(args)
	subcommand, subArgs := gitShellSubcommand(fields)
	if subcommand != "tag" {
		return "", "", false
	}
	if len(subArgs) == 0 || gitTagArgsListOnly(subArgs) {
		return "", "", false
	}
	if hasToken(subArgs, "-d") || hasToken(subArgs, "--delete") {
		return "", "", false
	}
	for i := 0; i < len(subArgs); i++ {
		arg := strings.TrimSpace(subArgs[i])
		if arg == "" {
			continue
		}
		if gitTagFlagConsumesNext(arg) {
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if !strings.HasPrefix(arg, "v") {
			return "", "", false
		}
		tag = arg
		if i+1 < len(subArgs) {
			for j := i + 1; j < len(subArgs); j++ {
				candidate := strings.TrimSpace(subArgs[j])
				if gitTagFlagConsumesNext(candidate) {
					j++
					continue
				}
				candidate = strings.TrimSpace(candidate)
				if candidate == "" || strings.HasPrefix(candidate, "-") {
					continue
				}
				target = candidate
				break
			}
		}
		return tag, target, true
	}
	return "", "", false
}

func gitTagArgsListOnly(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		switch {
		case arg == "-l", arg == "--list", strings.HasPrefix(arg, "--list="):
			return true
		case strings.HasPrefix(arg, "-"):
			continue
		default:
			return false
		}
	}
	return true
}

func gitTagFlagConsumesNext(arg string) bool {
	switch arg {
	case "-m", "-F", "-u", "--message", "--file", "--local-user":
		return true
	default:
		return false
	}
}

func checkShellPolicy(raw json.RawMessage) error {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
	}
	if err := checkShellMarsHarnessCLIPolicy(args); err != nil {
		return err
	}
	cmd := strings.Join(args.Argv, " ")
	if strings.TrimSpace(args.ShellCommand) != "" {
		cmd = args.ShellCommand
	}
	if err := checkShellTicketPathPolicy(cmd); err != nil {
		return err
	}
	if operation, ok := forbiddenShellOperation(cmd); ok {
		return fmt.Errorf("policy: shell_exec command contains forbidden operation %q", operation)
	}
	if operation, ok := dependencyShellOperation(cmd); ok {
		return fmt.Errorf("policy: shell_exec command %q mutates dependency state; use dependency_sync so workspace hygiene preflight and postflight run", operation)
	}
	if operation, ok := broadGeneratedTraversal(cmd); ok {
		return fmt.Errorf("policy: shell_exec command %q may flood context with generated dependency/build output; use file_search, grep, or add explicit generated-directory excludes", operation)
	}
	return nil
}

func checkShellMarsHarnessCLIPolicy(args shellExecArgs) error {
	if cliArgs, ok := shellExecMarsHarnessArgs(args); ok {
		encoded, _ := json.Marshal(cliArgs)
		return fmt.Errorf("policy: run mars-harness commands with mars_harness_cli, not shell_exec, so agents use the active harness executable instead of a stale PATH binary. Retry with mars_harness_cli args %s", encoded)
	}
	return nil
}

func shellExecMarsHarnessArgs(args shellExecArgs) ([]string, bool) {
	if len(args.Argv) > 0 {
		if filepath.Base(strings.TrimSpace(args.Argv[0])) == "mars-harness" {
			return append([]string(nil), args.Argv[1:]...), true
		}
		return nil, false
	}
	fields := strings.Fields(args.ShellCommand)
	for len(fields) > 0 && shellEnvAssignment(fields[0]) {
		fields = fields[1:]
	}
	if len(fields) == 0 || filepath.Base(strings.TrimSpace(fields[0])) != "mars-harness" {
		return nil, false
	}
	return append([]string(nil), fields[1:]...), true
}

func shellEnvAssignment(field string) bool {
	if field == "" || strings.HasPrefix(field, "-") {
		return false
	}
	idx := strings.IndexByte(field, '=')
	if idx <= 0 {
		return false
	}
	name := field[:idx]
	for i, r := range name {
		if r == '_' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') || (i > 0 && '0' <= r && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func validateShellExecPolicyArgs(raw json.RawMessage) (shellExecArgs, error) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return shellExecArgs{}, fmt.Errorf("shell_exec: parse arguments: %w", err)
	}
	if shellExecNoop(args) {
		return args, nil
	}
	hasArgv := len(args.Argv) > 0
	hasShell := strings.TrimSpace(args.ShellCommand) != ""
	if hasArgv == hasShell {
		return shellExecArgs{}, fmt.Errorf("shell_exec: provide exactly one of argv (non-empty) or shell_command")
	}
	if hasArgv && args.Argv[0] == "" {
		return shellExecArgs{}, fmt.Errorf("shell_exec: argv[0] must be non-empty")
	}
	if hasArgv {
		if err := validateShellExecArgv(args.Argv); err != nil {
			return shellExecArgs{}, err
		}
		if err := validateShellExecGitRemoteMutation(args.Argv, "argv"); err != nil {
			return shellExecArgs{}, err
		}
	} else {
		if err := validateShellExecShellCommand(args.ShellCommand); err != nil {
			return shellExecArgs{}, err
		}
		if err := validateShellExecGitRemoteMutation(strings.Fields(args.ShellCommand), "shell_command"); err != nil {
			return shellExecArgs{}, err
		}
	}
	return args, nil
}

func shellExecGeneratedArtifactCleanup(ctx context.Context, root Root, args shellExecArgs) (bool, error) {
	paths, ok := shellRemovalPathOperands(args)
	if !ok || len(paths) == 0 {
		return false, nil
	}
	for _, rel := range paths {
		if _, ok := validationArtifactPath(root, rel); ok {
			continue
		}
		generated, err := isUntrackedRootBuildArtifact(ctx, root, rel)
		if err != nil {
			return false, err
		}
		if !generated {
			return false, nil
		}
	}
	return true, nil
}

func checkShellBuildOutputPolicy(root Root, args shellExecArgs) error {
	output, implicit, ok := goBuildOutputPath(root, args)
	if !ok || strings.TrimSpace(output) == "" {
		return nil
	}
	if implicit {
		suggestion := validationBinaryOutputSuggestion(output)
		correction := goBuildValidationCorrection(args, suggestion)
		return fmt.Errorf("policy: go build without -o can create the build artifact %q inside the target repo; rerun exactly: %s, or use go test ./... for compile validation so repository diffs stay source-only", output, correction)
	}
	inside, err := pathResolvesInsideRepo(root, output)
	if err != nil {
		return err
	}
	if !inside {
		if _, ok := validationArtifactPath(root, output); ok {
			return nil
		}
		suggestion := validationBinaryOutputSuggestion(output)
		correction := goBuildValidationCorrection(args, suggestion)
		return fmt.Errorf("policy: go build output %q is outside the target repo but is not a tracked validation artifact; rerun exactly: %s so stale temp binaries are blocked and same-session validation can be trusted", output, correction)
	}
	suggestion := validationBinaryOutputSuggestion(output)
	correction := goBuildValidationCorrection(args, suggestion)
	return fmt.Errorf("policy: go build output %q would create a build artifact inside the target repo; rerun exactly: %s, then run or delete that external validation binary and keep repo diffs source-only", output, correction)
}

func goBuildValidationCorrection(args shellExecArgs, suggestion string) string {
	fields := goBuildCommandFields(args)
	if len(fields) < 2 {
		raw, _ := json.Marshal([]string{"go", "build", "-o", suggestion})
		return "shell_exec argv " + string(raw)
	}
	corrected := make([]string, 0, len(fields)+2)
	corrected = append(corrected, cleanShellDisplayToken(fields[0]), cleanShellDisplayToken(fields[1]))
	inserted := false
	for i := 2; i < len(fields); i++ {
		field := strings.TrimSpace(fields[i])
		if field == "" || shellControlToken(field) {
			break
		}
		switch {
		case field == "-o":
			if !inserted {
				corrected = append(corrected, "-o", suggestion)
				inserted = true
			}
			if i+1 < len(fields) {
				i++
			}
		case strings.HasPrefix(field, "-o="):
			if !inserted {
				corrected = append(corrected, "-o", suggestion)
				inserted = true
			}
		default:
			corrected = append(corrected, cleanShellDisplayToken(field))
		}
	}
	if !inserted {
		corrected = append([]string{corrected[0], corrected[1], "-o", suggestion}, corrected[2:]...)
	}
	raw, _ := json.Marshal(corrected)
	return "shell_exec argv " + string(raw)
}

func goBuildCommandFields(args shellExecArgs) []string {
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellCommandFields(args.ShellCommand)
	}
	for i := 0; i < len(fields)-1; i++ {
		if filepathBase(cleanShellPathToken(fields[i])) == "go" && cleanShellPathToken(fields[i+1]) == "build" {
			return fields[i:]
		}
	}
	return nil
}

func validationBinaryOutputSuggestion(output string) string {
	base := filepath.Base(cleanShellPathToken(output))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "app"
	}
	return filepath.ToSlash(filepath.Join("/tmp", base+"-validation"))
}

func cleanShellDisplayToken(field string) string {
	field = strings.TrimSpace(field)
	field = strings.TrimPrefix(field, "1>")
	field = strings.TrimPrefix(field, "2>")
	field = strings.TrimLeft(field, "><")
	return strings.Trim(field, `"'`)
}

func goBuildOutputPath(root Root, args shellExecArgs) (string, bool, bool) {
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellCommandFields(args.ShellCommand)
	}
	for i := 0; i < len(fields)-1; i++ {
		if filepathBase(cleanShellPathToken(fields[i])) != "go" || cleanShellPathToken(fields[i+1]) != "build" {
			continue
		}
		return goBuildOutputPathFromFields(root, fields[i:])
	}
	return "", false, false
}

func goBuildOutputPathFromFields(root Root, fields []string) (string, bool, bool) {
	if len(fields) < 2 {
		return "", false, false
	}
	for i := 2; i < len(fields); i++ {
		field := strings.TrimSpace(fields[i])
		if shellControlToken(field) {
			break
		}
		switch {
		case field == "-o":
			if i+1 < len(fields) {
				return cleanShellPathToken(fields[i+1]), false, true
			}
			return "", false, true
		case strings.HasPrefix(field, "-o="):
			return cleanShellPathToken(strings.TrimPrefix(field, "-o=")), false, true
		}
	}
	return goBuildDefaultOutputName(root), true, true
}

func shellControlToken(field string) bool {
	switch strings.TrimSpace(field) {
	case "&&", "||", "|", ";", "&":
		return true
	default:
		return false
	}
}

func goBuildDefaultOutputName(root Root) string {
	if data, err := os.ReadFile(filepath.Join(root.Abs(), "go.mod")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(strings.TrimSpace(line))
			if len(fields) == 2 && fields[0] == "module" {
				if base := filepath.Base(strings.TrimRight(fields[1], "/")); base != "" && base != "." && base != string(filepath.Separator) {
					return base
				}
				break
			}
		}
	}
	if base := filepath.Base(root.Abs()); base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return "app"
}

func shellCommandFields(cmd string) []string {
	raw := strings.Fields(cmd)
	fields := make([]string, 0, len(raw))
	for _, field := range raw {
		field = strings.TrimSpace(strings.Trim(field, `"'`))
		field = strings.TrimRight(field, ";")
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

func pathResolvesInsideRepo(root Root, path string) (bool, error) {
	path = cleanShellPathToken(path)
	if path == "" {
		return false, nil
	}
	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		resolved, err := root.ResolvePath(path)
		if err != nil {
			return false, err
		}
		abs = filepath.Clean(resolved)
	}
	rel, err := filepath.Rel(root.Abs(), abs)
	if err != nil {
		return false, err
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))), nil
}

func shellRemovalPathOperands(args shellExecArgs) ([]string, bool) {
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		if shellCommandHasControlSyntax(args.ShellCommand) {
			return nil, false
		}
		fields = strings.Fields(args.ShellCommand)
	}
	if len(fields) < 2 {
		return nil, false
	}
	cmd := filepathBase(strings.Trim(fields[0], `"'`))
	if cmd != "rm" && cmd != "unlink" {
		return nil, false
	}
	var paths []string
	for _, field := range fields[1:] {
		field = strings.TrimSpace(strings.Trim(field, `"'`))
		if field == "" || field == "--" {
			continue
		}
		if strings.HasPrefix(field, "-") {
			if strings.ContainsAny(field, "rR") {
				return nil, false
			}
			continue
		}
		paths = append(paths, cleanShellPathToken(field))
	}
	return paths, len(paths) > 0
}

func isUntrackedRootBuildArtifact(ctx context.Context, root Root, rel string) (bool, error) {
	rel = cleanRepoPath(rel)
	if rel == "" || rel == "." || strings.Contains(rel, "/") {
		return false, nil
	}
	if !isAllowedRootBuildArtifactName(root, rel) {
		return false, nil
	}
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false, nil
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return false, nil
	}
	ls, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard", "--", rel)
	if err != nil {
		return false, err
	}
	if ls.ExitCode != 0 || !lineListContains(ls.Output, rel) {
		return false, nil
	}
	return fileLooksBinary(abs), nil
}

func isAllowedRootBuildArtifactName(root Root, name string) bool {
	if name == filepath.Base(root.Abs()) {
		return true
	}
	return name == goModuleBinaryName(root)
}

func goModuleBinaryName(root Root) string {
	abs, err := root.ResolvePath("go.mod")
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || fields[0] != "module" {
			continue
		}
		modulePath := strings.Trim(fields[1], `"`)
		moduleName := filepath.Base(strings.TrimSuffix(modulePath, "/"))
		if moduleName == "." || moduleName == string(filepath.Separator) || strings.Contains(moduleName, "/") {
			return ""
		}
		return moduleName
	}
	return ""
}

func fileLooksBinary(abs string) bool {
	f, err := os.Open(abs)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	return bytes.Contains(buf[:n], []byte{0})
}

func lineListContains(output, want string) bool {
	want = cleanRepoPath(want)
	for _, line := range strings.Split(output, "\n") {
		if cleanRepoPath(line) == want {
			return true
		}
	}
	return false
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
		abs, err := root.ResolvePath(source)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
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

func shellExecMovesBacklogTicketToInProgress(raw json.RawMessage) bool {
	return shellExecMovesTicketToInProgress(raw)
}

func shellExecMovesTicketToInProgress(raw json.RawMessage) bool {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return false
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
			return true
		}
	}
	return false
}

func shellExecMovesInProgressTicketToDone(raw json.RawMessage) bool {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return false
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
		_, state, ok := ticketLifecyclePathIdentity(cleanRepoPath(cleanShellPathToken(source)))
		if ok && state == "in-progress" {
			return true
		}
	}
	return false
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
		dir, err := root.ResolvePath(filepath.Join("docs", "tickets", candidateState))
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
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

func uniqueStringsPreserveOrder(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func dependencyShellOperation(cmd string) (string, bool) {
	fields := shellFields(cmd)
	if len(fields) == 0 {
		return "", false
	}
	for i, field := range fields {
		switch filepathBase(field) {
		case "npm":
			if nextTokenIs(fields, i, "install") || nextTokenIs(fields, i, "i") || nextTokenIs(fields, i, "ci") {
				return "npm " + fields[i+1], true
			}
		case "pnpm", "yarn", "bun":
			if nextTokenIs(fields, i, "install") || nextTokenIs(fields, i, "i") {
				return filepathBase(field) + " " + fields[i+1], true
			}
		case "go":
			if nextTokenIs(fields, i, "get") {
				return "go get", true
			}
			if i+2 < len(fields) && fields[i+1] == "mod" && fields[i+2] == "download" {
				return "go mod download", true
			}
		case "cargo":
			if nextTokenIs(fields, i, "fetch") {
				return "cargo fetch", true
			}
		case "pip", "pip3":
			if nextTokenIs(fields, i, "install") {
				return filepathBase(field) + " install", true
			}
		case "python", "python3":
			if i+3 < len(fields) && fields[i+1] == "-m" && fields[i+2] == "pip" && fields[i+3] == "install" {
				return filepathBase(field) + " -m pip install", true
			}
		case "bundle":
			if nextTokenIs(fields, i, "install") {
				return "bundle install", true
			}
		case "composer":
			if nextTokenIs(fields, i, "install") {
				return "composer install", true
			}
		}
	}
	return "", false
}

func nextTokenIs(fields []string, index int, token string) bool {
	return index+1 < len(fields) && fields[index+1] == token
}

func broadGeneratedTraversal(cmd string) (string, bool) {
	fields := shellFields(cmd)
	if len(fields) == 0 || hasGeneratedExcludeToken(fields) {
		return "", false
	}
	for i, field := range fields {
		switch filepathBase(field) {
		case "find":
			if i+1 < len(fields) && (fields[i+1] == "." || fields[i+1] == "./") {
				return "find .", true
			}
		case "ls":
			if hasToken(fields[i+1:], "-r") {
				for _, token := range fields[i+1:] {
					if token == "." || token == "./" {
						return "ls recursive/broad root", true
					}
				}
			}
		case "cat":
			for _, token := range fields[i+1:] {
				if token == "." || token == "./" {
					return "cat .", true
				}
			}
		}
	}
	return "", false
}

func hasGeneratedExcludeToken(fields []string) bool {
	for _, field := range fields {
		for _, dir := range generatedWorkspaceDirs {
			if strings.Contains(field, dir) {
				return true
			}
		}
	}
	return false
}

func checkShellTicketPathPolicy(cmd string) error {
	for _, field := range shellFields(cmd) {
		rel := cleanShellPathToken(field)
		if rel == "" {
			continue
		}
		lowerRel := strings.ToLower(cleanRepoPath(rel))
		if lowerRel == "" || lowerRel == "docs/tickets/readme.md" {
			continue
		}
		if !strings.HasPrefix(lowerRel, "docs/tickets/") || !strings.HasSuffix(lowerRel, ".md") {
			continue
		}
		parts := strings.Split(lowerRel, "/")
		if len(parts) < 4 || !isTicketLifecycleDir(parts[2]) {
			return fmt.Errorf("policy: ticket markdown must live under docs/tickets/backlog, docs/tickets/in-progress, docs/tickets/in-review, or docs/tickets/done; use ticket_create for new tickets instead of shell_exec to %s", rel)
		}
	}
	return nil
}

func cleanShellPathToken(field string) string {
	field = strings.TrimSpace(field)
	field = strings.TrimPrefix(field, "1>")
	field = strings.TrimPrefix(field, "2>")
	field = strings.TrimLeft(field, "><")
	field = strings.TrimPrefix(field, "./")
	return strings.Trim(field, `"'`)
}

func shellExecReadOnly(raw json.RawMessage) bool {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return false
	}
	if shellExecNoop(args) {
		return true
	}
	if args.Background {
		return false
	}
	if len(args.Argv) > 0 {
		return shellTokensReadOnly(args.Argv)
	}
	cmd := strings.TrimSpace(args.ShellCommand)
	if cmd == "" || shellCommandHasControlSyntax(cmd) {
		return false
	}
	return shellTokensReadOnly(shellFields(cmd))
}

func shellCommandHasControlSyntax(cmd string) bool {
	for _, token := range []string{"|", "&&", "||", ";", ">", "<", "`", "$(", "\n"} {
		if strings.Contains(cmd, token) {
			return true
		}
	}
	return false
}

func shellTokensReadOnly(raw []string) bool {
	if len(raw) == 0 {
		return false
	}
	fields := make([]string, 0, len(raw))
	for _, field := range raw {
		field = strings.TrimSpace(strings.ToLower(field))
		field = strings.Trim(field, `"'`)
		if field != "" {
			fields = append(fields, field)
		}
	}
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "ls", "pwd", "cat", "head", "tail", "wc", "test", "grep", "rg":
		return true
	case "sed":
		return sedReadOnly(fields[1:])
	case "find":
		return findReadOnly(fields[1:])
	case "git":
		return gitShellReadOnly(fields)
	default:
		return false
	}
}

func filepathBase(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func sedReadOnly(args []string) bool {
	hasNoPrint := false
	for _, arg := range args {
		switch {
		case arg == "-n":
			hasNoPrint = true
		case strings.HasPrefix(arg, "-") && strings.Contains(arg, "n"):
			hasNoPrint = true
		case arg == "-i" || arg == "--in-place" || strings.HasPrefix(arg, "-i"):
			return false
		}
	}
	return hasNoPrint
}

func findReadOnly(args []string) bool {
	for _, arg := range args {
		switch {
		case arg == "-delete", arg == "-exec", arg == "-execdir", arg == "-ok", arg == "-okdir":
			return false
		case strings.HasPrefix(arg, "-fprint"):
			return false
		}
	}
	return true
}

func gitShellReadOnly(fields []string) bool {
	subcommand, args := gitShellSubcommand(fields)
	switch subcommand {
	case "status", "diff", "log", "show", "rev-parse", "ls-files":
		return !hasGitShellOutputFlag(args)
	case "branch":
		return len(args) == 1 && args[0] == "--show-current"
	default:
		return false
	}
}

func gitShellSubcommand(fields []string) (string, []string) {
	if len(fields) == 0 || filepathBase(fields[0]) != "git" {
		return "", nil
	}
	for i := 1; i < len(fields); i++ {
		token := fields[i]
		switch {
		case token == "-c" || token == "-C":
			i++
			continue
		case strings.HasPrefix(token, "--git-dir") || strings.HasPrefix(token, "--work-tree"):
			continue
		case strings.HasPrefix(token, "-"):
			continue
		default:
			return token, fields[i+1:]
		}
	}
	return "", nil
}

func hasGitShellOutputFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--output" || strings.HasPrefix(arg, "--output=") {
			return true
		}
	}
	return false
}

func forbiddenShellOperation(cmd string) (string, bool) {
	fields := shellFields(cmd)
	if len(fields) == 0 {
		return "", false
	}
	if hasGitSubcommand(fields, "push") && hasGitForcePushFlag(fields) {
		return "git push --force", true
	}
	if hasGitSubcommand(fields, "reset") && hasToken(fields, "--hard") {
		return "git reset --hard", true
	}
	if hasGitSubcommand(fields, "clean") && hasGitCleanForceDelete(fields) {
		return "git clean -fd", true
	}
	if hasGitSubcommand(fields, "branch") && hasGitBranchDelete(fields) {
		return "git branch -d", true
	}
	if hasGitSubcommand(fields, "rm") {
		return "git rm", true
	}
	if hasGitSubcommand(fields, "checkout") && hasToken(fields, "-b") {
		return "git checkout -b", true
	}
	if hasRootRemoval(fields) {
		return "rm -rf /", true
	}
	if operation, ok := hasShellRemoval(fields); ok {
		return operation, true
	}
	if hasFindDelete(fields) {
		return "find -delete", true
	}
	return "", false
}

func shellFields(cmd string) []string {
	raw := strings.Fields(strings.ToLower(cmd))
	fields := make([]string, 0, len(raw))
	for _, field := range raw {
		field = strings.Trim(field, `"'`)
		field = strings.TrimRight(field, ";")
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

func shellFieldsPreserveCase(cmd string) []string {
	raw := strings.Fields(cmd)
	fields := make([]string, 0, len(raw))
	for _, field := range raw {
		field = strings.Trim(field, `"'`)
		field = strings.TrimRight(field, ";")
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

func hasGitSubcommand(fields []string, subcommand string) bool {
	for i := 0; i < len(fields); i++ {
		if fields[i] != "git" && !strings.HasSuffix(fields[i], "/git") {
			continue
		}
		for j := i + 1; j < len(fields); j++ {
			token := fields[j]
			if token == "-c" && j+1 < len(fields) {
				j++
				continue
			}
			if strings.HasPrefix(token, "--git-dir") || strings.HasPrefix(token, "--work-tree") {
				continue
			}
			if token == subcommand {
				return true
			}
			if strings.HasPrefix(token, "-") {
				continue
			}
			break
		}
	}
	return false
}

func hasToken(fields []string, want string) bool {
	for _, field := range fields {
		if field == want {
			return true
		}
	}
	return false
}

func hasGitForcePushFlag(fields []string) bool {
	for _, field := range fields {
		switch {
		case field == "-f", field == "--force", field == "--force-with-lease":
			return true
		case strings.HasPrefix(field, "--force="), strings.HasPrefix(field, "--force-with-lease="):
			return true
		}
	}
	return false
}

func hasGitCleanForceDelete(fields []string) bool {
	force := false
	dirs := false
	for _, field := range fields {
		switch {
		case field == "-f", field == "--force":
			force = true
		case field == "-d":
			dirs = true
		case strings.HasPrefix(field, "-") && !strings.HasPrefix(field, "--"):
			if strings.Contains(field, "f") {
				force = true
			}
			if strings.Contains(field, "d") {
				dirs = true
			}
		}
	}
	return force && dirs
}

func hasGitBranchDelete(fields []string) bool {
	for _, field := range fields {
		switch field {
		case "-d", "--delete", "--force":
			return true
		}
	}
	return false
}

func hasRootRemoval(fields []string) bool {
	for i := 0; i < len(fields); i++ {
		if fields[i] != "rm" {
			continue
		}
		flags := ""
		for j := i + 1; j < len(fields); j++ {
			token := fields[j]
			if token == "--" {
				continue
			}
			if strings.HasPrefix(token, "-") {
				flags += token
				continue
			}
			if token == "/" && strings.Contains(flags, "r") && strings.Contains(flags, "f") {
				return true
			}
		}
	}
	return false
}

func hasShellRemoval(fields []string) (string, bool) {
	for _, field := range fields {
		switch field {
		case "rm", "rmdir", "unlink":
			return field, true
		}
	}
	return "", false
}

func hasFindDelete(fields []string) bool {
	for i, field := range fields {
		if field != "find" && !strings.HasSuffix(field, "/find") {
			continue
		}
		for _, token := range fields[i+1:] {
			if token == "-delete" {
				return true
			}
		}
	}
	return false
}

func validateRepoDiff(ctx context.Context, root Root, session Session) error {
	if err := checkDiffForSecrets(ctx, root); err != nil {
		return err
	}
	limits := session.SafetyLimits
	if limits == (safety.Limits{}) {
		limits = safety.DefaultLimits()
	}
	stats, err := diffStats(ctx, root)
	if err != nil {
		return err
	}
	if err := safety.Check(stats, limits); err != nil {
		if hint := buildArtifactCleanupHint(ctx, root, stats, limits); hint != "" {
			return fmt.Errorf("%w. %s", err, hint)
		}
		return err
	}
	return nil
}

// ValidateRepoDiff checks the current repository diff against the same safety
// limits enforced after mutating tool calls.
func ValidateRepoDiff(ctx context.Context, root Root, session Session) error {
	return validateRepoDiff(ctx, root, session)
}

func buildArtifactCleanupHint(ctx context.Context, root Root, stats safety.DiffStats, limits safety.Limits) string {
	if limits.MaxLinesPerFile <= 0 {
		return ""
	}
	for rel, lines := range stats.LinesPerFile {
		if lines <= limits.MaxLinesPerFile {
			continue
		}
		generated, err := isUntrackedRootBuildArtifact(ctx, root, rel)
		if err != nil || !generated {
			continue
		}
		return fmt.Sprintf("Generated build artifact %q can be cleaned with `rm %s`, then rerun the blocked command", rel, rel)
	}
	return ""
}

func checkDiffForSecrets(ctx context.Context, root Root) error {
	files, err := changedFiles(ctx, root)
	if err != nil {
		return err
	}
	for _, rel := range files {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			return err
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if hits := safety.ScanForSecrets(rel, string(b)); len(hits) > 0 {
			return fmt.Errorf("policy: secret scanner blocked %s:%d (%s)", hits[0].File, hits[0].Line, hits[0].Pattern)
		}
	}
	return nil
}

func changedFiles(ctx context.Context, root Root) ([]string, error) {
	seen := map[string]bool{}
	tr, err := runGit(ctx, root, "diff", "--name-only", "HEAD", "--")
	if err != nil {
		return nil, err
	}
	var files []string
	if tr.ExitCode == 0 {
		for _, line := range strings.Split(tr.Output, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !IsGeneratedWorkspacePath(line) && !seen[line] {
				seen[line] = true
				files = append(files, line)
			}
		}
	}
	untracked, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	if untracked.ExitCode == 0 {
		for _, line := range strings.Split(untracked.Output, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !IsGeneratedWorkspacePath(line) && !seen[line] {
				seen[line] = true
				files = append(files, line)
			}
		}
	}
	return files, nil
}

func diffStats(ctx context.Context, root Root) (safety.DiffStats, error) {
	stats := safety.DiffStats{LinesPerFile: map[string]int{}}
	numstat, err := runGit(ctx, root, "diff", "--numstat", "HEAD", "--")
	if err != nil {
		return stats, err
	}
	if numstat.ExitCode != 0 {
		return stats, nil
	}
	for _, line := range strings.Split(numstat.Output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		added := atoiDiffField(fields[0])
		deleted := atoiDiffField(fields[1])
		path := strings.Join(fields[2:], " ")
		if IsGeneratedWorkspacePath(path) || IsGeneratedDependencyMetadataPath(path) {
			continue
		}
		lines := added + deleted
		stats.FilesChanged++
		stats.LinesPerFile[path] = lines
		stats.TotalLines += lines
	}
	untracked, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return stats, err
	}
	var untrackedPaths []string
	if untracked.ExitCode == 0 {
		for _, rel := range strings.Split(untracked.Output, "\n") {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}
			untrackedPaths = append(untrackedPaths, rel)
		}
	}
	status, err := runGit(ctx, root, "diff", "--name-status", "HEAD", "--")
	if err != nil {
		return stats, err
	}
	if status.ExitCode != 0 {
		return stats, nil
	}
	var lifecycleCounterpartPaths []string
	lifecycleCounterpartPaths = append(lifecycleCounterpartPaths, untrackedPaths...)
	for _, line := range strings.Split(status.Output, "\n") {
		trimmed := strings.TrimSpace(line)
		fields := strings.Fields(trimmed)
		code := ""
		if len(fields) > 0 {
			code = fields[0]
		}
		for _, path := range diffNameStatusAddedPaths(fields) {
			if _, _, ok := ticketLifecyclePathIdentity(path); ok {
				lifecycleCounterpartPaths = append(lifecycleCounterpartPaths, path)
			}
		}
		if !strings.HasPrefix(code, "D") || len(fields) < 2 {
			continue
		}
		path := fields[len(fields)-1]
		if IsGeneratedWorkspacePath(path) {
			continue
		}
		if isTicketLifecycleMoveDeletion(root, path, lifecycleCounterpartPaths) {
			continue
		}
		stats.Deletions++
	}
	if untracked.ExitCode == 0 {
		for _, rel := range untrackedPaths {
			rel = strings.TrimSpace(rel)
			if rel == "" || IsGeneratedWorkspacePath(rel) || IsGeneratedDependencyMetadataPath(rel) {
				continue
			}
			abs, err := root.ResolvePath(rel)
			if err != nil {
				return stats, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			lines := strings.Count(string(b), "\n")
			if len(b) > 0 && !strings.HasSuffix(string(b), "\n") {
				lines++
			}
			stats.FilesChanged++
			stats.LinesPerFile[rel] = lines
			stats.TotalLines += lines
		}
	}
	return stats, nil
}

func diffNameStatusAddedPaths(fields []string) []string {
	if len(fields) < 2 {
		return nil
	}
	code := fields[0]
	switch {
	case strings.HasPrefix(code, "A"):
		return []string{fields[len(fields)-1]}
	case strings.HasPrefix(code, "R"), strings.HasPrefix(code, "C"):
		if len(fields) >= 3 {
			return []string{fields[len(fields)-1]}
		}
	}
	return nil
}

func isTicketLifecycleMoveDeletion(root Root, deletedPath string, candidatePaths []string) bool {
	deletedID, deletedState, ok := ticketLifecyclePathIdentity(deletedPath)
	if !ok {
		return false
	}
	if ticketLifecycleCounterpartInCandidates(deletedPath, deletedID, deletedState, candidatePaths) {
		return true
	}
	return ticketLifecycleCounterpartExists(root, deletedPath, deletedID, deletedState)
}

func ticketLifecycleCounterpartInCandidates(deletedPath, deletedID, deletedState string, candidatePaths []string) bool {
	for _, rel := range candidatePaths {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" || rel == filepath.ToSlash(deletedPath) {
			continue
		}
		addedID, addedState, ok := ticketLifecyclePathIdentity(rel)
		if !ok {
			continue
		}
		if addedID == deletedID && addedState != deletedState {
			return true
		}
	}
	return false
}

func ticketLifecycleCounterpartExists(root Root, deletedPath, deletedID, deletedState string) bool {
	pattern := filepath.Join(root.Abs(), "docs", "tickets", "*", "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	deletedPath = filepath.ToSlash(deletedPath)
	for _, match := range matches {
		rel, err := filepath.Rel(root.Abs(), match)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == deletedPath {
			continue
		}
		id, state, ok := ticketLifecyclePathIdentity(rel)
		if ok && id == deletedID && state != deletedState {
			return true
		}
	}
	return false
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

func atoiDiffField(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
