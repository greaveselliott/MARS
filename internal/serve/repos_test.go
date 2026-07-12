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
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func makeHarnessDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	harnessDir := filepath.Join(dir, ".harness")
	require.NoError(t, os.MkdirAll(harnessDir, 0o755))
	manifest := []byte("name: test-repo\nroles:\n  ci-fixer:\n    prompt: prompts/ci.md\n    triggers:\n      - workflow_run.conclusion == \"failure\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(harnessDir, "manifest.yaml"), manifest, 0o644))
	return dir
}

func TestRepoRegistryLegacyFixture(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "legacy-repos.db")
	testutil.WriteSQLiteFixture(t, dbPath, `
CREATE TABLE repos (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  remote TEXT NOT NULL DEFAULT '',
  branch TEXT NOT NULL DEFAULT 'main',
  added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`, `
INSERT INTO repos(id, path, remote, branch, added_at)
VALUES('repo-legacy', '/tmp/legacy-target', 'owner/legacy', 'main', '2026-05-19 00:00:00');
`)
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)
	testutil.AssertSQLiteColumns(t, db, "repos", "id", "path", "remote", "branch", "added_at")

	repos, err := reg.List(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, "repo-legacy", repos[0].ID)
}

func TestRepoRegistry_Register_happy(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	repoDir := makeHarnessDir(t)
	ctx := context.Background()

	id, err := reg.Register(ctx, repoDir, "owner/repo", "main")
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	repos, err := reg.List(ctx)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, repoDir, repos[0].Path)
	assert.Equal(t, "owner/repo", repos[0].Remote)
	assert.Equal(t, "main", repos[0].Branch)
}

func TestRepoRegistry_Register_upsert(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	repoDir := makeHarnessDir(t)
	ctx := context.Background()

	id1, err := reg.Register(ctx, repoDir, "owner/repo", "main")
	require.NoError(t, err)

	id2, err := reg.Register(ctx, repoDir, "owner/repo", "develop")
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "upsert should preserve the original ID")

	repos, err := reg.List(ctx)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "develop", repos[0].Branch, "branch should be updated by upsert")
}

func TestRepoRegistryRegisterPreservesRemoteAndBranchOnEmptyUpdate(t *testing.T) {
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)
	repoDir := makeHarnessDir(t)
	ctx := context.Background()
	id, err := reg.Register(ctx, repoDir, "Owner/Repo", "release/Main")
	require.NoError(t, err)
	id2, err := reg.Register(ctx, repoDir, "", "")
	require.NoError(t, err)
	require.Equal(t, id, id2)
	record, err := reg.FindByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "owner/repo", record.Remote)
	require.Equal(t, "release/Main", record.Branch)
	has, err := reg.HasWebhookRepositories(ctx)
	require.NoError(t, err)
	require.True(t, has)
}

func TestRepoRegistryEmptyRemoteNeverEnablesWebhookPolicy(t *testing.T) {
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)
	_, err = reg.Register(context.Background(), makeHarnessDir(t), "", "main")
	require.NoError(t, err)
	has, err := reg.HasWebhookRepositories(context.Background())
	require.NoError(t, err)
	require.False(t, has)
	found, err := reg.FindByRemote(context.Background(), "")
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestNormalizeRemote(t *testing.T) {
	t.Parallel()
	got, err := NormalizeRemote("Owner/Repo")
	require.NoError(t, err)
	require.Equal(t, "owner/repo", got)
	for _, remote := range []string{"owner", " owner/repo", "owner/repo ", "owner/repo/extra", "owner/re po", "owner\\repo", "owner/repo?x", "owner/repo#x", "owner/repo\n"} {
		_, err := NormalizeRemote(remote)
		require.Error(t, err)
		require.Contains(t, err.Error(), "owner/repo")
	}
}

func TestRepoRegistryRejectsUnsafeBranch(t *testing.T) {
	t.Parallel()
	reg, err := NewRepoRegistry(openTestDB(t))
	require.NoError(t, err)
	for _, branch := range []string{" main", "main ", "main branch", "main\\other", "main?x", "main#x", "main\nother"} {
		_, err := reg.Register(context.Background(), makeHarnessDir(t), "owner/repo", branch)
		require.Error(t, err, branch)
		require.Contains(t, err.Error(), "branch")
	}
}

func TestRepoRegistry_Register_missingManifest(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	_, err = reg.Register(context.Background(), t.TempDir(), "owner/repo", "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest.yaml")
}

func TestRepoRegistry_List_empty(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	repos, err := reg.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestRepoRegistry_FindByRemote_found(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	repoDir := makeHarnessDir(t)
	ctx := context.Background()

	_, err = reg.Register(ctx, repoDir, "owner/cool-repo", "main")
	require.NoError(t, err)

	found, err := reg.FindByRemote(ctx, "OWNER/COOL-REPO")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "owner/cool-repo", found.Remote)
	assert.Equal(t, repoDir, found.Path)
}

func TestRepoRegistry_FindByRemote_notFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	found, err := reg.FindByRemote(context.Background(), "owner/nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepoRegistry_Remove(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	repoDir := makeHarnessDir(t)
	ctx := context.Background()

	id, err := reg.Register(ctx, repoDir, "owner/repo", "main")
	require.NoError(t, err)

	err = reg.Remove(ctx, id)
	require.NoError(t, err)

	repos, err := reg.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestRepoRegistry_Remove_notFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	reg, err := NewRepoRegistry(db)
	require.NoError(t, err)

	err = reg.Remove(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestNewRepoRegistry_nilDB(t *testing.T) {
	t.Parallel()
	_, err := NewRepoRegistry(nil)
	assert.Error(t, err)
}
