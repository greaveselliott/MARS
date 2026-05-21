/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-007-guardrails-and-safety.md
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
	if err := preToolPolicy(ctx, root, "ticket_create", first); err == nil || !strings.Contains(err.Error(), "already created a target-owned finding ticket") {
		t.Fatalf("expected same-run dogfood ticket block, got %v", err)
	}

	second := []byte(`{"title":"Finding two","priority":"high","kind":"intervention-debt","dedupe_key":"dogfood:repo:role:target:two","body":"## Context\nx"}`)
	if err := preToolPolicy(ctx, root, "ticket_create", second); err == nil || !strings.Contains(err.Error(), "already created a target-owned finding ticket") {
		t.Fatalf("expected same-run dogfood ticket block for a second finding, got %v", err)
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

func TestShellExecPolicyBlocksTicketDoneMoveWithUncommittedImplementationChanges(t *testing.T) {
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
- F-001-S001
end_to_end_evidence: required
evidence_links:
- go test ./cmd/note-stats
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
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore: seed ticket"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	source := filepath.Join(dir, "cmd", "note-stats", "main.go")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(source, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/in-progress/T-001-ship.md","docs/tickets/done/"]}`))
	if err == nil {
		t.Fatal("expected ticket done move to require separate implementation commit")
	}
	if !strings.Contains(err.Error(), "product/source changes to be committed first") ||
		!strings.Contains(err.Error(), "cmd/note-stats/main.go") {
		t.Fatalf("expected implementation commit guidance, got %v", err)
	}
}

func TestShellExecPolicyAllowsTicketDoneMoveWithDirtyTicketEvidenceOnly(t *testing.T) {
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
- F-001-S001
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore: seed ticket"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- go test ./cmd/note-stats
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/in-progress/T-001-ship.md","docs/tickets/done/"]}`))
	if err != nil {
		t.Fatalf("expected ticket-only dirty done move to pass, got %v", err)
	}
}

func TestShellExecPolicyAllowsEvidencedEnablerTicketDoneMove(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-002-remediate.md", `---
id: T-002
work_type: enabler
end_to_end_evidence: required
evidence_links: ["go test ./..."]
verified_by: engineer
---
# Remediate
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/in-progress/T-002-remediate.md docs/tickets/done/"}`))
	if err != nil {
		t.Fatalf("expected evidenced enabler ticket done move to pass, got %v", err)
	}
}

func TestShellExecPolicyBlocksEnablerTicketDoneMoveWithoutEvidence(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-002-remediate.md", `---
id: T-002
work_type: enabler
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---
# Remediate
`)

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/in-progress/T-002-remediate.md docs/tickets/done/"}`))
	if err == nil {
		t.Fatal("expected missing enabler evidence to block done move")
	}
	if strings.Contains(err.Error(), "work_type: feature") {
		t.Fatalf("expected enabler evidence guidance without forcing feature type, got %v", err)
	}
	if !strings.Contains(err.Error(), "evidence_links") || !strings.Contains(err.Error(), "verified_by") {
		t.Fatalf("expected missing evidence fields, got %v", err)
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

func TestFileWritePolicyBlocksScenarioIDsThatDoNotMatchFeatureContract(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")

	raw := []byte(`{"path":"docs/features/F-001-product-walking-skeleton.md","content":"# F-001\n\n### F-001-S001: Product brief becomes slice\n\nGiven a brief\nWhen planning runs\nThen F-001 stays aligned\n\n### F-002-S001: Core CLI functionality\n\nGiven text input\nWhen the CLI runs\nThen it counts text\n"}`)
	err := preToolPolicy(context.Background(), root, "file_write", raw)
	if err == nil {
		t.Fatal("expected cross-feature scenario heading to be blocked")
	}
	if !strings.Contains(err.Error(), "feature ID does not match the file") ||
		!strings.Contains(err.Error(), "F-002-S001") ||
		!strings.Contains(err.Error(), "expected F-001-SNNN") {
		t.Fatalf("expected scenario/file mismatch guidance, got %v", err)
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

func TestCTOFileWritePolicyAllowsTechnicalPlanningAndBlocksImplementation(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})

	for _, path := range []string{
		"docs/design-docs/current-architecture.md",
		"docs/reports/strategy/technical-risk.md",
		"docs/goals/observations.md",
	} {
		raw := []byte(`{"path":"` + path + `","content":"# Technical Planning\n"}`)
		if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
			t.Fatalf("expected CTO technical planning write %s to pass, got %v", path, err)
		}
	}

	for _, path := range []string{"go.mod", "README.md", "cmd/temperature-json-cli/main.go", "cmd/temperature-json-cli/main_test.go"} {
		raw := []byte(`{"path":"` + path + `","content":"package main\n"}`)
		err := preToolPolicy(ctx, root, "file_write", raw)
		if err == nil {
			t.Fatalf("expected CTO implementation write %s to be blocked", path)
		}
		if !strings.Contains(err.Error(), "cto may only write technical planning artifacts") ||
			!strings.Contains(err.Error(), "ticket_create and Engineer delivery") {
			t.Fatalf("expected CTO planning boundary error for %s, got %v", path, err)
		}
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

func TestSuccessfulDispositionBlocksUnresolvedTicketCreationFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	ctx := WithSession(context.Background(), Session{
		Role:       "cto-weekly",
		ToolCounts: map[string]int{ticketCreationOutstandingFailureKey: 1},
	})

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`))
	if err == nil {
		t.Fatal("expected successful disposition to be blocked by unresolved ticket_create failure")
	}
	if !strings.Contains(err.Error(), "ticket creation failed earlier") || !strings.Contains(err.Error(), "bdd_scenarios") {
		t.Fatalf("expected ticket creation guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"blocked","next_need":"ticket_breakdown","reason":"ticket_create failed"}`)); err != nil {
		t.Fatalf("expected blocked disposition to remain available, got %v", err)
	}
}

