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
	"sort"
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

var toolsFeatureScenarioIDPattern = regexp.MustCompile(`\bF-\d{3}-S\d{3}\b`)
var toolsFeatureScenarioHeadingPattern = regexp.MustCompile(`(?mi)^###\s+(F-\d{3}-S\d{3})\b.*$`)

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
	ticketDoneMoveSuccessKey                  = "ticket:lifecycle_done_move:success"
	ticketDoneMoveLastIDKey                   = "ticket:lifecycle_done_move:last_id"
	ctoHandoffRequiredScenariosKey            = "cto:handoff_required_scenarios"
	reviewTerminalDispositionRequiredKey      = "review:terminal_disposition:required"
	ticketCreationOutstandingFailureKey       = "ticket_create:failure:outstanding"
	browserProductSmokeSuccessKey             = "validation:browser_product_smoke:success"
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
		return checkTicketCreatePolicy(ctx, root, session, hasSession, raw)
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
		if err := checkShellNodeCheckHTMLPolicy(args); err != nil {
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
		if err := checkEngineerShellExecBeforeTicketClaim(ctx, root, session, hasSession, raw, generatedArtifactCleanup); err != nil {
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
		if err := checkEngineerBrowserFrameworkTicketDoneMovePolicy(root, session, hasSession, raw); err != nil {
			return err
		}
		if err := checkShellTicketDoneEvidencePolicy(ctx, root, raw); err != nil {
			return err
		}
		if generatedArtifactCleanup {
			return nil
		}
		if hasSession && planningRoleCannotMutateWithShell(session.Role) && !shellExecReadOnly(raw) {
			role := strings.ToLower(strings.TrimSpace(session.Role))
			return fmt.Errorf("policy: %s cannot run mutating shell_exec; use file_write for owned planning artifacts, git tools for commit/status, ticket_create for tickets, and Engineer/dependency tools for implementation or dependency changes", role)
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

func planningRoleCannotMutateWithShell(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "ceo", "head-of-strategy", "coo", "cto", "cto-weekly":
		return true
	default:
		return false
	}
}

func checkTicketCreatePolicy(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession {
		return nil
	}
	var args ticketCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	args = withInferredTicketCreateScenarios(ctx, args)
	if err := checkTicketCreatePlanningOrder(root, session, hasSession, args); err != nil {
		return err
	}
	if err := checkBrowserFrameworkTicketCreatePolicy(root, session, hasSession, args); err != nil {
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

func checkBrowserFrameworkTicketCreatePolicy(root Root, session Session, hasSession bool, args ticketCreateArgs) error {
	if !hasSession {
		return nil
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role != "cto" && role != "cto-weekly" {
		return nil
	}
	workType := normalizeWorkType(args.Kind, args.WorkType)
	if workType != "feature" {
		return nil
	}
	if err := checkProductCapabilityScenarioCoverage(root); err != nil {
		return err
	}
	if !projectBriefMentionsFramework(root, "phaser") || projectBriefNamesGoBackend(root) {
		return nil
	}
	body := strings.ToLower(args.Title + "\n" + args.Source + "\n" + args.Body)
	badGoShape := []string{"go.mod", "go module", "go cli", "golang", "cmd/"}
	for _, marker := range badGoShape {
		if strings.Contains(body, marker) {
			return fmt.Errorf("policy: Phaser/JavaScript target tickets must default to a browser JavaScript shape such as package.json, index.html, and src/*.js with npm run build evidence. Do not prescribe Go CLI paths, go.mod, or cmd/* unless the README explicitly names a Go backend")
		}
	}
	if phaserTicketPrescribesCDNRuntime(body) {
		return fmt.Errorf("policy: Phaser/JavaScript target tickets must require a local phaser npm dependency, package build evidence, and browser-product smoke evidence. Do not prescribe CDN-only Phaser script tags or CDN loading acceptance criteria")
	}
	return nil
}

func phaserTicketPrescribesCDNRuntime(body string) bool {
	body = strings.ToLower(strings.TrimSpace(body))
	if !strings.Contains(body, "phaser") || !strings.Contains(body, "cdn") {
		return false
	}
	negated := regexp.MustCompile(`\b(no|not|avoid|without|disallow|reject|block|cannot|can't|must not|do not|don't|never)\b[^.\n]{0,64}\bcdn\b`)
	if negated.MatchString(body) {
		return false
	}
	badPhrases := []string{
		"cdn-only",
		"cdn script",
		"script tag",
		"load from cdn",
		"loads from cdn",
		"loaded from cdn",
		"loaded by cdn",
		"use cdn",
		"uses cdn",
		"using cdn",
	}
	for _, phrase := range badPhrases {
		if strings.Contains(body, phrase) {
			return true
		}
	}
	return false
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
				return fmt.Errorf("policy: feature ticket_create is missing bdd_scenarios. Retry ticket_create with bdd_scenarios as a JSON array, for example %s, matching the next product scenario(s) before Engineer handoff", quoteStringArray(next))
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
			if firstMissing != "" {
				return fmt.Errorf("policy: feature ticket_create cannot include already-covered scenario(s) %s for %s. Create the next ticket for %s only, or group it with later uncovered adjacent scenarios", strings.Join(alreadyCovered, ", "), id, firstMissing)
			}
			return fmt.Errorf("policy: feature ticket_create cannot include already-covered scenario(s) %s for %s; all contract scenarios appear to be ticketed already", strings.Join(alreadyCovered, ", "), id)
		}
		firstMissing := firstUncoveredFeatureScenario(root, id)
		if firstMissing != "" && !scenarioListContains(args.BDDScenarios, firstMissing) {
			return fmt.Errorf("policy: feature ticket_create must start with the earliest uncovered scenario %s for %s. Create the next ticket from that scenario, or include it in this scenario group before later scenarios", firstMissing, id)
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
	featuresDir, err := root.ResolvePath(filepath.Join("docs", "features"))
	if err != nil {
		return ""
	}
	matches, err := filepath.Glob(filepath.Join(featuresDir, featureID+"*.md"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	return matches[0]
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
	data, err := os.ReadFile(featurePath)
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
			data, err := os.ReadFile(filepath.Join(root.Abs(), filepath.FromSlash(t.RelPath)))
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
	featuresDir, err := root.ResolvePath(filepath.Join("docs", "features"))
	if err != nil {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(featuresDir, "F-*.md"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, match := range matches {
		if data, err := os.ReadFile(match); err == nil && featureContractSuperseded(string(data)) {
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

func featureContractSuperseded(content string) bool {
	return regexp.MustCompile(`(?im)^\s*-\s*status:\s*superseded\b`).MatchString(content)
}

func countCoveredFeatureScenarios(scenarios []string, covered map[string]bool) int {
	n := 0
	for _, scenario := range scenarios {
		if covered[scenario] {
			n++
		}
	}
	return n
}

func firstUncoveredFeatureScenarios(scenarios []string, covered map[string]bool, limit int) []string {
	if limit <= 0 {
		return nil
	}
	var out []string
	for _, scenario := range scenarios {
		if covered[scenario] {
			continue
		}
		out = append(out, scenario)
		if len(out) >= limit {
			break
		}
	}
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
	if toolName == "git_commit" && gitCommitTouchesOnlyWorkspaceNoise(ctx, root, raw) {
		return nil
	}
	if toolName == "git_commit" && worktreeHasInProgressToDoneTicketMove(ctx, root) {
		return nil
	}
	if engineerCompletedTicketThisRun(session) {
		return fmt.Errorf("policy: engineer already moved product ticket %s to docs/tickets/done in this run. Do not claim or mutate another ticket in the same job; commit any remaining lifecycle changes, git_push if a remote exists, then record job_disposition_record with ticket_id %s and next_need qa_review", engineerCompletedTicketID(session), engineerCompletedTicketID(session))
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

func checkEngineerPostValidationCompletionShellPolicy(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[validationCommandSuccessKey] == 0 {
		return nil
	}
	if shellExecMovesInProgressTicketToDone(raw) {
		return nil
	}
	args, err := decodeShellExecArgs(raw)
	if err == nil && shellExecRunsRecordedValidationArtifact(session, root, args) {
		return nil
	}
	if err == nil && shellExecStopsTrackedBackgroundPID(args) {
		return nil
	}
	if err == nil && engineerPostCommitBrowserValidationAllowed(root, session, args) {
		return nil
	}
	if err == nil {
		if blockErr := checkEngineerBrowserPostBuildSmokeOnlyPolicy(ctx, root, session, args); blockErr != nil {
			return blockErr
		}
	}
	if counts["tool:git_commit:success"] == 0 {
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
		if engineerBrowserFrameworkEvidenceComplete(root, session) {
			return fmt.Errorf(
				"policy: engineer already has successful browser-framework build and product-smoke validation with dirty implementation or ticket work for %s. Do not call shell_exec again except tracked PID cleanup. %s",
				tickets[0].ID,
				engineerDirtyPostValidationGuidance(tickets[0], blockingFiles),
			)
		}
		return nil
	}
	if blockers := engineerBrowserFrameworkCompletionBlockers(root, session); len(blockers) > 0 {
		return fmt.Errorf(
			"policy: engineer cannot close browser-framework product ticket %s yet: %s. Fix the source or package validation surface, rerun validation, then update ticket evidence and move the ticket to done",
			tickets[0].ID,
			strings.Join(blockers, "; "),
		)
	}
	return fmt.Errorf(
		"policy: engineer already has successful validation and a clean implementation commit while product ticket %s remains in progress. Do not call shell_exec again except the exact lifecycle move. Next use file_read on %q, then file_write the same ticket with evidence_links and verified_by populated, then run shell_exec argv [\"git\", \"mv\", %q, \"docs/tickets/done/\"], commit the lifecycle move, and record job_disposition_record with ticket_id %s and next_need qa_review",
		tickets[0].ID,
		tickets[0].RelPath,
		tickets[0].RelPath,
		tickets[0].ID,
	)
}

func engineerBrowserFrameworkEvidenceComplete(root Root, session Session) bool {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return false
	}
	counts := session.ToolCounts
	if counts == nil || counts[validationCommandSuccessKey] == 0 {
		return false
	}
	return len(engineerBrowserFrameworkCompletionBlockers(root, session)) == 0
}

func checkEngineerBrowserPostBuildSmokeOnlyPolicy(ctx context.Context, root Root, session Session, args shellExecArgs) error {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework || !browserFrameworkRequiresProductSmoke(root) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[buildCommandSuccessKey] == 0 || counts[browserProductSmokeSuccessKey] > 0 {
		return nil
	}
	if shellExecRunsBuildCommand(args) || shellExecRunsBrowserProductSmokeCommand(args) || shellExecStopsTrackedBackgroundPID(args) {
		return nil
	}
	files, err := changedFiles(ctx, root)
	if err != nil || len(dispositionBlockingFiles(files)) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer has successful browser-framework build evidence but still needs browser-product smoke before more shell validation. Run %s. Do not inspect dist/assets, require('phaser'), require browser bundles from Node, run node --check on HTML, or use trivial environment probes as substitutes for mounted product UI evidence",
		browserProductSmokeCommandGuidance(root),
	)
}

func engineerPostCommitBrowserValidationAllowed(root Root, session Session, args shellExecArgs) bool {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return false
	}
	counts := session.ToolCounts
	if counts == nil {
		return false
	}
	if info.HasBuildScript && counts[buildCommandSuccessKey] == 0 && shellExecRunsBuildCommand(args) {
		return true
	}
	if browserFrameworkRequiresProductSmoke(root) && counts[buildCommandSuccessKey] > 0 &&
		counts[browserProductSmokeSuccessKey] == 0 && shellExecRunsBrowserProductSmokeCommand(args) {
		return true
	}
	return false
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
	if !sourceFileRequiresDocSync(rel) && !pathLooksLikeFixtureOrTestdata(rel) && !pathLooksLikeTestFile(rel) {
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
	if pathLooksLikeTestFile(lowerRel) {
		return true
	}
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

func pathLooksLikeTestFile(rel string) bool {
	lowerRel := strings.ToLower(filepath.ToSlash(cleanRepoPath(rel)))
	if lowerRel == "" {
		return false
	}
	base := filepath.Base(lowerRel)
	for _, suffix := range []string{
		"_test.go",
		".test.js",
		".test.jsx",
		".test.ts",
		".test.tsx",
		".test.mjs",
		".spec.js",
		".spec.jsx",
		".spec.ts",
		".spec.tsx",
		".spec.mjs",
	} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	wrapped := "/" + lowerRel + "/"
	return strings.Contains(wrapped, "/tests/") || strings.Contains(wrapped, "/test/")
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
			parts = append(parts, "Latest failing output (compact): "+compactPolicyFailureOutput(output))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "Rerun a same-lane go test, npm test, cargo test, go build, or equivalent build/test command successfully before continuing.")
	}
	parts = append(parts, "If the failing assertion matches the ticket, README, or BDD contract, edit the implementation rather than deleting or weakening the test.")
	return " " + strings.Join(parts, " ")
}

func compactPolicyFailureOutput(output string) string {
	output = strings.Join(strings.Fields(output), " ")
	output, truncated := TruncateUTF8(output, 180)
	output = strings.TrimSpace(output)
	if truncated {
		output += "..."
	}
	return output
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
	if pathLooksLikeTestFile(lowerRel) {
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
	var out []ticketstate.Ticket
	for _, status := range []string{ticketstate.StatusInReview, ticketstate.StatusDone} {
		tickets, err := ticketstate.ListStatus(root.Abs(), status)
		if err != nil {
			return nil, err
		}
		out = append(out, ordinaryProductTickets(tickets)...)
	}
	if targetID := engineerReworkTicketIDFromTrigger(session.Trigger); targetID != "" {
		for _, ticket := range out {
			if strings.EqualFold(strings.TrimSpace(ticket.ID), targetID) {
				return []ticketstate.Ticket{ticket}, nil
			}
		}
		return nil, nil
	}
	return out, nil
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
	if err := checkEngineerTicketEvidenceWriteRequiresValidation(root, session, hasSession, args.Path, args.Content); err != nil {
		return err
	}
	if err := checkEngineerUnresolvedTestBuildValidationBeforeFileWrite(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkEngineerUnresolvedRuntimeValidationBeforeDoneFileWrite(session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkFeatureFileWritePolicy(root, session, hasSession, args.Path); err != nil {
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
	if err := checkEngineerBrowserFrameworkImplementationShapePolicy(root, session, hasSession, args.Path); err != nil {
		return err
	}
	if err := checkEngineerBrowserFrameworkPackageWritePolicy(root, session, hasSession, args.Path, args.Content); err != nil {
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
	case "debug", "probe", "scratch", "tmp", "temp", "test", "validate", "validation", "verify", "smoke", "smoke-test", "test-server":
		return true
	default:
		return strings.HasPrefix(stem, "test-") ||
			strings.HasSuffix(stem, "-test") ||
			strings.Contains(stem, "validate") ||
			strings.Contains(stem, "validation") ||
			strings.Contains(stem, "smoke") ||
			strings.Contains(stem, "probe") ||
			strings.Contains(stem, "scratch") ||
			strings.Contains(stem, "verify")
	}
}

func rootScratchValidationExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".sh", ".go", ".html", ".htm", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py", ".rb":
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

func checkEngineerBrowserFrameworkImplementationShapePolicy(root Root, session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if !projectBriefMentionsFramework(root, "phaser") || projectBriefNamesGoBackend(root) {
		return nil
	}
	rel = cleanRepoPath(rel)
	lower := strings.ToLower(rel)
	if lower == "go.mod" || strings.HasSuffix(lower, "/go.mod") || strings.HasSuffix(lower, ".go") && strings.HasPrefix(lower, "cmd/") {
		return fmt.Errorf("policy: Phaser/JavaScript target implementation should use package.json, index.html, and src/*.js with local phaser dependency/build evidence. Do not add Go module or cmd/*.go scaffolding unless README explicitly names a Go backend")
	}
	return nil
}

func checkEngineerBrowserFrameworkPackageWritePolicy(root Root, session Session, hasSession bool, rel, content string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if !projectBriefMentionsFramework(root, "phaser") || projectBriefNamesGoBackend(root) {
		return nil
	}
	rel = cleanRepoPath(rel)
	lower := strings.ToLower(rel)
	switch lower {
	case "package.json":
		var pkg struct {
			Scripts         map[string]string `json:"scripts"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal([]byte(content), &pkg); err != nil {
			return nil
		}
		hasPhaser := false
		for dep := range pkg.Dependencies {
			if strings.EqualFold(strings.TrimSpace(dep), "phaser") {
				hasPhaser = true
			}
		}
		for dep := range pkg.DevDependencies {
			if strings.EqualFold(strings.TrimSpace(dep), "phaser") {
				hasPhaser = true
			}
		}
		if !hasPhaser {
			return fmt.Errorf("policy: Phaser browser targets must declare a local phaser npm dependency in package.json; do not rely on CDN-only runtime")
		}
		hasBuild := false
		for name, script := range pkg.Scripts {
			if buildScriptName(name) && !packageBuildScriptNoop(script) {
				hasBuild = true
				break
			}
		}
		if !hasBuild {
			return fmt.Errorf("policy: Phaser browser targets must include a deterministic package build script in package.json, such as vite build, tsc --noEmit, or another command that fails on broken source; echo, true, and node --check-only scripts are not enough")
		}
		for name, script := range pkg.Scripts {
			if !runtimeScriptName(name) {
				continue
			}
			if port := reservedHarnessPortInScript(script); port != "" {
				return fmt.Errorf("policy: package.json script %q uses reserved Mars Harness port %s. Use an application dev port such as 5173 so target servers do not collide with local inference/runtime ports", name, port)
			}
			if phaserRuntimeScriptUsesStaticSourceServer(script) {
				return fmt.Errorf("policy: package.json script %q starts a static source server for a Phaser app. Use Vite dev/preview, for example `vite --host 127.0.0.1 --port 5173` or `npm run build && vite preview --host 127.0.0.1 --port 5173`, so local npm modules are bundled correctly", name)
			}
		}
	}
	if htmlSourcePath(lower) {
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, "<script") && strings.Contains(lowerContent, "phaser") &&
			(strings.Contains(lowerContent, "http://") || strings.Contains(lowerContent, "https://") || strings.Contains(lowerContent, "cdn.")) {
			return fmt.Errorf("policy: Phaser browser targets should use the local phaser npm dependency and package build/runtime validation, not a CDN-only Phaser script tag in index.html")
		}
	}
	switch lower {
	case "vite.config.js", "vite.config.ts":
		if viteConfigImportsPhaserRuntime(content) {
			return fmt.Errorf("policy: Phaser Vite config runs in Node during build and must not import Phaser, browser globals, or src/* game modules. Keep vite.config limited to Vite/plugin configuration, and import Phaser/game code from the browser entrypoint instead")
		}
		if viteConfigExternalizesPhaser(content) {
			return fmt.Errorf("policy: Phaser Vite config must not externalize phaser from the production bundle; remove rollupOptions.external entries for phaser so npm run build proves the browser can load the local dependency")
		}
	}
	if javascriptSourcePath(lower) && !browserFrameworkValidationHelperPath(lower) {
		if findings := phaserSingleFileSourceFindings(rel, content); len(findings) > 0 {
			return fmt.Errorf("policy: Phaser source file has lifecycle/import issue: %s", strings.Join(findings, "; "))
		}
	}
	return nil
}

func reservedHarnessPortInScript(script string) string {
	for _, port := range []string{"18080", "18081", "18082", "18083", "18084", "18085", "18086", "18087", "18088", "18089"} {
		if regexp.MustCompile(`(^|[^0-9])` + regexp.QuoteMeta(port) + `([^0-9]|$)`).MatchString(script) {
			return port
		}
	}
	return ""
}

func phaserRuntimeScriptUsesStaticSourceServer(script string) bool {
	script = strings.ToLower(strings.TrimSpace(script))
	for _, marker := range []string{
		"python -m http.server",
		"python3 -m http.server",
		"http-server",
		"live-server",
	} {
		if strings.Contains(script, marker) {
			return true
		}
	}
	if regexp.MustCompile(`(^|[;&|]\s*)serve(?:\s+|$)`).MatchString(script) && !strings.Contains(script, "vite preview") {
		return true
	}
	return false
}

func viteConfigImportsPhaserRuntime(content string) bool {
	lower := strings.ToLower(content)
	runtimeMarkers := []string{
		"from 'phaser'",
		`from "phaser"`,
		"require('phaser')",
		`require("phaser")`,
		"from './src",
		`from "./src`,
		"from '../src",
		`from "../src`,
		"require('./src",
		`require("./src`,
		"require('../src",
		`require("../src`,
	}
	for _, marker := range runtimeMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func viteConfigExternalizesPhaser(content string) bool {
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "external") || !strings.Contains(lower, "phaser") {
		return false
	}
	externalRe := regexp.MustCompile(`(?s)\bexternal\s*:\s*(?:\[[^\]]*['"]phaser['"]|['"]phaser['"]|\([^)]*phaser)`)
	return externalRe.MatchString(lower)
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
		if strings.HasPrefix(rel, "docs/exec-plans/active/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
			return fmt.Errorf("policy: keep exactly one active exec plan; update docs/exec-plans/active/current-operating-plan.md with the current failing scenario instead of creating %s", rel)
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
	if session.ToolCounts != nil && session.ToolCounts[validationCommandSuccessKey] > 0 {
		if blockers := engineerBrowserFrameworkCompletionBlockers(root, session); len(blockers) > 0 {
			return fmt.Errorf(
				"policy: engineer cannot populate ticket evidence for browser-framework work in %s yet: %s. Add or fix package build/browser validation, run it successfully in this job, then update evidence_links with the concrete commands",
				rel,
				strings.Join(blockers, "; "),
			)
		}
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

func checkFeatureFileWritePolicy(root Root, session Session, hasSession bool, rel string) error {
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
	if hasSession && !roleMayWriteFeatureContracts(session.Role) {
		role := strings.ToLower(strings.TrimSpace(session.Role))
		return fmt.Errorf("policy: feature contract %s already exists as %s; %s cannot write feature contracts. Record strategy in allowed strategy artifacts or hand off to COO to update %s; do not create duplicate feature path %s", featureID, existing, role, existing, rel)
	}
	return fmt.Errorf("policy: feature contract %s already exists as %s; update the canonical contract instead of creating duplicate feature path %s", featureID, existing, rel)
}

func roleMayWriteFeatureContracts(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", "coo":
		return true
	default:
		return false
	}
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
	if err := checkPlanningDispositionFeatureSpecificity(root, session, args.Status); err != nil {
		return err
	}
	if err := checkCTODispositionTicketBatch(root, session, args.Status, args.NextNeed, args.SuggestedRole); err != nil {
		return err
	}
	if err := checkReviewDispositionValidationEvidence(root, session, args.Status, args.TicketID); err != nil {
		return err
	}
	if err := checkReviewChangesRequestedFeedbackOwnership(root, session, args.Status, args.NextNeed, raw); err != nil {
		return err
	}
	if err := checkDogfoodDispositionValidationEvidence(root, session, args.Status); err != nil {
		return err
	}
	if err := checkSuccessfulDispositionUnresolvedTicketCreation(root, session, args.Status, args.NextNeed, args.SuggestedRole); err != nil {
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

func checkCTODispositionTicketBatch(root Root, session Session, status, nextNeed, suggestedRole string) error {
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role != "cto" && role != "cto-weekly" {
		return nil
	}
	if !successfulDispositionStatus(status) {
		return nil
	}
	nextNeed = strings.ToLower(strings.TrimSpace(nextNeed))
	suggestedRole = strings.ToLower(strings.TrimSpace(suggestedRole))
	if nextNeed != "implementation" && suggestedRole != "engineer" {
		return nil
	}
	for _, featureID := range featureContractIDs(root) {
		scenarios, covered := featureScenarioCoverage(root, featureID)
		if len(scenarios) < 2 {
			continue
		}
		requiredScenarios := earlyCTOHandoffRequiredScenarios(root, featureID, scenarios)
		if len(requiredScenarios) == 0 {
			continue
		}
		required := len(requiredScenarios)
		coveredCount := countCoveredFeatureScenarios(requiredScenarios, covered)
		if coveredCount >= required {
			continue
		}
		next := firstUncoveredFeatureScenarios(requiredScenarios, covered, required-coveredCount)
		recordCTOHandoffRequiredScenarios(session, next)
		return fmt.Errorf("policy: cto cannot hand off implementation for %s after covering only %d/%d early product scenario(s). Create a small product backlog batch with ticket_create before Engineer handoff: cover the next product scenario(s) %s, or group adjacent bounded product scenarios in one ticket when that is the clearer slice", featureID, coveredCount, required, strings.Join(next, ", "))
	}
	return nil
}

func recordCTOHandoffRequiredScenarios(session Session, scenarios []string) {
	if len(scenarios) == 0 || session.ToolState == nil {
		return
	}
	session.ToolState[ctoHandoffRequiredScenariosKey] = strings.Join(scenarios, ",")
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

func earlyCTOHandoffRequiredScenarios(root Root, featureID string, scenarios []string) []string {
	productScenarios := productScenarioIDsForHandoff(root, featureID, scenarios)
	if len(productScenarios) > 0 {
		if len(productScenarios) > 3 {
			return productScenarios[:3]
		}
		return productScenarios
	}
	required := len(scenarios)
	if required > 3 {
		required = 3
	}
	return append([]string(nil), scenarios[:required]...)
}

func productScenarioIDsForHandoff(root Root, featureID string, scenarios []string) []string {
	featurePath := featureContractPath(root, featureID)
	if featurePath == "" {
		return nil
	}
	data, err := os.ReadFile(featurePath)
	if err != nil {
		return nil
	}
	sections := orderedFeatureScenarioSections(string(data))
	if len(sections) == 0 {
		return nil
	}
	byID := make(map[string]string, len(sections))
	for _, section := range sections {
		byID[section.ID] = section.Text
	}
	requiredCapabilities := projectBriefCapabilityPhrases(root)
	var out []string
	for _, scenario := range scenarios {
		text := byID[scenario]
		if text == "" {
			continue
		}
		if scenarioCoversProductCapability(text, requiredCapabilities) {
			out = append(out, scenario)
		}
	}
	return out
}

func scenarioCoversProductCapability(text string, requiredCapabilities []string) bool {
	if len(requiredCapabilities) > 0 {
		for _, phrase := range requiredCapabilities {
			if capabilityPhraseCovered(text, phrase) {
				return true
			}
		}
		return false
	}
	return scenarioLooksProductImplementation(text)
}

func scenarioLooksProductImplementation(text string) bool {
	surface := normalizeCapabilitySurface(text)
	if strings.Contains(surface, "harness telemetry") ||
		strings.Contains(surface, "intervention debt") ||
		strings.Contains(surface, "governance expansion") ||
		strings.Contains(surface, "wider automation") {
		return false
	}
	for _, marker := range []string{
		"visible",
		"runnable",
		"inspectable",
		"user can",
		"player",
		"game",
		"app",
		"browser",
		"build",
		"feature",
		"behavior",
		"behaviour",
	} {
		if strings.Contains(surface, marker) {
			return true
		}
	}
	return false
}

type featureScenarioSection struct {
	ID   string
	Text string
}

func orderedFeatureScenarioSections(content string) []featureScenarioSection {
	matches := toolsFeatureScenarioHeadingPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	var out []featureScenarioSection
	for i, match := range matches {
		id := strings.ToUpper(content[match[2]:match[3]])
		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		} else if outOfScope := strings.Index(strings.ToLower(content[start:]), "\n## out of scope"); outOfScope >= 0 {
			end = start + outOfScope
		}
		out = append(out, featureScenarioSection{
			ID:   id,
			Text: content[start:end],
		})
	}
	return out
}

func checkReviewChangesRequestedFeedbackOwnership(root Root, session Session, status, nextNeed string, raw json.RawMessage) error {
	if !reviewRoleRequiresValidationEvidence(session.Role) ||
		strings.ToLower(strings.TrimSpace(status)) != "changes_requested" ||
		strings.ToLower(strings.TrimSpace(nextNeed)) != "implementation_rework" {
		return nil
	}
	if !repoBrowserFrameworkInfo(root).UsesFramework {
		return nil
	}
	if findings := browserFrameworkSourceFindings(root); len(findings) > 0 {
		return nil
	}
	text := strings.ToLower(string(raw))
	buildSucceeded := session.ToolCounts != nil && session.ToolCounts[buildCommandSuccessKey] > 0 ||
		strings.Contains(text, "build succeeded") ||
		strings.Contains(text, "build passed")
	foundationValidationSignal := strings.Contains(text, "implementation is correct") ||
		strings.Contains(text, "smoke test validation error") ||
		strings.Contains(text, "test should") ||
		strings.Contains(text, "test needs") ||
		(buildSucceeded && (strings.Contains(text, "server not running") ||
			strings.Contains(text, "dev server") ||
			strings.Contains(text, "localhost:5173") ||
			strings.Contains(text, "curl test")))
	if !foundationValidationSignal {
		return nil
	}
	return fmt.Errorf("policy: %s changes_requested feedback appears to route a foundation validation/test wording issue to Engineer even though browser-framework source inspection is clean. Do not send implementation_rework when the implementation is described as correct; approve with corrected evidence, or record a foundation/dogfood finding for the validation helper instead", strings.ToLower(strings.TrimSpace(session.Role)))
}

func checkSuccessfulDispositionUnresolvedTicketCreation(root Root, session Session, status, nextNeed, suggestedRole string) error {
	if !successfulDispositionStatus(status) || session.ToolCounts == nil {
		return nil
	}
	if session.ToolCounts[ticketCreationOutstandingFailureKey] == 0 {
		return nil
	}
	if ctoImplementationHandoffTicketBatchSatisfied(root, session.Role, nextNeed, suggestedRole) {
		return nil
	}
	if planningRoleCanHandOffTicketCreation(session.Role, nextNeed, suggestedRole) {
		return nil
	}
	return fmt.Errorf("policy: job_disposition_record cannot record a successful disposition while ticket creation failed earlier in this job and no successful ticket_create followed. Retry ticket_create with valid JSON, including bdd_scenarios as an array like [\"F-001-S002\"], or record status blocked with the exact ticket_create error as the blocker")
}

func ctoImplementationHandoffTicketBatchSatisfied(root Root, role, nextNeed, suggestedRole string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "cto" && role != "cto-weekly" {
		return false
	}
	nextNeed = strings.ToLower(strings.TrimSpace(nextNeed))
	suggestedRole = strings.ToLower(strings.TrimSpace(suggestedRole))
	if nextNeed != "implementation" && suggestedRole != "engineer" {
		return false
	}
	checked := false
	for _, featureID := range featureContractIDs(root) {
		scenarios, covered := featureScenarioCoverage(root, featureID)
		if len(scenarios) < 2 {
			continue
		}
		requiredScenarios := earlyCTOHandoffRequiredScenarios(root, featureID, scenarios)
		if len(requiredScenarios) == 0 {
			continue
		}
		checked = true
		if countCoveredFeatureScenarios(requiredScenarios, covered) < len(requiredScenarios) {
			return false
		}
	}
	return checked
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
	if blockers := browserFrameworkCompletionBlockers(root, session, true); len(blockers) > 0 {
		return fmt.Errorf(
			"policy: %s cannot approve browser-framework ticket %s yet: %s. Record changes_requested with feedback.for_role engineer, next_need implementation_rework, and evidence_links naming the missing or failing validation",
			strings.ToLower(strings.TrimSpace(session.Role)),
			strings.ToUpper(strings.TrimSpace(ticketID)),
			strings.Join(blockers, "; "),
		)
	}
	if browserFrameworkRequiresProductSmoke(root) && counts[browserProductSmokeSuccessKey] == 0 {
		return fmt.Errorf(
			"policy: %s cannot approve browser-framework ticket %s from HTTP/build evidence alone. Run a browser product smoke or equivalent source/runtime assertion that checks the framework mounted real product UI state such as Phaser game/canvas behavior, then approve or request changes. Suggested smoke: %s",
			strings.ToLower(strings.TrimSpace(session.Role)),
			strings.ToUpper(strings.TrimSpace(ticketID)),
			browserProductSmokeCommandGuidance(root),
		)
	}
	return nil
}

func checkDogfoodDispositionValidationEvidence(root Root, session Session, status string) error {
	if strings.ToLower(strings.TrimSpace(session.Role)) != "dogfood" || !successfulReviewDispositionStatus(status) {
		return nil
	}
	if !browserFrameworkRequiresProductSmoke(root) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil {
		counts = map[string]int{}
	}
	if counts[browserProductSmokeSuccessKey] > 0 {
		return nil
	}
	return fmt.Errorf("policy: dogfood cannot approve browser-framework E2E from curl/HTTP reachability alone. Run a browser product smoke or equivalent source/runtime assertion that checks real product UI state such as Phaser game/canvas behavior, or create a target-owned finding and record changes_requested")
}

func checkPlanningDispositionFeatureSpecificity(root Root, session Session, status string) error {
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role != "coo" || !successfulDispositionStatus(status) {
		return nil
	}
	featurePath, err := root.ResolvePath(filepath.Join("docs", "features", "F-001-product-walking-skeleton.md"))
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(featurePath)
	if err != nil {
		return nil
	}
	content := strings.ToLower(string(data))
	if !projectBriefHasConcreteProductIntent(root) {
		return nil
	}
	if !featureContractSuperseded(string(data)) && (strings.Contains(content, "starter contract is seeded") ||
		strings.Contains(content, "replace placeholder nouns")) {
		return fmt.Errorf("policy: coo cannot complete planning while docs/features/F-001-product-walking-skeleton.md still contains starter-placeholder contract text. Rewrite the feature contract with product-specific business logic, scenario schedule, and current failing scenario from README/active goals before handing off to CTO")
	}
	if err := checkProductCapabilityScenarioCoverage(root); err != nil {
		return err
	}
	return nil
}

func checkProductCapabilityScenarioCoverage(root Root) error {
	required := projectBriefCapabilityPhrases(root)
	if len(required) == 0 {
		return nil
	}
	contents := productCapabilityCoverageFeatureContents(root)
	if len(contents) == 0 {
		return nil
	}
	var scenarioParts []string
	var outlineParts []string
	var descopedParts []string
	var outOfScopeParts []string
	for _, content := range contents {
		scenarioParts = append(scenarioParts, featureScenarioSurface(content))
		outlineParts = append(outlineParts, featureScenarioOutlineSurface(content))
		descopedParts = append(descopedParts, featureDescopedSurface(content))
		outOfScopeParts = append(outOfScopeParts, featureOutOfScopeSurface(content))
	}
	scenarioSurface := strings.Join(scenarioParts, "\n")
	outlineSurface := strings.Join(outlineParts, "\n")
	descopedSurface := strings.Join(descopedParts, "\n")
	outOfScopeSurface := strings.Join(outOfScopeParts, "\n")
	surface := scenarioSurface + "\n" + descopedSurface
	outlineAndDescopedSurface := outlineSurface + "\n" + descopedSurface
	var missing []string
	var outOfScopeRequired []string
	var outlineMissing []string
	for _, phrase := range required {
		if strings.TrimSpace(outOfScopeSurface) != "" &&
			outOfScopeSurfaceRequiresDescoping(outOfScopeSurface, phrase) &&
			!capabilityPhraseCovered(descopedSurface, phrase) {
			outOfScopeRequired = append(outOfScopeRequired, phrase)
			continue
		}
		if !capabilityPhraseCovered(surface, phrase) {
			missing = append(missing, phrase)
			continue
		}
		if !capabilityPhraseCovered(outlineAndDescopedSurface, phrase) {
			outlineMissing = append(outlineMissing, phrase)
		}
	}
	if len(outOfScopeRequired) > 0 {
		return fmt.Errorf(
			"policy: active feature contracts list explicit product brief capabilities under Out of Scope without Descoped Scenarios rationale: %s. Move required capabilities into the Scenario Schedule/scenario headings, or descoped scenarios with explicit reasons before creating tickets or handing off to CTO",
			strings.Join(outOfScopeRequired, ", "),
		)
	}
	if len(missing) == 0 {
		if len(outlineMissing) == 0 {
			return nil
		}
		return fmt.Errorf(
			"policy: active feature contract scenario outline does not break out product brief capabilities: %s. Rewrite the Scenario Schedule entries or scenario headings so distinct product capabilities are visible before CTO ticketing; do not hide them inside one broad runnable/inspectable scenario",
			strings.Join(outlineMissing, ", "),
		)
	}
	return fmt.Errorf(
		"policy: active feature contract scenario schedule does not cover product brief capabilities: %s. Rewrite the Scenario Schedule and scenario headings so every explicit brief capability is an in-scope scenario or is deliberately listed under Descoped Scenarios before creating tickets or handing off to CTO",
		strings.Join(missing, ", "),
	)
}

func productCapabilityCoverageFeatureContents(root Root) []string {
	featuresDir, err := root.ResolvePath(filepath.Join("docs", "features"))
	if err != nil {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(featuresDir, "F-*.md"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)
	var active []string
	var fallback []string
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.EqualFold(filepath.Base(match), "F-001-product-walking-skeleton.md") {
			fallback = append(fallback, content)
		}
		if featureContractSuperseded(content) {
			continue
		}
		active = append(active, content)
	}
	if len(active) > 0 {
		return active
	}
	return fallback
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
	if guidance := browserFrameworkTerminalDispositionGuidance(root, session); guidance != "" {
		return guidance
	}
	if repoHasGoSourceFiles(root) && !repoHasTestFiles(root) {
		return "Call job_disposition_record with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links explaining that Go source files exist but no _test.go files assert the completed behavior."
	}
	if repoHasTestFiles(root) && counts[testCommandSuccessKey] == 0 {
		return "Run the repository's authoritative test command such as go test ./... before approval, or record job_disposition_record with status changes_requested if tests cannot be run."
	}
	if roleRequiresDocSyncForSuccessfulDisposition(role) && counts["tool:docsync_audit:success"] == 0 {
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
	info := repoBrowserFrameworkInfo(root)
	if info.UsesFramework && info.HasBuildScript && counts[buildCommandSuccessKey] == 0 {
		return false
	}
	if info.UsesFramework && info.HasBuildScript && counts[buildCommandSuccessKey] > 0 && counts[browserProductSmokeSuccessKey] == 0 {
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
	if shellExecStopsTrackedBackgroundPID(args) {
		return nil
	}
	if shellExecReadOnly(raw) {
		return nil
	}
	if shellExecRunsValidationCommand(args) || shellExecRunsRecordedValidationArtifact(session, root, args) || shellExecRunsHTTPProbe(args) {
		return nil
	}
	return fmt.Errorf("policy: %s shell_exec is validation-only; do not run mutating setup, package initialization, product edits, cleanup, or broad discovery such as %q. Use read-only tools, tests/builds/runtime probes, or record job_disposition_record with a blocker", role, shellExecCommandDisplay(args))
}

func shellExecStopsTrackedBackgroundPID(args shellExecArgs) bool {
	if len(args.Argv) < 2 || strings.TrimSpace(args.ShellCommand) != "" || filepathBase(args.Argv[0]) != "kill" {
		return false
	}
	active := map[int]bool{}
	for _, pid := range trackedBackgroundPIDs() {
		active[pid] = true
	}
	if len(active) == 0 {
		return false
	}
	sawPID := false
	for _, arg := range args.Argv[1:] {
		arg = strings.TrimSpace(strings.Trim(arg, `"'`))
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		pid, err := strconv.Atoi(arg)
		if err != nil || !active[pid] {
			return false
		}
		sawPID = true
	}
	return sawPID
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
	return shellExecRunsTestCommand(args) || shellExecRunsBuildCommand(args) || shellExecRunsRuntimeValidationCommand(args) || shellExecRunsHTTPProbe(args)
}

func shellExecRunsValidationCommandForSession(session *Session, root Root, args shellExecArgs) bool {
	if shellExecCountsAsValidationEvidence(args) {
		return true
	}
	if session == nil {
		return false
	}
	return shellExecRunsRecordedValidationArtifact(*session, root, args)
}

func shellExecCountsAsValidationEvidence(args shellExecArgs) bool {
	if shellExecRunsTestCommand(args) || shellExecRunsBuildCommand(args) || shellExecRunsHTTPProbe(args) {
		return true
	}
	if args.Background {
		return false
	}
	return shellExecRunsRuntimeValidationCommand(args)
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

func checkShellNodeCheckHTMLPolicy(args shellExecArgs) error {
	if !shellExecNodeCheckHTML(args) {
		return nil
	}
	return fmt.Errorf("policy: node --check only validates JavaScript source, not HTML files. Do not run node --check on .html/.htm entries; validate browser targets with a real package build such as npm run build and a browser/product smoke that loads the HTML")
}

func shellExecNodeCheckHTML(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	for i := 0; i < len(fields)-2; i++ {
		if filepathBase(fields[i]) != "node" {
			continue
		}
		flag := fields[i+1]
		if flag != "--check" && flag != "-c" {
			continue
		}
		target := strings.ToLower(cleanShellPathToken(fields[i+2]))
		if strings.HasSuffix(target, ".html") || strings.HasSuffix(target, ".htm") {
			return true
		}
	}
	return false
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

func shellExecRunsBrowserProductSmokeCommand(args shellExecArgs) bool {
	display := strings.ToLower(shellExecCommandDisplay(args))
	if display == "" {
		return false
	}
	if strings.Contains(display, "playwright") ||
		strings.Contains(display, "puppeteer") ||
		strings.Contains(display, "document.queryselector") ||
		strings.Contains(display, "getelementbyid") ||
		strings.Contains(display, "queryselector") {
		return strings.Contains(display, "canvas") ||
			strings.Contains(display, "#game") ||
			strings.Contains(display, "phaser") ||
			strings.Contains(display, "score") ||
			strings.Contains(display, "game over")
	}
	if strings.Contains(display, "phaser") &&
		(strings.Contains(display, "canvas") ||
			strings.Contains(display, "new phaser.game") ||
			strings.Contains(display, "scene") ||
			strings.Contains(display, "sprite") ||
			strings.Contains(display, "game object")) {
		return true
	}
	return false
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

func packageBuildScriptNoop(script string) bool {
	script = strings.TrimSpace(strings.ToLower(script))
	if script == "" {
		return true
	}
	if packageBuildScriptOnlySyntaxCheck(script) {
		return true
	}
	for _, marker := range []string{
		"vite build", "next build", "webpack", "rollup", "parcel", "astro build",
		"tsc", "esbuild", "npm run", "pnpm run", "yarn ", "bun run",
		"node ", "deno ", "make ",
	} {
		if strings.Contains(script, marker) {
			return false
		}
	}
	parts := strings.Split(script, "&&")
	if len(parts) == 0 {
		return true
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == ":" || part == "true" || part == "exit 0" ||
			strings.HasPrefix(part, "echo ") ||
			strings.HasPrefix(part, "printf ") ||
			strings.HasPrefix(part, "mkdir ") ||
			strings.HasPrefix(part, "cp ") ||
			strings.HasPrefix(part, "copy ") ||
			strings.HasPrefix(part, "rsync ") ||
			strings.HasPrefix(part, "touch ") ||
			strings.HasPrefix(part, "live-server") ||
			strings.HasPrefix(part, "http-server") ||
			strings.HasPrefix(part, "serve ") ||
			strings.Contains(part, "python -m http.server") ||
			strings.Contains(part, "python3 -m http.server") {
			continue
		}
		return false
	}
	return true
}

func packageBuildScriptOnlySyntaxCheck(script string) bool {
	parts := strings.Split(script, "&&")
	checked := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "node --check ") || strings.HasPrefix(part, "node -c ") {
			checked = true
			continue
		}
		return false
	}
	return checked
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

type browserFrameworkInfo struct {
	UsesFramework          bool
	FrameworkNames         []string
	DeclaredFrameworkNames []string
	HasPackageManifest     bool
	HasBuildScript         bool
	NoopBuildScripts       []string
}

func repoBrowserFrameworkInfo(root Root) browserFrameworkInfo {
	var info browserFrameworkInfo
	seen := map[string]bool{}
	addFramework := func(name string, declared bool) {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			return
		}
		if !seen[name] {
			seen[name] = true
			info.FrameworkNames = append(info.FrameworkNames, name)
		}
		if declared && !frameworkListContains(info.DeclaredFrameworkNames, name) {
			info.DeclaredFrameworkNames = append(info.DeclaredFrameworkNames, name)
		}
	}
	path, err := root.ResolvePath("package.json")
	if err == nil {
		if data, err := os.ReadFile(path); err == nil {
			var pkg struct {
				Scripts         map[string]string `json:"scripts"`
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if err := json.Unmarshal(data, &pkg); err == nil {
				info.HasPackageManifest = true
				for name, script := range pkg.Scripts {
					if buildScriptName(name) {
						if packageBuildScriptNoop(script) {
							info.NoopBuildScripts = append(info.NoopBuildScripts, name)
							continue
						}
						info.HasBuildScript = true
					}
				}
				frameworks := map[string]string{
					"@vitejs/plugin-react": "react",
					"@vitejs/plugin-vue":   "vue",
					"babylonjs":            "babylon",
					"next":                 "next",
					"phaser":               "phaser",
					"pixi.js":              "pixi",
					"react":                "react",
					"svelte":               "svelte",
					"three":                "three",
					"vite":                 "vite",
					"vue":                  "vue",
				}
				for dep := range pkg.Dependencies {
					if name, ok := frameworks[strings.ToLower(strings.TrimSpace(dep))]; ok {
						addFramework(name, true)
					}
				}
				for dep := range pkg.DevDependencies {
					if name, ok := frameworks[strings.ToLower(strings.TrimSpace(dep))]; ok {
						addFramework(name, true)
					}
				}
			}
		}
	}
	if projectBriefMentionsFramework(root, "phaser") || repoHasPhaserScriptTag(root) {
		addFramework("phaser", false)
	}
	info.UsesFramework = len(info.FrameworkNames) > 0
	return info
}

func browserFrameworkCompletionBlockers(root Root, session Session, requireBuildRun bool) []string {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return nil
	}
	var blockers []string
	frameworks := strings.Join(info.FrameworkNames, ", ")
	if frameworks == "" {
		frameworks = "browser framework"
	}
	if frameworkListContains(info.FrameworkNames, "phaser") && !frameworkListContains(info.DeclaredFrameworkNames, "phaser") {
		if info.HasPackageManifest {
			blockers = append(blockers, "project references Phaser but package.json does not declare a local phaser dependency; add phaser to package.json instead of relying on CDN-only runtime")
		} else {
			blockers = append(blockers, "project references Phaser but has no package.json; create a JavaScript package manifest with a local phaser dependency and deterministic npm run build")
		}
	}
	if !info.HasPackageManifest {
		blockers = append(blockers, fmt.Sprintf("project references %s but no package.json is present; add package.json with a deterministic build/static validation command such as npm run build", frameworks))
	} else if len(info.NoopBuildScripts) > 0 && !info.HasBuildScript {
		blockers = append(blockers, fmt.Sprintf("package.json build script for %s is a no-op (%s); replace it with a deterministic build/static validation command such as vite build, tsc --noEmit, or another command that can fail when the browser app is broken", frameworks, strings.Join(info.NoopBuildScripts, ", ")))
	} else if !info.HasBuildScript {
		blockers = append(blockers, fmt.Sprintf("package.json declares %s but no build script; add a deterministic build/static validation command such as npm run build", frameworks))
	} else if requireBuildRun {
		counts := session.ToolCounts
		if counts == nil || counts[buildCommandSuccessKey] == 0 {
			blockers = append(blockers, fmt.Sprintf("package.json declares %s but npm run build or equivalent has not passed in this job", frameworks))
		}
	}
	blockers = append(blockers, browserFrameworkSourceFindings(root)...)
	return blockers
}

func engineerBrowserFrameworkCompletionBlockers(root Root, session Session) []string {
	blockers := browserFrameworkCompletionBlockers(root, session, true)
	if !browserFrameworkRequiresProductSmoke(root) {
		return blockers
	}
	counts := session.ToolCounts
	if counts == nil || counts[browserProductSmokeSuccessKey] == 0 {
		blockers = append(blockers, "browser-framework product smoke has not passed in this job; run "+browserProductSmokeCommandGuidance(root)+". node --check, grep-only evidence, and repo-root scratch scripts are insufficient")
	}
	return blockers
}

func browserProductSmokeCommandGuidance(root Root) string {
	info := repoBrowserFrameworkInfo(root)
	if frameworkListContains(info.FrameworkNames, "phaser") {
		return `shell_exec argv ["node","-e","const fs=require('fs'); const htmlPath=['src/index.html','index.html'].find(p=>fs.existsSync(p)); if(!htmlPath) throw new Error('missing index.html'); const html=fs.readFileSync(htmlPath,'utf8'); const lower=html.toLowerCase(); if(lower.includes('phaser')&&(lower.includes('cdn')||lower.includes('http'))) throw new Error('CDN Phaser script tag is not bundled'); if(!html.includes('main.js')) throw new Error('missing main.js module script'); const mainPath=fs.existsSync('src/main.js')?'src/main.js':'main.js'; const main=fs.readFileSync(mainPath,'utf8'); if(!main.includes(\"import Phaser from 'phaser'\")&&!main.includes('import Phaser from \"phaser\"')) throw new Error('missing import Phaser from phaser'); const games=main.split('new Phaser.Game').length-1; if(games!==1) throw new Error('expected exactly one new Phaser.Game'); if(!main.includes('parent')) throw new Error('missing parent game container'); console.log('browser smoke: Phaser canvas #game new Phaser.Game');"]`
	}
	return "Playwright/Puppeteer or an equivalent source/runtime assertion that proves the browser app mounts real UI state"
}

func browserFrameworkTerminalDispositionGuidance(root Root, session Session) string {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return ""
	}
	counts := session.ToolCounts
	if counts == nil {
		counts = map[string]int{}
	}
	sourceFindings := browserFrameworkSourceFindings(root)
	if len(sourceFindings) > 0 || !info.HasBuildScript {
		blockers := browserFrameworkCompletionBlockers(root, session, false)
		return "Call job_disposition_record with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links explaining that browser-framework completion is not proven: " + strings.Join(blockers, "; ") + "."
	}
	if counts[buildCommandSuccessKey] == 0 {
		return "Run the browser-framework build command such as npm run build before approval, or record job_disposition_record with status changes_requested if the project has no runnable build validation."
	}
	if browserFrameworkRequiresProductSmoke(root) && counts[browserProductSmokeSuccessKey] == 0 {
		return "Run a browser product smoke or equivalent source/runtime assertion that checks real product UI state such as Phaser game/canvas behavior before approval, such as " + browserProductSmokeCommandGuidance(root) + ", or record job_disposition_record with status changes_requested."
	}
	return ""
}

func browserFrameworkRequiresProductSmoke(root Root) bool {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return false
	}
	return true
}

func browserFrameworkSourceFindings(root Root) []string {
	info := repoBrowserFrameworkInfo(root)
	if !frameworkListContains(info.FrameworkNames, "phaser") {
		return nil
	}
	var findings []string
	findings = append(findings, phaserGoModuleFindings(root)...)
	jsModules := map[string]string{}
	jsModulePaths := []string{}
	htmlFiles := map[string]string{}
	htmlPaths := []string{}
	_ = filepath.WalkDir(root.Abs(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
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
		rel = filepath.ToSlash(rel)
		lowerRel := strings.ToLower(rel)
		if !javascriptSourcePath(lowerRel) && !htmlSourcePath(lowerRel) {
			return nil
		}
		if browserFrameworkValidationHelperPath(lowerRel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		if (lowerRel == "vite.config.js" || lowerRel == "vite.config.ts") && viteConfigExternalizesPhaser(content) {
			findings = append(findings, fmt.Sprintf("%s externalizes phaser from the Vite bundle; remove rollupOptions.external for phaser so the browser build proves the local dependency loads", rel))
		}
		if htmlSourcePath(lowerRel) {
			findings = append(findings, phaserHTMLFindings(rel, content)...)
			htmlFiles[rel] = content
			htmlPaths = append(htmlPaths, rel)
			return nil
		}
		jsModules[rel] = content
		jsModulePaths = append(jsModulePaths, rel)
		for _, id := range []string{"preload", "create", "update"} {
			if phaserSceneReferencesIdentifier(content, id) && !jsDefinesOrImportsIdentifier(content, id) {
				findings = append(findings, fmt.Sprintf("%s references Phaser scene callback %q without defining or importing it in the same module", rel, id))
			}
		}
		findings = append(findings, phaserSingleFileSourceFindings(rel, content)...)
		return nil
	})
	for _, rel := range jsModulePaths {
		findings = append(findings, jsLocalNamedImportFindings(rel, jsModules[rel], jsModules)...)
		findings = append(findings, jsMissingLocalExportImportFindings(rel, jsModules[rel], jsModules)...)
	}
	for _, rel := range htmlPaths {
		findings = append(findings, htmlClassicScriptModuleFindings(rel, htmlFiles[rel], jsModules)...)
	}
	return findings
}

func javascriptSourcePath(lowerRel string) bool {
	return strings.HasSuffix(lowerRel, ".js") ||
		strings.HasSuffix(lowerRel, ".mjs") ||
		strings.HasSuffix(lowerRel, ".jsx") ||
		strings.HasSuffix(lowerRel, ".ts") ||
		strings.HasSuffix(lowerRel, ".tsx")
}

func browserFrameworkValidationHelperPath(lowerRel string) bool {
	lowerRel = filepath.ToSlash(strings.ToLower(strings.TrimSpace(lowerRel)))
	base := filepath.Base(lowerRel)
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	if strings.HasPrefix(lowerRel, "test/") || strings.HasPrefix(lowerRel, "tests/") {
		return true
	}
	if strings.HasPrefix(lowerRel, "scripts/") &&
		(strings.Contains(base, "validate") || strings.Contains(base, "smoke") || strings.Contains(base, "probe")) {
		return true
	}
	if strings.Contains(lowerRel, "/") {
		return false
	}
	return strings.HasPrefix(base, "validate-") ||
		strings.HasSuffix(base, "-validation.js") ||
		strings.Contains(base, "smoke") ||
		strings.Contains(base, "probe")
}

func htmlSourcePath(lowerRel string) bool {
	return strings.HasSuffix(lowerRel, ".html") || strings.HasSuffix(lowerRel, ".htm")
}

func projectBriefMentionsFramework(root Root, framework string) bool {
	framework = strings.ToLower(strings.TrimSpace(framework))
	if framework == "" {
		return false
	}
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
		filepath.Join("docs", "features", "F-001-product-walking-skeleton.md"),
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), framework) {
			return true
		}
	}
	return false
}

func projectBriefHasConcreteProductIntent(root Root) bool {
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(data))
		for _, marker := range []string{"create ", "build ", "implement ", "game", "app", "application", "service", "tool", "website", "dashboard"} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func projectBriefCapabilityPhrases(root Root) []string {
	text := projectBriefSourceText(root)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	labelTokens := projectBriefLabelTokens(root)
	markers := []string{
		" should include ",
		" must include ",
		" include ",
		" includes ",
		" including ",
		" should implement ",
		" must implement ",
		" implement ",
		" implements ",
		" should support ",
		" must support ",
		" support ",
		" supports ",
		" should detect ",
		" must detect ",
		" should allow ",
		" must allow ",
		" should let ",
		" must let ",
		" features:",
		" features include ",
	}
	seen := map[string]bool{}
	var phrases []string
	for _, sentence := range splitBriefSentences(text) {
		lower := " " + strings.ToLower(sentence) + " "
		for _, marker := range markers {
			idx := strings.Index(lower, marker)
			if idx < 0 {
				continue
			}
			segment := strings.TrimSpace(lower[idx+len(marker):])
			for _, phrase := range splitCapabilitySegment(segment) {
				if isValidationEvidenceCapabilityPhrase(phrase) {
					continue
				}
				phrase = stripCapabilityLabelTokens(phrase, labelTokens)
				if len(capabilityKeywords(phrase)) == 0 {
					continue
				}
				key := strings.ToLower(phrase)
				if seen[key] {
					continue
				}
				seen[key] = true
				phrases = append(phrases, phrase)
			}
		}
	}
	return phrases
}

func projectBriefLabelTokens(root Root) map[string]bool {
	out := map[string]bool{}
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#") {
				continue
			}
			trimmed = strings.TrimLeft(trimmed, "#")
			trimmed = strings.TrimSpace(trimmed)
			if trimmed == "" {
				continue
			}
			for _, field := range strings.Fields(normalizeCapabilitySurface(trimmed)) {
				key := capabilityKeyword(field)
				if key == "" || capabilityLabelKeepWords[key] {
					continue
				}
				out[key] = true
			}
		}
	}
	return out
}

func stripCapabilityLabelTokens(phrase string, labels map[string]bool) string {
	if len(labels) == 0 {
		return phrase
	}
	fields := strings.Fields(normalizeCapabilitySurface(phrase))
	if len(fields) == 0 {
		return phrase
	}
	var kept []string
	removed := false
	for _, field := range fields {
		key := capabilityKeyword(field)
		if key != "" && labels[key] {
			removed = true
			continue
		}
		kept = append(kept, field)
	}
	candidate := cleanCapabilityPhrase(strings.Join(kept, " "))
	if candidate == "" || len(capabilityKeywords(candidate)) == 0 {
		if removed {
			return ""
		}
		return phrase
	}
	return candidate
}

func projectBriefSourceText(root Root) string {
	var b strings.Builder
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		b.WriteByte('\n')
		b.Write(data)
	}
	return b.String()
}

func splitBriefSentences(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	numberedListPattern := regexp.MustCompile(`^\d+\.\s+`)
	var normalized strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			normalized.WriteString(". ")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "#") ||
			numberedListPattern.MatchString(trimmed) {
			normalized.WriteString(". ")
		}
		normalized.WriteString(trimmed)
		normalized.WriteByte(' ')
	}
	fields := regexp.MustCompile(`[.!?]+`).Split(normalized.String(), -1)
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func splitCapabilitySegment(segment string) []string {
	segment = strings.TrimSpace(segment)
	segment = strings.Trim(segment, " .:-")
	segment = stripCapabilityCategoryPrefix(segment)
	if segment == "" {
		return nil
	}
	segment = regexp.MustCompile(`\b(and|plus)\b`).ReplaceAllString(segment, ",")
	rawParts := strings.FieldsFunc(segment, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	var phrases []string
	skipValidationTail := false
	for _, part := range rawParts {
		phrase := cleanCapabilityPhrase(part)
		if phrase == "" {
			continue
		}
		if isValidationEvidenceCapabilityPhrase(phrase) {
			skipValidationTail = true
			continue
		}
		if skipValidationTail && isValidationEvidenceTailPhrase(phrase) {
			continue
		}
		skipValidationTail = false
		if len(capabilityKeywords(phrase)) == 0 {
			continue
		}
		phrases = append(phrases, phrase)
	}
	return phrases
}

func stripCapabilityCategoryPrefix(segment string) string {
	idx := strings.Index(segment, ":")
	if idx < 0 {
		return segment
	}
	prefix := strings.ToLower(strings.TrimSpace(segment[:idx]))
	if strings.Contains(prefix, "mechanic") ||
		strings.Contains(prefix, "capabilit") ||
		strings.Contains(prefix, "feature") ||
		strings.Contains(prefix, "behavior") ||
		strings.Contains(prefix, "behaviour") {
		return strings.TrimSpace(segment[idx+1:])
	}
	return segment
}

func cleanCapabilityPhrase(phrase string) string {
	phrase = strings.TrimSpace(strings.ToLower(phrase))
	phrase = strings.Trim(phrase, "`*_ .:-")
	for _, prefix := range []string{"a ", "an ", "the "} {
		phrase = strings.TrimPrefix(phrase, prefix)
	}
	for _, suffix := range []string{" behavior", " behaviour", " feature", " features", " functionality", " flow", " capability", " capabilities"} {
		phrase = strings.TrimSuffix(phrase, suffix)
	}
	phrase = strings.Join(strings.Fields(phrase), " ")
	if len(phrase) < 3 {
		return ""
	}
	return phrase
}

func isValidationEvidenceCapabilityPhrase(phrase string) bool {
	normalized := normalizeCapabilitySurface(phrase)
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"evidence",
		"smoke evidence",
		"smoke test",
		"validation evidence",
		"test evidence",
		"prove",
		"proves",
		"proving",
		"verified by",
		"build artifact",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isValidationEvidenceTailPhrase(phrase string) bool {
	switch normalizeCapabilitySurface(phrase) {
	case "mount", "mounts", "play", "plays", "load", "loads", "run", "runs":
		return true
	default:
		return false
	}
}

func featureScenarioSurface(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "## scenario schedule")
	if start < 0 {
		start = strings.Index(lower, "## scenarios")
	}
	if start < 0 {
		return ""
	}
	end := strings.Index(lower[start:], "\n## out of scope")
	if end >= 0 {
		return lower[start : start+end]
	}
	return lower[start:]
}

func featureScenarioOutlineSurface(content string) string {
	lower := strings.ToLower(content)
	var parts []string
	start := strings.Index(lower, "## scenario schedule")
	if start >= 0 {
		end := strings.Index(lower[start+1:], "\n## ")
		if end >= 0 {
			parts = append(parts, lower[start:start+1+end])
		} else {
			parts = append(parts, lower[start:])
		}
	}
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "### f-") {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

func featureOutOfScopeSurface(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "## out of scope")
	if start < 0 {
		return ""
	}
	end := strings.Index(lower[start+1:], "\n## ")
	if end >= 0 {
		return lower[start : start+1+end]
	}
	return lower[start:]
}

func outOfScopeSurfaceRequiresDescoping(surface, phrase string) bool {
	for _, line := range strings.Split(surface, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "##") {
			continue
		}
		if outOfScopeLineIsExplanation(line) {
			continue
		}
		if outOfScopeLineLeavesBasicCapabilityInScope(line) {
			continue
		}
		if capabilityPhraseCovered(line, phrase) {
			return true
		}
	}
	return false
}

func outOfScopeLineIsExplanation(line string) bool {
	normalized := normalizeCapabilitySurface(line)
	if strings.HasPrefix(normalized, "the following") ||
		strings.HasPrefix(normalized, "following ") ||
		strings.HasPrefix(normalized, "none ") {
		return true
	}
	for _, marker := range []string{
		"clear reason",
		"clear reasons",
		"explicit rationale",
		"explicit rationales",
		"listed under out of scope",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func outOfScopeLineLeavesBasicCapabilityInScope(line string) bool {
	normalized := normalizeCapabilitySurface(line)
	switch normalized {
	case "animation", "animations", "animation polish", "animation only polish", "animation-only polish",
		"visual polish", "visual effects",
		"preview", "previews", "next piece preview", "next piece previews", "piece preview", "piece previews",
		"sound", "sounds", "sound effects", "audio", "audio feedback",
		"multiplayer", "multiplayer support", "multiplayer functionality",
		"mobile touch controls", "touch controls", "touch input",
		"hold piece", "hold queue", "hard drop":
		return true
	}
	for _, prefix := range []string{
		"animation for ",
		"animations for ",
		"animated ",
		"animation polish for ",
		"animation-only polish for ",
		"visual polish for ",
		"visual effects for ",
		"preview for ",
		"previews for ",
		"next piece preview for ",
		"sound for ",
		"sounds for ",
		"audio for ",
		"multiplayer for ",
		"mobile touch controls for ",
		"touch controls for ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	if strings.Contains(normalized, "advanced") && strings.Contains(normalized, "beyond basic") {
		return true
	}
	if strings.Contains(normalized, "advanced") &&
		(strings.Contains(normalized, "scoring") ||
			strings.Contains(normalized, "score") ||
			strings.Contains(normalized, "combo") ||
			strings.Contains(normalized, "back to back") ||
			strings.Contains(normalized, "changes basic")) {
		return true
	}
	if strings.Contains(normalized, "combo") || strings.Contains(normalized, "back to back") {
		return true
	}
	if strings.Contains(normalized, "beyond") {
		return true
	}
	if strings.Contains(normalized, "high score") || strings.Contains(normalized, "persistence") || strings.Contains(normalized, "persisted") {
		return true
	}
	return false
}

func featureDescopedSurface(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "## descoped scenarios")
	if start < 0 {
		return ""
	}
	end := strings.Index(lower[start+1:], "\n## ")
	if end >= 0 {
		return lower[start : start+1+end]
	}
	return lower[start:]
}

func capabilityPhraseCovered(surface, phrase string) bool {
	surface = normalizeCapabilitySurface(surface)
	phrase = normalizeCapabilitySurface(phrase)
	if phrase == "" {
		return true
	}
	if strings.Contains(surface, phrase) {
		return true
	}
	surfaceKeys := capabilityKeywordSet(surface)
	keys := capabilityKeywords(phrase)
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if key == "move" && (surfaceKeys["left"] || surfaceKeys["right"] || surfaceKeys["down"] || (surfaceKeys["control"] && surfaceKeys["keyboard"])) {
			continue
		}
		if !surfaceKeys[key] {
			return false
		}
	}
	return true
}

func normalizeCapabilitySurface(text string) string {
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func capabilityKeywordSet(text string) map[string]bool {
	out := map[string]bool{}
	for _, field := range strings.Fields(text) {
		if key := capabilityKeyword(field); key != "" {
			out[key] = true
		}
	}
	return out
}

func capabilityKeywords(phrase string) []string {
	seen := map[string]bool{}
	var out []string
	for _, field := range strings.Fields(normalizeCapabilitySurface(phrase)) {
		key := capabilityKeyword(field)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func capabilityKeyword(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	if token == "" || capabilityStopWords[token] {
		return ""
	}
	switch {
	case strings.HasPrefix(token, "rotat"):
		return "rotat"
	case strings.HasPrefix(token, "scor"):
		return "score"
	case strings.HasPrefix(token, "track"):
		return "score"
	case strings.HasPrefix(token, "clear"):
		return "clear"
	case strings.HasPrefix(token, "mov"):
		return "move"
	case token == "over" || strings.HasPrefix(token, "end"):
		return "gameover"
	case strings.HasPrefix(token, "restart"):
		return "restart"
	case strings.HasPrefix(token, "playfield"):
		return "playfield"
	case strings.HasPrefix(token, "keyboard"):
		return "keyboard"
	case strings.HasPrefix(token, "browser"):
		return "browser"
	case strings.HasPrefix(token, "canvas"):
		return "canvas"
	case strings.HasPrefix(token, "collision"):
		return "collision"
	case strings.HasPrefix(token, "lock"):
		return "lock"
	}
	for _, suffix := range []string{"ing", "ed", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			token = strings.TrimSuffix(token, suffix)
			break
		}
	}
	if len(token) < 4 || capabilityStopWords[token] {
		return ""
	}
	return token
}

var capabilityStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "another": true, "are": true, "as": true, "be": true, "by": true,
	"can": true, "complete": true, "condition": true, "conditions": true, "described": true,
	"core": true, "detect": true, "detected": true, "detection": true, "display": true, "displayed": true, "displays": true,
	"fall": true, "falling": true, "fill": true, "fills": true, "filled": true, "for": true, "from": true, "full": true,
	"gameplay": true, "handle": true, "handled": true, "game": true, "games": true, "in": true,
	"include": true, "includes": true, "including": true, "inspect": true, "inspected": true, "into": true, "local": true, "locally": true, "of": true,
	"mechanic": true, "mechanics": true, "on": true, "open": true, "opened": true, "or": true, "product": true, "project": true,
	"piece": true, "pieces": true, "playable": true, "player": true, "players": true, "reach": true, "reaches": true, "round": true, "rounds": true, "run": true, "see": true, "stack": true,
	"show": true, "showing": true, "shows": true, "that": true, "the": true, "to": true, "using": true, "user": true, "users": true,
	"usable": true, "useful": true, "version": true, "when": true, "with": true,
}

var capabilityLabelKeepWords = map[string]bool{
	"application": true,
	"board":       true,
	"calendar":    true,
	"chat":        true,
	"dashboard":   true,
	"editor":      true,
	"form":        true,
	"service":     true,
	"site":        true,
	"task":        true,
	"tracker":     true,
	"workflow":    true,
}

func projectBriefNamesGoBackend(root Root) bool {
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(data))
		for _, marker := range []string{"go backend", "golang backend", "go server", "golang server", "go cli", "golang cli"} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func repoHasPhaserScriptTag(root Root) bool {
	found := false
	_ = filepath.WalkDir(root.Abs(), func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "<script") && strings.Contains(lower, "phaser") {
			found = true
		}
		return nil
	})
	return found
}

func phaserGoModuleFindings(root Root) []string {
	var findings []string
	_ = filepath.WalkDir(root.Abs(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if strings.ToLower(d.Name()) != "go.mod" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !strings.Contains(strings.ToLower(string(data)), "phaser") {
			return nil
		}
		rel, err := filepath.Rel(root.Abs(), path)
		if err != nil {
			return nil
		}
		findings = append(findings, fmt.Sprintf("%s declares a Phaser-related Go module dependency; Phaser JS targets should use package.json with the phaser npm dependency, not go.mod", filepath.ToSlash(rel)))
		return nil
	})
	return findings
}

func phaserHTMLFindings(rel, content string) []string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "<script") && strings.Contains(lower, "phaser") &&
		(strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "cdn.")) {
		return []string{fmt.Sprintf("%s loads Phaser from a CDN/script tag; Phaser JS targets must use the local phaser npm dependency through the module/bundler entrypoint", rel)}
	}
	return nil
}

func phaserSingleFileSourceFindings(rel, content string) []string {
	var findings []string
	findings = append(findings, phaserMissingImportFindings(rel, content)...)
	findings = append(findings, phaserUnboundSceneHelperFindings(rel, content)...)
	findings = append(findings, phaserSceneContextFindings(rel, content)...)
	findings = append(findings, phaserGameConstructionFindings(rel, content)...)
	return findings
}

func frameworkListContains(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func phaserSceneReferencesIdentifier(content, id string) bool {
	pattern := regexp.MustCompile(`(?m)\b` + regexp.QuoteMeta(id) + `\s*:\s*` + regexp.QuoteMeta(id) + `\b`)
	return pattern.MatchString(content)
}

func jsDefinesOrImportsIdentifier(content, id string) bool {
	quoted := regexp.QuoteMeta(id)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)\bfunction\s+` + quoted + `\s*\(`),
		regexp.MustCompile(`(?m)\b(?:const|let|var)\s+` + quoted + `\b`),
		regexp.MustCompile(`(?m)\bimport\b[^\n;]*\b` + quoted + `\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func phaserUnboundSceneHelperFindings(rel, content string) []string {
	var findings []string
	helperRe := regexp.MustCompile(`(?s)function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\([^)]*\)\s*\{[^}]*\bthis\.add\.`)
	for _, match := range helperRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		name := match[1]
		bareCall := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\(\s*\)`)
		callCount := len(bareCall.FindAllStringIndex(content, -1))
		if callCount > 1 {
			findings = append(findings, fmt.Sprintf("%s defines Phaser helper %s using this.add but calls it without binding or passing the scene", rel, name))
		}
	}
	return findings
}

func phaserMissingImportFindings(rel, content string) []string {
	if !strings.Contains(content, "Phaser.") && !regexp.MustCompile(`\bextends\s+Phaser\.`).MatchString(content) {
		return nil
	}
	if jsDefinesOrImportsIdentifier(content, "Phaser") {
		return nil
	}
	return []string{fmt.Sprintf("%s uses Phaser global APIs without importing or defining Phaser in the same module; import Phaser from 'phaser' or avoid referencing the global", rel)}
}

func phaserSceneContextFindings(rel, content string) []string {
	var findings []string
	sceneAPIRe := regexp.MustCompile(`\bthis\.(?:add|cameras|input|load|make|physics|time|tweens)\b`)
	if strings.Contains(content, "new Phaser.Game") && strings.Contains(content, ".bind(this)") && sceneAPIRe.MatchString(content) {
		findings = append(findings, fmt.Sprintf("%s binds Phaser scene callbacks to wrapper this while using Phaser scene APIs; Phaser scene lifecycle methods must run with scene context or receive the scene explicitly", rel))
	}
	gameInstanceSceneAPIRe := regexp.MustCompile(`\bthis\.(?:game|gameInstance)\.(?:add|input|load|make|physics)\b`)
	if gameInstanceSceneAPIRe.MatchString(content) {
		findings = append(findings, fmt.Sprintf("%s uses the Phaser game instance as a scene API surface; drawing, input, and factory calls must run from the Phaser scene context", rel))
	}
	return findings
}

func phaserGameConstructionFindings(rel, content string) []string {
	if !strings.Contains(content, "new Phaser.Game") {
		return nil
	}
	var findings []string
	count := strings.Count(content, "new Phaser.Game")
	if count > 1 {
		findings = append(findings, fmt.Sprintf("%s constructs Phaser.Game %d times; create exactly one game instance from the browser entrypoint", rel, count))
	}
	for _, callback := range []string{"preload", "create", "update"} {
		if phaserNewGameInsideFunction(content, callback) {
			findings = append(findings, fmt.Sprintf("%s constructs new Phaser.Game inside scene callback %s; create the game once at module startup and let scene callbacks use the scene instance", rel, callback))
		}
	}
	return findings
}

func phaserNewGameInsideFunction(content, name string) bool {
	re := regexp.MustCompile(`\bfunction\s+` + regexp.QuoteMeta(name) + `\s*\([^)]*\)\s*\{`)
	for _, loc := range re.FindAllStringIndex(content, -1) {
		open := strings.LastIndex(content[loc[0]:loc[1]], "{")
		if open < 0 {
			continue
		}
		open += loc[0]
		close := jsMatchingBrace(content, open)
		if close < 0 {
			continue
		}
		body := content[open+1 : close]
		if regexp.MustCompile(`\bnew\s+Phaser\.Game\b`).MatchString(body) {
			return true
		}
	}
	return false
}

func jsMatchingBrace(content string, open int) int {
	if open < 0 || open >= len(content) || content[open] != '{' {
		return -1
	}
	depth := 0
	inSingle := false
	inDouble := false
	inTemplate := false
	lineComment := false
	blockComment := false
	escaped := false
	for i := open; i < len(content); i++ {
		ch := content[i]
		next := byte(0)
		if i+1 < len(content) {
			next = content[i+1]
		}
		if lineComment {
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if inSingle || inDouble || inTemplate {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if inSingle && ch == '\'' {
				inSingle = false
			}
			if inDouble && ch == '"' {
				inDouble = false
			}
			if inTemplate && ch == '`' {
				inTemplate = false
			}
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			blockComment = true
			i++
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func jsLocalNamedImportFindings(rel, content string, modules map[string]string) []string {
	var findings []string
	importRe := regexp.MustCompile(`(?m)\bimport\s*(?:type\s*)?\{([^}]+)\}\s*from\s*["']([^"']+)["']`)
	for _, match := range importRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 3 {
			continue
		}
		spec := strings.TrimSpace(match[2])
		if !strings.HasPrefix(spec, ".") {
			continue
		}
		names := jsNamedImportNames(match[1])
		if len(names) == 0 {
			continue
		}
		targetRel, ok := resolveLocalJSModuleRel(rel, spec, modules)
		if !ok {
			findings = append(findings, fmt.Sprintf("%s imports {%s} from %s but no matching local module file was found", rel, strings.Join(names, ", "), spec))
			continue
		}
		exported := jsExportedNames(modules[targetRel])
		for _, name := range names {
			if name == "default" {
				continue
			}
			if !exported[name] {
				findings = append(findings, fmt.Sprintf("%s imports {%s} from %s but %s does not export it", rel, name, targetRel, targetRel))
			}
		}
	}
	return findings
}

func jsMissingLocalExportImportFindings(rel, content string, modules map[string]string) []string {
	if !jsContainsModuleSyntax(content) {
		return nil
	}
	exportsByName := map[string][]string{}
	for moduleRel, moduleContent := range modules {
		if moduleRel == rel {
			continue
		}
		for name := range jsExportedNames(moduleContent) {
			exportsByName[name] = append(exportsByName[name], moduleRel)
		}
	}
	var findings []string
	for name, exporters := range exportsByName {
		if !jsUsesIdentifier(content, name) || jsDefinesOrImportsIdentifier(content, name) {
			continue
		}
		sort.Strings(exporters)
		findings = append(findings, fmt.Sprintf("%s uses %s but does not import it from local module %s", rel, name, exporters[0]))
	}
	sort.Strings(findings)
	return findings
}

func jsUsesIdentifier(content, id string) bool {
	quoted := regexp.QuoteMeta(id)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)\b` + quoted + `\s*\(`),
		regexp.MustCompile(`(?m)\bnew\s+` + quoted + `\b`),
		regexp.MustCompile(`(?m)\b` + quoted + `\s*\.`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func jsNamedImportNames(raw string) []string {
	var names []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "type ")
		if part == "" {
			continue
		}
		asParts := regexp.MustCompile(`(?i)\s+as\s+`).Split(part, 2)
		name := strings.TrimSpace(asParts[0])
		fields := strings.Fields(name)
		if len(fields) == 0 {
			continue
		}
		name = fields[0]
		if regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
			names = append(names, name)
		}
	}
	return names
}

func resolveLocalJSModuleRel(sourceRel, spec string, modules map[string]string) (string, bool) {
	spec = strings.TrimSpace(spec)
	if i := strings.IndexAny(spec, "?#"); i >= 0 {
		spec = spec[:i]
	}
	if spec == "" || !strings.HasPrefix(spec, ".") {
		return "", false
	}
	sourceDir := filepath.ToSlash(filepath.Dir(sourceRel))
	if sourceDir == "." {
		sourceDir = ""
	}
	base := filepath.ToSlash(filepath.Clean(filepath.Join(sourceDir, spec)))
	if base == "." || strings.HasPrefix(base, "../") || base == ".." {
		return "", false
	}
	candidates := []string{base}
	if ext := strings.ToLower(filepath.Ext(base)); ext == "" {
		for _, suffix := range []string{".js", ".mjs", ".jsx", ".ts", ".tsx"} {
			candidates = append(candidates, base+suffix)
		}
		for _, suffix := range []string{"index.js", "index.mjs", "index.jsx", "index.ts", "index.tsx"} {
			candidates = append(candidates, filepath.ToSlash(filepath.Join(base, suffix)))
		}
	}
	for _, candidate := range candidates {
		candidate = cleanRepoPath(candidate)
		if _, ok := modules[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

func jsExportedNames(content string) map[string]bool {
	exported := map[string]bool{}
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)\bexport\s+(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
		regexp.MustCompile(`(?m)\bexport\s+class\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
		regexp.MustCompile(`(?m)\bexport\s+(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
	} {
		for _, match := range pattern.FindAllStringSubmatch(content, -1) {
			if len(match) >= 2 {
				exported[match[1]] = true
			}
		}
	}
	exportListRe := regexp.MustCompile(`(?s)\bexport\s*\{([^}]+)\}`)
	for _, match := range exportListRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			part = strings.TrimSpace(part)
			part = strings.TrimPrefix(part, "type ")
			if part == "" {
				continue
			}
			asParts := regexp.MustCompile(`(?i)\s+as\s+`).Split(part, 2)
			name := strings.TrimSpace(asParts[len(asParts)-1])
			if fields := strings.Fields(name); len(fields) > 0 {
				name = fields[0]
			}
			if regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
				exported[name] = true
			}
		}
	}
	return exported
}

func htmlClassicScriptModuleFindings(rel, content string, modules map[string]string) []string {
	var findings []string
	tagRe := regexp.MustCompile(`(?is)<script\b([^>]*)>`)
	srcRe := regexp.MustCompile(`(?is)\bsrc\s*=\s*["']([^"']+)["']`)
	typeModuleRe := regexp.MustCompile(`(?is)\btype\s*=\s*["']module["']`)
	for _, tag := range tagRe.FindAllStringSubmatch(content, -1) {
		if len(tag) < 2 {
			continue
		}
		attrs := tag[1]
		if typeModuleRe.MatchString(attrs) {
			continue
		}
		src := srcRe.FindStringSubmatch(attrs)
		if len(src) < 2 {
			continue
		}
		targetRel, ok := resolveHTMLScriptRel(rel, src[1], modules)
		if !ok {
			continue
		}
		if jsContainsModuleSyntax(modules[targetRel]) {
			findings = append(findings, fmt.Sprintf("%s loads %s as a classic script but %s contains ES module import/export syntax; use type=\"module\" or bundle the entrypoint", rel, targetRel, targetRel))
		}
	}
	return findings
}

func resolveHTMLScriptRel(sourceRel, spec string, modules map[string]string) (string, bool) {
	spec = strings.TrimSpace(spec)
	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") || strings.HasPrefix(spec, "//") {
		return "", false
	}
	if i := strings.IndexAny(spec, "?#"); i >= 0 {
		spec = spec[:i]
	}
	sourceDir := filepath.ToSlash(filepath.Dir(sourceRel))
	if sourceDir == "." {
		sourceDir = ""
	}
	base := filepath.ToSlash(filepath.Clean(filepath.Join(sourceDir, spec)))
	if base == "." || strings.HasPrefix(base, "../") || base == ".." {
		return "", false
	}
	base = cleanRepoPath(base)
	_, ok := modules[base]
	return base, ok
}

func jsContainsModuleSyntax(content string) bool {
	return regexp.MustCompile(`(?m)^\s*(?:import|export)\b`).MatchString(content)
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
	if successfulReviewDispositionStatus(status) {
		if blockers := engineerBrowserFrameworkCompletionBlockers(root, session); len(blockers) > 0 {
			return fmt.Errorf(
				"policy: engineer cannot record a successful browser-framework disposition yet: %s. Fix the implementation or package build validation, rerun validation, update ticket evidence, and then record qa_review",
				strings.Join(blockers, "; "),
			)
		}
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
		rel := filepath.ToSlash(strings.TrimSpace(file))
		if rel == runtimeLearningsPath || IsWorkspaceNoisePath(rel) {
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

func checkEngineerBrowserFrameworkTicketDoneMovePolicy(root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
	}
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellFieldsPreserveCase(args.ShellCommand)
	}
	if len(ticketDoneMoveSources(fields)) == 0 {
		return nil
	}
	if blockers := engineerBrowserFrameworkCompletionBlockers(root, session); len(blockers) > 0 {
		return fmt.Errorf(
			"policy: engineer cannot move browser-framework ticket to docs/tickets/done yet: %s. Fix the implementation or package build surface, rerun validation, then update evidence and move the ticket",
			strings.Join(blockers, "; "),
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
			if line != "" && !IsGeneratedWorkspacePath(line) && !IsWorkspaceNoisePath(line) && !seen[line] {
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
			if line != "" && !IsGeneratedWorkspacePath(line) && !IsWorkspaceNoisePath(line) && !seen[line] {
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
