/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/planhygiene"
)

func TestActivePlanHygiene(t *testing.T) {
	root := repoRoot(t)
	report, err := planhygiene.CheckRepo(root)
	if err != nil {
		t.Fatalf("active-plan hygiene check: %v", err)
	}
	if !report.OK() {
		var details []string
		for _, issue := range report.Issues {
			details = append(details, issue.Path+": "+issue.Message+"; fix: "+issue.Fix)
		}
		t.Fatalf("active-plan hygiene failed:\n%s", strings.Join(details, "\n"))
	}
}

func TestActivePlanHygieneDetectsStaleFixture(t *testing.T) {
	root := t.TempDir()
	writePlanHygieneFile(t, root, "docs/exec-plans/active/current-operating-plan.md", strings.Join([]string{
		"# Current Operating Plan",
		"",
		"**Status:** Active",
		"**Priority:** P0",
		"",
		"## Current Truth",
		"",
		"- `docs/tickets/backlog/` contains MH-001.",
		"- Latest checks observed during review.",
		"- Verification evidence was recorded on 2026-01-01.",
		"- Owner: TBD",
	}, "\n"))
	writePlanHygieneFile(t, root, "docs/tickets/done/MH-001-done.md", "# done\n")

	report, err := planhygiene.Check(planhygiene.Config{
		RepoPath: root,
		Now:      time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("active-plan hygiene check: %v", err)
	}
	assertPlanHygieneIssue(t, report, "active plan lists MH-001 as backlog but ticket is done")
	assertPlanHygieneIssue(t, report, "active plan uses relative status language without an absolute date")
	assertPlanHygieneIssue(t, report, "active plan verification note from 2026-01-01 is stale")
	assertPlanHygieneIssue(t, report, "active plan contains unresolved TBD placeholder")
}

func TestSingleActiveExecPlan(t *testing.T) {
	root := repoRoot(t)
	activeDir := filepath.Join(root, "docs", "exec-plans", "active")
	entries, err := os.ReadDir(activeDir)
	if err != nil {
		t.Fatalf("read active exec plan dir: %v", err)
	}

	var activePlans []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		activePlans = append(activePlans, entry.Name())
	}
	if len(activePlans) != 1 {
		t.Fatalf("docs/exec-plans/active must contain exactly one markdown exec plan, got %v", activePlans)
	}

	activePath := filepath.Join(activeDir, activePlans[0])
	activeData, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read active exec plan: %v", err)
	}
	activeText := string(activeData)
	if !strings.Contains(activeText, "**Status:** Active") {
		t.Fatalf("%s must declare **Status:** Active", filepath.ToSlash(filepath.Join("docs/exec-plans/active", activePlans[0])))
	}
	if !strings.Contains(activeText, "**Priority:**") {
		t.Fatalf("%s must declare **Priority:**", filepath.ToSlash(filepath.Join("docs/exec-plans/active", activePlans[0])))
	}
	requirePlanDependencyMetadata(t, filepath.ToSlash(filepath.Join("docs/exec-plans/active", activePlans[0])), activeText)
	requirePlanOperatingModelMetadata(t, filepath.ToSlash(filepath.Join("docs/exec-plans/active", activePlans[0])), activeText)

	execRoot := filepath.Join(root, "docs", "exec-plans")
	err = filepath.WalkDir(execRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		if path == activePath {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "**Status:** Active") {
			rel, _ := filepath.Rel(root, path)
			t.Fatalf("%s declares active status; only %s may be active", filepath.ToSlash(rel), filepath.ToSlash(filepath.Join("docs/exec-plans/active", activePlans[0])))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk exec plans: %v", err)
	}
}

func TestBacklogExecPlansHavePriority(t *testing.T) {
	root := repoRoot(t)
	backlogDir := filepath.Join(root, "docs", "exec-plans", "backlog")
	entries, err := os.ReadDir(backlogDir)
	if err != nil {
		t.Fatalf("read backlog exec plan dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(backlogDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		text := string(data)
		if !strings.Contains(text, "**Status:** Backlog") {
			t.Fatalf("docs/exec-plans/backlog/%s must declare **Status:** Backlog", entry.Name())
		}
		if !strings.Contains(text, "**Priority:**") {
			t.Fatalf("docs/exec-plans/backlog/%s must declare **Priority:**", entry.Name())
		}
		requirePlanDependencyMetadata(t, filepath.ToSlash(filepath.Join("docs/exec-plans/backlog", entry.Name())), text)
		requirePlanOperatingModelMetadata(t, filepath.ToSlash(filepath.Join("docs/exec-plans/backlog", entry.Name())), text)
	}
}

func requirePlanDependencyMetadata(t *testing.T, rel, text string) {
	t.Helper()
	for _, label := range []string{"**Depends On:**", "**Blocks:**", "**Related Tickets:**"} {
		if !strings.Contains(text, label) {
			t.Fatalf("%s must declare %s metadata", rel, label)
		}
	}
}

func requirePlanOperatingModelMetadata(t *testing.T, rel, text string) {
	t.Helper()
	for _, label := range []string{
		"**Goals:**",
		"**BDD Feature:**",
		"**Hypothesis:**",
		"**Success Evidence:**",
		"**Falsification Evidence:**",
		"**Scenario Schedule:**",
		"**Current Failing Scenario:**",
		"**Walking Skeleton Slice:**",
		"**Learning Or MVP Outcome:**",
	} {
		if !strings.Contains(text, label) {
			t.Fatalf("%s must declare %s operating-model metadata", rel, label)
		}
	}
}

func writePlanHygieneFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertPlanHygieneIssue(t *testing.T, report planhygiene.Report, want string) {
	t.Helper()
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, want) {
			if strings.TrimSpace(issue.Fix) == "" {
				t.Fatalf("issue %q has no remediation", want)
			}
			return
		}
	}
	t.Fatalf("expected active-plan hygiene issue containing %q, got %+v", want, report.Issues)
}
