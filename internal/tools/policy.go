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

var toolsFeatureIDPattern = regexp.MustCompile(`\bF-\d{3}\b`)
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

func repoFileExists(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
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
	for _, featureID := range ctoHandoffFeatureContractIDs(root) {
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

func ctoHandoffFeatureContractIDs(root Root) []string {
	planIDs := activePlanFeatureIDs(root)
	if len(planIDs) == 0 {
		return featureContractIDs(root)
	}
	existing := map[string]bool{}
	for _, id := range featureContractIDs(root) {
		existing[id] = true
	}
	var out []string
	for _, id := range planIDs {
		if existing[id] {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return featureContractIDs(root)
	}
	return out
}

func activePlanFeatureIDs(root Root) []string {
	path, err := root.ResolvePath(filepath.Join("docs", "exec-plans", "active", "current-operating-plan.md"))
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ids []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "**BDD Feature:**") &&
			!strings.HasPrefix(trimmed, "**Scenario Schedule:**") &&
			!strings.HasPrefix(trimmed, "**Current Failing Scenario:**") {
			continue
		}
		for _, scenarioID := range toolsFeatureScenarioIDPattern.FindAllString(trimmed, -1) {
			id := featureIDFromScenarioIDMust(scenarioID)
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		for _, id := range toolsFeatureIDPattern.FindAllString(trimmed, -1) {
			id = strings.ToUpper(id)
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
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
	for _, featureID := range ctoHandoffFeatureContractIDs(root) {
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

