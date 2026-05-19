/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	changelogReleaseRE = regexp.MustCompile(`(?m)^## \[([0-9]+\.[0-9]+\.[0-9]+)\] - [^\n]+\n<!-- mars-harness-release: version=([^ ]+) commit=([0-9a-fA-F]+) -->`)
	changelogCommitRE  = regexp.MustCompile(`\(([0-9a-fA-F]{7,40})\)`)
	sectionHeadingRE   = regexp.MustCompile(`(?m)^### `)
)

type BackfillConfig struct {
	RepoRoot   string
	MinVersion string
	MaxVersion string
	DryRun     bool
	Check      bool
}

type BackfillResult struct {
	Entries      []BackfillEntryResult
	Changed      []SemVer
	UpdatedFiles []string
	Changelog    string
}

type BackfillEntryResult struct {
	Version     SemVer
	BaseRef     string
	HeadRef     string
	CommitCount int
	Changed     bool
}

type changelogEntry struct {
	Version SemVer
	Header  string
	Body    string
	Raw     string
	Commit  string
}

func BackfillNotes(ctx context.Context, cfg BackfillConfig) (BackfillResult, error) {
	repoRoot := strings.TrimSpace(cfg.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("release backfill-notes: resolve repo path: %w", err)
	}
	minVersion, hasMin, err := parseOptionalVersion("min-version", cfg.MinVersion)
	if err != nil {
		return BackfillResult{}, err
	}
	maxVersion, hasMax, err := parseOptionalVersion("max-version", cfg.MaxVersion)
	if err != nil {
		return BackfillResult{}, err
	}

	changelogPath := filepath.Join(absRepo, "CHANGELOG.md")
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("release backfill-notes: read CHANGELOG.md: %w", err)
	}
	preamble, entries, err := parseChangelogEntries(string(data))
	if err != nil {
		return BackfillResult{}, err
	}
	if len(entries) == 0 {
		return BackfillResult{}, fmt.Errorf("release backfill-notes: CHANGELOG.md has no mars-harness release markers")
	}

	var result BackfillResult
	rebuilt := make([]string, len(entries))
	for i, entry := range entries {
		rebuilt[i] = entry.Raw
		if !versionInRange(entry.Version, minVersion, hasMin, maxVersion, hasMax) {
			continue
		}

		head, err := resolveCommit(ctx, absRepo, entry.Commit)
		if err != nil {
			return BackfillResult{}, fmt.Errorf("release backfill-notes: release %s marker %s is unavailable: %w", entry.Version, entry.Commit, err)
		}
		base := ""
		if i+1 < len(entries) {
			base, err = resolveCommit(ctx, absRepo, entries[i+1].Commit)
			if err != nil {
				return BackfillResult{}, fmt.Errorf("release backfill-notes: release %s base marker %s is unavailable: %w", entry.Version, entries[i+1].Commit, err)
			}
		}

		commits, err := readEntryCommits(ctx, absRepo, entry, base, head)
		if err != nil {
			return BackfillResult{}, err
		}
		if len(commits) == 0 {
			return BackfillResult{}, fmt.Errorf("release backfill-notes: release %s has no non-release commits in marker range", entry.Version)
		}

		item := BackfillEntryResult{
			Version:     entry.Version,
			BaseRef:     shortRef(base),
			HeadRef:     shortRef(head),
			CommitCount: len(commits),
		}
		if hasCompleteCurrentNarrative(entry.Body) {
			result.Entries = append(result.Entries, item)
			continue
		}

		next := buildBackfilledEntry(entry, renderReleaseNarrative(commits))
		changed := strings.TrimSpace(next) != strings.TrimSpace(entry.Raw)
		item.Changed = changed
		rebuilt[i] = next
		result.Entries = append(result.Entries, item)
		if changed {
			result.Changed = append(result.Changed, entry.Version)
		}
	}
	sort.SliceStable(result.Entries, func(i, j int) bool {
		return compareSemVer(result.Entries[i].Version, result.Entries[j].Version) > 0
	})

	var buf bytes.Buffer
	buf.WriteString(preamble)
	for _, raw := range rebuilt {
		buf.WriteString(raw)
	}
	result.Changelog = strings.TrimRight(buf.String(), "\n") + "\n"

	if len(result.Changed) == 0 {
		return result, nil
	}
	if cfg.Check {
		return result, fmt.Errorf("release backfill-notes: CHANGELOG.md has %d release entries that need backfill", len(result.Changed))
	}
	if cfg.DryRun {
		return result, nil
	}
	if err := os.WriteFile(changelogPath, []byte(result.Changelog), 0o644); err != nil {
		return BackfillResult{}, fmt.Errorf("release backfill-notes: write CHANGELOG.md: %w", err)
	}
	result.UpdatedFiles = []string{"CHANGELOG.md"}
	return result, nil
}

