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
	"strconv"
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

func TestCTODispositionAllowsCoveredBatchAfterDuplicateTicketCreateFailure(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writePolicyTicket(t, dir, "backlog", "T-001-visible-playfield.md", `---
id: T-001
title: Visible playfield
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
title: Falling tetrominoes
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
	writePolicyTicket(t, dir, "backlog", "T-003-keyboard-controls.md", `---
id: T-003
title: Keyboard controls
work_type: feature
bdd_scenarios:
- F-001-S003
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-003
`)
	ctx := WithSession(context.Background(), Session{
		Role:       "cto-weekly",
		ToolCounts: map[string]int{ticketCreationOutstandingFailureKey: 1},
	})

	err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`))
	if err != nil {
		t.Fatalf("expected covered CTO ticket batch to permit Engineer handoff after duplicate ticket_create failure, got %v", err)
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

func TestReviewShellExecPolicyAllowsTrackedBackgroundKill(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	t.Cleanup(KillBackgroundProcs)

	started, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","sleep 5"],"background":true}`))
	require.NoError(t, err)
	pid := backgroundPIDFromOutput(t, started.Output)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})

	raw := []byte(`{"argv":["kill","-TERM","` + strconv.Itoa(pid) + `"]}`)
	if err := preToolPolicy(ctx, root, "shell_exec", raw); err != nil {
		t.Fatalf("expected QA to stop tracked background PID, got %v", err)
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

func TestReviewTerminalEvidenceIgnoresBackgroundServerStart(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}}
	raw := json.RawMessage(`{"argv":["python3","-m","http.server","8080","--bind","127.0.0.1"],"background":true}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	if ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence to wait for a concrete probe, not just background server startup")
	}

	probeRaw := json.RawMessage(`{"argv":["curl","-fsS","http://127.0.0.1:8080/"]}`)
	recordSessionToolOutcome(session, root, "shell_exec", probeRaw, ToolResult{ExitCode: 0}, nil)
	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence after static HTTP probe, read evidence, and docsync_audit")
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

func TestQAChangesRequestedBlocksFoundationValidationIssueRoutedToEngineer(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	writeValidPhaserSource(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey:   1,
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
		"tool:file_read:success":      1,
		"tool:docsync_audit:success":  1,
	}})

	raw := json.RawMessage(`{
  "status":"changes_requested",
  "ticket_id":"T-001",
  "next_need":"implementation_rework",
  "reason":"Browser smoke test validation error; implementation is correct but the test should look for Phaser.Game differently.",
  "feedback":{"for_role":"engineer","summary":"Browser smoke test validation error","requested_change":"The implementation is correct; the test should be corrected."}
}`)
	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA rework routing to be blocked for foundation validation issue")
	}
	if !strings.Contains(err.Error(), "foundation validation/test wording issue") ||
		!strings.Contains(err.Error(), "Do not send implementation_rework") {
		t.Fatalf("expected foundation-validation routing guidance, got %v", err)
	}
}

func TestQAChangesRequestedBlocksDevServerSetupFailureRoutedToEngineer(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	writeValidPhaserSource(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey:  1,
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}})

	raw := json.RawMessage(`{
  "status":"changes_requested",
  "ticket_id":"T-001",
  "next_need":"implementation_rework",
  "reason":"Build succeeded but browser smoke test failed due to server not running. The curl test to localhost:5173 failed.",
  "feedback":{"for_role":"engineer","summary":"Build succeeded but smoke tests failed","requested_change":"Ensure the development server is running during testing and verify the browser smoke test works."}
}`)
	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA rework routing to be blocked for QA-owned dev-server setup failure")
	}
	if !strings.Contains(err.Error(), "foundation validation/test wording issue") ||
		!strings.Contains(err.Error(), "Do not send implementation_rework") {
		t.Fatalf("expected foundation-validation routing guidance, got %v", err)
	}
}

func TestCOOCompletionRequiresProductSpecificFeatureContract(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	if err := os.WriteFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"), []byte("# F-001\n\n## Business Logic\n\nThis starter contract is seeded from README and active goals. Product rules, workflow branches, state transitions belong here.\n\n## Step-By-Step Behavior\n\nReplace placeholder nouns with real product terms.\n"), 0o644); err != nil {
		t.Fatalf("write starter feature: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`))
	if err == nil {
		t.Fatal("expected COO completion to require product-specific contract rewrite")
	}
	if !strings.Contains(err.Error(), "starter-placeholder") || !strings.Contains(err.Error(), "product-specific") {
		t.Fatalf("expected feature specificity guidance, got %v", err)
	}
}

func TestCOOCompletionAllowsProductSpecificContractWithBusinessLogicLanguage(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001, G-002
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris: Create a complete playable Tetris game using Phaser JS.

## Business Logic

This feature defines the core gameplay mechanics for a playable Tetris game.
Product rules, workflow branches, state transitions, validations, permissions,
scoring decisions, routing rules, and user-visible outcomes for the Tetris game
are defined here before the first visible product behavior is delivered.

## Step-By-Step Behavior

1. A visible playfield is rendered
2. Tetrominoes fall at a constant rate
3. Player can move tetrominoes left/right with keyboard
4. Player can rotate tetrominoes with keyboard
5. Lines clear when completed
6. Score increases when lines are cleared
7. Game over state triggers when new tetrominoes cannot enter the playfield
8. Game can be restarted after game over

## Scenario Schedule

1. F-001-S001 - Playfield is visible and properly sized
2. F-001-S002 - Tetrominoes fall at a constant rate
3. F-001-S003 - Player can move tetrominoes with keyboard controls
4. F-001-S004 - Player can rotate tetrominoes with keyboard controls
5. F-001-S005 - Lines clear when completed
6. F-001-S006 - Score increases when lines are cleared
7. F-001-S007 - Game over state triggers when tetrominoes cannot enter the playfield
8. F-001-S008 - Game can be restarted after game over

## Scenarios

### F-001-S001: Playfield Is Visible And Properly Sized

Given a user wants to play Tetris
When the game initializes
Then a visible playfield is displayed with correct dimensions

### F-001-S002: Tetrominoes Fall At A Constant Rate

Given a visible playfield exists
When the game starts
Then tetrominoes begin falling at a constant rate

### F-001-S003: Player Can Move Tetrominoes With Keyboard Controls

Given a falling tetromino exists
When the player presses left or right arrow keys
Then the tetromino moves within the playfield boundaries

### F-001-S004: Player Can Rotate Tetrominoes With Keyboard Controls

Given a falling tetromino exists
When the player presses rotate
Then the tetromino rotates within the playfield boundaries

### F-001-S005: Lines Clear When Completed

Given a tetromino has settled
When a horizontal line becomes complete
Then the line clears and blocks above it move down

### F-001-S006: Score Increases When Lines Are Cleared

Given lines have been cleared
When the game updates the score
Then the score increases based on the number of lines cleared

### F-001-S007: Game Over State Triggers When Tetrominoes Cannot Enter The Playfield

Given tetrominoes are falling
When a new tetromino cannot enter the playfield
Then the game transitions to game over state

### F-001-S008: Game Can Be Restarted After Game Over

Given the game is in game over state
When the player restarts the game
Then the game resets and play begins again

## Descoped Scenarios

None.
`
	if err := os.WriteFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write product feature: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected product-specific contract to pass despite business-logic glossary language, got %v", err)
	}
}

func TestCOOCompletionRequiresBriefCapabilitiesInScenarioSchedule(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writeTetrisFeatureWithGenericScenarios(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`))
	if err == nil {
		t.Fatal("expected COO completion to require scenario coverage for explicit brief capabilities")
	}
	if !strings.Contains(err.Error(), "scenario schedule does not cover product brief capabilities") ||
		!strings.Contains(err.Error(), "falling tetrominoes") ||
		!strings.Contains(err.Error(), "line clearing") {
		t.Fatalf("expected missing capability guidance, got %v", err)
	}
}

