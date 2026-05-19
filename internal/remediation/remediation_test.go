/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package remediation

import (
	"testing"

	"github.com/greaveselliott/mars-harness/internal/telemetry"
)

func TestDefaultRegistryListsKnownRecipes(t *testing.T) {
	registry := DefaultRegistry()
	recipes := registry.List()
	want := []string{
		"dependency:sync-before-repair",
		"dirty-worktree:blocker",
		"doctor:known-remediation",
		"generated-docs:update-missing-defaults",
		"manifest:validate-or-init",
		"model-artifact:checksum-mismatch",
		"scanner:dedupe-duplicate-tickets",
		"stale-ticket:drain-in-progress",
	}
	if len(recipes) != len(want) {
		t.Fatalf("expected %d recipes, got %d", len(want), len(recipes))
	}
	seen := map[string]bool{}
	for i, recipe := range recipes {
		if recipe.ID != want[i] {
			t.Fatalf("recipe %d = %q, want %q", i, recipe.ID, want[i])
		}
		if seen[recipe.ID] {
			t.Fatalf("duplicate recipe ID %q", recipe.ID)
		}
		seen[recipe.ID] = true
		if recipe.Title == "" || recipe.Summary == "" || recipe.Target == "" || recipe.NextAction == "" {
			t.Fatalf("recipe %q must describe title, summary, target, and next action", recipe.ID)
		}
	}
}

func TestApplicableRecipesMatchByCategoryAndMessage(t *testing.T) {
	registry := DefaultRegistry()

	workspace := registry.Applicable(Signal{
		Category: telemetry.CategoryWorkspaceHygiene,
		Message:  "workspace_hygiene_blocked: generated node_modules is dirty",
	})
	assertRecipeIDs(t, workspace, "dependency:sync-before-repair", "dirty-worktree:blocker")

	stale := registry.Applicable(Signal{
		Category: telemetry.CategoryUnknown,
		Message:  "stale in-progress ticket docs/tickets/in-progress/T-001.md",
	})
	assertRecipeIDs(t, stale, "stale-ticket:drain-in-progress")
}

func TestPlanSkipsUnsafeRecipesAndMarksAutoSafeRecipesReady(t *testing.T) {
	registry := DefaultRegistry()

	plan := registry.Plan(Signal{
		Category: telemetry.CategoryUnknown,
		Message:  "missing generated docs and operating-model drift",
	})
	if len(plan.Attempts) != 1 {
		t.Fatalf("expected one generated-docs attempt, got %#v", plan.Attempts)
	}
	attempt := plan.Attempts[0]
	if attempt.RecipeID != "generated-docs:update-missing-defaults" {
		t.Fatalf("unexpected recipe %q", attempt.RecipeID)
	}
	if attempt.Status != AttemptReady {
		t.Fatalf("expected auto-safe recipe ready, got %s", attempt.Status)
	}

	plan = registry.Plan(Signal{
		Category: telemetry.CategoryModelUnavailable,
		Message:  "models download: checksum mismatch for qwen.gguf",
	})
	if len(plan.Attempts) != 1 {
		t.Fatalf("expected one checksum attempt, got %#v", plan.Attempts)
	}
	attempt = plan.Attempts[0]
	if attempt.Status != AttemptSkippedApprovalRequired {
		t.Fatalf("expected checksum repair to require approval, got %s", attempt.Status)
	}
	if attempt.Safety != SafetyApprovalRequired {
		t.Fatalf("expected approval safety, got %s", attempt.Safety)
	}
}

func assertRecipeIDs(t *testing.T, recipes []Recipe, want ...string) {
	t.Helper()
	if len(recipes) != len(want) {
		t.Fatalf("expected recipes %v, got %#v", want, recipes)
	}
	for i, recipe := range recipes {
		if recipe.ID != want[i] {
			t.Fatalf("recipe %d = %q, want %q", i, recipe.ID, want[i])
		}
	}
}
