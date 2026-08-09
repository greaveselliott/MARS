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
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/greaveselliott/mars/internal/network"
)

const (
	defaultListenAddr    = "127.0.0.1:9999"
	defaultSetupTimeout  = 5 * time.Minute
	credentialsFileName  = "github-app.json"
	credentialsDirName   = ".mars"
	manifestCallbackPath = "/callback"
	manifestSetupPath    = "/setup"
	setupStateBytes      = 32
	maxSetupBodyBytes    = 1
	maxConversionBytes   = 1 << 20
	maxManifestCodeBytes = 256
	manifestExchangeWait = 30 * time.Second
)

var setupPageTemplate = template.Must(template.New("github-setup").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>MARS GitHub App setup</title></head>
<body><main><h1>Register the MARS GitHub App</h1>
<form method="post" action="{{.Action}}"><input type="hidden" name="manifest" value="{{.Manifest}}">
<button type="submit">Register MARS GitHub App</button></form></main></body></html>`))

// SetupConfig controls the App manifest flow.
type SetupConfig struct {
	ListenAddr string
	Timeout    time.Duration
	GitHubURL  string // default: https://github.com (override for GHE)
}

func (c SetupConfig) withDefaults() SetupConfig {
	if c.ListenAddr == "" {
		c.ListenAddr = defaultListenAddr
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultSetupTimeout
	}
	if c.GitHubURL == "" {
		c.GitHubURL = "https://github.com"
	}
	c.GitHubURL = strings.TrimRight(c.GitHubURL, "/")
	return c
}

// appManifest returns the JSON manifest for the GitHub App.
func appManifest(callbackURL string) map[string]any {
	return map[string]any{
		"name":            "mars",
		"url":             "https://github.com/greaveselliott/MARS",
		"hook_attributes": map[string]any{"url": callbackURL, "active": true},
		"redirect_url":    callbackURL,
		"callback_urls":   []string{callbackURL},
		"setup_url":       callbackURL,
		"public":          false,
		"default_permissions": map[string]string{
			"checks":   "write",
			"contents": "read",
			"issues":   "write",
			"metadata": "read",
			"statuses": "write",
		},
		"default_events": []string{
			"push",
			"check_suite",
			"workflow_run",
			"merge_group",
		},
	}
}

// manifestConversionResponse is the subset of fields returned by POST /app-manifests/{code}/conversions.
type manifestConversionResponse struct {
	ID            int64  `json:"id"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	PEM           string `json:"pem"`
	WebhookSecret string `json:"webhook_secret"`
}

type setupDependencies struct {
	random   io.Reader
	listen   func(string, string) (net.Listener, error)
	exchange func(context.Context, string, string) (*AppCredentials, error)
	persist  func(*AppCredentials) (*AppCredentials, error)
	output   io.Writer
}

func defaultSetupDependencies() setupDependencies {
	return setupDependencies{
		random:   rand.Reader,
		listen:   net.Listen,
		exchange: exchangeManifestCode,
		persist:  persistSetupCredentials,
		output:   os.Stdout,
	}
}

type setupOutcome struct {
	credentials *AppCredentials
	err         error
}

type setupAdmission struct {
	mu       sync.Mutex
	state    string
	consumed bool
	terminal bool
}

func (a *setupAdmission) claim(candidate string) (matched, unavailable bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal || a.consumed {
		return false, true
	}
	if len(candidate) != len(a.state) || subtle.ConstantTimeCompare([]byte(candidate), []byte(a.state)) != 1 {
		return false, false
	}
	a.consumed = true
	return true, false
}

func (a *setupAdmission) finish() {
	a.mu.Lock()
	a.terminal = true
	a.state = ""
	a.mu.Unlock()
}

// RunSetup performs the GitHub App manifest flow through a single-use,
// literal-loopback callback. Only non-secret App identity is returned.
func RunSetup(ctx context.Context, cfg SetupConfig) (*AppCredentials, error) {
	return runSetup(ctx, cfg, defaultSetupDependencies())
}