func TestCapabilityCoverageTreatsPiecesAsTetrominoesForLocking(t *testing.T) {
	t.Parallel()
	surface := `
## Scenario Schedule

1. F-001-S006 - Tetrominoes Lock Into Stack On Contact

### F-001-S006: Tetrominoes Lock Into Stack On Contact

Given a tetromino is falling
When it makes contact with the bottom or another piece
Then the tetromino locks into the stack and a new tetromino begins
`
	if !capabilityPhraseCovered(surface, "lock pieces into the stack") {
		t.Fatal("expected tetromino locking language to cover piece-locking capability")
	}
}

func TestCOOCompletionRejectsCollapsedProductCapabilityScenario(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBriefWithEvidenceSentence(t, dir)
	writeTetrisFeatureWithCollapsedProductScenario(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`))
	if err == nil {
		t.Fatal("expected COO completion to reject a broad runnable scenario that hides product capabilities")
	}
	if !strings.Contains(err.Error(), "scenario outline does not break out product brief capabilities") ||
		!strings.Contains(err.Error(), "falling tetrominoes") ||
		!strings.Contains(err.Error(), "line clearing") {
		t.Fatalf("expected product capability outline guidance, got %v", err)
	}
}

func TestCOOCompletionIgnoresValidationEvidenceAndAcceptsControlSynonym(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBriefWithEvidenceSentence(t, dir)
	writeTetrisFeatureWithControlsScenarioNoSmokeScenario(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected validation evidence prose to be ignored as product scope and keyboard controls to cover movement, got %v", err)
	}
}

func TestCOOCompletionDoesNotTreatMobileTouchControlsAsMovementDescoped(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writeTetrisFeatureWithMobileTouchControlsOutOfScope(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if capabilityPhraseCovered("mobile touch controls", "move") {
		t.Fatal("expected mobile touch controls not to cover or descope basic movement")
	}
	if !capabilityPhraseCovered("keyboard controls work", "move") {
		t.Fatal("expected keyboard controls to remain movement coverage")
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected alternate mobile/touch input scope not to block covered keyboard movement, got %v", err)
	}
}

func TestCOOCompletionIgnoresCapabilityCategoryPrefix(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	visionDir := filepath.Join(dir, "docs", "product-specs")
	if err := os.MkdirAll(visionDir, 0o755); err != nil {
		t.Fatalf("mkdir vision dir: %v", err)
	}
	vision := `# Vision

The target must include all core Tetris mechanics: visible playfield grid,
falling tetrominoes, keyboard movement and rotation, line clearing, scoring,
game over, and restart.
`
	if err := os.WriteFile(filepath.Join(visionDir, "vision.md"), []byte(vision), 0o644); err != nil {
		t.Fatalf("write vision: %v", err)
	}
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected category-prefixed mechanics list to pass capability coverage, got %v", err)
	}
}

func TestCOOCompletionIgnoresGenericGameplayMechanicsGoalHeading(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "docs", "goals"), 0o755); err != nil {
		t.Fatalf("mkdir goals: %v", err)
	}
	activeGoals := `# Active Goals

## G-002: Implement core Tetris gameplay mechanics

- Product Brief: Core Tetris gameplay mechanics for Demo Tetris 56
- Hypothesis: Implementing core Tetris mechanics will deliver a playable game.
- Success Evidence: The game allows users to move, rotate, and place tetrominoes; clear lines; track score; and end game when stack fills.
`
	if err := os.WriteFile(filepath.Join(dir, "docs", "goals", "active.md"), []byte(activeGoals), 0o644); err != nil {
		t.Fatalf("write active goals: %v", err)
	}
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	required := strings.Join(projectBriefCapabilityPhrases(root), ", ")
	if strings.Contains(required, "core tetris gameplay mechanics") {
		t.Fatalf("expected generic gameplay heading to be ignored, got %q", required)
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected concrete mechanics to satisfy capability coverage without generic heading phrase, got %v", err)
	}
}

func TestCOOCompletionRejectsBriefCapabilitiesInOutOfScope(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writeTetrisFeatureWithRequiredCapabilitiesOutOfScope(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`))
	if err == nil {
		t.Fatal("expected COO completion to reject required brief capabilities in Out of Scope")
	}
	if !strings.Contains(err.Error(), "Out of Scope") ||
		!strings.Contains(err.Error(), "game over") ||
		!strings.Contains(err.Error(), "restart") {
		t.Fatalf("expected Out of Scope capability guidance, got %v", err)
	}
}

func TestCOOCompletionAllowsAdvancedOutOfScopeQualifierForCoveredBasicCapabilities(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writeTetrisFeatureWithAdvancedOnlyOutOfScope(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected advanced-only out-of-scope text to leave covered basic capabilities in scope, got %v", err)
	}
}

