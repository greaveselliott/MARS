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
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

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
	if !reviewSuccessfulValidationEvidence(counts) {
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
	if blockers := staticBrowserCompletionBlockers(root, session); len(blockers) > 0 {
		return fmt.Errorf(
			"policy: %s cannot approve static browser ticket %s yet: %s. Record changes_requested with feedback.for_role engineer when the served-page smoke cannot be produced",
			strings.ToLower(strings.TrimSpace(session.Role)),
			strings.ToUpper(strings.TrimSpace(ticketID)),
			strings.Join(blockers, "; "),
		)
	}
	return nil
}

func checkReviewTerminalDispositionOnly(ctx context.Context, root Root, session Session, hasSession bool, name string) error {
	if !hasSession || !reviewRoleRequiresValidationEvidence(session.Role) || name == "job_disposition_record" {
		return nil
	}
	if path := missingAgentSmokeReviewReportPath(root, session); path != "" && ReviewTerminalEvidenceSatisfied(root, &session) {
		switch name {
		case "file_write", "git_status", "git_commit":
			return nil
		default:
			return fmt.Errorf("policy: %s already has sufficient validation for agent-smoke but the required report is missing. Do not call more tools except file_write to %s, git_status, git_commit, then job_disposition_record", strings.ToLower(strings.TrimSpace(session.Role)), path)
		}
	}
	if agentSmokeReviewReportHasUncommittedChanges(ctx, root, session) {
		switch name {
		case "file_write", "git_status", "git_commit":
			return nil
		}
	}
	if session.ToolCounts == nil || session.ToolCounts[reviewTerminalDispositionRequiredKey] == 0 {
		return nil
	}
	if path := missingAgentSmokeReviewReportPath(root, session); path != "" {
		switch name {
		case "file_write", "git_status", "git_commit", "job_disposition_record":
			return nil
		default:
			return fmt.Errorf("policy: %s already has sufficient validation for agent-smoke but the required report is missing. Do not call more tools except file_write to %s, git_status, git_commit, then job_disposition_record", strings.ToLower(strings.TrimSpace(session.Role)), path)
		}
	}
	return fmt.Errorf("policy: %s already received terminal disposition guidance after successful validation and a blocked shell_exec no-op. Do not call more tools except job_disposition_record. %s", strings.ToLower(strings.TrimSpace(session.Role)), reviewTerminalDispositionGuidance(root, session))
}

func checkAgentSmokeReviewReportCommitSequence(ctx context.Context, root Root, session Session, hasSession bool, name string) error {
	if !hasSession {
		return nil
	}
	blocking := agentSmokeReviewReportBlockingFiles(ctx, root, session)
	if len(blocking) == 0 {
		return nil
	}
	switch name {
	case "file_write", "git_status", "git_commit":
		return nil
	default:
		return fmt.Errorf("policy: agent-smoke role report is written but uncommitted: %s. Next run git_status, git_commit the report, then continue with docsync_audit or job_disposition_record", strings.Join(blocking, ", "))
	}
}

func agentSmokeReviewReportHasUncommittedChanges(ctx context.Context, root Root, session Session) bool {
	return len(agentSmokeReviewReportBlockingFiles(ctx, root, session)) > 0
}

