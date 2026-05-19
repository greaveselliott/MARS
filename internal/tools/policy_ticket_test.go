/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTicketCreatePolicyBlocksEngineerOrdinaryBacklogDuringEligibleInProgress(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyPlan(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
blocker: none
blocked_by: []
---

# Active
`)

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "ticket_create", []byte(`{"title":"New backlog","priority":"high","work_type":"enabler","body":"## Context\nx"}`))
	if err == nil {
		t.Fatal("expected ordinary ticket creation to be blocked")
	}
	if !strings.Contains(err.Error(), "eligible in-progress tickets remain") {
		t.Fatalf("expected in-progress policy error, got %v", err)
	}
}

func TestTicketCreatePolicyAllowsEngineerLinkedDependencyTicket(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyPlan(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
blocker: none
blocked_by: []
---

# Active
`)

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "ticket_create", []byte(`{"title":"Install missing SDK","priority":"high","work_type":"enabler","dedupe_key":"dep:T-001:sdk","metadata":{"blocks":"T-001"},"body":"## Context\nx"}`))
	if err != nil {
		t.Fatalf("expected linked dependency ticket to pass, got %v", err)
	}
}

func TestTicketCreatePolicyRequiresPlanBeforeTickets(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)

	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "ticket_create", []byte(`{"title":"Implement checkout","priority":"high","work_type":"enabler","body":"## Context\nx"}`))
	if err == nil {
		t.Fatal("expected ticket creation to require active exec plan")
	}
	if !strings.Contains(err.Error(), "exec plan, feature contract, ticket, delivery") {
		t.Fatalf("expected planning order error, got %v", err)
	}
}

func TestTicketCreatePolicyRequiresFeatureContractBeforeFeatureTicket(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyPlan(t, dir)

	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})
	raw := []byte(`{"title":"Implement checkout","priority":"high","work_type":"feature","bdd_scenarios":["F-002-S001"],"body":"## Context\nx"}`)
	err := preToolPolicy(ctx, root, "ticket_create", raw)
	if err == nil {
		t.Fatal("expected feature ticket creation to require feature contract")
	}
	if !strings.Contains(err.Error(), "docs/features/F-002") {
		t.Fatalf("expected feature contract path error, got %v", err)
	}

	writePolicyFeature(t, dir, "F-002-checkout.md")
	if err := preToolPolicy(ctx, root, "ticket_create", raw); err != nil {
		t.Fatalf("expected feature ticket to pass after feature contract exists, got %v", err)
	}
}

func TestTicketCreatePolicyCapsDogfoodBySeverityAndDedupe(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := Session{Role: "dogfood", ToolCounts: map[string]int{}}
	ctx := WithSession(context.Background(), session)

	first := []byte(`{"title":"Finding one","priority":"high","kind":"intervention-debt","dedupe_key":"dogfood:repo:role:target:one","body":"## Context\nx"}`)
	if err := preToolPolicy(ctx, root, "ticket_create", first); err != nil {
		t.Fatalf("first dogfood ticket should pass: %v", err)
	}
	if err := preToolPolicy(ctx, root, "ticket_create", first); err == nil || !strings.Contains(err.Error(), "repeated dedupe key") {
		t.Fatalf("expected repeated dedupe key block, got %v", err)
	}

	second := []byte(`{"title":"Finding two","priority":"high","kind":"intervention-debt","dedupe_key":"dogfood:repo:role:target:two","body":"## Context\nx"}`)
	third := []byte(`{"title":"Finding three","priority":"high","kind":"intervention-debt","dedupe_key":"dogfood:repo:role:other:three","body":"## Context\nx"}`)
	fourth := []byte(`{"title":"Finding four","priority":"high","kind":"intervention-debt","dedupe_key":"dogfood:repo:role:other:four","body":"## Context\nx"}`)
	if err := preToolPolicy(ctx, root, "ticket_create", second); err != nil {
		t.Fatalf("second dogfood ticket should pass: %v", err)
	}
	if err := preToolPolicy(ctx, root, "ticket_create", third); err != nil {
		t.Fatalf("third dogfood ticket should pass: %v", err)
	}
	if err := preToolPolicy(ctx, root, "ticket_create", fourth); err == nil || !strings.Contains(err.Error(), "high-severity") {
		t.Fatalf("expected severity cap block, got %v", err)
	}
}

