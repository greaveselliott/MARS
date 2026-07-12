/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/github-app-integration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

	require.Equal(t, "mars", m["name"])
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
	require.Contains(t, events, "merge_group")
	require.NotContains(t, events, "issue_comment")
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

func TestPersistSetupCredentialsStoresSecretOwnerOnlyAndDoesNotReturnIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	secret := "0123456789abcdef0123456789abcdef"
	returned, err := persistSetupCredentials(&AppCredentials{AppID: 1, ClientID: "client", WebhookSecret: secret})
	require.NoError(t, err)
	require.Empty(t, returned.WebhookSecret)
	info, err := os.Stat(credentialsPath())
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	loaded, err := loadPersistedWebhookSecret()
	require.NoError(t, err)
	require.Equal(t, secret, loaded)
}

func TestPersistSetupCredentialsWriteFailureClearsReturnedSecret(t *testing.T) {
	homeFile := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(homeFile, []byte("x"), 0o600))
	t.Setenv("HOME", homeFile)
	secret := "0123456789abcdef0123456789abcdef"
	returned, err := persistSetupCredentials(&AppCredentials{WebhookSecret: secret})
	require.Error(t, err)
	require.NotNil(t, returned)
	require.Empty(t, returned.WebhookSecret)
	require.NotContains(t, err.Error(), secret)
}

func TestResolveWebhookSecretEnvironmentOverridesFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Dir(credentialsPath()), 0o700))
	require.NoError(t, os.WriteFile(credentialsPath(), []byte(`{"webhook_secret":"fallback-secret"}`), 0o644))
	secret, err := ResolveWebhookSecret("environment-secret")
	require.NoError(t, err)
	require.Equal(t, "environment-secret", secret)
}

func TestLoadPersistedWebhookSecretRejectsUnsafeModeMalformedAndOversizedFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		mode os.FileMode
		body []byte
	}{
		"unsafe mode": {0o644, []byte(`{"webhook_secret":"secret"}`)},
		"malformed":   {0o600, []byte(`{`)},
		"oversized":   {0o600, bytes.Repeat([]byte("x"), (1<<20)+1)},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			require.NoError(t, os.MkdirAll(filepath.Dir(credentialsPath()), 0o700))
			require.NoError(t, os.WriteFile(credentialsPath(), tc.body, tc.mode))
			_, err := loadPersistedWebhookSecret()
			require.Error(t, err)
		})
	}
}

func TestLoadPersistedWebhookSecretMissingIsDisabledNotError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	secret, err := loadPersistedWebhookSecret()
	require.NoError(t, err)
	require.Empty(t, secret)
}

func TestPersistSetupCredentialsAtomicallyReplacesDestinationSymlinkWithoutFollowingTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Dir(credentialsPath()), 0o700))
	target := filepath.Join(t.TempDir(), "symlink-target.json")
	require.NoError(t, os.WriteFile(target, []byte("sentinel-target"), 0o600))
	require.NoError(t, os.Symlink(target, credentialsPath()))
	secret := "new-owner-only-webhook-secret-1234"
	returned, err := persistSetupCredentials(&AppCredentials{WebhookSecret: secret})
	require.NoError(t, err)
	require.Empty(t, returned.WebhookSecret)
	targetBody, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "sentinel-target", string(targetBody), "atomic rename must replace the symlink entry, not follow its target")
	info, err := os.Lstat(credentialsPath())
	require.NoError(t, err)
	require.True(t, info.Mode().IsRegular())
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	loaded, err := loadPersistedWebhookSecret()
	require.NoError(t, err)
	require.Equal(t, secret, loaded)
}

func TestPersistSetupCredentialsRejectsSymlinkParent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	realDir := filepath.Join(t.TempDir(), "real-mars")
	require.NoError(t, os.MkdirAll(realDir, 0o700))
	require.NoError(t, os.Symlink(realDir, filepath.Join(home, credentialsDirName)))
	returned, err := persistSetupCredentials(&AppCredentials{WebhookSecret: "must-not-be-written"})
	require.Error(t, err)
	require.Empty(t, returned.WebhookSecret)
	require.NoFileExists(t, filepath.Join(realDir, credentialsFileName))
}

func TestPersistSetupCredentialsCleansTempWhenDestinationIsDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(credentialsPath(), 0o700))
	returned, err := persistSetupCredentials(&AppCredentials{WebhookSecret: "must-not-be-returned"})
	require.Error(t, err)
	require.Empty(t, returned.WebhookSecret)
	matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(credentialsPath()), "."+credentialsFileName+".tmp-*"))
	require.NoError(t, globErr)
	require.Empty(t, matches)
}

func TestLoadPersistedWebhookSecretRejectsOpenSwapBeforeRead(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Dir(credentialsPath()), 0o700))
	require.NoError(t, os.WriteFile(credentialsPath(), []byte(`{"webhook_secret":"original"}`), 0o600))
	replacement := filepath.Join(t.TempDir(), "replacement.json")
	require.NoError(t, os.WriteFile(replacement, []byte(`{"webhook_secret":"must-not-leak"}`), 0o600))
	secret, err := loadPersistedWebhookSecretFile(credentialsPath(), func(path string) (*os.File, error) {
		require.NoError(t, os.Remove(path))
		require.NoError(t, os.Symlink(replacement, path))
		return os.Open(path)
	})
	require.Error(t, err)
	require.Empty(t, secret)
	require.NotContains(t, err.Error(), "must-not-leak")
}

func TestLoadPersistedWebhookSecretRejectsNonRegularDestination(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Dir(credentialsPath()), 0o700))
	target := filepath.Join(t.TempDir(), "target.json")
	require.NoError(t, os.WriteFile(target, []byte(`{"webhook_secret":"must-not-load"}`), 0o600))
	require.NoError(t, os.Symlink(target, credentialsPath()))
	secret, err := loadPersistedWebhookSecret()
	require.Error(t, err)
	require.Empty(t, secret)
}
