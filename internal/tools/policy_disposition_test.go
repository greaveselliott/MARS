/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
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
)

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
