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
}

func TestLoad_parsesYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("models_dir: /custom/models\nlog_format: json\n"), 0o644))

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "/custom/models", cfg.ModelsDir)
	require.Equal(t, "json", cfg.LogFormat)
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
