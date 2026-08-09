/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-017-open-source-publication.md
*/
package selfupdate

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	require.Equal(t, VersionEqual, CompareVersions("0.6.0", "v0.6.0"))
	require.Equal(t, VersionBehind, CompareVersions("0.6.0", "0.7.0"))
	require.Equal(t, VersionAhead, CompareVersions("0.8.0", "0.7.0"))
	require.Equal(t, VersionUnknown, CompareVersions("dev", "0.7.0"))
}

func TestLatestRelease_readsGitHubStyleTag(t *testing.T) {
	t.Parallel()
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/releases/latest", r.URL.Path)
		return textResponse(http.StatusOK, `{"tag_name":"v0.7.0"}`), nil
	})

	version, err := LatestRelease(context.Background(), client, "https://example.test/releases/latest")
	require.NoError(t, err)
	require.Equal(t, "0.7.0", version)
}

func TestLatestReleaseInfoDoesNotSendCredentialsToCustomOrigin(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-canary")
	endpoint := "https://example.test/releases/latest"
	parsed, err := url.Parse(endpoint)
	require.NoError(t, err)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	jar.SetCookies(parsed, []*http.Cookie{{Name: "session", Value: "cookie-canary"}})
	client := &http.Client{Jar: jar, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("Cookie"))
		deadline, ok := r.Context().Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), 5*time.Second)
		return textResponse(http.StatusOK, `{"tag_name":"v0.7.0"}`), nil
	})}

	_, err = LatestReleaseInfo(context.Background(), client, endpoint)
	require.NoError(t, err)
}

func TestLatestReleaseInfoRejectsRedirects(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-canary")
	requests := 0
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "Bearer gh-token-canary", r.Header.Get("Authorization"))
		response := textResponse(http.StatusFound, "redirect")
		response.Header.Set("Location", "https://example.test/collect")
		return response, nil
	})

	_, err := LatestReleaseInfo(context.Background(), client, DefaultLatestReleaseURL)
	require.ErrorContains(t, err, "returned HTTP 302")
	require.Equal(t, 1, requests)
}

func TestLatestReleaseInfoRejectsUnsafeEndpointsBeforeRequest(t *testing.T) {
	for _, endpoint := range []string{
		"http://example.test/releases/latest",
		"https://user:" + "password@example.test/releases/latest",
		"https:///missing-host",
		"https://example.test/releases/latest#fragment",
	} {
		t.Run(endpoint, func(t *testing.T) {
			requests := 0
			client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
				requests++
				return textResponse(http.StatusOK, `{}`), nil
			})
			_, err := LatestReleaseInfo(context.Background(), client, endpoint)
			require.ErrorContains(t, err, "endpoint must be HTTPS")
			require.NotContains(t, err.Error(), "password")
			require.Equal(t, 0, requests)
		})
	}
}

func TestLatestReleaseInfoBoundsAndRedactsMetadata(t *testing.T) {
	t.Run("oversized", func(t *testing.T) {
		client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
			return textResponse(http.StatusOK, strings.Repeat("x", maxLatestReleaseBytes+1)), nil
		})
		_, err := LatestReleaseInfo(context.Background(), client, "https://example.test/releases/latest")
		require.ErrorContains(t, err, "exceeds")
	})

	t.Run("request error", func(t *testing.T) {
		const canary = "query-secret-canary"
		client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("network failure: " + canary)
		})
		_, err := LatestReleaseInfo(context.Background(), client, "https://example.test/releases/latest?token="+canary)
		require.ErrorContains(t, err, "metadata request failed")
		require.NotContains(t, err.Error(), canary)
	})
}

func TestDefaultRepoFullNameUsesCanonicalMARSRepo(t *testing.T) {
	t.Parallel()
	require.Equal(t, "greaveselliott/MARS", DefaultRepoFullName)
	require.Contains(t, DefaultLatestReleaseURL, "/repos/greaveselliott/MARS/")
}

func TestLatestReleaseInfoReportsPrivateReleaseAuthHint(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-canary")
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer gh-token-canary", r.Header.Get("Authorization"))
		return textResponse(http.StatusUnauthorized, `{"message":"bad credentials"}`), nil
	})

	_, err := LatestReleaseInfo(context.Background(), client, DefaultLatestReleaseURL)
	require.ErrorContains(t, err, "auth github check")
	require.NotContains(t, err.Error(), "gh-token-canary")
}

func TestLatestReleaseInfoExplainsAnonymousOnlyCustomOrigin(t *testing.T) {
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		require.Empty(t, r.Header.Get("Authorization"))
		response := textResponse(http.StatusUnauthorized, `{"message":"bad credentials"}`)
		response.Status = "401 ghp_STATUS_CANARY"
		return response, nil
	})

	_, err := LatestReleaseInfo(context.Background(), client, "https://example.test/releases/latest")
	require.ErrorContains(t, err, "anonymous-only")
	require.NotContains(t, err.Error(), "ghp_STATUS_CANARY")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func fakeHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
