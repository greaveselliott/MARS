package doctor

import (
	"os"
	"path/filepath"
	"testing"

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
	os.WriteFile(cfgPath, []byte("test: true\n"), 0o644)

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