func TestCOOCompletionIgnoresActiveGoalNonGoalsAndOperationalConstraints(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	writeDemoTetrisActiveGoalsWithNonGoals(t, dir)
	writeDemoTetrisFeatureWithDescopedAdvancedMechanics(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	required := strings.Join(projectBriefCapabilityPhrases(root), ", ")
	for _, unexpected := range []string{"build", "hold piece", "install", "player open the game locally"} {
		if strings.Contains(required, unexpected) {
			t.Fatalf("expected capability extraction to ignore %q from operational constraints/non-goals, got %q", unexpected, required)
		}
	}
	for _, expected := range []string{"playfield", "keyboard", "restart"} {
		if !strings.Contains(required, expected) {
			t.Fatalf("expected capability extraction to retain product capability %q, got %q", expected, required)
		}
	}
	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected operational build scripts and non-goal hold piece text not to block COO handoff, got %v", err)
	}
}

func TestCOOCompletionAcceptsExpandedDemoTetrisScenarioSchedule(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	writeExpandedDemoTetrisFeature(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected expanded product scenario schedule to cover explicit brief capabilities, got %v", err)
	}
}

func TestCOOCompletionUsesActiveProductFeatureWhenF001IsSuperseded(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "docs", "features"), 0o755); err != nil {
		t.Fatalf("mkdir features: %v", err)
	}
	f001 := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: superseded

## Business Logic

This starter contract is superseded by F-002.

## Scenario Schedule

1. F-001-S001 - Project brief becomes a visible product slice

## Scenarios

### F-001-S001: Project Brief Becomes A Visible Product Slice

Given a product brief exists
When planning runs
Then product work is selected
`
	if err := os.WriteFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"), []byte(f001), 0o644); err != nil {
		t.Fatalf("write F-001: %v", err)
	}
	f002 := `# F-002: Core Tetris Gameplay

- Feature ID: F-002
- Goals: G-001
- Status: active

## Business Logic

The first visible product slice covers browser access, playfield rendering,
falling tetrominoes, keyboard movement, rotation, piece locking, line clearing,
score tracking, game over, and restart behavior.

## Scenario Schedule

1. F-002-S001 - Player can open the game locally and see the Tetris playfield
2. F-002-S002 - Falling tetrominoes move and rotate with the keyboard
3. F-002-S003 - Pieces lock into the stack and clear full lines for score
4. F-002-S004 - Game over appears when the stack fills and restart begins another round

## Scenarios

### F-002-S001: Player Can Open The Game Locally And See The Tetris Playfield

Given the app is installed locally
When the player opens the game in a browser
Then the Tetris playfield is visible

### F-002-S002: Falling Tetrominoes Move And Rotate With The Keyboard

Given a tetromino is falling
When the player presses keyboard controls
Then the tetromino moves and rotates

### F-002-S003: Pieces Lock Into The Stack And Clear Full Lines For Score

Given pieces are falling
When they land and full lines form
Then pieces lock into the stack, lines clear, and score increases

### F-002-S004: Game Over And Restart

Given the stack fills
When game over is reached
Then the player can restart for another round

## Descoped Scenarios

None.
`
	if err := os.WriteFile(filepath.Join(dir, "docs", "features", "F-002-core-tetris-gameplay.md"), []byte(f002), 0o644); err != nil {
		t.Fatalf("write F-002: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected active F-002 to satisfy coverage while F-001 is superseded, got %v", err)
	}
	ids := strings.Join(featureContractIDs(root), ",")
	if strings.Contains(ids, "F-001") || !strings.Contains(ids, "F-002") {
		t.Fatalf("expected active feature IDs to exclude superseded F-001 and include F-002, got %q", ids)
	}
}

func TestCOOCompletionAcceptsGroupedGameOverScenarioSchedule(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	writeGroupedDemoTetrisFeature(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected grouped game-over/restart scenario schedule to cover explicit brief capabilities, got %v", err)
	}
}

func TestCOOCompletionAcceptsRefinedDemoTetrisScenarioSchedule(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	writeRefinedDemoTetrisFeature(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected refined product scenario schedule to cover explicit brief capabilities, got %v", err)
	}
}

func TestCOOCompletionAllowsHighScorePersistenceOutOfScope(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	writeDemoTetrisFeatureWithHighScoreOutOfScope(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected high-score persistence out-of-scope text to leave basic score tracking in scope, got %v", err)
	}
}

func TestCOOCompletionAllowsOutOfScopeIntroAndAdvancedScoringSystems(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisBriefWithOperationalPreference(t, dir)
	writeDemoTetrisFeatureWithOutOfScopeIntroAndAdvancedScoring(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected out-of-scope intro text and advanced scoring systems not to descope basic line clearing/scoring, got %v", err)
	}
}

func TestCOOCompletionAllowsAnimationOnlyOutOfScopeForCoveredGameplay(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDemoTetrisExplicitListBrief(t, dir)
	writeDemoTetrisFeatureWithAnimationPolishOutOfScope(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected animation-polish out-of-scope text not to descope falling pieces or line clearing, got %v", err)
	}
}

func TestCOOCompletionAcceptsOutcomeGlueWhenCapabilitiesAreBrokenOut(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeGenericOutcomeBrief(t, dir)
	writeGenericFeatureWithOutcomeScenarios(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected readable/usable outcome phrasing not to block broken-out product capabilities, got %v", err)
	}
}

func TestCOOCompletionIgnoresProjectNameTokensFromBriefHeadings(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	readme := `# Orion Ledger

