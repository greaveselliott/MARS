/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/scoring-system.md
- docs/features/F-008-scoring-trust-quality.md
*/
package trust

import (
	"context"
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

// Level represents the progressive trust level.
type Level string

const (
	LevelObserver    Level = "observer"
	LevelContributor Level = "contributor"
	LevelAutonomous  Level = "autonomous"
)

// ParseLevel normalizes configured trust levels. "operator" is accepted as a
// legacy manifest alias for contributor because early deployed harness learnings
// used that term before manifest trust defaults were wired into runtime policy.
func ParseLevel(raw string) (Level, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return "", false
	case string(LevelObserver):
		return LevelObserver, true
	case string(LevelContributor), "operator":
		return LevelContributor, true
	case string(LevelAutonomous):
		return LevelAutonomous, true
	default:
		return "", false
	}
}

const (
	observerTrialThreshold     = 5
	autonomousScoreThreshold   = 0.8
	autonomousCountThreshold   = 20
	demoteAutonomousThreshold  = 0.6
	demoteContributorThreshold = 0.3
)

// Capabilities returns what a trust level allows.
func Capabilities(l Level) []string {
	switch l {
	case LevelObserver:
		return []string{"repo_read", "file_read", "file_search", "grep", "git_status", "git_diff"}
	case LevelContributor:
		return []string{"repo_read", "repo_write", "file_read", "file_search", "grep", "shell_exec", "file_write", "ticket_create", "record_decision", "git_status", "git_diff", "git_commit", "git_push_main"}
	case LevelAutonomous:
		return []string{"repo_read", "repo_write", "file_read", "file_search", "grep", "shell_exec", "file_write", "ticket_create", "record_decision", "git_status", "git_diff", "git_commit", "git_push_main", "self_schedule", "evolve_policy"}
	default:
		return nil
	}
}

// Entry is a trust record for a role+repo combination.
type Entry struct {
	Role      string
	RepoID    string
	Level     Level
	TrialRuns int // completed runs at current level
	UpdatedAt time.Time
}

// Evaluate checks if a role should be promoted or demoted based on the current
// entry, accuracy score, and total outcome count. Returns the recommended level.
func Evaluate(entry *Entry, score float64, outcomeCount int) Level {
	if entry == nil {
		return LevelObserver
	}

	switch entry.Level {
	case LevelObserver:
		if entry.TrialRuns >= observerTrialThreshold {
			return LevelContributor
		}
		return LevelObserver

	case LevelContributor:
		if score < demoteContributorThreshold {
			return LevelObserver
		}
		if score >= autonomousScoreThreshold && outcomeCount >= autonomousCountThreshold {
			return LevelAutonomous
		}
		return LevelContributor

	case LevelAutonomous:
		if score < demoteAutonomousThreshold {
			return LevelContributor
		}
		return LevelAutonomous

	default:
		return LevelObserver
	}
}

