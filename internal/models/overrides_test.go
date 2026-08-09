/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
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
	canonicalRepo, err := filepath.EvalSymlinks(repo)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(canonicalRepo, modelOverridesPath), path)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())

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

func TestModelOverrideRejectsSymlinkLeafWithoutOutsideMutation(t *testing.T) {
	repo := harnessRepo(t)
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	original := []byte("version: 2\ntiers: {}\n")
	require.NoError(t, os.WriteFile(outside, original, 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(repo, modelOverridesPath)))

	resolved, ok, err := ResolveModelOverride(repo, "qa", "fast")
	require.Error(t, err)
	require.False(t, ok)
	require.Empty(t, resolved)
	require.NotContains(t, err.Error(), outside)

	path, err := SetModelOverride(repo, "fast", "", ModelOverride{Provider: ProviderOllama, Model: "qwen"})
	require.Error(t, err)
	require.Empty(t, path)
	require.NotContains(t, err.Error(), outside)
	data, readErr := os.ReadFile(outside)
	require.NoError(t, readErr)
	require.Equal(t, original, data)
}

func TestSetModelOverridePreservesExistingMode(t *testing.T) {
	repo := harnessRepo(t)
	path := filepath.Join(repo, modelOverridesPath)
	require.NoError(t, os.WriteFile(path, []byte("version: 2\n"), 0o600))
	require.NoError(t, os.Chmod(path, 0o600))

	_, err := SetModelOverride(repo, "fast", "", ModelOverride{Provider: ProviderOllama, Model: "qwen"})
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func harnessRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	return repo
}
