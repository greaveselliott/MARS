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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := OpenStore(filepath.Join(dir, "scoring.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenStore_missingParentIsActionable(t *testing.T) {
	_, err := OpenStore(filepath.Join(t.TempDir(), "missing", "scoring.db"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database directory")
	assert.Contains(t, err.Error(), "mars-harness register --repo")
}

func TestOpenStoreLegacyFixture(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "legacy-scoring.db")
	testutil.WriteSQLiteFixture(t, path, `
CREATE TABLE outcomes (
  id          TEXT PRIMARY KEY,
  job_id      TEXT NOT NULL,
  repo_id     TEXT NOT NULL,
  role        TEXT NOT NULL,
  type        TEXT NOT NULL,
  details     TEXT NOT NULL DEFAULT '',
  recorded_at INTEGER NOT NULL
);
`, `
CREATE TABLE scores (
  role        TEXT NOT NULL,
  repo_id     TEXT NOT NULL,
  value       REAL NOT NULL,
  sample_size INTEGER NOT NULL,
  window_days INTEGER NOT NULL,
  formula     TEXT NOT NULL,
  computed_at INTEGER NOT NULL,
  PRIMARY KEY (role, repo_id)
);
`, `
INSERT INTO outcomes(id, job_id, repo_id, role, type, details, recorded_at)
VALUES('outcome-legacy', 'job-1', 'repo-1', 'engineer', 'passed', '{}', 1779148800);
`)

	s, err := OpenStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	testutil.AssertSQLiteIndexes(t, s.db, "idx_outcomes_role_repo_time")

	score, err := s.ComputeScore(context.Background(), "engineer", "repo-1", 30)
	require.NoError(t, err)
	require.Equal(t, 1, score.SampleSize)
	require.Equal(t, 1.0, score.Value)
}

func TestComputeScore_emptyReturnsZero(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	sc, err := s.ComputeScore(ctx, "ci-fix", "repo-1", 30)
	require.NoError(t, err)
	assert.Equal(t, 0.0, sc.Value)
	assert.Equal(t, 0, sc.SampleSize)
	assert.Equal(t, "v1", sc.Formula)
}

func TestComputeScore_allMergedReturnsOne(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	for i := range 10 {
		require.NoError(t, s.RecordOutcome(ctx, Outcome{
			JobID:      fmt.Sprintf("job-%d", i),
			RepoID:     "repo-1",
			Role:       "pr-gen",
			Type:       OutcomeMerged,
			RecordedAt: time.Now().UTC(),
		}))
	}

	sc, err := s.ComputeScore(ctx, "pr-gen", "repo-1", 30)
	require.NoError(t, err)
	assert.Equal(t, 1.0, sc.Value)
	assert.Equal(t, 10, sc.SampleSize)
}

func TestComputeScore_mixedOutcomes(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	outcomes := []OutcomeType{
		OutcomeMerged, OutcomeMerged, OutcomeMerged, OutcomeMerged, OutcomeMerged,
		OutcomePassed, OutcomePassed, OutcomePassed,
		OutcomeClosed, OutcomeClosed,
		OutcomeFailed, OutcomeFailed, OutcomeFailed,
		OutcomeNoop, OutcomeNoop,
		OutcomeTimeout, OutcomeTimeout, OutcomeTimeout, OutcomeTimeout, OutcomeTimeout,
	}

	for i, ot := range outcomes {
		require.NoError(t, s.RecordOutcome(ctx, Outcome{
			JobID:      fmt.Sprintf("job-%d", i),
			RepoID:     "repo-1",
			Role:       "ci-fix",
			Type:       ot,
			RecordedAt: time.Now().UTC(),
		}))
	}

	// positive = 5 merged + 3 passed = 8
	// denominator = all 20 outcomes; timeouts are trunk-native failures
	sc, err := s.ComputeScore(ctx, "ci-fix", "repo-1", 30)
	require.NoError(t, err)
	assert.InDelta(t, 8.0/20.0, sc.Value, 0.001)
	assert.Equal(t, 20, sc.SampleSize)
}

func TestComputeScore_timeoutCountsAsNegative(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "j-1", RepoID: "repo-1", Role: "deploy",
		Type: OutcomeMerged, RecordedAt: time.Now().UTC(),
	}))
	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "j-2", RepoID: "repo-1", Role: "deploy",
		Type: OutcomeTimeout, RecordedAt: time.Now().UTC(),
	}))

	sc, err := s.ComputeScore(ctx, "deploy", "repo-1", 30)
	require.NoError(t, err)
	assert.Equal(t, 0.5, sc.Value)
	assert.Equal(t, 2, sc.SampleSize)
}

