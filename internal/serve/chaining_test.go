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
  cto-weekly:
    prompt: roles/cto.md
`
	if err := os.WriteFile(filepath.Join(harnessDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"orchestrator", "qa", "cto"} {
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
