package scoring

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

// OutcomeType is the terminal state of a job from an accuracy perspective.
type OutcomeType string

const (
	OutcomeMerged  OutcomeType = "merged"
	OutcomePassed  OutcomeType = "passed"
	OutcomeClosed  OutcomeType = "closed"
	OutcomeFailed  OutcomeType = "failed"
	OutcomeNoop    OutcomeType = "noop"
	OutcomeTimeout OutcomeType = "timeout"
)

const defaultWindowDays = 30

// Outcome records the real-world result of an agent job.
type Outcome struct {
	ID         string
	JobID      string
	RepoID     string
	Role       string
	Type       OutcomeType
	Details    string // JSON metadata
	RecordedAt time.Time
}

// Score is a computed accuracy score for a role+repo.
type Score struct {
	Role       string
	RepoID     string
	Value      float64 // 0.0–1.0
	SampleSize int
	WindowDays int
	Formula    string // "v1"
	ComputedAt time.Time
}

// Store persists outcomes and scores in SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// OpenStore opens or creates a SQLite-backed scoring store at dbPath.
func OpenStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("scoring: open sqlite %q: %w", dbPath, err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS outcomes (
  id          TEXT PRIMARY KEY,
  job_id      TEXT NOT NULL,
  repo_id     TEXT NOT NULL,
  role        TEXT NOT NULL,
  type        TEXT NOT NULL,
  details     TEXT NOT NULL DEFAULT '',
  recorded_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_outcomes_role_repo_time ON outcomes(role, repo_id, recorded_at);

CREATE TABLE IF NOT EXISTS scores (
  role        TEXT NOT NULL,
  repo_id     TEXT NOT NULL,
  value       REAL NOT NULL,
  sample_size INTEGER NOT NULL,
  window_days INTEGER NOT NULL,
  formula     TEXT NOT NULL,
  computed_at INTEGER NOT NULL,
  PRIMARY KEY (role, repo_id)
);
`)
	if err != nil {
		return fmt.Errorf("scoring: init schema: %w", err)
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

// RecordOutcome persists a single outcome. If o.ID is empty, a UUID v4 is generated.
func (s *Store) RecordOutcome(ctx context.Context, o Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if o.ID == "" {
		o.ID = newUUID()
	}
	if o.RecordedAt.IsZero() {
		o.RecordedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO outcomes(id, job_id, repo_id, role, type, details, recorded_at)
VALUES(?,?,?,?,?,?,?)`,
		o.ID, o.JobID, o.RepoID, o.Role, string(o.Type), o.Details, o.RecordedAt.Unix())
	if err != nil {
		return fmt.Errorf("scoring: record outcome: %w", err)
	}
	slog.Debug("scoring: recorded outcome", "id", o.ID, "job", o.JobID, "type", o.Type)
	return nil
}

// ComputeScore calculates and caches the accuracy score for a role+repo pair
// using outcomes within the given window. Formula v1:
//
//	(merged + passed) / (merged + passed + closed + failed + noop)
//
// Timeout outcomes are excluded from the denominator.
func (s *Store) ComputeScore(ctx context.Context, role, repoID string, windowDays int) (Score, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if windowDays <= 0 {
		windowDays = defaultWindowDays
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -windowDays).Unix()

	rows, err := s.db.QueryContext(ctx, `
SELECT type, COUNT(*) FROM outcomes
WHERE role = ? AND repo_id = ? AND recorded_at >= ?
GROUP BY type`, role, repoID, cutoff)
	if err != nil {
		return Score{}, fmt.Errorf("scoring: compute query: %w", err)
	}
	defer rows.Close()

	counts := make(map[OutcomeType]int)
	for rows.Next() {
		var typ string
		var cnt int
		if err := rows.Scan(&typ, &cnt); err != nil {
			return Score{}, fmt.Errorf("scoring: compute scan: %w", err)
		}
		counts[OutcomeType(typ)] = cnt
	}
	if err := rows.Err(); err != nil {
		return Score{}, fmt.Errorf("scoring: compute rows: %w", err)
	}

	positive := counts[OutcomeMerged] + counts[OutcomePassed]
	denominator := positive + counts[OutcomeClosed] + counts[OutcomeFailed] + counts[OutcomeNoop]

	var value float64
	if denominator > 0 {
		value = float64(positive) / float64(denominator)
	}

	now := time.Now().UTC()
	sc := Score{
		Role:       role,
		RepoID:     repoID,
		Value:      value,
		SampleSize: denominator,
		WindowDays: windowDays,
		Formula:    "v1",
		ComputedAt: now,
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO scores(role, repo_id, value, sample_size, window_days, formula, computed_at)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(role, repo_id) DO UPDATE SET
  value=excluded.value, sample_size=excluded.sample_size,
  window_days=excluded.window_days, formula=excluded.formula,
  computed_at=excluded.computed_at`,
		sc.Role, sc.RepoID, sc.Value, sc.SampleSize, sc.WindowDays, sc.Formula, now.Unix())
	if err != nil {
		return Score{}, fmt.Errorf("scoring: cache score: %w", err)
	}

	slog.Debug("scoring: computed", "role", role, "repo", repoID, "value", value, "samples", denominator)
	return sc, nil
}

// GetScore retrieves the cached score for a role+repo. Returns nil if no score exists.
func (s *Store) GetScore(ctx context.Context, role, repoID string) (*Score, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT role, repo_id, value, sample_size, window_days, formula, computed_at
FROM scores WHERE role = ? AND repo_id = ?`, role, repoID)

	var sc Score
	var computedAt int64
	err := row.Scan(&sc.Role, &sc.RepoID, &sc.Value, &sc.SampleSize, &sc.WindowDays, &sc.Formula, &computedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scoring: get score: %w", err)
	}
	sc.ComputedAt = time.Unix(computedAt, 0).UTC()
	return &sc, nil
}

func newUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
