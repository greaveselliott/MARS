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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultListenAddr    = ":9999"
	defaultSetupTimeout  = 5 * time.Minute
	credentialsFileName  = "github-app.json"
	credentialsDirName   = ".mars"
	manifestCallbackPath = "/callback"
	manifestSetupPath    = "/setup"
)

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

// RunSetup performs the GitHub App manifest flow. It starts a temporary local
// HTTP server, prints a URL for the user to visit, and waits for the callback.
// Returns AppCredentials on success or a timeout error.
func RunSetup(ctx context.Context, cfg SetupConfig) (*AppCredentials, error) {
	cfg = cfg.withDefaults()

	resultCh := make(chan *AppCredentials, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()

	callbackURL := fmt.Sprintf("http://localhost%s%s", cfg.ListenAddr, manifestCallbackPath)
	manifest := appManifest(callbackURL)
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("github setup: marshal manifest: %w", err)
	}

	mux.HandleFunc(manifestSetupPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := fmt.Sprintf(`<!DOCTYPE html>
<html><body>
<form method="post" action="%s/settings/apps/new">
<input type="hidden" name="manifest" value='%s'>
<button type="submit" style="font-size:1.2em;padding:12px 24px;cursor:pointer">
Register MARS GitHub App
</button>
</form>
</body></html>`, cfg.GitHubURL, strings.ReplaceAll(string(manifestJSON), "'", "&#39;"))
		fmt.Fprint(w, page)
	})

	mux.HandleFunc(manifestCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing 'code' parameter — the GitHub callback must include a manifest exchange code", http.StatusBadRequest)
			return
		}

		creds, err := exchangeManifestCode(ctx, cfg.GitHubURL, code)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
			errCh <- err
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><h2>MARS GitHub App registered successfully!</h2><p>You can close this window.</p></body></html>`)
		resultCh <- creds
	})

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	go func() {
		slog.Info("github setup: starting manifest flow server", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("github setup: server failed: %w", err)
		}
	}()

	setupURL := fmt.Sprintf("http://localhost%s%s", cfg.ListenAddr, manifestSetupPath)
	slog.Info("github setup: visit this URL to register your GitHub App", "url", setupURL)
	fmt.Printf("\n  Open this URL to register your GitHub App:\n  %s\n\n", setupURL)

	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	var creds *AppCredentials
	select {
	case creds = <-resultCh:
	case err := <-errCh:
		shutdownSrv(srv)
		return nil, err
	case <-timeoutCtx.Done():
		shutdownSrv(srv)
		return nil, fmt.Errorf("github setup: timed out after %s waiting for manifest callback — restart and try again", cfg.Timeout)
	}

	shutdownSrv(srv)

	creds, err = persistSetupCredentials(creds)
	if err != nil {
		slog.Warn("github setup: credentials obtained but failed to write to disk", "error", err)
		return creds, err
	}
	slog.Info("github setup: credentials saved", "path", credentialsPath())
	return creds, nil
}

func persistSetupCredentials(creds *AppCredentials) (*AppCredentials, error) {
	if creds == nil {
		return nil, fmt.Errorf("github setup: credentials are missing")
	}
	if err := writeCredentials(creds); err != nil {
		creds.WebhookSecret = ""
		return creds, fmt.Errorf("github setup: save credentials: %w; the webhook secret was not returned", err)
	}
	creds.WebhookSecret = ""
	return creds, nil
}

// exchangeManifestCode POSTs to /app-manifests/{code}/conversions and returns credentials.
func exchangeManifestCode(ctx context.Context, githubURL, code string) (*AppCredentials, error) {
	apiBase := strings.Replace(githubURL, "https://github.com", "https://api.github.com", 1)
	url := fmt.Sprintf("%s/app-manifests/%s/conversions", apiBase, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build conversion request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return nil, fmt.Errorf("conversion returned %s: %s — verify the code is valid and hasn't been used", resp.Status, strings.TrimSpace(string(b)))
	}

	var conv manifestConversionResponse
	if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
		return nil, fmt.Errorf("decode conversion response: %w", err)
	}

	return &AppCredentials{
		AppID:         conv.ID,
		ClientID:      conv.ClientID,
		ClientSecret:  conv.ClientSecret,
		PEM:           conv.PEM,
		WebhookSecret: conv.WebhookSecret,
	}, nil
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
