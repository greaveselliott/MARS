/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/scoring-system.md
- docs/features/F-008-scoring-trust-quality.md
*/
package scoring

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// OutcomeType is the terminal state of a job from an accuracy perspective.
type OutcomeType string

const (
	OutcomePassed           OutcomeType = "passed"
	OutcomeCommitted        OutcomeType = "committed"
	OutcomeChecksPassed     OutcomeType = "checks_passed"
	OutcomeChecksFailed     OutcomeType = "checks_failed"
	OutcomeGuardrailBlocked OutcomeType = "guardrail_blocked"
	OutcomeReverted         OutcomeType = "reverted"
	OutcomeHumanFollowup    OutcomeType = "human_followup"
	OutcomeFailed           OutcomeType = "failed"
	OutcomeNoop             OutcomeType = "noop"
	OutcomeTimeout          OutcomeType = "timeout"

	// Legacy outcome names are kept for reading older databases.
	OutcomeMerged OutcomeType = "merged"
	OutcomeClosed OutcomeType = "closed"
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

// RoleRepo identifies a role+repo pair with recorded terminal outcomes.
type RoleRepo struct {
	Role   string
	RepoID string
}

// OutcomeCount is an aggregate terminal outcome bucket.
type OutcomeCount struct {
	Role   string
	RepoID string
	Type   OutcomeType
	Count  int
}

// Store persists outcomes and scores in SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// OpenStore opens or creates a SQLite-backed scoring store at dbPath.
func OpenStore(dbPath string) (*Store, error) {
	if err := validateDBPath(dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("scoring: open sqlite %q: %w", dbPath, err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		if isSQLiteOpenUnavailable(err) {
			return nil, fmt.Errorf("scoring: database at %s is unavailable — run `mars setup`, run `mars register --repo <path>`, or pass --db with a writable SQLite path", dbPath)
		}
		return nil, err
	}
	return s, nil
}

func validateDBPath(dbPath string) error {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return fmt.Errorf("scoring: database path is empty — pass --db <path> or run `mars register --repo <path>`")
	}
	if dbPath == ":memory:" || strings.HasPrefix(dbPath, "file:") {
		return nil
	}
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("scoring: database directory %s does not exist for %s — run `mars setup`, run `mars register --repo <path>`, or create the directory before retrying", dir, dbPath)
		}
		return fmt.Errorf("scoring: check database directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("scoring: database parent %s is not a directory — pass --db with a writable database file path", dir)
	}
	return nil
}

func isSQLiteOpenUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unable to open database file") ||
		strings.Contains(msg, "out of memory (14)") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "is a directory")
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
//	positive / (positive + negative)
//
// Positive outcomes are completed work and passing trunk checks. Negative
// outcomes include failed checks, guardrail blocks, reverts, human follow-ups,
// noops, timeouts, and failed runs. Legacy PR outcomes are still understood.
func (s *Store) ComputeScore(ctx context.Context, role, repoID string, windowDays int) (Score, error) {
	return s.ComputeScoreAt(ctx, role, repoID, windowDays, time.Now().UTC())
}