The first slice must include Orion Ledger transaction import, ledger balance reconciliation, and report export.
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	featurePath := filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(featurePath), 0o755); err != nil {
		t.Fatalf("mkdir feature dir: %v", err)
	}
	feature := `# F-001: Product Walking Skeleton

## Business Logic

The product must include transaction import, balance reconciliation, and report export.

## Scenario Schedule

1. F-001-S001 - Import transactions from a local file
2. F-001-S002 - Reconcile balances
3. F-001-S003 - Export a report

## Scenarios

### F-001-S001: Import Transactions From A Local File

Given a local file exists
When the user imports it
Then transactions are visible

### F-001-S002: Reconcile Balances

Given imported transactions exist
When the user reconciles the account
Then the balance reconciliation is visible

### F-001-S003: Export A Report

Given reconciled data exists
When the user exports a report
Then a report file is produced

## Out of Scope

- Bank integrations

## Descoped Scenarios

None.
`
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	required := strings.Join(projectBriefCapabilityPhrases(root), ", ")
	if strings.Contains(required, "orion") || strings.Contains(required, "ledger transaction") {
		t.Fatalf("expected product label tokens to be stripped from required capabilities, got %q", required)
	}
	ctx := WithSession(context.Background(), Session{Role: "coo", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"ticket_breakdown","suggested_role":"cto-weekly"}`)); err != nil {
		t.Fatalf("expected generic product label tokens not to block capability coverage, got %v", err)
	}
}

func TestCapabilityMatchingIgnoresIncludingAndDetectionGlue(t *testing.T) {
	t.Parallel()

	if !capabilityPhraseCovered("visible board grid is displayed", "core product including visible grid") {
		t.Fatal("expected including/core product glue not to block visible grid coverage")
	}
	if !capabilityPhraseCovered("game over is detected when stack fills the playfield", "game over detection") {
		t.Fatal("expected detection glue not to block game-over coverage")
	}
	if !capabilityPhraseCovered("game over is detected when stack fills playfield", "show game over when the stack fills") {
		t.Fatal("expected show/display glue not to block game-over coverage")
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

func TestCTOCompletionRequiresEarlyScenarioTicketBatch(t *testing.T) {
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

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`))
	if err == nil {
		t.Fatal("expected CTO implementation handoff to require an early ticket batch")
	}
	if !strings.Contains(err.Error(), "small product backlog batch") || !strings.Contains(err.Error(), "F-001-S002") {
		t.Fatalf("expected ticket batch guidance, got %v", err)
	}
}

func TestCTOCompletionUsesActivePlanFeatureForEarlyScenarioBatch(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlanForFeature(t, dir, "F-002", "F-002-S001, F-002-S002, F-002-S003", "F-002-S002")
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writeTetrisFeatureContract(t, dir, "F-002-core-mechanics.md", []string{
		"F-002-S001",
		"F-002-S002",
		"F-002-S003",
		"F-002-S004",
	})
	writePolicyTicket(t, dir, "done", "T-001-board-setup.md", `---
id: T-001
title: Implement board setup
work_type: feature
bdd_scenarios:
- F-002-S001
end_to_end_evidence: required
evidence_links: []
verified_by: engineer
blocker: none
blocked_by: []
---

# T-001
`)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`))
	if err == nil {
		t.Fatal("expected CTO handoff to require the active plan feature's next scenarios")
	}
	if !strings.Contains(err.Error(), "F-002") ||
		!strings.Contains(err.Error(), "F-002-S002") ||
		strings.Contains(err.Error(), "F-001") {
		t.Fatalf("expected active-plan feature guidance, got %v", err)
	}
}

func TestCTOCompletionIgnoresUnselectedStarterFeatureWhenActivePlanBatchIsCovered(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlanForFeature(t, dir, "F-002", "F-002-S001, F-002-S002, F-002-S003", "F-002-S002")
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writeTetrisFeatureContract(t, dir, "F-002-core-mechanics.md", []string{
		"F-002-S001",
		"F-002-S002",
		"F-002-S003",
	})
	writePolicyTicket(t, dir, "backlog", "T-001-active-plan-batch.md", `---
id: T-001
title: Implement active plan batch
work_type: feature
bdd_scenarios:
- F-002-S001
- F-002-S002
- F-002-S003
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-001
`)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`)); err != nil {
		t.Fatalf("expected active-plan batch to permit Engineer handoff despite unselected starter feature, got %v", err)
	}
}