func TestFileWritePolicyBlocksTicketRootMarkdown(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)

	err := preToolPolicy(context.Background(), root, "file_write", []byte(`{"path":"docs/tickets/T-002-root-ticket.md","content":"# Bad\n"}`))
	if err == nil {
		t.Fatal("expected root ticket file_write to be blocked")
	}
	if !strings.Contains(err.Error(), "ticket markdown must live under docs/tickets/backlog") {
		t.Fatalf("expected lifecycle path policy error, got %v", err)
	}
}

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
	_, root := setupPolicyTicketRepo(t)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/backlog/T-001-existing.md docs/tickets/in-progress/"}`))
	if err != nil {
		t.Fatalf("expected lifecycle ticket move to pass, got %v", err)
	}
}

func TestShellExecPolicyBlocksFeatureTicketDoneMoveWithoutEvidence(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---
# Ship
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/in-progress/T-001-ship.md docs/tickets/done/"}`))
	if err == nil {
		t.Fatal("expected missing feature evidence to block done move")
	}
	if !strings.Contains(err.Error(), "cannot move to docs/tickets/done without BDD scenario evidence") {
		t.Fatalf("expected evidence policy error, got %v", err)
	}
	if !strings.Contains(err.Error(), "evidence_links") || !strings.Contains(err.Error(), "verified_by") {
		t.Fatalf("expected missing evidence fields in error, got %v", err)
	}
}

func TestShellExecPolicyAllowsFeatureTicketDoneMoveWithEvidence(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["game.test.js", "python -m http.server 8080"]
verified_by: engineer
---
# Ship
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/in-progress/T-001-ship.md docs/tickets/done/"}`))
	if err != nil {
		t.Fatalf("expected evidenced feature ticket done move to pass, got %v", err)
	}
}

func TestShellExecPolicyAllowsFeatureTicketDoneMoveWithMultilineEvidence(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links:
  - "public/index.html"
  - "npm run start"
verified_by: engineer
---
# Ship
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/in-progress/T-001-ship.md docs/tickets/done/"}`))
	if err != nil {
		t.Fatalf("expected multiline evidenced feature ticket done move to pass, got %v", err)
	}
}

func TestShellExecPolicyBlocksFeatureTicketDoneCopy(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "backlog", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["public/index.html"]
verified_by: engineer
---
# Ship
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"cp docs/tickets/backlog/T-001-ship.md docs/tickets/done/"}`))
	if err == nil {
		t.Fatal("expected done copy to be blocked")
	}
	if !strings.Contains(err.Error(), "cannot be copied into docs/tickets/done") {
		t.Fatalf("expected copy-to-done error, got %v", err)
	}
}

func TestFileWritePolicyBlocksDoneFeatureTicketWithoutEvidence(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "done", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["game.test.js"]
verified_by: engineer
---
# Ship
`)

	raw := []byte(`{"path":"docs/tickets/done/T-001-ship.md","content":"---\nid: T-001\nwork_type: feature\nbdd_scenarios: [\"F-001-S001\"]\nend_to_end_evidence: required\nevidence_links: []\nverified_by: TBD\n---\n# Ship\n"}`)
	err := preToolPolicy(context.Background(), root, "file_write", raw)
	if err == nil {
		t.Fatal("expected missing done-ticket evidence to block file_write")
	}
	if !strings.Contains(err.Error(), "cannot be saved in docs/tickets/done without BDD scenario evidence") {
		t.Fatalf("expected done ticket evidence error, got %v", err)
	}
}