func runSetup(ctx context.Context, cfg SetupConfig, deps setupDependencies) (*AppCredentials, error) {
	cfg = cfg.withDefaults()
	if err := network.ValidateLiteralLoopbackAddress("GitHub App setup listener", cfg.ListenAddr); err != nil {
		return nil, err
	}

	stateBytes := make([]byte, setupStateBytes)
	if _, err := io.ReadFull(deps.random, stateBytes); err != nil {
		return nil, errors.New("github setup: could not generate one-time callback state; retry on a healthy operating system")
	}
	state := hex.EncodeToString(stateBytes)

	listener, err := deps.listen("tcp", cfg.ListenAddr)
	if err != nil {
		return nil, errors.New("github setup: could not start the loopback callback listener; ensure the configured port is available and retry")
	}
	defer listener.Close()

	flowCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	outcomes := make(chan setupOutcome, 1)
	admission := &setupAdmission{state: state}
	handler, err := newSetupHandler(flowCtx, cfg, admission, outcomes, deps)
	if err != nil {
		admission.finish()
		return nil, err
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		MaxHeaderBytes:    64 << 10,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case serverErrors <- errors.New("github setup: the loopback callback server stopped; restart setup and try again"):
			default:
			}
		}
	}()

	setupURL := "http://" + cfg.ListenAddr + manifestSetupPath
	slog.Info("github setup: manifest flow is ready on loopback")
	if deps.output != nil {
		_, _ = fmt.Fprintf(deps.output, "\n  Open this URL to register your GitHub App:\n  %s\n\n", setupURL)
	}

	var outcome setupOutcome
	select {
	case outcome = <-outcomes:
	case err := <-serverErrors:
		admission.finish()
		shutdownSrv(srv)
		return nil, err
	case <-flowCtx.Done():
		admission.finish()
		shutdownSrv(srv)
		return nil, errors.New("github setup: timed out waiting for the one-time callback; restart setup and try again")
	}

	admission.finish()
	shutdownSrv(srv)
	if outcome.err != nil {
		return nil, outcome.err
	}
	return outcome.credentials, nil
}

func newSetupHandler(ctx context.Context, cfg SetupConfig, admission *setupAdmission, outcomes chan<- setupOutcome, deps setupDependencies) (http.Handler, error) {
	callbackURL := "http://" + cfg.ListenAddr + manifestCallbackPath
	manifestJSON, err := json.Marshal(appManifest(callbackURL))
	if err != nil {
		return nil, errors.New("github setup: could not prepare the GitHub App manifest; retry setup")
	}
	action := cfg.GitHubURL + "/settings/apps/new?state=" + url.QueryEscape(admission.state)
	githubOrigin, err := url.Parse(cfg.GitHubURL)
	if err != nil || githubOrigin.Scheme != "https" || githubOrigin.Host == "" || githubOrigin.User != nil || githubOrigin.RawQuery != "" || githubOrigin.Fragment != "" {
		return nil, errors.New("github setup: GitHub URL must be an HTTPS origin")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSetupSecurityHeaders(w, githubOrigin.Scheme+"://"+githubOrigin.Host)
		if r.Host != cfg.ListenAddr {
			http.Error(w, "GitHub App setup rejected the request.", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "GitHub App setup requires GET.", http.StatusMethodNotAllowed)
			return
		}
		if !emptySetupBody(w, r) {
			http.Error(w, "GitHub App setup requires an empty request body.", http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case manifestSetupPath:
			if r.URL.RawQuery != "" {
				http.Error(w, "GitHub App setup rejected the request.", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := setupPageTemplate.Execute(w, struct {
				Action   string
				Manifest string
			}{Action: action, Manifest: string(manifestJSON)}); err != nil {
				http.Error(w, "GitHub App setup could not render the registration page.", http.StatusInternalServerError)
			}
		case manifestCallbackPath:
			handleManifestCallback(ctx, w, r, cfg, admission, outcomes, deps)
		default:
			http.Error(w, "GitHub App setup route not found.", http.StatusNotFound)
		}
	}), nil
}

func handleManifestCallback(ctx context.Context, w http.ResponseWriter, r *http.Request, cfg SetupConfig, admission *setupAdmission, outcomes chan<- setupOutcome, deps setupDependencies) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(query) != 2 || len(query["code"]) != 1 || !validManifestCode(query.Get("code")) || len(query["state"]) != 1 || query.Get("state") == "" {
		http.Error(w, "GitHub App setup callback is invalid.", http.StatusBadRequest)
		return
	}
	matched, unavailable := admission.claim(query.Get("state"))
	if unavailable {
		http.Error(w, "GitHub App setup callback is no longer available.", http.StatusConflict)
		return
	}
	if !matched {
		http.Error(w, "GitHub App setup callback state is invalid.", http.StatusForbidden)
		return
	}

	exchangeCtx, cancel := context.WithTimeout(ctx, manifestExchangeWait)
	defer cancel()
	creds, err := deps.exchange(exchangeCtx, cfg.GitHubURL, query.Get("code"))
	if err != nil {
		admission.finish()
		outcome := setupOutcome{err: errors.New("github setup: GitHub did not complete the one-time manifest exchange; restart setup and try again")}
		outcomes <- outcome
		http.Error(w, "GitHub App setup failed. Close this window and retry.", http.StatusBadGateway)
		return
	}
	identity, err := deps.persist(creds)
	if err != nil {
		admission.finish()
		outcome := setupOutcome{err: errors.New("github setup: could not save GitHub App credentials; check owner-only access to the MARS config directory and retry")}
		outcomes <- outcome
		http.Error(w, "GitHub App setup failed. Close this window and retry.", http.StatusInternalServerError)
		return
	}
	identity = sanitizedAppIdentity(identity)
	admission.finish()
	outcomes <- setupOutcome{credentials: identity}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, "<!doctype html><html><body><h1>MARS GitHub App registered.</h1><p>You can close this window.</p></body></html>")
}

