/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/docsync"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

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
	if err := checkAgentSmokeDispositionRequiredEvidence(root, session, args.Status); err != nil {
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
	if guidance := agentSmokeDependencyLockDispositionGuidance(root, files); guidance != "" {
		return fmt.Errorf("policy: job_disposition_record cannot complete while repository has uncommitted changes: %s. %s", summarizeChangedFiles(files), guidance)
	}
	return fmt.Errorf("policy: job_disposition_record cannot complete while repository has uncommitted changes: %s. Run git_status, commit the changed work with git_commit, then record the disposition", summarizeChangedFiles(files))
}

func agentSmokeDependencyLockDispositionGuidance(root Root, files []string) string {
	if !agentSmokeContractPresent(root) {
		return ""
	}
	var locks []string
	for _, file := range files {
		rel := filepath.ToSlash(strings.TrimSpace(file))
		switch rel {
		case "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb":
			locks = append(locks, rel)
		}
	}
	if len(locks) == 0 {
		return ""
	}
	return fmt.Sprintf("In agent-smoke, dependency lockfiles created by dependency_sync are validation provenance, not cleanup debris. Do not remove them with shell_exec. Run git_status, then git_commit with paths [%s], then retry job_disposition_record.", quotedPathList(locks))
}

func agentSmokeContractPresent(root Root) bool {
	contractPath := filepath.Join(root.Abs(), "docs", "validation", "agent-smoke", "current-case.md")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "# Agent Smoke Case Contract")
}

func quotedPathList(paths []string) string {
	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		quoted = append(quoted, fmt.Sprintf("%q", path))
	}
	return strings.Join(quoted, ", ")
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
		if len(scenarios) == 0 {
			continue
		}
		if !featureHasCompletedValidationTicket(root, featureID) {
			requiredScenarios := firstSliceCTOHandoffRequiredScenarios(root, featureID, scenarios)
			if len(requiredScenarios) == 0 {
				continue
			}
			required := requiredScenarios[0]
			if featureHasExactFirstSliceTicket(root, featureID, required) {
				continue
			}
			next := []string{required}
			recordCTOHandoffRequiredScenarios(session, next)
			return fmt.Errorf("policy: cto cannot hand off implementation for %s before the first executable slice is ticketed. %s", featureID, ctoFirstSliceTicketCreateGuidance(next))
		}
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
		return fmt.Errorf("policy: cto cannot hand off implementation for %s after covering only %d/%d early product scenario(s). %s", featureID, coveredCount, required, ctoTicketCreateGuidance(next))
	}
	return nil
}

func firstSliceCTOHandoffRequiredScenarios(root Root, featureID string, scenarios []string) []string {
	if len(scenarios) == 0 {
		return nil
	}
	if current := activePlanCurrentFailingScenarios(root, featureID, scenarios); len(current) > 0 {
		return current[:1]
	}
	if productScenarios := productScenarioIDsForHandoff(root, featureID, scenarios); len(productScenarios) > 0 {
		return productScenarios[:1]
	}
	return scenarios[:1]
}

func activePlanCurrentFailingScenarios(root Root, featureID string, scenarios []string) []string {
	path, err := root.ResolvePath(filepath.Join("docs", "exec-plans", "active", "current-operating-plan.md"))
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	scenarioSet := map[string]bool{}
	for _, scenario := range scenarios {
		scenarioSet[strings.ToUpper(strings.TrimSpace(scenario))] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "**Current Failing Scenario:**") {
			continue
		}
		for _, scenario := range toolsFeatureScenarioIDPattern.FindAllString(trimmed, -1) {
			scenario = strings.ToUpper(strings.TrimSpace(scenario))
			if seen[scenario] || !scenarioSet[scenario] || featureIDFromScenarioIDMust(scenario) != featureID {
				continue
			}
			seen[scenario] = true
			out = append(out, scenario)
		}
	}
	return out
}

func featureHasCompletedValidationTicket(root Root, featureID string) bool {
	tickets, err := ticketstate.List(root.Abs())
	if err != nil {
		return false
	}
	for _, t := range tickets {
		if t.Status != ticketstate.StatusDone || t.Kind == "intervention-debt" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root.Abs(), filepath.FromSlash(t.RelPath)))
		if err != nil {
			continue
		}
		if !scenarioListIncludesFeature(orderedFeatureScenarioIDs(string(data)), featureID) {
			continue
		}
		frontmatter := parseTicketPolicyFrontmatter(string(data))
		if hasFirstProofBuildSmokeEvidence(frontmatter) {
			return true
		}
	}
	return false
}

