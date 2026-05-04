/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/product-specs/product-surface.md
*/
package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_returnsResults(t *testing.T) {
	t.Parallel()
	results := Run(Config{
		ConfigPath: "/nonexistent/config.yaml",
		DBPath:     "/nonexistent/mars.db",
		SkipRemote: true,
	})
	assert.GreaterOrEqual(t, len(results), 5)

	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
		assert.Contains(t, []string{"ok", "warn", "fail"}, r.Status)
		assert.NotEmpty(t, r.Message)
		assert.Greater(t, r.Duration.Nanoseconds(), int64(0))
	}
	assert.True(t, names["go-version"])
	assert.True(t, names["config-file"])
	assert.True(t, names["models-dir"])
	assert.True(t, names["database"])
	assert.True(t, names["llama-server"])
	assert.True(t, names["disk-space"])
	assert.True(t, names["version-drift"])
	assert.True(t, names["operating-model"])
	assert.True(t, names["role-registry"])
	assert.True(t, names["active-plan-hygiene"])
	assert.True(t, names["ticket-drain"])
}

func TestCheckGoVersion_findsGo(t *testing.T) {
	t.Parallel()
	result := checkGoVersion(Config{})
	assert.Equal(t, "go-version", result.Name)
	assert.Equal(t, statusOK, result.Status, "expected go to be installed; message: %s", result.Message)
}

func TestParseGoVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input         string
		expectMajor   int
		expectMinor   int
		expectFailure bool
	}{
		{"go version go1.22.4 darwin/arm64", 1, 22, false},
		{"go version go1.23.0 linux/amd64", 1, 23, false},
		{"go version go1.21.0 linux/amd64", 1, 21, false},
		{"no version here", 0, 0, true},
	}
	for _, tt := range tests {
		major, minor, err := parseGoVersion(tt.input)
		if tt.expectFailure {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.expectMajor, major)
			assert.Equal(t, tt.expectMinor, minor)
		}
	}
}

func TestCheckConfigFile_exists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("test: true\n"), 0o644))

	result := checkConfigFile(Config{ConfigPath: cfgPath})
	assert.Equal(t, statusOK, result.Status)
}

func TestCheckConfigFile_missing(t *testing.T) {
	t.Parallel()
	result := checkConfigFile(Config{ConfigPath: "/nonexistent/config.yaml"})
	assert.Equal(t, statusWarn, result.Status)
	assert.NotEmpty(t, result.Fix)
}

func TestCheckDBAccessible_dirExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "mars.db")

	result := checkDBAccessible(Config{DBPath: dbPath})
	assert.Equal(t, statusOK, result.Status)
}

func TestCheckDBAccessible_dirMissing(t *testing.T) {
	t.Parallel()
	result := checkDBAccessible(Config{DBPath: "/nonexistent/dir/mars.db"})
	assert.Equal(t, statusWarn, result.Status)
	assert.NotEmpty(t, result.Fix)
}

func TestCheckDiskSpace_passes(t *testing.T) {
	t.Parallel()
	result := checkDiskSpace(Config{})
	assert.Equal(t, "disk-space", result.Name)
	assert.Contains(t, []string{statusOK, statusFail, statusWarn}, result.Status)
}

func TestCheckVersionDrift_reportsMissingHarnessMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".harness"), 0o755))

	result := checkVersionDrift(Config{
		RepoPath:       dir,
		CurrentVersion: "0.6.0",
		SkipRemote:     true,
	})
	assert.Equal(t, "version-drift", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Fix, "update harness")
}

func TestCheckOperatingModelHealth_reportsDrift(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result := checkOperatingModelHealth(Config{RepoPath: dir})
	assert.Equal(t, "operating-model", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "operating model drift")
	assert.Contains(t, result.Fix, "update harness")
}

func TestCheckRoleRegistryHealthPassesGeneratedRegistry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, scanner.Init(dir, false))

	result := checkRoleRegistryHealth(Config{RepoPath: dir})
	assert.Equal(t, "role-registry", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "matches manifest")
}

