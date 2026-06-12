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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestTicketDoneMoveSourcesPreserveShellCommandPathCase(t *testing.T) {
	t.Parallel()
	fields := shellFieldsPreserveCase(`git mv docs/tickets/in-progress/T-001-Ship.md docs/tickets/done/`)
	sources := ticketDoneMoveSources(fields)
	if len(sources) != 1 {
		t.Fatalf("expected one ticket done move source, got %v", sources)
	}
	if sources[0] != "docs/tickets/in-progress/T-001-Ship.md" {
		t.Fatalf("expected source path case to be preserved, got %q", sources[0])
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

func TestShellExecPolicyAllowsTicketDoneMoveWithOnlyWorkspaceNoiseDirty(t *testing.T) {
	t.Parallel()
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "in-progress"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("baseline\n"), 0o644))
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["npm test"]
verified_by: engineer
---
# Ship
`)
	runTestGit(t, dir, "add", ".DS_Store", "docs/tickets/in-progress/T-001-ship.md")
	runTestGit(t, dir, "commit", "-m", "seed ticket and workspace noise")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("changed\n"), 0o644))

	err := preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"git mv docs/tickets/in-progress/T-001-ship.md docs/tickets/done/"}`))
	if err != nil {
		t.Fatalf("expected workspace noise not to block ticket done move, got %v", err)
	}
}

func TestGitCommitPolicyBlocksWorkspaceNoiseWithoutReopenLoop(t *testing.T) {
	t.Parallel()
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))
	writePolicyTicket(t, dir, "done", "T-001-ship.md", `---
id: T-001
work_type: feature
bdd_scenarios: ["F-001-S001"]
evidence_links: ["npm test"]
verified_by: engineer
---
# Ship
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", ".DS_Store", "docs/tickets/done/T-001-ship.md")
	runTestGit(t, dir, "commit", "-m", "seed done ticket and workspace noise")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("changed\n"), 0o644))

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"chore: remove workspace noise","paths":[".DS_Store"]}`))
	if err == nil {
		t.Fatal("expected workspace noise commit to be blocked")
	}
	if !strings.Contains(err.Error(), "workspace noise paths cannot be committed") {
		t.Fatalf("expected workspace noise policy instead of ticket reopen loop, got %v", err)
	}
	if strings.Contains(err.Error(), "must reopen product ticket") {
		t.Fatalf("workspace noise should not force product ticket reopen, got %v", err)
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

func TestDuplicateFeatureContractGuidanceRespectsRoleWriteBoundary(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	raw := []byte(`{"path":"docs/features/F-001.md","content":"# F-001\n"}`)

	ceoCtx := WithSession(context.Background(), Session{Role: "ceo", ToolCounts: map[string]int{}})
	err := preToolPolicy(ceoCtx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected CEO duplicate feature write to be blocked")
	}
	for _, want := range []string{"ceo cannot write feature contracts", "hand off to COO", "docs/features/F-001-product-walking-skeleton.md"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected CEO role-aware duplicate guidance containing %q, got %v", want, err)
		}
	}

	cooCtx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})
	err = preToolPolicy(cooCtx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected COO duplicate feature write to be blocked")
	}
	if !strings.Contains(err.Error(), "update the canonical contract") {
		t.Fatalf("expected COO canonical contract guidance, got %v", err)
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

func TestCOOFileWritePolicyBlocksSecondActiveExecPlanWithSpecificGuidance(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"docs/exec-plans/active/current-failing-scenario.md","content":"# Current Failing Scenario\n"}`))
	if err == nil {
		t.Fatal("expected COO second active exec-plan write to be blocked")
	}
	if !strings.Contains(err.Error(), "keep exactly one active exec plan") ||
		!strings.Contains(err.Error(), "current-operating-plan.md") {
		t.Fatalf("expected single-active-plan guidance, got %v", err)
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

func TestPlanningRoleShellExecPolicyBlocksMutatingCommands(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)

	for _, role := range []string{"ceo", "head-of-strategy", "coo", "cto", "cto-weekly"} {
		role := role
		t.Run(role, func(t *testing.T) {
			ctx := WithSession(context.Background(), Session{Role: role, ToolCounts: map[string]int{}})

			if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"git status --short"}`)); err != nil {
				t.Fatalf("expected read-only %s shell command to pass, got %v", role, err)
			}
			for _, raw := range [][]byte{
				[]byte(`{"shell_command":"touch index.html"}`),
				[]byte(`{"shell_command":"npm init -y"}`),
			} {
				err := preToolPolicy(ctx, root, "shell_exec", raw)
				if err == nil {
					t.Fatalf("expected mutating %s shell command %s to be blocked", role, raw)
				}
				if !strings.Contains(err.Error(), role+" cannot run mutating shell_exec") {
					t.Fatalf("expected planner shell boundary error for %s, got %v", role, err)
				}
			}
			if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"npm install phaser"}`)); err == nil {
				t.Fatalf("expected package-manager shell mutation to be blocked for %s", role)
			}
		})
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

func TestEngineerLifecycleMoveCommitAllowedWithBacklogRemaining(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship first slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- npm run build
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship first slice
`)
	writePolicyTicket(t, dir, "backlog", "T-002-next-slice.md", `---
id: T-002
title: Ship next slice
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# Ship next slice
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore: seed tickets"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "mv", "docs/tickets/in-progress/T-001-ship.md", "docs/tickets/done/"); err != nil {
		t.Fatalf("git mv done: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		ticketDoneMoveSuccessKey: 1,
	}, ToolState: map[string]string{ticketDoneMoveLastIDKey: "T-001"}})
	if err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"chore(tickets): move T-001 to done"}`)); err != nil {
		t.Fatalf("expected lifecycle move commit to be allowed even with backlog, got %v", err)
	}
}