func TestCTOCompletionAllowsGroupedEarlyScenarioCoverage(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBrief(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithFullScenarioSchedule(t, dir)
	writePolicyTicket(t, dir, "backlog", "T-001-first-playable-slice.md", `---
id: T-001
title: Implement first playable slice
work_type: feature
bdd_scenarios:
- F-001-S001
- F-001-S002
- F-001-S003
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-001
`)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`)); err != nil {
		t.Fatalf("expected CTO handoff to pass once early scenarios are ticketed, got %v", err)
	}
}

func TestCTOCompletionDoesNotRequireProcessOnlyEvidenceScenario(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writeDetailedTetrisBriefWithEvidenceSentence(t, dir)
	writePolicyPlan(t, dir)
	writeTetrisFeatureWithCollapsedProductScenario(t, dir)
	writePolicyTicket(t, dir, "backlog", "T-001-first-playable-slice.md", `---
id: T-001
title: Implement first playable Tetris slice
work_type: feature
bdd_scenarios:
- F-001-S001
- F-001-S002
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# T-001
`)
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"completed","next_need":"implementation","suggested_role":"engineer"}`)); err != nil {
		t.Fatalf("expected CTO handoff to ignore process-only evidence scenario once product scenario is ticketed, got %v", err)
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

func writeDetailedTetrisBriefWithEvidenceSentence(t *testing.T, repoRoot string) {
	t.Helper()
	content := `# Phaser Tetris Demo

Create Tetris using Phaser JS.

The product must include a visible playfield grid, falling tetrominoes,
keyboard movement and rotation, line clearing, scoring, game over, and restart
behavior. It should run locally in a browser, use a local Phaser dependency,
and include enough build or smoke evidence to prove the game mounts and plays.
`
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
}

func writeDemoTetrisBriefWithOperationalPreference(t *testing.T, repoRoot string) {
	t.Helper()
	content := `# Demo Tetris

Create a browser game: Tetris using Phaser JS.

The finished project should let a player open the game locally, see a Tetris playfield, move and rotate falling tetrominoes with the keyboard, clear full lines, track score, reach game over when the stack fills, and restart for another round. Prefer a small Phaser/Vite implementation with clear npm scripts for install, build, and local validation.
`
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
}

func writeDemoTetrisActiveGoalsWithNonGoals(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "goals", "active.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir goals: %v", err)
	}
	content := `# Active Goals

## G-001: Deliver first visible product slice

- ID: G-001
- Status: active
- Scope:
  - Must include a playable Tetris playfield (10x20 grid)
  - Must support keyboard controls for moving and rotating tetrominoes
  - Must implement basic game mechanics: falling blocks, line clearing, scoring
  - Must detect and handle game over conditions
  - Must allow restart for another round
  - Must be built with Phaser JS and Vite
  - Must have clear npm scripts for install, build, and local validation
- Non-Goals:
  - Full feature set of classic Tetris (no advanced features like hard drop, hold piece, etc.)
  - Advanced scoring system beyond basic line clearing points
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write active goals: %v", err)
	}
}

func writeDemoTetrisFeatureWithDescopedAdvancedMechanics(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO

## Business Logic

This feature defines the minimum viable product for a Phaser Tetris game.

## Step-By-Step Behavior

The scenarios below define the concrete product capabilities required for the first visible product slice.

## Scenario Schedule

1. F-001-S001 - playable Tetris playfield 10x20 grid is displayed
2. F-001-S002 - keyboard controls move and rotate tetrominoes
3. F-001-S003 - falling blocks, lines clear, and score tracking work
4. F-001-S004 - game over conditions are handled
5. F-001-S005 - restart functionality works

## Scenarios

### F-001-S001: Playable Tetris Playfield 10x20 Grid Is Displayed

Given the player opens the game
When the game initializes
Then a 10x20 playfield is displayed

### F-001-S002: Keyboard Controls Move And Rotate Tetrominoes

Given a tetromino is falling
When the player presses movement or rotation keys
Then the tetromino moves or rotates inside the playfield

### F-001-S003: Falling Blocks, Lines Clear, And Score Tracking Work

Given blocks are falling
When a line is completed
Then the line clears and the score increases

### F-001-S004: Game Over Conditions Are Handled

Given the stack reaches the spawn area
When a new piece cannot spawn
Then the game shows a game over state

### F-001-S005: Restart Functionality Works

Given the game is over
When the player restarts
Then the board resets and a new round starts

## Out of Scope

- Building every product feature in the first slice
- Treating harness self-improvement as the first target product feature
- Closing feature tickets without evidence

## Descoped Scenarios

### F-001-S998: Full Feature Set Of Classic Tetris

Given the need to maintain scope for the first product slice
When evaluating advanced Tetris features
Then features like hard drop, hold piece, and other advanced mechanics are deliberately excluded from the first implementation to maintain focus on core gameplay mechanics

### F-001-S999: Advanced Scoring System

Given the need to maintain scope for the first product slice
When evaluating scoring mechanics
Then advanced scoring beyond basic line clearing points is deliberately excluded from the first implementation
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeExpandedDemoTetrisFeature(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris: Create a browser game: Tetris using Phaser JS.

## Business Logic

This feature defines the minimal viable product behavior for Demo Tetris. The core Tetris mechanics must be implemented to provide a functional user experience with:
- A visible playfield grid
- Tetromino piece movement and rotation
- Line clearing mechanics
- Score tracking
- Game over conditions
- Keyboard controls for user interaction
- Restart functionality for another round

## Step-By-Step Behavior

The scenarios below define the specific product capabilities that must be implemented for the first visible product slice of Demo Tetris.

## Scenario Schedule

1. F-001-S001 — Project brief becomes a visible product slice (Completed)
2. F-001-S002 — User can see a Tetris playfield
3. F-001-S003 — User can move falling tetrominoes with keyboard controls
4. F-001-S004 — User can rotate falling tetrominoes with keyboard controls
5. F-001-S005 — User can clear full lines
6. F-001-S006 — User can track score
7. F-001-S007 — User can reach game over when stack fills
8. F-001-S008 — User can restart for another round
9. F-001-S009 — All core Tetris mechanics work together in a runnable product

## Scenarios

### F-001-S001: Project Brief Becomes A Visible Product Slice

Given README or active goals describe the product to build
When the first planning pass runs
Then the active plan and this feature contract name the smallest visible product behavior for Demo Tetris instead of generic harness operations

### F-001-S002: User Can See A Tetris Playfield

Given the product is being implemented
When the first product behavior is created
Then a user can see a visible Tetris playfield grid (10x20 grid)

### F-001-S003: User Can Move Falling Tetrominoes With Keyboard Controls

Given a Tetris playfield is visible
When a tetromino is falling
Then the user can move it left/right with arrow keys

### F-001-S004: User Can Rotate Falling Tetrominoes With Keyboard Controls

Given a Tetris playfield is visible and a tetromino is falling
When the user presses the rotation key
Then the tetromino rotates in place

### F-001-S005: User Can Clear Full Lines

Given a Tetris playfield is visible and tetrominoes are placed
When full lines are completed
Then those lines are cleared and the playfield is updated

### F-001-S006: User Can Track Score

Given a Tetris playfield is visible
When lines are cleared
Then the score increases appropriately

### F-001-S007: User Can Reach Game Over When Stack Fills

Given a Tetris playfield is visible
When new tetrominoes can no longer enter the playfield
Then the game ends with a game over state

### F-001-S008: User Can Restart For Another Round

Given a game over state
When the user restarts
Then a new game begins with a fresh playfield

### F-001-S009: All Core Tetris Mechanics Work Together

Given all individual mechanics are implemented
When the user runs the product
Then all core Tetris mechanics work together in a playable experience

## Out of Scope

- Building every product feature in the first slice
- Treating harness self-improvement as the first target product feature
- Closing feature tickets without evidence
- Advanced game features like level progression, next piece preview, or sound effects

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeGroupedDemoTetrisFeature(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris: Create a browser game: Tetris using Phaser JS.

## Business Logic

This feature defines the minimum viable product behavior for a Tetris game built with Phaser JS. The product must allow a user to:
- Open the game in a browser
- See a Tetris playfield
- Move and rotate falling tetrominoes with keyboard controls
- Clear full lines
- Track score
- Reach game over when the stack fills
- Restart for another round

## Step-By-Step Behavior

The scenarios below define the specific, product-focused behavior needed for the first working Tetris implementation. These scenarios are tied to runnable or inspectable evidence.

## Scenario Schedule

1. F-001-S001 - User can open Tetris game in browser
2. F-001-S002 - User can see Tetris playfield with falling tetrominoes
3. F-001-S003 - User can move and rotate falling tetrominoes with keyboard
4. F-001-S004 - User can clear full lines and track score
5. F-001-S005 - Game ends when stack fills and user can restart

## Scenarios

### F-001-S001: User can open Tetris game in browser

Given the repository is set up with Phaser/Vite
When a user runs the local development server
Then the Tetris game loads in the browser

### F-001-S002: User can see Tetris playfield with falling tetrominoes

Given the game has loaded in the browser
When the game starts
Then a Tetris playfield is visible with a falling tetromino

### F-001-S003: User can move and rotate falling tetrominoes with keyboard

Given the game is running and a tetromino is falling
When the user presses movement or rotation keys
Then the tetromino moves or rotates accordingly

### F-001-S004: User can clear full lines and track score

Given the game is running and a tetromino has landed
When full horizontal lines are completed
Then those lines are cleared and score is incremented

### F-001-S005: Game ends when stack fills and user can restart

Given the game is running and tetrominoes are stacking
When the stack reaches the top of the playfield
Then the game ends and shows game over screen
And the user can restart the game

## Out of Scope

- Building every product feature in the first slice
- Treating harness self-improvement as the first target product feature
- Closing feature tickets without evidence
- Advanced game features like level progression or next piece preview

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeRefinedDemoTetrisFeature(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris: Create a browser game: Tetris using Phaser JS.

## Business Logic

This feature defines the minimum viable product walking skeleton for Demo Tetris. The implementation must include:
- A playable Tetris game using Phaser JS
- A visible playfield (10x20 grid)
- Keyboard controls for moving and rotating tetrominoes
- Basic game mechanics (piece movement, line clearing)
- Game over detection when stack reaches top
- Restart functionality
- Score tracking
- A build and run process that works locally

## Step-By-Step Behavior

The scenarios below define the concrete behavior for the first product slice. Each scenario must have runnable or inspectable evidence.

## Scenario Schedule

1. F-001-S001 - Project brief is implemented as visible product slice
2. F-001-S002 - First product behavior is runnable or inspectable with basic mechanics
3. F-001-S003 - Playfield is visible and keyboard controls work
4. F-001-S004 - Tetrominoes move and rotate with keyboard
5. F-001-S005 - Full lines are cleared and score is tracked
6. F-001-S006 - Game over and restart functionality
7. F-001-S007 - Build and run process works locally

## Scenarios

### F-001-S001: Project Brief Is Implemented As Visible Product Slice

Given the project brief describes a browser-based Tetris game using Phaser JS
When the first implementation pass runs
Then the repository contains a working Tetris implementation with Phaser JS that can be built and run in a browser

### F-001-S002: First Product Behavior Is Runnable Or Inspectable With Basic Mechanics

Given the first product scenario is selected and a basic Tetris implementation exists
When the engineer completes the first ordinary product ticket
Then a user can run, open, or inspect a playable Tetris game in a browser with:
- A visible 10x20 playfield
- Basic piece falling mechanics
- Score tracking
- Game over detection

### F-001-S003: Playfield Is Visible And Keyboard Controls Work

Given the game has a basic implementation
When the engineer completes the next ticket
Then the user can see a visible playfield and control tetrominoes with keyboard

### F-001-S004: Tetrominoes Move And Rotate With Keyboard

Given the basic game is running
When the engineer completes the next ticket
Then tetrominoes can be moved left/right and rotated using keyboard controls

### F-001-S005: Full Lines Are Cleared And Score Is Tracked

Given tetrominoes can be moved and rotated
When the engineer completes the next ticket
Then full lines are cleared from the playfield and score is updated

### F-001-S006: Game Over And Restart Functionality

Given lines can be cleared and score tracked
When the engineer completes the next ticket
Then game over is detected when stack reaches top and restart functionality works

### F-001-S007: Build And Run Process Works Locally

Given all game mechanics are implemented
When the engineer completes the final ticket in the sequence
Then a user can install, build, and run the game locally with clear npm scripts

## Out of Scope

- Building every product feature in the first slice
- Treating harness self-improvement as the first target product feature
- Closing feature tickets without evidence
- Complex game logic beyond the minimum viable implementation

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeDemoTetrisFeatureWithHighScoreOutOfScope(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris: Create a browser game: Tetris using Phaser JS.

## Business Logic

This feature defines the minimum viable product behavior for Demo Tetris. The implementation must include visible playfield, keyboard controls, line clearing, score tracking, game over, and restart behavior.

## Step-By-Step Behavior

The scenarios below define the concrete product capabilities needed for the first visible product slice.

## Scenario Schedule

1. F-001-S001 - User can access the game locally via browser
2. F-001-S002 - Game field displays with falling tetrominoes
3. F-001-S003 - Player can move tetrominoes with keyboard
4. F-001-S004 - Player can rotate tetrominoes with keyboard
5. F-001-S005 - Player can clear full lines and score points
6. F-001-S006 - Game ends when stack reaches top of playfield
7. F-001-S007 - Player can restart game after game over

## Scenarios

### F-001-S001: User Can Access The Game Locally Via Browser

Given a user has installed dependencies and run the local server
When they open a browser to the local game URL
Then they see the game interface with a playable field

### F-001-S002: Game Field Displays With Falling Tetrominoes

Given the game is running
When the game starts
Then a 10x20 game grid is displayed with a falling tetromino

### F-001-S003: Player Can Move Tetrominoes With Keyboard

Given a tetromino is falling
When the user presses left/right arrow keys
Then the tetromino moves horizontally in the specified direction

### F-001-S004: Player Can Rotate Tetrominoes With Keyboard

Given a tetromino is falling
When the user presses the up arrow key or spacebar
Then the tetromino rotates 90 degrees clockwise

### F-001-S005: Player Can Clear Full Lines And Score Points

Given a tetromino has landed and formed a complete horizontal line
When that line is cleared
Then the score increases and the remaining tetrominoes above shift down

### F-001-S006: Game Ends When Stack Reaches Top Of Playfield

Given a tetromino is falling
When it lands and the playfield is full
Then the game ends with a game over screen

### F-001-S007: Player Can Restart Game After Game Over

Given the game has ended
When the player presses a restart key or button
Then the game resets to initial state and starts a new round

## Out of Scope

- Advanced features like level progression, next piece preview, or sound
- Sound effects or animations beyond basic movement
- Multiplayer functionality
- High score tracking or persistence
- Complex UI elements beyond the game grid and score display

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeDemoTetrisFeatureWithOutOfScopeIntroAndAdvancedScoring(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris 61: Create a playable Tetris game using Phaser JS.

## Business Logic

This feature implements the first visible product slice of Demo Tetris. It includes all core gameplay mechanics required to deliver a functional Tetris game that can be run and inspected in a browser. The feature focuses on the minimal set of behaviors that constitute a playable Tetris game without advanced features like hold piece, previews, high-score persistence, multiplayer, or mobile touch controls.

## Step-By-Step Behavior

The scenarios below define the concrete product capabilities needed for the first visible product slice.

## Scenario Schedule

1. F-001-S001 - Playfield grid is visible and properly sized
2. F-001-S002 - Tetrominoes spawn and fall at a regular interval
3. F-001-S003 - Player can move tetrominoes left and right using keyboard
4. F-001-S004 - Player can rotate tetrominoes using keyboard
5. F-001-S005 - Tetrominoes lock into place when they hit the bottom or another piece
6. F-001-S006 - Complete lines are cleared and score is updated
7. F-001-S007 - Game over is detected when new tetrominoes can't spawn
8. F-001-S008 - Player can restart the game

## Scenarios

### F-001-S001: Playfield Grid Is Visible And Properly Sized

Given a user visits the Tetris game page
When the game loads
Then a 10x20 grid is visible with clear cell boundaries

### F-001-S002: Tetrominoes Spawn And Fall At A Regular Interval

Given the game is running
When a tetromino spawns
Then it begins falling at a regular interval

### F-001-S003: Player Can Move Tetrominoes Left And Right Using Keyboard

Given a tetromino is falling
When the player presses left or right arrow keys
Then the tetromino moves one cell in the respective direction

### F-001-S004: Player Can Rotate Tetrominoes Using Keyboard

Given a tetromino is falling
When the player presses the up arrow key
Then the tetromino rotates 90 degrees clockwise

### F-001-S005: Tetrominoes Lock Into Place When They Hit The Bottom Or Another Piece

Given a tetromino is falling
When it hits the bottom of the grid or another locked piece
Then the tetromino becomes locked in place

### F-001-S006: Complete Lines Are Cleared And Score Is Updated

Given a tetromino locks and completes one or more horizontal lines
When those lines become complete
Then the completed lines are cleared from the grid
And the score is updated based on the number of lines cleared

### F-001-S007: Game Over Is Detected When New Tetrominoes Can't Spawn

Given the grid is full and no new tetrominoes can spawn
When a new tetromino cannot be placed
Then the game displays a game over state

### F-001-S008: Player Can Restart The Game

Given the game is in a game over state
When the player presses the restart key
Then the game resets to initial state with a new grid

## Out of Scope

The following capabilities are explicitly out of scope for the first visible product slice and are descoped with clear reasons:

- Hold piece functionality - Descoped: Advanced feature that requires additional game state management and UI elements
- Preview of next piece - Descoped: Advanced feature that requires additional UI and game state management
- High-score persistence - Descoped: Advanced feature requiring local storage or backend integration
- Multiplayer capabilities - Descoped: Advanced feature requiring network architecture and game state synchronization
- Mobile touch controls - Descoped: Advanced feature requiring different input handling mechanisms
- Advanced scoring systems (e.g., combos, back-to-back) - Descoped: Advanced feature that changes basic scoring behavior
- Sound effects or music - Descoped: Advanced feature that enhances but doesn't change core gameplay
- Animations beyond basic movement - Descoped: Advanced feature that enhances but doesn't change core gameplay

## Descoped Scenarios

None - all descoped capabilities are listed under Out of Scope with explicit rationale.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeDemoTetrisExplicitListBrief(t *testing.T, repoRoot string) {
	t.Helper()
	content := `# Demo Tetris

