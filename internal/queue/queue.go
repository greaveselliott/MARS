package queue

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// JobStatus represents the lifecycle of a job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusClaimed   JobStatus = "claimed"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusCancelled JobStatus = "cancelled"
)

const leaseTimeout = 5 * time.Minute

// Job represents a queued agent job.
type Job struct {
	ID             string
	RepoID         string
	Role           string
	Trigger        string // JSON payload
	IdempotencyKey string
	Status         JobStatus
	ClaimedBy      string // worker ID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
	Error          string
}

// Queue is a SQLite-backed job queue with per-repo serialization.
type Queue struct {
	db *sql.DB
	mu sync.Mutex
}

// Open opens or creates a SQLite-backed job queue at dbPath.
func Open(dbPath string) (*Queue, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("queue: open sqlite %q: %w", dbPath, err)
	}

	q := &Queue{db: db}
	if err := q.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return q, nil
}

func (q *Queue) initSchema() error {
	_, err := q.db.Exec(`
CREATE TABLE IF NOT EXISTS jobs (
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
CREATE INDEX IF NOT EXISTS idx_jobs_repo_status_created ON jobs(repo_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_idempotency ON jobs(idempotency_key) WHERE idempotency_key != '';
`)
	if err != nil {
		return fmt.Errorf("queue: init schema: %w", err)
	}
	return nil
}

// Close releases the database handle.
func (q *Queue) Close() error {
	if q == nil || q.db == nil {
		return nil
	}
	return q.db.Close()
}

// Enqueue adds a job to the queue. If the job's IdempotencyKey is non-empty and
// matches an active (pending/claimed/running) job, the existing job ID is returned.
func (q *Queue) Enqueue(ctx context.Context, job Job) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()

	if job.IdempotencyKey != "" {
		var existingID string
		err := q.db.QueryRowContext(ctx, `
SELECT id FROM jobs
WHERE idempotency_key = ? AND status IN ('pending','claimed','running')
LIMIT 1`, job.IdempotencyKey).Scan(&existingID)
		if err == nil {
			slog.Debug("queue: idempotent duplicate", "existing_id", existingID, "key", job.IdempotencyKey)
			return existingID, nil
		}
		if err != sql.ErrNoRows {
			return "", fmt.Errorf("queue: check idempotency: %w", err)
		}
	}

	if job.ID == "" {
		job.ID = newUUID()
	}

	_, err := q.db.ExecContext(ctx, `
INSERT INTO jobs(id, repo_id, role, trigger_payload, idempotency_key, status, claimed_by, created_at, updated_at, error_msg)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
		job.ID, job.RepoID, job.Role, job.Trigger, job.IdempotencyKey,
		string(StatusPending), "", now.Unix(), now.Unix(), "")
	if err != nil {
		return "", fmt.Errorf("queue: enqueue: %w", err)
	}
	slog.Debug("queue: enqueued", "id", job.ID, "repo", job.RepoID, "role", job.Role)
	return job.ID, nil
}

// Claim atomically picks the next eligible pending job and marks it as claimed.
// Per-repo serialization: a job is only claimable if no running job exists for
// the same repo_id. Stale claimed jobs (older than leaseTimeout) are reset first.
func (q *Queue) Claim(ctx context.Context, workerID string) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	expiry := now.Add(-leaseTimeout).Unix()

	// Reset stale claimed jobs so they can be re-claimed.
	_, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'pending', claimed_by = '', updated_at = ?
WHERE status = 'claimed' AND updated_at < ?`, now.Unix(), expiry)
	if err != nil {
		return nil, fmt.Errorf("queue: reset stale leases: %w", err)
	}

	// Find the oldest pending job whose repo has no claimed or running job.
	row := q.db.QueryRowContext(ctx, `
SELECT id, repo_id, role, trigger_payload, idempotency_key, status,
       claimed_by, created_at, updated_at, completed_at, error_msg
FROM jobs
WHERE status = 'pending'
  AND repo_id NOT IN (SELECT DISTINCT repo_id FROM jobs WHERE status IN ('claimed','running'))
ORDER BY created_at ASC
LIMIT 1`)

	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queue: claim scan: %w", err)
	}

	_, err = q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'claimed', claimed_by = ?, updated_at = ?
WHERE id = ?`, workerID, now.Unix(), job.ID)
	if err != nil {
		return nil, fmt.Errorf("queue: claim update: %w", err)
	}

	job.Status = StatusClaimed
	job.ClaimedBy = workerID
	job.UpdatedAt = now
	return job, nil
}

// MarkRunning transitions a claimed job to running. Called by the worker before
// executing the job handler.
func (q *Queue) MarkRunning(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'running', updated_at = ?
WHERE id = ? AND status = 'claimed'`, now.Unix(), jobID)
	if err != nil {
		return fmt.Errorf("queue: mark running: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue: mark running: job %q not in claimed state", jobID)
	}
	return nil
}

// Complete marks a job as successfully completed.
func (q *Queue) Complete(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'completed', completed_at = ?, updated_at = ?
WHERE id = ? AND status = 'running'`, now.Unix(), now.Unix(), jobID)
	if err != nil {
		return fmt.Errorf("queue: complete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue: complete: job %q not in running state", jobID)
	}
	return nil
}

// Fail marks a job as failed with an error message.
func (q *Queue) Fail(ctx context.Context, jobID, errMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'failed', error_msg = ?, completed_at = ?, updated_at = ?
WHERE id = ? AND status = 'running'`, errMsg, now.Unix(), now.Unix(), jobID)
	if err != nil {
		return fmt.Errorf("queue: fail: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue: fail: job %q not in running state", jobID)
	}
	return nil
}

// Cancel marks a pending job as cancelled. Only pending jobs can be cancelled.
func (q *Queue) Cancel(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'cancelled', updated_at = ?
WHERE id = ? AND status = 'pending'`, now.Unix(), jobID)
	if err != nil {
		return fmt.Errorf("queue: cancel: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue: cancel: job %q not in pending state", jobID)
	}
	return nil
}

// PruneTTL deletes completed and failed jobs older than maxAge. Returns
// the number of rows deleted.
func (q *Queue) PruneTTL(ctx context.Context, maxAge time.Duration) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().UTC().Add(-maxAge).Unix()
	res, err := q.db.ExecContext(ctx, `
DELETE FROM jobs
WHERE status IN ('completed','failed') AND completed_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("queue: prune: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Get retrieves a single job by ID. Returns nil if not found.
func (q *Queue) Get(ctx context.Context, jobID string) (*Job, error) {
	row := q.db.QueryRowContext(ctx, `
SELECT id, repo_id, role, trigger_payload, idempotency_key, status,
       claimed_by, created_at, updated_at, completed_at, error_msg
FROM jobs WHERE id = ?`, jobID)
	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queue: get %q: %w", jobID, err)
	}
	return job, nil
}

func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	var status string
	var createdAt, updatedAt int64
	var completedAt sql.NullInt64
	err := row.Scan(
		&j.ID, &j.RepoID, &j.Role, &j.Trigger, &j.IdempotencyKey,
		&status, &j.ClaimedBy, &createdAt, &updatedAt, &completedAt, &j.Error,
	)
	if err != nil {
		return nil, err
	}
	j.Status = JobStatus(status)
	j.CreatedAt = time.Unix(createdAt, 0).UTC()
	j.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0).UTC()
		j.CompletedAt = &t
	}
	return &j, nil
}

func newUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