func agentSmokeReviewReportBlockingFiles(ctx context.Context, root Root, session Session) []string {
	reportPath := agentSmokeReviewReportPath(root, session)
	if reportPath == "" {
		return nil
	}
	if _, err := root.RepoFS().Stat(reportPath); err != nil {
		return nil
	}
	files, err := changedFiles(ctx, root)
	if err != nil {
		return nil
	}
	var blocking []string
	for _, file := range files {
		file = filepath.ToSlash(strings.TrimSpace(file))
		if file == reportPath {
			blocking = append(blocking, file)
		}
	}
	return blocking
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
	if blockers := staticBrowserCompletionBlockers(root, session); len(blockers) > 0 {
		return "Call job_disposition_record with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links explaining that static browser completion is not proven: " + strings.Join(blockers, "; ") + "."
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
	if path := missingAgentSmokeReviewReportPath(root, session); path != "" {
		return "Write the exact agent-smoke report with file_write path " + path + ", run git_status and git_commit, then call job_disposition_record with the required terminal disposition fields."
	}
	return "Call job_disposition_record now with status approved, ticket_id, next_need security_review or no_need, and evidence_links naming the build/test/runtime commands that passed."
}

func reviewSuccessfulValidationEvidence(counts map[string]int) bool {
	if counts == nil {
		return false
	}
	return counts[validationCommandSuccessKey] > 0 ||
		counts[testCommandSuccessKey] > 0 ||
		counts[buildCommandSuccessKey] > 0 ||
		counts[browserProductSmokeSuccessKey] > 0
}

func missingAgentSmokeReviewReportPath(root Root, session Session) string {
	reportPath := agentSmokeReviewReportPath(root, session)
	if reportPath == "" {
		return ""
	}
	if _, err := root.RepoFS().Stat(reportPath); err == nil {
		return ""
	}
	return reportPath
}

func agentSmokeReviewReportPath(root Root, session Session) string {
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role != "qa" && role != "security" && role != "dogfood" {
		return ""
	}
	data, err := root.RepoFS().ReadFile(filepath.Join("docs", "validation", "agent-smoke", "current-case.md"))
	if err != nil || !strings.Contains(string(data), "# Agent Smoke Case Contract") {
		return ""
	}
	caseID := agentSmokeContractField(string(data), "Case")
	if caseID == "" {
		return ""
	}
	return "docs/reports/" + role + "/" + caseID + ".md"
}

// ReviewTerminalEvidenceSatisfied reports whether a review role has gathered
// enough clean evidence that the agent loop should force the terminal
// disposition path instead of letting more inspection consume the job.
func ReviewTerminalEvidenceSatisfied(root Root, session *Session) bool {
	if session == nil || !reviewRoleRequiresValidationEvidence(session.Role) {
		return false
	}
	state := session.ReviewDeliveryState()
	if state.Phase == DeliveryPhaseTerminalDisposition {
		return true
	}
	if state.Phase != DeliveryPhaseValidated {
		return false
	}
	counts := session.ToolCounts
	if counts == nil {
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
	if len(staticBrowserCompletionBlockers(root, *session)) > 0 {
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
		if reviewerInValidatedPhase(session) && counts != nil && counts[testCommandFailureKey] == 0 && counts[buildCommandFailureKey] == 0 && counts[validationCommandFailureKey] == 0 {
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
		if reviewerInValidationFailedPhase(session) {
			return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders after failing validation. Stop shell validation and call job_disposition_record now with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links naming the failing command/output", role)
		}
		return fmt.Errorf("policy: %s cannot use shell_exec no-op placeholders during review; run a concrete read-only inspection or validation command, or record job_disposition_record with the quality decision", role)
	}
	if shellExecStopsTrackedBackgroundPID(session, args) {
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

func reviewRoleRequiresValidationEvidence(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "qa", "security":
		return true
	default:
		return false
	}
}

func shellExecRunsValidationCommand(args shellExecArgs) bool {
	return shellExecRunsTestCommand(args) || shellExecRunsBuildCommand(args) || shellExecRunsRuntimeValidationCommand(args) || shellExecRunsHTTPProbe(args)
}

func roleRequiresDocSyncForSuccessfulDisposition(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "engineer", "pipeline-fixer", "qa", "security", "dogfood", "release-manager", "dependency-manager":
		return true
	default:
		return false
	}
}

func repoHasTestFiles(root Root) bool {
	hasTests := false
	_ = fs.WalkDir(root.RepoFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || hasTests {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return fs.SkipDir
			default:
				return nil
			}
		}
		if testFilePath(filepath.ToSlash(path)) {
			hasTests = true
			return fs.SkipAll
		}
		return nil
	})
	return hasTests
}

func repoHasGoSourceFiles(root Root) bool {
	hasSource := false
	_ = fs.WalkDir(root.RepoFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || hasSource {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return fs.SkipDir
			default:
				return nil
			}
		}
		rel := filepath.ToSlash(strings.ToLower(strings.TrimSpace(path)))
		if strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go") {
			hasSource = true
			return fs.SkipAll
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