func TestPlanningRoleCanHandOffUnownedTicketCreationFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	ctx := WithSession(context.Background(), Session{
		Role:       "coo",
		ToolCounts: map[string]int{ticketCreationOutstandingFailureKey: 1},
	})

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`))
	if err != nil {
		t.Fatalf("expected COO ticket_breakdown handoff to remain available, got %v", err)
	}

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`))
	if err == nil || !strings.Contains(err.Error(), "ticket creation failed earlier") {
		t.Fatalf("expected implementation handoff to remain blocked, got %v", err)
	}
}

func TestEngineerClaimPolicyRequiresInProgressBeforeProductMutation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
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
	if err := checkEngineerClaimBeforeProductMutation(ctx, root, session, true, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/backlog/T-001-ship.md docs/tickets/in-progress/"}`)); err != nil {
		t.Fatalf("expected backlog-to-in-progress claim move to pass, got %v", err)
	}
	if err := checkEngineerClaimBeforeProductMutation(ctx, root, session, true, "shell_exec", []byte(`{"argv":"[\"git\", \"mv\", \"docs/tickets/backlog/T-001-ship.md\", \"docs/tickets/in-progress/\"]"}`)); err != nil {
		t.Fatalf("expected JSON-string argv claim move to pass, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["ls","docs/tickets/backlog/"]}`)); err == nil || !strings.Contains(err.Error(), "must claim T-001 before running shell_exec") {
		t.Fatalf("expected read-only shell_exec to be blocked before claim, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`)); err == nil || !strings.Contains(err.Error(), "must claim T-001 before running shell_exec") {
		t.Fatalf("expected no-op shell_exec to be redirected to claim before ticket claim, got %v", err)
	}

	if err := os.Rename(
		filepath.Join(dir, "docs", "tickets", "backlog", "T-001-ship.md"),
		filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-ship.md"),
	); err != nil {
		t.Fatalf("move ticket to in-progress: %v", err)
	}
	raw, err := json.Marshal(map[string]string{
		"path": "src/index.html",
		"content": `<!--
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
-->
<h1>Ship</h1>
`,
	})
	if err != nil {
		t.Fatalf("marshal file_write: %v", err)
	}
	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected product file_write after claim to pass, got %v", err)
	}
}

func TestEngineerPostValidationCommitBlocksExploratoryShellUntilTicketDone(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module note-stats\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go test ./...
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
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement ticket"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:git_commit:success":   1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["ls","-la","/tmp/"]}`))
	if err == nil {
		t.Fatal("expected post-validation exploratory shell to be blocked")
	}
	if !strings.Contains(err.Error(), "successful validation and a clean implementation commit") ||
		!strings.Contains(err.Error(), "file_read") ||
		!strings.Contains(err.Error(), "file_write") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected ticket completion guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/in-progress/T-001-ship.md","docs/tickets/done/"]}`))
	if err != nil {
		t.Fatalf("expected ticket lifecycle move to remain allowed, got %v", err)
	}
}

func TestEngineerPostValidationAllowsFreshExternalValidationArtifact(t *testing.T) {
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
- go build -o /tmp/note-stats-validation ./cmd/note-stats
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
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): reopen T-001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:                                1,
		"tool:git_commit:success":                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"): 1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text","hello world"]}`))
	if err != nil {
		t.Fatalf("expected fresh validation artifact execution to pass, got %v", err)
	}
}

func TestEngineerCannotCompleteTicketWithUnresolvedRuntimeValidationFailure(t *testing.T) {
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
- /tmp/note-stats-validation --text "hello world"
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

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:               1,
		unexpectedRuntimeValidationOutstandingKey: 1,
		"tool:git_commit:success":                 1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/in-progress/T-001-ship.md","docs/tickets/done/"]}`))
	if err == nil {
		t.Fatal("expected unresolved runtime validation failure to block done move")
	}
	if !strings.Contains(err.Error(), "unexpected runtime validation failure is unresolved") ||
		!strings.Contains(err.Error(), "rerun the exact failing command successfully") ||
		!strings.Contains(err.Error(), "missing-required-input") ||
		!strings.Contains(err.Error(), "expected_exit_code") {
		t.Fatalf("expected runtime validation repair guidance, got %v", err)
	}

	if err := runGitExit0(context.Background(), root, "mv", "docs/tickets/in-progress/T-001-ship.md", "docs/tickets/done/"); err != nil {
		t.Fatalf("stage done move: %v", err)
	}
	err = preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"chore(tickets): move T-001 to done"}`))
	if err == nil {
		t.Fatal("expected unresolved runtime validation failure to block done commit")
	}

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"in_review","ticket_id":"T-001","next_need":"qa_review"}`))
	if err == nil {
		t.Fatal("expected unresolved runtime validation failure to block successful disposition")
	}
}