func TestCheckRoleRegistryHealthReportsDrift(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, scanner.Init(dir, false))
	require.NoError(t, os.Remove(filepath.Join(dir, "docs", "roles", "ROLES.md")))

	result := checkRoleRegistryHealth(Config{RepoPath: dir})
	assert.Equal(t, "role-registry", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "role registry is missing")
	assert.Contains(t, result.Fix, "docs/roles/ROLES.md")
}

func TestCheckRoleRegistryHealthSkipsWithoutRepo(t *testing.T) {
	t.Parallel()
	result := checkRoleRegistryHealth(Config{})
	assert.Equal(t, "role-registry", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "skipped")
}

func TestCheckActivePlanHygieneReportsIssue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDoctorFile(t, dir, "docs/exec-plans/active/current-operating-plan.md", "# Current Operating Plan\n\n**Status:** Active\n**Priority:** P0\n\n- `docs/tickets/backlog/` contains MH-001.\n")
	writeDoctorFile(t, dir, "docs/exec-plans/active/second-plan.md", "# Second Plan\n\n**Status:** Active\n**Priority:** P1\n")
	writeDoctorFile(t, dir, "docs/tickets/done/MH-001-done.md", "# done\n")

	result := checkActivePlanHygiene(Config{RepoPath: dir})
	assert.Equal(t, "active-plan-hygiene", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "active-plan hygiene")
	assert.NotEmpty(t, result.Fix)
}

func TestCheckActivePlanHygieneSkipsWithoutRepo(t *testing.T) {
	t.Parallel()
	result := checkActivePlanHygiene(Config{})
	assert.Equal(t, "active-plan-hygiene", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "skipped")
}

func TestCheckTicketDrainHealthReportsStaleInProgress(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDoctorFile(t, dir, "docs/tickets/in-progress/T-001-stale.md", `---
id: T-001
title: Stale
last_attempt: "2026-04-01"
blocker: none
blocked_by: []
---

# Stale
`)

	result := checkTicketDrainHealth(Config{RepoPath: dir})
	assert.Equal(t, "ticket-drain", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "stale eligible in-progress")
	assert.Contains(t, result.Fix, "blocked_by")
	assert.Contains(t, result.Fix, "mars-harness run janitor")
}

func TestCheckTicketDrainHealthSkipsBlockedInProgress(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDoctorFile(t, dir, "docs/tickets/in-progress/T-001-blocked.md", `---
id: T-001
title: Blocked
last_attempt: "2026-04-01"
blocker: "waiting for T-002"
blocked_by: ["T-002"]
---

# Blocked
`)

	result := checkTicketDrainHealth(Config{RepoPath: dir})
	assert.Equal(t, "ticket-drain", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "no stale")
}

func TestFormatText(t *testing.T) {
	t.Parallel()
	results := []CheckResult{
		{Name: "test-check", Status: statusOK, Message: "all good"},
		{Name: "test-warn", Status: statusWarn, Message: "minor issue", Fix: "run fix"},
	}
	out := FormatText(results)
	assert.Contains(t, out, "test-check")
	assert.Contains(t, out, "all good")
	assert.Contains(t, out, "fix: run fix")
}

func TestFormatJSON(t *testing.T) {
	t.Parallel()
	results := []CheckResult{
		{Name: "test-check", Status: statusOK, Message: "ok"},
	}
	out, err := FormatJSON(results)
	require.NoError(t, err)
	assert.Contains(t, out, `"name": "test-check"`)
	assert.Contains(t, out, `"status": "ok"`)
}

func TestHasFailures(t *testing.T) {
	t.Parallel()
	assert.False(t, HasFailures([]CheckResult{
		{Status: statusOK},
		{Status: statusWarn},
	}))
	assert.True(t, HasFailures([]CheckResult{
		{Status: statusOK},
		{Status: statusFail},
	}))
}

func TestStatusIcon(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ok", statusIcon(statusOK))
	assert.Equal(t, "!!", statusIcon(statusWarn))
	assert.Equal(t, "FAIL", statusIcon(statusFail))
	assert.Equal(t, "??", statusIcon("unknown"))
}

func writeDoctorFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
