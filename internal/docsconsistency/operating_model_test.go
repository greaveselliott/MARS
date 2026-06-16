/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/index.md
- docs/design-docs/validation-matrix-gating.md
- docs/features/F-001-delivery-operating-model.md
- docs/roles/personas/foundation-maintainer.md
- docs/validation/README.md
- docs/validation/reports/
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/docsync"
)

func TestAD074OperatingModelArtifactsExist(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"docs/design-docs/delivery-operating-model.md":        {"AD-074", "BDD-Led Goal-Driven Walking-Skeleton", "AD-097", "Business logic is first-class BDD", "AD-098", "No stale documentation", "AD-108", "Remote Trunk Freshness And Immediate Publishing", "AD-138", "Live Demo Improvement Loop", "origin/main", "MarsDocSync"},
		"docs/design-docs/harness-operating-model.md":         {"AD-084", "Planner", "End-to-End Tester", "`domain`", "`mode`", "AD-274", "foundation-maintainer"},
		"docs/design-docs/conversation-as-system-record.md":   {"AD-086", "Conversation As System Record", "Chat summaries can help humans catch up", "active-plan hygiene checker"},
		"docs/goals/README.md":                                {"Goal Schema", "Autonomous Goal Rule", "Dedupe Key"},
		"docs/goals/active.md":                                {"G-001", "Status: active", "Hypothesis"},
		"docs/goals/observations.md":                          {"weak/noisy evidence"},
		"docs/goals/superseded.md":                            {"Superseded Goals"},
		"docs/design-docs/code-documentation-map.md":          {"Code Documentation Map", "MarsDocSync", "docsync audit", "Package Map", "documentation-sync-architecture.md", "cli-tool-skill-sync.md"},
		"docs/design-docs/cli-tool-skill-sync.md":             {"AD-103", "CLI Tool And Skill Synchronization", "Universal Operating Model", "mars_harness_cli", "repo shortcut map"},
		"docs/design-docs/documentation-sync-architecture.md": {"AD-102", "Documentation Sync", "Universal Operating Model", "Architecture", "docsync_audit", "Generated Target Layer"},
		"docs/features/README.md":                             {"BDD Feature Contracts", "Business Logic Is First-Class BDD", "No Stale Documentation", "cli-tool-skill-sync.md", "Given/When/Then", "Scenario Schedule"},
		"docs/features/F-001-delivery-operating-model.md":     {"Feature ID: F-001", "Scenario Schedule", "F-001-S008", "F-001-S009", "F-001-S010", "F-001-S011", "F-001-S012", "F-001-S014", "Remote Trunk Freshness And Immediate Publishing", "No Stale Documentation", "Given", "When", "Then"},
		"docs/tickets/README.md":                              {"work_type", "bdd_scenarios", "end_to_end_evidence", "verified_by"},
		"docs/QUALITY_SCORE.md":                               {"shipped feature scenarios", "enabler work"},
	}
	for rel, needles := range required {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must contain %q", rel, needle)
			}
		}
	}
}

func TestOperatingModelCodeFilesDeclareDocSyncMetadata(t *testing.T) {
	root := repoRoot(t)
	report, err := docsync.Audit(docsync.Config{RepoRoot: root})
	if err != nil {
		t.Fatalf("docsync audit: %v", err)
	}
	if report.OK() {
		return
	}
	var lines []string
	for i, finding := range report.Findings {
		if i >= 20 {
			lines = append(lines, "...")
			break
		}
		lines = append(lines, finding.Path+": "+finding.Message)
	}
	t.Fatalf("docsync audit failed:\n%s", strings.Join(lines, "\n"))
}

