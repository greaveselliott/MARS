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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/githubauth"
)

const (
	DefaultRepoFullName     = "greaveselliott/MARS"
	DefaultLatestReleaseURL = "https://api.github.com/repos/" + DefaultRepoFullName + "/releases/latest"
	maxLatestReleaseBytes   = 256 << 10
)

// ReleaseInfo is the subset of GitHub release metadata used by update checks.
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
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
func LatestReleaseInfo(ctx context.Context, client *http.Client, endpoint string) (ReleaseInfo, error) {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultLatestReleaseURL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" {
		return ReleaseInfo{}, fmt.Errorf("latest release: endpoint must be HTTPS with a host and without user information or fragments")
	}
	if client == nil {
		client = &http.Client{}
	}
	clonedClient := *client
	if clonedClient.Timeout <= 0 {
		clonedClient.Timeout = 5 * time.Second
	}
	clonedClient.Jar = nil
	clonedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &clonedClient
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: cannot build the validated metadata request")
	}
	setGitHubHeaders(req, "mars-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: metadata request failed; check access to the configured HTTPS endpoint")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReleaseInfo{}, fmt.Errorf("latest release: endpoint returned HTTP %d%s", resp.StatusCode, githubAuthHint(parsed.String(), resp.StatusCode))
	}
	if resp.ContentLength > maxLatestReleaseBytes {
		return ReleaseInfo{}, fmt.Errorf("latest release: metadata response exceeds %d bytes", maxLatestReleaseBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxLatestReleaseBytes+1))
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: cannot read metadata response")
	}
	if len(raw) == 0 {
		return ReleaseInfo{}, fmt.Errorf("latest release: metadata response is empty")
	}
	if len(raw) > maxLatestReleaseBytes {
		return ReleaseInfo{}, fmt.Errorf("latest release: metadata response exceeds %d bytes", maxLatestReleaseBytes)
	}
	var payload ReleaseInfo
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ReleaseInfo{}, fmt.Errorf("latest release: metadata response is not valid JSON")
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
	if exactGitHubAPIURL(req.URL.String()) {
		setGitHubAuth(req)
	}
}

func setGitHubAuth(req *http.Request) {
	githubauth.Apply(req, githubauth.Options{})
}

func githubAuthHint(requestURL string, statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		if exactGitHubAPIURL(requestURL) {
			return "\nGitHub release metadata access was denied. Run `mars auth github check`; signed updates resolve optional GitHub auth without printing token values."
		}
		return "\ncustom release metadata endpoints are anonymous-only; use the official MARS endpoint or omit --latest-release-url."
	default:
		return ""
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
