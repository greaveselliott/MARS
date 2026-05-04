/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/foundationtelemetry"
	_ "modernc.org/sqlite"
)

// Store persists telemetry events in SQLite so failure patterns survive restarts.
type Store struct {
	db *sql.DB
}

// RoleCategoryCount is an aggregate telemetry bucket for recurring-pattern detection.
type RoleCategoryCount struct {
	RepoID       string
	Role         string
	Category     FailureCategory
	Count        int
	DistinctJobs int
	FirstSeen    time.Time
	LastSeen     time.Time
}

// OutboxRecord is a queued anonymous foundation telemetry report.
type OutboxRecord struct {
	ID            string
	SchemaVersion int
	CreatedAt     time.Time
	WindowStart   time.Time
	WindowEnd     time.Time
	PayloadHash   string
	PayloadJSON   string
	Status        string
	Attempts      int
	NextAttemptAt time.Time
	LastError     string
}

// OpenStore opens or creates a SQLite database for telemetry at dbPath.
func OpenStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("telemetry: open sqlite %q: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS telemetry_events (
  id        TEXT PRIMARY KEY,
  timestamp INTEGER NOT NULL,
  job_id    TEXT NOT NULL,
  repo_id   TEXT NOT NULL,
  role      TEXT NOT NULL,
  category  TEXT NOT NULL,
  message   TEXT NOT NULL,
  remedied  INTEGER NOT NULL DEFAULT 0,
  action    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_telem_role_cat_ts ON telemetry_events(role, category, timestamp);
CREATE INDEX IF NOT EXISTS idx_telem_ts ON telemetry_events(timestamp);
CREATE TABLE IF NOT EXISTS telemetry_report_outbox (
  id TEXT PRIMARY KEY,
  schema_version INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  window_start INTEGER NOT NULL,
  window_end INTEGER NOT NULL,
  payload_hash TEXT NOT NULL UNIQUE,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_telemetry_report_outbox_status
ON telemetry_report_outbox(status, next_attempt_at);
`)
	if err != nil {
		return fmt.Errorf("telemetry: init schema: %w", err)
	}
	return nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Save persists a single telemetry event.
func (s *Store) Save(evt Event) error {
	if s == nil || s.db == nil {
		return nil
	}
	remedied := 0
	if evt.Remedied {
		remedied = 1
	}
	_, err := s.db.Exec(`
INSERT OR REPLACE INTO telemetry_events(id, timestamp, job_id, repo_id, role, category, message, remedied, action)
VALUES(?,?,?,?,?,?,?,?,?)`,
		evt.ID, evt.Timestamp.Unix(), evt.JobID, evt.RepoID, evt.Role,
		string(evt.Category), evt.Message, remedied, evt.Action)
	if err != nil {
		return fmt.Errorf("telemetry: save event %q: %w", evt.ID, err)
	}
	return nil
}

// Recent returns the most recent N events ordered by timestamp descending.
func (s *Store) Recent(limit int) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
SELECT id, timestamp, job_id, repo_id, role, category, message, remedied, action
FROM telemetry_events
ORDER BY timestamp DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("telemetry: query recent: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var evt Event
		var ts int64
		var remedied int
		var cat string
		if err := rows.Scan(&evt.ID, &ts, &evt.JobID, &evt.RepoID, &evt.Role, &cat, &evt.Message, &remedied, &evt.Action); err != nil {
			return nil, fmt.Errorf("telemetry: scan event: %w", err)
		}
		evt.Timestamp = time.Unix(ts, 0).UTC()
		evt.Category = FailureCategory(cat)
		evt.Remedied = remedied != 0
		events = append(events, evt)
	}
	return events, rows.Err()
}

// CountByRoleCategory returns the number of events matching role+category
// since the given time. Used for recurring pattern detection.
func (s *Store) CountByRoleCategory(role string, cat FailureCategory, since time.Time) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var count int
	err := s.db.QueryRow(`
SELECT COUNT(*) FROM telemetry_events
WHERE role = ? AND category = ? AND timestamp >= ?`,
		role, string(cat), since.Unix()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("telemetry: count by role/category: %w", err)
	}
	return count, nil
}

// RoleCategoryCountsSince returns counts grouped by repo, role, and category.
func (s *Store) RoleCategoryCountsSince(since time.Time) ([]RoleCategoryCount, error) {
	return s.RoleCategoryCountsBetween(since, time.Now().UTC())
}

// RoleCategoryCountsBetween returns counts grouped by repo, role, and category.
func (s *Store) RoleCategoryCountsBetween(start, end time.Time) ([]RoleCategoryCount, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}
	rows, err := s.db.Query(`
SELECT repo_id, role, category, COUNT(*), COUNT(DISTINCT job_id), MIN(timestamp), MAX(timestamp)
FROM telemetry_events
WHERE timestamp >= ? AND timestamp <= ?
GROUP BY repo_id, role, category
ORDER BY repo_id, role, category`, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("telemetry: role/category counts: %w", err)
	}
	defer rows.Close()

	var counts []RoleCategoryCount
	for rows.Next() {
		var rc RoleCategoryCount
		var cat string
		var firstSeen int64
		var lastSeen int64
		if err := rows.Scan(&rc.RepoID, &rc.Role, &cat, &rc.Count, &rc.DistinctJobs, &firstSeen, &lastSeen); err != nil {
			return nil, fmt.Errorf("telemetry: scan role/category count: %w", err)
		}
		rc.Category = FailureCategory(cat)
		rc.FirstSeen = time.Unix(firstSeen, 0).UTC()
		rc.LastSeen = time.Unix(lastSeen, 0).UTC()
		counts = append(counts, rc)
	}
	return counts, rows.Err()
}

// EnqueueAnonymousReport stores a sanitized foundation telemetry report in the
// local outbox. Duplicate payloads are compacted by payload_hash.
func (s *Store) EnqueueAnonymousReport(ctx context.Context, report foundationtelemetry.AnonymousReport) (OutboxRecord, error) {
	if s == nil || s.db == nil {
		return OutboxRecord{}, fmt.Errorf("telemetry: store unavailable")
	}
	if err := foundationtelemetry.ValidateReport(report); err != nil {
		return OutboxRecord{}, err
	}
	hash, err := foundationtelemetry.PayloadHash(report)
	if err != nil {
		return OutboxRecord{}, err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return OutboxRecord{}, fmt.Errorf("telemetry: marshal anonymous report: %w", err)
	}
	id := "out-" + hash[:16]
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO telemetry_report_outbox(
  id, schema_version, created_at, window_start, window_end, payload_hash, payload_json, status
) VALUES(?,?,?,?,?,?,?,'pending')
ON CONFLICT(payload_hash) DO UPDATE SET
  payload_json = excluded.payload_json,
  window_start = excluded.window_start,
  window_end = excluded.window_end,
  status = CASE WHEN telemetry_report_outbox.status = 'sent' THEN 'sent' ELSE 'pending' END`,
		id, report.SchemaVersion, now.Unix(), report.WindowStart.Unix(), report.WindowEnd.Unix(), hash, string(payload))
	if err != nil {
		return OutboxRecord{}, fmt.Errorf("telemetry: enqueue anonymous report: %w", err)
	}
	return s.outboxByHash(ctx, hash)
}

