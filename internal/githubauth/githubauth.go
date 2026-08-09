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
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/childenv"
	"github.com/greaveselliott/mars/internal/config"
)

const (
	DefaultRepoFullName = "greaveselliott/MARS"

	StatusOK   = "ok"
	StatusWarn = "warn"
	StatusFail = "fail"

	AccessAnonymous     = "anonymous"
	AccessAuthenticated = "authenticated"
	AccessUnavailable   = "unavailable"

	SourceNone           = "none"
	SourceEnvGHToken     = "env-gh-token"
	SourceEnvGitHubToken = "env-github-token"
	SourceGHCLI          = "gh-cli"
	SourceConfig         = "config"
)

var (
	ErrGHCLIMissing = errors.New("gh not found in PATH")
	ErrGHCLIEmpty   = errors.New("gh auth token returned an empty token")
)

// Token contains a resolved credential and its non-secret source.
type Token struct {
	Value  string `json:"-"`
	Source string `json:"source"`
}

// ResolveResult describes token resolution without exposing the token.
type ResolveResult struct {
	Token      Token  `json:"-"`
	GHCLIError string `json:"gh_cli_error,omitempty"`
	ConfigErr  string `json:"config_error,omitempty"`
}

// Report is the structured readiness result shared by CLI, doctor, setup, and
// the github_auth_check universal tool.
type Report struct {
	Status        string `json:"status"`
	AccessClass   string `json:"access_class"`
	AuthSource    string `json:"auth_source"`
	RepoAccess    string `json:"repo_access"`
	ReleaseAccess string `json:"release_access"`
	Message       string `json:"message"`
	NextAction    string `json:"next_action,omitempty"`
}

// ClearLocalReport describes removal of only MARS's stored GitHub fallback.
type ClearLocalReport struct {
	Status  string `json:"status"`
	Cleared bool   `json:"cleared"`
	Message string `json:"message"`
}

// Options configures optional token resolution and release access checks.
type Options struct {
	Env          func(string) string
	ConfigPath   string
	ConfigToken  string
	DisableGHCLI bool
	GHAuthToken  func(context.Context) (string, error)
	HTTPClient   *http.Client
	RepoFullName string
	ReleaseURL   string
	Timeout      time.Duration
}

// ResolveToken resolves optional GitHub auth in this order: GH_TOKEN,
// GITHUB_TOKEN, GitHub CLI auth, then local config.
func ResolveToken(ctx context.Context, opts Options) ResolveResult {
	env := opts.Env
	if env == nil {
		env = os.Getenv
	}
	if token := strings.TrimSpace(env("GH_TOKEN")); token != "" {
		return ResolveResult{Token: Token{Value: token, Source: SourceEnvGHToken}}
	}
	if token := strings.TrimSpace(env("GITHUB_TOKEN")); token != "" {
		return ResolveResult{Token: Token{Value: token, Source: SourceEnvGitHubToken}}
	}

	var ghErr string
	if !opts.DisableGHCLI {
		fn := opts.GHAuthToken
		if fn == nil {
			fn = DefaultGHAuthToken
		}
		if token, err := fn(ctx); err == nil && strings.TrimSpace(token) != "" {
			return ResolveResult{Token: Token{Value: strings.TrimSpace(token), Source: SourceGHCLI}}
		} else if err != nil {
			ghErr = err.Error()
		}
	}

	configToken := strings.TrimSpace(opts.ConfigToken)
	var cfgErr string
	if configToken == "" {
		path := strings.TrimSpace(opts.ConfigPath)
		if path == "" {
			path = config.DefaultPath()
		}
		cfg, err := config.Load(path)
		if err != nil {
			cfgErr = err.Error()
		} else {
			configToken = strings.TrimSpace(cfg.GitHubToken)
		}
	}
	if configToken != "" {
		return ResolveResult{Token: Token{Value: configToken, Source: SourceConfig}, GHCLIError: ghErr, ConfigErr: cfgErr}
	}

	return ResolveResult{Token: Token{Source: SourceNone}, GHCLIError: ghErr, ConfigErr: cfgErr}
}

