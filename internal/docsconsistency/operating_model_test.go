package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAD074OperatingModelArtifactsExist(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"docs/design-docs/delivery-operating-model.md":    {"AD-074", "BDD-Led Goal-Driven Walking-Skeleton"},
		"docs/design-docs/harness-operating-model.md":     {"AD-084", "Planner", "End-to-End Tester", "`domain`", "`mode`"},
		"docs/goals/README.md":                            {"Goal Schema", "Autonomous Goal Rule", "Dedupe Key"},
		"docs/goals/active.md":                            {"G-001", "Status: active", "Hypothesis"},
		"docs/goals/observations.md":                      {"weak/noisy evidence"},
		"docs/goals/superseded.md":                        {"Superseded Goals"},
		"docs/features/README.md":                         {"BDD Feature Contracts", "Given/When/Then", "Scenario Schedule"},
		"docs/features/F-001-delivery-operating-model.md": {"Feature ID: F-001", "Scenario Schedule", "Given", "When", "Then"},
		"docs/tickets/README.md":                          {"work_type", "bdd_scenarios", "end_to_end_evidence", "verified_by"},
		"docs/QUALITY_SCORE.md":                           {"shipped feature scenarios", "enabler work"},
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
