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
	"encoding/json"
	"errors"
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
	t.Setenv("MARS_GITHUB_TOKEN", "")
	cfgPath := filepath.Join(dir, ".mars", "config.yaml")
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

func TestCheckUsesAnonymousFirstAndRetriesOnlyExactOfficialDenials(t *testing.T) {
	for _, denied := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		t.Run(http.StatusText(denied), func(t *testing.T) {
			requestCount := 0
			resolverCalls := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					require.Empty(t, req.Header.Get("Authorization"))
					return githubAuthResponse(denied, "private response body"), nil
				}
				require.Equal(t, ReleaseAPIURL(""), req.URL.String())
				require.Equal(t, "Bearer optional-token", req.Header.Get("Authorization"))
				return githubAuthResponse(http.StatusOK, "authenticated response body"), nil
			})}

			report := Check(context.Background(), Options{
				Env: func(string) string { return "" },
				GHAuthToken: func(context.Context) (string, error) {
					resolverCalls++
					return "optional-token", nil
				},
				HTTPClient: client,
			})

			require.Equal(t, StatusOK, report.Status)
			require.Equal(t, AccessAuthenticated, report.AccessClass)
			require.Equal(t, SourceGHCLI, report.AuthSource)
			require.Equal(t, 2, requestCount)
			require.Equal(t, 1, resolverCalls)
			require.NotContains(t, report.Message, "optional-token")
			require.NotContains(t, report.Message, "response body")
		})
	}
}

func TestCheckAnonymousSuccessNeverResolvesCredentials(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, ReleaseAPIURL(""), req.URL.String())
		require.Empty(t, req.Header.Get("Authorization"))
		return githubAuthResponse(http.StatusOK, "ignored response body"), nil
	})}
	report := Check(context.Background(), Options{
		Env: func(string) string { return "env-token" },
		GHAuthToken: func(context.Context) (string, error) {
			t.Fatal("anonymous success resolved GitHub CLI credentials")
			return "", nil
		},
		HTTPClient: client,
	})
	require.Equal(t, AccessAnonymous, report.AccessClass)
	require.Equal(t, SourceNone, report.AuthSource)
}

func TestCheckNeverResolvesCredentialsForTransportRedirectUnexpectedOrCustomOrigin(t *testing.T) {
	tests := []struct {
		name       string
		releaseURL string
		response   func(*http.Request) (*http.Response, error)
	}{
		{
			name: "transport failure",
			response: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("transport detail optional-token private-path")
			},
		},
		{
			name: "redirect",
			response: func(*http.Request) (*http.Response, error) {
				resp := githubAuthResponse(http.StatusFound, "redirect body optional-token")
				resp.Header.Set("Location", "https://credentials.example.test/private")
				return resp, nil
			},
		},
		{
			name: "unexpected status",
			response: func(*http.Request) (*http.Response, error) {
				return githubAuthResponse(http.StatusTooManyRequests, "rate limit body optional-token"), nil
			},
		},
		{
			name:       "custom origin denial",
			releaseURL: "https://metadata.example.test/private/releases/latest",
			response: func(*http.Request) (*http.Response, error) {
				return githubAuthResponse(http.StatusNotFound, "custom private body optional-token"), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			resolverCalls := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requestCount++
				require.Empty(t, req.Header.Get("Authorization"))
				return tt.response(req)
			})}
			report := Check(context.Background(), Options{
				ReleaseURL: tt.releaseURL,
				Env:        func(string) string { return "optional-token" },
				GHAuthToken: func(context.Context) (string, error) {
					resolverCalls++
					return "optional-token", nil
				},
				HTTPClient: client,
			})
			require.Equal(t, AccessUnavailable, report.AccessClass)
			require.Equal(t, SourceNone, report.AuthSource)
			require.Equal(t, 1, requestCount)
			require.Zero(t, resolverCalls)
			encoded, err := json.Marshal(report)
			require.NoError(t, err)
			for _, forbidden := range []string{"optional-token", "private-path", "response body", "redirect body", "rate limit body", "custom private body", tt.releaseURL} {
				if forbidden != "" {
					require.NotContains(t, string(encoded), forbidden)
				}
			}
		})
	}
}

func TestAuthenticatedSetupDistinguishesAuthFailures(t *testing.T) {
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
			report := checkToken(context.Background(), Options{
				HTTPClient: client,
				ReleaseURL: "https://api.example.test/repos/private/project/releases/latest",
			}, Token{Value: "token", Source: SourceConfig})
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
	t.Setenv("MARS_GITHUB_TOKEN", "")
	report := Check(context.Background(), Options{
		Env:          func(string) string { return "" },
		DisableGHCLI: true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return githubAuthResponse(http.StatusNotFound, "private response"), nil
		})},
	})

	require.Equal(t, StatusFail, report.Status)
	require.Equal(t, AccessUnavailable, report.AccessClass)
	require.Equal(t, SourceNone, report.AuthSource)
	require.Contains(t, report.NextAction, "mars auth github setup")
	require.NotContains(t, report.Message, "GH_TOKEN=")
}

func githubAuthResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestSetupPersistsGHCLIFallbackAfterSuccessfulCheck(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mars", "config.yaml")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer gh-cli-token", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})}

	report, err := Setup(context.Background(), Options{
		Env:         func(string) string { return "" },
		ConfigPath:  cfgPath,
		GHAuthToken: func(context.Context) (string, error) { return "gh-cli-token", nil },
		HTTPClient:  client,
		ReleaseURL:  "https://api.example.test/repos/private/project/releases/latest",
	})

	require.NoError(t, err)
	require.Equal(t, StatusOK, report.Status)
	require.Equal(t, SourceGHCLI, report.AuthSource)
	require.Contains(t, report.Message, "saved a local fallback")
	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "github_token: gh-cli-token")
	info, err := os.Stat(cfgPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSetupDoesNotPersistEnvTokenImplicitly(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mars", "config.yaml")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer env-token", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})}

	report, err := Setup(context.Background(), Options{
		Env: func(key string) string {
			if key == "GH_TOKEN" {
				return "env-token"
			}
			return ""
		},
		ConfigPath: cfgPath,
		HTTPClient: client,
		ReleaseURL: "https://api.example.test/repos/private/project/releases/latest",
	})

	require.NoError(t, err)
	require.Equal(t, StatusOK, report.Status)
	require.Equal(t, SourceEnvGHToken, report.AuthSource)
	_, err = os.Stat(cfgPath)
	require.True(t, os.IsNotExist(err))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
