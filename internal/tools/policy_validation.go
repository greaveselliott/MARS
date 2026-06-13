/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

func checkEngineerPostValidationCompletionShellPolicy(ctx context.Context, root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || !engineerInValidatedPhase(session) {
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
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedRuntimeLane(session) {
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
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedRuntimeLane(session) {
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

func engineerInValidationFailedTestBuildLane(session Session) bool {
	state := session.EngineerDeliveryState()
	return state.Phase == DeliveryPhaseValidationFailed && state.RepairLane == RepairLaneTestBuild
}

func engineerInValidationFailedRuntimeLane(session Session) bool {
	state := session.EngineerDeliveryState()
	return state.Phase == DeliveryPhaseValidationFailed && state.RepairLane == RepairLaneRuntime
}

func checkEngineerUnresolvedTestBuildValidationBeforeCompletion(ctx context.Context, root Root, session Session, hasSession bool, toolName string, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedTestBuildLane(session) {
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
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedRuntimeLane(session) {
		return nil
	}
	return unresolvedRuntimeValidationCommitError(session)
}

func checkEngineerUnresolvedTestBuildValidationBeforeCommit(session Session, hasSession bool) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedTestBuildLane(session) {
		return nil
	}
	return unresolvedTestBuildValidationCommitError(session)
}

func checkEngineerUnresolvedTestBuildValidationBeforeFileWrite(session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedTestBuildLane(session) {
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
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedRuntimeLane(session) {
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
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedTestBuildLane(session) {
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
	if strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedTestBuildLane(session) {
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
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !engineerInValidationFailedRuntimeLane(session) {
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

func runtimeValidationFailureEditWatermarkKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:runtime_unexpected_failure_edit_watermark:%s", shellExecCommandFingerprint(args))
}

func testBuildValidationFailureFingerprintKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:test_build_failure:%s", shellExecCommandFingerprint(args))
}

func testBuildValidationRepairKey(args shellExecArgs) string {
	return fmt.Sprintf("validation:test_build_repair:%s", shellExecCommandFingerprint(args))
}

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
