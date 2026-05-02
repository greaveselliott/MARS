package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
