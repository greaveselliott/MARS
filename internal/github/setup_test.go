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
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
		ListenAddr: "127.0.0.1:8888",
		Timeout:    2 * time.Minute,
		GitHubURL:  "https://ghe.corp.com/",
	}.withDefaults()
	require.Equal(t, "127.0.0.1:8888", cfg.ListenAddr)
	require.Equal(t, 2*time.Minute, cfg.Timeout)
	require.Equal(t, "https://ghe.corp.com", cfg.GitHubURL)
}

func TestAppManifest_structure(t *testing.T) {
	t.Parallel()
	m := appManifest("http://127.0.0.1:9999/callback")

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
		require.Equal(t, "/app-manifests/test-code-123/conversions", r.URL.EscapedPath())

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
		http.Error(w, `{"message":"provider-secret-body"}`, http.StatusNotFound)
	}))
	t.Cleanup(convSrv.Close)

	_, err := exchangeManifestCode(context.Background(), convSrv.URL, "secret-code")
	require.Error(t, err)
	require.EqualError(t, err, "github manifest exchange: GitHub rejected the one-time code")
	require.NotContains(t, err.Error(), "provider-secret-body")
	require.NotContains(t, err.Error(), "secret-code")
	require.NotContains(t, err.Error(), convSrv.URL)
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

func TestRunSetupRejectsBindAndRandomFailuresBeforeListening(t *testing.T) {
	t.Parallel()
	var randomReads atomic.Int32
	var listens atomic.Int32
	deps := setupDependencies{
		random: readerFunc(func([]byte) (int, error) {
			randomReads.Add(1)
			return 0, errors.New("rng-secret")
		}),
		listen: func(string, string) (net.Listener, error) {
			listens.Add(1)
			return nil, errors.New("listen-secret")
		},
		output: io.Discard,
	}
	_, err := runSetup(context.Background(), SetupConfig{ListenAddr: "0.0.0.0:9999"}, deps)
	require.EqualError(t, err, "GitHub App setup listener must use a literal loopback IP and TCP port such as 127.0.0.1:9092 or [::1]:9092")
	require.Zero(t, randomReads.Load())
	require.Zero(t, listens.Load())

	_, err = runSetup(context.Background(), SetupConfig{ListenAddr: "127.0.0.1:9999"}, deps)
	require.EqualError(t, err, "github setup: could not generate one-time callback state; retry on a healthy operating system")
	require.EqualValues(t, 1, randomReads.Load())
	require.Zero(t, listens.Load())
}

func TestSetupHandlerServesOfficialStateProtocolWithSecurityHeaders(t *testing.T) {
	handler, _, _, state := setupHandlerForTest(t, setupDependencies{})
	recorder := serveSetupRequest(handler, http.MethodGet, "http://127.0.0.1:9999/setup", "127.0.0.1:9999", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, `method="post"`)
	require.Contains(t, body, `action="https://github.com/settings/apps/new?state=`+state+`"`)
	require.Contains(t, body, `name="manifest"`)
	require.Contains(t, body, "http://127.0.0.1:9999/callback")
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Contains(t, recorder.Header().Get("Content-Security-Policy"), "form-action https://github.com")
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.NotContains(t, body, "client_secret")
	require.NotContains(t, body, "webhook_secret")
}

