/*
MarsDocSync:
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultRepoFullName     = "greaveselliott/mars-harness"
	DefaultLatestReleaseURL = "https://api.github.com/repos/" + DefaultRepoFullName + "/releases/latest"
)

// ReleaseAsset is the subset of GitHub release asset metadata the harness
// needs for update and release verification.
type ReleaseAsset struct {
	APIURL             string `json:"url"`
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// ReleaseInfo is the subset of GitHub release metadata used by update and
// release verification commands.
type ReleaseInfo struct {
	TagName string         `json:"tag_name"`
	Name    string         `json:"name"`
	HTMLURL string         `json:"html_url"`
	Assets  []ReleaseAsset `json:"assets"`
}

// ReleaseAssetReport records whether a release satisfies the binary asset
// contract expected by installers and self-updates.
type ReleaseAssetReport struct {
	TagName  string   `json:"tag_name"`
	Version  string   `json:"version"`
	URL      string   `json:"url"`
	Required []string `json:"required"`
	Found    []string `json:"found"`
	Missing  []string `json:"missing"`
	OK       bool     `json:"ok"`
}

// VersionRelation describes how two semantic versions compare.
type VersionRelation string

const (
	VersionEqual   VersionRelation = "equal"
	VersionBehind  VersionRelation = "behind"
	VersionAhead   VersionRelation = "ahead"
	VersionUnknown VersionRelation = "unknown"
)

// LatestRelease fetches the newest published release version from a GitHub
// compatible releases/latest endpoint.
func LatestRelease(ctx context.Context, client *http.Client, url string) (string, error) {
	release, err := LatestReleaseInfo(ctx, client, url)
	if err != nil {
		return "", err
	}
	version := NormalizeVersion(release.TagName)
	if version == "" {
		version = NormalizeVersion(release.Name)
	}
	if version == "" {
		return "", fmt.Errorf("latest release: response missing tag_name")
	}
	return version, nil
}

// LatestReleaseInfo fetches the newest published release metadata from a
// GitHub-compatible releases/latest endpoint.
func LatestReleaseInfo(ctx context.Context, client *http.Client, url string) (ReleaseInfo, error) {
	if strings.TrimSpace(url) == "" {
		url = DefaultLatestReleaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: build request: %w", err)
	}
	setGitHubHeaders(req, "mars-harness-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReleaseInfo{}, fmt.Errorf("latest release: %s returned %s%s", url, resp.Status, githubAuthHint(resp.StatusCode))
	}
	var payload ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: decode response: %w", err)
	}
	version := NormalizeVersion(payload.TagName)
	if version == "" {
		version = NormalizeVersion(payload.Name)
	}
	if version == "" {
		return ReleaseInfo{}, fmt.Errorf("latest release: response missing tag_name")
	}
	return payload, nil
}

func setGitHubHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	setGitHubAuth(req)
}

func setGitHubDownloadHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", userAgent)
	setGitHubAuth(req)
}

func setGitHubAuth(req *http.Request) {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return
	}
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func githubAuthHint(statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "\nAuthenticate private releases by exporting GH_TOKEN or GITHUB_TOKEN with repository contents read access. With GitHub CLI auth, run `GH_TOKEN=\"$(gh auth token)\" mars-harness update tool`."
	default:
		return ""
	}
}

// ReleaseAPIURL returns a GitHub-compatible release metadata endpoint for a
// repository and release version.
func ReleaseAPIURL(repoFullName, version string) string {
	repo := strings.TrimSpace(repoFullName)
	if repo == "" {
		repo = DefaultRepoFullName
	}
	v := strings.TrimSpace(version)
	if v == "" || v == DefaultVersion {
		return "https://api.github.com/repos/" + repo + "/releases/latest"
	}
	return "https://api.github.com/repos/" + repo + "/releases/tags/" + releaseTag(v)
}

// ExpectedReleaseAssetNames returns the complete release asset contract for a
// Mars Harness GitHub release.
func ExpectedReleaseAssetNames() []string {
	return []string{
		"mars-harness-linux-amd64",
		"mars-harness-linux-arm64",
		"mars-harness-darwin-amd64",
		"mars-harness-darwin-arm64",
		"checksums.txt",
	}
}

// VerifyReleaseAssets fetches release metadata and reports whether all expected
// binary assets and checksums.txt are present.
func VerifyReleaseAssets(ctx context.Context, client *http.Client, releaseURL string) (ReleaseAssetReport, error) {
	release, err := LatestReleaseInfo(ctx, client, releaseURL)
	if err != nil {
		return ReleaseAssetReport{}, err
	}
	return VerifyReleaseAssetInfo(release), nil
}

// VerifyReleaseAssetInfo reports whether release metadata satisfies the asset
// contract without making network calls.
func VerifyReleaseAssetInfo(release ReleaseInfo) ReleaseAssetReport {
	required := ExpectedReleaseAssetNames()
	foundSet := make(map[string]bool, len(release.Assets))
	for _, asset := range release.Assets {
		if asset.Name != "" {
			foundSet[asset.Name] = true
		}
	}
	found := make([]string, 0, len(foundSet))
	missing := make([]string, 0)
	for _, name := range required {
		if foundSet[name] {
			found = append(found, name)
			continue
		}
		missing = append(missing, name)
	}
	sort.Strings(found)
	return ReleaseAssetReport{
		TagName:  release.TagName,
		Version:  NormalizeVersion(release.TagName),
		URL:      release.HTMLURL,
		Required: required,
		Found:    found,
		Missing:  missing,
		OK:       len(missing) == 0,
	}
}

// CompareVersions compares dotted semantic versions. Unknown/dev values return
// VersionUnknown so callers can report drift without pretending certainty.
func CompareVersions(current, latest string) VersionRelation {
	currentParts, okCurrent := parseSemver(current)
	latestParts, okLatest := parseSemver(latest)
	if !okCurrent || !okLatest {
		if NormalizeVersion(current) == NormalizeVersion(latest) && NormalizeVersion(current) != "" {
			return VersionEqual
		}
		return VersionUnknown
	}
	for i := 0; i < len(currentParts); i++ {
		if currentParts[i] < latestParts[i] {
			return VersionBehind
		}
		if currentParts[i] > latestParts[i] {
			return VersionAhead
		}
	}
	return VersionEqual
}

// NormalizeVersion removes common release prefixes and suffixes for display and
// comparison.
func NormalizeVersion(version string) string {
	v := strings.TrimSpace(version)
	v = strings.TrimPrefix(v, "refs/tags/")
	v = strings.TrimPrefix(v, "version ")
	v = strings.TrimPrefix(v, "v")
	return strings.TrimSpace(v)
}

func parseSemver(version string) ([3]int, bool) {
	var parts [3]int
	v := NormalizeVersion(version)
	if v == "" {
		return parts, false
	}
	v = strings.SplitN(v, "-", 2)[0]
	raw := strings.Split(v, ".")
	if len(raw) != 3 {
		return parts, false
	}
	for i, part := range raw {
		n, err := strconv.Atoi(part)
		if err != nil {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}
