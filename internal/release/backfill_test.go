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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackfillNotesRewritesHistoricalNarrativeAndPreservesBuckets(t *testing.T) {
	t.Parallel()
	dir, first, second := backfillFixture(t)
	require.NoError(t, os.Chmod(filepath.Join(dir, "CHANGELOG.md"), 0o600))

	result, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir})
	require.NoError(t, err)
	require.ElementsMatch(t, []SemVer{{Major: 0, Minor: 1, Patch: 0}, {Major: 0, Minor: 1, Patch: 1}}, result.Changed)
	require.Len(t, result.Entries, 2)
	require.ElementsMatch(t, []string{"CHANGELOG.md"}, result.UpdatedFiles)

	changelog := readChangelog(t, dir)
	require.NotContains(t, changelog, "### Why This Release Matters")
	require.Contains(t, changelog, "## [0.1.1] - 2026-05-02")
	require.Contains(t, changelog, "### Impact")
	require.Contains(t, changelog, "Operators and future agents get clearer guidance because document release backfill.")
	require.Contains(t, changelog, "### Why")
	require.Contains(t, changelog, "This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.")
	require.Contains(t, changelog, "### What Changed")
	require.Contains(t, changelog, "Changed document release backfill ("+second.short+").")
	require.Contains(t, changelog, "### Documentation")
	require.Contains(t, changelog, "- **docs:** Document release backfill ("+second.short+")")
	require.Contains(t, changelog, "## [0.1.0] - 2026-05-01")
	require.Contains(t, changelog, "**cli:** Operators gain new capability: add first command.")
	require.Contains(t, changelog, "- **cli:** Add first command ("+first.short+")")
	requireReleaseFileMode(t, filepath.Join(dir, "CHANGELOG.md"), 0o600)
}

func TestBackfillNotesRejectsSymlinkedChangelogWithoutOutsideMutation(t *testing.T) {
	repo := t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "CHANGELOG.md")
	original := []byte("# Outside changelog\n")
	require.NoError(t, os.WriteFile(sentinel, original, 0o600))
	require.NoError(t, os.Symlink(sentinel, filepath.Join(repo, "CHANGELOG.md")))

	_, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: repo})
	require.Error(t, err)
	data, readErr := os.ReadFile(sentinel)
	require.NoError(t, readErr)
	require.Equal(t, original, data)
}

func TestBackfillNotesDryRunDoesNotWrite(t *testing.T) {
	t.Parallel()
	dir, _, _ := backfillFixture(t)
	before := readChangelog(t, dir)

	result, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir, DryRun: true})
	require.NoError(t, err)
	require.NotEmpty(t, result.Changed)
	require.Empty(t, result.UpdatedFiles)
	require.Equal(t, before, readChangelog(t, dir))
}

func TestBackfillNotesCheckReportsStaleAndPassesAfterRewrite(t *testing.T) {
	t.Parallel()
	dir, _, _ := backfillFixture(t)

	_, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir, Check: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "need backfill")

	_, err = BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir})
	require.NoError(t, err)
	result, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir, Check: true})
	require.NoError(t, err)
	require.Empty(t, result.Changed)
}

func TestBackfillNotesPreservesCompleteCurrentNarrative(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs.txt"), []byte("docs"), 0o644))
	gitCommit(t, dir, "docs(release): document release handoff")
	head := gitOutput(t, dir, "rev-parse", "--short=12", "HEAD")
	short := gitOutput(t, dir, "rev-parse", "--short=7", "HEAD")
	changelog := `# Changelog

## [0.1.0] - 2026-05-02
<!-- mars-release: version=0.1.0 commit=` + head + ` -->

### Impact
- **release:** Operators retain the richer release explanation already written for this version.

### Why
- **release:** This matters because historical backfill should fill gaps without flattening human-quality release history.

### What Changed
- **release:** The entry already explains dispatch, evidence, and operator-visible behavior.

### Documentation
- **release:** Document release handoff (` + short + `)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(changelog), 0o644))

	result, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir, Check: true})
	require.NoError(t, err)
	require.Empty(t, result.Changed)
	require.Equal(t, changelog, readChangelog(t, dir))
}

func TestBackfillNotesParsesLegacyReleaseMarker(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs.txt"), []byte("docs"), 0o644))
	gitCommit(t, dir, "docs(release): document release handoff")
	head := gitOutput(t, dir, "rev-parse", "--short=12", "HEAD")
	changelog := `# Changelog

## [0.1.0] - 2026-05-02
<!-- mars-harness-release: version=0.1.0 commit=` + head + ` -->

### Documentation
- Document release handoff
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(changelog), 0o644))

	result, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir})
	require.NoError(t, err)
	require.NotEmpty(t, result.Changed)
}