// Store manages trust levels in SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// OpenStore opens or creates a SQLite-backed trust store at dbPath.
func OpenStore(dbPath string) (*Store, error) {
	if err := validateDBPath(dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("trust: open sqlite %q: %w", dbPath, err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		if isSQLiteOpenUnavailable(err) {
			return nil, fmt.Errorf("trust: database at %s is unavailable — run `mars-harness setup`, run `mars-harness register --repo <path>`, or pass --db with a writable SQLite path", dbPath)
		}
		return nil, err
	}
	return s, nil
}

func validateDBPath(dbPath string) error {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return fmt.Errorf("trust: database path is empty — pass --db <path> or run `mars-harness register --repo <path>`")
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
			return fmt.Errorf("trust: database directory %s does not exist for %s — run `mars-harness setup`, run `mars-harness register --repo <path>`, or create the directory before retrying", dir, dbPath)
		}
		return fmt.Errorf("trust: check database directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("trust: database parent %s is not a directory — pass --db with a writable database file path", dir)
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
CREATE TABLE IF NOT EXISTS trust_entries (
  role       TEXT NOT NULL,
  repo_id    TEXT NOT NULL,
  level      TEXT NOT NULL DEFAULT 'observer',
  trial_runs INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (role, repo_id)
);
CREATE TABLE IF NOT EXISTS trust_events (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  role       TEXT NOT NULL,
  repo_id    TEXT NOT NULL,
  level      TEXT NOT NULL,
  reason     TEXT NOT NULL,
  recorded_at INTEGER NOT NULL
);
	`)
	if err != nil {
		return fmt.Errorf("trust: init schema: %w", err)
	}
	return nil
}

// List returns all trust entries ordered by repo and role.
func (s *Store) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT role, repo_id, level, trial_runs, updated_at
FROM trust_entries
ORDER BY repo_id, role`)
	if err != nil {
		return nil, fmt.Errorf("trust: list: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var updatedAt int64
		if err := rows.Scan(&e.Role, &e.RepoID, (*string)(&e.Level), &e.TrialRuns, &updatedAt); err != nil {
			return nil, fmt.Errorf("trust: list scan: %w", err)
		}
		e.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("trust: list rows: %w", err)
	}
	return entries, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Get retrieves the trust entry for a role+repo. Returns nil if not found.
func (s *Store) Get(ctx context.Context, role, repoID string) (*Entry, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT role, repo_id, level, trial_runs, updated_at
FROM trust_entries WHERE role = ? AND repo_id = ?`, role, repoID)

	var e Entry
	var updatedAt int64
	err := row.Scan(&e.Role, &e.RepoID, (*string)(&e.Level), &e.TrialRuns, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("trust: get %q/%q: %w", role, repoID, err)
	}
	e.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &e, nil
}

// Set creates or updates the trust level for a role+repo. Resets trial runs
// when the level changes.
func (s *Store) Set(ctx context.Context, role, repoID string, level Level) error {
	return s.SetWithReason(ctx, role, repoID, level, "")
}

// SetWithReason creates or updates the trust level and records an audit event when reason is provided.
func (s *Store) SetWithReason(ctx context.Context, role, repoID string, level Level, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	existing, err := s.getLocked(ctx, role, repoID)
	if err != nil {
		return err
	}

	trialRuns := 0
	if existing != nil && existing.Level == level {
		trialRuns = existing.TrialRuns
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO trust_entries(role, repo_id, level, trial_runs, updated_at)
VALUES(?,?,?,?,?)
ON CONFLICT(role, repo_id) DO UPDATE SET
  level=excluded.level, trial_runs=excluded.trial_runs, updated_at=excluded.updated_at`,
		role, repoID, string(level), trialRuns, now.Unix())
	if err != nil {
		return fmt.Errorf("trust: set %q/%q: %w", role, repoID, err)
	}
	if strings.TrimSpace(reason) != "" {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO trust_events(role, repo_id, level, reason, recorded_at)
VALUES(?,?,?,?,?)`, role, repoID, string(level), strings.TrimSpace(reason), now.Unix()); err != nil {
			return fmt.Errorf("trust: record event %q/%q: %w", role, repoID, err)
		}
	}
	slog.Debug("trust: set level", "role", role, "repo", repoID, "level", level)
	return nil
}

// IncrementTrialRuns increments the trial run count for a role+repo.
// If no entry exists, one is created at observer level with trial_runs=1.
func (s *Store) IncrementTrialRuns(ctx context.Context, role, repoID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
INSERT INTO trust_entries(role, repo_id, level, trial_runs, updated_at)
VALUES(?,?,?,1,?)
ON CONFLICT(role, repo_id) DO UPDATE SET
  trial_runs = trial_runs + 1, updated_at = excluded.updated_at`,
		role, repoID, string(LevelObserver), now.Unix())
	if err != nil {
		return fmt.Errorf("trust: increment trial runs %q/%q: %w", role, repoID, err)
	}
	return nil
}

func (s *Store) getLocked(ctx context.Context, role, repoID string) (*Entry, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT role, repo_id, level, trial_runs, updated_at
FROM trust_entries WHERE role = ? AND repo_id = ?`, role, repoID)

	var e Entry
	var updatedAt int64
	err := row.Scan(&e.Role, &e.RepoID, (*string)(&e.Level), &e.TrialRuns, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("trust: get locked %q/%q: %w", role, repoID, err)
	}
	e.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &e, nil
}
