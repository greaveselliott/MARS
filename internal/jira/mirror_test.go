/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
*/
package jira

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/integrations"
)

func TestMirrorIssueDisabledWithoutBoardDrivenConfig(t *testing.T) {
	repoRoot := t.TempDir()
	result, err := MirrorIssue(context.Background(), []Repository{{
		ID:     "repo-1",
		Path:   repoRoot,
		Config: integrations.Defaults(),
	}}, Issue{Key: "DEMO-1", Project: "DEMO", Summary: "Do the thing"})
	if err != nil {
		t.Fatalf("MirrorIssue: %v", err)
	}
	if result.Status != StatusDisabled {
		t.Fatalf("expected disabled result, got %#v", result)
	}
	if result.LLMJobsEnqueued != 0 {
		t.Fatalf("jira ingestion must not enqueue LLM jobs, got %d", result.LLMJobsEnqueued)
	}
	assertNoMarkdownTickets(t, repoRoot)
}

func TestMirrorIssueDropsUnmappedAndAmbiguousProjects(t *testing.T) {
	repoRoot := t.TempDir()
	baseCfg := boardDrivenConfig(filepath.Base(repoRoot))
	tests := []struct {
		name   string
		cfg    integrations.Config
		issue  Issue
		reason string
	}{
		{
			name:   "unmapped project",
			cfg:    baseCfg,
			issue:  Issue{Key: "OTHER-1", Project: "OTHER", Summary: "Wrong project"},
			reason: "unmapped_project",
		},
		{
			name: "unmapped repo",
			cfg: func() integrations.Config {
				cfg := baseCfg
				cfg.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "DEMO", Repo: "other-repo"}}
				return cfg
			}(),
			issue:  Issue{Key: "DEMO-2", Project: "DEMO", Summary: "Wrong repo"},
			reason: "unmapped_repo",
		},
		{
			name: "ambiguous duplicate mapping",
			cfg: func() integrations.Config {
				cfg := baseCfg
				cfg.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{
					{Project: "DEMO", Repo: filepath.Base(repoRoot)},
					{Project: "DEMO", Repo: filepath.Base(repoRoot)},
				}
				return cfg
			}(),
			issue:  Issue{Key: "DEMO-3", Project: "DEMO", Summary: "Duplicate map"},
			reason: "ambiguous_project_repo_map",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: tt.cfg}}, tt.issue)
			if err != nil {
				t.Fatalf("MirrorIssue: %v", err)
			}
			if result.Status != StatusDropped || result.Reason != tt.reason {
				t.Fatalf("expected dropped/%s, got %#v", tt.reason, result)
			}
			if result.LLMJobsEnqueued != 0 {
				t.Fatalf("jira ingestion must not enqueue LLM jobs, got %d", result.LLMJobsEnqueued)
			}
			assertNoMarkdownTickets(t, repoRoot)
		})
	}
}

func TestMirrorIssueCreatesAndReconcilesSingleBacklogTicket(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	issue := Issue{
		Key:          "DEMO-10",
		Project:      "DEMO",
		Summary:      "Mirror checkout requirement",
		Description:  "Customers can complete checkout.",
		Created:      "2026-06-22T09:00:00.000+0000",
		Updated:      "2026-06-23T09:00:00.000+0000",
		Priority:     "P1",
		Sprint:       "Sprint 42",
		SprintActive: true,
		Rank:         "0|i0002f:",
		Status:       "Ready for Dev",
		Epic:         "DEMO-EPIC",
		BlockedBy:    []string{"DEMO-1"},
	}
	result, err := MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, issue)
	if err != nil {
		t.Fatalf("MirrorIssue create: %v", err)
	}
	if result.Status != StatusCreated || !result.Created {
		t.Fatalf("expected created, got %#v", result)
	}
	if result.LLMJobsEnqueued != 0 {
		t.Fatalf("jira ingestion must not enqueue LLM jobs, got %d", result.LLMJobsEnqueued)
	}
	ticketPath := filepath.Join(repoRoot, filepath.FromSlash(result.TicketPath))
	content := readFile(t, ticketPath)
	for _, want := range []string{
		`jira_key: "DEMO-10"`,
		`jira_updated: "2026-06-23T09:00:00.000+0000"`,
		`jira_created: "2026-06-22T09:00:00.000+0000"`,
		`priority: "P1"`,
		`sprint: "Sprint 42"`,
		`sprint_active: true`,
		`rank: "0|i0002f:"`,
		`jira_status: "Ready for Dev"`,
		`blocked_by: ["DEMO-1"]`,
		`epic: "DEMO-EPIC"`,
		"Customers can complete checkout.",
		scopedMarker,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("ticket missing %q:\n%s", want, content)
		}
	}

	updated := issue
	updated.Summary = "Mirror changed checkout requirement"
	updated.Description = "Customers can complete checkout with saved cards."
	updated.Priority = "P2"
	updated.Updated = "2026-06-23T10:00:00.000+0000"
	result, err = MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, updated)
	if err != nil {
		t.Fatalf("MirrorIssue reconcile: %v", err)
	}
	if result.Status != StatusReconciled || result.TicketPath == "" {
		t.Fatalf("expected reconciled, got %#v", result)
	}
	if result.TicketPath != filepath.ToSlash(filepath.Join("docs", "tickets", "backlog", filepath.Base(ticketPath))) {
		t.Fatalf("expected same ticket path, got %s", result.TicketPath)
	}
	if got := countMarkdownTickets(t, repoRoot); got != 1 {
		t.Fatalf("expected exactly one mirrored ticket, got %d", got)
	}
	content = readFile(t, ticketPath)
	for _, want := range []string{
		`priority: "P2"`,
		`jira_updated: "2026-06-23T10:00:00.000+0000"`,
		"Mirror changed checkout requirement",
		"Customers can complete checkout with saved cards.",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("reconciled ticket missing %q:\n%s", want, content)
		}
	}
}

