/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/github-app-integration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeWebhookActorIDs(t *testing.T) {
	t.Parallel()
	ids, err := NormalizeWebhookActorIDs([]int64{42, 84, 42})
	require.NoError(t, err)
	require.Equal(t, []int64{42, 84}, ids)
	overLimit := make([]int64, MaxWebhookActorIDs+1)
	for i := range overLimit {
		overLimit[i] = int64(i + 1)
	}
	for _, ids := range [][]int64{{42, 0}, {-1}, overLimit} {
		_, err := NormalizeWebhookActorIDs(ids)
		require.Error(t, err)
	}
	duplicates := make([]int64, MaxWebhookActorIDs+1)
	for i := range duplicates {
		duplicates[i] = 42
	}
	ids, err = NormalizeWebhookActorIDs(duplicates)
	require.NoError(t, err)
	require.Equal(t, []int64{42}, ids)
}

func TestNormalizeRepositoryRejectsUnsafeIdentifiers(t *testing.T) {
	t.Parallel()
	repo, err := NormalizeRepository("Owner.Repo/Name_1")
	require.NoError(t, err)
	require.Equal(t, "owner.repo/name_1", repo)
	for _, value := range []string{"", "owner", "../repo", "owner/..", " owner/repo", "owner/repo ", "owner/re po", "owner\\repo", "owner/repo?x", "owner/repo#x", "owner/repo\n", "owner/repo/extra", "own er/repo", strings.Repeat("a", MaxRepositoryName+1) + "/repo"} {
		_, err := NormalizeRepository(value)
		require.Error(t, err, value)
	}
}

func TestValidateBranchRejectsUnsafeIdentifiers(t *testing.T) {
	t.Parallel()
	for _, branch := range []string{"main", "release/Main", "feature/safe-name", "feature/name.LOCK"} {
		require.NoError(t, ValidateBranch(branch), branch)
	}
	for _, branch := range []string{"", ".main", "-main", "feature/.hidden", "feature/name.lock", "a/b.lock/c", " main", "main ", "main branch", "main\\other", "main?x", "main#x", "main\nother", "main..other", "main@{x", "/main", "main/", "main.lock", strings.Repeat("a", MaxBranchName+1)} {
		require.Error(t, ValidateBranch(branch), branch)
	}
}

func TestValidBoundedToken(t *testing.T) {
	t.Parallel()
	require.True(t, validBoundedToken("delivery-123", 32))
	for _, value := range []string{"", " leading", "has space", "line\nbreak", strings.Repeat("x", 33)} {
		require.False(t, validBoundedToken(value, 32))
	}
}