// Check probes release metadata anonymously first. Credentials are resolved
// only after an exact denial from the exact official GitHub API endpoint, and
// at most one authenticated retry is sent to that same URL.
func Check(ctx context.Context, opts Options) Report {
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint := strings.TrimSpace(opts.ReleaseURL)
	if endpoint == "" {
		endpoint = ReleaseAPIURL(opts.RepoFullName)
	}
	client := checkHTTPClient(opts)
	statusCode, err := requestReleaseMetadata(ctx, client, endpoint, Token{})
	if err != nil {
		return unavailableReport(StatusWarn, SourceNone,
			"release metadata is unavailable",
			"Check network and TLS connectivity, then rerun `mars auth github check`.")
	}
	if statusCode == http.StatusOK {
		return Report{
			Status:        StatusOK,
			AccessClass:   AccessAnonymous,
			AuthSource:    SourceNone,
			RepoAccess:    "ok",
			ReleaseAccess: "ok",
			Message:       "MARS release metadata is available anonymously",
		}
	}
	if !credentialRetryStatus(statusCode) || !exactOfficialReleaseURL(endpoint, opts.RepoFullName) {
		return unavailableReport(StatusWarn, SourceNone,
			"release metadata is unavailable anonymously",
			"Retry the official release metadata check later; credentials were not resolved or sent.")
	}

	resolved := ResolveToken(ctx, opts)
	if strings.TrimSpace(resolved.Token.Value) == "" {
		return unavailableReport(StatusFail, SourceNone,
			"release metadata is unavailable anonymously and no optional GitHub credential is configured",
			"For a private fork or rate-limited check, run `gh auth login` and then `mars auth github setup`.")
	}
	statusCode, err = requestReleaseMetadata(ctx, client, endpoint, resolved.Token)
	if err != nil {
		return unavailableReport(StatusWarn, resolved.Token.Source,
			"release metadata is unavailable after the optional authenticated retry",
			"Check network and TLS connectivity, then rerun `mars auth github check`.")
	}
	if statusCode == http.StatusOK {
		return Report{
			Status:        StatusOK,
			AccessClass:   AccessAuthenticated,
			AuthSource:    resolved.Token.Source,
			RepoAccess:    "ok",
			ReleaseAccess: "ok",
			Message:       "MARS release metadata is available with optional GitHub authentication",
		}
	}
	return unavailableReport(StatusFail, resolved.Token.Source,
		"release metadata is unavailable after the optional authenticated retry",
		"Refresh the optional GitHub credential or retry when the official release is available.")
}

// ClearLocal removes only the stored github_token fallback from the selected
// MARS config. It does not resolve or mutate environment, GitHub CLI, GitHub
// App, repository, or remote credentials.
func ClearLocal(opts Options) (ClearLocalReport, error) {
	cleared, err := config.ClearStoredGitHubToken(opts.ConfigPath)
	if err != nil {
		return ClearLocalReport{}, errors.New("auth github clear-local: could not update the selected config — ensure it is a regular owner-writable YAML file and retry")
	}
	if !cleared {
		return ClearLocalReport{Status: StatusOK, Message: "no stored GitHub fallback was present"}, nil
	}
	return ClearLocalReport{Status: StatusOK, Cleared: true, Message: "stored GitHub fallback cleared"}, nil
}

func checkHTTPClient(opts Options) *http.Client {
	base := opts.HTTPClient
	if base == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		base = &http.Client{Timeout: timeout}
	}
	client := *base
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

