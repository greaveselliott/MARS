/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package remediation

import (
	"strings"
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
		"optional-tool:install-guidance",
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

func TestPlanKeepsDirtyWorktreeAsOperatorBlocker(t *testing.T) {
	registry := DefaultRegistry()

	plan := registry.Plan(Signal{
		Category: telemetry.CategoryWorkspaceHygiene,
		Message:  "dirty working tree has user changes before run",
	})
	attempt := requireAttempt(t, plan, "dirty-worktree:blocker")
	if attempt.Status != AttemptSkippedOperatorRequired {
		t.Fatalf("expected dirty worktree to require operator confirmation, got %s", attempt.Status)
	}
	if attempt.Safety != SafetyOperatorRequired {
		t.Fatalf("expected operator safety, got %s", attempt.Safety)
	}
	assertContains(t, attempt.Commands, "git status --short")
	assertNotContainsText(t, strings.Join(attempt.Commands, "\n"), "git reset --hard")
	assertNotContainsText(t, strings.Join(attempt.Commands, "\n"), "git checkout --")
	assertNotContainsText(t, strings.Join(attempt.Commands, "\n"), "git clean -")
	assertContainsText(t, attempt.NextAction, "separate user work")
	assertContainsText(t, attempt.NextAction, "operator")
}

func TestPlanRequiresApprovalForDestructiveRecipes(t *testing.T) {
	registry := DefaultRegistry()

	for _, recipe := range registry.List() {
		if !recipe.Destructive {
			continue
		}
		plan := registry.Plan(Signal{
			Category: recipe.Categories[0],
			Message:  recipe.MessageContains[0],
		})
		attempt := requireAttempt(t, plan, recipe.ID)
		if attempt.Status != AttemptSkippedApprovalRequired {
			t.Fatalf("destructive recipe %q status = %s, want %s", recipe.ID, attempt.Status, AttemptSkippedApprovalRequired)
		}
		if attempt.Safety != recipe.Safety {
			t.Fatalf("destructive recipe %q safety = %s, want %s", recipe.ID, attempt.Safety, recipe.Safety)
		}
		assertContainsText(t, attempt.Reason, "approval")
	}
}

func TestDefaultRecipesDoNotOfferDestructiveGitMutations(t *testing.T) {
	registry := DefaultRegistry()
	destructiveGit := []string{"git reset --hard", "git checkout --", "git clean -"}

	for _, recipe := range registry.List() {
		for _, command := range recipe.Commands {
			for _, disallowed := range destructiveGit {
				if strings.Contains(command, disallowed) {
					t.Fatalf("recipe %q command %q includes destructive git mutation %q", recipe.ID, command, disallowed)
				}
			}
		}
	}
}

func TestPlanTreatsMissingOptionalToolsAsGuidance(t *testing.T) {
	registry := DefaultRegistry()

	plan := registry.Plan(Signal{
		Category: telemetry.CategoryUnknown,
		Message:  "llama-server not found in PATH; optional tool not installed for hosted model run",
	})
	attempt := requireAttempt(t, plan, "optional-tool:install-guidance")
	if attempt.Status != AttemptSkippedOperatorRequired {
		t.Fatalf("expected optional-tool guidance to require operator confirmation, got %s", attempt.Status)
	}
	if attempt.Safety != SafetyOperatorRequired {
		t.Fatalf("expected operator safety, got %s", attempt.Safety)
	}
	assertContains(t, attempt.Commands, "mars-harness doctor --repo <repo>")
	assertContainsText(t, attempt.NextAction, "record a skip/blocker")
	assertContainsText(t, attempt.NextAction, "do not mark remediation successful")
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

func requireAttempt(t *testing.T, plan Plan, id string) Attempt {
	t.Helper()
	for _, attempt := range plan.Attempts {
		if attempt.RecipeID == id {
			return attempt
		}
	}
	t.Fatalf("expected attempt %q, got %#v", id, plan.Attempts)
	return Attempt{}
}

func assertContains(t *testing.T, got []string, want string) {
	t.Helper()
	for _, candidate := range got {
		if candidate == want {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, got)
}

func assertContainsText(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}

func assertNotContainsText(t *testing.T, got string, disallowed string) {
	t.Helper()
	if strings.Contains(got, disallowed) {
		t.Fatalf("expected %q not to contain %q", got, disallowed)
	}
}
