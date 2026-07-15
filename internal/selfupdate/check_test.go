/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-017-open-source-publication.md
*/
package selfupdate

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	require.Equal(t, VersionEqual, CompareVersions("0.6.0", "v0.6.0"))
	require.Equal(t, VersionBehind, CompareVersions("0.6.0", "0.7.0"))
	require.Equal(t, VersionAhead, CompareVersions("0.8.0", "0.7.0"))
	require.Equal(t, VersionUnknown, CompareVersions("dev", "0.7.0"))
}

func TestLatestRelease_readsGitHubStyleTag(t *testing.T) {
	t.Parallel()
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/releases/latest", r.URL.Path)
		return textResponse(http.StatusOK, `{"tag_name":"v0.7.0"}`), nil
	})

	version, err := LatestRelease(context.Background(), client, "https://example.test/releases/latest")
	require.NoError(t, err)
	require.Equal(t, "0.7.0", version)
}

func TestDefaultRepoFullNameUsesCanonicalMARSRepo(t *testing.T) {
	t.Parallel()
	require.Equal(t, "greaveselliott/MARS", DefaultRepoFullName)
	require.Contains(t, DefaultLatestReleaseURL, "/repos/greaveselliott/MARS/")
}

func TestLatestReleaseInfoReportsPrivateReleaseAuthHint(t *testing.T) {
	t.Parallel()
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		return textResponse(http.StatusUnauthorized, `{"message":"bad credentials"}`), nil
	})

	_, err := LatestReleaseInfo(context.Background(), client, "https://example.test/releases/latest")
	require.ErrorContains(t, err, "GH_TOKEN")
	require.ErrorContains(t, err, "private releases")
}

func TestVerifyReleaseAssetsReportsMissingAssets(t *testing.T) {
	t.Parallel()
	report := VerifyReleaseAssetInfo(ReleaseInfo{
		TagName: "v1.2.3",
		HTMLURL: "https://example.test/release",
		Assets: []ReleaseAsset{
			{Name: "mars-linux-amd64"},
			{Name: "checksums.txt"},
		},
	})

	require.False(t, report.OK)
	require.Equal(t, "1.2.3", report.Version)
	require.Contains(t, report.Found, "mars-linux-amd64")
	require.Contains(t, report.Found, "checksums.txt")
	require.ElementsMatch(t, []string{
		"mars-linux-arm64",
		"mars-darwin-amd64",
		"mars-darwin-arm64",
		"mars-harness-linux-amd64",
		"mars-harness-linux-arm64",
		"mars-harness-darwin-amd64",
		"mars-harness-darwin-arm64",
	}, report.Missing)
}

func TestVerifyReleaseAssetInfoRejectsExtraAndDuplicateNames(t *testing.T) {
	t.Parallel()
	assets := make([]ReleaseAsset, 0, len(ExpectedReleaseAssetNames())+2)
	for _, name := range ExpectedReleaseAssetNames() {
		assets = append(assets, ReleaseAsset{Name: name})
	}
	assets = append(assets, ReleaseAsset{Name: ExpectedReleaseAssetNames()[0]}, ReleaseAsset{Name: "unexpected.bin"})
	report := VerifyReleaseAssetInfo(ReleaseInfo{TagName: "v1.2.3", Assets: assets})
	require.False(t, report.OK)
	require.Equal(t, []string{"<redacted-asset-name>"}, report.Extra)
	require.Equal(t, []string{ExpectedReleaseAssetNames()[0] + " (2 copies)"}, report.Duplicate)
}

func TestVerifyReleaseAssetInfoNeverTranscribesUnknownAssetNames(t *testing.T) {
	t.Parallel()
	const opaqueCredential = "AKIAIOSFODNN7EXAMPLE"
	report := VerifyReleaseAssetInfo(ReleaseInfo{
		TagName: "v1.2.3",
		Assets: []ReleaseAsset{
			{Name: opaqueCredential},
			{Name: opaqueCredential},
		},
	})
	diagnostics := strings.Join(append(append([]string{}, report.Extra...), report.Duplicate...), " ")
	require.NotContains(t, diagnostics, opaqueCredential)
	require.Contains(t, diagnostics, "<redacted-asset-name>")
}

func TestVerifyReleaseAssetInfoRedactsHostileExtraAndDuplicateNames(t *testing.T) {
	t.Parallel()
	assets := make([]ReleaseAsset, 0, len(ExpectedReleaseAssetNames())+2)
	for _, name := range ExpectedReleaseAssetNames() {
		assets = append(assets, ReleaseAsset{Name: name})
	}
	const hostile = "ghp_SECRET\nforged"
	assets = append(assets, ReleaseAsset{Name: hostile}, ReleaseAsset{Name: hostile})
	report := VerifyReleaseAssetInfo(ReleaseInfo{TagName: "v1.2.3", Assets: assets})
	require.False(t, report.OK)
	diagnostics := strings.Join(append(append([]string{}, report.Extra...), report.Duplicate...), " ")
	require.Contains(t, diagnostics, "<redacted-asset-name>")
	require.NotContains(t, diagnostics, "ghp_SECRET")
	require.NotContains(t, diagnostics, "\n")
}

func TestReleaseAPIURLBuildsLatestAndTaggedURLs(t *testing.T) {
	t.Parallel()
	require.Equal(t,
		"https://api.github.com/repos/example/project/releases/latest",
		ReleaseAPIURL("example/project", "latest"))
	require.Equal(t,
		"https://api.github.com/repos/example/project/releases/tags/v1.2.3",
		ReleaseAPIURL("example/project", "1.2.3"))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func fakeHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