// PendingReports returns pending outbox reports whose retry time has arrived.
func (s *Store) PendingReports(ctx context.Context, now time.Time, limit int) ([]OutboxRecord, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, schema_version, created_at, window_start, window_end, payload_hash, payload_json, status, attempts, next_attempt_at, last_error
FROM telemetry_report_outbox
WHERE status = 'pending' AND next_attempt_at <= ?
ORDER BY created_at
LIMIT ?`, now.Unix(), limit)
	if err != nil {
		return nil, fmt.Errorf("telemetry: query anonymous outbox: %w", err)
	}
	defer rows.Close()
	var out []OutboxRecord
	for rows.Next() {
		rec, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// MarkReportSent marks an outbox report sent.
func (s *Store) MarkReportSent(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE telemetry_report_outbox SET status = 'sent', last_error = '' WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("telemetry: mark report sent: %w", err)
	}
	return nil
}

// MarkReportFailed records a retryable send failure.
func (s *Store) MarkReportFailed(ctx context.Context, id string, nextAttempt time.Time, sendErr error) error {
	if s == nil || s.db == nil {
		return nil
	}
	if nextAttempt.IsZero() {
		nextAttempt = time.Now().UTC().Add(time.Hour)
	}
	msg := ""
	if sendErr != nil {
		msg = sendErr.Error()
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE telemetry_report_outbox
SET attempts = attempts + 1, next_attempt_at = ?, last_error = ?
WHERE id = ?`, nextAttempt.Unix(), msg, id)
	if err != nil {
		return fmt.Errorf("telemetry: mark report failed: %w", err)
	}
	return nil
}