Build a browser Tetris game using Phaser JS. The first playable slice should include a visible board, falling tetromino pieces, keyboard movement and rotation, collision detection, line clearing, visible score, game over when the stack reaches the top, and restart after game over.

Use a small, dependency-light implementation that can be installed, built, and smoke-tested locally. Keep sound effects, multiplayer, mobile touch controls, next-piece preview, combo scoring, high-score persistence, and animation polish out of scope for the first slice.
`
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
}

func writeGenericOutcomeBrief(t *testing.T, repoRoot string) {
	t.Helper()
	content := `# Work Board Demo

Create a browser-based work board.

The first useful product should let a user open the app locally and see a usable task board. It should include:

- A visible board with To Do, In Progress, and Done columns.
- Task cards with title and description.
- Create-task workflow.
- Move-task workflow between columns.
- Filter by status.
- A visible task count.
- Empty-state messaging.
- Reset sample data.

Out of scope for the first build:

- Authentication.
- Cloud sync.
- Notifications.
- Team permissions.
- Mobile offline support.
- Analytics dashboards.
`
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
}

func writeDemoTetrisFeatureWithAnimationPolishOutOfScope(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Demo Tetris: Build a browser Tetris game using Phaser JS.

## Business Logic

The product must include a visible board, falling tetromino pieces, keyboard movement and rotation, collision detection, line clearing, visible score, and game over when the stack reaches the top.

## Step-By-Step Behavior

The scenarios below define the concrete product capabilities needed for the first visible product slice.

## Scenario Schedule

1. F-001-S001 - Visible board is displayed with grid structure
2. F-001-S002 - Tetromino pieces fall at regular intervals
3. F-001-S003 - Player can move pieces horizontally with keyboard
4. F-001-S004 - Player can rotate pieces with keyboard
5. F-001-S005 - Collision detection works for piece movement and placement
6. F-001-S006 - Line clearing occurs when rows are complete
7. F-001-S007 - Visible score tracking updates with cleared lines
8. F-001-S008 - Game-over state triggers when stack reaches top
9. F-001-S009 - Player can restart after game over

## Scenarios

### F-001-S001: Visible Board Is Displayed With Grid Structure

Given the game has started
When the game initializes
Then a visible grid-based Tetris board should be displayed with a defined width and height

### F-001-S002: Tetromino Pieces Fall At Regular Intervals

Given the game is running
When a tetromino piece is created
Then the piece should fall at a regular, predictable interval

### F-001-S003: Player Can Move Pieces Horizontally With Keyboard

Given a tetromino piece is falling
When player presses left or right arrow keys
Then the piece should move horizontally in the specified direction

### F-001-S004: Player Can Rotate Pieces With Keyboard

Given a tetromino piece is falling
When player presses up arrow key or spacebar
Then the piece should rotate in place

### F-001-S005: Collision Detection Works For Piece Movement And Placement

Given a tetromino piece is falling
When the piece attempts to move into a blocked position
Then the movement should be prevented

### F-001-S006: Line Clearing Occurs When Rows Are Complete

Given a tetromino piece is placed
When a complete horizontal line is formed
Then that line should be cleared from the board

### F-001-S007: Visible Score Tracking Updates With Cleared Lines

Given a line is cleared
When the line clear is recorded
Then the score should increase based on the number of lines cleared
And the visible score display tracks the current score

### F-001-S008: Game-Over State Triggers When Stack Reaches Top

Given a tetromino piece is placed
When the new piece cannot enter the board due to stack height
Then the game should transition to game-over state

### F-001-S009: Player Can Restart After Game Over

Given the game is over
When the player chooses restart
Then the board resets for another round

## Out of Scope

- Sound effects or audio feedback
- Animation polish
- Next piece preview
- Combo scoring system
- High score persistence or save/load functionality
- Multiplayer support

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
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

func writeTetrisFeatureWithControlsScenarioNoSmokeScenario(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Phaser Tetris Demo

## Business Logic

The product must include a visible playfield grid, falling tetrominoes, keyboard controls using left/right/down/rotate keys, line clearing, scoring, game over, and restart behavior.

## Step-By-Step Behavior

The scenarios below define the minimum viable product behavior.

## Scenario Schedule

1. F-001-S001 - visible playfield grid is displayed
2. F-001-S002 - falling tetrominoes respond to keyboard controls left right down rotate
3. F-001-S003 - line clearing updates scoring
4. F-001-S004 - game over and restart work

## Scenarios

### F-001-S001: Visible Playfield Grid Is Displayed

Given the game is launched
When the scene starts
Then the visible playfield grid is displayed

### F-001-S002: Falling Tetrominoes Respond To Keyboard Controls

Given a tetromino is falling
When the player presses left, right, down, or rotate keys
Then the tetromino responds to keyboard controls

### F-001-S003: Line Clearing Updates Scoring

Given a line is complete
When the board resolves the line
Then line clearing increments the score

### F-001-S004: Game Over And Restart Work

Given pieces reach the spawn area
When the game cannot place a new tetromino
Then game over appears and the player can restart

## Out of Scope

- Menus and online features

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeTetrisFeatureWithCollapsedProductScenario(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

## Business Logic

The product must include a visible playfield grid, falling tetrominoes, keyboard movement and rotation, line clearing, scoring, game over, and restart behavior.

## Step-By-Step Behavior

The scenarios below define the specific product behavior that must be implemented to satisfy the first visible product slice.

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
And the game shows a visible playfield grid
And the game has falling tetrominoes
And the game accepts keyboard movement and rotation
And the game clears lines when they are completed
And the game shows scoring
And the game has game over and restart behavior

### F-001-S003: Product Evidence Comes Before Governance Expansion

Given harness telemetry or intervention debt exists
When no visible product behavior has been delivered yet
Then product planning and ordinary product tickets stay ahead of automatic intervention-debt work

## Out of Scope

- Building every product feature in the first slice

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeTetrisFeatureWithRequiredCapabilitiesOutOfScope(t *testing.T, repoRoot string) {
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

1. F-001-S001 - playfield grid is visible
2. F-001-S002 - tetrominoes fall
3. F-001-S003 - keyboard movement and rotation work

## Scenarios

### F-001-S001: Playfield Grid Is Visible

Given the game is launched
When the scene starts
Then the playfield grid is visible

### F-001-S002: Tetrominoes Fall

Given the playfield is visible
When the game starts
Then tetrominoes fall into the playfield

### F-001-S003: Keyboard Movement And Rotation Work

Given a tetromino is falling
When the player presses movement or rotation keys
Then the tetromino responds

## Out of Scope

- Scoring system
- Line clearing mechanics
- Game over conditions
- Restart functionality

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeTetrisFeatureWithAdvancedOnlyOutOfScope(t *testing.T, repoRoot string) {
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

- Advanced scoring or game modes beyond basic line clearing
- Menus and online features

## Descoped Scenarios

None.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
}

func writeGenericFeatureWithOutcomeScenarios(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "features", "F-001-product-walking-skeleton.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	content := `# F-001: Product Walking Skeleton

