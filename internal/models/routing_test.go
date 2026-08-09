/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/llm"
	"github.com/stretchr/testify/require"
)

func TestResolveLocalBundleAutoSelectsBestEligible(t *testing.T) {
	hw := hardware.Summary{
		Profile: hardware.ProfileHigh,
		RAMMiB:  64 * 1024,
		OS:      "darwin",
		Arch:    "arm64",
		GPUs:    []hardware.GPU{{Name: "Apple Silicon", Driver: "Metal", VRAMMiB: 64 * 1024}},
	}

	bundle, report, err := ResolveLocalBundle(hw, LocalBundleAuto)
	require.NoError(t, err)
	require.Equal(t, LocalBundleBalanced, bundle.ID)
	require.Equal(t, LocalBundleBalanced, report.SelectedBundle)
}

func TestResolveLocalBundleRejectsUnsupportedExplicitBundle(t *testing.T) {
	hw := hardware.Summary{
		Profile: hardware.ProfileLow,
		RAMMiB:  16 * 1024,
		OS:      "linux",
		Arch:    "amd64",
		GPUs:    []hardware.GPU{{Name: "NVIDIA", Driver: "CUDA", VRAMMiB: 8 * 1024}},
	}

	_, _, err := ResolveLocalBundle(hw, LocalBundleQuality)
	require.ErrorContains(t, err, "not eligible")
	require.ErrorContains(t, err, "dedicated VRAM")
}

func TestResolveProviderRouteReadsLocalEnvWithoutSerializingSecret(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", ".env.local"), []byte("ANTHROPIC_API_KEY=secret-value\n"), 0o600))

	route, err := ResolveProviderRoute(repo, ProviderRoute{
		Routing:   RoutingCloud,
		Provider:  ProviderAnthropic,
		Model:     "claude-test",
		APIKeyEnv: "ANTHROPIC_API_KEY",
	})
	require.NoError(t, err)
	require.Equal(t, "secret-value", route.APIKey)
	require.NotEmpty(t, route.Endpoint)
	require.Equal(t, "ANTHROPIC_API_KEY", route.APIKeyEnv)
}

func TestWriteLocalCredentialWritesEnvLocalAndExample(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	t.Setenv("ANTHROPIC_API_KEY", "secret-value")

	localPath, examplePath, err := WriteLocalCredential(repo, "ANTHROPIC_API_KEY")
	require.NoError(t, err)
	require.FileExists(t, localPath)
	require.FileExists(t, examplePath)

	info, err := os.Stat(localPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	localData, err := os.ReadFile(localPath)
	require.NoError(t, err)
	require.Contains(t, string(localData), "ANTHROPIC_API_KEY=secret-value")
	exampleData, err := os.ReadFile(examplePath)
	require.NoError(t, err)
	require.Contains(t, string(exampleData), "ANTHROPIC_API_KEY=")
	require.NotContains(t, string(exampleData), "secret-value")
	exampleInfo, err := os.Stat(examplePath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), exampleInfo.Mode().Perm())
}

func TestWriteLocalCredentialRejectsSymlinksWithoutReturningSecretOrMutatingOutside(t *testing.T) {
	const secret = "credential-must-not-escape"

	t.Run("harness parent", func(t *testing.T) {
		repo := t.TempDir()
		outsideDir := t.TempDir()
		sentinel := filepath.Join(outsideDir, ".env.local")
		require.NoError(t, os.WriteFile(sentinel, []byte("ORIGINAL=unchanged\n"), 0o600))
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(repo, ".harness")))
		t.Setenv("MARS_TEST_API_KEY", secret)

		localPath, examplePath, err := WriteLocalCredential(repo, "MARS_TEST_API_KEY")
		require.Error(t, err)
		require.Empty(t, localPath)
		require.Empty(t, examplePath)
		require.NotContains(t, err.Error(), secret)
		require.NotContains(t, err.Error(), outsideDir)
		data, readErr := os.ReadFile(sentinel)
		require.NoError(t, readErr)
		require.Equal(t, "ORIGINAL=unchanged\n", string(data))
	})

	t.Run("credential leaf", func(t *testing.T) {
		repo := harnessRepo(t)
		sentinel := filepath.Join(t.TempDir(), "outside.env")
		require.NoError(t, os.WriteFile(sentinel, []byte("ORIGINAL=unchanged\n"), 0o600))
		require.NoError(t, os.Symlink(sentinel, filepath.Join(repo, ".harness", ".env.local")))
		t.Setenv("MARS_TEST_API_KEY", secret)

		localPath, examplePath, err := WriteLocalCredential(repo, "MARS_TEST_API_KEY")
		require.Error(t, err)
		require.Empty(t, localPath)
		require.Empty(t, examplePath)
		require.NotContains(t, err.Error(), secret)
		require.NotContains(t, err.Error(), sentinel)
		data, readErr := os.ReadFile(sentinel)
		require.NoError(t, readErr)
		require.Equal(t, "ORIGINAL=unchanged\n", string(data))
	})
}

