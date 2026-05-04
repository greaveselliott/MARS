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
