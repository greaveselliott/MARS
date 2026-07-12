/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
*/
package queue

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greaveselliott/mars/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempQueue(t *testing.T) *Queue {
	t.Helper()
	dir := t.TempDir()
	q, err := Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })
	return q
}

func TestQueueOpenMigratesLegacyJobsTableBeforeCreatingIndexes(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "legacy.db")

	testutil.WriteSQLiteFixture(t, dbPath, `
CREATE TABLE jobs (
  id              TEXT PRIMARY KEY,
  repo_id         TEXT NOT NULL,
  role            TEXT NOT NULL,
  trigger_payload TEXT NOT NULL DEFAULT '',
  idempotency_key TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT 'pending',
  claimed_by      TEXT NOT NULL DEFAULT '',
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL,
  completed_at    INTEGER,
  error_msg       TEXT NOT NULL DEFAULT ''
);
`)

	q, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })

	testutil.AssertSQLiteColumns(t, q.db, "jobs", "payload_mode", "concurrency_group", "daily_cap")
	testutil.AssertSQLiteIndexes(t, q.db, "idx_jobs_concurrency_status")

	id, err := q.Enqueue(context.Background(), Job{
		RepoID:           "repo-1",
		Role:             "release-manager",
		ConcurrencyGroup: "release",
		DailyCap:         1,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestQueue_enqueueAndClaim(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{
		RepoID:  "repo-1",
		Role:    "ci-fix",
		Trigger: `{"pr":42}`,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	job, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, id, job.ID)
	assert.Equal(t, StatusClaimed, job.Status)
	assert.Equal(t, "w-1", job.ClaimedBy)

	require.NoError(t, q.MarkRunning(ctx, job.ID))
	require.NoError(t, q.Complete(ctx, job.ID))

	got, err := q.Get(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, got.Status)
	assert.NotNil(t, got.CompletedAt)
}

func TestQueueActiveJobsAndRepoScopedOrphanReset(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	first, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer", Trigger: `{"n":1}`})
	require.NoError(t, err)
	second, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "qa", Trigger: `{"n":2}`})
	require.NoError(t, err)
	other, err := q.Enqueue(ctx, Job{RepoID: "repo-2", Role: "ceo", Trigger: `{"n":3}`})
	require.NoError(t, err)

	claimed, err := q.Claim(ctx, "worker-1")
	require.NoError(t, err)
	require.Equal(t, first, claimed.ID)
	require.NoError(t, q.MarkRunning(ctx, claimed.ID))

	active, err := q.ActiveJobsForRepo(ctx, "repo-1", 10)
	require.NoError(t, err)
	require.Len(t, active, 2)

	reset, err := q.ResetOrphansForRepo(ctx, "repo-1", "restart reconciliation")
	require.NoError(t, err)
	require.Equal(t, 1, reset)

	gotFirst, err := q.Get(ctx, first)
	require.NoError(t, err)
	require.Equal(t, StatusFailed, gotFirst.Status)
	require.Contains(t, gotFirst.Error, "restart reconciliation")

	gotSecond, err := q.Get(ctx, second)
	require.NoError(t, err)
	require.Equal(t, StatusPending, gotSecond.Status)
	gotOther, err := q.Get(ctx, other)
	require.NoError(t, err)
	require.Equal(t, StatusPending, gotOther.Status)

	active, err = q.ActiveJobsForRepo(ctx, "repo-1", 10)
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, second, active[0].ID)
}

func TestQueue_idempotency(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id1, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "ci-fix",
		IdempotencyKey: "dedup-key-1",
	})
	require.NoError(t, err)

	id2, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "ci-fix",
		IdempotencyKey: "dedup-key-1",
	})
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "same idempotency key should return same job ID")

	// Complete the first, then a new enqueue with the same key should create a new job.
	job, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, job.ID))
	require.NoError(t, q.Complete(ctx, job.ID))

	id3, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "ci-fix",
		IdempotencyKey: "dedup-key-1",
	})
	require.NoError(t, err)
	assert.NotEqual(t, id1, id3, "completed job should allow new enqueue with same key")
}