func parseOptionalVersion(flag, value string) (SemVer, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return SemVer{}, false, nil
	}
	version, err := ParseSemVer(value)
	if err != nil {
		return SemVer{}, false, fmt.Errorf("release backfill-notes: invalid %s: %w", flag, err)
	}
	return version, true, nil
}

func parseChangelogEntries(text string) (string, []changelogEntry, error) {
	matches := changelogReleaseRE.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil, nil
	}
	entries := make([]changelogEntry, 0, len(matches))
	for i, match := range matches {
		rawStart := match[0]
		headerEnd := match[1]
		rawEnd := len(text)
		if i+1 < len(matches) {
			rawEnd = matches[i+1][0]
		}
		versionText := text[match[2]:match[3]]
		markerVersion := text[match[4]:match[5]]
		if markerVersion != versionText {
			return "", nil, fmt.Errorf("release backfill-notes: heading version %s does not match marker version %s", versionText, markerVersion)
		}
		version, err := ParseSemVer(versionText)
		if err != nil {
			return "", nil, err
		}
		entries = append(entries, changelogEntry{
			Version: version,
			Header:  text[rawStart:headerEnd],
			Body:    text[headerEnd:rawEnd],
			Raw:     text[rawStart:rawEnd],
			Commit:  text[match[6]:match[7]],
		})
	}
	return text[:matches[0][0]], entries, nil
}

func versionInRange(version, min SemVer, hasMin bool, max SemVer, hasMax bool) bool {
	if hasMin && compareSemVer(version, min) < 0 {
		return false
	}
	if hasMax && compareSemVer(version, max) > 0 {
		return false
	}
	return true
}

func compareSemVer(a, b SemVer) int {
	if a.Major != b.Major {
		return a.Major - b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor - b.Minor
	}
	return a.Patch - b.Patch
}