func validManifestCode(code string) bool {
	if len(code) == 0 || len(code) > maxManifestCodeBytes {
		return false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~' {
			continue
		}
		return false
	}
	return true
}

func emptySetupBody(w http.ResponseWriter, r *http.Request) bool {
	if r.ContentLength > 0 {
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSetupBodyBytes)
	body, err := io.ReadAll(r.Body)
	return err == nil && len(body) == 0
}

func setSetupSecurityHeaders(w http.ResponseWriter, githubOrigin string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action "+githubOrigin+"; frame-ancestors 'none'")
	w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}

func persistSetupCredentials(creds *AppCredentials) (*AppCredentials, error) {
	if creds == nil {
		return nil, fmt.Errorf("github setup: credentials are missing")
	}
	if err := writeCredentials(creds); err != nil {
		return sanitizedAppIdentity(creds), fmt.Errorf("github setup: save credentials: %w; no secret credential was returned", err)
	}
	return sanitizedAppIdentity(creds), nil
}

func sanitizedAppIdentity(creds *AppCredentials) *AppCredentials {
	if creds == nil {
		return nil
	}
	return &AppCredentials{AppID: creds.AppID, ClientID: creds.ClientID}
}

// exchangeManifestCode POSTs to /app-manifests/{code}/conversions and returns credentials.
func exchangeManifestCode(ctx context.Context, githubURL, code string) (*AppCredentials, error) {
	conversionURL, err := manifestConversionURL(githubURL, code)
	if err != nil {
		return nil, errors.New("github manifest exchange: invalid GitHub origin")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, conversionURL, nil)
	if err != nil {
		return nil, errors.New("github manifest exchange: could not prepare the request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{
		Timeout: manifestExchangeWait,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("github manifest exchange: request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New("github manifest exchange: GitHub rejected the one-time code")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxConversionBytes+1))
	if err != nil || len(body) > maxConversionBytes {
		return nil, errors.New("github manifest exchange: response was invalid")
	}
	var conv manifestConversionResponse
	if err := json.Unmarshal(body, &conv); err != nil || conv.ID <= 0 || conv.ClientID == "" || conv.ClientSecret == "" || conv.PEM == "" || conv.WebhookSecret == "" {
		return nil, errors.New("github manifest exchange: response was incomplete")
	}

	return &AppCredentials{
		AppID:         conv.ID,
		ClientID:      conv.ClientID,
		ClientSecret:  conv.ClientSecret,
		PEM:           conv.PEM,
		WebhookSecret: conv.WebhookSecret,
	}, nil
}

func manifestConversionURL(githubURL, code string) (string, error) {
	base, err := url.Parse(strings.TrimRight(githubURL, "/"))
	if err != nil || base.Scheme != "https" && base.Scheme != "http" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return "", errors.New("invalid GitHub origin")
	}
	if strings.EqualFold(base.Hostname(), "github.com") {
		base.Scheme = "https"
		base.Host = "api.github.com"
		base.Path = ""
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/app-manifests/" + url.PathEscape(code) + "/conversions"
	return base.String(), nil
}

func credentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, credentialsDirName, credentialsFileName)
}

func writeCredentials(creds *AppCredentials) error {
	dir := filepath.Dir(credentialsPath())
	if info, err := os.Lstat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("secure config directory %s: path must be a real directory, not a symlink or other file", dir)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create config directory %s: %w", dir, err)
		}
	} else {
		return fmt.Errorf("inspect config directory %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	var file *os.File
	var tempPath string
	for attempt := 0; attempt < 10; attempt++ {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return fmt.Errorf("create credentials nonce: %w", err)
		}
		tempPath = filepath.Join(dir, "."+credentialsFileName+".tmp-"+hex.EncodeToString(nonce[:]))
		file, err = os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return fmt.Errorf("create owner-only credentials temp file: %w", err)
		}
	}
	if file == nil {
		return fmt.Errorf("create owner-only credentials temp file: exhausted collision retries")
	}
	defer os.Remove(tempPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure credentials temp file: %w", err)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		_ = file.Close()
		return fmt.Errorf("secure credentials temp file: expected regular 0600 file")
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write credentials temp file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync credentials temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close credentials temp file: %w", err)
	}
	if err := os.Rename(tempPath, credentialsPath()); err != nil {
		return fmt.Errorf("atomically replace %s: %w", credentialsPath(), err)
	}
	return nil
}

