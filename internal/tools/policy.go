/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-007-guardrails-and-safety.md
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
	dogfoodTicketCreateLimitTotal       = 5
	dogfoodTicketCreateLimitPerSeverity = 3
	dogfoodTicketCreateLimitPerGroup    = 2
	runtimeLearningsPath                = ".harness/learnings.yaml"
)

func preToolPolicy(ctx context.Context, root Root, name string, raw json.RawMessage) error {
	session, hasSession := SessionFromContext(ctx)
	if hasSession {
		if err := enforceTrust(session, name); err != nil {
			return err
		}
	}

	switch name {
	case "file_write":
		if err := checkFileWritePolicy(root, session, hasSession, raw); err != nil {
			return err
		}
		return checkEngineerClaimBeforeProductMutation(root, session, hasSession, name, raw)
	case "ticket_create":
		return checkTicketCreatePolicy(root, session, hasSession, raw)
	case "job_disposition_record":
		return checkJobDispositionRecordPolicy(ctx, root, session, hasSession, raw)
	case "dependency_sync", "mars_harness_cli":
		return checkEngineerClaimBeforeProductMutation(root, session, hasSession, name, raw)
	case "git_commit":
		if err := checkEngineerClaimBeforeProductMutation(root, session, hasSession, name, raw); err != nil {
			return err
		}
		var args gitCommitArgs
		if err := json.Unmarshal(raw, &args); err == nil {
			if err := checkGitCommitGeneratedWorkspacePolicy(ctx, root, args); err != nil {
				return err
			}
		}
		return validateRepoDiff(ctx, root, session)
	case "git_push":
		return checkGitPushPolicy(ctx, root, raw)
	case "shell_exec":
		generatedArtifactCleanup, err := shellExecGeneratedArtifactCleanup(ctx, root, raw)
		if err != nil {
			return err
		}
		if !generatedArtifactCleanup {
			if err := checkShellPolicy(raw); err != nil {
				return err
			}
		}
		if err := checkShellTicketDoneEvidencePolicy(root, raw); err != nil {
			return err
		}
		if generatedArtifactCleanup {
			return nil
		}
		if hasSession && strings.ToLower(strings.TrimSpace(session.Role)) == "coo" && !shellExecReadOnly(raw) {
			return fmt.Errorf("policy: coo cannot run mutating shell_exec; update planning docs with file_write and use git tools for commit/push, while implementation stays behind CTO tickets and Engineer delivery")
		}
		if !shellExecReadOnly(raw) {
			if err := checkEngineerClaimBeforeProductMutation(root, session, hasSession, name, raw); err != nil {
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
		parts := strings.Split(strings.TrimSpace(strings.ToUpper(scenario)), "-")
		if len(parts) < 2 || parts[0] != "F" || parts[1] == "" {
			continue
		}
		id := "F-" + parts[1]
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
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

func checkEngineerClaimBeforeProductMutation(root Root, session Session, hasSession bool, toolName string, raw json.RawMessage) error {
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
	if len(inProgress) > 0 {
		return nil
	}
	backlog, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusBacklog)
	if err != nil {
		return fmt.Errorf("policy: inspect backlog tickets before %s: %w", toolName, err)
	}
	backlog = ordinaryProductTickets(backlog)
	if len(backlog) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer must claim a product ticket before %s mutates product files; move %s from %s to docs/tickets/in-progress/ with git mv, commit the claim, then continue",
		toolName,
		backlog[0].ID,
		backlog[0].RelPath,
	)
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
		return shellExecMovesBacklogTicketToInProgress(raw)
	default:
		return false
	}
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
	if len(dupes) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: feature contract %s has duplicate scenario ID heading(s): %s; each scenario heading such as `### F-001-S001` may appear once. Read the current file and replace the existing scenario section in one full-file write; do not append a second heading. Scenario Schedule list entries may repeat the ID and are not the duplicate.",
		rel,
		formatFeatureScenarioDuplicates(dupes),
	)
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
		Status   string `json:"status"`
		TicketID string `json:"ticket_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	if err := checkEngineerDispositionTicketState(root, session, args.Status, args.TicketID); err != nil {
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
		return nil
	}
	return fmt.Errorf("policy: job_disposition_record cannot complete while repository has uncommitted changes: %s. Run git_status, commit the changed work with git_commit, then record the disposition", summarizeChangedFiles(files))
}

func checkEngineerDispositionTicketState(root Root, session Session, status, ticketID string) error {
	if strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
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

func checkShellPolicy(raw json.RawMessage) error {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
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

func shellExecGeneratedArtifactCleanup(ctx context.Context, root Root, raw json.RawMessage) (bool, error) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return false, nil
	}
	paths, ok := shellRemovalPathOperands(args)
	if !ok || len(paths) == 0 {
		return false, nil
	}
	for _, rel := range paths {
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

func checkShellTicketDoneEvidencePolicy(root Root, raw json.RawMessage) error {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
	}
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellFields(args.ShellCommand)
	}
	if copies := ticketDoneCopySources(fields); len(copies) > 0 {
		return fmt.Errorf(
			"policy: feature ticket %s cannot be copied into docs/tickets/done; update evidence in the current lifecycle file, then use git mv so only one lifecycle copy exists",
			filepath.Base(copies[0]),
		)
	}
	for _, source := range ticketDoneMoveSources(fields) {
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

func ticketDoneCopySources(fields []string) []string {
	var sources []string
	for i, field := range fields {
		if filepathBase(field) != "cp" {
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
		switch filepathBase(field) {
		case "git":
			if i+1 >= len(fields) || strings.ToLower(strings.TrimSpace(fields[i+1])) != "mv" {
				continue
			}
			if source, dest, ok := ticketMoveOperands(fields[i+2:]); ok && ticketMoveTargetsDone(source, dest) {
				sources = append(sources, cleanShellPathToken(source))
			}
		case "mv":
			if i > 0 && filepathBase(fields[i-1]) == "git" {
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
	var args shellExecArgs
	if err := json.Unmarshal(raw, &args); err != nil {
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
		return sourceOK && sourceState == "backlog"
	}
	_, destState, destOK := ticketLifecyclePathIdentity(dest)
	_, sourceState, sourceOK := ticketLifecyclePathIdentity(source)
	return destOK && destState == "in-progress" && sourceOK && sourceState == "backlog"
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
	if workType != "feature" {
		missing = append(missing, "work_type: feature")
	}
	if endToEndEvidence != "required" {
		missing = append(missing, "end_to_end_evidence: required")
	}
	if ticketEvidenceFieldEmpty(frontmatter["bdd_scenarios"]) {
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
	return safety.Check(stats, limits)
}

// ValidateRepoDiff checks the current repository diff against the same safety
// limits enforced after mutating tool calls.
func ValidateRepoDiff(ctx context.Context, root Root, session Session) error {
	return validateRepoDiff(ctx, root, session)
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
