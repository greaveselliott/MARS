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
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
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
	ID               string
	RepoID           string
	Role             string
	Trigger          string // JSON payload
	PayloadMode      string
	ConcurrencyGroup string
	DailyCap         int
	IdempotencyKey   string
	Status           JobStatus
	ClaimedBy        string // worker ID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
	Error            string
}

// Queue is a SQLite-backed job queue with per-repo serialization.
type Queue struct {
	db *sql.DB
	mu sync.Mutex
}

// RecoveryRepairReport summarizes queue self-healing for active recovery jobs.
type RecoveryRepairReport struct {
	StaleFailed         int
	DuplicatesCancelled int
	DuplicatesFailed    int
	ActiveGroups        int
}

// Total returns the number of jobs repaired.
func (r RecoveryRepairReport) Total() int {
	return r.StaleFailed + r.DuplicatesCancelled + r.DuplicatesFailed
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
  payload_mode    TEXT NOT NULL DEFAULT '',
  concurrency_group TEXT NOT NULL DEFAULT '',
  daily_cap       INTEGER NOT NULL DEFAULT 0,
  idempotency_key TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT 'pending',
  claimed_by      TEXT NOT NULL DEFAULT '',
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL,
  completed_at    INTEGER,
  error_msg       TEXT NOT NULL DEFAULT ''
);
`)
	if err != nil {
		return fmt.Errorf("queue: init schema: %w", err)
	}
	for _, col := range []struct {
		name string
		def  string
	}{
		{name: "payload_mode", def: "TEXT NOT NULL DEFAULT ''"},
		{name: "concurrency_group", def: "TEXT NOT NULL DEFAULT ''"},
		{name: "daily_cap", def: "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := q.ensureJobsColumn(col.name, col.def); err != nil {
			return err
		}
	}
	_, err = q.db.Exec(`
CREATE INDEX IF NOT EXISTS idx_jobs_repo_status_created ON jobs(repo_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_idempotency ON jobs(idempotency_key) WHERE idempotency_key != '';
CREATE INDEX IF NOT EXISTS idx_jobs_concurrency_status ON jobs(concurrency_group, status) WHERE concurrency_group != '';
`)
	if err != nil {
		return fmt.Errorf("queue: init indexes: %w", err)
	}
	return nil
}

func (q *Queue) ensureJobsColumn(name, definition string) error {
	rows, err := q.db.Query(`PRAGMA table_info(jobs)`)
	if err != nil {
		return fmt.Errorf("queue: inspect jobs schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("queue: scan jobs schema: %w", err)
		}
		if colName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("queue: read jobs schema: %w", err)
	}
	if _, err := q.db.Exec(fmt.Sprintf("ALTER TABLE jobs ADD COLUMN %s %s", name, definition)); err != nil {
		return fmt.Errorf("queue: add jobs.%s column: %w", name, err)
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
	job.PayloadMode = strings.TrimSpace(job.PayloadMode)
	job.ConcurrencyGroup = strings.TrimSpace(job.ConcurrencyGroup)
	if job.DailyCap < 0 {
		job.DailyCap = 0
	}
	if job.DailyCap > 0 {
		if job.ConcurrencyGroup == "" {
			job.ConcurrencyGroup = fmt.Sprintf("%s:%s", job.RepoID, job.Role)
		}
		if existingID, capped, err := q.dailyCapReached(ctx, job.ConcurrencyGroup, job.DailyCap, now); err != nil {
			return "", err
		} else if capped {
			slog.Info("queue: daily cap reached",
				"group", job.ConcurrencyGroup,
				"cap", job.DailyCap,
				"existing_id", existingID,
			)
			return existingID, nil
		}
	}

	_, err := q.db.ExecContext(ctx, `
INSERT INTO jobs(id, repo_id, role, trigger_payload, payload_mode, concurrency_group, daily_cap, idempotency_key, status, claimed_by, created_at, updated_at, error_msg)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		job.ID, job.RepoID, job.Role, job.Trigger, job.PayloadMode, job.ConcurrencyGroup, job.DailyCap, job.IdempotencyKey,
		string(StatusPending), "", now.Unix(), now.Unix(), "")
	if err != nil {
		return "", fmt.Errorf("queue: enqueue: %w", err)
	}
	slog.Debug("queue: enqueued", "id", job.ID, "repo", job.RepoID, "role", job.Role)
	return job.ID, nil
}