func TestMirrorIssueRequiresConfiguredWorkspaceAndLabels(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	cfg.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "DEMO", Repo: filepath.Base(repoRoot)}}
	cfg.Ingestion.JIRA.BaseURL = "https://jira.example"
	cfg.Ingestion.JIRA.Scope.AllowedWorkspaces = []string{
		"https://jira.example/jira/software/c/projects/DEMO/boards/42/backlog",
	}
	cfg.Ingestion.JIRA.Scope.RequiredLabels = []string{"allowed-intake"}

	baseIssue := Issue{
		Key:         "DEMO-1",
		Project:     "DEMO",
		URL:         "https://jira.example/rest/api/3/issue/DEMO-1",
		Summary:     "Contained board opportunity",
		Description: "Only this labelled workspace issue can mirror.",
		Priority:    "P1",
		Status:      "Ready for Dev",
	}

	missingLabel := baseIssue
	missingLabel.Labels = []string{"customer-support"}
	result, err := MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, missingLabel)
	if err != nil {
		t.Fatalf("MirrorIssue missing label: %v", err)
	}
	if result.Status != StatusDropped || result.Reason != "scope_required_label_missing" {
		t.Fatalf("expected required-label drop, got %#v", result)
	}
	assertNoMarkdownTickets(t, repoRoot)

	wrongWorkspace := baseIssue
	wrongWorkspace.URL = "https://other.example/rest/api/3/issue/DEMO-1"
	wrongWorkspace.Labels = []string{"allowed-intake"}
	result, err = MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, wrongWorkspace)
	if err != nil {
		t.Fatalf("MirrorIssue wrong workspace: %v", err)
	}
	if result.Status != StatusDropped || result.Reason != "scope_workspace_mismatch" {
		t.Fatalf("expected workspace drop, got %#v", result)
	}
	assertNoMarkdownTickets(t, repoRoot)

	allowed := baseIssue
	allowed.Labels = []string{"allowed-intake", "product"}
	result, err = MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, allowed)
	if err != nil {
		t.Fatalf("MirrorIssue allowed: %v", err)
	}
	if result.Status != StatusCreated {
		t.Fatalf("expected scoped issue to create, got %#v", result)
	}
	ticket := onlyTicketContent(t, repoRoot)
	for _, want := range []string{`jira_key: "DEMO-1"`, "allowed-intake", "Only this labelled workspace issue can mirror."} {
		if !strings.Contains(ticket, want) {
			t.Fatalf("scoped ticket missing %q:\n%s", want, ticket)
		}
	}
}