func TestEngineerMustCommitTicketDoneMoveBeforeClaimingNextTicket(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship first slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- npm run build
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship first slice
`)
	writePolicyTicket(t, dir, "backlog", "T-002-next-slice.md", `---
id: T-002
title: Ship next slice
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# Ship next slice
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore: seed tickets"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "mv", "docs/tickets/in-progress/T-001-ship.md", "docs/tickets/done/"); err != nil {
		t.Fatalf("git mv done: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/backlog/T-002-next-slice.md","docs/tickets/in-progress/"]}`))
	if err == nil {
		t.Fatal("expected next ticket claim to wait for lifecycle commit")
	}
	if !strings.Contains(err.Error(), "lifecycle move") || !strings.Contains(err.Error(), "record job_disposition_record") {
		t.Fatalf("expected lifecycle commit guidance, got %v", err)
	}
}

func TestEngineerCannotClaimNextTicketAfterCompletingTicketInSameRun(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "backlog", "T-002-next-slice.md", `---
id: T-002
title: Ship next slice
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# Ship next slice
`)

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		ticketDoneMoveSuccessKey: 1,
	}, ToolState: map[string]string{ticketDoneMoveLastIDKey: "T-001"}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/backlog/T-002-next-slice.md","docs/tickets/in-progress/"]}`))
	if err == nil {
		t.Fatal("expected next ticket claim to be blocked after completing one ticket")
	}
	if !strings.Contains(err.Error(), "already completed product ticket T-001") ||
		!strings.Contains(err.Error(), "Do not claim another ticket") ||
		!strings.Contains(err.Error(), "next_need qa_review") {
		t.Fatalf("expected one-ticket handoff guidance, got %v", err)
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

func TestEngineerReworkUsesDispatchTicketBeforeOlderDoneTicket(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "done", "T-001-ship-first-slice.md", `---
id: T-001
title: Ship first slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
---

# Ship first slice
`)
	writePolicyTicket(t, dir, "done", "T-003-ship-current-slice.md", `---
id: T-003
title: Ship current slice
work_type: feature
bdd_scenarios:
- F-001-S003
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
---

# Ship current slice
`)

	trigger := `{"type":"dispatch","target_role":"engineer","source_disposition":{"status":"changes_requested","next_need":"implementation_rework","ticket_id":"T-003"}}`
	ctx := WithSession(context.Background(), Session{Role: "engineer", Trigger: trigger, ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected shell_exec to require reopening the dispatch ticket")
	}
	if !strings.Contains(err.Error(), "must reopen T-003") ||
		strings.Contains(err.Error(), "must reopen T-001") {
		t.Fatalf("expected rework guidance for T-003 only, got %v", err)
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

func TestReviewHTTPProbeBeforeServerStartIsProcedureFailure(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["curl","-f","http://localhost:5173/"]}`)
	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{
		ExitCode: 7,
		Stderr:   "curl: (7) Failed to connect to localhost port 5173 after 0 ms: Couldn't connect to server",
	}, nil)
	if session.ToolCounts[validationProcedureFailureKey] != 1 {
		t.Fatalf("expected HTTP probe connection failure to be a validation-procedure failure, got counts %#v", session.ToolCounts)
	}
	if session.ToolCounts[validationCommandFailureKey] != 0 {
		t.Fatalf("expected no product validation failure for pre-server curl, got counts %#v", session.ToolCounts)
	}

	ctx := WithSession(context.Background(), *session)
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["npm","run","dev"],"background":true}`))
	if err != nil {
		t.Fatalf("expected reviewer to recover by starting the dev server after pre-server curl, got %v", err)
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

func TestCTOTicketCreateRequiresBriefCapabilitiesInScenarioSchedule(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithGenericScenarios(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement visible playfield grid",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S001"},
		EndToEndEvidence: "required",
		Body:             "## Requirements\nCreate a basic HTML page, package.json, and src/main.js that render a playfield grid.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "ticket_create", raw)
	if err == nil {
		t.Fatal("expected CTO ticket_create to require product capability scenario coverage")
	}
	if !strings.Contains(err.Error(), "scenario schedule does not cover product brief capabilities") ||
		!strings.Contains(err.Error(), "keyboard movement") ||
		!strings.Contains(err.Error(), "restart") {
		t.Fatalf("expected missing capability guidance, got %v", err)
	}
}

func TestCTOTicketCreateRequiresEarliestUncoveredFeatureScenario(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement keyboard controls",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S002"},
		EndToEndEvidence: "required",
		Body:             "## Requirements\nImplement keyboard controls after the playfield exists.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "ticket_create", raw)
	if err == nil {
		t.Fatal("expected CTO ticket_create to require earliest uncovered scenario")
	}
	if !strings.Contains(err.Error(), "earliest uncovered scenario F-001-S001") {
		t.Fatalf("expected earliest-scenario guidance, got %v", err)
	}
}

func TestCTOTicketCreateAllowsNextScenarioWhenEarlierScenarioAlreadyTicketed(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writePolicyTicket(t, dir, "backlog", "T-001-visible-playfield.md", `---
id: T-001
title: Implement visible playfield
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-001
`)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement falling tetrominoes",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S002"},
		EndToEndEvidence: "required",
		Body:             "## Requirements\nImplement falling tetrominoes after the playfield ticket.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := preToolPolicy(ctx, root, "ticket_create", raw); err != nil {
		t.Fatalf("expected next scenario ticket to pass once earlier scenario is already ticketed, got %v", err)
	}
}

