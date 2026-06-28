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
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := OpenStore(filepath.Join(dir, "trust.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenStore_missingParentIsActionable(t *testing.T) {
	_, err := OpenStore(filepath.Join(t.TempDir(), "missing", "trust.db"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database directory")
	assert.Contains(t, err.Error(), "mars register --repo")
}

func TestOpenStoreLegacyFixture(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "legacy-trust.db")
	testutil.WriteSQLiteFixture(t, path, `
CREATE TABLE trust_entries (
  role       TEXT NOT NULL,
  repo_id    TEXT NOT NULL,
  level      TEXT NOT NULL DEFAULT 'observer',
  trial_runs INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (role, repo_id)
);
`, `
CREATE TABLE trust_events (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  role       TEXT NOT NULL,
  repo_id    TEXT NOT NULL,
  level      TEXT NOT NULL,
  reason     TEXT NOT NULL,
  recorded_at INTEGER NOT NULL
);
`, `
INSERT INTO trust_entries(role, repo_id, level, trial_runs, updated_at)
VALUES('engineer', 'repo-1', 'observer', 2, 1);
`, `
INSERT INTO trust_events(role, repo_id, level, reason, recorded_at)
VALUES('engineer', 'repo-1', 'observer', 'legacy fixture', 1);
`)

	s, err := OpenStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	testutil.AssertSQLiteColumns(t, s.db, "trust_entries", "role", "repo_id", "level", "trial_runs", "updated_at")

	got, err := s.Get(context.Background(), "engineer", "repo-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, LevelObserver, got.Level)
	require.Equal(t, 2, got.TrialRuns)
}

func TestEvaluate_observerToContributor(t *testing.T) {
	entry := &Entry{
		Role:      "ci-fix",
		RepoID:    "repo-1",
		Level:     LevelObserver,
		TrialRuns: 5,
	}
	assert.Equal(t, LevelContributor, Evaluate(entry, 0.0, 0))
}

func TestEvaluate_contributorToAutonomous(t *testing.T) {
	entry := &Entry{
		Role:      "ci-fix",
		RepoID:    "repo-1",
		Level:     LevelContributor,
		TrialRuns: 25,
	}
	assert.Equal(t, LevelAutonomous, Evaluate(entry, 0.85, 25))
}

func TestEvaluate_demotionOnScoreDrop(t *testing.T) {
	t.Run("autonomous to contributor", func(t *testing.T) {
		entry := &Entry{Level: LevelAutonomous, TrialRuns: 30}
		assert.Equal(t, LevelContributor, Evaluate(entry, 0.55, 30))
	})

	t.Run("contributor to observer", func(t *testing.T) {
		entry := &Entry{Level: LevelContributor, TrialRuns: 10}
		assert.Equal(t, LevelObserver, Evaluate(entry, 0.2, 10))
	})
}

func TestEvaluate_staysAtLevel(t *testing.T) {
	t.Run("observer stays below trial threshold", func(t *testing.T) {
		entry := &Entry{Level: LevelObserver, TrialRuns: 3}
		assert.Equal(t, LevelObserver, Evaluate(entry, 0.0, 0))
	})

	t.Run("contributor stays with moderate score", func(t *testing.T) {
		entry := &Entry{Level: LevelContributor, TrialRuns: 10}
		assert.Equal(t, LevelContributor, Evaluate(entry, 0.7, 15))
	})

	t.Run("contributor stays with high score but insufficient outcomes", func(t *testing.T) {
		entry := &Entry{Level: LevelContributor, TrialRuns: 10}
		assert.Equal(t, LevelContributor, Evaluate(entry, 0.9, 10))
	})

	t.Run("autonomous stays with good score", func(t *testing.T) {
		entry := &Entry{Level: LevelAutonomous, TrialRuns: 50}
		assert.Equal(t, LevelAutonomous, Evaluate(entry, 0.85, 50))
	})

	t.Run("nil entry returns observer", func(t *testing.T) {
		assert.Equal(t, LevelObserver, Evaluate(nil, 0.9, 100))
	})
}

func TestCapabilities_observer(t *testing.T) {
	caps := Capabilities(LevelObserver)
	assert.Contains(t, caps, "repo_read")
	assert.Contains(t, caps, "file_read")
	assert.NotContains(t, caps, "repo_write")
	assert.NotContains(t, caps, "git_push_main")
}

func TestCapabilities_contributor(t *testing.T) {
	caps := Capabilities(LevelContributor)
	assert.Contains(t, caps, "file_write")
	assert.Contains(t, caps, "git_commit")
	assert.Contains(t, caps, "git_push_main")
	assert.NotContains(t, caps, "git_branch")
	assert.NotContains(t, caps, "create_pr")
	assert.NotContains(t, caps, "merge")
}

func TestCapabilities_autonomous(t *testing.T) {
	caps := Capabilities(LevelAutonomous)
	assert.Contains(t, caps, "file_write")
	assert.Contains(t, caps, "self_schedule")
	assert.Contains(t, caps, "evolve_policy")
	assert.NotContains(t, caps, "create_pr")
	assert.NotContains(t, caps, "merge")
}

func TestStore_roundTrip(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	got, err := s.Get(ctx, "ci-fix", "repo-1")
	require.NoError(t, err)
	assert.Nil(t, got, "non-existent entry should return nil")

	require.NoError(t, s.Set(ctx, "ci-fix", "repo-1", LevelContributor))

	got, err = s.Get(ctx, "ci-fix", "repo-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, LevelContributor, got.Level)
	assert.Equal(t, 0, got.TrialRuns, "trial runs reset on level change")
	assert.False(t, got.UpdatedAt.IsZero())

	require.NoError(t, s.Set(ctx, "ci-fix", "repo-1", LevelContributor))
	got, err = s.Get(ctx, "ci-fix", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 0, got.TrialRuns, "trial runs preserved when level unchanged")
}

func TestStore_incrementTrialRuns(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	require.NoError(t, s.IncrementTrialRuns(ctx, "deploy", "repo-1"))
	got, err := s.Get(ctx, "deploy", "repo-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, LevelObserver, got.Level, "auto-created entry should be observer")
	assert.Equal(t, 1, got.TrialRuns)

	for range 4 {
		require.NoError(t, s.IncrementTrialRuns(ctx, "deploy", "repo-1"))
	}
	got, err = s.Get(ctx, "deploy", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 5, got.TrialRuns)

	require.NoError(t, s.Set(ctx, "deploy", "repo-1", LevelContributor))
	got, err = s.Get(ctx, "deploy", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 0, got.TrialRuns, "trial runs reset on level change")

	require.NoError(t, s.IncrementTrialRuns(ctx, "deploy", "repo-1"))
	got, err = s.Get(ctx, "deploy", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 1, got.TrialRuns)
}
