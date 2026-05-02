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
	require.Contains(t, string(changelog), "### Features")
	require.Contains(t, string(changelog), "**api:** Add search endpoint")
	require.Contains(t, string(changelog), "### Fixes")
	require.Contains(t, string(changelog), "Handle empty results")
}

func TestPrepareUsesChangelogMarkerAsBase(t *testing.T) {
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
	require.Len(t, result.Commits, 1, "release note commits should not appear in later patch notes")
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