func (q *Queue) dailyCapReached(ctx context.Context, group string, cap int, now time.Time) (string, bool, error) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	var count int
	var existingID string
	err := q.db.QueryRowContext(ctx, `
SELECT COUNT(*), COALESCE(MAX(id), '')
FROM jobs
WHERE concurrency_group = ?
  AND created_at >= ?
  AND status != 'cancelled'`, group, dayStart).Scan(&count, &existingID)
	if err != nil {
		return "", false, fmt.Errorf("queue: check daily cap for group %q: %w", group, err)
	}
	return existingID, count >= cap, nil
}

// Claim atomically picks the next eligible pending job and marks it as claimed.
// Per-repo serialization: a job is only claimable if no running job exists for
// the same repo_id or concurrency group. Stale claimed jobs (older than
// leaseTimeout) are reset first. Running jobs are left alone; the orchestrator
// watchdog handles genuinely stuck running work using a much longer window.
func (q *Queue) Claim(ctx context.Context, workerID string) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	expiry := now.Add(-leaseTimeout).Unix()

	// Reset stale claimed jobs so they can be re-claimed. Running jobs may be
	// healthy long-running work, so they are not reset by claim polling.
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'pending', claimed_by = '', updated_at = ?
WHERE status = 'claimed' AND updated_at < ?`, now.Unix(), expiry)
	if err != nil {
		return nil, fmt.Errorf("queue: reset stale leases: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.Info("queue: reset stale jobs", "count", n)
	}

	// Find the oldest pending job whose repo has no claimed or running job.
	row := q.db.QueryRowContext(ctx, `
SELECT id, repo_id, role, trigger_payload, payload_mode, concurrency_group, daily_cap, idempotency_key, status,
       claimed_by, created_at, updated_at, completed_at, error_msg
FROM jobs
WHERE status = 'pending'
  AND repo_id NOT IN (SELECT DISTINCT repo_id FROM jobs WHERE status IN ('claimed','running'))
  AND (
    concurrency_group = ''
    OR concurrency_group NOT IN (
      SELECT DISTINCT concurrency_group FROM jobs
      WHERE status IN ('claimed','running') AND concurrency_group != ''
    )
  )
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

// ResetOrphans marks all claimed/running jobs as failed. Call at startup
// to clear jobs orphaned by a previous crash.
func (q *Queue) ResetOrphans(ctx context.Context) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'failed', error_msg = 'orphaned by process restart', updated_at = ?, completed_at = ?
WHERE status IN ('claimed','running')`, now.Unix(), now.Unix())
	if err != nil {
		return 0, fmt.Errorf("queue: reset orphans: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// FailStuckRunningJobs marks running jobs as failed when they have not updated
// for a long watchdog window. This is intentionally separate from Claim so
// normal long-running jobs are not interrupted by worker polling.
func (q *Queue) FailStuckRunningJobs(ctx context.Context, staleAfter time.Duration, reason string) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if staleAfter <= 0 {
		staleAfter = 6 * time.Hour
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "orchestrator watchdog: running job exceeded stale window"
	}

	now := time.Now().UTC()
	cutoff := now.Add(-staleAfter).Unix()
	res, err := q.db.ExecContext(ctx, `
UPDATE jobs SET status = 'failed', claimed_by = '', error_msg = ?, completed_at = ?, updated_at = ?
WHERE status = 'running' AND updated_at < ?`, reason, now.Unix(), now.Unix(), cutoff)
	if err != nil {
		return 0, fmt.Errorf("queue: fail stuck running jobs: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// RepairActiveRecoveryJobs collapses active auto-recovery storms.
//
// Older versions used timestamped recovery idempotency keys, so a repeatedly
// failing recovery job could leave many pending recovery jobs for the same repo
// and role. This keeps at most one fresh active recovery job per repo/role and
// fails stale claimed/running recovery jobs.
func (q *Queue) RepairActiveRecoveryJobs(ctx context.Context, staleAfter time.Duration) (RecoveryRepairReport, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if staleAfter <= 0 {
		staleAfter = leaseTimeout
	}

	rows, err := q.db.QueryContext(ctx, `
SELECT id, repo_id, role, trigger_payload, payload_mode, concurrency_group, daily_cap, idempotency_key, status,
       claimed_by, created_at, updated_at, completed_at, error_msg