func TestComputeScore_windowRespected(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	oldTime := time.Now().UTC().AddDate(0, 0, -60)
	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "old-1", RepoID: "repo-1", Role: "ci-fix",
		Type: OutcomeFailed, RecordedAt: oldTime,
	}))

	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "new-1", RepoID: "repo-1", Role: "ci-fix",
		Type: OutcomeMerged, RecordedAt: time.Now().UTC(),
	}))

	sc, err := s.ComputeScore(ctx, "ci-fix", "repo-1", 30)
	require.NoError(t, err)
	assert.Equal(t, 1.0, sc.Value, "old outcome outside window should be excluded")
	assert.Equal(t, 1, sc.SampleSize)
}

func TestRecordOutcome_roundTrip(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	o := Outcome{
		JobID:      "job-abc",
		RepoID:     "repo-1",
		Role:       "pr-gen",
		Type:       OutcomePassed,
		Details:    `{"pr":42}`,
		RecordedAt: time.Now().UTC(),
	}
	require.NoError(t, s.RecordOutcome(ctx, o))

	sc, err := s.ComputeScore(ctx, "pr-gen", "repo-1", 30)
	require.NoError(t, err)
	assert.Equal(t, 1.0, sc.Value)
	assert.Equal(t, 1, sc.SampleSize)

	cached, err := s.GetScore(ctx, "pr-gen", "repo-1")
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, sc.Value, cached.Value)
	assert.Equal(t, sc.SampleSize, cached.SampleSize)

	missing, err := s.GetScore(ctx, "unknown-role", "repo-1")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestComputeScore_minimumSampleSize(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()

	for i := range 3 {
		require.NoError(t, s.RecordOutcome(ctx, Outcome{
			JobID: fmt.Sprintf("job-%d", i), RepoID: "repo-1", Role: "ci-fix",
			Type: OutcomeMerged, RecordedAt: time.Now().UTC(),
		}))
	}

	sc, err := s.ComputeScore(ctx, "ci-fix", "repo-1", 30)
	require.NoError(t, err)
	assert.Equal(t, 1.0, sc.Value)
	assert.Equal(t, 3, sc.SampleSize, "SampleSize should reflect actual count even below minimum threshold")
	assert.Less(t, sc.SampleSize, 5, "below minimum threshold of 5 outcomes")
}

func TestRoleReposWithOutcomesAndOutcomeCounts(t *testing.T) {
	s := tempStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "j-1", RepoID: "repo-1", Role: "engineer",
		Type: OutcomePassed, Details: `{"remediation_attempts":[]}`, RecordedAt: now,
	}))
	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "j-2", RepoID: "repo-1", Role: "engineer",
		Type: OutcomeFailed, RecordedAt: now,
	}))
	require.NoError(t, s.RecordOutcome(ctx, Outcome{
		JobID: "j-3", RepoID: "repo-2", Role: "qa",
		Type: OutcomeNoop, RecordedAt: now,
	}))

	pairs, err := s.RoleReposWithOutcomes(ctx, "repo-1", now.Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, []RoleRepo{{Role: "engineer", RepoID: "repo-1"}}, pairs)

	counts, err := s.OutcomeCounts(ctx, "repo-1", now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, counts, 2)
	byType := map[OutcomeType]int{}
	for _, count := range counts {
		byType[count.Type] = count.Count
	}
	assert.Equal(t, 1, byType[OutcomePassed])
	assert.Equal(t, 1, byType[OutcomeFailed])

	outcomes, err := s.OutcomesSince(ctx, "repo-1", now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, outcomes, 2)
	hasDetails := false
	for _, outcome := range outcomes {
		assert.Equal(t, "repo-1", outcome.RepoID)
		if outcome.Details != "" {
			hasDetails = true
		}
	}
	assert.True(t, hasDetails)
}
