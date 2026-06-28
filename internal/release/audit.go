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
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

// AuditConfig controls release mirror auditing.
type AuditConfig struct {
	RepoRoot     string
	RepoFullName string
	Limit        int
	Client       *http.Client
	// ListTags and ListReleases are injectable for tests. Defaults read local
	// git tags and the GitHub releases API.
	ListTags       func(ctx context.Context) ([]string, error)
	ListReleases   func(ctx context.Context) ([]selfupdate.ReleaseInfo, error)
	CommandContext func(context.Context, string, ...string) *exec.Cmd
}

// AuditFindingClass classifies one release mirror defect.
type AuditFindingClass string

const (
	// AuditMissingRelease means a published version tag has no GitHub
	// Release object at all.
	AuditMissingRelease AuditFindingClass = "missing_release"
	// AuditNotesOnly means a GitHub Release object exists but required
	// binary assets or checksums are missing.
	AuditNotesOnly AuditFindingClass = "notes_only"
)

// AuditFinding is one version whose release mirror is incomplete.
type AuditFinding struct {
	TagName     string            `json:"tag_name"`
	Class       AuditFindingClass `json:"class"`
	Missing     []string          `json:"missing,omitempty"`
	Remediation string            `json:"remediation"`
}

// AuditResult reports the audited tags and any incomplete release mirrors.
type AuditResult struct {
	RepoFullName string         `json:"repo_full_name"`
	Checked      []string       `json:"checked"`
	Findings     []AuditFinding `json:"findings"`
	Skipped      bool           `json:"skipped"`
	SkipReason   string         `json:"skip_reason,omitempty"`
}

// Audit detects notes-only and missing GitHub releases for the newest local
// version tags so release self-verification does not stop at the most recent
// publication (T-026, AD-282). The GitHub mirror is optional infrastructure:
// when tags or the releases API are unavailable the audit reports a skip with
// the blocker instead of failing the pipeline.
func Audit(ctx context.Context, cfg AuditConfig) (AuditResult, error) {
	repoFullName := strings.TrimSpace(cfg.RepoFullName)
	if repoFullName == "" {
		repoFullName = selfupdate.DefaultRepoFullName
	}
	limit := cfg.Limit
	if limit <= 0 {
		limit = 10
	}
	result := AuditResult{RepoFullName: repoFullName}

	listTags := cfg.ListTags
	if listTags == nil {
		listTags = func(ctx context.Context) ([]string, error) {
			return gitVersionTags(ctx, cfg.CommandContext, cfg.RepoRoot)
		}
	}
	tags, err := listTags(ctx)
	if err != nil {
		result.Skipped = true
		result.SkipReason = fmt.Sprintf("cannot list local version tags: %v", err)
		return result, nil
	}
	tags = newestVersionTags(tags, limit)
	if len(tags) == 0 {
		result.Skipped = true
		result.SkipReason = "no vX.Y.Z tags found in the repository"
		return result, nil
	}

	listReleases := cfg.ListReleases
	if listReleases == nil {
		listReleases = func(ctx context.Context) ([]selfupdate.ReleaseInfo, error) {
			return fetchGitHubReleases(ctx, cfg.Client, repoFullName)
		}
	}
	releases, err := listReleases(ctx)
	if err != nil {
		result.Skipped = true
		result.SkipReason = fmt.Sprintf("cannot list GitHub releases for %s: %v", repoFullName, err)
		return result, nil
	}
	byTag := make(map[string]selfupdate.ReleaseInfo, len(releases))
	for _, rel := range releases {
		if rel.TagName != "" {
			byTag[rel.TagName] = rel
		}
	}

	for _, tag := range tags {
		result.Checked = append(result.Checked, tag)
		rel, ok := byTag[tag]
		if !ok {
			result.Findings = append(result.Findings, AuditFinding{
				TagName: tag,
				Class:   AuditMissingRelease,
				Remediation: fmt.Sprintf(
					"checkout the %s release-note commit, then run: mars release publish-assets --repo . --version %s --upload github",
					tag, tag),
			})
			continue
		}
		report := selfupdate.VerifyReleaseAssetInfo(rel)
		if !report.OK {
			result.Findings = append(result.Findings, AuditFinding{
				TagName: tag,
				Class:   AuditNotesOnly,
				Missing: report.Missing,
				Remediation: fmt.Sprintf(
					"checkout the %s release-note commit, then run: mars release publish-assets --repo . --version %s --upload github",
					tag, tag),
			})
		}
	}
	return result, nil
}

func gitVersionTags(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot string) ([]string, error) {
	if runner == nil {
		runner = exec.CommandContext
	}
	out, err := commandOutputErr(ctx, runner, repoRoot, "git", "tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// newestVersionTags keeps well-formed vX.Y.Z tags sorted newest-first, bounded
// by limit.
func newestVersionTags(tags []string, limit int) []string {
	type parsed struct {
		tag   string
		parts [3]int
	}
	var versions []parsed
	for _, tag := range tags {
		parts, ok := parseTagSemver(tag)
		if !ok {
			continue
		}
		versions = append(versions, parsed{tag: tag, parts: parts})
	}
	sort.Slice(versions, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if versions[i].parts[k] != versions[j].parts[k] {
				return versions[i].parts[k] > versions[j].parts[k]
			}
		}
		return versions[i].tag > versions[j].tag
	})
	if len(versions) > limit {
		versions = versions[:limit]
	}
	out := make([]string, 0, len(versions))
	for _, v := range versions {
		out = append(out, v.tag)
	}
	return out
}

func parseTagSemver(tag string) ([3]int, bool) {
	var parts [3]int
	v := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	raw := strings.Split(v, ".")
	if len(raw) != 3 {
		return parts, false
	}
	for i, part := range raw {
		n, err := strconv.Atoi(part)
		if err != nil || part == "" {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}

func fetchGitHubReleases(ctx context.Context, client *http.Client, repoFullName string) ([]selfupdate.ReleaseInfo, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	url := "https://api.github.com/repos/" + repoFullName + "/releases?per_page=100"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("release audit: build request: %w", err)
	}
	selfupdate.SetGitHubAPIHeaders(req, "mars-release-audit")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("release audit: request %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("release audit: %s returned %s", url, resp.Status)
	}
	var releases []selfupdate.ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("release audit: decode response: %w", err)
	}
	return releases, nil
}
