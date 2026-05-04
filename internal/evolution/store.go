/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-improvement.md
- docs/features/F-012-self-improvement-loop.md
*/
package evolution

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists interventions and evolution history in SQLite (MH-019, MH-020).
type Store struct {
	db *sql.DB
}

// OpenStore opens or creates a SQLite database for evolution tracking.
func OpenStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("evolution: open sqlite %q: %w", dbPath, err)
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
CREATE TABLE IF NOT EXISTS interventions (
  id          TEXT PRIMARY KEY,
  job_id      TEXT NOT NULL,
  repo_id     TEXT NOT NULL,
  role        TEXT NOT NULL,
  type        TEXT NOT NULL,
  evidence    TEXT NOT NULL DEFAULT '',
  detected_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_interventions_repo_role ON interventions(repo_id, role);

CREATE TABLE IF NOT EXISTS evolutions (
  id           TEXT PRIMARY KEY,
  role         TEXT NOT NULL,
  repo_id      TEXT NOT NULL,
  result       TEXT NOT NULL DEFAULT '',
  score_before REAL NOT NULL DEFAULT 0,
  score_after  REAL NOT NULL DEFAULT 0,
  created_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_evolutions_role ON evolutions(role);
CREATE INDEX IF NOT EXISTS idx_evolutions_role_created ON evolutions(role, created_at);
`)
	if err != nil {
		return fmt.Errorf("evolution: init schema: %w", err)
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

// SaveIntervention persists a detected intervention.
func (s *Store) SaveIntervention(ctx context.Context, i Intervention) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("evolution: store is nil")
	}
	if i.ID == "" {
		i.ID = newUUID()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO interventions(id, job_id, repo_id, role, type, evidence, detected_at)
VALUES(?,?,?,?,?,?,?)`,
		i.ID, i.JobID, i.RepoID, i.Role, string(i.Type), i.Evidence, i.DetectedAt.Unix())
	if err != nil {
		return fmt.Errorf("evolution: save intervention %q: %w", i.ID, err)
	}
	slog.Debug("evolution: saved intervention", "id", i.ID, "type", i.Type, "role", i.Role)
	return nil
}

// GetInterventions returns the most recent interventions for a repo+role pair.
func (s *Store) GetInterventions(ctx context.Context, repoID, role string, limit int) ([]Intervention, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("evolution: store is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, job_id, repo_id, role, type, evidence, detected_at
FROM interventions
WHERE repo_id = ? AND role = ?
ORDER BY detected_at DESC
LIMIT ?`, repoID, role, limit)
	if err != nil {
		return nil, fmt.Errorf("evolution: get interventions: %w", err)
	}
	defer rows.Close()

	var result []Intervention
	for rows.Next() {
		var iv Intervention
		var ivType string
		var detectedAt int64
		if err := rows.Scan(&iv.ID, &iv.JobID, &iv.RepoID, &iv.Role, &ivType, &iv.Evidence, &detectedAt); err != nil {
			return nil, fmt.Errorf("evolution: scan intervention: %w", err)
		}
		iv.Type = InterventionType(ivType)
		iv.DetectedAt = time.Unix(detectedAt, 0).UTC()
		result = append(result, iv)
	}
	return result, rows.Err()
}

// SaveEvolution persists a Reviewer run result.
func (s *Store) SaveEvolution(ctx context.Context, e Evolution) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("evolution: store is nil")
	}
	if e.ID == "" {
		e.ID = newUUID()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO evolutions(id, role, repo_id, result, score_before, score_after, created_at)
VALUES(?,?,?,?,?,?,?)`,
		e.ID, e.Role, e.RepoID, e.Result, e.ScoreBefore, e.ScoreAfter, e.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("evolution: save evolution %q: %w", e.ID, err)
	}
	slog.Debug("evolution: saved evolution", "id", e.ID, "role", e.Role)
	return nil
}

// GetEvolutions returns the most recent evolutions for a role.
func (s *Store) GetEvolutions(ctx context.Context, role string, limit int) ([]Evolution, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("evolution: store is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, role, repo_id, result, score_before, score_after, created_at
FROM evolutions
WHERE role = ?
ORDER BY created_at DESC
LIMIT ?`, role, limit)
	if err != nil {
		return nil, fmt.Errorf("evolution: get evolutions: %w", err)
	}
	defer rows.Close()

	var result []Evolution
	for rows.Next() {
		var ev Evolution
		var createdAt int64
		if err := rows.Scan(&ev.ID, &ev.Role, &ev.RepoID, &ev.Result, &ev.ScoreBefore, &ev.ScoreAfter, &createdAt); err != nil {
			return nil, fmt.Errorf("evolution: scan evolution: %w", err)
		}
		ev.CreatedAt = time.Unix(createdAt, 0).UTC()
		result = append(result, ev)
	}
	return result, rows.Err()
}

// CountRecentEvolutions counts evolutions for a role since a given time.
func (s *Store) CountRecentEvolutions(ctx context.Context, role string, since time.Time) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("evolution: store is nil")
	}
	var count int
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM evolutions
WHERE role = ? AND created_at >= ?`, role, since.Unix()).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("evolution: count recent evolutions: %w", err)
	}
	return count, nil
}

func newUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
