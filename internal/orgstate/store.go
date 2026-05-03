package orgstate

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Disposition is the structured outcome a role records before dispatch routing.
type Disposition struct {
	JobID          string    `json:"job_id"`
	RepoID         string    `json:"repo_id"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	NextNeed       string    `json:"next_need,omitempty"`
	SuggestedRole  string    `json:"suggested_role,omitempty"`
	TicketID       string    `json:"ticket_id,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	EvidenceLinks  []string  `json:"evidence_links,omitempty"`
	ApprovalID     string    `json:"approval_id,omitempty"`
	WorkProductIDs []string  `json:"work_product_ids,omitempty"`
	BlockedBy      []string  `json:"blocked_by,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	RecordedAt     time.Time `json:"recorded_at"`
}

// Decision is the deterministic routing decision recorded after a disposition.
type Decision struct {
	ID              string    `json:"id"`
	JobID           string    `json:"job_id"`
	RepoID          string    `json:"repo_id"`
	SourceRole      string    `json:"source_role"`
	TicketID        string    `json:"ticket_id,omitempty"`
	NextNeed        string    `json:"next_need,omitempty"`
	NextRole        string    `json:"next_role,omitempty"`
	DecisionKind    string    `json:"decision_kind"`
	Reason          string    `json:"reason"`
	StopReason      string    `json:"stop_reason,omitempty"`
	TicketStateHash string    `json:"ticket_state_hash,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// Store persists orchestration state in SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// OpenStore opens or creates a SQLite-backed orgstate store at dbPath.
func OpenStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("orgstate: open sqlite %q: %w", dbPath, err)
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
CREATE TABLE IF NOT EXISTS job_dispositions (
  job_id            TEXT PRIMARY KEY,
  repo_id           TEXT NOT NULL,
  role              TEXT NOT NULL,
  status            TEXT NOT NULL,
  next_need         TEXT NOT NULL DEFAULT '',
  suggested_role    TEXT NOT NULL DEFAULT '',
  ticket_id         TEXT NOT NULL DEFAULT '',
  reason            TEXT NOT NULL DEFAULT '',
  evidence_json     TEXT NOT NULL DEFAULT '[]',
  approval_id       TEXT NOT NULL DEFAULT '',
  work_products_json TEXT NOT NULL DEFAULT '[]',
  blocked_by_json   TEXT NOT NULL DEFAULT '[]',
  trace_id          TEXT NOT NULL DEFAULT '',
  recorded_at       INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_job_dispositions_repo_time ON job_dispositions(repo_id, recorded_at);

CREATE TABLE IF NOT EXISTS orchestration_decisions (
  id                TEXT PRIMARY KEY,
  job_id            TEXT NOT NULL,
  repo_id           TEXT NOT NULL,
  source_role       TEXT NOT NULL,
  ticket_id         TEXT NOT NULL DEFAULT '',
  next_need         TEXT NOT NULL DEFAULT '',
  next_role         TEXT NOT NULL DEFAULT '',
  decision_kind     TEXT NOT NULL,
  reason            TEXT NOT NULL,
  stop_reason       TEXT NOT NULL DEFAULT '',
  ticket_state_hash TEXT NOT NULL DEFAULT '',
  created_at        INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_orchestration_decisions_repo_time ON orchestration_decisions(repo_id, created_at);
CREATE INDEX IF NOT EXISTS idx_orchestration_decisions_loop ON orchestration_decisions(repo_id, ticket_id, next_need, next_role, ticket_state_hash);

CREATE TABLE IF NOT EXISTS approvals (
  id          TEXT PRIMARY KEY,
  repo_id     TEXT NOT NULL,
  ticket_id   TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL,
  requested_by TEXT NOT NULL DEFAULT '',
  reviewer_role TEXT NOT NULL DEFAULT '',
  reason      TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL,
  resolved_at INTEGER
);

CREATE TABLE IF NOT EXISTS work_products (
  id          TEXT PRIMARY KEY,
  repo_id     TEXT NOT NULL,
  ticket_id   TEXT NOT NULL DEFAULT '',
  job_id      TEXT NOT NULL DEFAULT '',
  role        TEXT NOT NULL DEFAULT '',
  kind        TEXT NOT NULL DEFAULT '',
  title       TEXT NOT NULL DEFAULT '',
  path_or_url TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS organization_repos (
  organization_id TEXT NOT NULL,
  repo_id         TEXT NOT NULL,
  added_at        INTEGER NOT NULL,
  PRIMARY KEY (organization_id, repo_id)
);
`)
	if err != nil {
		return fmt.Errorf("orgstate: init schema: %w", err)
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

// RecordDisposition validates and upserts a job disposition.
func (s *Store) RecordDisposition(ctx context.Context, d Disposition) error {
	if err := ValidateDisposition(d); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if d.RecordedAt.IsZero() {
		d.RecordedAt = time.Now().UTC()
	}
	evidence, _ := json.Marshal(d.EvidenceLinks)
	workProducts, _ := json.Marshal(d.WorkProductIDs)
	blockedBy, _ := json.Marshal(d.BlockedBy)

	_, err := s.db.ExecContext(ctx, `
INSERT INTO job_dispositions(job_id, repo_id, role, status, next_need, suggested_role, ticket_id, reason, evidence_json, approval_id, work_products_json, blocked_by_json, trace_id, recorded_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(job_id) DO UPDATE SET
  repo_id=excluded.repo_id, role=excluded.role, status=excluded.status,
  next_need=excluded.next_need, suggested_role=excluded.suggested_role,
  ticket_id=excluded.ticket_id, reason=excluded.reason, evidence_json=excluded.evidence_json,
  approval_id=excluded.approval_id, work_products_json=excluded.work_products_json,
  blocked_by_json=excluded.blocked_by_json, trace_id=excluded.trace_id,
  recorded_at=excluded.recorded_at`,
		d.JobID, d.RepoID, d.Role, d.Status, d.NextNeed, d.SuggestedRole, d.TicketID, d.Reason,
		string(evidence), d.ApprovalID, string(workProducts), string(blockedBy), d.TraceID, d.RecordedAt.Unix())
	if err != nil {
		return fmt.Errorf("orgstate: record disposition: %w", err)
	}
	return nil
}

// ValidateDisposition checks the minimum contract for dispatch-mode routing.
func ValidateDisposition(d Disposition) error {
	if strings.TrimSpace(d.JobID) == "" {
		return fmt.Errorf("orgstate: disposition job_id is required")
	}
	if strings.TrimSpace(d.RepoID) == "" {
		return fmt.Errorf("orgstate: disposition repo_id is required")
	}
	if strings.TrimSpace(d.Role) == "" {
		return fmt.Errorf("orgstate: disposition role is required")
	}
	switch strings.TrimSpace(d.Status) {
	case "completed", "approved", "blocked", "in_review", "changes_requested", "no_work", "failed", "ambiguous":
	default:
		return fmt.Errorf("orgstate: invalid disposition status %q", d.Status)
	}
	if strings.TrimSpace(d.Reason) == "" && d.Status != "completed" && d.Status != "approved" {
		return fmt.Errorf("orgstate: disposition reason is required for status %q", d.Status)
	}
	if d.Status == "blocked" && strings.TrimSpace(d.NextNeed) == "" && strings.TrimSpace(d.SuggestedRole) == "" {
		return fmt.Errorf("orgstate: blocked disposition requires next_need or suggested_role")
	}
	return nil
}

// GetDisposition returns the recorded disposition for a job.
func (s *Store) GetDisposition(ctx context.Context, jobID string) (*Disposition, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT job_id, repo_id, role, status, next_need, suggested_role, ticket_id, reason, evidence_json, approval_id, work_products_json, blocked_by_json, trace_id, recorded_at
FROM job_dispositions WHERE job_id = ?`, jobID)
	d, err := scanDisposition(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// RecentDispositions returns recent dispositions for a repo, newest first.
func (s *Store) RecentDispositions(ctx context.Context, repoID string, limit int) ([]Disposition, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT job_id, repo_id, role, status, next_need, suggested_role, ticket_id, reason, evidence_json, approval_id, work_products_json, blocked_by_json, trace_id, recorded_at
FROM job_dispositions
WHERE (? = '' OR repo_id = ?)
ORDER BY recorded_at DESC
LIMIT ?`, repoID, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("orgstate: recent dispositions: %w", err)
	}
	defer rows.Close()
	var out []Disposition
	for rows.Next() {
		d, err := scanDisposition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RecordDecision persists an orchestration decision.
func (s *Store) RecordDecision(ctx context.Context, d Decision) (Decision, error) {
	if strings.TrimSpace(d.ID) == "" {
		d.ID = newUUID()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(d.JobID) == "" || strings.TrimSpace(d.RepoID) == "" || strings.TrimSpace(d.SourceRole) == "" {
		return Decision{}, fmt.Errorf("orgstate: decision job_id, repo_id, and source_role are required")
	}
	if strings.TrimSpace(d.DecisionKind) == "" {
		return Decision{}, fmt.Errorf("orgstate: decision_kind is required")
	}
	if strings.TrimSpace(d.NextRole) == "" && strings.TrimSpace(d.StopReason) == "" {
		return Decision{}, fmt.Errorf("orgstate: decision requires next_role or stop_reason")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO orchestration_decisions(id, job_id, repo_id, source_role, ticket_id, next_need, next_role, decision_kind, reason, stop_reason, ticket_state_hash, created_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		d.ID, d.JobID, d.RepoID, d.SourceRole, d.TicketID, d.NextNeed, d.NextRole, d.DecisionKind, d.Reason, d.StopReason, d.TicketStateHash, d.CreatedAt.Unix())
	if err != nil {
		return Decision{}, fmt.Errorf("orgstate: record decision: %w", err)
	}
	return d, nil
}

// RecentDecisions returns recent decisions for loop detection or dashboard display.
func (s *Store) RecentDecisions(ctx context.Context, repoID string, limit int) ([]Decision, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, job_id, repo_id, source_role, ticket_id, next_need, next_role, decision_kind, reason, stop_reason, ticket_state_hash, created_at
FROM orchestration_decisions
WHERE (? = '' OR repo_id = ?)
ORDER BY created_at DESC
LIMIT ?`, repoID, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("orgstate: recent decisions: %w", err)
	}
	defer rows.Close()
	var out []Decision
	for rows.Next() {
		d, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDisposition(row scanner) (Disposition, error) {
	var d Disposition
	var evidence, workProducts, blockedBy string
	var recordedAt int64
	err := row.Scan(&d.JobID, &d.RepoID, &d.Role, &d.Status, &d.NextNeed, &d.SuggestedRole, &d.TicketID, &d.Reason, &evidence, &d.ApprovalID, &workProducts, &blockedBy, &d.TraceID, &recordedAt)
	if err != nil {
		return Disposition{}, err
	}
	_ = json.Unmarshal([]byte(evidence), &d.EvidenceLinks)
	_ = json.Unmarshal([]byte(workProducts), &d.WorkProductIDs)
	_ = json.Unmarshal([]byte(blockedBy), &d.BlockedBy)
	d.RecordedAt = time.Unix(recordedAt, 0).UTC()
	return d, nil
}

func scanDecision(row scanner) (Decision, error) {
	var d Decision
	var createdAt int64
	if err := row.Scan(&d.ID, &d.JobID, &d.RepoID, &d.SourceRole, &d.TicketID, &d.NextNeed, &d.NextRole, &d.DecisionKind, &d.Reason, &d.StopReason, &d.TicketStateHash, &createdAt); err != nil {
		return Decision{}, err
	}
	d.CreatedAt = time.Unix(createdAt, 0).UTC()
	return d, nil
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("decision-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
