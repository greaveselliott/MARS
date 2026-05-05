/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/product-specs/product-surface.md
*/
package githubauth

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveTokenPrefersEnvThenGHCLIThenConfig(t *testing.T) {
	ctx := context.Background()

	got := ResolveToken(ctx, Options{
		Env: func(key string) string {
			if key == "GH_TOKEN" {
				return "gh-token"
			}
			if key == "GITHUB_TOKEN" {
				return "github-token"
			}
			return ""
		},
		GHAuthToken: func(context.Context) (string, error) { return "gh-cli-token", nil },
		ConfigToken: "config-token",
	})
	require.Equal(t, SourceEnvGHToken, got.Token.Source)
	require.Equal(t, "gh-token", got.Token.Value)

	got = ResolveToken(ctx, Options{
		Env: func(key string) string {
			if key == "GITHUB_TOKEN" {
				return "github-token"
			}
			return ""
		},
		GHAuthToken: func(context.Context) (string, error) { return "gh-cli-token", nil },
		ConfigToken: "config-token",
	})
	require.Equal(t, SourceEnvGitHubToken, got.Token.Source)

	got = ResolveToken(ctx, Options{
		Env:         func(string) string { return "" },
		GHAuthToken: func(context.Context) (string, error) { return "gh-cli-token", nil },
		ConfigToken: "config-token",
	})
	require.Equal(t, SourceGHCLI, got.Token.Source)

	got = ResolveToken(ctx, Options{
		Env:          func(string) string { return "" },
		DisableGHCLI: true,
		ConfigToken:  "config-token",
	})
	require.Equal(t, SourceConfig, got.Token.Source)
}

func TestResolveTokenLoadsConfiguredLocalToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("MARS_HARNESS_GITHUB_TOKEN", "")
	cfgPath := filepath.Join(dir, ".mars-harness", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o755))
	require.NoError(t, os.WriteFile(cfgPath, []byte("github_token: local-token\n"), 0o600))

	got := ResolveToken(context.Background(), Options{
		Env:          func(string) string { return "" },
		DisableGHCLI: true,
		ConfigPath:   cfgPath,
	})
	require.Equal(t, SourceConfig, got.Token.Source)
	require.Equal(t, "local-token", got.Token.Value)
}

func TestCheckDistinguishesAuthFailures(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		header      http.Header
		wantStatus  string
		wantMessage string
		wantNext    string
	}{
		{name: "ok", statusCode: http.StatusOK, wantStatus: StatusOK, wantMessage: "can read"},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantStatus: StatusFail, wantMessage: "rejected", wantNext: "gh auth login"},
		{name: "forbidden", statusCode: http.StatusForbidden, wantStatus: StatusFail, wantMessage: "forbids", wantNext: "contents read access"},
		{name: "sso", statusCode: http.StatusForbidden, header: http.Header{"X-Github-Sso": []string{"required"}}, wantStatus: StatusFail, wantMessage: "forbids", wantNext: "Authorize SSO"},
		{name: "not_found", statusCode: http.StatusNotFound, wantStatus: StatusFail, wantMessage: "cannot see", wantNext: "with access"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "Bearer token", req.Header.Get("Authorization"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Status:     http.StatusText(tt.statusCode),
					Header:     tt.header,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			})}
			report := Check(context.Background(), Options{
				Env:          func(string) string { return "" },
				DisableGHCLI: true,
				ConfigToken:  "token",
				HTTPClient:   client,
				ReleaseURL:   "https://api.example.test/repos/private/project/releases/latest",
			})
			require.Equal(t, tt.wantStatus, report.Status)
			require.Contains(t, report.Message, tt.wantMessage)
			if tt.wantNext != "" {
				require.Contains(t, report.NextAction, tt.wantNext)
			}
			require.NotContains(t, report.NextAction, "Bearer")
			require.NotContains(t, report.Message, "ghs_")
			require.NotContains(t, report.NextAction, "ghs_")
		})
	}
}

func TestCheckWithoutTokenReturnsSetupGuidance(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MARS_HARNESS_GITHUB_TOKEN", "")
	report := Check(context.Background(), Options{
		Env:          func(string) string { return "" },
		DisableGHCLI: true,
	})

	require.Equal(t, StatusFail, report.Status)
	require.Equal(t, SourceNone, report.AuthSource)
	require.Contains(t, report.NextAction, "mars-harness auth github setup")
	require.NotContains(t, report.Message, "GH_TOKEN=")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