func TestCTOTicketCreateRejectsAlreadyCoveredScenarioGroup(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writePolicyTicket(t, dir, "backlog", "T-001-visible-playfield.md", `---
id: T-001
title: Implement visible playfield
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-001
`)
	writePolicyTicket(t, dir, "backlog", "T-002-falling-tetrominoes.md", `---
id: T-002
title: Implement falling tetrominoes
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-002
`)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement first playable Tetris slice",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S001", "F-001-S002", "F-001-S003"},
		EndToEndEvidence: "required",
		Body:             "## Requirements\nGroup the first three scenarios.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "ticket_create", raw)
	if err == nil {
		t.Fatal("expected already-covered scenarios to be blocked")
	}
	if !strings.Contains(err.Error(), "already-covered scenario(s) F-001-S001, F-001-S002") ||
		!strings.Contains(err.Error(), "Create the next ticket for F-001-S003 only") {
		t.Fatalf("expected covered-scenario guidance, got %v", err)
	}

	raw, err = json.Marshal(ticketCreateArgs{
		Title:            "Implement keyboard controls",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S003"},
		EndToEndEvidence: "required",
		Body:             "## Requirements\nImplement keyboard movement and rotation.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := preToolPolicy(ctx, root, "ticket_create", raw); err != nil {
		t.Fatalf("expected next uncovered scenario alone to pass, got %v", err)
	}
}

