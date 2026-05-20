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
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/greaveselliott/mars-harness/internal/queue"
)

func setupChainingFixture(t *testing.T) (string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	harnessDir := filepath.Join(repoRoot, ".harness")
	if err := os.MkdirAll(filepath.Join(harnessDir, "roles"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := `
name: chain-test
roles:
  pipeline-fixer:
    prompt: roles/fixer.md
    triggers:
      - workflow_run.conclusion == "failure"
    then: [qa]
  qa:
    prompt: roles/qa.md
    triggers:
      - pull_request.opened
`
	if err := os.WriteFile(filepath.Join(harnessDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "roles", "fixer.md"), []byte("Fix CI."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "roles", "qa.md"), []byte("Run QA."), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(t.TempDir(), "chain-test.db")
	return repoRoot, dbPath
}

func setupDispatchFixture(t *testing.T) (string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	harnessDir := filepath.Join(repoRoot, ".harness")
	if err := os.MkdirAll(filepath.Join(harnessDir, "roles"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := `
name: dispatch-test
orchestration_mode: dispatch
roles:
  orchestrator:
    prompt: roles/orchestrator.md
  qa:
    prompt: roles/qa.md
  dogfood:
    prompt: roles/dogfood.md
  release-manager:
    prompt: roles/release-manager.md
  cto-weekly:
    prompt: roles/cto.md
  coo:
    prompt: roles/coo.md
  engineer:
    prompt: roles/engineer.md
`
	if err := os.WriteFile(filepath.Join(harnessDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"orchestrator", "qa", "dogfood", "release-manager", "cto", "coo", "engineer"} {
		if err := os.WriteFile(filepath.Join(harnessDir, "roles", role+".md"), []byte(role+" prompt"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dbPath := filepath.Join(t.TempDir(), "dispatch-test.db")
	return repoRoot, dbPath
}

func TestHandleJobComplete_enqueuesChainedJobs(t *testing.T) {
	repoRoot, dbPath := setupChainingFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/repo", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-123",
		RepoID: repoID,
		Role:   "pipeline-fixer",
	}

	srv.handleJobComplete(ctx, completedJob)

	chainedJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if chainedJob == nil {
		t.Fatal("expected a chained QA job to be enqueued")
	}

	if chainedJob.Role != "qa" {
		t.Errorf("expected role=qa, got %s", chainedJob.Role)
	}
	if chainedJob.RepoID != repoID {
		t.Errorf("expected repo_id=%s, got %s", repoID, chainedJob.RepoID)
	}

	var trigger map[string]string
	if err := json.Unmarshal([]byte(chainedJob.Trigger), &trigger); err != nil {
		t.Fatalf("unmarshal trigger: %v", err)
	}

	if trigger["type"] != "chain" {
		t.Errorf("expected trigger type=chain, got %q", trigger["type"])
	}
	if trigger["source_role"] != "pipeline-fixer" {
		t.Errorf("expected source_role=pipeline-fixer, got %q", trigger["source_role"])
	}
	if trigger["source_job"] != "job-123" {
		t.Errorf("expected source_job=job-123, got %q", trigger["source_job"])
	}
}

func TestHandleJobComplete_noChainWhenThenEmpty(t *testing.T) {
	repoRoot, dbPath := setupChainingFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/repo2", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-456",
		RepoID: repoID,
		Role:   "qa",
	}

	srv.handleJobComplete(ctx, completedJob)

	claimed, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed != nil {
		t.Errorf("expected no chained job, but got role=%s", claimed.Role)
	}
}

func TestHandleJobComplete_dispatchTriggerCarriesStructuredDispositionToOrchestrator(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-structured",
		RepoID: repoID,
		Role:   "qa",
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "qa",
		Status:        "changes_requested",
		Reason:        "ticket is under-specified",
		NextNeed:      "ticket_breakdown",
		TicketID:      "T-123",
		EvidenceLinks: []string{"docs/tickets/in-review/T-123.md"},
		TraceID:       "trace-123",
		Feedback: orgstate.Feedback{
			ForRole:         "cto-weekly",
			Summary:         "ticket is too vague",
			RequestedChange: "split the implementation ticket",
			Severity:        "blocking",
			EvidenceLinks:   []string{"docs/tickets/in-review/T-123.md"},
		},
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "orchestrator" {
		t.Fatalf("expected orchestrator, got %s", dispatchJob.Role)
	}

	var trigger dispatchTriggerPayload
	if err := json.Unmarshal([]byte(dispatchJob.Trigger), &trigger); err != nil {
		t.Fatalf("unmarshal trigger: %v", err)
	}
	if trigger.Type != "dispatch" {
		t.Fatalf("expected dispatch trigger, got %q", trigger.Type)
	}
	if trigger.TargetRole != "orchestrator" {
		t.Fatalf("expected target_role orchestrator, got %q", trigger.TargetRole)
	}
	if trigger.SourceDisposition.TicketID != "T-123" {
		t.Fatalf("expected ticket id in source disposition, got %q", trigger.SourceDisposition.TicketID)
	}
	if trigger.SourceDisposition.TraceID != "trace-123" {
		t.Fatalf("expected trace id, got %q", trigger.SourceDisposition.TraceID)
	}
	if got := trigger.SourceDisposition.Feedback.ForRole; got != "cto-weekly" {
		t.Fatalf("expected feedback.for_role cto-weekly, got %q", got)
	}
	if got := trigger.SourceDisposition.EvidenceLinks; len(got) != 1 || got[0] != "docs/tickets/in-review/T-123.md" {
		t.Fatalf("expected evidence links in source disposition, got %#v", got)
	}
}

func TestHandleJobComplete_completedRoleWithSuggestedRoleDispatchesDirectly(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/direct-dispatch", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-coo-complete",
		RepoID: repoID,
		Role:   "coo",
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "coo",
		Status:        "completed",
		NextNeed:      "ticket_breakdown",
		SuggestedRole: "cto-weekly",
		Reason:        "product plan is ready for implementation ticket shaping",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "cto-weekly" {
		t.Fatalf("expected direct cto-weekly dispatch, got %s", dispatchJob.Role)
	}

	var trigger dispatchTriggerPayload
	if err := json.Unmarshal([]byte(dispatchJob.Trigger), &trigger); err != nil {
		t.Fatalf("unmarshal trigger: %v", err)
	}
	if trigger.SourceRole != "coo" {
		t.Fatalf("expected source role coo, got %q", trigger.SourceRole)
	}
	if trigger.TargetRole != "cto-weekly" {
		t.Fatalf("expected target_role cto-weekly, got %q", trigger.TargetRole)
	}
	if trigger.DecisionKind != "deterministic" {
		t.Fatalf("expected deterministic direct dispatch, got %q", trigger.DecisionKind)
	}
}

func TestHandleJobComplete_orchestratorDispatchCarriesCleanedHandoffToTarget(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch2", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-orchestrator",
		RepoID: repoID,
		Role:   "orchestrator",
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:    completedJob.ID,
		RepoID:   repoID,
		Role:     "orchestrator",
		Status:   "completed",
		NextNeed: "ticket_breakdown",
		Handoff: orgstate.Handoff{
			TargetRole:      "cto-weekly",
			Ask:             "shape implementation tickets from QA feedback",
			Context:         "QA found ticket T-123 too vague",
			Constraints:     []string{"do not route directly to Engineer"},
			ExpectedOutput:  "ready implementation ticket",
			SuccessEvidence: []string{"docs/tickets/backlog/T-123.md"},
		},
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "cto-weekly" {
		t.Fatalf("expected cto-weekly, got %s", dispatchJob.Role)
	}

	var trigger dispatchTriggerPayload
	if err := json.Unmarshal([]byte(dispatchJob.Trigger), &trigger); err != nil {
		t.Fatalf("unmarshal trigger: %v", err)
	}
	if trigger.TargetRole != "cto-weekly" {
		t.Fatalf("expected target_role cto-weekly, got %q", trigger.TargetRole)
	}
	if got := trigger.SourceDisposition.Handoff.Ask; got != "shape implementation tickets from QA feedback" {
		t.Fatalf("expected handoff ask in trigger, got %q", got)
	}
	if got := trigger.SourceDisposition.Handoff.Constraints; len(got) != 1 || got[0] != "do not route directly to Engineer" {
		t.Fatalf("expected handoff constraints in trigger, got %#v", got)
	}
}

func TestHandleJobComplete_engineerDispatchRequiresOpenProductTicket(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-ticket-prereq", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-orchestrator-no-ticket",
		RepoID: repoID,
		Role:   "orchestrator",
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "orchestrator",
		Status:        "completed",
		NextNeed:      "implementation",
		SuggestedRole: "engineer",
		Reason:        "ready for implementation",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "cto-weekly" {
		t.Fatalf("expected cto-weekly ticket-shaping fallback, got %s", dispatchJob.Role)
	}
}

func TestHandleJobComplete_engineerDispatchAllowedWithOpenProductTicket(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)
	ticketDir := filepath.Join(repoRoot, "docs", "tickets", "backlog")
	if err := os.MkdirAll(ticketDir, 0o755); err != nil {
		t.Fatalf("mkdir backlog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ticketDir, "T-001-product.md"), []byte("---\nid: T-001\nkind: feature\n---\n# Product\n"), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-with-ticket", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-orchestrator-with-ticket",
		RepoID: repoID,
		Role:   "orchestrator",
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "orchestrator",
		Status:        "completed",
		NextNeed:      "implementation",
		SuggestedRole: "engineer",
		Reason:        "ready for implementation",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "engineer" {
		t.Fatalf("expected engineer dispatch with open product ticket, got %s", dispatchJob.Role)
	}
}

func TestHandleJobComplete_reviewReworkReusesExistingDoneProductTicket(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)
	ticketDir := filepath.Join(repoRoot, "docs", "tickets", "done")
	if err := os.MkdirAll(ticketDir, 0o755); err != nil {
		t.Fatalf("mkdir done: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ticketDir, "T-001-product.md"), []byte("---\nid: T-001\nkind: feature\n---\n# Product\n"), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-review-rework", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	sourceDisposition := orgstate.Disposition{
		JobID:         "job-qa",
		RepoID:        repoID,
		Role:          "qa",
		Status:        "changes_requested",
		NextNeed:      "implementation_rework",
		SuggestedRole: "engineer",
		TicketID:      "T-001",
		Reason:        "test evidence is missing",
	}
	priorDecision := orgstate.Decision{
		ID:           "decision-qa",
		JobID:        "job-qa",
		RepoID:       repoID,
		SourceRole:   "qa",
		TicketID:     "T-001",
		NextNeed:     "implementation_rework",
		NextRole:     "orchestrator",
		DecisionKind: "orchestrator_review",
		Reason:       "terminal disposition returned to Orchestrator",
	}
	trigger, err := json.Marshal(newDispatchTriggerPayloadForSource("qa", "job-qa", priorDecision, sourceDisposition))
	if err != nil {
		t.Fatalf("marshal trigger: %v", err)
	}

	completedJob := &queue.Job{
		ID:      "job-orchestrator-after-qa-rework",
		RepoID:  repoID,
		Role:    "orchestrator",
		Trigger: string(trigger),
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "orchestrator",
		Status:        "changes_requested",
		NextNeed:      "implementation_rework",
		SuggestedRole: "engineer",
		TicketID:      "T-001",
		Reason:        "route QA rework to Engineer",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "engineer" {
		t.Fatalf("expected engineer rework dispatch for existing done ticket, got %s", dispatchJob.Role)
	}
}

func TestHandleJobComplete_completedEngineerWithoutOpenTicketRoutesQA(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-engineer-done", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	sourceDisposition := orgstate.Disposition{
		JobID:    "job-engineer",
		RepoID:   repoID,
		Role:     "engineer",
		Status:   "completed",
		NextNeed: "implementation",
		Reason:   "implementation ticket completed",
	}
	priorDecision := orgstate.Decision{
		ID:           "decision-engineer",
		JobID:        "job-engineer",
		RepoID:       repoID,
		SourceRole:   "engineer",
		NextNeed:     "implementation",
		NextRole:     "orchestrator",
		DecisionKind: "orchestrator_review",
		Reason:       "terminal disposition returned to Orchestrator",
	}
	trigger, err := json.Marshal(newDispatchTriggerPayloadForSource("engineer", "job-engineer", priorDecision, sourceDisposition))
	if err != nil {
		t.Fatalf("marshal trigger: %v", err)
	}

	completedJob := &queue.Job{
		ID:      "job-orchestrator-after-engineer",
		RepoID:  repoID,
		Role:    "orchestrator",
		Trigger: string(trigger),
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "orchestrator",
		Status:        "completed",
		NextNeed:      "implementation",
		SuggestedRole: "engineer",
		Reason:        "ready for implementation",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "qa" {
		t.Fatalf("expected qa dispatch after completed engineer ticket, got %s", dispatchJob.Role)
	}
}

func TestHandleJobComplete_completedEngineerWithoutOpenTicketOverridesCTOPlanning(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-engineer-done-cto", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	sourceDisposition := orgstate.Disposition{
		JobID:    "job-engineer",
		RepoID:   repoID,
		Role:     "engineer",
		Status:   "completed",
		NextNeed: "implementation",
		Reason:   "implementation ticket completed",
	}
	priorDecision := orgstate.Decision{
		ID:           "decision-engineer",
		JobID:        "job-engineer",
		RepoID:       repoID,
		SourceRole:   "engineer",
		NextNeed:     "implementation",
		NextRole:     "orchestrator",
		DecisionKind: "orchestrator_review",
		Reason:       "terminal disposition returned to Orchestrator",
	}
	trigger, err := json.Marshal(newDispatchTriggerPayloadForSource("engineer", "job-engineer", priorDecision, sourceDisposition))
	if err != nil {
		t.Fatalf("marshal trigger: %v", err)
	}

	completedJob := &queue.Job{
		ID:      "job-orchestrator-after-engineer-cto",
		RepoID:  repoID,
		Role:    "orchestrator",
		Trigger: string(trigger),
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:         completedJob.ID,
		RepoID:        repoID,
		Role:          "orchestrator",
		Status:        "completed",
		NextNeed:      "ticket_breakdown",
		SuggestedRole: "cto-weekly",
		Reason:        "create the next implementation ticket",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "qa" {
		t.Fatalf("expected qa dispatch before another CTO planning pass, got %s", dispatchJob.Role)
	}
}

func TestHandleJobComplete_qaApprovalAfterEngineerRoutesForwardNotBackToQA(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)
	ticketDir := filepath.Join(repoRoot, "docs", "tickets", "done")
	if err := os.MkdirAll(ticketDir, 0o755); err != nil {
		t.Fatalf("mkdir done: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ticketDir, "T-001-product.md"), []byte("---\nid: T-001\nkind: feature\n---\n# Product\n"), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-qa-approved", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	sourceDisposition := orgstate.Disposition{
		JobID:    "job-engineer",
		RepoID:   repoID,
		Role:     "engineer",
		Status:   "completed",
		NextNeed: "qa_review",
		TicketID: "T-001",
		Reason:   "implementation ticket completed",
	}
	priorDecision := orgstate.Decision{
		ID:           "decision-engineer",
		JobID:        "job-engineer",
		RepoID:       repoID,
		SourceRole:   "engineer",
		TicketID:     "T-001",
		NextNeed:     "qa_review",
		NextRole:     "qa",
		DecisionKind: "deterministic",
		Reason:       "route completed implementation to QA",
	}
	trigger, err := json.Marshal(newDispatchTriggerPayloadForSource("engineer", "job-engineer", priorDecision, sourceDisposition))
	if err != nil {
		t.Fatalf("marshal trigger: %v", err)
	}

	completedJob := &queue.Job{
		ID:      "job-qa-approved",
		RepoID:  repoID,
		Role:    "qa",
		Trigger: string(trigger),
	}
	err = srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:    completedJob.ID,
		RepoID:   repoID,
		Role:     "qa",
		Status:   "approved",
		NextNeed: "dogfood_validation",
		TicketID: "T-001",
		Reason:   "ticket verified by QA",
	})
	if err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	dispatchJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if dispatchJob == nil {
		t.Fatal("expected dispatch job")
	}
	if dispatchJob.Role != "dogfood" {
		t.Fatalf("expected dogfood dispatch after QA approval, got %s", dispatchJob.Role)
	}
}

func TestHandleJobComplete_releaseBlockedStopsWithoutDogfoodLoop(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/release-blocked", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	completedJob := &queue.Job{
		ID:     "job-release-blocked",
		RepoID: repoID,
		Role:   "release-manager",
	}
	if err := srv.orgStore.RecordDisposition(ctx, orgstate.Disposition{
		JobID:    completedJob.ID,
		RepoID:   repoID,
		Role:     "release-manager",
		Status:   "blocked",
		NextNeed: "release_blocked",
		TicketID: "T-001",
		Reason:   "No remote is configured after local release notes and tag checks.",
	}); err != nil {
		t.Fatalf("RecordDisposition: %v", err)
	}

	srv.handleJobComplete(ctx, completedJob)

	if got := countJobsByStatusAndRole(t, srv, "pending", "dogfood"); got != 0 {
		t.Fatalf("expected no Dogfood loop for release_blocked publication, got %d", got)
	}
	if got := countJobsByStatusAndRole(t, srv, "pending", "orchestrator"); got != 0 {
		t.Fatalf("expected no Orchestrator loop for release_blocked publication, got %d", got)
	}
	decisions, err := srv.orgStore.RecentDecisions(ctx, repoID, 1)
	if err != nil {
		t.Fatalf("RecentDecisions: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected one stopped decision, got %d", len(decisions))
	}
	if decisions[0].StopReason != "release publication blocked" {
		t.Fatalf("expected release publication blocked stop, got %+v", decisions[0])
	}
}

func TestHandleJobFailed_failedOrchestratorFallsForwardFromSourceDisposition(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-fallback", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	sourceDisposition := orgstate.Disposition{
		JobID:    "job-coo",
		RepoID:   repoID,
		Role:     "coo",
		Status:   "completed",
		NextNeed: "ticket_breakdown",
		Reason:   "plan and feature contract are ready for technical ticket shaping",
	}
	priorDecision := orgstate.Decision{
		ID:           "decision-coo",
		JobID:        "job-coo",
		RepoID:       repoID,
		SourceRole:   "coo",
		NextNeed:     "ticket_breakdown",
		NextRole:     "orchestrator",
		DecisionKind: "orchestrator_review",
		Reason:       "terminal disposition returned to Orchestrator",
	}
	trigger, err := json.Marshal(newDispatchTriggerPayloadForSource("coo", "job-coo", priorDecision, sourceDisposition))
	if err != nil {
		t.Fatalf("marshal trigger: %v", err)
	}

	failedJob := &queue.Job{
		ID:      "job-orchestrator-failed",
		RepoID:  repoID,
		Role:    "orchestrator",
		Trigger: string(trigger),
	}
	srv.handleJobFailed(ctx, failedJob, errTest("executor: agent ended with max_turns"))

	fallbackJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if fallbackJob == nil {
		t.Fatal("expected deterministic fallback job")
	}
	if fallbackJob.Role != "cto-weekly" {
		t.Fatalf("expected cto-weekly fallback, got %s", fallbackJob.Role)
	}

	var fallbackTrigger dispatchTriggerPayload
	if err := json.Unmarshal([]byte(fallbackJob.Trigger), &fallbackTrigger); err != nil {
		t.Fatalf("unmarshal fallback trigger: %v", err)
	}
	if fallbackTrigger.SourceRole != "coo" {
		t.Fatalf("expected source_role coo, got %q", fallbackTrigger.SourceRole)
	}
	if fallbackTrigger.SourceJob != "job-coo" {
		t.Fatalf("expected source_job job-coo, got %q", fallbackTrigger.SourceJob)
	}
	if fallbackTrigger.TargetRole != "cto-weekly" {
		t.Fatalf("expected target_role cto-weekly, got %q", fallbackTrigger.TargetRole)
	}
	if fallbackTrigger.SourceDisposition.NextNeed != "ticket_breakdown" {
		t.Fatalf("expected source next_need ticket_breakdown, got %q", fallbackTrigger.SourceDisposition.NextNeed)
	}
}

func TestHandleJobFailed_failedOrchestratorWithoutSourceDoesNotRecurse(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/dispatch-no-source", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	failedJob := &queue.Job{
		ID:      "job-orchestrator-failed-no-source",
		RepoID:  repoID,
		Role:    "orchestrator",
		Trigger: `{"type":"dispatch","source_role":"orchestrator","source_job":"job-orchestrator-failed-no-source"}`,
	}
	srv.handleJobFailed(ctx, failedJob, errTest("executor: agent ended with max_turns"))

	claimed, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed != nil {
		t.Fatalf("expected no recursive Orchestrator job, got role=%s", claimed.Role)
	}
}

func TestHandleJobFailed_ticketGateEnqueuesBoundedEngineerRepair(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/ticket-gate-repair", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	failedJob := &queue.Job{
		ID:     "job-engineer-ticket-gate",
		RepoID: repoID,
		Role:   "engineer",
	}
	srv.handleJobFailed(ctx, failedJob, errTest("executor: ticket gate: feature ticket T-001.md cannot move to done without BDD scenario evidence: missing evidence_links"))

	repairJob, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if repairJob == nil {
		t.Fatal("expected ticket-gate repair job")
	}
	if repairJob.Role != "engineer" {
		t.Fatalf("expected engineer repair job, got %s", repairJob.Role)
	}
	var trigger map[string]string
	if err := json.Unmarshal([]byte(repairJob.Trigger), &trigger); err != nil {
		t.Fatalf("unmarshal trigger: %v", err)
	}
	if trigger["type"] != "ticket_gate_repair" {
		t.Fatalf("expected ticket_gate_repair trigger, got %q", trigger["type"])
	}
	if trigger["repair_scope"] != "ticket_lifecycle_and_evidence_only" {
		t.Fatalf("expected narrow repair scope, got %q", trigger["repair_scope"])
	}
}

func TestHandleJobFailed_ticketGateRepairDoesNotReenqueue(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/ticket-gate-no-loop", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	failedJob := &queue.Job{
		ID:      "job-engineer-ticket-gate-repair",
		RepoID:  repoID,
		Role:    "engineer",
		Trigger: `{"type":"ticket_gate_repair","source_job":"job-engineer-ticket-gate"}`,
	}
	srv.handleJobFailed(ctx, failedJob, errTest("executor: ticket gate: still missing evidence_links"))

	claimed, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed != nil {
		t.Fatalf("expected no recursive ticket-gate repair job, got role=%s", claimed.Role)
	}
}

func TestHandleJobFailed_dispatchProtocolDoesNotRouteOrchestrator(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/protocol-stop", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	failedJob := &queue.Job{
		ID:     "job-qa-protocol",
		RepoID: repoID,
		Role:   "qa",
	}
	srv.handleJobFailed(ctx, failedJob, errTest("executor: dispatch mode requires qa to call job_disposition_record before completing"))

	if got := countJobsByStatusAndRole(t, srv, "pending", "orchestrator"); got != 0 {
		t.Fatalf("expected dispatch protocol failure to stop without Orchestrator, got %d", got)
	}
}

func TestHandleJobFailed_maxTurnsDoesNotRouteOrchestrator(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/max-turn-stop", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	failedJob := &queue.Job{
		ID:     "job-engineer-max-turns",
		RepoID: repoID,
		Role:   "engineer",
	}
	srv.handleJobFailed(ctx, failedJob, errTest("executor: agent ended with max_turns"))

	if got := countJobsByStatusAndRole(t, srv, "pending", "orchestrator"); got != 0 {
		t.Fatalf("expected max_turns to stop without Orchestrator, got %d", got)
	}
	if got := countJobsByStatusAndRole(t, srv, "pending", "cto-weekly"); got != 0 {
		t.Fatalf("expected max_turns to stop without CTO ticket-shaping, got %d", got)
	}
}

