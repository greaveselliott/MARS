/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/product-specs/product-surface.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_missingFileReturnsDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, cfg.ModelsDir)
	require.Equal(t, 9091, cfg.WebhookPort)
	require.Equal(t, 9090, cfg.DashboardPort)
	require.Equal(t, "text", cfg.LogFormat)
	require.Equal(t, "auto", cfg.PerformanceProfile)
	require.Equal(t, 1, cfg.LlamaParallel)
	require.Equal(t, "auto", cfg.LlamaFlashAttention)
}

func TestLoad_parsesYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("models_dir: /custom/models\nlog_format: json\nperformance_profile: balanced\nllama_parallel: 2\nllama_threads: 6\nllama_mlock: true\n"), 0o644))

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "/custom/models", cfg.ModelsDir)
	require.Equal(t, "json", cfg.LogFormat)
	require.Equal(t, "balanced", cfg.PerformanceProfile)
	require.Equal(t, 2, cfg.LlamaParallel)
	require.Equal(t, 6, cfg.LlamaThreads)
	require.True(t, cfg.LlamaMLock)
}

func TestLoad_envOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("models_dir: /from/yaml\n"), 0o644))

	t.Setenv("MARS_HARNESS_MODELS_DIR", "/from/env")
	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "/from/env", cfg.ModelsDir)
}

func TestLoad_invalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{bad yaml"), 0o644))

	_, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse")
}

func TestLoad_envPortOverride(t *testing.T) {
	t.Setenv("MARS_HARNESS_WEBHOOK_PORT", "8080")
	cfg, err := Load("/nonexistent/config.yaml")
	require.NoError(t, err)
	require.Equal(t, 8080, cfg.WebhookPort)
}

func TestLoad_envInferenceOverrides(t *testing.T) {
	t.Setenv("MARS_HARNESS_PERFORMANCE_PROFILE", "speed")
	t.Setenv("MARS_HARNESS_LLAMA_PARALLEL", "3")
	t.Setenv("MARS_HARNESS_LLAMA_THREADS_BATCH", "4")
	t.Setenv("MARS_HARNESS_LLAMA_FLASH_ATTENTION", "on")
	t.Setenv("MARS_HARNESS_LLAMA_MLOCK", "true")

	cfg, err := Load("/nonexistent/config.yaml")
	require.NoError(t, err)
	require.Equal(t, "speed", cfg.PerformanceProfile)
	require.Equal(t, 3, cfg.LlamaParallel)
	require.Equal(t, 4, cfg.LlamaThreadsBatch)
	require.Equal(t, "on", cfg.LlamaFlashAttention)
	require.True(t, cfg.LlamaMLock)
}