// ResolveWebhookSecret gives the environment value strict precedence and uses
// the owner-only GitHub App credentials file only when the environment is
// absent. An absent credentials file keeps optional webhook ingress disabled.
func ResolveWebhookSecret(environmentValue string) (string, error) {
	if environmentValue != "" {
		return environmentValue, nil
	}
	return loadPersistedWebhookSecret()
}

func loadPersistedWebhookSecret() (string, error) {
	return loadPersistedWebhookSecretFile(credentialsPath(), os.Open)
}

func loadPersistedWebhookSecretFile(path string, openFile func(string) (*os.File, error)) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("github webhook fallback: inspect %s: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return "", fmt.Errorf("github webhook fallback: %s must be a regular owner-only 0600 file; run chmod 600 %s or rerun GitHub App setup", path, path)
	}
	file, err := openFile(path)
	if err != nil {
		return "", fmt.Errorf("github webhook fallback: open %s: %w", path, err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("github webhook fallback: inspect opened %s: %w", path, err)
	}
	if !openedInfo.Mode().IsRegular() || openedInfo.Mode().Perm() != 0o600 || !os.SameFile(info, openedInfo) {
		return "", fmt.Errorf("github webhook fallback: %s changed during open or is not the same regular owner-only 0600 file", path)
	}
	const maxCredentialsBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxCredentialsBytes+1))
	if err != nil {
		return "", fmt.Errorf("github webhook fallback: read %s: %w", path, err)
	}
	if len(data) > maxCredentialsBytes {
		return "", fmt.Errorf("github webhook fallback: %s exceeds the 1 MiB credentials limit", path)
	}
	var creds AppCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", fmt.Errorf("github webhook fallback: parse %s: %w", path, err)
	}
	return creds.WebhookSecret, nil
}

func shutdownSrv(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