func TestCancelStaleTicketOwnerSurveyJobs(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/stale-ticket-owner", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	trigger := `{"type":"orchestrator.survey","signal":"eligible_in_progress_ticket","tickets":[{"id":"T-001","path":"docs/tickets/in-progress/T-001-demo.md","status":"in-progress"}]}`
	if _, err := srv.queue.Enqueue(ctx, queue.Job{
		RepoID:  repoID,
		Role:    "engineer",
		Trigger: trigger,
	}); err != nil {
		t.Fatalf("enqueue survey job: %v", err)
	}

	cancelled := srv.cancelStaleTicketOwnerSurveyJobs(ctx, RepoRecord{ID: repoID, Path: repoRoot})
	if cancelled != 1 {
		t.Fatalf("expected one stale ticket-owner survey job cancelled, got %d", cancelled)
	}
	if got := countJobsByStatusAndRole(t, srv, "cancelled", "engineer"); got != 1 {
		t.Fatalf("expected cancelled engineer survey job, got %d", got)
	}
}

func TestCancelStaleTicketOwnerSurveyJobsKeepsLiveTicket(t *testing.T) {
	repoRoot, dbPath := setupDispatchFixture(t)
	ticketDir := filepath.Join(repoRoot, "docs", "tickets", "in-progress")
	if err := os.MkdirAll(ticketDir, 0o755); err != nil {
		t.Fatalf("mkdir ticket dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ticketDir, "T-001-demo.md"), []byte(`---
id: T-001
title: Demo ticket
blocker: none
blocked_by: []
---
# Demo ticket
`), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/live-ticket-owner", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	trigger := `{"type":"orchestrator.survey","signal":"eligible_in_progress_ticket","tickets":[{"id":"T-001","path":"docs/tickets/in-progress/T-001-demo.md","status":"in-progress"}]}`
	if _, err := srv.queue.Enqueue(ctx, queue.Job{
		RepoID:  repoID,
		Role:    "engineer",
		Trigger: trigger,
	}); err != nil {
		t.Fatalf("enqueue survey job: %v", err)
	}

	cancelled := srv.cancelStaleTicketOwnerSurveyJobs(ctx, RepoRecord{ID: repoID, Path: repoRoot})
	if cancelled != 0 {
		t.Fatalf("expected live ticket-owner survey job to remain pending, cancelled %d", cancelled)
	}
	if got := countJobsByStatusAndRole(t, srv, "pending", "engineer"); got != 1 {
		t.Fatalf("expected pending engineer survey job, got %d", got)
	}
}
