/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

func completeRelease(tag string) selfupdate.ReleaseInfo {
	assets := make([]selfupdate.ReleaseAsset, 0, 5)
	for _, name := range selfupdate.ExpectedReleaseAssetNames() {
		assets = append(assets, selfupdate.ReleaseAsset{Name: name})
	}
	return selfupdate.ReleaseInfo{TagName: tag, Name: tag, Assets: assets}
}

func TestAuditReportsCleanMirror(t *testing.T) {
	result, err := Audit(context.Background(), AuditConfig{
		ListTags: func(context.Context) ([]string, error) {
			return []string{"v0.1.0", "v0.2.0"}, nil
		},
		ListReleases: func(context.Context) ([]selfupdate.ReleaseInfo, error) {
			return []selfupdate.ReleaseInfo{completeRelease("v0.1.0"), completeRelease("v0.2.0")}, nil
		},
	})
	require.NoError(t, err)
	assert.False(t, result.Skipped)
	assert.Equal(t, []string{"v0.2.0", "v0.1.0"}, result.Checked)
	assert.Empty(t, result.Findings)
}

func TestAuditDetectsNotesOnlyRelease(t *testing.T) {
	notesOnly := selfupdate.ReleaseInfo{TagName: "v0.2.0", Name: "v0.2.0"}
	result, err := Audit(context.Background(), AuditConfig{
		ListTags: func(context.Context) ([]string, error) {
			return []string{"v0.1.0", "v0.2.0"}, nil
		},
		ListReleases: func(context.Context) ([]selfupdate.ReleaseInfo, error) {
			return []selfupdate.ReleaseInfo{completeRelease("v0.1.0"), notesOnly}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	finding := result.Findings[0]
	assert.Equal(t, "v0.2.0", finding.TagName)
	assert.Equal(t, AuditNotesOnly, finding.Class)
	assert.Contains(t, finding.Missing, "checksums.txt")
	assert.Contains(t, finding.Remediation, "no built-in producer is available for v0.2.0")
	assert.Contains(t, finding.Remediation, "repository's approved release workflow")
}

func TestAuditDetectsMissingReleaseObject(t *testing.T) {
	result, err := Audit(context.Background(), AuditConfig{
		ListTags: func(context.Context) ([]string, error) {
			return []string{"v0.3.0"}, nil
		},
		ListReleases: func(context.Context) ([]selfupdate.ReleaseInfo, error) {
			return nil, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, AuditMissingRelease, result.Findings[0].Class)
	assert.Contains(t, result.Findings[0].Remediation, "no built-in producer is available for v0.3.0")
	assert.Contains(t, result.Findings[0].Remediation, "repository's approved release workflow")
}

func TestAuditHonorsLimitNewestFirst(t *testing.T) {
	result, err := Audit(context.Background(), AuditConfig{
		Limit: 2,
		ListTags: func(context.Context) ([]string, error) {
			return []string{"v0.9.0", "v0.10.1", "v0.10.0", "not-a-version", "v0.2.9"}, nil
		},
		ListReleases: func(context.Context) ([]selfupdate.ReleaseInfo, error) {
			return []selfupdate.ReleaseInfo{completeRelease("v0.10.1"), completeRelease("v0.10.0")}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.10.1", "v0.10.0"}, result.Checked)
	assert.Empty(t, result.Findings)
}

func TestAuditSkipsWhenReleasesUnavailable(t *testing.T) {
	result, err := Audit(context.Background(), AuditConfig{
		ListTags: func(context.Context) ([]string, error) {
			return []string{"v0.1.0"}, nil
		},
		ListReleases: func(context.Context) ([]selfupdate.ReleaseInfo, error) {
			return nil, errors.New("api unavailable")
		},
	})
	require.NoError(t, err)
	assert.True(t, result.Skipped)
	assert.Contains(t, result.SkipReason, "api unavailable")
	assert.Empty(t, result.Findings)
}

func TestAuditSkipsWhenNoVersionTags(t *testing.T) {
	result, err := Audit(context.Background(), AuditConfig{
		ListTags: func(context.Context) ([]string, error) {
			return []string{"random-tag"}, nil
		},
		ListReleases: func(context.Context) ([]selfupdate.ReleaseInfo, error) {
			t.Fatal("releases must not be fetched without version tags")
			return nil, nil
		},
	})
	require.NoError(t, err)
	assert.True(t, result.Skipped)
	assert.Contains(t, result.SkipReason, "no vX.Y.Z tags")
}