func TestCTOTicketCreateAllowsScenarioGroupStartingWithEarliestUncoveredScenario(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement first playable Tetris slice",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S001", "F-001-S002"},
		EndToEndEvidence: "required",
		Body:             "## Requirements\nImplement the visible playfield and first movement controls in one bounded slice.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := preToolPolicy(ctx, root, "ticket_create", raw); err != nil {
		t.Fatalf("expected scenario group with earliest scenario to pass, got %v", err)
	}
}

func TestCTOTicketCreateInfersPendingHandoffScenarios(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writePolicyTicket(t, dir, "backlog", "T-001-visible-playfield.md", `---
id: T-001
title: Implement visible playfield
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-001
`)
	session := Session{
		Role:       "cto-weekly",
		ToolCounts: map[string]int{},
		ToolState: map[string]string{
			ctoHandoffRequiredScenariosKey: "F-001-S002,F-001-S003",
		},
	}
	ctx := WithSession(context.Background(), session)
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement tetromino movement and rotation",
		Priority:         "high",
		WorkType:         "feature",
		EndToEndEvidence: "required",
		Body:             "## Requirements\nImplement keyboard movement and rotation for falling tetrominoes.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := preToolPolicy(ctx, root, "ticket_create", raw); err != nil {
		t.Fatalf("expected pending CTO handoff scenarios to satisfy ticket_create policy, got %v", err)
	}
	if _, err := handleTicketCreate(ctx, root, raw); err != nil {
		t.Fatalf("ticket_create: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-002-implement-tetromino-movement-and-rotation.md"))
	if err != nil {
		t.Fatalf("read created ticket: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "F-001-S002") || !strings.Contains(content, "F-001-S003") {
		t.Fatalf("expected inferred BDD scenarios in created ticket, got:\n%s", content)
	}
}

func TestRecordSessionToolOutcomeTreatsNodeCheckMissingFileAsProcedureFailure(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["node","--check","main.js"]}`)
	result := ToolResult{
		ExitCode: 1,
		Stderr:   "Error: Cannot find module '/tmp/demo/main.js'\n  code: 'MODULE_NOT_FOUND'",
	}

	recordSessionToolOutcome(session, root, "shell_exec", raw, result, nil)

	if session.ToolCounts[validationProcedureFailureKey] != 1 {
		t.Fatalf("expected validation procedure failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] != 0 {
		t.Fatalf("expected no unresolved runtime validation failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolState[validationProcedureFailureCommandKey] == "" {
		t.Fatalf("expected procedure failure command to be recorded, got state %+v", session.ToolState)
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

func TestEngineerFailingBuildGuidanceCompactsRepeatedOutput(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	longOutput := strings.Repeat("vite build failed because Phaser browser runtime was imported from vite.config.js and window is not defined ", 20)
	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			testBuildValidationOutstandingKey: 1,
			buildCommandFailureKey:            1,
		},
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["npm","run","build"]}`,
			testBuildValidationOutputKey:  longOutput,
		},
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["node","scripts/probe.js"]}`))
	if err == nil {
		t.Fatal("expected unresolved build failure to block unrelated shell_exec")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Latest failing output (compact):") || !strings.Contains(msg, "npm") {
		t.Fatalf("expected compact unresolved guidance, got %v", err)
	}
	if strings.Count(msg, "window is not defined") > 1 || len(msg) > 950 {
		t.Fatalf("expected compact repeated-output guidance, len=%d msg=%v", len(msg), msg)
	}
}

func TestEngineerFailingTestAllowsIntegrationTestRepairWrite(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-playfield.md", "# T-001\n")

	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			testBuildValidationOutstandingKey: 1,
			testCommandFailureKey:             1,
		},
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["npm","run","test:integration"]}`,
			testBuildValidationOutputKey:  "jest-environment-jsdom cannot be found",
		},
	}
	ctx := WithSession(context.Background(), session)
	raw := []byte(`{"path":"tests/integration/playfield.test.js","content":"import { describe, expect, test } from '@jest/globals';\n\ndescribe('playfield', () => {\n  test('renders', () => {\n    expect(true).toBe(true);\n  });\n});\n"}`)

	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected same-lane integration test repair write to be allowed, got %v", err)
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

