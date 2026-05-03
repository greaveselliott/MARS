package models

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetAndResolveTierModelOverride(t *testing.T) {
	repo := harnessRepo(t)

	path, err := SetModelOverride(repo, "fast", "", ModelOverride{
		Provider: ProviderOllama,
		Model:    "qwen3.6:27b",
		Reason:   "benchmark candidate",
	})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repo, modelOverridesPath), path)

	override, ok, err := ResolveModelOverride(repo, "qa", "fast")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, ProviderOllama, override.Provider)
	require.Equal(t, "qwen3.6:27b", override.Model)
	require.Equal(t, DefaultOllamaEndpoint, override.Endpoint)
	require.Equal(t, "benchmark candidate", override.Reason)
}

func TestRoleOverrideTakesPrecedence(t *testing.T) {
	repo := harnessRepo(t)
	_, err := SetModelOverride(repo, "coding", "", ModelOverride{
		Provider: ProviderOllama,
		Model:    "laguna-xs.2",
	})
	require.NoError(t, err)
	_, err = SetModelOverride(repo, "", "engineer", ModelOverride{
		Provider: ProviderOpenAICompatible,
		Model:    "remote-coder",
		Endpoint: "http://models.test/v1",
	})
	require.NoError(t, err)

	override, ok, err := ResolveModelOverride(repo, "engineer", "coding")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, ProviderOpenAICompatible, override.Provider)
	require.Equal(t, "remote-coder", override.Model)
	require.Equal(t, "http://models.test/v1", override.Endpoint)
}

func TestSetModelOverrideValidatesScopeAndEndpoint(t *testing.T) {
	repo := harnessRepo(t)

	_, err := SetModelOverride(repo, "fast", "qa", ModelOverride{Provider: ProviderOllama, Model: "qwen"})
	require.ErrorContains(t, err, "exactly one")

	_, err = SetModelOverride(repo, "unknown", "", ModelOverride{Provider: ProviderOllama, Model: "qwen"})
	require.ErrorContains(t, err, "unsupported tier")

	_, err = SetModelOverride(repo, "fast", "", ModelOverride{Provider: ProviderOpenAICompatible, Model: "remote"})
	require.ErrorContains(t, err, "--endpoint is required")
}

func harnessRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	return repo
}