func TestExternalValidationArtifactMustBeBuiltInSameSession(t *testing.T) {
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
- /tmp/note-stats-validation --text hello
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

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text","hello world"]}`))
	if err == nil {
		t.Fatal("expected stale external validation artifact to be blocked")
	}
	if !strings.Contains(err.Error(), "must be built in this role session") ||
		!strings.Contains(err.Error(), "shell_exec argv") {
		t.Fatalf("expected freshness guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), `shell_exec argv ["go","build","-o","/tmp/note-stats-validation","."]`) {
		t.Fatalf("expected exact build correction, got %v", err)
	}
}

func TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion(t *testing.T) {
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
- go run cmd/note-stats/main.go --text "hello world"
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
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement note stats"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	recordSessionToolOutcome(session, root, "shell_exec", json.RawMessage(`{"argv":["go","run","cmd/note-stats/main.go","--text","hello world"]}`), ToolResult{ExitCode: 0}, nil)
	session.ToolCounts["tool:git_commit:success"] = 1
	ctx := WithSession(context.Background(), *session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected post-runtime-validation no-op to be redirected")
	}
	if !strings.Contains(err.Error(), "successful validation and a clean implementation commit") ||
		!strings.Contains(err.Error(), "file_read") ||
		!strings.Contains(err.Error(), "file_write") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected ticket completion guidance after runtime validation, got %v", err)
	}
}

