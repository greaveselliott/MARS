/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/github-app-integration.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
*/
package serve

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	gh "github.com/greaveselliott/mars/internal/github"
	_ "modernc.org/sqlite"
)

// RepoRecord represents a registered repository.
type RepoRecord struct {
	ID      string
	Path    string
	Remote  string
	Branch  string
	AddedAt time.Time
}

// RepoRegistry manages registered repositories in SQLite.
type RepoRegistry struct {
	db *sql.DB
}

// NewRepoRegistry opens or creates the repos table in the given DB.
func NewRepoRegistry(db *sql.DB) (*RepoRegistry, error) {
	if db == nil {
		return nil, fmt.Errorf("serve: NewRepoRegistry requires a non-nil *sql.DB — open a database first")
	}

	const ddl = `CREATE TABLE IF NOT EXISTS repos (
		id TEXT PRIMARY KEY,
		path TEXT NOT NULL UNIQUE,
		remote TEXT NOT NULL DEFAULT '',
		branch TEXT NOT NULL DEFAULT 'main',
		added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(ddl); err != nil {
		return nil, fmt.Errorf("serve: failed to create repos table — check SQLite permissions: %w", err)
	}

	return &RepoRegistry{db: db}, nil
}

// Register adds a repo. Validates that .harness/manifest.yaml exists at path.
// Uses a UUID as ID. Upserts by path (if already registered, updates remote/branch).
func (r *RepoRegistry) Register(ctx context.Context, path, remote, branch string) (string, error) {
	manifestPath := filepath.Join(path, ".harness", "manifest.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		return "", fmt.Errorf("serve: cannot register repo at %s — missing .harness/manifest.yaml; run `mars init` in the repo first", path)
	}

	normalizedRemote, err := NormalizeRemote(remote)
	if err != nil {
		return "", err
	}
	requestedBranch := branch
	if branch != "" {
		if err := gh.ValidateBranch(branch); err != nil {
			return "", fmt.Errorf("serve: %w", err)
		}
	}

	id, err := newUUID()
	if err != nil {
		return "", fmt.Errorf("serve: failed to generate repo ID: %w", err)
	}

	const query = `
		INSERT INTO repos (id, path, remote, branch)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			remote = CASE WHEN excluded.remote = '' THEN repos.remote ELSE excluded.remote END,
			branch = CASE WHEN ? = '' THEN repos.branch ELSE excluded.branch END`

	if branch == "" {
		branch = "main"
	}
	if _, err := r.db.ExecContext(ctx, query, id, path, normalizedRemote, branch, requestedBranch); err != nil {
		return "", fmt.Errorf("serve: failed to register repo at %s: %w", path, err)
	}

	// On upsert the original ID is preserved; fetch it back.
	var actualID, actualRemote, actualBranch string
	if err := r.db.QueryRowContext(ctx, "SELECT id, remote, branch FROM repos WHERE path = ?", path).Scan(&actualID, &actualRemote, &actualBranch); err != nil {
		return "", fmt.Errorf("serve: registered repo but failed to read back ID: %w", err)
	}

	slog.Info("serve: repo registered", "id", actualID, "path", path, "remote", actualRemote, "branch", actualBranch)
	return actualID, nil
}

// List returns all registered repos.
func (r *RepoRegistry) List(ctx context.Context) ([]RepoRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, path, remote, branch, added_at FROM repos ORDER BY added_at")
	if err != nil {
		return nil, fmt.Errorf("serve: failed to list repos: %w", err)
	}
	defer rows.Close()

	var repos []RepoRecord
	for rows.Next() {
		var rec RepoRecord
		if err := rows.Scan(&rec.ID, &rec.Path, &rec.Remote, &rec.Branch, &rec.AddedAt); err != nil {
			return nil, fmt.Errorf("serve: failed to scan repo row: %w", err)
		}
		repos = append(repos, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("serve: error iterating repo rows: %w", err)
	}
	return repos, nil
}

// FindByID finds a repo by its ID.
func (r *RepoRegistry) FindByID(ctx context.Context, id string) (*RepoRecord, error) {
	var rec RepoRecord
	err := r.db.QueryRowContext(ctx,
		"SELECT id, path, remote, branch, added_at FROM repos WHERE id = ?", id,
	).Scan(&rec.ID, &rec.Path, &rec.Remote, &rec.Branch, &rec.AddedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("serve: failed to find repo by id %q: %w", id, err)
	}
	return &rec, nil
}

// FindByRemote finds a repo by its GitHub full_name.
func (r *RepoRegistry) FindByRemote(ctx context.Context, fullName string) (*RepoRecord, error) {
	fullName, err := NormalizeRemote(fullName)
	if err != nil || fullName == "" {
		return nil, nil
	}
	var rec RepoRecord
	err = r.db.QueryRowContext(ctx,
		"SELECT id, path, remote, branch, added_at FROM repos WHERE lower(remote) = ?", fullName,
	).Scan(&rec.ID, &rec.Path, &rec.Remote, &rec.Branch, &rec.AddedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("serve: failed to find repo by remote %q: %w", fullName, err)
	}
	return &rec, nil
}

// HasWebhookRepositories reports whether at least one exact non-empty remote
// is registered. Empty remotes never enable GitHub webhook ingress.
func (r *RepoRegistry) HasWebhookRepositories(ctx context.Context) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM repos WHERE remote != '' AND branch != ''`).Scan(&count); err != nil {
		return false, fmt.Errorf("serve: count webhook repositories: %w", err)
	}
	return count > 0, nil
}

// NormalizeRemote validates and case-normalizes a GitHub owner/repo identity.
// An empty value is allowed so local-only repositories can remain registered.
func NormalizeRemote(remote string) (string, error) {
	if remote == "" {
		return "", nil
	}
	normalized, err := gh.NormalizeRepository(remote)
	if err != nil {
		return "", fmt.Errorf("serve: %w; use --remote myorg/myrepo", err)
	}
	return normalized, nil
}

// Remove unregisters a repo by ID.
func (r *RepoRegistry) Remove(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM repos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("serve: failed to remove repo %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("serve: repo %s not found — it may have already been removed", id)
	}
	slog.Info("serve: repo removed", "id", id)
	return nil
}

// newUUID generates a v4 UUID using crypto/rand.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
