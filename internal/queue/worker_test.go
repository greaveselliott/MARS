/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/features/F-006-queue-and-orchestration.md
*/
package queue

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func testQueue(t *testing.T) *Queue {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "worker-test.db")
	q, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open queue: %v", err)
	}
	t.Cleanup(func() { q.Close() })
	return q
}

func TestWorkerPool_OnComplete_fires_on_success(t *testing.T) {
	q := testQueue(t)

	_, err := q.Enqueue(context.Background(), Job{
		RepoID:         "repo-1",
		Role:           "fixer",
		Trigger:        `{"type":"test"}`,
		IdempotencyKey: "test-success",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var completedRole atomic.Value

	wp := NewWorkerPool(q, WorkerConfig{
		Concurrency:  1,
		PollInterval: 50 * time.Millisecond,
		OnJob: func(_ context.Context, _ *Job) error {
			return nil
		},
		OnComplete: func(_ context.Context, job *Job) {
			completedRole.Store(job.Role)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	deadline := time.After(3 * time.Second)
	for {
		if v := completedRole.Load(); v != nil {
			if v.(string) == "fixer" {
				break
			}
		}
		select {
		case <-deadline:
			t.Fatal("OnComplete did not fire within 3 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()
	wp.Stop()
}

func TestWorkerPool_OnComplete_does_not_fire_on_failure(t *testing.T) {
	q := testQueue(t)

	_, err := q.Enqueue(context.Background(), Job{
		RepoID:         "repo-1",
		Role:           "broken",
		Trigger:        `{"type":"test"}`,
		IdempotencyKey: "test-failure",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var completeCalled atomic.Bool
	var jobProcessed atomic.Bool

	wp := NewWorkerPool(q, WorkerConfig{
		Concurrency:  1,
		PollInterval: 50 * time.Millisecond,
		OnJob: func(_ context.Context, _ *Job) error {
			jobProcessed.Store(true)
			return errors.New("intentional failure")
		},
		OnComplete: func(_ context.Context, _ *Job) {
			completeCalled.Store(true)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	deadline := time.After(2 * time.Second)
	for {
		if jobProcessed.Load() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("job was not processed within 2 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	time.Sleep(200 * time.Millisecond)

	if completeCalled.Load() {
		t.Error("OnComplete should not fire when job fails")
	}

	cancel()
	wp.Stop()
}

func TestWorkerPool_Pause_blocks_new_claims(t *testing.T) {
	q := testQueue(t)

	var claimed atomic.Int32

	wp := NewWorkerPool(q, WorkerConfig{
		Concurrency:  1,
		PollInterval: 50 * time.Millisecond,
		OnJob: func(_ context.Context, _ *Job) error {
			claimed.Add(1)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	wp.Pause()
	if !wp.IsPaused() {
		t.Error("expected IsPaused() to return true after Pause()")
	}

	_, err := q.Enqueue(context.Background(), Job{
		RepoID:         "repo-1",
		Role:           "engineer",
		Trigger:        `{"type":"test"}`,
		IdempotencyKey: "pause-test",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	if claimed.Load() != 0 {
		t.Errorf("expected 0 claims while paused, got %d", claimed.Load())
	}

	cancel()
	wp.Stop()
}

func TestWorkerPool_Resume_after_pause_claims_jobs(t *testing.T) {
	q := testQueue(t)

	var claimed atomic.Int32

	wp := NewWorkerPool(q, WorkerConfig{
		Concurrency:  1,
		PollInterval: 50 * time.Millisecond,
		OnJob: func(_ context.Context, _ *Job) error {
			claimed.Add(1)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	wp.Pause()

	_, err := q.Enqueue(context.Background(), Job{
		RepoID:         "repo-1",
		Role:           "engineer",
		Trigger:        `{"type":"test"}`,
		IdempotencyKey: "resume-test",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	if claimed.Load() != 0 {
		t.Fatal("job was claimed while paused")
	}

	wp.Resume()
	if wp.IsPaused() {
		t.Error("expected IsPaused() to return false after Resume()")
	}

	deadline := time.After(3 * time.Second)
	for claimed.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("job was not claimed within 3 seconds after resume")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()
	wp.Stop()
}
