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
	"strings"
	"testing"
)

func TestShellExecPolicyBlocksTicketRootMarkdown(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"touch docs/tickets/T-002-root-ticket.md"}`))
	if err == nil {
		t.Fatal("expected root ticket shell_exec to be blocked")
	}
	if !strings.Contains(err.Error(), "ticket markdown must live under docs/tickets/backlog") {
		t.Fatalf("expected lifecycle path policy error, got %v", err)
	}
}

func TestShellExecPolicyAllowsTicketLifecycleMove(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketGitRepo(t)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/backlog/T-001-existing.md docs/tickets/in-progress/"}`))
	if err != nil {
		t.Fatalf("expected lifecycle ticket move to pass, got %v", err)
	}
}

func TestEngineerRepeatedNoopAfterValidationBlocksWithCommitGuidance(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go build -o /tmp/note-stats-validation
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): claim T-001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write dirty implementation: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		shellNoopFailureKey:         1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected repeated no-op to be blocked")
	}
	if !strings.Contains(err.Error(), "repeated shell_exec no-op") ||
		!strings.Contains(err.Error(), "git_commit") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected commit/disposition guidance, got %v", err)
	}
}

func TestEngineerRepeatedNoopBeforeImplementationRedirectsToFileWrite(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
---

# Ship note stats
`)

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		shellNoopFailureKey: 1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected pre-implementation repeated no-op to be blocked")
	}
	if !strings.Contains(err.Error(), "repeated shell_exec no-op before implementation") ||
		!strings.Contains(err.Error(), "file_read") ||
		!strings.Contains(err.Error(), "file_write") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected implementation guidance, got %v", err)
	}
}
