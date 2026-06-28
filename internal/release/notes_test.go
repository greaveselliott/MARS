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
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrepareGeneratesVersionAndChangelog(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal", "buildinfo"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "buildinfo", "version.go"), []byte("package buildinfo\n\nconst DefaultVersion = \"0.1.0\"\n"), 0o644))
	gitCommit(t, dir, "chore: initial release state")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0o644))
	gitCommit(t, dir, "feat(api): add search endpoint")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bug.txt"), []byte("fix"), 0o644))
	gitCommit(t, dir, "fix: handle empty results")

	result, err := Prepare(context.Background(), Config{
		RepoRoot: dir,
		Bump:     BumpAuto,
		Now:      time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, "0.2.0", result.NextVersion.String())
	require.Equal(t, BumpMinor, result.Bump)
	require.ElementsMatch(t, []string{"VERSION", "internal/buildinfo/version.go", "CHANGELOG.md"}, result.UpdatedFiles)

	version, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	require.NoError(t, err)
	require.Equal(t, "0.2.0\n", string(version))

	buildInfo, err := os.ReadFile(filepath.Join(dir, "internal", "buildinfo", "version.go"))
	require.NoError(t, err)
	require.Contains(t, string(buildInfo), `DefaultVersion = "0.2.0"`)

	changelog, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	require.NoError(t, err)
	require.Contains(t, string(changelog), "## [0.2.0] - 2026-05-02")
	require.Contains(t, string(changelog), "### Impact")
	require.Contains(t, string(changelog), "**api:** Operators gain new capability: add search endpoint.")
	require.Contains(t, string(changelog), "Operators see improved reliability because handle empty results.")
	require.Contains(t, string(changelog), "### Why")
	require.Contains(t, string(changelog), "**api:** This matters because add search endpoint was missing from the shipped capability set.")
	require.Contains(t, string(changelog), "This matters because handle empty results closes a failure mode or degraded path.")
	require.Contains(t, string(changelog), "### What Changed")
	require.Contains(t, string(changelog), "**api:** Changed add search endpoint")
	require.Contains(t, string(changelog), "Changed handle empty results")
	require.Contains(t, string(changelog), "### Features")
	require.Contains(t, string(changelog), "**api:** Add search endpoint")
	require.Contains(t, string(changelog), "### Fixes")
	require.Contains(t, string(changelog), "Handle empty results")
}

func TestRenderReleaseNarrativeUsesImpactWhyAndWhat(t *testing.T) {
	t.Parallel()
	commits := []Commit{
		{Short: "aaa111", Type: "docs", Scope: "release", Message: "document release process", Body: "Impact: Operators can audit release impact.\nWhy: The changelog needs narrative context.\nWhat: Added impact, why, and what sections."},
		{Short: "bbb222", Type: "test", Message: "cover changelog generation"},
		{Short: "ccc333", Type: "chore", Scope: "deps", Message: "update go modules"},
	}

	summary := renderReleaseNarrative(commits)

	require.Contains(t, summary, "### Impact")
	require.Contains(t, summary, "**release:** Operators can audit release impact.")
	require.Contains(t, summary, "**deps:** Maintainers get a healthier project surface because update go modules.")
	require.Contains(t, summary, "The release carries stronger evidence because cover changelog generation.")
	require.Contains(t, summary, "### Why")
	require.Contains(t, summary, "**release:** The changelog needs narrative context.")
	require.Contains(t, summary, "**deps:** This matters because project health work keeps future delivery predictable.")
	require.Contains(t, summary, "This matters because the project needs durable evidence that the behavior keeps working.")
	require.Contains(t, summary, "### What Changed")
	require.Contains(t, summary, "**release:** Added impact, why, and what sections (aaa111).")
	require.Contains(t, summary, "**deps:** Changed update go modules (ccc333).")
	require.Contains(t, summary, "Changed cover changelog generation (bbb222).")
}

func TestRenderReleaseNarrativeProfilesStructuredDispatch(t *testing.T) {
	t.Parallel()
	commits := []Commit{
		{
			Short:   "c436460",
			Type:    "fix",
			Scope:   "orchestration",
			Message: "carry structured handoff through dispatch",
			Subject: "fix(orchestration): carry structured handoff through dispatch",
		},
	}

	summary := renderReleaseNarrative(commits)

	require.Contains(t, summary, "**orchestration:** Operators and agents get a more reliable delivery loop because handoff and feedback now travel as first-class runtime data through Orchestrator dispatch.")
	require.Contains(t, summary, "**orchestration:** This matters because operating-model shifts lose value when the next owner, expected correction, or supporting evidence only exists in free-form transcript text.")
	require.Contains(t, summary, "**orchestration:** Dispatch triggers now carry the source disposition, including status, next need, ticket ID, reason, evidence links, trace ID, handoff, and feedback, so Orchestrator can validate one target owner before enqueueing follow-up work (c436460).")
	require.NotContains(t, summary, "Operators see improved reliability because carry structured handoff through dispatch.")
}

