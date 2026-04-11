package evolution

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "evo.sqlite")
	uri := "file:" + path + "?mode=rwc"
	s, err := OpenStore(uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// --- Detector tests ---

func TestDetect_humanCommitIsIntervention(t *testing.T) {
	t.Parallel()
	commits := []CommitInfo{
		{SHA: "aaa", Author: "mars-harness[bot]", FilesChanged: []string{"main.go"}, HasDiff: true},
		{SHA: "bbb", Author: "human-dev", FilesChanged: []string{"fix.go"}, HasDiff: true},
	}
	typ, ev := Detect("mars-harness[bot]", commits)
	require.Equal(t, TypeClear, typ)

	var parsed evidence
	require.NoError(t, json.Unmarshal([]byte(ev), &parsed))
	require.Contains(t, parsed.SHAs, "bbb")
	require.Contains(t, parsed.FilesChanged, "fix.go")
	require.NotContains(t, parsed.SHAs, "aaa")
}

func TestDetect_commentOnlyIsNonIntervention(t *testing.T) {
	t.Parallel()
	commits := []CommitInfo{
		{SHA: "aaa", Author: "mars-harness[bot]", FilesChanged: []string{"main.go"}, HasDiff: true},
		{SHA: "bbb", Author: "human-dev", FilesChanged: nil, HasDiff: false},
	}
	typ, _ := Detect("mars-harness[bot]", commits)
	require.Equal(t, TypeNonIntervention, typ)
}

func TestDetect_squashMergeDetected(t *testing.T) {
	t.Parallel()
	commits := []CommitInfo{
		{SHA: "squash-abc", Author: "human-dev", FilesChanged: []string{"a.go", "b.go", "c.go"}, HasDiff: true},
	}
	typ, ev := Detect("mars-harness[bot]", commits)
	require.Equal(t, TypeClear, typ)

	var parsed evidence
	require.NoError(t, json.Unmarshal([]byte(ev), &parsed))
	require.Equal(t, []string{"squash-abc"}, parsed.SHAs)
	require.Len(t, parsed.FilesChanged, 3)
}

// --- Reviewer tests ---

func TestCanReview_rateLimitBlocks(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.SaveEvolution(ctx, Evolution{
		ID:          "ev-1",
		Role:        "ci-fixer",
		RepoID:      "repo-1",
		Result:      "{}",
		ScoreBefore: 0.5,
		ScoreAfter:  0.6,
		CreatedAt:   time.Now().UTC(),
	}))

	cfg := ReviewerConfig{MaxRunsPerDay: 1, AutoDisableAfter: 3}
	ok, reason := CanReview(store, "ci-fixer", cfg)
	require.False(t, ok)
	require.Contains(t, reason, "rate limit")
}

func TestCanReview_autoDisableAfterWorsening(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		require.NoError(t, store.SaveEvolution(ctx, Evolution{
			ID:          newUUID(),
			Role:        "doc-writer",
			RepoID:      "repo-1",
			Result:      "{}",
			ScoreBefore: 0.8,
			ScoreAfter:  0.7,
			CreatedAt:   now.Add(-time.Duration(48+i) * time.Hour),
		}))
	}

	cfg := ReviewerConfig{MaxRunsPerDay: 10, AutoDisableAfter: 3}
	ok, reason := CanReview(store, "doc-writer", cfg)
	require.False(t, ok)
	require.Contains(t, reason, "auto-disabled")
}

// --- Store tests ---

func TestStore_interventionRoundTrip(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()

	iv := Intervention{
		ID:         "iv-1",
		JobID:      "job-10",
		RepoID:     "repo-1",
		Role:       "ci-fixer",
		Type:       TypeClear,
		Evidence:   `{"shas":["abc"]}`,
		DetectedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, store.SaveIntervention(ctx, iv))

	got, err := store.GetInterventions(ctx, "repo-1", "ci-fixer", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, iv.ID, got[0].ID)
	require.Equal(t, iv.JobID, got[0].JobID)
	require.Equal(t, iv.Type, got[0].Type)
	require.Equal(t, iv.Evidence, got[0].Evidence)
	require.Equal(t, iv.DetectedAt, got[0].DetectedAt)
}

func TestStore_evolutionRoundTrip(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()

	ev := Evolution{
		ID:          "ev-rt-1",
		Role:        "pr-writer",
		RepoID:      "repo-2",
		Result:      `{"classification":"missing_test"}`,
		ScoreBefore: 0.6,
		ScoreAfter:  0.75,
		CreatedAt:   time.Date(2026, 4, 11, 8, 0, 0, 0, time.UTC),
	}
	require.NoError(t, store.SaveEvolution(ctx, ev))

	got, err := store.GetEvolutions(ctx, "pr-writer", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, ev.ID, got[0].ID)
	require.Equal(t, ev.Role, got[0].Role)
	require.Equal(t, ev.RepoID, got[0].RepoID)
	require.Equal(t, ev.Result, got[0].Result)
	require.InDelta(t, ev.ScoreBefore, got[0].ScoreBefore, 0.001)
	require.InDelta(t, ev.ScoreAfter, got[0].ScoreAfter, 0.001)
	require.Equal(t, ev.CreatedAt, got[0].CreatedAt)
}

func TestStore_countRecentEvolutions(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()

	require.NoError(t, store.SaveEvolution(ctx, Evolution{
		ID: "ev-recent", Role: "ci-fixer", RepoID: "r1", Result: "{}",
		CreatedAt: now.Add(-1 * time.Hour),
	}))
	require.NoError(t, store.SaveEvolution(ctx, Evolution{
		ID: "ev-old", Role: "ci-fixer", RepoID: "r1", Result: "{}",
		CreatedAt: now.Add(-48 * time.Hour),
	}))
	require.NoError(t, store.SaveEvolution(ctx, Evolution{
		ID: "ev-other-role", Role: "doc-writer", RepoID: "r1", Result: "{}",
		CreatedAt: now.Add(-1 * time.Hour),
	}))

	count, err := store.CountRecentEvolutions(ctx, "ci-fixer", now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