func TestWriteLocalCredentialTightensModePreservesLocalKeysAndScrubsExampleValues(t *testing.T) {
	repo := harnessRepo(t)
	localPath := filepath.Join(repo, ".harness", ".env.local")
	examplePath := filepath.Join(repo, ".harness", ".env.example")
	require.NoError(t, os.WriteFile(localPath, []byte("OTHER_TOKEN=keep-me\n"), 0o644))
	require.NoError(t, os.Chmod(localPath, 0o644))
	require.NoError(t, os.WriteFile(examplePath, []byte("OTHER_TOKEN=remove-me\nOLD=also-remove\n"), 0o600))
	require.NoError(t, os.Chmod(examplePath, 0o600))
	t.Setenv("ANTHROPIC_API_KEY", "new-secret")

	gotLocal, gotExample, err := WriteLocalCredential(repo, "ANTHROPIC_API_KEY")
	require.NoError(t, err)
	localInfo, err := os.Stat(gotLocal)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), localInfo.Mode().Perm())
	localData, err := os.ReadFile(gotLocal)
	require.NoError(t, err)
	require.Contains(t, string(localData), "OTHER_TOKEN=keep-me")
	require.Contains(t, string(localData), "ANTHROPIC_API_KEY=new-secret")

	exampleInfo, err := os.Stat(gotExample)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), exampleInfo.Mode().Perm())
	exampleData, err := os.ReadFile(gotExample)
	require.NoError(t, err)
	require.Equal(t, "ANTHROPIC_API_KEY=\nOLD=\nOTHER_TOKEN=\n", string(exampleData))
}

func TestEnsureEnvExampleRejectsSymlinkLeafWithoutOutsideMutation(t *testing.T) {
	repo := harnessRepo(t)
	sentinel := filepath.Join(t.TempDir(), "outside.env")
	original := []byte("EXISTING=outside-value\n")
	require.NoError(t, os.WriteFile(sentinel, original, 0o600))
	require.NoError(t, os.Symlink(sentinel, filepath.Join(repo, ".harness", ".env.example")))

	path, err := EnsureEnvExample(repo, "NEW_API_KEY")
	require.Error(t, err)
	require.Empty(t, path)
	require.NotContains(t, err.Error(), sentinel)
	require.NotContains(t, err.Error(), "outside-value")
	data, readErr := os.ReadFile(sentinel)
	require.NoError(t, readErr)
	require.Equal(t, original, data)
}

func TestLookupCredentialReportsLocalParseFailure(t *testing.T) {
	repo := harnessRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", ".env.local"), []byte("not-an-env-assignment\n"), 0o600))
	t.Setenv("MARS_TEST_API_KEY", "")

	value, found, err := LookupCredential(repo, "MARS_TEST_API_KEY")
	require.ErrorContains(t, err, "parse .harness/.env.local")
	require.Empty(t, value)
	require.False(t, found)

	_, err = ResolveProviderRoute(repo, ProviderRoute{
		Routing:   RoutingCloud,
		Provider:  ProviderOpenAI,
		Model:     "model-under-test",
		APIKeyEnv: "MARS_TEST_API_KEY",
	})
	require.ErrorContains(t, err, "read credential env MARS_TEST_API_KEY")
	require.NotContains(t, err.Error(), "is not set")
}