func TestQueueWebhookIdempotencySurvivesCompletionAndRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "queue.db")
	q, err := Open(dbPath)
	require.NoError(t, err)
	ctx := context.Background()
	jobs := []Job{{RepoID: "repo-1", Role: "engineer", Trigger: `{"type":"push"}`, IdempotencyKey: "webhook:sha:repo-1:engineer"}}

	ids, duplicate, err := q.EnqueueWebhook(ctx, "delivery-1", "body-sha-1", jobs, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, duplicate)
	require.Len(t, ids, 1)
	job, err := q.Claim(ctx, "worker")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, job.ID))
	require.NoError(t, q.Complete(ctx, job.ID))
	require.NoError(t, q.Close())

	q, err = Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })
	ids, duplicate, err = q.EnqueueWebhook(ctx, "delivery-1", "body-sha-1", jobs, 24*time.Hour)
	require.NoError(t, err)
	require.True(t, duplicate)
	require.Empty(t, ids)
	ids, duplicate, err = q.EnqueueWebhook(ctx, "delivery-2", "body-sha-1", jobs, 24*time.Hour)
	require.NoError(t, err)
	require.True(t, duplicate, "changed delivery with same body must remain idempotent")
	require.Empty(t, ids)
	recent, err := q.RecentJobs(ctx, 10)
	require.NoError(t, err)
	require.Len(t, recent, 1)
}

func TestQueueWebhookFailureRollsBackReceipt(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()
	_, duplicate, err := q.EnqueueWebhook(ctx, "delivery-bad", "body-bad", []Job{{RepoID: "", Role: "engineer"}}, 24*time.Hour)
	require.Error(t, err)
	require.False(t, duplicate)
	ids, duplicate, err := q.EnqueueWebhook(ctx, "delivery-bad", "body-bad", []Job{{RepoID: "repo", Role: "engineer"}}, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, duplicate)
	require.Len(t, ids, 1)
}

func TestQueueWebhookReceiptExpiresAfterTTL(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()
	_, duplicate, err := q.EnqueueWebhook(ctx, "delivery-expiring", "body-expiring", nil, time.Hour)
	require.NoError(t, err)
	require.False(t, duplicate)
	_, err = q.db.Exec(`UPDATE webhook_receipts SET created_at = ? WHERE delivery_id = ?`, time.Now().Add(-2*time.Hour).Unix(), "delivery-expiring")
	require.NoError(t, err)
	_, duplicate, err = q.EnqueueWebhook(ctx, "delivery-expiring", "body-expiring", nil, time.Hour)
	require.NoError(t, err)
	require.False(t, duplicate, "expired durable replay identity should be accepted as a new delivery")
}

func TestQueueWebhookIdempotencySurvivesFailedJob(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()
	jobs := []Job{{RepoID: "repo-failed", Role: "engineer", IdempotencyKey: "webhook:failed"}}
	_, duplicate, err := q.EnqueueWebhook(ctx, "delivery-failed", "body-failed", jobs, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, duplicate)
	job, err := q.Claim(ctx, "worker")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, job.ID))
	require.NoError(t, q.Fail(ctx, job.ID, "deterministic failure"))
	_, duplicate, err = q.EnqueueWebhook(ctx, "delivery-failed", "body-failed", jobs, 24*time.Hour)
	require.NoError(t, err)
	require.True(t, duplicate)
	recent, err := q.RecentJobs(ctx, 10)
	require.NoError(t, err)
	require.Len(t, recent, 1)
	require.Equal(t, StatusFailed, recent[0].Status)
}