func TestFileWritePolicyBlocksDoneFeatureTicketDuplicateLifecycleCopy(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	content := `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["public/index.html"]
verified_by: engineer
---
# Ship
`
	writePolicyTicket(t, dir, "backlog", "T-001-ship.md", content)
	writePolicyTicket(t, dir, "done", "T-001-ship.md", content)

	raw, err := json.Marshal(map[string]string{
		"path":    "docs/tickets/done/T-001-ship.md",
		"content": content,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = preToolPolicy(context.Background(), root, "file_write", raw)
	if err == nil {
		t.Fatal("expected duplicate lifecycle copy to be blocked")
	}
	if !strings.Contains(err.Error(), "same ticket still exists") {
		t.Fatalf("expected duplicate lifecycle error, got %v", err)
	}
}

func TestFileWritePolicyBlocksDuplicateFeatureScenarioHeadings(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)

	content := "# F-001\n\n### F-001-S001: First\n\nGiven one\n\n### F-001-S001: Duplicate\n\nGiven two\n"
	raw, err := json.Marshal(map[string]string{
		"path":    "docs/features/F-001-product-walking-skeleton.md",
		"content": content,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = preToolPolicy(context.Background(), root, "file_write", raw)
	if err == nil {
		t.Fatal("expected duplicate scenario headings to be blocked")
	}
	if !strings.Contains(err.Error(), "duplicate scenario ID") {
		t.Fatalf("expected duplicate scenario ID error, got %v", err)
	}
	if !strings.Contains(err.Error(), "heading lines 3, 7") {
		t.Fatalf("expected duplicate heading line numbers, got %v", err)
	}
	if !strings.Contains(err.Error(), "Scenario Schedule list entries may repeat the ID") {
		t.Fatalf("expected schedule guidance, got %v", err)
	}
}

func TestCEOFileWritePolicyAllowsStrategyDocsAndBlocksPlanningArtifacts(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "ceo", ToolCounts: map[string]int{}})

	for _, path := range []string{
		"docs/goals/active.md",
		"docs/goals/observations.md",
		"docs/product-specs/vision.md",
		"docs/reports/strategy/strategy-memo-2026-05-19.md",
	} {
		raw := []byte(`{"path":"` + path + `","content":"# Strategy\n"}`)
		if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
			t.Fatalf("expected CEO strategy write %s to pass, got %v", path, err)
		}
	}

	for _, path := range []string{
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/exec-plans/backlog/next-slice.md",
		"docs/features/F-002-checkout.md",
	} {
		raw := []byte(`{"path":"` + path + `","content":"# Planning\n"}`)
		err := preToolPolicy(ctx, root, "file_write", raw)
		if err == nil {
			t.Fatalf("expected CEO planning write %s to be blocked", path)
		}
		if !strings.Contains(err.Error(), "belongs to COO/CTO handoff") {
			t.Fatalf("expected CEO handoff boundary error, got %v", err)
		}
	}
}

func TestFileWritePolicyBlocksDuplicateFeatureContractID(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")

	for _, path := range []string{
		"docs/features/F-001.md",
		"docs/features/F-001-S001-project-brief-becomes-visible-product-slice.md",
	} {
		err := preToolPolicy(context.Background(), root, "file_write", []byte(`{"path":"`+path+`","content":"# F-001\n"}`))
		if err == nil {
			t.Fatalf("expected duplicate feature contract write to %s to be blocked", path)
		}
		if !strings.Contains(err.Error(), "canonical contract") {
			t.Fatalf("expected canonical contract policy error, got %v", err)
		}
	}

	if err := preToolPolicy(context.Background(), root, "file_write", []byte(`{"path":"docs/features/F-002.md","content":"# F-002\n"}`)); err != nil {
		t.Fatalf("expected feature write to pass when no contract with the same ID exists, got %v", err)
	}
	if err := preToolPolicy(context.Background(), root, "file_write", []byte(`{"path":"docs/features/F-001-product-walking-skeleton.md","content":"# F-001 updated\n"}`)); err != nil {
		t.Fatalf("expected canonical feature contract update to pass, got %v", err)
	}
}

func TestCOOFileWritePolicyAllowsPlanningDocsAndBlocksImplementation(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	for _, path := range []string{
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/exec-plans/backlog/next-slice.md",
		"docs/features/F-001-product-walking-skeleton.md",
		"docs/goals/observations.md",
	} {
		raw := []byte(`{"path":"` + path + `","content":"# Planning\n"}`)
		if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
			t.Fatalf("expected COO planning write %s to pass, got %v", path, err)
		}
	}

	err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"index.html","content":"<main></main>\n"}`))
	if err == nil {
		t.Fatal("expected COO implementation write to be blocked")
	}
	if !strings.Contains(err.Error(), "coo may only write planning artifacts") {
		t.Fatalf("expected COO planning boundary error, got %v", err)
	}
}

func TestCOOShellExecPolicyBlocksMutatingCommands(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"git status --short"}`)); err != nil {
		t.Fatalf("expected read-only COO shell command to pass, got %v", err)
	}
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"touch index.html"}`))
	if err == nil {
		t.Fatal("expected mutating COO shell command to be blocked")
	}
	if !strings.Contains(err.Error(), "coo cannot run mutating shell_exec") {
		t.Fatalf("expected COO shell boundary error, got %v", err)
	}
}

func TestDogfoodFileWritePolicyBlocksProductMutation(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "dogfood", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"package.json","content":"{}"}`))
	if err == nil {
		t.Fatal("expected dogfood product write to be blocked")
	}
	if !strings.Contains(err.Error(), "dogfood is observation-first") {
		t.Fatalf("expected dogfood observation-first error, got %v", err)
	}

	raw := []byte(`{"path":"docs/reports/dogfood/dogfood-2026-05-19.md","content":"# Evidence\n"}`)
	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected dogfood evidence report write to pass, got %v", err)
	}
}

