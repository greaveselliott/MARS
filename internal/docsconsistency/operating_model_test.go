/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/harness-operating-model.md
- docs/features/F-001-delivery-operating-model.md
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
			"Evidence And Classification",
			"Assumption Confidence Matrix",
			"`Assumption`",
			"`Evidence`",
			"`Confidence`",
			"`Validation Required`",
			"`0.0` to `1.0`",
			"below `0.9`",
			"scripted model endpoints cannot increase confidence",
		},
		"docs/design-docs/index.md": {
			"AD-298",
			"confidence-gated planning",
			"foundation-operating-model.md",
		},
		"docs/roles/personas/foundation-maintainer.md": {
			"confidence-gated planning",
			"Assumption Confidence Matrix",
			"decision-complete",
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
