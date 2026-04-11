package trust

import (
	"context"
	"path/filepath"
	"testing"

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
	assert.ElementsMatch(t, []string{"file_read", "grep", "shell_exec_readonly"}, caps)
}

func TestCapabilities_contributor(t *testing.T) {
	caps := Capabilities(LevelContributor)
	assert.Contains(t, caps, "file_write")
	assert.Contains(t, caps, "git_commit")
	assert.Contains(t, caps, "git_branch")
	assert.NotContains(t, caps, "create_pr")
	assert.NotContains(t, caps, "merge")
}

func TestCapabilities_autonomous(t *testing.T) {
	caps := Capabilities(LevelAutonomous)
	assert.Contains(t, caps, "create_pr")
	assert.Contains(t, caps, "merge")
	assert.Contains(t, caps, "file_write")
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
