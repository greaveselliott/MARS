/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

func TestEnforceEngineerTicketPrerequisite_ctoHandoffWithoutOpenTicketRoutesToQAWhenDone(t *testing.T) {
	t.Parallel()
	manifest := &bundle.Manifest{Roles: map[string]bundle.RoleConfig{
		"qa":         {},
		"cto-weekly": {},
	}}
	snap := ticketSnapshot{
		Done: []string{"T-001-feature.md"},
		Details: map[string]ticketstate.Ticket{
			"T-001-feature.md": {ID: "T-001", Name: "T-001-feature.md", Status: ticketstate.StatusDone, Kind: "standard"},
		},
	}
	source := &orgstate.Disposition{
		Role:          "cto-weekly",
		Status:        "completed",
		NextNeed:      "implementation",
		SuggestedRole: "engineer",
	}
	decision := orgstate.Decision{NextRole: "engineer", SourceRole: "orchestrator", Reason: "ready for implementation"}
	got := enforceEngineerTicketPrerequisite(decision, snap, manifest, source)
	if got.NextRole != "qa" {
		t.Fatalf("expected qa, got %q", got.NextRole)
	}
	if got.NextNeed != "qa_review" {
		t.Fatalf("expected qa_review, got %q", got.NextNeed)
	}
}

func TestEnforceEngineerTicketPrerequisite_ctoHandoffWithoutOpenTicketEscalatesToCOO(t *testing.T) {
	t.Parallel()
	manifest := &bundle.Manifest{Roles: map[string]bundle.RoleConfig{
		"coo":        {},
		"cto-weekly": {},
	}}
	source := &orgstate.Disposition{
		Role:          "cto-weekly",
		Status:        "completed",
		NextNeed:      "implementation",
		SuggestedRole: "engineer",
	}
	decision := orgstate.Decision{NextRole: "engineer", SourceRole: "orchestrator"}
	got := enforceEngineerTicketPrerequisite(decision, ticketSnapshot{}, manifest, source)
	if got.NextRole != "coo" {
		t.Fatalf("expected coo, got %q", got.NextRole)
	}
	if got.NextNeed != "ticket_breakdown" {
		t.Fatalf("expected ticket_breakdown, got %q", got.NextNeed)
	}
}

func TestTicketSnapshotRoutingHashIgnoresDeferredInterventionDebt(t *testing.T) {
	base := ticketSnapshot{
		Backlog: []string{"T-001-feature.md"},
		Details: map[string]ticketstate.Ticket{
			"T-001-feature.md": {ID: "T-001", Name: "T-001-feature.md", Status: ticketstate.StatusBacklog, Kind: "standard"},
		},
	}
	withDeferred := ticketSnapshot{
		Backlog: []string{"T-001-feature.md", "T-002-intervention.md"},
		Details: map[string]ticketstate.Ticket{
			"T-001-feature.md":      {ID: "T-001", Name: "T-001-feature.md", Status: ticketstate.StatusBacklog, Kind: "standard"},
			"T-002-intervention.md": {ID: "T-002", Name: "T-002-intervention.md", Status: ticketstate.StatusBacklog, Kind: "intervention-debt", Priority: "medium"},
		},
	}
	if base.routingHash() != withDeferred.routingHash() {
		t.Fatalf("expected deferred intervention-debt to be ignored in routing hash")
	}
	if base.hash() == withDeferred.hash() {
		t.Fatalf("full ticket hash should still reflect deferred intervention-debt")
	}
}