func TestQueue_activeJobForRepoRole(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer"})
	require.NoError(t, err)
	_, err = q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "cto-weekly"})
	require.NoError(t, err)
	_, err = q.Enqueue(ctx, Job{RepoID: "repo-2", Role: "engineer"})
	require.NoError(t, err)

	active, err := q.ActiveJobForRepoRole(ctx, "repo-1", "engineer")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id, active.ID)
	assert.Equal(t, StatusPending, active.Status)

	job, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job)
	require.NoError(t, q.MarkRunning(ctx, job.ID))

	active, err = q.ActiveJobForRepoRole(ctx, "repo-1", "engineer")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id, active.ID)
	assert.Equal(t, StatusRunning, active.Status)

	require.NoError(t, q.Complete(ctx, id))
	active, err = q.ActiveJobForRepoRole(ctx, "repo-1", "engineer")
	require.NoError(t, err)
	assert.Nil(t, active)
}

func TestQueue_perRepoSerialization(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, Job{RepoID: "repo-A", Role: "build"})
	require.NoError(t, err)
	_, err = q.Enqueue(ctx, Job{RepoID: "repo-A", Role: "deploy"})
	require.NoError(t, err)

	// Claim and start running the first job for repo-A.
	job1, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job1)
	require.NoError(t, q.MarkRunning(ctx, job1.ID))

	// Second claim for repo-A should return nil (repo locked).
	job2, err := q.Claim(ctx, "w-2")
	require.NoError(t, err)
	assert.Nil(t, job2, "should not claim while another job is running for same repo")

	// A job for a different repo should still be claimable.
	_, err = q.Enqueue(ctx, Job{RepoID: "repo-B", Role: "test"})
	require.NoError(t, err)
	jobB, err := q.Claim(ctx, "w-2")
	require.NoError(t, err)
	require.NotNil(t, jobB)
	assert.Equal(t, "repo-B", jobB.RepoID)

	// Complete repo-A's running job; second pending job becomes claimable.
	require.NoError(t, q.Complete(ctx, job1.ID))
	job2, err = q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job2)
	assert.Equal(t, "repo-A", job2.RepoID)
}

func TestQueue_claimPrioritizesDispatchBeforeScheduledWork(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	scheduledID, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "engineer",
		IdempotencyKey: "sched:repo-1:engineer:1779148800",
	})
	require.NoError(t, err)
	dispatchID, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "orchestrator",
		IdempotencyKey: "dispatch:coo-job:repo-1:orchestrator",
	})
	require.NoError(t, err)
	require.NotEqual(t, scheduledID, dispatchID)

	job, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, dispatchID, job.ID)
	assert.Equal(t, "orchestrator", job.Role)
}

func TestQueue_concurrencyGroupSerialization(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, Job{RepoID: "repo-A", Role: "release-manager", ConcurrencyGroup: "release"})
	require.NoError(t, err)
	_, err = q.Enqueue(ctx, Job{RepoID: "repo-B", Role: "release-manager", ConcurrencyGroup: "release"})
	require.NoError(t, err)

	job1, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job1)
	require.NoError(t, q.MarkRunning(ctx, job1.ID))

	job2, err := q.Claim(ctx, "w-2")
	require.NoError(t, err)
	assert.Nil(t, job2, "same concurrency group should not be claimed concurrently across repos")

	require.NoError(t, q.Complete(ctx, job1.ID))
	job2, err = q.Claim(ctx, "w-2")
	require.NoError(t, err)
	require.NotNil(t, job2)
	assert.Equal(t, "release", job2.ConcurrencyGroup)
}

func TestQueue_dailyCapConstrainsRepeatedScheduling(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	first, err := q.Enqueue(ctx, Job{
		RepoID:           "repo-1",
		Role:             "janitor",
		ConcurrencyGroup: "ticket:repo-1:stale",
		DailyCap:         1,
	})
	require.NoError(t, err)

	second, err := q.Enqueue(ctx, Job{
		RepoID:           "repo-1",
		Role:             "janitor",
		ConcurrencyGroup: "ticket:repo-1:stale",
		DailyCap:         1,
	})
	require.NoError(t, err)
	assert.Equal(t, first, second, "daily cap should return the existing capped job id")

	pending, err := q.CountByStatus(ctx, string(StatusPending))
	require.NoError(t, err)
	assert.Equal(t, 1, pending)
}