func hasFirstProofBuildSmokeEvidence(frontmatter map[string]string) bool {
	if len(missingFeatureTicketEvidence(frontmatter)) > 0 {
		return false
	}
	evidence := strings.ToLower(frontmatter["evidence_links"] + "\n" + frontmatter["verified_by"])
	hasBuildOrTest := strings.Contains(evidence, "go test") ||
		strings.Contains(evidence, "go build") ||
		strings.Contains(evidence, "npm test") ||
		strings.Contains(evidence, "npm run build") ||
		strings.Contains(evidence, "npm run test") ||
		strings.Contains(evidence, "cargo test") ||
		strings.Contains(evidence, "cargo build")
	hasSmoke := strings.Contains(evidence, "smoke") ||
		strings.Contains(evidence, "curl") ||
		strings.Contains(evidence, "http") ||
		strings.Contains(evidence, "browser") ||
		strings.Contains(evidence, "playwright") ||
		strings.Contains(evidence, "python -m http.server") ||
		strings.Contains(evidence, "/health")
	return hasBuildOrTest && hasSmoke
}

func featureHasExactFirstSliceTicket(root Root, featureID, requiredScenario string) bool {
	tickets, err := ticketstate.List(root.Abs())
	if err != nil {
		return false
	}
	requiredScenario = strings.ToUpper(strings.TrimSpace(requiredScenario))
	for _, t := range tickets {
		if t.Kind == "intervention-debt" || (t.Status != ticketstate.StatusBacklog && t.Status != ticketstate.StatusInProgress) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root.Abs(), filepath.FromSlash(t.RelPath)))
		if err != nil {
			continue
		}
		scenarios := featureScenariosForID(orderedFeatureScenarioIDs(string(data)), featureID)
		if len(scenarios) == 1 && scenarios[0] == requiredScenario {
			return true
		}
	}
	return false
}

func scenarioListIncludesFeature(scenarios []string, featureID string) bool {
	featureID = strings.ToUpper(strings.TrimSpace(featureID))
	for _, scenario := range scenarios {
		if featureIDFromScenarioIDMust(scenario) == featureID {
			return true
		}
	}
	return false
}

func recordCTOHandoffRequiredScenarios(session Session, scenarios []string) {
	if len(scenarios) == 0 || session.ToolState == nil {
		return
	}
	session.ToolState[ctoHandoffRequiredScenariosKey] = strings.Join(scenarios, ",")
}