func TestValidateEngineerTicketGate_allowsCompletedTicket(t *testing.T) {
	before := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		Done: []string{"T-000-setup.md", "T-001-fix-build.md"},
	}

	if err := validateEngineerTicketGate(before, after); err != nil {
		t.Fatalf("expected completed ticket to pass gate, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsDrainingOneExistingInProgressTicket(t *testing.T) {
	before := ticketSnapshot{
		InProgress: []string{"T-001-fix-build.md", "T-002-auth.md"},
		Done:       []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		InProgress: []string{"T-002-auth.md"},
		Done:       []string{"T-000-setup.md", "T-001-fix-build.md"},
	}

	if err := validateEngineerTicketGate(before, after); err != nil {
		t.Fatalf("expected draining one in-progress ticket to pass, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksNewInProgressHandoff(t *testing.T) {
	before := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
	}
	after := ticketSnapshot{
		InProgress: []string{"T-001-fix-build.md", "T-002-auth.md"},
		Done:       []string{"T-000-setup.md"},
	}

	err := validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected gate to block in-progress tickets")
	}
	if !strings.Contains(err.Error(), "cannot hand off") {
		t.Fatalf("expected handoff error, got %v", err)
	}
	if !strings.Contains(err.Error(), "T-002-auth.md") {
		t.Fatalf("expected ticket names in error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksExistingInProgressWithoutCompletion(t *testing.T) {
	before := ticketSnapshot{
		InProgress: []string{"T-001-fix-build.md", "T-002-auth.md"},
		Done:       []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		InProgress: []string{"T-001-fix-build.md", "T-002-auth.md"},
		Done:       []string{"T-000-setup.md"},
	}

	err := validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected gate to block unchanged in-progress tickets")
	}
	if !strings.Contains(err.Error(), "without completing") {
		t.Fatalf("expected completion error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksReturningInProgressToBacklog(t *testing.T) {
	before := ticketSnapshot{
		InProgress: []string{"T-001-fix-build.md"},
		Done:       []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}

	err := validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected gate to block moving in-progress work away from done")
	}
	if !strings.Contains(err.Error(), "without moving them to done") {
		t.Fatalf("expected done-move error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsReturningInProgressToBacklogWithBlocker(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "in-progress", "T-001-fix-build.md", `---
id: T-001
title: Fix build
blocker: none
blocked_by: []
---

# T-001
`)
	before, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	if err := os.Rename(
		filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-fix-build.md"),
		filepath.Join(dir, "docs", "tickets", "backlog", "T-001-fix-build.md"),
	); err != nil {
		t.Fatalf("move ticket: %v", err)
	}
	writeTicketGateContent(t, dir, "backlog", "T-001-fix-build.md", `---
id: T-001
title: Fix build
blocker: "missing SDK; install before retry"
blocked_by: []
next_action: "Install SDK and resume"
---

# T-001
`)
	after, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}

	if err := validateEngineerTicketGate(before, after); err != nil {
		t.Fatalf("expected blocker return to backlog to pass, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsInProgressBlockedByDependencyTicket(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
blocker: none
blocked_by: []
---

# T-001
`)
	before, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	writeTicketGateContent(t, dir, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
blocker: "waiting for deterministic setup"
blocked_by: ["T-002"]
next_action: "Complete T-002 first"
---

# T-001
`)
	writeTicketGateContent(t, dir, "backlog", "T-002-setup.md", `---
id: T-002
title: Setup
dedupe_key: "dep:T-001:setup"
metadata:
  blocks: "T-001"
---

# T-002
`)
	after, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}

	if err := validateEngineerTicketGate(before, after); err != nil {
		t.Fatalf("expected linked dependency blocker to pass, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksSelfReferencedDependency(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
blocker: none
blocked_by: []
---

# T-001
`)
	before, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	writeTicketGateContent(t, dir, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
blocker: "waiting"
blocked_by: ["T-001"]
---

# T-001
`)
	after, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}

	err = validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected self-referenced dependency to fail")
	}
	if !strings.Contains(err.Error(), "without completing") {
		t.Fatalf("expected blocker-progress error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsOnlyBlockedInProgressToAvoidRetryLoop(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "in-progress", "T-001-blocked.md", `---
id: T-001
title: Blocked
blocker: "waiting for T-002"
blocked_by: ["T-002"]
---

# T-001
`)
	writeTicketGateContent(t, dir, "backlog", "T-002-dependency.md", `---
id: T-002
title: Dependency
---

# T-002
`)
	before, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	after, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}

	if err := validateEngineerTicketGate(before, after); err == nil {
		t.Fatal("expected backlog dependency to require progress")
	}

	if err := os.Remove(filepath.Join(dir, "docs", "tickets", "backlog", "T-002-dependency.md")); err != nil {
		t.Fatalf("remove dependency: %v", err)
	}
	before, err = snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot before blocked-only: %v", err)
	}
	after, err = snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshot after blocked-only: %v", err)
	}
	if err := validateEngineerTicketGate(before, after); err != nil {
		t.Fatalf("expected blocked-only in-progress state to pass, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksNoCompletionWithOpenWork(t *testing.T) {
	before := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}

	err := validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected gate to block no-progress completion")
	}
	if !strings.Contains(err.Error(), "without moving any ticket") {
		t.Fatalf("expected no-completion error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsNoOpenWork(t *testing.T) {
	if err := validateEngineerTicketGate(ticketSnapshot{}, ticketSnapshot{}); err != nil {
		t.Fatalf("expected no open work to pass gate, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksFeatureDoneWithoutBDDEvidence(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "done", "T-001-feature.md", `---
id: T-001
title: Feature
priority: high
work_type: feature
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# T-001: Feature
`)

	err := validateEngineerTicketGateWithEvidence(dir,
		ticketSnapshot{Backlog: []string{"T-001-feature.md"}},
		ticketSnapshot{Done: []string{"T-001-feature.md"}},
	)
	if err == nil {
		t.Fatal("expected gate to block feature ticket without evidence")
	}
	if !strings.Contains(err.Error(), "BDD scenario evidence") {
		t.Fatalf("expected BDD evidence error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsFeatureDoneWithBDDEvidence(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "done", "T-001-feature.md", `---
id: T-001
title: Feature
priority: high
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["go test ./..."]
verified_by: "engineer"
---

# T-001: Feature
`)

	err := validateEngineerTicketGateWithEvidence(dir,
		ticketSnapshot{Backlog: []string{"T-001-feature.md"}},
		ticketSnapshot{Done: []string{"T-001-feature.md"}},
	)
	if err != nil {
		t.Fatalf("expected feature ticket with evidence to pass, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsFeatureDoneWithMultilineBDDEvidence(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "done", "T-001-feature.md", `---
id: T-001
title: Feature
priority: high
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- node test-game.js
- index.html
verified_by: "engineer"
---

# T-001: Feature
`)

	err := validateEngineerTicketGateWithEvidence(dir,
		ticketSnapshot{Backlog: []string{"T-001-feature.md"}},
		ticketSnapshot{Done: []string{"T-001-feature.md"}},
	)
	if err != nil {
		t.Fatalf("expected multiline BDD evidence to pass, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsEnablerDoneWithoutBDDEvidence(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateContent(t, dir, "done", "T-001-enabler.md", `---
id: T-001
title: Enabler
priority: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: TBD
---

# T-001: Enabler
`)

	err := validateEngineerTicketGateWithEvidence(dir,
		ticketSnapshot{Backlog: []string{"T-001-enabler.md"}},
		ticketSnapshot{Done: []string{"T-001-enabler.md"}},
	)
	if err != nil {
		t.Fatalf("expected enabler ticket to pass without BDD evidence, got %v", err)
	}
}

func TestBDDOperatingModel_goalToFeaturePlanTicketEvidenceDone(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(dir, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}

	// Given a generated target harness, the source of truth is visible before work starts.
	assertFileContains(t, dir, "docs/goals/active.md", "G-001")
	assertFileContains(t, dir, "docs/features/F-001-product-walking-skeleton.md", "Feature ID: F-001")
	assertFileContains(t, dir, "docs/exec-plans/active/current-operating-plan.md", "**Current Failing Scenario:** F-001-S001")
	assertFileContains(t, dir, "docs/tickets/README.md", "bdd_scenarios")

	// When the fake engineer transcript completes one feature scenario with evidence.
	doneName := "T-001-operating-model-loop.md"
	writeTicketGateContent(t, dir, "done", doneName, `---
id: T-001
title: Operating model loop
priority: high
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: ["go test ./internal/serve -run TestBDDOperatingModel_goalToFeaturePlanTicketEvidenceDone"]
verified_by: "fake-llm engineer transcript"
---

# T-001: Operating model loop
`)

	// Then the completion gate accepts the done ticket and the remaining schedule stays visible.
	err := validateEngineerTicketGateWithEvidence(dir,
		ticketSnapshot{Backlog: []string{doneName}},
		ticketSnapshot{Done: []string{doneName}},
	)
	if err != nil {
		t.Fatalf("expected BDD artifact loop to pass, got %v", err)
	}
	assertFileContains(t, dir, "docs/features/F-001-product-walking-skeleton.md", "F-001-S002")
}

func TestSnapshotTickets_listsMarkdownTicketsOnly(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateFile(t, dir, "backlog", "T-002-beta.md")
	writeTicketGateFile(t, dir, "backlog", "T-001-alpha.md")
	writeTicketGateFile(t, dir, "backlog", "README.md")
	writeTicketGateFile(t, dir, "backlog", "notes.txt")
	writeTicketGateFile(t, dir, "in-progress", "T-003-gamma.md")

	snap, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshotTickets: %v", err)
	}
	if got := strings.Join(snap.Backlog, ","); got != "T-001-alpha.md,T-002-beta.md" {
		t.Fatalf("unexpected backlog files: %s", got)
	}
	if got := strings.Join(snap.InProgress, ","); got != "T-003-gamma.md" {
		t.Fatalf("unexpected in-progress files: %s", got)
	}
}

func writeTicketGateFile(t *testing.T, repoPath, status, name string) {
	t.Helper()
	writeTicketGateContent(t, repoPath, status, name, "# ticket\n")
}

func writeTicketGateContent(t *testing.T, repoPath, status, name, content string) {
	t.Helper()
	path := filepath.Join(repoPath, "docs", "tickets", status, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFileContains(t *testing.T, repoPath, relPath, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoPath, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s must contain %q", relPath, want)
	}
}