func writeDetailedTetrisBrief(t *testing.T, repoRoot string) {
	t.Helper()
	content := `# Phaser Tetris Demo

Create a complete playable Tetris game using Phaser JS.

The delivered version should include a visible playfield, falling tetrominoes,
keyboard movement and rotation, line clearing, scoring, game over, and restart
behavior.
`
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
}

func writeTetrisFeatureWithGenericScenarios(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Phaser Tetris Demo

## Business Logic

The product must include a visible playfield grid, falling tetrominoes, keyboard movement and rotation, line clearing, scoring, game over, and restart behavior.

## Step-By-Step Behavior

The scenarios below define the minimum viable product behavior.

## Scenario Schedule

1. F-001-S001 - project brief becomes a visible product slice
2. F-001-S002 - user can run or inspect the first product behavior
3. F-001-S003 - product evidence is captured before wider automation work

## Scenarios

### F-001-S001: Project Brief Becomes A Visible Product Slice

Given README or active goals describe the product to build
When the first planning pass runs
Then the active plan and this feature contract name the smallest visible product behavior instead of generic harness operations

### F-001-S002: First Product Behavior Is Runnable Or Inspectable

Given the first product scenario is selected
When Engineer completes the first ordinary product ticket
Then a user can run, open, or inspect the behavior described by the product brief

### F-001-S003: Product Evidence Comes Before Governance Expansion

Given harness telemetry or intervention debt exists
When no visible product behavior has been delivered yet
Then product planning and ordinary product tickets stay ahead of automatic intervention-debt work

## Out of Scope

- None.

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeTetrisFeatureWithFullScenarioSchedule(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Phaser Tetris Demo

## Business Logic

The product must include a visible playfield grid, falling tetrominoes, keyboard movement and rotation, line clearing, scoring, game over, and restart behavior.

## Step-By-Step Behavior

The scenarios below define the minimum viable product behavior.

## Scenario Schedule

1. F-001-S001 - visible playfield grid is rendered
2. F-001-S002 - falling tetrominoes spawn and descend
3. F-001-S003 - keyboard movement and rotation work
4. F-001-S004 - line clearing updates scoring
5. F-001-S005 - game over and restart work

## Scenarios

### F-001-S001: Visible Playfield Grid Is Rendered

Given the game is launched
When the scene starts
Then the visible playfield grid is rendered

### F-001-S002: Falling Tetrominoes Spawn And Descend

Given the playfield is visible
When the game starts
Then falling tetrominoes descend into the playfield

### F-001-S003: Keyboard Movement And Rotation Work

Given a tetromino is falling
When the player presses movement or rotation keys
Then the tetromino responds

### F-001-S004: Line Clearing Updates Scoring

Given a line is complete
When the board resolves the line
Then line clearing increments the score

### F-001-S005: Game Over And Restart Work

Given pieces reach the spawn area
When the game cannot place a new tetromino
Then game over appears and the player can restart

## Out of Scope

- Sound effects

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
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

func writeTetrisFeatureContract(t *testing.T, repoRoot, name string, scenarios []string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	var b strings.Builder
	b.WriteString("# Feature\n\n")
	b.WriteString("## Business Logic\n\n")
	b.WriteString("The product includes visible Tetris gameplay with falling blocks, movement, and scoring.\n\n")
	b.WriteString("## Scenario Schedule\n\n")
	for i, scenario := range scenarios {
		fmt.Fprintf(&b, "%d. %s - product behavior\n", i+1, scenario)
	}
	b.WriteString("\n## Scenarios\n\n")
	for _, scenario := range scenarios {
		fmt.Fprintf(&b, "### %s: Product Behavior\n\nGiven the game is running\nWhen the user plays\nThen product behavior is visible\n\n", scenario)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
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