func TestResolveProviderRouteMissingCredentialUsesPathRedactedRemediation(t *testing.T) {
	repo := harnessRepo(t)
	t.Setenv("MARS_TEST_API_KEY", "")

	_, err := ResolveProviderRoute(repo, ProviderRoute{
		Routing:   RoutingCloud,
		Provider:  ProviderOpenAI,
		Model:     "model-under-test",
		APIKeyEnv: "MARS_TEST_API_KEY",
	})
	require.ErrorContains(t, err, "--repo <path>")
	require.NotContains(t, err.Error(), repo)
	require.NotContains(t, err.Error(), "--repo .")
}

func TestSelectableProvidersHaveOfficialDocsAndAdapterEvidence(t *testing.T) {
	for _, spec := range ProviderSpecs() {
		if !spec.Selectable {
			require.NotEmpty(t, spec.UnavailableReason, "unselectable provider %s should explain why", spec.Name)
			continue
		}
		require.NotEmpty(t, spec.OfficialDocs, "selectable provider %s needs official doc evidence", spec.Name)
		require.Contains(t, []string{"openai_chat", "anthropic_messages"}, spec.Adapter, "selectable provider %s needs request-capture adapter coverage", spec.Name)
	}
}

func TestSelectableOpenAICompatibleProvidersUseChatCompletionsShape(t *testing.T) {
	for _, spec := range ProviderSpecs() {
		if !spec.Selectable || spec.Adapter != "openai_chat" {
			continue
		}
		t.Run(spec.Name, func(t *testing.T) {
			var gotPath string
			var gotAuth string
			var gotBody llm.ChatCompletionRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &gotBody))
				_ = json.NewEncoder(w).Encode(llm.ChatCompletionResponse{
					Choices: []llm.Choice{{Message: llm.Message{Role: "assistant", Content: "ok"}}},
				})
			}))
			defer srv.Close()

			repo := t.TempDir()
			apiKeyEnv := spec.DefaultAPIKeyEnv
			if apiKeyEnv == "" && spec.Name != ProviderOllama {
				apiKeyEnv = "TEST_PROVIDER_API_KEY"
			}
			if apiKeyEnv != "" {
				t.Setenv(apiKeyEnv, "provider-secret")
			}

			endpoint := srv.URL
			if spec.Name == ProviderGemini {
				endpoint += "/v1beta/openai"
			}
			route, err := ResolveProviderRoute(repo, ProviderRoute{
				Routing:   RoutingCloud,
				Provider:  spec.Name,
				Model:     "model-under-test",
				Endpoint:  endpoint,
				APIKeyEnv: apiKeyEnv,
			})
			require.NoError(t, err)

			client, err := llm.NewClient(llm.Config{
				BaseURL:    route.Endpoint,
				Provider:   route.Provider,
				APIKey:     route.APIKey,
				Model:      route.Model,
				HTTPClient: srv.Client(),
				MaxRetries: 1,
			})
			require.NoError(t, err)
			_, err = client.ChatCompletion(context.Background(), llm.ChatCompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
				Tools: []llm.ToolDefinition{{
					Type:     "function",
					Function: llm.FunctionSpec{Name: "file_read", Parameters: map[string]any{"type": "object"}},
				}},
			})
			require.NoError(t, err)
			require.True(t, strings.HasSuffix(gotPath, "/chat/completions"), "unexpected request path %s", gotPath)
			if spec.Name == ProviderGemini {
				require.Equal(t, "/v1beta/openai/chat/completions", gotPath)
			} else {
				require.Equal(t, "/v1/chat/completions", gotPath)
			}
			require.Equal(t, "model-under-test", gotBody.Model)
			require.Len(t, gotBody.Tools, 1)
			if spec.Name == ProviderOllama {
				require.Empty(t, gotAuth)
			} else {
				require.Equal(t, "Bearer provider-secret", gotAuth)
			}
		})
	}
}
