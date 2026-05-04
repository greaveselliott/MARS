/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-011-optional-github-integration.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL      = "https://api.github.com"
	defaultMaxRetries   = 3
	tokenRefreshBuffer  = 60 * time.Second
	jwtSkew             = 60 * time.Second
	jwtLifetime         = 10 * time.Minute
	rateLimitSleepFloor = 1 * time.Second
)

// ClientConfig holds GitHub API client configuration.
type ClientConfig struct {
	Mode    AuthMode
	BaseURL string // default: https://api.github.com

	// App mode fields
	AppID      int64
	PrivateKey *rsa.PrivateKey
	InstallID  int64

	// PAT mode field
	Token string

	HTTPClient *http.Client
	MaxRetries int
}

// Client provides GitHub REST API operations with automatic token management.
type Client struct {
	cfg      ClientConfig
	http     *http.Client
	baseURL  string
	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// NewClient creates a GitHub API client. It validates the config and
// returns an actionable error when required fields are missing.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	switch cfg.Mode {
	case AuthApp:
		if cfg.AppID == 0 {
			return nil, fmt.Errorf("github: AppID is required for app auth mode — set ClientConfig.AppID")
		}
		if cfg.PrivateKey == nil {
			return nil, fmt.Errorf("github: PrivateKey is required for app auth mode — provide the App's PEM private key")
		}
		if cfg.InstallID == 0 {
			return nil, fmt.Errorf("github: InstallID is required for app auth mode — find it in your GitHub App installation settings")
		}
	case AuthPAT:
		if cfg.Token == "" {
			return nil, fmt.Errorf("github: Token is required for PAT auth mode — generate one at github.com/settings/tokens")
		}
	default:
		return nil, fmt.Errorf("github: unknown auth mode %q — use AuthApp or AuthPAT", cfg.Mode)
	}

	if cfg.MaxRetries < 1 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	c := &Client{
		cfg:     cfg,
		http:    cfg.HTTPClient,
		baseURL: cfg.BaseURL,
	}

	if cfg.Mode == AuthPAT {
		c.token = cfg.Token
		c.tokenExp = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	return c, nil
}

// CreatePR creates a pull request and returns the created resource.
func (c *Client) CreatePR(ctx context.Context, owner, repo, title, body, head, base string) (*PullRequest, error) {
	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
	var pr PullRequest
	if err := c.doJSON(ctx, http.MethodPost, path, payload, &pr); err != nil {
		return nil, fmt.Errorf("github: create PR on %s/%s: %w", owner, repo, err)
	}
	return &pr, nil
}

// CreateCheckRun creates a check run on the given commit SHA.
func (c *Client) CreateCheckRun(ctx context.Context, owner, repo, name, sha string, status CheckStatus) (*CheckRun, error) {
	payload := map[string]string{
		"name":     name,
		"head_sha": sha,
		"status":   string(status),
	}
	path := fmt.Sprintf("/repos/%s/%s/check-runs", owner, repo)
	var cr CheckRun
	if err := c.doJSON(ctx, http.MethodPost, path, payload, &cr); err != nil {
		return nil, fmt.Errorf("github: create check run on %s/%s@%s: %w", owner, repo, sha[:minInt(7, len(sha))], err)
	}
	return &cr, nil
}

// UpdateCheckRun updates an existing check run's status and optional conclusion.
func (c *Client) UpdateCheckRun(ctx context.Context, owner, repo string, checkID int64, status CheckStatus, conclusion string) error {
	payload := map[string]any{
		"status": string(status),
	}
	if conclusion != "" {
		payload["conclusion"] = conclusion
	}
	path := fmt.Sprintf("/repos/%s/%s/check-runs/%d", owner, repo, checkID)
	if err := c.doJSON(ctx, http.MethodPatch, path, payload, nil); err != nil {
		return fmt.Errorf("github: update check run %d on %s/%s: %w", checkID, owner, repo, err)
	}
	return nil
}

