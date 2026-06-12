/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiffStatsAllowsTicketLifecycleMoveDeletion(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	for _, status := range []string{"backlog", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", status), 0o755); err != nil {
			t.Fatalf("mkdir tickets: %v", err)
		}
	}
	backlog := filepath.Join(dir, "docs", "tickets", "backlog", "T-001-ship.md")
	if err := os.WriteFile(backlog, []byte("# Ship\n"), 0o644); err != nil {
		t.Fatalf("write backlog ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "docs/tickets/backlog/T-001-ship.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "tickets: seed backlog"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	done := filepath.Join(dir, "docs", "tickets", "done", "T-001-ship.md")
	if err := os.Rename(backlog, done); err != nil {
		t.Fatalf("move ticket: %v", err)
	}

	stats, err := diffStats(context.Background(), root)
	if err != nil {
		t.Fatalf("diffStats: %v", err)
	}
	if stats.Deletions != 0 {
		t.Fatalf("expected lifecycle move deletion to be ignored, got %d", stats.Deletions)
	}
}

func TestDiffStatsAllowsStagedTicketLifecycleMoveDeletion(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	for _, status := range []string{"backlog", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", status), 0o755); err != nil {
			t.Fatalf("mkdir tickets: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-ship.md"), []byte("# Ship\n"), 0o644); err != nil {
		t.Fatalf("write backlog ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "docs/tickets/backlog/T-001-ship.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "tickets: seed backlog"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "mv", "docs/tickets/backlog/T-001-ship.md", "docs/tickets/done/"); err != nil {
		t.Fatalf("git mv: %v", err)
	}

	stats, err := diffStats(context.Background(), root)
	if err != nil {
		t.Fatalf("diffStats: %v", err)
	}
	if stats.Deletions != 0 {
		t.Fatalf("expected staged lifecycle move deletion to be ignored, got %d", stats.Deletions)
	}
	if err := ValidateRepoDiff(context.Background(), root, Session{}); err != nil {
		t.Fatalf("expected staged lifecycle move to pass safety validation, got %v", err)
	}
}

func TestDiffStatsAllowsTicketLifecycleDuplicateCleanup(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	for _, status := range []string{"backlog", "in-progress"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", status), 0o755); err != nil {
			t.Fatalf("mkdir tickets: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-ship.md"), []byte("# Ship\n"), 0o644); err != nil {
		t.Fatalf("write backlog ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "docs/tickets/backlog/T-001-ship.md"); err != nil {
		t.Fatalf("git add backlog: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "tickets: seed backlog"); err != nil {
		t.Fatalf("git commit backlog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-ship.md"), []byte("# Ship\n"), 0o644); err != nil {
		t.Fatalf("write in-progress ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "docs/tickets/in-progress/T-001-ship.md"); err != nil {
		t.Fatalf("git add in-progress: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "tickets: duplicate claim"); err != nil {
		t.Fatalf("git commit in-progress: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-ship.md")); err != nil {
		t.Fatalf("remove duplicate backlog ticket: %v", err)
	}

	stats, err := diffStats(context.Background(), root)
	if err != nil {
		t.Fatalf("diffStats: %v", err)
	}
	if stats.Deletions != 0 {
		t.Fatalf("expected duplicate lifecycle cleanup deletion to be ignored, got %d", stats.Deletions)
	}
	if err := ValidateRepoDiff(context.Background(), root, Session{}); err != nil {
		t.Fatalf("expected duplicate lifecycle cleanup to pass safety validation, got %v", err)
	}
}

func TestDiffStatsStillCountsArbitraryDeletion(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "README.md")); err != nil {
		t.Fatalf("remove readme: %v", err)
	}

	stats, err := diffStats(context.Background(), root)
	if err != nil {
		t.Fatalf("diffStats: %v", err)
	}
	if stats.Deletions != 1 {
		t.Fatalf("expected arbitrary deletion to count, got %d", stats.Deletions)
	}
}