FROM jobs
WHERE status IN ('pending','claimed','running')
  AND (idempotency_key LIKE 'recover:%' OR trigger_payload LIKE '%auto_recover%')`)
	if err != nil {
		return RecoveryRepairReport{}, fmt.Errorf("queue: load active recovery jobs: %w", err)
	}

	groups := make(map[string][]Job)
	for rows.Next() {
		j, scanErr := scanJobFromRows(rows)
		if scanErr != nil {
			_ = rows.Close()
			return RecoveryRepairReport{}, fmt.Errorf("queue: scan active recovery job: %w", scanErr)
		}
		if !isRecoveryJob(j) {
			continue
		}
		key := j.RepoID + "\x00" + j.Role
		groups[key] = append(groups[key], j)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return RecoveryRepairReport{}, fmt.Errorf("queue: iterate active recovery jobs: %w", err)
	}
	if err := rows.Close(); err != nil {
		return RecoveryRepairReport{}, fmt.Errorf("queue: close active recovery rows: %w", err)
	}

	var report RecoveryRepairReport
	if len(groups) == 0 {
		return report, nil
	}

	now := time.Now().UTC()
	staleCutoff := now.Add(-staleAfter)
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return RecoveryRepairReport{}, fmt.Errorf("queue: begin recovery repair: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, jobs := range groups {
		report.ActiveGroups++
		sort.Slice(jobs, func(i, j int) bool {
			left := recoveryKeepRank(jobs[i])
			right := recoveryKeepRank(jobs[j])
			if left != right {
				return left < right
			}
			leftKey := recoveryKeyRank(jobs[i])
			rightKey := recoveryKeyRank(jobs[j])
			if leftKey != rightKey {
				return leftKey < rightKey
			}
			if !jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
				return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
			}
			return jobs[i].ID < jobs[j].ID
		})

		keeperID := ""
		for _, job := range jobs {
			if isStaleActiveJob(job, staleCutoff) {
				n, repairErr := failActiveJob(ctx, tx, job.ID, now, "self-healed stale recovery job")
				if repairErr != nil {
					return RecoveryRepairReport{}, repairErr
				}
				report.StaleFailed += n
				continue
			}

			if keeperID == "" {
				keeperID = job.ID
				continue
			}

			switch job.Status {
			case StatusPending:
				n, repairErr := cancelPendingJob(ctx, tx, job.ID, now, "self-healed duplicate recovery job")
				if repairErr != nil {
					return RecoveryRepairReport{}, repairErr
				}
				report.DuplicatesCancelled += n
			case StatusClaimed, StatusRunning:
				n, repairErr := failActiveJob(ctx, tx, job.ID, now, "self-healed duplicate active recovery job")
				if repairErr != nil {
					return RecoveryRepairReport{}, repairErr
				}
				report.DuplicatesFailed += n
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return RecoveryRepairReport{}, fmt.Errorf("queue: commit recovery repair: %w", err)
	}
	return report, nil
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
SELECT id, repo_id, role, trigger_payload, payload_mode, concurrency_group, daily_cap, idempotency_key, status,
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

// CountByStatus returns the number of jobs with the given status.
func (q *Queue) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE status = ?`, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("queue: count by status %q: %w", status, err)
	}
	return count, nil
}