func TestQueue_leaseExpiry(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "ci-fix"})
	require.NoError(t, err)

	job, err := q.Claim(ctx, "w-stale")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, id, job.ID)

	// Simulate stale lease by backdating updated_at.
	staleTime := time.Now().UTC().Add(-10 * time.Minute).Unix()
	_, err = q.db.ExecContext(ctx, `UPDATE jobs SET updated_at = ? WHERE id = ?`, staleTime, job.ID)
	require.NoError(t, err)

	// A new claim should reclaim the stale job.
	reclaimed, err := q.Claim(ctx, "w-fresh")
	require.NoError(t, err)
	require.NotNil(t, reclaimed)
	assert.Equal(t, id, reclaimed.ID)
	assert.Equal(t, "w-fresh", reclaimed.ClaimedBy)
}

func TestQueue_claimDoesNotResetHealthyRunningJob(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer"})
	require.NoError(t, err)
	job, err := q.Claim(ctx, "w-running")
	require.NoError(t, err)
	require.NotNil(t, job)
	require.NoError(t, q.MarkRunning(ctx, job.ID))

	staleTime := time.Now().UTC().Add(-10 * time.Minute).Unix()
	_, err = q.db.ExecContext(ctx, `UPDATE jobs SET updated_at = ? WHERE id = ?`, staleTime, id)
	require.NoError(t, err)

	reclaimed, err := q.Claim(ctx, "w-fresh")
	require.NoError(t, err)
	assert.Nil(t, reclaimed, "running jobs are left to the orchestrator watchdog")

	got, err := q.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, got.Status)
}

func TestQueue_failStuckRunningJobs(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	oldID, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer"})
	require.NoError(t, err)
	oldJob, err := q.Claim(ctx, "w-old")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, oldJob.ID))
	_, err = q.db.ExecContext(ctx, `UPDATE jobs SET updated_at = ? WHERE id = ?`, time.Now().UTC().Add(-8*time.Hour).Unix(), oldID)
	require.NoError(t, err)

	newID, err := q.Enqueue(ctx, Job{RepoID: "repo-2", Role: "engineer"})
	require.NoError(t, err)
	newJob, err := q.Claim(ctx, "w-new")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, newJob.ID))

	failed, err := q.FailStuckRunningJobs(ctx, 6*time.Hour, "watchdog test")
	require.NoError(t, err)
	assert.Equal(t, 1, failed)

	oldGot, err := q.Get(ctx, oldID)
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, oldGot.Status)
	assert.Contains(t, oldGot.Error, "watchdog test")

	newGot, err := q.Get(ctx, newID)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, newGot.Status)
}

func TestQueue_cancel(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "deploy"})
	require.NoError(t, err)

	require.NoError(t, q.Cancel(ctx, id))

	got, err := q.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, got.Status)

	// Cancelled job should not be claimable.
	claimed, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	assert.Nil(t, claimed)
}

func TestQueue_repairActiveRecoveryJobsCollapsesDuplicates(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		id, err := q.Enqueue(ctx, Job{
			RepoID:         "repo-1",
			Role:           "engineer",
			Trigger:        fmt.Sprintf(`{"type":"auto_recover","source_job":"job-%d"}`, i),
			IdempotencyKey: fmt.Sprintf("recover:repo-1:engineer:%d", i),
		})
		require.NoError(t, err)
		ids = append(ids, id)
	}

	report, err := q.RepairActiveRecoveryJobs(ctx, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, report.ActiveGroups)
	assert.Equal(t, 2, report.DuplicatesCancelled)
	assert.Equal(t, 2, report.Total())

	statuses := map[JobStatus]int{}
	for _, id := range ids {
		got, err := q.Get(ctx, id)
		require.NoError(t, err)
		statuses[got.Status]++
	}
	assert.Equal(t, 1, statuses[StatusPending])
	assert.Equal(t, 2, statuses[StatusCancelled])
}

