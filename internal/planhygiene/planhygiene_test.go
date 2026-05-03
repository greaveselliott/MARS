package planhygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckRepoValidPlanHygienePasses(t *testing.T) {
	t.Parallel()
	repo := validRepo(t)
	writeFile(t, repo, "docs/exec-plans/completed/.gitkeep", "")

	report, err := Check(Config{RepoPath: repo, Now: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected valid repo, got %+v", report.Issues)
	}
}

func TestCheckRepoDetectsMultipleActiveExecPlans(t *testing.T) {
	t.Parallel()
	repo := validRepo(t)
	writeFile(t, repo, "docs/exec-plans/active/second-plan.md", activePlan("MH-002", "2026-05-03"))

	report := mustCheck(t, repo)
	assertIssueContains(t, report, "docs/exec-plans/active must contain exactly one")
	assertIssueContains(t, report, "exactly one exec plan must declare **Status:** Active")
}

func TestCheckRepoDetectsBacklogPlanWithoutPriority(t *testing.T) {
	t.Parallel()
	repo := validRepo(t)
	writeFile(t, repo, "docs/exec-plans/backlog/missing-priority.md", strings.Join([]string{
		"# Waiting Plan",
		"",
		"**Status:** Backlog",
		"**Depends On:** None",
	}, "\n"))

	report := mustCheck(t, repo)
	assertIssueContains(t, report, "backlog exec plans must declare **Priority:** P0-P4")
}

func TestCheckRepoDetectsSupersededPlanWithoutPointer(t *testing.T) {
	t.Parallel()
	repo := validRepo(t)
	writeFile(t, repo, "docs/exec-plans/superseded/old-plan.md", strings.Join([]string{
		"# Old Plan",
		"",
		"**Status:** Superseded",
		"",
		"This is retained for history.",
	}, "\n"))

	report := mustCheck(t, repo)
	assertIssueContains(t, report, "superseded exec plans must point to the current active plan")
}

func TestCheckRepoDetectsStaleActivePlanStatusClaims(t *testing.T) {
	t.Parallel()
	repo := validRepo(t)
	writeFile(t, repo, "docs/exec-plans/active/current-operating-plan.md", strings.Join([]string{
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
		"- Next owner: TBD",
	}, "\n"))

	report, err := Check(Config{
		RepoPath: repo,
		Now:      time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	assertIssueContains(t, report, "active plan lists MH-001 as backlog but ticket is done")
	assertIssueContains(t, report, "active plan uses relative status language without an absolute date")
	assertIssueContains(t, report, "active plan verification note from 2026-01-01 is stale")
	assertIssueContains(t, report, "active plan contains unresolved TBD placeholder")
}

func validRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, repo, "docs/exec-plans/active/current-operating-plan.md", activePlan("MH-002", "2026-05-03"))
	writeFile(t, repo, "docs/exec-plans/backlog/waiting-plan.md", strings.Join([]string{
		"# Waiting Plan",
		"",
		"**Status:** Backlog",
		"**Priority:** P1",
		"**Depends On:** None",
		"**Blocks:** Nothing",
		"**Related Tickets:** MH-002",
	}, "\n"))
	writeFile(t, repo, "docs/exec-plans/superseded/old-plan.md", strings.Join([]string{
		"# Old Plan",
		"",
		"**Status:** Superseded pending reconciliation",
		"",
		"> Current status lives in [current-operating-plan.md](../active/current-operating-plan.md).",
	}, "\n"))
	writeFile(t, repo, "docs/tickets/done/MH-001-done.md", "# done\n")
	writeFile(t, repo, "docs/tickets/backlog/MH-002-backlog.md", "# backlog\n")
	return repo
}

func activePlan(backlogTicket, evidenceDate string) string {
	return strings.Join([]string{
		"# Current Operating Plan",
		"",
		"**Status:** Active",
		"**Priority:** P0",
		"",
		"## Current Truth",
		"",
		"- `docs/tickets/backlog/` contains " + backlogTicket + ".",
		"- Verification evidence recorded on " + evidenceDate + ".",
	}, "\n")
}

func mustCheck(t *testing.T, repo string) Report {
	t.Helper()
	report, err := Check(Config{
		RepoPath: repo,
		Now:      time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return report
}

func assertIssueContains(t *testing.T, report Report, want string) {
	t.Helper()
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, want) {
			if strings.TrimSpace(issue.Fix) == "" {
				t.Fatalf("issue %q has no remediation: %+v", want, issue)
			}
			return
		}
	}
	t.Fatalf("expected issue containing %q, got %+v", want, report.Issues)
}

func writeFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