// OutboxStats returns counts by outbox status.
func (s *Store) OutboxStats(ctx context.Context) (map[string]int, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM telemetry_report_outbox GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("telemetry: query outbox stats: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("telemetry: scan outbox stats: %w", err)
		}
		out[status] = count
	}
	return out, rows.Err()
}

func (s *Store) outboxByHash(ctx context.Context, hash string) (OutboxRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, schema_version, created_at, window_start, window_end, payload_hash, payload_json, status, attempts, next_attempt_at, last_error
FROM telemetry_report_outbox
WHERE payload_hash = ?`, hash)
	rec, err := scanOutbox(row)
	if err != nil {
		return OutboxRecord{}, fmt.Errorf("telemetry: read anonymous outbox record: %w", err)
	}
	return rec, nil
}

type outboxScanner interface {
	Scan(dest ...any) error
}

func scanOutbox(row outboxScanner) (OutboxRecord, error) {
	var rec OutboxRecord
	var createdAt, windowStart, windowEnd, nextAttempt int64
	if err := row.Scan(&rec.ID, &rec.SchemaVersion, &createdAt, &windowStart, &windowEnd, &rec.PayloadHash, &rec.PayloadJSON, &rec.Status, &rec.Attempts, &nextAttempt, &rec.LastError); err != nil {
		return OutboxRecord{}, fmt.Errorf("telemetry: scan anonymous outbox record: %w", err)
	}
	rec.CreatedAt = time.Unix(createdAt, 0).UTC()
	rec.WindowStart = time.Unix(windowStart, 0).UTC()
	rec.WindowEnd = time.Unix(windowEnd, 0).UTC()
	if nextAttempt > 0 {
		rec.NextAttemptAt = time.Unix(nextAttempt, 0).UTC()
	}
	rec.PayloadJSON = strings.TrimSpace(rec.PayloadJSON)
	return rec, nil
}

// LatestByRoleCategory returns the newest matching event for evidence links.
func (s *Store) LatestByRoleCategory(repoID, role string, cat FailureCategory, since time.Time) (*Event, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	row := s.db.QueryRow(`
SELECT id, timestamp, job_id, repo_id, role, category, message, remedied, action
FROM telemetry_events
WHERE repo_id = ? AND role = ? AND category = ? AND timestamp >= ?
ORDER BY timestamp DESC
LIMIT 1`, repoID, role, string(cat), since.Unix())

	var evt Event
	var ts int64
	var remedied int
	var catValue string
	err := row.Scan(&evt.ID, &ts, &evt.JobID, &evt.RepoID, &evt.Role, &catValue, &evt.Message, &remedied, &evt.Action)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("telemetry: latest role/category: %w", err)
	}
	evt.Timestamp = time.Unix(ts, 0).UTC()
	evt.Category = FailureCategory(catValue)
	evt.Remedied = remedied != 0
	return &evt, nil
}

// AllCategoryCounts returns aggregate counts per category from all stored events.
func (s *Store) AllCategoryCounts() (map[FailureCategory]int, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT category, COUNT(*) FROM telemetry_events GROUP BY category`)
	if err != nil {
		return nil, fmt.Errorf("telemetry: category counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[FailureCategory]int)
	for rows.Next() {
		var cat string
		var cnt int
		if err := rows.Scan(&cat, &cnt); err != nil {
			return nil, fmt.Errorf("telemetry: scan category count: %w", err)
		}
		counts[FailureCategory(cat)] = cnt
	}
	return counts, rows.Err()
}
