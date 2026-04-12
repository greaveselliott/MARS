package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerConfig configures a WorkerPool.
type WorkerConfig struct {
	Concurrency  int
	PollInterval time.Duration
	OnJob        func(ctx context.Context, job *Job) error
	OnComplete   func(ctx context.Context, job *Job)
	OnFail       func(ctx context.Context, job *Job, jobErr error)
}

// WorkerPool manages concurrent job workers.
type WorkerPool struct {
	q      *Queue
	cfg    WorkerConfig
	cancel context.CancelFunc
	wg     sync.WaitGroup
	paused atomic.Bool
}

// NewWorkerPool creates a worker pool that pulls jobs from q.
func NewWorkerPool(q *Queue, cfg WorkerConfig) *WorkerPool {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	return &WorkerPool{q: q, cfg: cfg}
}

// Start launches cfg.Concurrency goroutines that poll for jobs.
func (wp *WorkerPool) Start(ctx context.Context) {
	ctx, wp.cancel = context.WithCancel(ctx)
	for i := range wp.cfg.Concurrency {
		wp.wg.Add(1)
		go wp.run(ctx, fmt.Sprintf("worker-%d", i))
	}
	slog.Info("queue: worker pool started", "concurrency", wp.cfg.Concurrency)
}

// Stop cancels all workers and blocks until they drain or the timeout
// expires. A hard ceiling prevents hung jobs from blocking shutdown
// indefinitely (e.g. after a system sleep kills inference).
func (wp *WorkerPool) Stop() {
	if wp.cancel != nil {
		wp.cancel()
	}

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("queue: worker pool stopped gracefully")
	case <-time.After(30 * time.Second):
		slog.Warn("queue: worker pool drain timed out after 30s — forcing shutdown")
	}
}

// Pause stops workers from claiming new jobs. Already-running jobs
// continue to completion.
func (wp *WorkerPool) Pause() {
	wp.paused.Store(true)
	slog.Info("queue: worker pool paused")
}

// Resume allows workers to claim jobs again after a Pause.
func (wp *WorkerPool) Resume() {
	wp.paused.Store(false)
	slog.Info("queue: worker pool resumed")
}

// IsPaused reports whether the pool is currently paused.
func (wp *WorkerPool) IsPaused() bool {
	return wp.paused.Load()
}

func (wp *WorkerPool) run(ctx context.Context, workerID string) {
	defer wp.wg.Done()
	ticker := time.NewTicker(wp.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wp.poll(ctx, workerID)
		}
	}
}

func (wp *WorkerPool) poll(ctx context.Context, workerID string) {
	if wp.paused.Load() {
		return
	}

	job, err := wp.q.Claim(ctx, workerID)
	if err != nil {
		slog.Error("queue: claim failed", "worker", workerID, "error", err)
		return
	}
	if job == nil {
		return
	}

	slog.Info("queue: job claimed", "worker", workerID, "job", job.ID, "role", job.Role)

	if err := wp.q.MarkRunning(ctx, job.ID); err != nil {
		slog.Error("queue: mark running failed", "job", job.ID, "error", err)
		return
	}

	if err := wp.cfg.OnJob(ctx, job); err != nil {
		slog.Error("queue: job failed", "job", job.ID, "error", err)
		_ = wp.q.Fail(ctx, job.ID, err.Error())
		if wp.cfg.OnFail != nil {
			wp.cfg.OnFail(ctx, job, err)
		}
		return
	}

	if err := wp.q.Complete(ctx, job.ID); err != nil {
		slog.Error("queue: complete failed", "job", job.ID, "error", err)
		return
	}

	if wp.cfg.OnComplete != nil {
		wp.cfg.OnComplete(ctx, job)
	}
}