func TestReconcilePreservesLifecycleEvidenceScopedMarkerAndAgentNotes(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	doneDir := filepath.Join(repoRoot, "docs", "tickets", "done")
	if err := os.MkdirAll(doneDir, 0o755); err != nil {
		t.Fatalf("mkdir done: %v", err)
	}
	ticketPath := filepath.Join(doneDir, "T-777-existing-mirror.md")
	original := `---
id: T-777
title: "Existing mirrored ticket"
priority: "P1"
complexity: medium
work_type: feature
bdd_scenarios: ["F-000-S001"]
end_to_end_evidence: required
evidence_links: ["keep-this-link"]
verified_by: "QA"
owner: "engineer"
last_attempt: "2026-06-22"
blocker: "none"
blocked_by: ["DEMO-1"]
trace_id: "trace-123"
next_action: "Do not overwrite"
jira_key: "DEMO-777"
jira_updated: "old"
jira_created: "old"
sprint: "Old sprint"
sprint_active: false
rank: "old-rank"
jira_status: "To Do"
epic: "OLD-EPIC"
source: jira
created: 2026-06-22
depends_on: []
---

# T-777: Existing mirrored ticket

## JIRA Source

<!-- mars-harness:jira-owned:start -->
old requirements
<!-- mars-harness:jira-owned:end -->

## Harness Scope

<!-- mars-harness:scoped-marker -->
SCOPED BLOCK MUST STAY BYTE IDENTICAL

## Agent Notes

- Engineer note with exact spacing.
- QA note with evidence.
`
	if err := os.WriteFile(ticketPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}
	ownedBefore := harnessOwnedSlice(original)
	evidenceBefore := frontmatterLines(original, []string{
		"end_to_end_evidence",
		"evidence_links",
		"verified_by",
		"owner",
		"last_attempt",
		"trace_id",
		"next_action",
	})

	result, err := MirrorIssue(context.Background(), []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, Issue{
		Key:          "DEMO-777",
		Project:      "DEMO",
		Summary:      "Updated mirrored ticket",
		Description:  "New requirements from JIRA.",
		Created:      "2026-06-20T09:00:00.000+0000",
		Updated:      "2026-06-23T11:00:00.000+0000",
		Priority:     "P3",
		Sprint:       "Sprint 99",
		SprintActive: true,
		Rank:         "new-rank",
		Status:       "Ready for Dev",
		Epic:         "DEMO-EPIC",
	})
	if err != nil {
		t.Fatalf("MirrorIssue reconcile: %v", err)
	}
	if result.Status != StatusReconciled {
		t.Fatalf("expected reconciled, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "docs", "tickets", "backlog", "T-777-existing-mirror.md")); !os.IsNotExist(err) {
		t.Fatalf("expected lifecycle directory to remain done")
	}
	updated := readFile(t, ticketPath)
	if got := harnessOwnedSlice(updated); got != ownedBefore {
		t.Fatalf("harness-owned body changed\nbefore:\n%s\nafter:\n%s", ownedBefore, got)
	}
	if got := frontmatterLines(updated, []string{
		"end_to_end_evidence",
		"evidence_links",
		"verified_by",
		"owner",
		"last_attempt",
		"trace_id",
		"next_action",
	}); got != evidenceBefore {
		t.Fatalf("harness-owned frontmatter changed\nbefore:\n%s\nafter:\n%s", evidenceBefore, got)
	}
	for _, want := range []string{
		`priority: "P3"`,
		`jira_updated: "2026-06-23T11:00:00.000+0000"`,
		`jira_status: "Ready for Dev"`,
		"New requirements from JIRA.",
	} {
		if !strings.Contains(updated, want) {
			t.Fatalf("updated ticket missing %q:\n%s", want, updated)
		}
	}
}

func boardDrivenConfig(repoName string) integrations.Config {
	cfg := integrations.Defaults()
	cfg.FlowProfile = integrations.FlowProfileBoardDriven
	cfg.Ingestion.JIRA.Enabled = true
	cfg.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "DEMO", Repo: repoName}}
	cfg.Ingestion.JIRA.WebhookSecretEnv = "JIRA_WEBHOOK_SECRET"
	cfg.Ingestion.JIRA.Fields.Sprint = "customfield_sprint"
	cfg.Ingestion.JIRA.Fields.Rank = "customfield_rank"
	cfg.Ingestion.JIRA.Fields.EpicLink = "customfield_epic"
	return cfg
}

func assertNoMarkdownTickets(t *testing.T, repoRoot string) {
	t.Helper()
	if got := countMarkdownTickets(t, repoRoot); got != 0 {
		t.Fatalf("expected no markdown tickets, got %d", got)
	}
}

func countMarkdownTickets(t *testing.T, repoRoot string) int {
	t.Helper()
	count := 0
	root := filepath.Join(repoRoot, "docs", "tickets")
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) == ".md" {
			count++
		}
		return nil
	})
	return count
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func harnessOwnedSlice(content string) string {
	idx := strings.Index(content, "## Harness Scope")
	if idx < 0 {
		return ""
	}
	return content[idx:]
}

func frontmatterLines(content string, keys []string) string {
	keySet := map[string]bool{}
	for _, key := range keys {
		keySet[key] = true
	}
	var out []string
	for _, line := range strings.Split(content, "\n") {
		key, _, ok := strings.Cut(line, ":")
		if ok && keySet[strings.TrimSpace(key)] {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