// ComputeScoreAt is ComputeScore with an explicit reference time. The window
// cutoff is computed relative to now instead of the wall clock, so callers
// that evaluate evidence at a pinned point in time (for example quality-score
// export) get window behavior consistent with their other queries.
func (s *Store) ComputeScoreAt(ctx context.Context, role, repoID string, windowDays int, now time.Time) (Score, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if windowDays <= 0 {
		windowDays = defaultWindowDays
	}

	cutoff := now.UTC().AddDate(0, 0, -windowDays).Unix()

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

	positive := counts[OutcomePassed] + counts[OutcomeCommitted] + counts[OutcomeChecksPassed] + counts[OutcomeMerged]
	negative := counts[OutcomeChecksFailed] +
		counts[OutcomeGuardrailBlocked] +
		counts[OutcomeReverted] +
		counts[OutcomeHumanFollowup] +
		counts[OutcomeClosed] +
		counts[OutcomeFailed] +
		counts[OutcomeNoop] +
		counts[OutcomeTimeout]
	denominator := positive + negative

	var value float64
	if denominator > 0 {
		value = float64(positive) / float64(denominator)
	}

	computedAt := now.UTC()
	sc := Score{
		Role:       role,
		RepoID:     repoID,
		Value:      value,
		SampleSize: denominator,
		WindowDays: windowDays,
		Formula:    "v1",
		ComputedAt: computedAt,
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO scores(role, repo_id, value, sample_size, window_days, formula, computed_at)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(role, repo_id) DO UPDATE SET
  value=excluded.value, sample_size=excluded.sample_size,
  window_days=excluded.window_days, formula=excluded.formula,
  computed_at=excluded.computed_at`,
		sc.Role, sc.RepoID, sc.Value, sc.SampleSize, sc.WindowDays, sc.Formula, computedAt.Unix())
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

// ListScores returns cached scores ordered by repo and role.
func (s *Store) ListScores(ctx context.Context) ([]Score, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT role, repo_id, value, sample_size, window_days, formula, computed_at
FROM scores
ORDER BY repo_id, role`)
	if err != nil {
		return nil, fmt.Errorf("scoring: list scores: %w", err)
	}
	defer rows.Close()

	var scores []Score
	for rows.Next() {
		var sc Score
		var computedAt int64
		if err := rows.Scan(&sc.Role, &sc.RepoID, &sc.Value, &sc.SampleSize, &sc.WindowDays, &sc.Formula, &computedAt); err != nil {
			return nil, fmt.Errorf("scoring: list scores scan: %w", err)
		}
		sc.ComputedAt = time.Unix(computedAt, 0).UTC()
		scores = append(scores, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scoring: list scores rows: %w", err)
	}
	return scores, nil
}

// RoleReposWithOutcomes returns role+repo pairs with terminal outcomes since
// the cutoff. Empty repoID means all repos in the database.
func (s *Store) RoleReposWithOutcomes(ctx context.Context, repoID string, since time.Time) ([]RoleRepo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT role, repo_id
FROM outcomes
WHERE recorded_at >= ? AND (? = '' OR repo_id = ?)
GROUP BY role, repo_id
ORDER BY repo_id, role`, since.Unix(), repoID, repoID)
	if err != nil {
		return nil, fmt.Errorf("scoring: list role repos with outcomes: %w", err)
	}
	defer rows.Close()

	var pairs []RoleRepo
	for rows.Next() {
		var pair RoleRepo
		if err := rows.Scan(&pair.Role, &pair.RepoID); err != nil {
			return nil, fmt.Errorf("scoring: scan role repo: %w", err)
		}
		pairs = append(pairs, pair)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scoring: role repo rows: %w", err)
	}
	return pairs, nil
}

// OutcomeCounts returns aggregate terminal outcome counts since the cutoff.
// Empty repoID means all repos in the database.
func (s *Store) OutcomeCounts(ctx context.Context, repoID string, since time.Time) ([]OutcomeCount, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT role, repo_id, type, COUNT(*)
FROM outcomes
WHERE recorded_at >= ? AND (? = '' OR repo_id = ?)
GROUP BY role, repo_id, type
ORDER BY repo_id, role, type`, since.Unix(), repoID, repoID)
	if err != nil {
		return nil, fmt.Errorf("scoring: outcome counts: %w", err)
	}
	defer rows.Close()

	var counts []OutcomeCount
	for rows.Next() {
		var count OutcomeCount
		var typ string
		if err := rows.Scan(&count.Role, &count.RepoID, &typ, &count.Count); err != nil {
			return nil, fmt.Errorf("scoring: scan outcome count: %w", err)
		}
		count.Type = OutcomeType(typ)
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scoring: outcome count rows: %w", err)
	}
	return counts, nil
}

// OutcomesSince returns terminal outcomes with details since the cutoff. Empty
// repoID means all repos in the database.
func (s *Store) OutcomesSince(ctx context.Context, repoID string, since time.Time) ([]Outcome, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, job_id, repo_id, role, type, details, recorded_at
FROM outcomes
WHERE recorded_at >= ? AND (? = '' OR repo_id = ?)
ORDER BY recorded_at DESC`, since.Unix(), repoID, repoID)
	if err != nil {
		return nil, fmt.Errorf("scoring: outcomes since: %w", err)
	}
	defer rows.Close()

	var outcomes []Outcome
	for rows.Next() {
		var outcome Outcome
		var typ string
		var recordedAt int64
		if err := rows.Scan(&outcome.ID, &outcome.JobID, &outcome.RepoID, &outcome.Role, &typ, &outcome.Details, &recordedAt); err != nil {
			return nil, fmt.Errorf("scoring: scan outcome: %w", err)
		}
		outcome.Type = OutcomeType(typ)
		outcome.RecordedAt = time.Unix(recordedAt, 0).UTC()
		outcomes = append(outcomes, outcome)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scoring: outcomes rows: %w", err)
	}
	return outcomes, nil
}

func newUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