func TestQueue_repairActiveRecoveryJobsKeepsCanonicalKey(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	legacyID, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "engineer",
		Trigger:        `{"type":"auto_recover","source_job":"legacy"}`,
		IdempotencyKey: "recover:repo-1:engineer:1777679172228739000",
	})
	require.NoError(t, err)
	canonicalID, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "engineer",
		Trigger:        `{"type":"auto_recover","source_job":"canonical"}`,
		IdempotencyKey: "recover:repo-1:engineer",
	})
	require.NoError(t, err)

	report, err := q.RepairActiveRecoveryJobs(ctx, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, report.DuplicatesCancelled)

	canonical, err := q.Get(ctx, canonicalID)
	require.NoError(t, err)
	assert.Equal(t, StatusPending, canonical.Status)

	legacy, err := q.Get(ctx, legacyID)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, legacy.Status)
}

func TestQueue_repairActiveRecoveryJobsFailsStaleRunning(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "engineer",
		Trigger:        `{"type":"auto_recover","source_job":"job-1"}`,
		IdempotencyKey: "recover:repo-1:engineer",
	})
	require.NoError(t, err)

	job, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, job.ID))

	staleTime := time.Now().UTC().Add(-20 * time.Minute).Unix()
	_, err = q.db.ExecContext(ctx, `UPDATE jobs SET updated_at = ? WHERE id = ?`, staleTime, id)
	require.NoError(t, err)

	report, err := q.RepairActiveRecoveryJobs(ctx, 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, report.StaleFailed)
	assert.Equal(t, 1, report.Total())

	got, err := q.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, got.Status)
	assert.Contains(t, got.Error, "self-healed stale recovery job")
}

func TestQueue_repairActiveRecoveryJobsIgnoresNormalJobs(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer"})
	require.NoError(t, err)
	_, err = q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer"})
	require.NoError(t, err)

	report, err := q.RepairActiveRecoveryJobs(ctx, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 0, report.Total())

	pending, err := q.CountByStatus(ctx, string(StatusPending))
	require.NoError(t, err)
	assert.Equal(t, 2, pending)
}

func TestQueue_pruneTTL(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	// Create a completed job with backdated completed_at.
	id, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "ci-fix"})
	require.NoError(t, err)
	job, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, job.ID))
	require.NoError(t, q.Complete(ctx, job.ID))

	oldTime := time.Now().UTC().Add(-60 * 24 * time.Hour).Unix()
	_, err = q.db.ExecContext(ctx, `UPDATE jobs SET completed_at = ? WHERE id = ?`, oldTime, id)
	require.NoError(t, err)

	// Create a fresh completed job.
	id2, err := q.Enqueue(ctx, Job{RepoID: "repo-2", Role: "test"})
	require.NoError(t, err)
	j2, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NoError(t, q.MarkRunning(ctx, j2.ID))
	require.NoError(t, q.Complete(ctx, j2.ID))

	pruned, err := q.PruneTTL(ctx, 30*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	// Old job deleted, fresh job still present.
	got, err := q.Get(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, got)

	got2, err := q.Get(ctx, id2)
	require.NoError(t, err)
	require.NotNil(t, got2)
}

func TestQueue_stressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	dir := t.TempDir()
	q, err := Open(filepath.Join(dir, "stress.db"))
	require.NoError(t, err)
	defer func() { _ = q.Close() }()

	ctx := context.Background()
	repos := []string{"repo-0", "repo-1", "repo-2", "repo-3", "repo-4"}
	jobsPerRepo := 4

	for _, repo := range repos {
		for j := range jobsPerRepo {
			_, err := q.Enqueue(ctx, Job{
				RepoID: repo,
				Role:   fmt.Sprintf("role-%d", j),
			})
			require.NoError(t, err)
		}
	}

	// Track which repos are concurrently running to verify serialization.
	var mu sync.Mutex
	running := make(map[string]bool)
	var violations atomic.Int32
	var completed atomic.Int32

	handler := func(_ context.Context, job *Job) error {
		mu.Lock()
		if running[job.RepoID] {
			violations.Add(1)
		}
		running[job.RepoID] = true
		mu.Unlock()

		time.Sleep(5 * time.Millisecond)

		mu.Lock()
		delete(running, job.RepoID)
		mu.Unlock()

		completed.Add(1)
		return nil
	}

	pool := NewWorkerPool(q, WorkerConfig{
		Concurrency:  10,
		PollInterval: 10 * time.Millisecond,
		OnJob:        handler,
	})

	poolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pool.Start(poolCtx)

	// Wait until all jobs are processed.
	totalJobs := int32(len(repos) * jobsPerRepo)
	require.Eventually(t, func() bool {
		return completed.Load() >= totalJobs
	}, 30*time.Second, 50*time.Millisecond)

	pool.Stop()

	assert.Equal(t, int32(0), violations.Load(), "per-repo serialization violated")
	assert.Equal(t, totalJobs, completed.Load())
}