func TestSetupHandlerRejectsMethodHostBodyAndInvalidCallbackWithoutConsumingState(t *testing.T) {
	var exchanges atomic.Int32
	deps := setupDependencies{
		exchange: func(context.Context, string, string) (*AppCredentials, error) {
			exchanges.Add(1)
			return &AppCredentials{AppID: 1, ClientID: "client", ClientSecret: "secret", PEM: "pem", WebhookSecret: "webhook"}, nil
		},
		persist: func(creds *AppCredentials) (*AppCredentials, error) { return creds, nil },
	}
	handler, outcomes, _, state := setupHandlerForTest(t, deps)
	for name, request := range map[string]struct {
		method string
		url    string
		host   string
		body   string
		status int
	}{
		"method":         {http.MethodPost, "http://127.0.0.1:9999/setup", "127.0.0.1:9999", "", http.StatusMethodNotAllowed},
		"host":           {http.MethodGet, "http://127.0.0.1:9999/setup", "attacker.invalid", "", http.StatusForbidden},
		"body":           {http.MethodGet, "http://127.0.0.1:9999/setup", "127.0.0.1:9999", "x", http.StatusBadRequest},
		"setup query":    {http.MethodGet, "http://127.0.0.1:9999/setup?extra=1", "127.0.0.1:9999", "", http.StatusBadRequest},
		"callback query": {http.MethodGet, "http://127.0.0.1:9999/callback?code=code", "127.0.0.1:9999", "", http.StatusBadRequest},
		"unsafe code":    {http.MethodGet, "http://127.0.0.1:9999/callback?code=bad%2Fcode&state=" + state, "127.0.0.1:9999", "", http.StatusBadRequest},
		"wrong state":    {http.MethodGet, "http://127.0.0.1:9999/callback?code=code&state=wrong", "127.0.0.1:9999", "", http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := serveSetupRequest(handler, request.method, request.url, request.host, request.body)
			require.Equal(t, request.status, recorder.Code)
		})
	}
	require.Zero(t, exchanges.Load())

	recorder := serveSetupRequest(handler, http.MethodGet, "http://127.0.0.1:9999/callback?code=code&state="+state, "127.0.0.1:9999", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.EqualValues(t, 1, exchanges.Load())
	require.NotNil(t, (<-outcomes).credentials)
}

func TestSetupHandlerAllowsOneConcurrentCallbackAndRejectsReplay(t *testing.T) {
	var exchanges atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	deps := setupDependencies{
		exchange: func(context.Context, string, string) (*AppCredentials, error) {
			exchanges.Add(1)
			close(started)
			<-release
			return &AppCredentials{AppID: 1, ClientID: "client", ClientSecret: "secret", PEM: "pem", WebhookSecret: "webhook"}, nil
		},
		persist: func(creds *AppCredentials) (*AppCredentials, error) { return creds, nil },
	}
	handler, outcomes, _, state := setupHandlerForTest(t, deps)
	callback := "http://127.0.0.1:9999/callback?code=one-time-code&state=" + state
	first := make(chan *httptest.ResponseRecorder, 1)
	go func() { first <- serveSetupRequest(handler, http.MethodGet, callback, "127.0.0.1:9999", "") }()
	<-started

	const competitors = 8
	statuses := make(chan int, competitors)
	var wg sync.WaitGroup
	for i := 0; i < competitors; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- serveSetupRequest(handler, http.MethodGet, callback, "127.0.0.1:9999", "").Code
		}()
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		require.Equal(t, http.StatusConflict, status)
	}
	close(release)
	require.Equal(t, http.StatusOK, (<-first).Code)
	require.NotNil(t, (<-outcomes).credentials)
	require.EqualValues(t, 1, exchanges.Load())
	require.Equal(t, http.StatusConflict, serveSetupRequest(handler, http.MethodGet, callback, "127.0.0.1:9999", "").Code)
	require.EqualValues(t, 1, exchanges.Load())
}