// RecentJobs returns the most recent jobs (newest first), limited to maxRows.
func (q *Queue) RecentJobs(ctx context.Context, maxRows int) ([]Job, error) {
	rows, err := q.db.QueryContext(ctx, `
SELECT id, repo_id, role, trigger_payload, payload_mode, concurrency_group, daily_cap, idempotency_key, status,
       claimed_by, created_at, updated_at, completed_at, error_msg
FROM jobs ORDER BY created_at DESC LIMIT ?`, maxRows)
	if err != nil {
		return nil, fmt.Errorf("queue: recent jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		var status string
		var createdAt, updatedAt int64
		var completedAt sql.NullInt64
		if err := rows.Scan(
			&j.ID, &j.RepoID, &j.Role, &j.Trigger, &j.PayloadMode, &j.ConcurrencyGroup, &j.DailyCap, &j.IdempotencyKey,
			&status, &j.ClaimedBy, &createdAt, &updatedAt, &completedAt, &j.Error,
		); err != nil {
			return jobs, fmt.Errorf("queue: scan recent job: %w", err)
		}
		j.Status = JobStatus(status)
		j.CreatedAt = time.Unix(createdAt, 0).UTC()
		j.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0).UTC()
			j.CompletedAt = &t
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// JobCountsByHour returns job counts grouped by hour for the last N hours.
// Each entry has the hour timestamp, completed count, and failed count.
type HourlyCount struct {
	Hour      string `json:"hour"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
	Total     int    `json:"total"`
}

func (q *Queue) JobCountsByHour(ctx context.Context, hours int) ([]HourlyCount, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
	rows, err := q.db.QueryContext(ctx, `
SELECT
  strftime('%Y-%m-%dT%H:00:00Z', created_at, 'unixepoch') AS hour,
  SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed,
  SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
  COUNT(*) AS total
FROM jobs
WHERE created_at >= ?
GROUP BY hour
ORDER BY hour`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("queue: job counts by hour: %w", err)
	}
	defer rows.Close()

	var counts []HourlyCount
	for rows.Next() {
		var hc HourlyCount
		if err := rows.Scan(&hc.Hour, &hc.Completed, &hc.Failed, &hc.Total); err != nil {
			return counts, fmt.Errorf("queue: scan hourly count: %w", err)
		}
		counts = append(counts, hc)
	}
	return counts, rows.Err()
}

func isRecoveryJob(job Job) bool {
	if strings.HasPrefix(job.IdempotencyKey, "recover:") {
		return true
	}
	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(job.Trigger), &payload); err != nil {
		return false
	}
	return payload.Type == "auto_recover"
}

func recoveryKeepRank(job Job) int {
	switch job.Status {
	case StatusRunning:
		return 0
	case StatusClaimed:
		return 1
	case StatusPending:
		return 2
	default:
		return 3
	}
}

func recoveryKeyRank(job Job) int {
	if job.IdempotencyKey == fmt.Sprintf("recover:%s:%s", job.RepoID, job.Role) {
		return 0
	}
	if strings.HasPrefix(job.IdempotencyKey, "recover:") {
		return 1
	}
	return 2
}

func isStaleActiveJob(job Job, cutoff time.Time) bool {
	return (job.Status == StatusClaimed || job.Status == StatusRunning) && job.UpdatedAt.Before(cutoff)
}

func failActiveJob(ctx context.Context, tx *sql.Tx, jobID string, now time.Time, errMsg string) (int, error) {
	res, err := tx.ExecContext(ctx, `
UPDATE jobs SET status = 'failed', claimed_by = '', error_msg = ?, completed_at = ?, updated_at = ?
WHERE id = ? AND status IN ('claimed','running')`, errMsg, now.Unix(), now.Unix(), jobID)
	if err != nil {
		return 0, fmt.Errorf("queue: fail active recovery job %q: %w", jobID, err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func cancelPendingJob(ctx context.Context, tx *sql.Tx, jobID string, now time.Time, errMsg string) (int, error) {
	res, err := tx.ExecContext(ctx, `
UPDATE jobs SET status = 'cancelled', claimed_by = '', error_msg = ?, updated_at = ?
WHERE id = ? AND status = 'pending'`, errMsg, now.Unix(), jobID)
	if err != nil {
		return 0, fmt.Errorf("queue: cancel duplicate recovery job %q: %w", jobID, err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	var status string
	var createdAt, updatedAt int64
	var completedAt sql.NullInt64
	err := row.Scan(
		&j.ID, &j.RepoID, &j.Role, &j.Trigger, &j.PayloadMode, &j.ConcurrencyGroup, &j.DailyCap, &j.IdempotencyKey,
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

func scanJobFromRows(rows *sql.Rows) (Job, error) {
	var j Job
	var status string
	var createdAt, updatedAt int64
	var completedAt sql.NullInt64
	err := rows.Scan(
		&j.ID, &j.RepoID, &j.Role, &j.Trigger, &j.PayloadMode, &j.ConcurrencyGroup, &j.DailyCap, &j.IdempotencyKey,
		&status, &j.ClaimedBy, &createdAt, &updatedAt, &completedAt, &j.Error,
	)
	if err != nil {
		return Job{}, err
	}
	j.Status = JobStatus(status)
	j.CreatedAt = time.Unix(createdAt, 0).UTC()
	j.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0).UTC()
		j.CompletedAt = &t
	}
	return j, nil
}

func newUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