func TestBackfillNotesHonorsVersionRange(t *testing.T) {
	t.Parallel()
	dir, _, _ := backfillFixture(t)

	result, err := BackfillNotes(context.Background(), BackfillConfig{
		RepoRoot:   dir,
		MinVersion: "0.1.1",
		MaxVersion: "0.1.1",
	})
	require.NoError(t, err)
	require.Equal(t, []SemVer{{Major: 0, Minor: 1, Patch: 1}}, result.Changed)

	changelog := readChangelog(t, dir)
	require.Contains(t, changelog, "## [0.1.1] - 2026-05-02")
	require.Contains(t, changelog, "Operators and future agents get clearer guidance because document release backfill.")
	require.Contains(t, changelog, "## [0.1.0] - 2026-05-01")
	require.Contains(t, changelog, "### Why This Release Matters\nInitial narrative.")
}

func TestBackfillNotesFallsBackToEntryRefsForNonLinearMarkerRanges(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0o644))
	gitCommit(t, dir, "feat(core): ship first feature")
	base := gitOutput(t, dir, "rev-parse", "--short=12", "HEAD")
	gitRun(t, dir, "checkout", "--orphan", "side")
	require.NoError(t, os.Remove(filepath.Join(dir, "feature.txt")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix"), 0o644))
	gitCommit(t, dir, "fix(core): repair side branch")
	head := gitOutput(t, dir, "rev-parse", "--short=12", "HEAD")
	short := gitOutput(t, dir, "rev-parse", "--short=7", "HEAD")
	changelog := `# Changelog

## [0.2.0] - 2026-05-02
<!-- mars-release: version=0.2.0 commit=` + head + ` -->

### Why This Release Matters
Old narrative.

### Fixes
- **core:** Repair side branch (` + short + `)

## [0.1.0] - 2026-05-01
<!-- mars-release: version=0.1.0 commit=` + base + ` -->

### Features
- **core:** Ship first feature (` + base[:7] + `)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(changelog), 0o644))

	result, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir, MinVersion: "0.2.0", MaxVersion: "0.2.0"})
	require.NoError(t, err)
	require.Equal(t, []SemVer{{Major: 0, Minor: 2, Patch: 0}}, result.Changed)
	require.Contains(t, readChangelog(t, dir), "**core:** Operators see improved reliability because repair side branch.")
}

func TestBackfillNotesFailsWhenMarkerIsUnavailable(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(`# Changelog

## [0.1.0] - 2026-05-01
<!-- mars-release: version=0.1.0 commit=deadbeefdead -->

### Features
- Missing marker
`), 0o644))

	_, err := BackfillNotes(context.Background(), BackfillConfig{RepoRoot: dir})
	require.Error(t, err)
	require.Contains(t, err.Error(), "marker deadbeefdead is unavailable")
}

type backfillCommit struct {
	full  string
	short string
}

func backfillFixture(t *testing.T) (string, backfillCommit, backfillCommit) {
	t.Helper()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli.txt"), []byte("cli"), 0o644))
	gitCommit(t, dir, "feat(cli): add first command")
	first := backfillCommit{
		full:  gitOutput(t, dir, "rev-parse", "HEAD"),
		short: gitOutput(t, dir, "rev-parse", "--short=7", "HEAD"),
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs.txt"), []byte("docs"), 0o644))
	gitCommit(t, dir, "docs: document release backfill")
	second := backfillCommit{
		full:  gitOutput(t, dir, "rev-parse", "HEAD"),
		short: gitOutput(t, dir, "rev-parse", "--short=7", "HEAD"),
	}
	changelog := `# Changelog

## [0.1.1] - 2026-05-02
<!-- mars-release: version=0.1.1 commit=` + second.full[:12] + ` -->

### Why This Release Matters
Old narrative.

### Documentation
- **docs:** Document release backfill (` + second.short + `)

## [0.1.0] - 2026-05-01
<!-- mars-release: version=0.1.0 commit=` + first.full[:12] + ` -->

### Why This Release Matters
Initial narrative.

### Features
- **cli:** Add first command (` + first.short + `)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(changelog), 0o644))
	return dir, first, second
}

func readChangelog(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	require.NoError(t, err)
	return string(data)
}