func TestActivePlanReferencesActiveGoalAndFeatureContract(t *testing.T) {
	root := repoRoot(t)
	activeGoals, err := os.ReadFile(filepath.Join(root, "docs", "goals", "active.md"))
	if err != nil {
		t.Fatalf("read active goals: %v", err)
	}
	feature, err := os.ReadFile(filepath.Join(root, "docs", "features", "F-001-delivery-operating-model.md"))
	if err != nil {
		t.Fatalf("read feature contract: %v", err)
	}
	plan, err := os.ReadFile(filepath.Join(root, "docs", "exec-plans", "active", "current-operating-plan.md"))
	if err != nil {
		t.Fatalf("read active plan: %v", err)
	}
	planText := string(plan)
	for _, marker := range []string{"G-001", "F-001"} {
		if !strings.Contains(string(activeGoals)+string(feature), marker) {
			t.Fatalf("source artifacts must define %s", marker)
		}
		if !strings.Contains(planText, marker) {
			t.Fatalf("active plan must reference %s", marker)
		}
	}
}

func TestConfidenceGatedPlanningOperatingModelIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"docs/design-docs/foundation-operating-model.md": {
			"AD-298",
			"Confidence-Gated Planning For Foundation Observations",
			"Primary Outcome Contract",
			"source-only foundation doctrine",
			"Evidence And Classification",
			"Assumption Confidence Matrix",
			"`Assumption`",
			"`Evidence`",
			"`Confidence`",
			"`Validation Required`",
			"`0.0` to `1.0`",
			"below `0.9`",
			"scripted model endpoints cannot increase confidence",
			"AD-300",
			"`primary_passed`",
			"`primary_failed`",
			"`primary_blocked`",
			"`supporting_only`",
		},
		"docs/design-docs/validation-matrix-gating.md": {
			"Primary outcome contract",
			"`Primary Outcome`",
			"`Primary Pass Gate`",
			"`Primary Status`",
			"`Current Primary Blocker`",
			"`Next Primary Action`",
			"`Supporting Evidence`",
			"`primary_passed`",
			"`primary_failed`",
			"`primary_blocked`",
			"`supporting_only`",
		},
		"docs/design-docs/index.md": {
			"AD-298",
			"AD-300",
			"confidence-gated planning",
			"primary outcome contract",
			"foundation-operating-model.md",
		},
		"docs/roles/personas/foundation-maintainer.md": {
			"confidence-gated planning",
			"Primary Outcome Contract",
			"Assumption Confidence Matrix",
			"Primary Status",
			"decision-complete",
		},
		"docs/validation/README.md": {
			"Primary Outcome Contract",
			"`Primary Outcome`",
			"`Primary Pass Gate`",
			"`Primary Status`",
			"`Current Primary Blocker`",
			"`Next Primary Action`",
			"`Supporting Evidence`",
			"`primary_passed`",
			"`primary_failed`",
			"`primary_blocked`",
			"`supporting_only`",
		},
	}

	for rel, needles := range required {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document confidence-gated planning; missing %q", rel, needle)
			}
		}
	}
}

func TestValidationReportsDeclarePrimaryOutcomeContract(t *testing.T) {
	root := repoRoot(t)
	reportsDir := filepath.Join(root, "docs", "validation", "reports")
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		t.Fatalf("read validation reports: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".md" {
			continue
		}
		path := filepath.Join(reportsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !validationReportNeedsPrimaryContract(name) && !validationReportHasDatedRunOnOrAfter(string(data), "2026-06-16") {
			continue
		}
		checkValidationReportPrimaryContract(t, name, string(data))
	}
}

func checkValidationReportPrimaryContract(t *testing.T, name, text string) {
	t.Helper()
	block, contractIndex, ok := primaryOutcomeContractBlock(text)
	if !ok {
		t.Fatalf("%s must declare ## Primary Outcome Contract", name)
	}
	if firstResultIndex := firstValidationResultSectionIndex(text); firstResultIndex >= 0 && contractIndex > firstResultIndex {
		t.Fatalf("%s must declare Primary Outcome Contract before summary/scope/results language", name)
	}
	requirePrimaryOutcomeFieldsInOrder(t, name, block)
	requireAllowedPrimaryStatus(t, name, block)
}

func validationReportNeedsPrimaryContract(name string) bool {
	if len(name) < len("2006-01-02") {
		return false
	}
	date := name[:len("2006-01-02")]
	if len(date) != 10 || date[4] != '-' || date[7] != '-' {
		return false
	}
	return date >= "2026-06-16"
}