func TestPrepareUsesChangelogMarkerAsBase(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	gitCommit(t, dir, "feat: initial feature")
	head := gitOutput(t, dir, "rev-parse", "--short=12", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.1.0] - 2026-05-01\n<!-- mars-release: version=0.1.0 commit="+head+" -->\n"), 0o644))
	gitCommit(t, dir, "release: notes 0.1.0")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix"), 0o644))
	gitCommit(t, dir, "fix: one more bug")

	result, err := Prepare(context.Background(), Config{
		RepoRoot: dir,
		Bump:     BumpAuto,
		DryRun:   true,
		Now:      time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, "0.1.1", result.NextVersion.String())
	require.Len(t, result.Commits, 1, "release note commits should not appear in later patch notes")
	require.Contains(t, result.Entry, "One more bug")
}

func TestPrepareUsesLegacyChangelogMarkerAsBase(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	gitCommit(t, dir, "feat: initial feature")
	head := gitOutput(t, dir, "rev-parse", "--short=12", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.1.0] - 2026-05-01\n<!-- mars-harness-release: version=0.1.0 commit="+head+" -->\n"), 0o644))
	gitCommit(t, dir, "release: notes 0.1.0")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix"), 0o644))
	gitCommit(t, dir, "fix: one more bug")

	result, err := Prepare(context.Background(), Config{
		RepoRoot: dir,
		Bump:     BumpAuto,
		DryRun:   true,
		Now:      time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, "0.1.1", result.NextVersion.String())
	require.Len(t, result.Commits, 1)
	require.Contains(t, result.Entry, "<!-- mars-release:")
}

func TestPrepareIgnoresStaleChangelogMarkerAndUsesCurrentVersionTag(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.1.0] - 2026-05-01\n<!-- mars-release: version=0.1.0 commit=deadbeefdead -->\n"), 0o644))
	gitCommit(t, dir, "release: notes 0.1.0")
	gitRun(t, dir, "tag", "v0.1.0")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix"), 0o644))
	gitCommit(t, dir, "fix: one more bug")

	result, err := Prepare(context.Background(), Config{
		RepoRoot: dir,
		Bump:     BumpAuto,
		DryRun:   true,
		Now:      time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, "v0.1.0", result.BaseRef)
	require.Len(t, result.Commits, 1)
	require.Contains(t, result.Entry, "One more bug")
}

func TestPrepareClassifiesDeliveryEvidenceFromDoneTickets(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "tickets", "done", "MH-040-feature.md"), []byte(`---
id: MH-040
title: Feature scenario
work_type: feature
bdd_scenarios: ["F-001-S001", "F-001-S002"]
---

# MH-040
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "tickets", "done", "MH-041-enabler.md"), []byte(`---
id: MH-041
title: Build enabler
work_type: enabler
bdd_scenarios: []
---

# MH-041
`), 0o644))
	gitCommit(t, dir, "chore: initial release state")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0o644))
	gitCommit(t, dir, "feat: deliver scenario evidence (MH-040)")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "enabler.txt"), []byte("enabler"), 0o644))
	gitCommit(t, dir, "docs: record enabler work (MH-041)")

	result, err := Prepare(context.Background(), Config{
		RepoRoot: dir,
		Bump:     BumpAuto,
		DryRun:   true,
		Now:      time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Contains(t, result.Entry, "### Delivery Evidence")
	require.Contains(t, result.Entry, "Shipped feature scenarios: MH-040: F-001-S001, F-001-S002")
	require.Contains(t, result.Entry, "Enabler work: MH-041: Build enabler")
}

func TestInferBump(t *testing.T) {
	t.Parallel()
	require.Equal(t, BumpPatch, inferBump([]Commit{{Type: "docs"}}))
	require.Equal(t, BumpMinor, inferBump([]Commit{{Type: "fix"}, {Type: "feat"}}))
	require.Equal(t, BumpMajor, inferBump([]Commit{{Type: "feat", Breaking: true}}))
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	return dir
}

func gitCommit(t *testing.T, dir, message string) {
	t.Helper()
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", message)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(bytesTrimSpace(out))
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = gitOutput(t, dir, args...)
}

func bytesTrimSpace(value []byte) []byte {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}
	return value
}