func TestEngineerPostValidationDirtyNoopBlocksBeforeGenericNoop(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-002-repair-route.md", `---
id: T-002
title: Repair route registration
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- curl http://localhost:8080/health
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Repair route registration
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): claim T-002"); err != nil {
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
		"tool:git_commit:success":   1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[":"]}`))
	if err == nil {
		t.Fatal("expected post-validation no-op with dirty work to be blocked")
	}
	if !strings.Contains(err.Error(), "successful validation and dirty implementation or ticket work") ||
		!strings.Contains(err.Error(), "T-002") ||
		!strings.Contains(err.Error(), "main.go") ||
		!strings.Contains(err.Error(), "git_commit") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected dirty-work convergence guidance, got %v", err)
	}
}

func TestEngineerPostValidationGateAllowsValidationWhileImplementationDirty(t *testing.T) {
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
- go test ./...
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
		"tool:git_commit:success":   1,
	}})
	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`)); err != nil {
		t.Fatalf("expected validation shell to pass while implementation is dirty, got %v", err)
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

func TestEngineerMustReopenDoneTicketBeforeProductMutation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "done", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	rawWrite := []byte(`{"path":"main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	err := preToolPolicy(ctx, root, "file_write", rawWrite)
	if err == nil {
		t.Fatal("expected product mutation to require reopening the done ticket")
	}
	if !strings.Contains(err.Error(), "reopen product ticket T-001") || !strings.Contains(err.Error(), "docs/tickets/in-progress") {
		t.Fatalf("expected reopen guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected shell_exec to require reopening the done ticket")
	}
	if !strings.Contains(err.Error(), "must reopen T-001") {
		t.Fatalf("expected shell reopen guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","/tmp/note-stats-validation"]}`))
	if err != nil {
		t.Fatalf("expected external validation artifact cleanup to pass after ticket completion, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/done/T-001-ship.md","docs/tickets/in-progress/"]}`))
	if err != nil {
		t.Fatalf("expected ticket reopen move to pass, got %v", err)
	}
}

func TestEngineerCompletionCommitAllowsTicketMoveToDoneWithProductFiles(t *testing.T) {
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
- go test ./...
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
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755); err != nil {
		t.Fatalf("mkdir command: %v", err)
	}
	source := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main_test.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write test: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "mv", "docs/tickets/in-progress/T-001-ship.md", "docs/tickets/done/"); err != nil {
		t.Fatalf("git mv done: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:git_commit:success":   1,
	}})
	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"feat: implement T-001"}`))
	if err != nil {
		t.Fatalf("expected final implementation commit with in-progress to done ticket move to pass, got %v", err)
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

func TestJobDispositionPolicyBlocksSuccessfulReviewWhenDocSyncFails(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "task-notes-api"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	source := filepath.Join(dir, "cmd", "task-notes-api", "main.go")
	if err := os.WriteFile(source, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testSource := filepath.Join(dir, "cmd", "task-notes-api", "main_test.go")
	if err := os.WriteFile(testSource, []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: seed source"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandSuccessKey:       1,
	}})
	raw := []byte(`{"status":"approved","next_need":"security_review","ticket_id":"T-001"}`)
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected successful disposition to be blocked by docsync findings")
	}
	if !strings.Contains(err.Error(), "successful disposition blocked by docsync_audit findings") {
		t.Fatalf("expected docsync policy error, got %v", err)
	}

	if err := os.WriteFile(source, []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write source metadata: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add metadata: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "docs: add docsync metadata"); err != nil {
		t.Fatalf("git commit metadata: %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected successful disposition to pass after docsync fix, got %v", err)
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

func TestDogfoodUncommittedFindingBlocksFurtherValidationAndTickets(t *testing.T) {
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

	const ticketPath = "docs/tickets/backlog/T-001-dogfood-missing-required-text-validation.md"
	_, err = CreateTicket(root, TicketInput{
		Title:      "[Dogfood] Missing required text validation",
		Priority:   "high",
		Complexity: "small",
		WorkType:   "enabler",
		Source:     "dogfood test 2026-05-20",
		Body:       "## Context\nDogfood found missing behavior.\n\n## Requirements\nValidate the required text argument.\n\n## Acceptance criteria\n- [ ] Rework is claimable",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "dogfood", ToolCounts: map[string]int{}})
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected further validation to be blocked while dogfood finding ticket is uncommitted")
	}
	if !strings.Contains(err.Error(), "uncommitted target-owned finding ticket") ||
		!strings.Contains(err.Error(), ticketPath) ||
		!strings.Contains(err.Error(), "before continuing validation or creating another ticket") {
		t.Fatalf("expected uncommitted finding guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "ticket_create", []byte(`{"title":"[Dogfood] Duplicate missing text validation","priority":"high","work_type":"enabler","body":"## Context\nx"}`))
	if err == nil {
		t.Fatal("expected duplicate ticket creation to be blocked while prior finding is uncommitted")
	}
	if !strings.Contains(err.Error(), "creating another ticket") {
		t.Fatalf("expected duplicate-ticket guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "git_status", []byte(`{}`)); err != nil {
		t.Fatalf("expected git_status to remain available, got %v", err)
	}
	rawCommit := []byte(`{"message":"dogfood: E2E validation findings 2026-05-20","paths":["` + ticketPath + `"]}`)
	if err := preToolPolicy(ctx, root, "git_commit", rawCommit); err != nil {
		t.Fatalf("expected git_commit to remain available for finding ticket, got %v", err)
	}
}

func TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation(t *testing.T) {
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
		t.Fatalf("git add readme: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "init"); err != nil {
		t.Fatalf("git commit readme: %v", err)
	}

	const ticketPath = "docs/tickets/backlog/T-001-dogfood-test-mismatch.md"
	if _, err := CreateTicket(root, TicketInput{
		Title:      "[Dogfood] Test mismatch",
		Priority:   "high",
		Complexity: "small",
		WorkType:   "enabler",
		Source:     "dogfood test 2026-05-20",
		Body:       "## Context\nDogfood found a failing validation test.\n",
	}); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "add", ticketPath); err != nil {
		t.Fatalf("git add ticket: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "dogfood: E2E validation findings 2026-05-20"); err != nil {
		t.Fatalf("git commit ticket: %v", err)
	}

	session := Session{Role: "dogfood", ToolCounts: map[string]int{"ticket_create:dogfood:total": 1}}
	ctx := WithSession(context.Background(), session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected further validation to be blocked after dogfood ticket creation")
	}
	if !strings.Contains(err.Error(), "already created a target-owned finding ticket") ||
		!strings.Contains(err.Error(), "record job_disposition_record") {
		t.Fatalf("expected disposition-before-validation guidance, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"changes_requested","ticket_id":"T-001","next_need":"implementation_rework"}`)); err != nil {
		t.Fatalf("expected disposition to remain available after committed dogfood ticket, got %v", err)
	}
}

func TestReviewApprovalRequiresPassingValidationWhenTestsExist(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	testPath := filepath.Join(dir, "cmd", "note-stats", "main_test.go")
	if err := os.MkdirAll(filepath.Dir(testPath), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(testPath, []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	raw := []byte(`{"status":"approved","ticket_id":"T-001","next_need":"security_review"}`)

	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA approval without tests to be blocked")
	}
	if !strings.Contains(err.Error(), "must run the repository's authoritative test command") {
		t.Fatalf("expected test command guidance, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "security", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandFailureKey:       1,
	}})
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected Security approval after failing tests to be blocked")
	}
	if !strings.Contains(err.Error(), "after a failing build or test command") {
		t.Fatalf("expected failing validation guidance, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:      1,
		expectedRuntimeFailureSuccessKey: 1,
		testCommandSuccessKey:            1,
	}})
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected QA approval after passing tests and an expected runtime error probe to pass, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		validationCommandFailureKey: 1,
		testCommandSuccessKey:       1,
	}})
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA approval after an unexpected runtime validation failure to be blocked")
	}
	if !strings.Contains(err.Error(), "unexpected failing validation command") {
		t.Fatalf("expected unexpected runtime validation guidance, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandSuccessKey:       1,
	}})
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected QA approval after passing tests to pass, got %v", err)
	}
}

func TestQAApprovalRequiresGoTestsForGoSource(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	sourcePath := filepath.Join(dir, "cmd", "note-stats", "main.go")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main

func main() {}
`), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandSuccessKey:       1,
	}})
	err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"approved","ticket_id":"T-001","next_need":"security_review"}`))
	if err == nil {
		t.Fatal("expected QA approval without Go tests to be blocked")
	}
	if !strings.Contains(err.Error(), "Go source files exist but no _test.go files are present") {
		t.Fatalf("expected Go test coverage guidance, got %v", err)
	}
}

func TestReviewValidationFailureBlocksFurtherShellBeforeDisposition(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		testCommandFailureKey:       1,
		validationCommandFailureKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected further QA shell validation to be blocked after validation failure")
	}
	if !strings.Contains(err.Error(), "already observed a failing build, test, or unexpected runtime validation command") ||
		!strings.Contains(err.Error(), "job_disposition_record") ||
		!strings.Contains(err.Error(), "changes_requested") {
		t.Fatalf("expected terminal changes_requested guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"changes_requested","ticket_id":"T-001","next_need":"implementation_rework"}`)); err != nil {
		t.Fatalf("expected changes_requested disposition to remain available after validation failure, got %v", err)
	}
}

