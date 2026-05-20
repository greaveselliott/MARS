/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/features/F-006-queue-and-orchestration.md
*/
package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempQueue(t *testing.T) *queue.Queue {
	t.Helper()
	dir := t.TempDir()
	q, err := queue.Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })
	return q
}

func TestScheduler_cronParsing(t *testing.T) {
	valid := []struct {
		name string
		expr string
	}{
		{"every minute", "* * * * *"},
		{"specific", "30 2 * * 1"},
		{"ranges", "0-30 9-17 * * 1-5"},
		{"steps", "*/5 * * * *"},
		{"lists", "0,15,30,45 * * * *"},
		{"combined", "0-30/10 9,17 1-15 * 0,6"},
	}

	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCron(tc.expr)
			assert.NoError(t, err)
		})
	}

	invalid := []struct {
		name string
		expr string
	}{
		{"too few fields", "* * *"},
		{"too many fields", "* * * * * *"},
		{"bad range", "60 * * * *"},
		{"bad step", "*/0 * * * *"},
		{"reversed range", "30-10 * * * *"},
		{"non-numeric", "abc * * * *"},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCron(tc.expr)
			assert.Error(t, err)
		})
	}
}

func TestScheduler_cronMatching(t *testing.T) {
	// 30 9 * * 1 = 09:30 every Monday
	expr, err := parseCron("30 9 * * 1")
	require.NoError(t, err)

	monday0930 := time.Date(2026, 4, 13, 9, 30, 0, 0, time.UTC) // Monday
	assert.True(t, expr.matches(monday0930))

	monday0931 := time.Date(2026, 4, 13, 9, 31, 0, 0, time.UTC)
	assert.False(t, expr.matches(monday0931))

	tuesday0930 := time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC) // Tuesday
	assert.False(t, expr.matches(tuesday0930))
}

func TestScheduler_timezoneHandling(t *testing.T) {
	q := tempQueue(t)
	s := New(q)

	now := time.Now()
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	nyNow := now.In(loc)

	cronExpr := formatMinuteHour(nyNow.Minute(), nyNow.Hour())

	err = s.Register(Schedule{
		Name:     "tz-test",
		RepoID:   "repo-tz",
		Role:     "nightly",
		Cron:     cronExpr,
		Timezone: "America/New_York",
		Trigger:  `{"type":"nightly"}`,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Give the initial tick time to fire.
	time.Sleep(200 * time.Millisecond)
	cancel()
	s.Stop()

	freshCtx := context.Background()
	job, err := q.Claim(freshCtx, "test-worker")
	require.NoError(t, err)
	require.NotNil(t, job, "scheduler should have enqueued a job for current NY time")
	assert.Equal(t, "repo-tz", job.RepoID)
}

func TestScheduler_fireOnce(t *testing.T) {
	q := tempQueue(t)
	s := New(q)

	now := time.Now().UTC()
	cronExpr := formatMinuteHour(now.Minute(), now.Hour())

	err := s.Register(Schedule{
		Name:    "fire-once",
		RepoID:  "repo-fo",
		Role:    "build",
		Cron:    cronExpr,
		Trigger: `{}`,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Manually tick multiple times in the same minute.
	s.tick(ctx)
	s.tick(ctx)
	s.tick(ctx)

	// Should only get one job enqueued (fire_once + idempotency key).
	job1, err := q.Claim(ctx, "w-1")
	require.NoError(t, err)
	require.NotNil(t, job1)

	job2, err := q.Claim(ctx, "w-2")
	require.NoError(t, err)
	assert.Nil(t, job2, "fire_once should prevent duplicate enqueue in same minute")
}

func TestScheduler_skipsWhenRepoRoleAlreadyActive(t *testing.T) {
	q := tempQueue(t)
	s := New(q)
	ctx := context.Background()

	activeID, err := q.Enqueue(ctx, queue.Job{
		RepoID: "repo-active",
		Role:   "engineer",
	})
	require.NoError(t, err)
	active, err := q.Claim(ctx, "w-active")
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, activeID, active.ID)
	require.NoError(t, q.MarkRunning(ctx, active.ID))

	now := time.Now().UTC()
	err = s.Register(Schedule{
		Name:    "repo-active:engineer",
		RepoID:  "repo-active",
		Role:    "engineer",
		Cron:    formatMinuteHour(now.Minute(), now.Hour()),
		Trigger: `{}`,
	})
	require.NoError(t, err)

	s.tick(ctx)

	jobs, err := q.RecentJobs(ctx, 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, activeID, jobs[0].ID)
}

func TestScheduler_registerValidation(t *testing.T) {
	q := tempQueue(t)
	s := New(q)

	err := s.Register(Schedule{
		Name: "bad-cron",
		Cron: "not a cron",
	})
	assert.Error(t, err)

	err = s.Register(Schedule{
		Name:     "bad-tz",
		Cron:     "* * * * *",
		Timezone: "Not/A/Timezone",
	})
	assert.Error(t, err)
}

func formatMinuteHour(minute, hour int) string {
	return fmt.Sprintf("%d %d * * *", minute, hour)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