func TestEngineerDispositionPolicyRequiresTicketDoneBeforeSuccess(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "backlog", "T-001-active.md", `---
id: T-001
title: Active
blocker: none
blocked_by: []
---

# Active
`)
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	raw := []byte(`{"status":"completed","ticket_id":"T-001","next_need":"qa_review"}`)

	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected successful disposition to be blocked while ticket is in backlog")
	}
	if !strings.Contains(err.Error(), "while it remains in docs/tickets/backlog") {
		t.Fatalf("expected backlog ticket state error, got %v", err)
	}

	if err := os.Rename(
		filepath.Join(dir, "docs", "tickets", "backlog", "T-001-active.md"),
		filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-active.md"),
	); err != nil {
		t.Fatalf("move ticket to in-progress: %v", err)
	}

	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected successful disposition to be blocked while ticket is in-progress")
	}
	if !strings.Contains(err.Error(), "while it remains in docs/tickets/in-progress") {
		t.Fatalf("expected in-progress ticket state error, got %v", err)
	}

	if err := os.Rename(
		filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-active.md"),
		filepath.Join(dir, "docs", "tickets", "done", "T-001-active.md"),
	); err != nil {
		t.Fatalf("move ticket to done: %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected disposition to pass after ticket moved to done, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"no_work","next_need":"no_need"}`)); err != nil {
		t.Fatalf("expected no_work disposition to pass without ticket_id, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","next_need":"qa_review"}`)); err == nil || !strings.Contains(err.Error(), "must name ticket_id") {
		t.Fatalf("expected completed disposition without ticket_id to be blocked, got %v", err)
	}
}

func TestEngineerClaimPolicyRequiresInProgressBeforeProductMutation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "backlog", "T-001-ship.md", `---
id: T-001
title: Ship
work_type: feature
blocker: none
blocked_by: []
---

# Ship
`)
	session := Session{Role: "engineer", ToolCounts: map[string]int{}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"src/index.html","content":"<h1>Ship</h1>\n"}`))
	if err == nil {
		t.Fatal("expected product file_write to require a claimed ticket")
	}
	if !strings.Contains(err.Error(), "must claim a product ticket") {
		t.Fatalf("expected claim policy error, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"docs/tickets/backlog/T-001-ship.md","content":"# Ship updated\n"}`)); err != nil {
		t.Fatalf("expected ticket-only update before claim to pass, got %v", err)
	}
	if err := checkEngineerClaimBeforeProductMutation(root, session, true, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/backlog/T-001-ship.md docs/tickets/in-progress/"}`)); err != nil {
		t.Fatalf("expected backlog-to-in-progress claim move to pass, got %v", err)
	}

	if err := os.Rename(
		filepath.Join(dir, "docs", "tickets", "backlog", "T-001-ship.md"),
		filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-ship.md"),
	); err != nil {
		t.Fatalf("move ticket to in-progress: %v", err)
	}
	if err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"src/index.html","content":"<h1>Ship</h1>\n"}`)); err != nil {
		t.Fatalf("expected product file_write after claim to pass, got %v", err)
	}
}

func TestJobDispositionPolicyRequiresCleanTreeForSuccessfulNonOrchestratorHandoff(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(dir, "docs", "features"), 0o755); err != nil {
		t.Fatalf("mkdir features: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"), []byte("# F-001\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw := []byte(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`)
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected dirty-tree disposition to be blocked")
	}
	if !strings.Contains(err.Error(), "git_commit") {
		t.Fatalf("expected git_commit remediation, got %v", err)
	}

	if err := runGitExit0(context.Background(), root, "add", "docs/features/F-001-product-walking-skeleton.md"); err != nil {
		t.Fatalf("git add feature: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "plan: add feature contract"); err != nil {
		t.Fatalf("git commit feature: %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected clean-tree disposition to pass, got %v", err)
	}
}

