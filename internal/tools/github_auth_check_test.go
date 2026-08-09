/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/release-versioning.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGithubAuthCheckReportsAnonymousAccessWithoutResolvingSecrets(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-secret-must-not-be-read")
	t.Setenv("GITHUB_TOKEN", "github-secret-must-not-be-read")
	t.Setenv("MARS_GITHUB_TOKEN", "config-secret-must-not-be-read")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	requests := 0
	http.DefaultTransport = githubAuthToolRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		require.Empty(t, req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"name":"public release"}`)),
		}, nil
	})

	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	res, err := handleGithubAuthCheck(context.Background(), root, []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, 1, requests)
	require.Contains(t, res.Output, `"status": "ok"`)
	require.Contains(t, res.Output, `"access_class": "anonymous"`)
	require.Contains(t, res.Output, `"auth_source": "none"`)
	require.NotContains(t, res.Output, "Bearer")
	require.NotContains(t, res.Output, "must-not-be-read")
}

func TestDefaultRegistry_includesGithubAuthCheck(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	require.Contains(t, reg.Names(), "github_auth_check")
}

type githubAuthToolRoundTripFunc func(*http.Request) (*http.Response, error)

func (f githubAuthToolRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
