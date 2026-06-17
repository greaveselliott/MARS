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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/docsync"
	"github.com/greaveselliott/mars-harness/internal/safety"
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
	staticProductSmokeSuccessKey              = "validation:static_product_smoke:success"
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
	if err := checkAgentSmokeReviewReportCommitSequence(ctx, root, session, hasSession, name); err != nil {
		return err
	}
	if err := checkEngineerMissingArgumentCorrectionOnly(session, hasSession, name); err != nil {
		return err
	}
	if err := checkReviewTerminalDispositionOnly(ctx, root, session, hasSession, name); err != nil {
		return err
	}

	switch name {
	case "file_read":
		return checkFileReadPolicy(root, raw)
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
		if args, err := decodeGitCommitArgs(raw); err == nil {
			if err := checkPlannerGitCommitPolicy(ctx, root, session, hasSession, args); err != nil {
				return err
			}
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
		if err := checkDogfoodBrowserPostBuildSmokeOnlyPolicy(root, session, hasSession, args); err != nil {
			return err
		}
		if err := checkAgentSmokePipelineFixerProjectValidationPolicy(root, session, hasSession, args); err != nil {
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

func repoFileExists(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
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
	if err := checkPackageScriptRuntimePolicy(args.Path, args.Content); err != nil {
		return err
	}
	if err := checkMakefileBuildOutputWritePolicy(root, args.Path, args.Content); err != nil {
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
	case "head-of-strategy":
		if headOfStrategyWritePath(rel) {
			return nil
		}
		return fmt.Errorf("policy: head-of-strategy may only write strategy artifacts under docs/goals/observations.md, docs/product-specs/vision.md, or docs/reports/strategy/; implementation path %s belongs behind COO/CTO handoff and Engineer delivery", rel)
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

func checkPlannerGitCommitPolicy(ctx context.Context, root Root, session Session, hasSession bool, args gitCommitArgs) error {
	if !hasSession || !planningRoleCannotMutateWithShell(session.Role) {
		return nil
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	paths := make([]string, 0, len(args.Paths))
	for _, path := range args.Paths {
		rel := cleanRepoPath(path)
		if rel != "" {
			paths = append(paths, rel)
		}
	}
	if len(paths) == 0 {
		var err error
		paths, err = changedFiles(ctx, root)
		if err != nil {
			return fmt.Errorf("policy: inspect changed files before %s git_commit: %w", role, err)
		}
	}
	for _, rel := range paths {
		rel = cleanRepoPath(rel)
		if rel == "" || plannerCommitAllowedPath(role, rel) {
			continue
		}
		return fmt.Errorf("policy: %s cannot git_commit product or downstream implementation path %s; leave product/source/package changes for Engineer and commit only owned planning artifacts", role, rel)
	}
	return nil
}

func plannerCommitAllowedPath(role, rel string) bool {
	rel = cleanRepoPath(rel)
	if rel == runtimeLearningsPath {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "ceo":
		return ceoStrategyWritePath(rel)
	case "head-of-strategy":
		return headOfStrategyWritePath(rel)
	case "coo":
		return cooPlanningWritePath(rel)
	case "cto", "cto-weekly":
		return ctoTechnicalPlanningWritePath(rel) || ticketWritePath(rel)
	default:
		return true
	}
}

func ticketWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	return rel == "docs/tickets/readme.md" || (strings.HasPrefix(rel, "docs/tickets/") && strings.HasSuffix(strings.ToLower(rel), ".md"))
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

func headOfStrategyWritePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == filepath.ToSlash(filepath.Join("docs", "goals", "observations.md")) {
		return true
	}
	if rel == filepath.ToSlash(filepath.Join("docs", "product-specs", "vision.md")) {
		return true
	}
	return strings.HasPrefix(rel, "docs/reports/strategy/") && strings.HasSuffix(strings.ToLower(rel), ".md")
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

const runtimeValidationEditAfterFailureKey = "validation:runtime_unexpected_failure:edit_after"

const testBuildValidationEditAfterFailureKey = "validation:test_build_failure:edit_after"

func shellExecCommandDisplay(args shellExecArgs) string {
	if len(args.Argv) > 0 {
		return strings.Join(args.Argv, " ")
	}
	return strings.TrimSpace(args.ShellCommand)
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

func shellControlToken(field string) bool {
	switch strings.TrimSpace(field) {
	case "&&", "||", "|", ";", "&":
		return true
	default:
		return false
	}
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