func featureScenariosForID(scenarios []string, featureID string) []string {
	featureID = strings.ToUpper(strings.TrimSpace(featureID))
	var out []string
	seen := map[string]bool{}
	for _, scenario := range scenarios {
		scenario = strings.ToUpper(strings.TrimSpace(scenario))
		if scenario == "" || featureIDFromScenarioIDMust(scenario) != featureID || seen[scenario] {
			continue
		}
		seen[scenario] = true
		out = append(out, scenario)
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

func checkAgentSmokeDispositionRequiredEvidence(root Root, session Session, status string) error {
	if !agentSmokeDispositionStatusRequiresEvidence(status) {
		return nil
	}
	contractPath := filepath.Join(root.Abs(), "docs", "validation", "agent-smoke", "current-case.md")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		return nil
	}
	contract := string(data)
	if !strings.Contains(contract, "# Agent Smoke Case Contract") {
		return nil
	}
	caseID := agentSmokeContractField(contract, "Case")
	role := strings.ToLower(strings.TrimSpace(session.Role))
	required := agentSmokeRequiredArtifactsFromContract(contract)
	switch role {
	case "qa":
		if caseID != "" {
			required = append(required, "docs/reports/qa/"+caseID+".md")
		}
	case "security":
		if caseID != "" {
			required = append(required, "docs/reports/security/"+caseID+".md")
		}
	case "dogfood":
		if caseID != "" {
			required = append(required, "docs/reports/dogfood/"+caseID+".md")
		}
	case "release-manager":
		stage := strings.ToLower(agentSmokeContractField(contract, "Stage"))
		if caseID != "" && strings.Contains(stage, "ready") {
			required = append(required, "docs/reports/release/"+caseID+".md")
		}
	}
	required = uniqueNonEmptyPaths(required)
	var missing []string
	for _, rel := range required {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err != nil {
			missing = append(missing, rel)
		}
	}
	if len(missing) == 0 {
		if role == "release-manager" && successfulDispositionStatus(status) {
			stage := strings.ToLower(agentSmokeContractField(contract, "Stage"))
			if caseID != "" && strings.Contains(stage, "ready") {
				return checkAgentSmokeReleaseTagAtHead(root, caseID)
			}
		}
		return nil
	}
	if caseID == "" {
		caseID = "unknown"
	}
	return fmt.Errorf("policy: agent-smoke %s cannot record terminal disposition before required evidence exists: %s. Create the missing artifact(s), run git_status, git_commit any changes, then retry job_disposition_record", caseID, strings.Join(missing, ", "))
}

func checkAgentSmokeReleaseTagAtHead(root Root, caseID string) error {
	versionPath, err := root.ResolvePath("VERSION")
	if err != nil {
		return nil
	}
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return nil
	}
	version := strings.TrimSpace(string(versionData))
	if version == "" {
		return nil
	}
	tag := "v" + strings.TrimPrefix(version, "v")
	list, err := runGit(context.Background(), root, "tag", "--list", tag)
	if err != nil {
		return nil
	}
	if strings.TrimSpace(list.Output) != tag {
		return fmt.Errorf("policy: agent-smoke %s cannot record release completion before local tag %s exists at HEAD. Write docs/reports/release/%s.md before the release-note commit, commit VERSION, CHANGELOG, and the report together with git_commit message %q, then create the tag with shell_exec argv [\"git\",\"tag\",\"%s\"] and retry job_disposition_record", caseID, tag, caseID, "release: notes "+strings.TrimPrefix(tag, "v"), tag)
	}
	tagCommit, err := runGit(context.Background(), root, "rev-list", "-n", "1", tag)
	if err != nil || tagCommit.ExitCode != 0 {
		return nil
	}
	headCommit, err := runGit(context.Background(), root, "rev-parse", "HEAD")
	if err != nil || headCommit.ExitCode != 0 {
		return nil
	}
	if strings.TrimSpace(tagCommit.Output) == strings.TrimSpace(headCommit.Output) {
		return nil
	}
	return fmt.Errorf("policy: agent-smoke %s cannot record release completion because local tag %s does not point at the current release-note evidence commit. The tag must point at a commit whose subject is %q and whose history includes docs/reports/release/%s.md. If the report was committed after the tag, update CHANGELOG.md with a report evidence line, commit it with git_commit message %q, then move the local tag to HEAD with shell_exec argv [\"git\",\"tag\",\"-f\",\"%s\"] and retry job_disposition_record", caseID, tag, "release: notes "+strings.TrimPrefix(tag, "v"), caseID, "release: notes "+strings.TrimPrefix(tag, "v"), tag)
}

func agentSmokeDispositionStatusRequiresEvidence(status string) bool {
	if successfulDispositionStatus(status) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(status), "changes_requested")
}

func agentSmokeContractField(contract, field string) string {
	pattern := regexp.MustCompile(`(?m)^-\s+` + regexp.QuoteMeta(field) + `:\s+` + "`" + `([^` + "`" + `]+)` + "`")
	match := pattern.FindStringSubmatch(contract)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func agentSmokeRequiredArtifactsFromContract(contract string) []string {
	lines := strings.Split(contract, "\n")
	inSection := false
	var required []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inSection = trimmed == "## Required Artifacts"
			continue
		}
		if !inSection || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		value = strings.Trim(value, "`")
		if value != "" {
			required = append(required, value)
		}
	}
	return required
}

func uniqueNonEmptyPaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, path := range paths {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
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

func successfulReviewDispositionStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "approved", "in_review":
		return true
	default:
		return false
	}
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
	if successfulReviewDispositionStatus(status) && engineerInValidationFailedRuntimeLane(session) {
		return unresolvedRuntimeValidationCompletionError("record a successful product disposition", session)
	}
	if successfulReviewDispositionStatus(status) && engineerInValidationFailedTestBuildLane(session) {
		return unresolvedTestBuildValidationCompletionError("record a successful product disposition", session)
	}
	if successfulReviewDispositionStatus(status) && engineerInValidatedBrowserCompletionPhase(root, session) {
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
