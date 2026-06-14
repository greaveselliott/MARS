/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package trace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure Go SQLite (CGO-free)
)

// Store persists traces in SQLite (MH-005).
type Store struct {
	db *sql.DB
}

// OpenStore opens or creates a SQLite database at path (e.g. ~/.mars-harness/state.db).
func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("trace: open sqlite %q: %w", path, err)
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
CREATE TABLE IF NOT EXISTS traces (
  trace_id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL,
  turns_jsonl TEXT NOT NULL,
  summary_json TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_traces_job_id ON traces(job_id);
`)
	if err != nil {
		return fmt.Errorf("trace: init schema: %w", err)
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

// Save inserts or replaces one trace row.
func (s *Store) Save(ctx context.Context, jobID, traceID, turnsJSONL, summaryJSON string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("trace: store is nil")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO traces(trace_id, job_id, turns_jsonl, summary_json, created_at)
VALUES(?,?,?,?,?)
ON CONFLICT(trace_id) DO UPDATE SET
  job_id=excluded.job_id,
  turns_jsonl=excluded.turns_jsonl,
  summary_json=excluded.summary_json,
  created_at=excluded.created_at
`, traceID, jobID, turnsJSONL, summaryJSON, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("trace: save trace %q: %w", traceID, err)
	}
	return nil
}

// GetLatestByJobID returns the most recent trace for jobID, or nil if none exists (not an error).
func (s *Store) GetLatestByJobID(ctx context.Context, jobID string) (*Record, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("trace: store is nil")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT trace_id, job_id, turns_jsonl, summary_json, created_at
FROM traces WHERE job_id = ? ORDER BY created_at DESC LIMIT 1
`, jobID)
	var rec Record
	var created int64
	err := row.Scan(&rec.TraceID, &rec.JobID, &rec.TurnsJSONL, &rec.SummaryJSON, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("trace: load trace for job %q: %w", jobID, err)
	}
	rec.CreatedAt = time.Unix(created, 0).UTC()
	return &rec, nil
}

// ListSince returns trace records created at or after since, newest first.
func (s *Store) ListSince(ctx context.Context, since time.Time) ([]Record, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("trace: store is nil")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT trace_id, job_id, turns_jsonl, summary_json, created_at
FROM traces
WHERE created_at >= ?
ORDER BY created_at DESC
`, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("trace: list since: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		var created int64
		if err := rows.Scan(&rec.TraceID, &rec.JobID, &rec.TurnsJSONL, &rec.SummaryJSON, &created); err != nil {
			return nil, fmt.Errorf("trace: scan trace: %w", err)
		}
		rec.CreatedAt = time.Unix(created, 0).UTC()
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("trace: iterate traces: %w", err)
	}
	return records, nil
}
