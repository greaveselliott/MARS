/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-012-self-improvement-loop.md
- docs/product-specs/product-surface.md
*/
package doctor

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars/internal/scanner"
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
	assert.True(t, names["release-access"])
	assert.True(t, names["version-drift"])
	assert.True(t, names["operating-model"])
	assert.True(t, names["deterministic-remediation"])
	assert.True(t, names["role-registry"])
	assert.True(t, names["active-plan-hygiene"])
	assert.True(t, names["ticket-drain"])
}

func TestCheckReleaseAccessSkipsWithSkipRemote(t *testing.T) {
	t.Parallel()
	result := checkReleaseAccess(Config{SkipRemote: true})
	assert.Equal(t, "release-access", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "skipped")
}

func TestCheckReleaseAccessUsesSelectedConfig(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("MARS_GITHUB_TOKEN", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	configPath := filepath.Join(t.TempDir(), "selected.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("github_token: selected-token\n"), 0o600))

	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	requests := 0
	http.DefaultTransport = doctorRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "https://api.github.com/repos/greaveselliott/MARS/releases/latest", req.URL.String())
		if requests == 1 {
			require.Empty(t, req.Header.Get("Authorization"))
			return doctorHTTPResponse(http.StatusNotFound), nil
		}
		require.Equal(t, "Bearer selected-token", req.Header.Get("Authorization"))
		return doctorHTTPResponse(http.StatusOK), nil
	})

	result := checkReleaseAccess(Config{ConfigPath: configPath})
	require.Equal(t, statusOK, result.Status)
	require.Contains(t, result.Message, "authenticated")
	require.Equal(t, 2, requests)
}

type doctorRoundTripFunc func(*http.Request) (*http.Response, error)

func (f doctorRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func doctorHTTPResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("{}")),
	}
}

func TestCheckModelRegistryRequiresExactProvenance(t *testing.T) {
	t.Parallel()

	result := checkModelRegistry(Config{})
	assert.Equal(t, "model-registry", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "exact artifact and publisher provenance")
}

func TestCheckGoVersionDoesNotRunOutsideMarsSource(t *testing.T) {
	t.Parallel()
	target := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(target, "go.mod"), []byte("module example.com/target\n"), 0o644))

	for _, cfg := range []Config{{}, {RepoPath: target}} {
		called := false
		result := checkGoVersionWithRunner(cfg, func() ([]byte, error) {
			called = true
			return nil, errors.New("must not run")
		})
		assert.Equal(t, "go-version", result.Name)
		assert.Equal(t, statusOK, result.Status)
		assert.Contains(t, result.Message, "not required")
		assert.False(t, called)
	}
}

func TestCheckGoVersionEnforcesExactMarsSourceFloor(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module github.com/greaveselliott/mars\n"), 0o644))

	tests := []struct {
		name       string
		output     string
		runErr     error
		wantStatus string
		wantText   string
	}{
		{name: "missing", runErr: errors.New("not found"), wantStatus: statusFail, wantText: "go not found"},
		{name: "malformed", output: "go version devel", wantStatus: statusFail, wantText: "could not parse"},
		{name: "below floor", output: "go version go1.25.11 darwin/arm64", wantStatus: statusFail, wantText: "need >= 1.25.12"},
		{name: "minimum", output: "go version go1.25.12 darwin/arm64", wantStatus: statusOK, wantText: "go 1.25.12"},
		{name: "release", output: "go version go1.27.0 darwin/arm64", wantStatus: statusOK, wantText: "go 1.27.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkGoVersionWithRunner(Config{RepoPath: repo}, func() ([]byte, error) {
				return []byte(tt.output), tt.runErr
			})
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Contains(t, result.Message, tt.wantText)
		})
	}
}

func TestParseGoVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input         string
		expectMajor   int
		expectMinor   int
		expectPatch   int
		expectFailure bool
	}{
		{"go version go1.25.11 darwin/arm64", 1, 25, 11, false},
		{"go version go1.25.12 linux/amd64", 1, 25, 12, false},
		{"go version go1.27.0 linux/arm64", 1, 27, 0, false},
		{"go version go1.25rc1 linux/amd64", 0, 0, 0, true},
		{"go version go1.25.12rc1 linux/amd64", 0, 0, 0, true},
		{"no version here", 0, 0, 0, true},
	}
	for _, tt := range tests {
		major, minor, patch, err := parseGoVersion(tt.input)
		if tt.expectFailure {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.expectMajor, major)
			assert.Equal(t, tt.expectMinor, minor)
			assert.Equal(t, tt.expectPatch, patch)
		}
	}
}

func TestGoVersionAtLeastMinimum(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		major, minor, patch int
		want                bool
	}{
		{1, 25, 11, false},
		{1, 25, 12, true},
		{1, 26, 0, true},
		{2, 0, 0, true},
	} {
		assert.Equal(t, tt.want, goVersionAtLeastMinimum(tt.major, tt.minor, tt.patch))
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

func TestCheckDeterministicRemediationHealthReportsMissingHarness(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result := checkDeterministicRemediationHealth(Config{RepoPath: dir})
	assert.Equal(t, "deterministic-remediation", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "manifest:validate-or-init")
	assert.Contains(t, result.Fix, "mars init")
}

func TestCheckDeterministicRemediationHealthReportsMissingMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harness", "manifest.yaml"), []byte("name: test\nroles: {}\n"), 0o644))

	result := checkDeterministicRemediationHealth(Config{RepoPath: dir})
	assert.Equal(t, "deterministic-remediation", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "generated-docs:update-missing-defaults")
	assert.Contains(t, result.Fix, "update harness")
}

func TestCheckDeterministicRemediationHealthPassesGeneratedHarness(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, scanner.Init(dir, false))

	result := checkDeterministicRemediationHealth(Config{RepoPath: dir})
	assert.Equal(t, "deterministic-remediation", result.Name)
	assert.Equal(t, statusOK, result.Status)
	assert.Contains(t, result.Message, "deterministic remediation recipes")
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
	assert.Contains(t, result.Fix, "mars run janitor")
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
