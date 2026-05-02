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

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
