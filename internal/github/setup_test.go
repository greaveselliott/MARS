/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-011-optional-github-integration.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetupConfig_withDefaults(t *testing.T) {
	t.Parallel()
	cfg := SetupConfig{}.withDefaults()
	require.Equal(t, defaultListenAddr, cfg.ListenAddr)
	require.Equal(t, defaultSetupTimeout, cfg.Timeout)
	require.Equal(t, "https://github.com", cfg.GitHubURL)
}

func TestSetupConfig_preservesValues(t *testing.T) {
	t.Parallel()
	cfg := SetupConfig{
		ListenAddr: ":8888",
		Timeout:    2 * time.Minute,
		GitHubURL:  "https://ghe.corp.com/",
	}.withDefaults()
	require.Equal(t, ":8888", cfg.ListenAddr)
	require.Equal(t, 2*time.Minute, cfg.Timeout)
	require.Equal(t, "https://ghe.corp.com", cfg.GitHubURL)
}

func TestAppManifest_structure(t *testing.T) {
	t.Parallel()
	m := appManifest("http://localhost:9999/callback")

	require.Equal(t, "mars-harness", m["name"])
	require.Equal(t, false, m["public"])

	perms, ok := m["default_permissions"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "write", perms["checks"])
	require.Equal(t, "read", perms["contents"])
	require.Equal(t, "write", perms["issues"])
	require.Equal(t, "read", perms["metadata"])
	require.Equal(t, "write", perms["statuses"])
	require.NotContains(t, perms, "pull_requests")

	events, ok := m["default_events"].([]string)
	require.True(t, ok)
	require.Contains(t, events, "push")
	require.Contains(t, events, "check_suite")
	require.Contains(t, events, "workflow_run")
	require.Contains(t, events, "issue_comment")
	require.NotContains(t, events, "pull_request")
}

func TestExchangeManifestCode_happyPath(t *testing.T) {
	t.Parallel()

	convSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Contains(t, r.URL.Path, "/app-manifests/test-code-123/conversions")

		w.WriteHeader(http.StatusCreated)
		pem := "-----BEGIN " + "RSA PRIVATE KEY-----\nfake\n-----END " + "RSA PRIVATE KEY-----"
		_ = json.NewEncoder(w).Encode(manifestConversionResponse{
			ID:            42,
			ClientID:      "Iv1.abc123",
			ClientSecret:  "secret456",
			PEM:           pem,
			WebhookSecret: "whsec_789",
		})
	}))
	t.Cleanup(convSrv.Close)

	creds, err := exchangeManifestCode(context.Background(), convSrv.URL, "test-code-123")
	require.NoError(t, err)
	require.Equal(t, int64(42), creds.AppID)
	require.Equal(t, "Iv1.abc123", creds.ClientID)
	require.Equal(t, "secret456", creds.ClientSecret)
	require.Contains(t, creds.PEM, "RSA PRIVATE KEY")
	require.Equal(t, "whsec_789", creds.WebhookSecret)
}

func TestExchangeManifestCode_invalidCode(t *testing.T) {
	t.Parallel()

	convSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	t.Cleanup(convSrv.Close)

	_, err := exchangeManifestCode(context.Background(), convSrv.URL, "bad-code")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Not Found")
}

func TestExchangeManifestCode_contextCancelled(t *testing.T) {
	t.Parallel()

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	t.Cleanup(slowSrv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := exchangeManifestCode(ctx, slowSrv.URL, "code")
	require.Error(t, err)
}

func TestRunSetup_timeout(t *testing.T) {
	t.Parallel()

	_, err := RunSetup(context.Background(), SetupConfig{
		ListenAddr: ":0",
		Timeout:    100 * time.Millisecond,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
}

func TestRunSetup_setupPageServed(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	callbackURL := "http://localhost:9999/callback"
	manifest := appManifest(callbackURL)
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	mux.HandleFunc(manifestSetupPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<form"))
		_, _ = w.Write(manifestJSON)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+manifestSetupPath, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func TestRunSetup_callbackMissingCode(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc(manifestCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing 'code' parameter", http.StatusBadRequest)
			return
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + manifestCallbackPath)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