func TestRecentJobs(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id, err := q.Enqueue(ctx, Job{
			RepoID: "repo-1",
			Role:   fmt.Sprintf("role-%d", i),
		})
		require.NoError(t, err)
		if i%2 == 0 {
			_ = q.Complete(ctx, id)
		}
	}

	jobs, err := q.RecentJobs(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 5)
	assert.True(t, jobs[0].CreatedAt.After(jobs[len(jobs)-1].CreatedAt) ||
		jobs[0].CreatedAt.Equal(jobs[len(jobs)-1].CreatedAt),
		"jobs should be newest first")
}

func TestRecentJobs_empty(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	jobs, err := q.RecentJobs(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

func TestJobCountsByHour(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	claimAndComplete := func(repoID, role string) {
		id, err := q.Enqueue(ctx, Job{RepoID: repoID, Role: role})
		require.NoError(t, err)
		job, err := q.Claim(ctx, "w-test")
		require.NoError(t, err)
		require.NoError(t, q.MarkRunning(ctx, job.ID))
		require.NoError(t, q.Complete(ctx, id))
	}

	claimAndFail := func(repoID, role, errMsg string) {
		id, err := q.Enqueue(ctx, Job{RepoID: repoID, Role: role})
		require.NoError(t, err)
		_, err = q.Claim(ctx, "w-test")
		require.NoError(t, err)
		require.NoError(t, q.MarkRunning(ctx, id))
		require.NoError(t, q.Fail(ctx, id, errMsg))
	}

	claimAndComplete("repo-1", "engineer")
	claimAndComplete("repo-1", "engineer")
	claimAndComplete("repo-1", "engineer")
	claimAndFail("repo-1", "qa", "test error")

	counts, err := q.JobCountsByHour(ctx, 1)
	require.NoError(t, err)
	require.Len(t, counts, 1, "all jobs created in the same hour")
	assert.Equal(t, 3, counts[0].Completed)
	assert.Equal(t, 1, counts[0].Failed)
	assert.Equal(t, 4, counts[0].Total)
}

func TestQueue_PreemptPending(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()

	id, err := q.Enqueue(ctx, Job{RepoID: "repo-1", Role: "engineer"})
	require.NoError(t, err)

	n, err := q.PreemptPending(ctx, "orchestrator stopped with pending work")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	job, err := q.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, job.Status)
	require.Contains(t, job.Error, "orchestrator stopped")
}

func TestQueue_EnqueueReactivatesCancelledSeed(t *testing.T) {
	q := tempQueue(t)
	ctx := context.Background()
	key := "seed:repo-1:ceo:bootstrap"

	firstID, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "ceo",
		IdempotencyKey: key,
	})
	require.NoError(t, err)

	n, err := q.PreemptPending(ctx, "orchestrator stopped with pending work")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	secondID, err := q.Enqueue(ctx, Job{
		RepoID:         "repo-1",
		Role:           "ceo",
		IdempotencyKey: key,
	})
	require.NoError(t, err)
	require.Equal(t, firstID, secondID, "bootstrap restart should reactivate the cancelled seed job")

	job, err := q.Get(ctx, secondID)
	require.NoError(t, err)
	require.Equal(t, StatusPending, job.Status)
	require.Empty(t, job.Error)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
