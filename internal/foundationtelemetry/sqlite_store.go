/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package foundationtelemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore persists anonymous foundation telemetry intake locally.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLiteStore opens or creates a collector SQLite database.
func OpenSQLiteStore(path string) (*SQLiteStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("foundation telemetry: sqlite db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("foundation telemetry: create sqlite directory: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("foundation telemetry: open sqlite %q: %w", path, err)
	}
	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS foundation_telemetry_reports (
  id TEXT PRIMARY KEY,
  received_at INTEGER NOT NULL,
  schema_version INTEGER NOT NULL,
  report_key TEXT NOT NULL,
  payload_hash TEXT NOT NULL UNIQUE,
  payload_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS foundation_telemetry_patterns (
  signature TEXT PRIMARY KEY,
  first_seen INTEGER NOT NULL,
  last_seen INTEGER NOT NULL,
  report_count INTEGER NOT NULL,
  install_window_count INTEGER NOT NULL,
  harness_versions TEXT NOT NULL,
  category TEXT NOT NULL,
  target TEXT NOT NULL,
  severity TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_foundation_telemetry_patterns_last_seen
ON foundation_telemetry_patterns(last_seen);

CREATE TABLE IF NOT EXISTS foundation_telemetry_pattern_reports (
  signature TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  report_key TEXT NOT NULL,
  PRIMARY KEY(signature, payload_hash)
);

CREATE TABLE IF NOT EXISTS foundation_telemetry_pattern_report_keys (
  signature TEXT NOT NULL,
  report_key TEXT NOT NULL,
  PRIMARY KEY(signature, report_key)
);
`)
	if err != nil {
		return fmt.Errorf("foundation telemetry: init sqlite schema: %w", err)
	}
	return nil
}

// Close closes the store.
func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SaveReport stores the raw anonymous envelope exactly as received.
func (s *SQLiteStore) SaveReport(ctx context.Context, report AnonymousReport) error {
	if s == nil || s.db == nil {
		return nil
	}
	if err := ValidateReport(report); err != nil {
		return err
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("foundation telemetry: marshal report: %w", err)
	}
	hash, err := PayloadHash(report)
	if err != nil {
		return err
	}
	id := "ftr-" + hash[:16]
	_, err = s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO foundation_telemetry_reports(
  id, received_at, schema_version, report_key, payload_hash, payload_json
) VALUES(?,?,?,?,?,?)`,
		id, time.Now().UTC().Unix(), report.SchemaVersion, report.ReportKey, hash, string(payload))
	if err != nil {
		return fmt.Errorf("foundation telemetry: save report: %w", err)
	}
	return nil
}

// UpsertPattern merges a cross-report aggregate.
func (s *SQLiteStore) UpsertPattern(ctx context.Context, pattern AggregatedPattern) error {
	if s == nil || s.db == nil {
		return nil
	}
	if strings.TrimSpace(pattern.Signature) == "" {
		return fmt.Errorf("foundation telemetry: pattern signature is required")
	}
	if pattern.FirstSeen.IsZero() {
		pattern.FirstSeen = time.Now().UTC()
	}
	if pattern.LastSeen.IsZero() {
		pattern.LastSeen = pattern.FirstSeen
	}
	if pattern.ReportCount <= 0 {
		pattern.ReportCount = 1
	}
	if pattern.InstallWindowCount <= 0 {
		pattern.InstallWindowCount = 1
	}
	reportDelta := pattern.ReportCount
	installDelta := pattern.InstallWindowCount

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("foundation telemetry: begin pattern upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if strings.TrimSpace(pattern.ReportHash) != "" {
		reportDelta = 0
		installDelta = 0
		reportKey := strings.TrimSpace(pattern.ReportKey)
		if reportKey == "" {
			reportKey = "unknown"
		}
		result, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO foundation_telemetry_pattern_reports(signature, payload_hash, report_key)
VALUES(?,?,?)`, pattern.Signature, pattern.ReportHash, reportKey)
		if err != nil {
			return fmt.Errorf("foundation telemetry: track pattern report: %w", err)
		}
		if rows, _ := result.RowsAffected(); rows > 0 {
			reportDelta = 1
		}
		result, err = tx.ExecContext(ctx, `
INSERT OR IGNORE INTO foundation_telemetry_pattern_report_keys(signature, report_key)
VALUES(?,?)`, pattern.Signature, reportKey)
		if err != nil {
			return fmt.Errorf("foundation telemetry: track pattern report key: %w", err)
		}
		if rows, _ := result.RowsAffected(); rows > 0 {
			installDelta = 1
		}
	}

	var existing AggregatedPattern
	var versionsJSON string
	row := tx.QueryRowContext(ctx, `
SELECT signature, first_seen, last_seen, report_count, install_window_count, harness_versions, category, target, severity
FROM foundation_telemetry_patterns
WHERE signature = ?`, pattern.Signature)
	var firstSeen, lastSeen int64
	err = row.Scan(&existing.Signature, &firstSeen, &lastSeen, &existing.ReportCount, &existing.InstallWindowCount, &versionsJSON, &existing.Category, &existing.Target, &existing.Severity)
	switch {
	case err == sql.ErrNoRows:
		versions, marshalErr := json.Marshal(mergeVersions(nil, pattern.HarnessVersions))
		if marshalErr != nil {
			return fmt.Errorf("foundation telemetry: marshal versions: %w", marshalErr)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO foundation_telemetry_patterns(
  signature, first_seen, last_seen, report_count, install_window_count, harness_versions, category, target, severity
) VALUES(?,?,?,?,?,?,?,?,?)`,
			pattern.Signature, pattern.FirstSeen.Unix(), pattern.LastSeen.Unix(), reportDelta,
			installDelta, string(versions), pattern.Category, pattern.Target, pattern.Severity)
		if err != nil {
			return fmt.Errorf("foundation telemetry: insert pattern: %w", err)
		}
	case err != nil:
		return fmt.Errorf("foundation telemetry: load pattern: %w", err)
	default:
		existing.FirstSeen = time.Unix(firstSeen, 0).UTC()
		existing.LastSeen = time.Unix(lastSeen, 0).UTC()
		_ = json.Unmarshal([]byte(versionsJSON), &existing.HarnessVersions)
		if pattern.FirstSeen.Before(existing.FirstSeen) {
			existing.FirstSeen = pattern.FirstSeen
		}
		if pattern.LastSeen.After(existing.LastSeen) {
			existing.LastSeen = pattern.LastSeen
		}
		versions, marshalErr := json.Marshal(mergeVersions(existing.HarnessVersions, pattern.HarnessVersions))
		if marshalErr != nil {
			return fmt.Errorf("foundation telemetry: marshal merged versions: %w", marshalErr)
		}
		_, err = tx.ExecContext(ctx, `
UPDATE foundation_telemetry_patterns
SET first_seen = ?, last_seen = ?, report_count = ?, install_window_count = ?,
    harness_versions = ?, category = ?, target = ?, severity = ?
WHERE signature = ?`,
			existing.FirstSeen.Unix(), existing.LastSeen.Unix(),
			existing.ReportCount+reportDelta,
			existing.InstallWindowCount+installDelta,
			string(versions), defaultString(pattern.Category, existing.Category),
			defaultString(pattern.Target, existing.Target),
			maxSeverity(existing.Severity, pattern.Severity), pattern.Signature)
		if err != nil {
			return fmt.Errorf("foundation telemetry: update pattern: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("foundation telemetry: commit pattern upsert: %w", err)
	}
	return nil
}

// PatternsSince returns aggregate patterns seen since the given time.
func (s *SQLiteStore) PatternsSince(ctx context.Context, since time.Time) ([]AggregatedPattern, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT signature, first_seen, last_seen, report_count, install_window_count, harness_versions, category, target, severity
FROM foundation_telemetry_patterns
WHERE last_seen >= ?
ORDER BY last_seen DESC`, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("foundation telemetry: query patterns: %w", err)
	}
	defer rows.Close()

	var out []AggregatedPattern
	for rows.Next() {
		var p AggregatedPattern
		var firstSeen, lastSeen int64
		var versionsJSON string
		if err := rows.Scan(&p.Signature, &firstSeen, &lastSeen, &p.ReportCount, &p.InstallWindowCount, &versionsJSON, &p.Category, &p.Target, &p.Severity); err != nil {
			return nil, fmt.Errorf("foundation telemetry: scan pattern: %w", err)
		}
		p.FirstSeen = time.Unix(firstSeen, 0).UTC()
		p.LastSeen = time.Unix(lastSeen, 0).UTC()
		_ = json.Unmarshal([]byte(versionsJSON), &p.HarnessVersions)
		out = append(out, p)
	}
	return out, rows.Err()
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func maxSeverity(left, right string) string {
	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	if rank[strings.ToLower(right)] > rank[strings.ToLower(left)] {
		return right
	}
	return left
}