func requestReleaseMetadata(ctx context.Context, client *http.Client, endpoint string, token Token) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mars-github-auth-check")
	if strings.TrimSpace(token.Value) != "" {
		req.Header.Set("Authorization", "Bearer "+token.Value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	return resp.StatusCode, nil
}

func credentialRetryStatus(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden || statusCode == http.StatusNotFound
}

func exactOfficialReleaseURL(endpoint, repoFullName string) bool {
	repo := repoName(repoFullName)
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || !safeGitHubName(parts[0]) || !safeGitHubName(parts[1]) {
		return false
	}
	return endpoint == "https://api.github.com/repos/"+parts[0]+"/"+parts[1]+"/releases/latest"
}

func safeGitHubName(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func unavailableReport(status, authSource, message, nextAction string) Report {
	return Report{
		Status:        status,
		AccessClass:   AccessUnavailable,
		AuthSource:    authSource,
		RepoAccess:    "unavailable",
		ReleaseAccess: "unavailable",
		Message:       message,
		NextAction:    nextAction,
	}
}

// Setup verifies private release access and persists an owner-only local
// fallback when setup resolved auth from GitHub CLI or an explicit config token.
func Setup(ctx context.Context, opts Options) (Report, error) {
	resolved := ResolveToken(ctx, opts)
	report := checkToken(ctx, opts, resolved.Token)
	if report.Status != StatusOK {
		return report, nil
	}

	token := strings.TrimSpace(resolved.Token.Value)
	if token == "" {
		return report, nil
	}
	if resolved.Token.Source != SourceGHCLI && !(resolved.Token.Source == SourceConfig && strings.TrimSpace(opts.ConfigToken) != "") {
		return report, nil
	}

	path := strings.TrimSpace(opts.ConfigPath)
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path)
	if err != nil {
		return report, fmt.Errorf("private release auth setup: load config: %w", err)
	}
	if strings.TrimSpace(cfg.GitHubToken) == token {
		return report, nil
	}
	cfg.GitHubToken = token
	if err := config.Save(path, cfg); err != nil {
		return report, fmt.Errorf("private release auth setup: save local fallback: %w", err)
	}
	if resolved.Token.Source == SourceGHCLI {
		report.Message += "; saved a local fallback from GitHub CLI auth"
	} else {
		report.Message += "; saved local private release auth"
	}
	return report, nil
}

func checkToken(ctx context.Context, opts Options, token Token) Report {
	if strings.TrimSpace(token.Value) == "" {
		return Report{
			Status:        StatusFail,
			AccessClass:   AccessUnavailable,
			AuthSource:    SourceNone,
			RepoAccess:    "unknown",
			ReleaseAccess: "unknown",
			Message:       "private release auth is not configured",
			NextAction:    "Run `gh auth login`, then `mars auth github setup`; for headless installs set GH_TOKEN or GITHUB_TOKEN with repository contents read access.",
		}
	}

	client := opts.HTTPClient
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	url := strings.TrimSpace(opts.ReleaseURL)
	if url == "" {
		url = ReleaseAPIURL(opts.RepoFullName)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Report{
			Status:        StatusWarn,
			AccessClass:   AccessUnavailable,
			AuthSource:    token.Source,
			RepoAccess:    "unknown",
			ReleaseAccess: "unknown",
			Message:       "could not build private release auth request",
			NextAction:    "Check the configured release endpoint and rerun `mars auth github check`.",
		}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mars-github-auth-check")
	req.Header.Set("Authorization", "Bearer "+token.Value)

	resp, err := client.Do(req)
	if err != nil {
		return Report{
			Status:        StatusWarn,
			AccessClass:   AccessUnavailable,
			AuthSource:    token.Source,
			RepoAccess:    "unknown",
			ReleaseAccess: "unknown",
			Message:       fmt.Sprintf("private release auth check could not reach GitHub: %v", err),
			NextAction:    "Check network connectivity, then rerun `mars auth github check`.",
		}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return Report{
			Status:        StatusOK,
			AccessClass:   AccessAuthenticated,
			AuthSource:    token.Source,
			RepoAccess:    "ok",
			ReleaseAccess: "ok",
			Message:       "private release auth can read MARS release metadata",
		}
	case http.StatusUnauthorized:
		return Report{
			Status:        StatusFail,
			AccessClass:   AccessUnavailable,
			AuthSource:    token.Source,
			RepoAccess:    "denied",
			ReleaseAccess: "denied",
			Message:       "GitHub rejected the private release token",
			NextAction:    "Run `gh auth login`, then `mars auth github setup`; headless installs should refresh GH_TOKEN or GITHUB_TOKEN.",
		}
	case http.StatusForbidden:
		next := "Use a token with repository contents read access for " + repoName(opts.RepoFullName) + ", then rerun `mars auth github setup`."
		if strings.TrimSpace(resp.Header.Get("X-GitHub-SSO")) != "" {
			next = "Authorize SSO for this token in GitHub, then rerun `mars auth github setup`."
		}
		return Report{
			Status:        StatusFail,
			AccessClass:   AccessUnavailable,
			AuthSource:    token.Source,
			RepoAccess:    "forbidden",
			ReleaseAccess: "forbidden",
			Message:       "GitHub forbids private release access for the resolved token",
			NextAction:    next,
		}
	case http.StatusNotFound:
		return Report{
			Status:        StatusFail,
			AccessClass:   AccessUnavailable,
			AuthSource:    token.Source,
			RepoAccess:    "not_found",
			ReleaseAccess: "not_found",
			Message:       "the resolved token cannot see MARS private releases",
			NextAction:    "Authenticate GitHub CLI as an account with access to " + repoName(opts.RepoFullName) + ", or set GH_TOKEN/GITHUB_TOKEN for that account.",
		}
	default:
		return Report{
			Status:        StatusWarn,
			AccessClass:   AccessUnavailable,
			AuthSource:    token.Source,
			RepoAccess:    "unknown",
			ReleaseAccess: "unknown",
			Message:       fmt.Sprintf("private release auth check returned %s", resp.Status),
			NextAction:    "Rerun `mars auth github check`; if it persists, verify GitHub API access and release publication.",
		}
	}
}

// Apply adds Authorization when a token is available and returns the auth source.
func Apply(req *http.Request, opts Options) string {
	if req == nil {
		return SourceNone
	}
	resolved := ResolveToken(req.Context(), opts)
	if strings.TrimSpace(resolved.Token.Value) == "" {
		return SourceNone
	}
	req.Header.Set("Authorization", "Bearer "+resolved.Token.Value)
	return resolved.Token.Source
}

// DefaultGHAuthToken returns the token already managed by GitHub CLI.
func DefaultGHAuthToken(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", ErrGHCLIMissing
	}
	cmdCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		cmdCtx, cancel = context.WithTimeout(ctx, 3*time.Second)
	}
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "gh", "auth", "token")
	if err := childenv.Apply(cmd); err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", ErrGHCLIEmpty
	}
	return token, nil
}

func ReleaseAPIURL(repoFullName string) string {
	return "https://api.github.com/repos/" + repoName(repoFullName) + "/releases/latest"
}

func repoName(repoFullName string) string {
	repo := strings.TrimSpace(repoFullName)
	if repo == "" {
		return DefaultRepoFullName
	}
	return repo
}