func TestReviewValidationFailureAllowsExactExpectedExitCorrection(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	session := Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey:                                1,
		testCommandSuccessKey:                                      1,
		validationCommandSuccessKey:                                1,
		validationArtifactSessionKey("/tmp/note-stats-validation"): 1,
		unexpectedRuntimeValidationFailureKey(failedArgs, 1):       1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)); err != nil {
		t.Fatalf("expected exact expected-exit correction to be allowed, got %v", err)
	}

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected unrelated validation to remain blocked")
	}
	if !strings.Contains(err.Error(), "rerun that exact command once with shell_exec expected_exit_code") {
		t.Fatalf("expected exact-rerun guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyBlocksMutatingSetupCommands(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","note-stats-cli"]}`))
	if err == nil {
		t.Fatal("expected QA mutating setup command to be blocked")
	}
	if !strings.Contains(err.Error(), "qa shell_exec is validation-only") ||
		!strings.Contains(err.Error(), "go mod init note-stats-cli") {
		t.Fatalf("expected validation-only shell guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyAllowsValidationCommands(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := Session{Role: "qa", ToolCounts: map[string]int{
		validationArtifactSessionKey("/tmp/note-stats-validation"): 1,
	}}
	ctx := WithSession(context.Background(), session)

	for _, raw := range [][]byte{
		[]byte(`{"argv":["go","test","./..."]}`),
		[]byte(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`),
		[]byte(`{"argv":["/tmp/note-stats-validation","--text","hello"]}`),
		[]byte(`{"argv":["curl","-fsS","http://127.0.0.1:8080/health"]}`),
	} {
		if err := preToolPolicy(ctx, root, "shell_exec", raw); err != nil {
			t.Fatalf("expected review validation command to pass for %s, got %v", string(raw), err)
		}
	}
}

func TestReviewShellExecPolicyBlocksNoopPlaceholders(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "cannot use shell_exec no-op placeholders") {
		t.Fatalf("expected no-op review guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesPostValidationNoopToDisposition(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		"tool:docsync_audit:success": 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "after successful validation") ||
		!strings.Contains(err.Error(), "job_disposition_record") ||
		!strings.Contains(err.Error(), "status approved") {
		t.Fatalf("expected successful-validation disposition guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesPostBuildNoopToTestsWhenTestsExist(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		buildCommandSuccessKey:      1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "authoritative test command") ||
		strings.Contains(err.Error(), "status approved") {
		t.Fatalf("expected missing-test guidance instead of approval guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesNoTestGoRepoToChangesRequested(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	src := filepath.Join(dir, "cmd", "temperature-json-cli", "main.go")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "changes_requested") ||
		!strings.Contains(err.Error(), "no _test.go files") {
		t.Fatalf("expected missing-test changes_requested guidance, got %v", err)
	}
}

func TestReviewTerminalEvidenceWaitsForDocSyncAudit(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:file_read:success":    1,
	}}

	if ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence to wait for docsync_audit")
	}

	session.ToolCounts["tool:docsync_audit:success"] = 1
	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence after validation, read evidence, and docsync_audit")
	}
}

func TestReviewTerminalEvidenceWaitsForTestsWhenTestFilesExist(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}}

	if ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence to wait for test command success when test files exist")
	}

	session.ToolCounts[testCommandSuccessKey] = 1
	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence after tests, validation, read evidence, and docsync_audit")
	}
}

func TestReviewTerminalDispositionRequiredBlocksFurtherShellExec(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:          1,
		reviewTerminalDispositionRequiredKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","./..."]}`))
	if err == nil {
		t.Fatal("expected further QA shell_exec to be blocked")
	}
	if !strings.Contains(err.Error(), "Do not call more tools except job_disposition_record") {
		t.Fatalf("expected terminal-only guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"approved","ticket_id":"T-001","next_need":"no_need"}`)); err != nil {
		t.Fatalf("expected terminal disposition to remain available, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesPostFailureNoopToChangesRequested(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "already observed a failing build, test, or unexpected runtime validation command") ||
		!strings.Contains(err.Error(), "job_disposition_record") ||
		!strings.Contains(err.Error(), "changes_requested") {
		t.Fatalf("expected failing-validation disposition guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksRepeatedRuntimeProbeUntilEdit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text",""]}`))
	if err == nil {
		t.Fatal("expected repeated runtime probe to be blocked until Engineer edits")
	}
	if !strings.Contains(err.Error(), "already failed unexpectedly") || !strings.Contains(err.Error(), "file_read/file_write") {
		t.Fatalf("expected edit-before-rerun guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureAllowsExactRerunAfterEdit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text",""]}`)); err != nil {
		t.Fatalf("expected exact runtime rerun after edit to be allowed, got %v", err)
	}
}

func TestExternalValidationArtifactMustBeRebuiltAfterRuntimeFailureEdit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                             1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs):          1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):                  0,
		runtimeValidationEditAfterFailureKey:                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):            1,
		validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation"): 0,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text",""]}`))
	if err == nil {
		t.Fatal("expected stale external validation artifact to be blocked")
	}
	if !strings.Contains(err.Error(), "built before a post-failure implementation edit") ||
		!strings.Contains(err.Error(), `shell_exec argv ["go","build","-o","/tmp/note-stats-validation","."]`) {
		t.Fatalf("expected rebuild guidance, got %v", err)
	}
}

func TestExternalValidationArtifactAllowsRerunAfterRuntimeFailureEditAndRebuild(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                             1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs):          1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):                  0,
		runtimeValidationEditAfterFailureKey:                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):            2,
		validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation"): 1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text",""]}`)); err != nil {
		t.Fatalf("expected rebuilt external validation artifact rerun to pass, got %v", err)
	}
}

