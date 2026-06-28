/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-012-self-improvement-loop.md
*/
package serve

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars/internal/queue"
	"github.com/greaveselliott/mars/internal/remediation"
	"github.com/greaveselliott/mars/internal/telemetry"
)

func TestHandleJobFailedRecordsDeterministicRemediationInScoreDetails(t *testing.T) {
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
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/remediation-score", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := srv.traceStore.Save(ctx, "job-remediation-score", "trace-remediation-score", "{}", "{}"); err != nil {
		t.Fatalf("save trace: %v", err)
	}

	failedJob := queueJob("job-remediation-score", repoID, "engineer")
	srv.handleJobFailed(ctx, &failedJob, errTest("executor: workspace_hygiene_blocked before role \"engineer\" run: dirty working tree containment exceeded"))

	var details string
	if err := srv.db.QueryRow(`SELECT details FROM outcomes WHERE job_id = ?`, failedJob.ID).Scan(&details); err != nil {
		t.Fatalf("query outcome details: %v", err)
	}

	var evidence remediationPlanEvidence
	if err := json.Unmarshal([]byte(details), &evidence); err != nil {
		t.Fatalf("outcome details should be JSON: %v\n%s", err, details)
	}
	if evidence.TraceID != "trace-remediation-score" {
		t.Fatalf("expected trace id in outcome details, got %q", evidence.TraceID)
	}
	if !strings.Contains(evidence.Error, "workspace_hygiene_blocked") {
		t.Fatalf("expected original failure in outcome details, got %q", evidence.Error)
	}
	if !remediationEvidenceIncludes(evidence.Attempts, "dirty-worktree:blocker") {
		t.Fatalf("expected dirty-worktree remediation attempt, got %#v", evidence.Attempts)
	}
}

func TestHandleJobFailedExecutesGeneratedDocsAutoSafeRemediation(t *testing.T) {
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
	repoID, err := srv.repos.Register(ctx, repoRoot, "owner/remediation-execute", "main")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	failedJob := queueJob("job-remediation-execute", repoID, "engineer")
	srv.handleJobFailed(ctx, &failedJob, errTest("missing generated docs and operating-model drift"))

	if _, err := os.Stat(filepath.Join(repoRoot, ".harness", "metadata.yaml")); err != nil {
		t.Fatalf("expected generated harness metadata after remediation execution: %v", err)
	}

	var details string
	if err := srv.db.QueryRow(`SELECT details FROM outcomes WHERE job_id = ?`, failedJob.ID).Scan(&details); err != nil {
		t.Fatalf("query outcome details: %v", err)
	}
	var evidence remediationPlanEvidence
	if err := json.Unmarshal([]byte(details), &evidence); err != nil {
		t.Fatalf("outcome details should be JSON: %v\n%s", err, details)
	}
	if !remediationExecutionIncludes(evidence.Executions, "generated-docs:update-missing-defaults", "applied") {
		t.Fatalf("expected applied generated-docs remediation execution, got %#v", evidence.Executions)
	}
}

func TestHandleRemediationExecutableReadyRecipeSuppressesGenericRetry(t *testing.T) {
	_, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	srv.handleRemediation(telemetry.Event{
		ID:       "evt-executable-recipe",
		JobID:    "job-executable-recipe",
		RepoID:   "repo-executable",
		Role:     "engineer",
		Category: telemetry.CategoryUnknown,
		Message:  "missing generated docs and operating-model drift",
		Action:   string(telemetry.ActionRetryLonger),
		Remedied: true,
	})

	claimed, err := srv.queue.Claim(context.Background(), "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed != nil {
		t.Fatalf("expected ready deterministic recipe to suppress generic retry, got %#v", claimed)
	}
}

func TestHandleRemediationAutoSafeWithoutExecutorDoesNotSuppressGenericRetry(t *testing.T) {
	_, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.remediators = remediation.NewRegistry([]remediation.Recipe{{
		ID:         "tool-timeout:auto-safe",
		Title:      "Auto-safe Timeout Repair",
		Summary:    "Test-only deterministic repair without a serve executor.",
		Target:     "tools",
		Categories: []telemetry.FailureCategory{telemetry.CategoryToolTimeout},
		Safety:     remediation.SafetyAutoSafe,
		NextAction: "Do not suppress generic retry until an executor exists.",
	}})

	srv.handleRemediation(telemetry.Event{
		ID:       "evt-ready-no-executor",
		JobID:    "job-ready-no-executor",
		RepoID:   "repo-ready-no-executor",
		Role:     "engineer",
		Category: telemetry.CategoryToolTimeout,
		Message:  "tool timed out",
		Action:   string(telemetry.ActionRetryLonger),
		Remedied: true,
	})

	claimed, err := srv.queue.Claim(context.Background(), "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected generic retry when auto-safe recipe has no executor")
	}
	if claimed.Role != "engineer" {
		t.Fatalf("expected retry role engineer, got %q", claimed.Role)
	}
}

func TestHandleRemediationOperatorRecipeDoesNotSuppressGenericRetry(t *testing.T) {
	_, dbPath := setupDispatchFixture(t)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.remediators = remediation.NewRegistry([]remediation.Recipe{{
		ID:         "tool-timeout:operator",
		Title:      "Operator Timeout Repair",
		Summary:    "Test-only deterministic repair.",
		Target:     "tools",
		Categories: []telemetry.FailureCategory{telemetry.CategoryToolTimeout},
		Safety:     remediation.SafetyOperatorRequired,
		NextAction: "Keep the operator-visible action but still allow generic retry.",
	}})

	srv.handleRemediation(telemetry.Event{
		ID:       "evt-operator-recipe",
		JobID:    "job-operator-recipe",
		RepoID:   "repo-operator",
		Role:     "engineer",
		Category: telemetry.CategoryToolTimeout,
		Message:  "tool timed out",
		Action:   string(telemetry.ActionRetryLonger),
		Remedied: true,
	})

	claimed, err := srv.queue.Claim(context.Background(), "test-worker")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected operator-required recipe to leave generic retry enabled")
	}
	if claimed.Role != "engineer" {
		t.Fatalf("expected retry role engineer, got %q", claimed.Role)
	}
}

func queueJob(id, repoID, role string) queue.Job {
	return queue.Job{ID: id, RepoID: repoID, Role: role}
}

func remediationEvidenceIncludes(attempts []remediationAttemptEvidence, recipeID string) bool {
	for _, attempt := range attempts {
		if attempt.RecipeID == recipeID {
			return true
		}
	}
	return false
}

func remediationExecutionIncludes(executions []remediationExecutionEvidence, recipeID, status string) bool {
	for _, execution := range executions {
		if execution.RecipeID == recipeID && execution.Status == status {
			return true
		}
	}
	return false
}