- Feature ID: F-001
- Goals: G-001
- Status: active
- Owner: CEO
- Product Brief: Work Board Demo: Create a browser-based work board.

## Business Logic

The product must include a visible board with workflow columns, task cards, task creation, task movement, status filtering, task count, empty-state messaging, and sample-data reset.

## Step-By-Step Behavior

The scenarios below define the concrete product capabilities for the first useful slice.

## Scenario Schedule

1. F-001-S001 - Display a visible board with workflow columns
2. F-001-S002 - Show task cards with title and description
3. F-001-S003 - Create new task cards
4. F-001-S004 - Move task cards between columns
5. F-001-S005 - Filter tasks by status
6. F-001-S006 - Display visible task count
7. F-001-S007 - Show empty-state messaging
8. F-001-S008 - Reset sample data

## Scenarios

### F-001-S001: Display a visible board with workflow columns

Given a browser-based work board
When the first product slice is implemented
Then a user can see To Do, In Progress, and Done columns

### F-001-S002: Show task cards with title and description

Given the board columns are visible
When sample work exists
Then task cards show a title and description

### F-001-S003: Create new task cards

Given the board is visible
When the user submits a new task
Then a task card appears in To Do

### F-001-S004: Move task cards between columns

Given a task card exists
When the user moves it to another status
Then the card appears in the matching column