func TestEngineerTicketEvidenceWriteRequiresValidation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-note-stats.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship note stats
`)
	content := `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
---

# Ship note stats
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected ticket evidence write to require successful validation")
	}
	if !strings.Contains(err.Error(), "before successful validation") ||
		!strings.Contains(err.Error(), "go test") {
		t.Fatalf("expected validation-first guidance, got %v", err)
	}
}

func TestEngineerTicketEvidenceWriteAllowedAfterValidation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-note-stats.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship note stats
`)
	content := `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
---

# Ship note stats
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected ticket evidence write after validation to pass, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksDifferentRuntimeProbe(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats"]}`))
	if err == nil {
		t.Fatal("expected different runtime probe to be blocked while exact repair is outstanding")
	}
	if !strings.Contains(err.Error(), "unresolved unexpected runtime validation failure from an earlier command") {
		t.Fatalf("expected exact-repair guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing-required-input") || !strings.Contains(err.Error(), "expected_exit_code") {
		t.Fatalf("expected missing-argument correction guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksShellWrapperBypass(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):   1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"cd /tmp && /tmp/note-stats-validation --text \"hello world\""}`))
	if err == nil {
		t.Fatal("expected shell wrapper runtime probe to be blocked while exact repair is outstanding")
	}
	if !strings.Contains(err.Error(), "shell wrappers") ||
		!strings.Contains(err.Error(), "rerun the exact failing command successfully") {
		t.Fatalf("expected shell-wrapper bypass guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksValidationUnrelatedShell(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected unrelated validation shell command to be blocked while exact repair is outstanding")
	}
	if !strings.Contains(err.Error(), "Do not run shell_exec for other probes") ||
		!strings.Contains(err.Error(), "rerun the exact failing command successfully") {
		t.Fatalf("expected exact-repair guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksExpectedExitRerun(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text",""],"expected_exit_code":1}`))
	if err == nil {
		t.Fatal("expected Engineer expected-exit rerun to be blocked after unexpected runtime failure")
	}
	if !strings.Contains(err.Error(), "cannot use expected_exit_code") ||
		!strings.Contains(err.Error(), "missing-required-input") {
		t.Fatalf("expected expected-exit block guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureAllowsMissingArgumentExpectedExitCorrection(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureKey(failedArgs, 1):         1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):   1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)); err != nil {
		t.Fatalf("expected Engineer missing-argument expected-exit correction to be allowed, got %v", err)
	}
}

func TestEngineerMissingArgumentRuntimeFailureBlocksUnrelatedMutation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			unexpectedRuntimeValidationOutstandingKey:                    1,
			unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		},
		ToolState: map[string]string{
			unexpectedRuntimeValidationMissingArgKey: "true",
			unexpectedRuntimeValidationCorrectionKey: `shell_exec {"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`,
		},
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"cmd/note-stats/main.go","content":"package main\n"}`))
	if err == nil {
		t.Fatal("expected unrelated mutation to be blocked until missing-argument correction")
	}
	if !strings.Contains(err.Error(), "Run the exact correction next") ||
		!strings.Contains(err.Error(), `"expected_exit_code":1`) ||
		!strings.Contains(err.Error(), "/tmp/note-stats-validation") {
		t.Fatalf("expected exact correction guidance, got %v", err)
	}
}

func TestEngineerMissingArgumentRuntimeFailureAllowsImplementationEditAfterCorrectionAttempt(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	correction := `shell_exec {"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`
	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			unexpectedRuntimeValidationOutstandingKey:                    1,
			unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		},
		ToolState: map[string]string{
			unexpectedRuntimeValidationMissingArgKey: "true",
			unexpectedRuntimeValidationCorrectionKey: correction,
			unexpectedRuntimeValidationAttemptedKey:  correction,
		},
	}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"cmd/note-stats/main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)); err != nil {
		t.Fatalf("expected implementation edit after failed missing-argument correction attempt, got %v", err)
	}

	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"fix cli missing input handling"}`))
	if err == nil {
		t.Fatal("expected commit to remain blocked until validation repairs the outstanding failure")
	}
	if !strings.Contains(err.Error(), "Run the exact correction next") ||
		!strings.Contains(err.Error(), `"expected_exit_code":1`) {
		t.Fatalf("expected commit to keep exact validation guidance, got %v", err)
	}
}

func TestEngineerPositiveRuntimeFailureBlocksImplementationCommit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"feat: implement note stats"}`))
	if err == nil {
		t.Fatal("expected implementation commit to be blocked while runtime acceptance failure is unresolved")
	}
	if !strings.Contains(err.Error(), "cannot commit product work") ||
		!strings.Contains(err.Error(), "Keep the failed implementation uncommitted") {
		t.Fatalf("expected commit block guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureAllowsStaleValidationArtifactRebuild(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                             1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs):          1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):                  0,
		runtimeValidationEditAfterFailureKey:                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):            1,
		validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation"): 0,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)); err != nil {
		t.Fatalf("expected stale validation artifact rebuild to remain available, got %v", err)
	}
}

func TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats/..."}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./cmd/note-stats/..."]}`,
		testBuildValidationScopeKey:   "cmd/note-stats/",
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text","hello"]}`))
	if err == nil {
		t.Fatal("expected runtime probe to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "failing test or build command") ||
		!strings.Contains(err.Error(), "test/build command") {
		t.Fatalf("expected test/build repair guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./cmd/note-stats"]}`))
	if err == nil {
		t.Fatal("expected unchanged same-lane test rerun to be blocked")
	}
	if !strings.Contains(err.Error(), "latest repair edit") ||
		!strings.Contains(err.Error(), "file_read/file_write") {
		t.Fatalf("expected edit-before-rerun guidance, got %v", err)
	}

	session.ToolCounts[testBuildValidationEditAfterFailureKey] = 1
	ctx = WithSession(context.Background(), session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","./cmd/note-stats"]}`))
	if err == nil {
		t.Fatal("expected build command to be blocked while test failure is unresolved")
	}
	if !strings.Contains(err.Error(), "unresolved test failure") {
		t.Fatalf("expected same-lane guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./cmd/note-stats"]}`)); err != nil {
		t.Fatalf("expected focused same-lane test rerun after source edit to be allowed, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"cd cmd/note-stats && go test -v ."}`)); err != nil {
		t.Fatalf("expected simple cd plus same-lane test shell command to be allowed, got %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", []byte(`{"path":"verify_functionality.sh","content":"#!/bin/sh\necho ok\n"}`))
	if err == nil {
		t.Fatal("expected helper script write to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "helper scripts") {
		t.Fatalf("expected helper script guidance, got %v", err)
	}

	sourceRaw := []byte(`{"path":"cmd/note-stats/main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	if err := preToolPolicy(ctx, root, "file_write", sourceRaw); err != nil {
		t.Fatalf("expected source repair write to be allowed, got %v", err)
	}

	rootSourceRaw := []byte(`{"path":"main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	err = preToolPolicy(ctx, root, "file_write", rootSourceRaw)
	if err == nil {
		t.Fatal("expected alternate root source write to be blocked while package test is unresolved")
	}
	if !strings.Contains(err.Error(), "failed test/build scope") && !strings.Contains(err.Error(), "alternate entrypoints") {
		t.Fatalf("expected failed-scope guidance, got %v", err)
	}
}

func TestEngineerFailingTestAllowsSameJobRepairTestFileRemoval(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main_test.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "old_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write old_test.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats"}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey:                              `shell_exec {"argv":["go","test","./cmd/note-stats"]}`,
		testBuildValidationOutputKey:                               "main_test.go: helper redeclared in this block\nold_test.go: other declaration of helper",
		testBuildRepairWritePathKey("cmd/note-stats/main_test.go"): "true",
		testBuildRepairWritePathKey("cmd/note-stats/main.go"):      "true",
	}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main_test.go"]}`)); err != nil {
		t.Fatalf("expected same-job repair test file removal to be allowed, got %v", err)
	}

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/old_test.go"]}`))
	if err == nil {
		t.Fatal("expected unmarked test file removal to remain blocked")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected failing-test repair-lane guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main.go"]}`))
	if err == nil {
		t.Fatal("expected source removal to remain blocked even when the source was rewritten during repair")
	}
}

func TestEngineerFailingTestAllowsMissingGoModuleBootstrap(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-notes-api.md", "# T-001\n")

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./internal/note"}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./internal/note"]}`,
		testBuildValidationOutputKey:  "go: cannot find main module, but found .git/config in /tmp/demo-notes-api\n\tto create a module there, run:\n\tgo mod init",
		testBuildValidationScopeKey:   "internal/note/",
	}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","demo-notes-api"]}`)); err != nil {
		t.Fatalf("expected missing Go module bootstrap to be allowed, got %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo-notes-api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","demo-notes-api"]}`))
	if err == nil {
		t.Fatal("expected go mod init to be blocked once go.mod exists")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected unresolved failure guidance, got %v", err)
	}

	if err := os.Remove(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("remove go.mod: %v", err)
	}
	session.ToolState[testBuildValidationOutputKey] = "FAIL: TestCreateNote expected title"
	ctx = WithSession(context.Background(), session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","demo-notes-api"]}`))
	if err == nil {
		t.Fatal("expected go mod init to stay blocked when failure output is not missing-module evidence")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected unresolved failure guidance, got %v", err)
	}
}

