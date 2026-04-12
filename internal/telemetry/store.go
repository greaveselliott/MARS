package telemetry

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists telemetry events in SQLite so failure patterns survive restarts.
type Store struct {
	db *sql.DB
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
