/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_testMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")

	result, err := Run(Config{TestMode: true})
	require.NoError(t, err)
	assert.Greater(t, result.StepsRun, 0)

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".mars-harness")
	assert.DirExists(t, baseDir)
	assert.FileExists(t, filepath.Join(baseDir, "config.yaml"))
	assert.FileExists(t, filepath.Join(baseDir, "hardware.yaml"))
	assert.FileExists(t, filepath.Join(home, ".profile"))
}

func TestRun_idempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")

	result1, err := Run(Config{TestMode: true})
	require.NoError(t, err)
	firstRun := result1.StepsRun

	result2, err := Run(Config{TestMode: true})
	require.NoError(t, err)
	assert.Equal(t, 0, result2.StepsRun, "second run should skip all steps")
	assert.Equal(t, firstRun, result2.StepsSkipped)
}

func TestRun_dryRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")

	result, err := Run(Config{TestMode: true, DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, 0, result.StepsRun)
	assert.Greater(t, result.StepsSkipped, 0)

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".mars-harness")
	_, statErr := os.Stat(baseDir)
	assert.True(t, os.IsNotExist(statErr), "dry-run should not create directories")
}

func TestRun_createsSubdirectories(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")

	_, err := Run(Config{TestMode: true})
	require.NoError(t, err)

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".mars-harness")
	for _, sub := range []string{"models", "bin", "traces", "db"} {
		assert.DirExists(t, filepath.Join(baseDir, sub))
	}
}

func TestBuildSteps_testModeSkipsDownloadAndGitHub(t *testing.T) {
	t.Parallel()
	steps := buildSteps("/tmp/test-mars", Config{TestMode: true})

	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	assert.NotContains(t, names, "install-llama-server")
	assert.NotContains(t, names, "download-models")
	assert.NotContains(t, names, "github-setup")
}

func TestBuildSteps_fullMode(t *testing.T) {
	t.Parallel()
	steps := buildSteps("/tmp/test-mars", Config{})

	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	assert.Contains(t, names, "create-directories")
	assert.Contains(t, names, "write-config")
	assert.Contains(t, names, "detect-hardware")
	assert.Contains(t, names, "configure-shell-path")
	assert.Contains(t, names, "install-llama-server")
	assert.Contains(t, names, "download-models")
	assert.NotContains(t, names, "github-setup")
}

func TestBuildSteps_githubOptIn(t *testing.T) {
	t.Parallel()
	steps := buildSteps("/tmp/test-mars", Config{EnableGitHub: true})

	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	assert.Contains(t, names, "github-setup")
}

func TestCreateDirectoriesStep_execute(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	baseDir := filepath.Join(dir, ".mars-harness")

	step := createDirectoriesStep(baseDir)

	done, err := step.Check()
	require.NoError(t, err)
	assert.False(t, done)

	require.NoError(t, step.Execute())

	done, err = step.Check()
	require.NoError(t, err)
	assert.True(t, done)
}

func TestWriteDefaultConfigStep_execute(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	baseDir := filepath.Join(dir, ".mars-harness")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	step := writeDefaultConfigStep(baseDir)

	done, _ := step.Check()
	assert.False(t, done)

	require.NoError(t, step.Execute())

	done, _ = step.Check()
	assert.True(t, done)

	data, err := os.ReadFile(filepath.Join(baseDir, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "models_dir")
	assert.Contains(t, string(data), "performance_profile: auto")
	assert.Contains(t, string(data), "llama_parallel: 1")
}

func TestDetectHardwareStep_execute(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	baseDir := filepath.Join(dir, ".mars-harness")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	step := detectHardwareStep(baseDir)

	done, _ := step.Check()
	assert.False(t, done)
	require.NoError(t, step.Execute())

	done, err := step.Check()
	require.NoError(t, err)
	assert.True(t, done)
	data, err := os.ReadFile(filepath.Join(baseDir, "hardware.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "profile:")
	assert.Contains(t, string(data), "os:")
}

func TestGithubPlaceholderStep_execute(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	baseDir := filepath.Join(dir, ".mars-harness")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	step := githubPlaceholderStep(baseDir)

	done, _ := step.Check()
	assert.False(t, done)
	require.NoError(t, step.Execute())

	done, err := step.Check()
	require.NoError(t, err)
	assert.True(t, done)
	data, err := os.ReadFile(filepath.Join(baseDir, ".github-configured"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "pending")
}
