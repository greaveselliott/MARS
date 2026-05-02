package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DefaultLatestReleaseURL = "https://api.github.com/repos/greaveselliott/mars-harness/releases/latest"

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
	if strings.TrimSpace(url) == "" {
		url = DefaultLatestReleaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("latest release: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mars-harness-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("latest release: request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("latest release: %s returned %s", url, resp.Status)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("latest release: decode response: %w", err)
	}
	version := NormalizeVersion(payload.TagName)
	if version == "" {
		version = NormalizeVersion(payload.Name)
	}
	if version == "" {
		return "", fmt.Errorf("latest release: response missing tag_name")
	}
	return version, nil
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