// PostComment posts a comment on a PR or issue.
func (c *Client) PostComment(ctx context.Context, owner, repo string, number int, body string) error {
	payload := map[string]string{"body": body}
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
	if err := c.doJSON(ctx, http.MethodPost, path, payload, nil); err != nil {
		return fmt.Errorf("github: post comment on %s/%s#%d: %w", owner, repo, number, err)
	}
	return nil
}

// doJSON performs an authenticated JSON request with retry and rate-limit handling.
func (c *Client) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	url := c.baseURL + path
	var lastErr error

	for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
		tok, err := c.resolveToken(ctx)
		if err != nil {
			return fmt.Errorf("resolve auth token: %w", err)
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("build HTTP request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+tok)
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request %s %s (attempt %d/%d): %w", method, path, attempt+1, c.cfg.MaxRetries, err)
			if attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			sleepCtx(ctx, backoff(attempt))
			continue
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			wait := rateLimitWait(resp)
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("rate limited (%s) on %s %s: %s", resp.Status, method, path, strings.TrimSpace(string(b)))
			slog.Warn("github: rate limited, backing off",
				"status", resp.StatusCode,
				"wait", wait,
				"path", path,
				"attempt", attempt+1,
			)
			if attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			sleepCtx(ctx, wait)
			continue
		}

		if resp.StatusCode >= 500 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("server error (%s) on %s %s: %s", resp.Status, method, path, strings.TrimSpace(string(b)))
			if attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			sleepCtx(ctx, backoff(attempt))
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
			_ = resp.Body.Close()
			return fmt.Errorf("unexpected status %s on %s %s: %s", resp.Status, method, path, strings.TrimSpace(string(b)))
		}

		if out != nil {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				_ = resp.Body.Close()
				return fmt.Errorf("decode response from %s %s: %w", method, path, err)
			}
		}
		_ = resp.Body.Close()
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("exhausted %d retries for %s %s", c.cfg.MaxRetries, method, path)
}

// resolveToken returns a valid bearer token, refreshing the installation token if needed.
func (c *Client) resolveToken(ctx context.Context) (string, error) {
	if c.cfg.Mode == AuthPAT {
		return c.cfg.Token, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExp.Add(-tokenRefreshBuffer)) {
		return c.token, nil
	}

	jwt, err := c.generateJWT()
	if err != nil {
		return "", fmt.Errorf("generate JWT: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", c.baseURL, c.cfg.InstallID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange JWT for installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return "", fmt.Errorf("installation token request failed (%s): %s — verify AppID (%d) and InstallID (%d) are correct",
			resp.Status, strings.TrimSpace(string(b)), c.cfg.AppID, c.cfg.InstallID)
	}

	var tok installationToken
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("decode installation token: %w", err)
	}

	c.token = tok.Token
	c.tokenExp = tok.ExpiresAt
	slog.Info("github: refreshed installation token", "expires_at", tok.ExpiresAt.Format(time.RFC3339))
	return c.token, nil
}

// generateJWT builds a RS256-signed JWT for GitHub App authentication.
// Claims: iss=appID, iat=now-60s, exp=now+10m.
func (c *Client) generateJWT() (string, error) {
	now := time.Now()
	header := base64URLEncode([]byte(`{"alg":"RS256","typ":"JWT"}`))

	claims := fmt.Sprintf(`{"iss":"%d","iat":%d,"exp":%d}`,
		c.cfg.AppID,
		now.Add(-jwtSkew).Unix(),
		now.Add(jwtLifetime).Unix(),
	)
	payload := base64URLEncode([]byte(claims))
	signingInput := header + "." + payload

	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.cfg.PrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("RSA sign JWT: %w", err)
	}

	return signingInput + "." + base64URLEncode(sig), nil
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func rateLimitWait(resp *http.Response) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if sec, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(strings.TrimSpace(reset), 10, 64); err == nil {
			wait := time.Until(time.Unix(ts, 0))
			if wait > 0 {
				return wait
			}
		}
	}
	return rateLimitSleepFloor
}

func backoff(attempt int) time.Duration {
	ms := 200
	for i := 0; i < attempt && ms < 5000; i++ {
		ms *= 2
	}
	if ms > 5000 {
		ms = 5000
	}
	return time.Duration(ms) * time.Millisecond
}

func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