func TestEngineerFailingTestAllowsRemovalOfTestFileWrittenBeforeFailure(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "old_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write old_test.go: %v", err)
	}

	session := Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	writtenTestRaw := []byte(`{"path":"cmd/note-stats/main_test.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	recordSessionToolOutcome(&session, root, "file_write", writtenTestRaw, ToolResult{}, nil)
	writtenSourceRaw := []byte(`{"path":"cmd/note-stats/main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	recordSessionToolOutcome(&session, root, "file_write", writtenSourceRaw, ToolResult{}, nil)

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats"}}
	session.ToolCounts[testBuildValidationOutstandingKey] = 1
	session.ToolCounts[testCommandFailureKey] = 1
	session.ToolCounts[testBuildValidationFailureFingerprintKey(failedArgs)] = 1
	session.ToolCounts[testBuildValidationFailureEditWatermarkKey(failedArgs)] = 0
	session.ToolState[testBuildValidationCommandKey] = `shell_exec {"argv":["go","test","./cmd/note-stats"]}`
	session.ToolState[testBuildValidationOutputKey] = "main_test.go: helper redeclared in this block\nold_test.go: other declaration of helper"
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"rm -f cmd/note-stats/main_test.go"}`)); err != nil {
		t.Fatalf("expected same-job pre-failure test file removal to be allowed, got %v", err)
	}

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/old_test.go"]}`))
	if err == nil {
		t.Fatal("expected pre-existing test file removal to remain blocked")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected failing-test repair-lane guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main.go"]}`))
	if err == nil {
		t.Fatal("expected source removal to remain blocked even when the source was written earlier in the job")
	}
}

func TestEngineerFailingTestBlocksSameJobTestRemovalForAssertionFailure(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")

	session := Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	writtenTestRaw := []byte(`{"path":"cmd/note-stats/main_test.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	recordSessionToolOutcome(&session, root, "file_write", writtenTestRaw, ToolResult{}, nil)

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats"}}
	session.ToolCounts[testBuildValidationOutstandingKey] = 1
	session.ToolCounts[testCommandFailureKey] = 1
	session.ToolCounts[testBuildValidationFailureFingerprintKey(failedArgs)] = 1
	session.ToolCounts[testBuildValidationFailureEditWatermarkKey(failedArgs)] = 0
	session.ToolState[testBuildValidationCommandKey] = `shell_exec {"argv":["go","test","./cmd/note-stats"]}`
	session.ToolState[testBuildValidationOutputKey] = "main_test.go:42: expected 3 items, got 2"
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main_test.go"]}`))
	if err == nil {
		t.Fatal("expected assertion-failure test removal to remain blocked")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected failing-test repair-lane guidance, got %v", err)
	}
}

func TestEngineerFailingTestBlocksCommitTicketEvidenceAndDisposition(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship note stats
`)
	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats/..."}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:                            1,
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./cmd/note-stats/..."]}`,
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"feat: implement note stats"}`))
	if err == nil {
		t.Fatal("expected product commit to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "cannot commit product work") ||
		!strings.Contains(err.Error(), "exact unresolved command") {
		t.Fatalf("expected commit block guidance, got %v", err)
	}

	ticketContent := `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links:
- go test ./cmd/note-stats/...
verified_by:
- engineer
---

# Ship note stats
`
	raw, marshalErr := json.Marshal(fileWriteArgs{
		Path:    filepath.Join("docs", "tickets", "in-progress", "T-001-note-stats.md"),
		Content: ticketContent,
	})
	if marshalErr != nil {
		t.Fatalf("marshal file_write: %v", marshalErr)
	}
	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected ticket evidence write to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "update ticket evidence") ||
		!strings.Contains(err.Error(), "failing test or build") {
		t.Fatalf("expected ticket evidence block guidance, got %v", err)
	}

	sourceRaw, marshalErr := json.Marshal(fileWriteArgs{
		Path: "cmd/note-stats/main.go",
		Content: `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`,
	})
	if marshalErr != nil {
		t.Fatalf("marshal source file_write: %v", marshalErr)
	}
	if err := preToolPolicy(ctx, root, "file_write", sourceRaw); err != nil {
		t.Fatalf("expected source repair write to remain available, got %v", err)
	}

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","ticket_id":"T-001","next_need":"qa_review"}`))
	if err == nil {
		t.Fatal("expected successful disposition to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "record a successful product disposition") ||
		!strings.Contains(err.Error(), "failing test or build") {
		t.Fatalf("expected disposition block guidance, got %v", err)
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

func TestFileWritePolicyRequiresDocSyncForSourceFiles(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")

	raw, err := json.Marshal(map[string]string{
		"path":    "src/main.go",
		"content": "package main\n\nfunc main() {}\n",
	})
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}
	err = preToolPolicy(context.Background(), root, "file_write", raw)
	if err == nil {
		t.Fatal("expected source file without MarsDocSync metadata to be blocked")
	}
	if !strings.Contains(err.Error(), "must include top-of-file MarsDocSync docs metadata") {
		t.Fatalf("expected MarsDocSync policy error, got %v", err)
	}

	raw, err = json.Marshal(map[string]string{
		"path": "src/main.go",
		"content": `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main

func main() {}
`,
	})
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}
	if err := preToolPolicy(context.Background(), root, "file_write", raw); err != nil {
		t.Fatalf("expected source file with existing doc metadata to pass, got %v", err)
	}
}

func TestFileWritePolicyRejectsSourceDocSyncMissingDoc(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	raw, err := json.Marshal(map[string]string{
		"path": "src/main.go",
		"content": `/*
MarsDocSync:
docs:
- docs/features/F-001-S002.md
*/
package main

func main() {}
`,
	})
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}
	err = preToolPolicy(context.Background(), root, "file_write", raw)
	if err == nil {
		t.Fatal("expected source file metadata pointing at a scenario ID path to be blocked")
	}
	if !strings.Contains(err.Error(), "references missing doc docs/features/F-001-S002.md") {
		t.Fatalf("expected missing doc policy error, got %v", err)
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