func TestJobDispositionPolicyRequiresCleanTreeForChangesRequestedHandoff(t *testing.T) {
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

	_, err = CreateTicket(root, TicketInput{
		Title:      "[Dogfood][Pre-flight] Missing game loop",
		Priority:   "high",
		Complexity: "small",
		WorkType:   "enabler",
		Source:     "dogfood test 2026-05-19",
		Body:       "## Context\nDogfood found missing behavior.\n\n## Requirements\nAdd it.\n\n## Acceptance criteria\n- [ ] Rework is claimable",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "dogfood", ToolCounts: map[string]int{}})
	raw := []byte(`{"status":"changes_requested","next_need":"implementation_rework","ticket_id":"T-001"}`)
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected changes_requested disposition to be blocked while ticket_create output is uncommitted")
	}
	if !strings.Contains(err.Error(), "docs/tickets/backlog/T-001-dogfood-pre-flight-missing-game-loop.md") {
		t.Fatalf("expected uncommitted ticket path in error, got %v", err)
	}

	if err := runGitExit0(context.Background(), root, "add", "docs/tickets/backlog/T-001-dogfood-pre-flight-missing-game-loop.md"); err != nil {
		t.Fatalf("git add ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "dogfood: E2E validation findings 2026-05-19"); err != nil {
		t.Fatalf("git commit ticket: %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected changes_requested disposition to pass after committing ticket, got %v", err)
	}
}

func TestJobDispositionPolicyIgnoresRuntimeLearningsOnlyDirtyState(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".harness"), 0o755); err != nil {
		t.Fatalf("mkdir harness: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".harness", "learnings.yaml"), []byte("schema_version: 1\n"), 0o644); err != nil {
		t.Fatalf("write learnings: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "README.md", ".harness/learnings.yaml"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".harness", "learnings.yaml"), []byte("schema_version: 1\nconventions:\n  dev_port: 8080\n"), 0o644); err != nil {
		t.Fatalf("update learnings: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	raw := []byte(`{"status":"completed","next_need":"review","suggested_role":"qa"}`)
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected runtime learnings-only dirty state to pass, got %v", err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "index.html"), []byte("<main></main>\n"), 0o644); err != nil {
		t.Fatalf("write product file: %v", err)
	}
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected product dirty state to remain blocked")
	}
	if !strings.Contains(err.Error(), "src/index.html") {
		t.Fatalf("expected product dirty path in error, got %v", err)
	}
	if strings.Contains(err.Error(), ".harness/learnings.yaml") {
		t.Fatalf("expected runtime learnings path to be omitted from blocking error, got %v", err)
	}
}

func TestJobDispositionPolicyLetsOrchestratorRouteDirtyState(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("demo updated\n"), 0o644); err != nil {
		t.Fatalf("write readme update: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "orchestrator", ToolCounts: map[string]int{}})
	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`))
	if err != nil {
		t.Fatalf("expected orchestrator disposition to remain able to route dirty state, got %v", err)
	}
}

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

func TestFileWritePolicyBlocksNewTicketMarkdownInLifecycleDir(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)

	err := preToolPolicy(context.Background(), root, "file_write", []byte(`{"path":"docs/tickets/backlog/T-002-new-ticket.md","content":"# Bad\n"}`))
	if err == nil {
		t.Fatal("expected new backlog ticket file_write to be blocked")
	}
	if !strings.Contains(err.Error(), "new ticket files must be created with ticket_create") {
		t.Fatalf("expected ticket_create policy error, got %v", err)
	}
}

func TestFileWritePolicyAllowsTicketReadmeAndExistingTicketUpdates(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "backlog", "T-001-existing.md", `---
id: T-001
title: Existing
---

# Existing
`)

	for _, raw := range [][]byte{
		[]byte(`{"path":"docs/tickets/README.md","content":"# Tickets\n"}`),
		[]byte(`{"path":"docs/tickets/backlog/T-001-existing.md","content":"# Existing updated\n"}`),
	} {
		if err := preToolPolicy(context.Background(), root, "file_write", raw); err != nil {
			t.Fatalf("expected file_write to pass for %s: %v", raw, err)
		}
	}
}

func setupPolicyTicketRepo(t *testing.T) (string, Root) {
	t.Helper()
	dir := t.TempDir()
	for _, status := range []string{"backlog", "in-progress", "in-review", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", status), 0o755); err != nil {
			t.Fatalf("mkdir tickets: %v", err)
		}
	}
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	return dir, root
}

func writePolicyTicket(t *testing.T, repoRoot, status, name, content string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "tickets", status, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}
}

func writePolicyPlan(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "exec-plans", "active", "current-operating-plan.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan: %v", err)
	}
	if err := os.WriteFile(path, []byte("# Current Operating Plan\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
}

func writePolicyFeature(t *testing.T, repoRoot, name string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	if err := os.WriteFile(path, []byte("# Feature\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}