func validationReportHasDatedRunOnOrAfter(text, cutoff string) bool {
	for _, line := range strings.Split(text, "\n") {
		if date, ok := validationReportDateFromLine(line); ok && date >= cutoff {
			return true
		}
	}
	return false
}

func validationReportDateFromLine(line string) (string, bool) {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"**Date:**", "Date:"} {
		if strings.HasPrefix(line, prefix) {
			return leadingDate(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		}
	}
	if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
		return leadingDate(strings.TrimSpace(strings.TrimLeft(line, "# ")))
	}
	return "", false
}

func leadingDate(text string) (string, bool) {
	if len(text) < len("2006-01-02") {
		return "", false
	}
	date := text[:len("2006-01-02")]
	if len(date) == 10 && date[4] == '-' && date[7] == '-' {
		return date, true
	}
	return "", false
}

func firstValidationResultSectionIndex(text string) int {
	first := -1
	for _, marker := range []string{"## Summary", "## Scope", "## Results", "## Completion Claim", "## Supporting Claims", "## Supporting Smoke Results"} {
		idx := strings.Index(text, marker)
		if idx >= 0 && (first < 0 || idx < first) {
			first = idx
		}
	}
	return first
}

func primaryOutcomeContractBlock(text string) (string, int, bool) {
	const heading = "## Primary Outcome Contract"
	start := strings.Index(text, heading)
	if start < 0 {
		return "", -1, false
	}
	rest := text[start+len(heading):]
	end := len(text)
	if next := strings.Index(rest, "\n## "); next >= 0 {
		end = start + len(heading) + next
	}
	return text[start:end], start, true
}

func requirePrimaryOutcomeFieldsInOrder(t *testing.T, name, block string) {
	t.Helper()
	cursor := 0
	for _, field := range []string{
		"**Primary Outcome:**",
		"**Primary Pass Gate:**",
		"**Primary Status:**",
		"**Current Primary Blocker:**",
		"**Next Primary Action:**",
		"**Supporting Evidence:**",
	} {
		idx := strings.Index(block[cursor:], field)
		if idx < 0 {
			t.Fatalf("%s missing primary outcome field %q in contract block", name, field)
		}
		cursor += idx + len(field)
	}
}

func requireAllowedPrimaryStatus(t *testing.T, name, block string) {
	t.Helper()
	var statusLines []string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**Primary Status:**") {
			statusLines = append(statusLines, strings.TrimSpace(strings.TrimPrefix(line, "**Primary Status:**")))
		}
	}
	if len(statusLines) != 1 {
		t.Fatalf("%s must declare exactly one Primary Status in the contract block, got %d", name, len(statusLines))
	}
	for _, status := range []string{"primary_passed", "primary_failed", "primary_blocked", "supporting_only"} {
		if statusLines[0] == "`"+status+"`" {
			return
		}
	}
	t.Fatalf("%s must use one of the allowed Primary Status values, got %q", name, statusLines[0])
}

func TestFeatureContractsDeclareRequiredFields(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "docs", "features"))
	if err != nil {
		t.Fatalf("read features: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || entry.Name() == "README.md" {
			continue
		}
		path := filepath.Join(root, "docs", "features", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		text := string(data)
		for _, label := range []string{
			"Feature ID:",
			"Goals:",
			"Status:",
			"Owner:",
			"Business Logic",
			"Step-By-Step Behavior",
			"Scenario Schedule",
			"Out of Scope",
			"Descoped Scenarios",
			"Evidence",
		} {
			if !strings.Contains(text, label) {
				t.Fatalf("docs/features/%s must declare %s", entry.Name(), label)
			}
		}
		if !strings.Contains(text, "Given") || !strings.Contains(text, "When") || !strings.Contains(text, "Then") {
			t.Fatalf("docs/features/%s must include Markdown Given/When/Then scenarios", entry.Name())
		}
	}
}
