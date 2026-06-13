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