func resolveCommit(ctx context.Context, repoRoot, ref string) (string, error) {
	out, err := git(ctx, repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func readCommitsThrough(ctx context.Context, repoRoot, baseRef, headRef string) ([]Commit, error) {
	spec := headRef
	if strings.TrimSpace(baseRef) != "" {
		spec = baseRef + ".." + headRef
	}
	args := []string{"log", "--reverse", "--format=%H%x1f%h%x1f%s%x1f%b%x1e", spec}
	out, err := git(ctx, repoRoot, args...)
	if err != nil {
		return nil, err
	}
	return parseGitLogCommits(out), nil
}

func readEntryCommits(ctx context.Context, repoRoot string, entry changelogEntry, baseRef, headRef string) ([]Commit, error) {
	if baseRef == "" || gitOK(ctx, repoRoot, "merge-base", "--is-ancestor", baseRef, headRef) {
		return readCommitsThrough(ctx, repoRoot, baseRef, headRef)
	}
	refs := changelogCommitRefs(entry.Body)
	if len(refs) == 0 {
		return nil, fmt.Errorf("release backfill-notes: release %s marker range is non-linear and the entry has no commit references to reuse", entry.Version)
	}
	return readCommitsByRefs(ctx, repoRoot, entry.Version, refs)
}

func readCommitsByRefs(ctx context.Context, repoRoot string, version SemVer, refs []string) ([]Commit, error) {
	commits := make([]Commit, 0, len(refs))
	seen := make(map[string]bool)
	for _, ref := range refs {
		out, err := git(ctx, repoRoot, "log", "-1", "--format=%H%x1f%h%x1f%s%x1f%b%x1e", ref)
		if err != nil {
			return nil, fmt.Errorf("release backfill-notes: release %s references unavailable commit %s: %w", version, ref, err)
		}
		items := parseGitLogCommits(out)
		if len(items) == 0 {
			continue
		}
		commit := items[0]
		if seen[commit.Hash] {
			continue
		}
		seen[commit.Hash] = true
		commits = append(commits, commit)
	}
	return commits, nil
}

func changelogCommitRefs(body string) []string {
	matches := changelogCommitRE.FindAllStringSubmatch(body, -1)
	refs := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		ref := strings.ToLower(match[1])
		if seen[ref] {
			continue
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	return refs
}

func parseGitLogCommits(out string) []Commit {
	out = strings.TrimRight(out, "\x1e\n")
	if strings.TrimSpace(out) == "" {
		return nil
	}
	records := strings.Split(out, "\x1e")
	commits := make([]Commit, 0, len(records))
	for _, record := range records {
		record = strings.Trim(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x1f", 4)
		if len(fields) < 4 {
			continue
		}
		commit := Commit{
			Hash:    strings.TrimSpace(fields[0]),
			Short:   strings.TrimSpace(fields[1]),
			Subject: strings.TrimSpace(fields[2]),
			Body:    strings.TrimSpace(fields[3]),
		}
		commit.Type, commit.Scope, commit.Message, commit.Breaking = parseSubject(commit.Subject, commit.Body)
		if commit.Type == "release" {
			continue
		}
		commits = append(commits, commit)
	}
	return commits
}

func buildBackfilledEntry(entry changelogEntry, narrative string) string {
	rest := stripLeadingNarrativeSections(entry.Body)
	parts := []string{strings.TrimRight(entry.Header, "\n"), "", strings.TrimSpace(narrative)}
	if rest != "" {
		parts = append(parts, "", rest)
	}
	return strings.Join(parts, "\n") + "\n\n"
}

func hasCompleteCurrentNarrative(body string) bool {
	sections := map[string]*strings.Builder{
		"impact":       &strings.Builder{},
		"why":          &strings.Builder{},
		"what changed": &strings.Builder{},
	}
	current := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "### ") {
			title := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "### ")))
			if _, ok := sections[title]; ok {
				current = title
			} else {
				current = ""
			}
			continue
		}
		if current != "" {
			sections[current].WriteString(line)
			sections[current].WriteByte('\n')
		}
	}
	for _, title := range []string{"impact", "why", "what changed"} {
		if strings.TrimSpace(sections[title].String()) == "" {
			return false
		}
	}
	return true
}

func stripLeadingNarrativeSections(body string) string {
	remaining := strings.TrimSpace(body)
	for {
		if !strings.HasPrefix(remaining, "### ") {
			return remaining
		}
		lineEnd := strings.Index(remaining, "\n")
		if lineEnd == -1 {
			if isNarrativeSectionTitle(strings.TrimPrefix(remaining, "### ")) {
				return ""
			}
			return remaining
		}
		title := strings.TrimSpace(strings.TrimPrefix(remaining[:lineEnd], "### "))
		if !isNarrativeSectionTitle(title) {
			return remaining
		}
		afterHeading := remaining[lineEnd+1:]
		next := sectionHeadingRE.FindStringIndex(afterHeading)
		if next == nil {
			return ""
		}
		remaining = strings.TrimSpace(afterHeading[next[0]:])
	}
}

func isNarrativeSectionTitle(title string) bool {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "impact", "why", "what changed", "why this release matters":
		return true
	default:
		return false
	}
}

func shortRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if len(ref) <= 12 {
		return ref
	}
	return ref[:12]
}