func TestSetupHandlerExchangeFailureAndTimeoutDoNotPersistOrExposeSecrets(t *testing.T) {
	for name, exchange := range map[string]func(context.Context, string, string) (*AppCredentials, error){
		"failure": func(context.Context, string, string) (*AppCredentials, error) {
			return nil, errors.New("provider-body code-secret client-secret pem-secret webhook-secret")
		},
		"timeout": func(ctx context.Context, _ string, _ string) (*AppCredentials, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	} {
		t.Run(name, func(t *testing.T) {
			var persists atomic.Int32
			deps := setupDependencies{
				exchange: exchange,
				persist: func(*AppCredentials) (*AppCredentials, error) {
					persists.Add(1)
					return nil, nil
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			handler, outcomes, _, state := setupHandlerForContext(t, ctx, deps)
			recorder := serveSetupRequest(handler, http.MethodGet, "http://127.0.0.1:9999/callback?code=code-secret&state="+state, "127.0.0.1:9999", "")
			require.Equal(t, http.StatusBadGateway, recorder.Code)
			require.Equal(t, "GitHub App setup failed. Close this window and retry.\n", recorder.Body.String())
			outcome := <-outcomes
			require.EqualError(t, outcome.err, "github setup: GitHub did not complete the one-time manifest exchange; restart setup and try again")
			require.Zero(t, persists.Load())
		})
	}
}

func TestRunSetupPersistsFullCredentialsAndReturnsOnlyIdentity(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	stateBytes := bytes.Repeat([]byte{0xab}, setupStateBytes)
	state := hex.EncodeToString(stateBytes)
	full := &AppCredentials{AppID: 42, ClientID: "client-id", ClientSecret: "client-secret", PEM: "pem-secret", WebhookSecret: "webhook-secret"}
	var exchanges atomic.Int32
	var persists atomic.Int32
	listenArgs := make(chan [2]string, 1)
	exchangeArgs := make(chan [2]string, 1)
	persisted := make(chan *AppCredentials, 1)
	deps := setupDependencies{
		random: bytes.NewReader(stateBytes),
		listen: func(networkName, requested string) (net.Listener, error) {
			listenArgs <- [2]string{networkName, requested}
			return listener, nil
		},
		exchange: func(_ context.Context, githubURL, code string) (*AppCredentials, error) {
			exchanges.Add(1)
			exchangeArgs <- [2]string{githubURL, code}
			return full, nil
		},
		persist: func(creds *AppCredentials) (*AppCredentials, error) {
			persists.Add(1)
			persisted <- creds
			return creds, nil
		},
		output: io.Discard,
	}
	resultCh := make(chan setupOutcome, 1)
	go func() {
		creds, err := runSetup(context.Background(), SetupConfig{ListenAddr: addr, Timeout: time.Second}, deps)
		resultCh <- setupOutcome{credentials: creds, err: err}
	}()
	callback := "http://" + addr + "/callback?code=one-time-code&state=" + state
	var response *http.Response
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		response, err = http.Get(callback)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	outcome := <-resultCh
	require.NoError(t, outcome.err)
	require.Equal(t, &AppCredentials{AppID: 42, ClientID: "client-id"}, outcome.credentials)
	require.Equal(t, [2]string{"tcp", addr}, <-listenArgs)
	require.Equal(t, [2]string{"https://github.com", "one-time-code"}, <-exchangeArgs)
	require.Equal(t, full, <-persisted)
	require.EqualValues(t, 1, exchanges.Load())
	require.EqualValues(t, 1, persists.Load())
}

func TestRunSetupOverallTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deps := setupDependencies{
		random: bytes.NewReader(make([]byte, setupStateBytes)),
		listen: func(string, string) (net.Listener, error) { return listener, nil },
		output: io.Discard,
	}
	_, err = runSetup(context.Background(), SetupConfig{ListenAddr: listener.Addr().String(), Timeout: 20 * time.Millisecond}, deps)
	require.EqualError(t, err, "github setup: timed out waiting for the one-time callback; restart setup and try again")
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func setupHandlerForTest(t *testing.T, deps setupDependencies) (http.Handler, <-chan setupOutcome, *setupAdmission, string) {
	return setupHandlerForContext(t, context.Background(), deps)
}

func setupHandlerForContext(t *testing.T, ctx context.Context, deps setupDependencies) (http.Handler, <-chan setupOutcome, *setupAdmission, string) {
	t.Helper()
	state := strings.Repeat("ab", setupStateBytes)
	admission := &setupAdmission{state: state}
	outcomes := make(chan setupOutcome, 1)
	handler, err := newSetupHandler(ctx, SetupConfig{ListenAddr: "127.0.0.1:9999", Timeout: time.Second, GitHubURL: "https://github.com"}, admission, outcomes, deps)
	require.NoError(t, err)
	return handler, outcomes, admission, state
}

func serveSetupRequest(handler http.Handler, method, target, host, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Host = host
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
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