### F-001-S005: Filter tasks by status

Given multiple task statuses exist
When the user filters by status
Then only matching tasks are shown

### F-001-S006: Display visible task count

Given tasks are visible
When the board renders
Then the task count is visible and accurate

### F-001-S007: Show empty-state messaging

Given no tasks match the current filter
When the board renders
Then an empty-state message is visible

### F-001-S008: Reset sample data

Given the board data has changed
When the user resets sample data
Then the starter tasks and columns are restored

## Out of Scope

- Authentication
- Cloud sync
- Notifications
- Team permissions
- Mobile offline support
- Analytics dashboards

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

func writeTetrisFeatureWithMobileTouchControlsOutOfScope(t *testing.T, repoRoot string) {
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
3. F-001-S003 - keyboard controls move and rotate tetrominoes
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

### F-001-S003: Keyboard Controls Move And Rotate Tetrominoes

Given a tetromino is falling
When the player presses keyboard controls
Then the tetromino moves and rotates

### F-001-S004: Line Clearing Updates Scoring

Given a line is complete
When the board resolves the line
Then line clearing increments the score

### F-001-S005: Game Over And Restart Work

Given pieces reach the spawn area
When the game cannot place a new tetromino
Then game over appears and the player can restart

## Out of Scope

- Mobile touch controls
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

func writePolicyPlanForFeature(t *testing.T, repoRoot, featureID, schedule, current string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "exec-plans", "active", "current-operating-plan.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan: %v", err)
	}
	content := fmt.Sprintf(`# Current Operating Plan

**BDD Feature:** %s
**Scenario Schedule:** %s
**Current Failing Scenario:** %s
`, featureID, schedule, current)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
