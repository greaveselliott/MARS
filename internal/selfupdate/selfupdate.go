/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/githubauth"
	"github.com/greaveselliott/mars-harness/internal/shellpath"
)

const (
	DefaultPackage            = "github.com/greaveselliott/mars-harness/cmd/mars-harness"
	DefaultVersion            = "latest"
	DefaultBinary             = "mars-harness"
	DefaultReleaseDownloadURL = "https://github.com/" + DefaultRepoFullName + "/releases/download"
)

// UpdateMethod describes how update tool obtains the replacement binary.
type UpdateMethod string

const (
	MethodReleaseAssets UpdateMethod = "release-assets"
	MethodSource        UpdateMethod = "source"
)

// Config controls an installed-tool update run.
type Config struct {
	Version          string
	InstallDir       string
	BinaryName       string
	Method           UpdateMethod
	DryRun           bool
	SkipShellPath    bool
	LatestReleaseURL string
	ReleaseBaseURL   string
	HTTPClient       *http.Client
}

// Plan is the resolved update action.
type Plan struct {
	Method             UpdateMethod     `json:"method"`
	Package            string           `json:"package,omitempty"`
	Version            string           `json:"version"`
	ReleaseTag         string           `json:"release_tag,omitempty"`
	InstallDir         string           `json:"install_dir"`
	BinaryPath         string           `json:"binary_path"`
	AssetName          string           `json:"asset_name,omitempty"`
	DownloadURL        string           `json:"download_url,omitempty"`
	ChecksumsURL       string           `json:"checksums_url,omitempty"`
	RequiresGitHubAuth bool             `json:"requires_github_auth,omitempty"`
	AuthSource         string           `json:"auth_source,omitempty"`
	Command            []string         `json:"command,omitempty"`
	ShellPath          shellpath.Result `json:"shell_path"`
	DryRun             bool             `json:"dry_run"`
}

// ResolvePlan computes the update action without executing it.
func ResolvePlan(cfg Config) (Plan, error) {
	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = DefaultVersion
	}
	version = strings.TrimPrefix(version, "@")

	binary := strings.TrimSpace(cfg.BinaryName)
	if binary == "" {
		binary = DefaultBinary
	}

	installDir := strings.TrimSpace(cfg.InstallDir)
	if installDir == "" {
		exe, err := os.Executable()
		if err != nil {
			return Plan{}, fmt.Errorf("update tool: resolve current executable: %w", err)
		}
		if eval, err := filepath.EvalSymlinks(exe); err == nil {
			exe = eval
		}
		installDir = filepath.Dir(exe)
	}
	absInstallDir, err := filepath.Abs(installDir)
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: resolve install dir %q: %w", installDir, err)
	}

	method, err := resolveMethod(version, cfg.Method)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Method:     method,
		Version:    displayVersion(version),
		ReleaseTag: releaseTag(version),
		InstallDir: absInstallDir,
		BinaryPath: filepath.Join(absInstallDir, binary),
		DryRun:     cfg.DryRun,
	}
	switch method {
	case MethodSource:
		pkg := DefaultPackage + "@" + version
		plan.Package = DefaultPackage
		plan.Command = []string{"go", "install", pkg}
	case MethodReleaseAssets:
		asset, err := releaseAssetName(binary, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return Plan{}, err
		}
		plan.AssetName = asset
		plan.DownloadURL = releaseDownloadURL(cfg.ReleaseBaseURL, plan.ReleaseTag, asset)
		plan.ChecksumsURL = releaseDownloadURL(cfg.ReleaseBaseURL, plan.ReleaseTag, "checksums.txt")
		plan.RequiresGitHubAuth = true
	default:
		return Plan{}, fmt.Errorf("update tool: unknown update method %q", method)
	}
	return plan, nil
}

// Run updates mars-harness into the resolved install directory.
func Run(ctx context.Context, cfg Config) (Plan, error) {
	plan, err := ResolvePlan(cfg)
	if err != nil {
		return Plan{}, err
	}
	if plan.Method == MethodSource {
		return runSource(ctx, cfg, plan)
	}
	return runReleaseAssets(ctx, cfg, plan)
}

func runSource(ctx context.Context, cfg Config, plan Plan) (Plan, error) {
	if cfg.DryRun {
		pathResult, pathErr := shellpath.Ensure(shellpath.Config{InstallDir: plan.InstallDir, DryRun: true})
		if pathErr == nil {
			plan.ShellPath = pathResult
		}
		return plan, nil
	}

	if err := os.MkdirAll(plan.InstallDir, 0o755); err != nil {
		return Plan{}, fmt.Errorf("update tool: create install dir %s: %w", plan.InstallDir, err)
	}

	cmd := exec.CommandContext(ctx, plan.Command[0], plan.Command[1:]...)
	cmd.Env = append(os.Environ(), "GOBIN="+plan.InstallDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: %s failed with GOBIN=%s: %w\n%s\nIf this is a permission error, rerun with an install dir you own or install from the source checkout with `make install`",
			strings.Join(plan.Command, " "), plan.InstallDir, err, strings.TrimSpace(string(output)))
	}
	if cfg.SkipShellPath {
		return plan, nil
	}
	pathResult, err := shellpath.Ensure(shellpath.Config{InstallDir: plan.InstallDir})
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: installed %s but could not configure shell PATH: %w", plan.BinaryPath, err)
	}
	plan.ShellPath = pathResult
	return plan, nil
}

func runReleaseAssets(ctx context.Context, cfg Config, plan Plan) (Plan, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	plan.AuthSource = githubauth.ResolveToken(ctx, githubauth.Options{}).Token.Source
	if cfg.DryRun {
		pathResult, pathErr := shellpath.Ensure(shellpath.Config{InstallDir: plan.InstallDir, DryRun: true})
		if pathErr == nil {
			plan.ShellPath = pathResult
		}
		return plan, nil
	}
	if shouldResolveReleaseAssetInfo(cfg, plan) {
		resolved, err := resolveReleaseAssetPlan(ctx, client, cfg, plan)
		if err != nil {
			return Plan{}, err
		}
		plan = resolved
	}
	if err := os.MkdirAll(plan.InstallDir, 0o755); err != nil {
		return Plan{}, fmt.Errorf("update tool: create install dir %s: %w", plan.InstallDir, err)
	}
	tmpDir, err := os.MkdirTemp(plan.InstallDir, ".mars-harness-update-*")
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: create temporary update directory in %s: %w\nRerun with --install-dir set to a directory you own, or use --source from a Go-enabled source checkout.",
			plan.InstallDir, err)
	}
	defer os.RemoveAll(tmpDir)

	tmpBinary := filepath.Join(tmpDir, plan.AssetName)
	if err := downloadFile(ctx, client, plan.DownloadURL, tmpBinary); err != nil {
		return Plan{}, fmt.Errorf("update tool: download %s: %w\nRelease %s must contain %s and checksums.txt. Run `mars-harness release verify-assets --version %s` before retrying.",
			plan.DownloadURL, err, plan.ReleaseTag, plan.AssetName, plan.ReleaseTag)
	}
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(ctx, client, plan.ChecksumsURL, checksumsPath); err != nil {
		return Plan{}, fmt.Errorf("update tool: download checksums %s: %w\nCannot verify binary integrity; aborting without replacing %s.",
			plan.ChecksumsURL, err, plan.BinaryPath)
	}
	expected, err := checksumForAsset(checksumsPath, plan.AssetName)
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: read checksum for %s: %w\nRelease %s is incomplete; aborting without replacing %s.",
			plan.AssetName, err, plan.ReleaseTag, plan.BinaryPath)
	}
	actual, err := sha256File(tmpBinary)
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: checksum downloaded binary: %w", err)
	}
	if !strings.EqualFold(expected, actual) {
		return Plan{}, fmt.Errorf("update tool: checksum mismatch for %s: got %s, want %s\nDownloaded file was kept in a temporary directory and %s was not replaced.",
			plan.AssetName, actual, expected, plan.BinaryPath)
	}
	if err := os.Chmod(tmpBinary, 0o755); err != nil {
		return Plan{}, fmt.Errorf("update tool: mark %s executable: %w", plan.AssetName, err)
	}
	if err := os.Rename(tmpBinary, plan.BinaryPath); err != nil {
		return Plan{}, fmt.Errorf("update tool: replace %s: %w\nRerun with --install-dir set to a directory you own, or use sudo with the installer script.",
			plan.BinaryPath, err)
	}
	if cfg.SkipShellPath {
		return plan, nil
	}
	pathResult, err := shellpath.Ensure(shellpath.Config{InstallDir: plan.InstallDir})
	if err != nil {
		return Plan{}, fmt.Errorf("update tool: installed %s but could not configure shell PATH: %w", plan.BinaryPath, err)
	}
	plan.ShellPath = pathResult
	return plan, nil
}

func shouldResolveReleaseAssetInfo(cfg Config, plan Plan) bool {
	if plan.Version == DefaultVersion {
		return true
	}
	return strings.TrimSpace(cfg.ReleaseBaseURL) == ""
}

func resolveReleaseAssetPlan(ctx context.Context, client *http.Client, cfg Config, plan Plan) (Plan, error) {
	releaseURL := cfg.LatestReleaseURL
	if plan.Version != DefaultVersion {
		releaseURL = ReleaseAPIURL(DefaultRepoFullName, plan.ReleaseTag)
	}
	release, err := LatestReleaseInfo(ctx, client, releaseURL)
	if err != nil {
		return Plan{}, err
	}
	report := VerifyReleaseAssetInfo(release)
	if !report.OK {
		return Plan{}, fmt.Errorf("update tool: latest release %s is missing required assets: %s\nPush tag %s from the release-note commit and let the Release workflow attach binaries before retrying.",
			releaseIdentity(release), strings.Join(report.Missing, ", "), releaseIdentity(release))
	}
	plan.Version = report.Version
	plan.ReleaseTag = release.TagName
	if plan.ReleaseTag == "" {
		plan.ReleaseTag = releaseTag(report.Version)
	}
	for _, asset := range release.Assets {
		switch asset.Name {
		case plan.AssetName:
			if asset.APIURL != "" {
				plan.DownloadURL = asset.APIURL
			} else if asset.BrowserDownloadURL != "" {
				plan.DownloadURL = asset.BrowserDownloadURL
			}
		case "checksums.txt":
			if asset.APIURL != "" {
				plan.ChecksumsURL = asset.APIURL
			} else if asset.BrowserDownloadURL != "" {
				plan.ChecksumsURL = asset.BrowserDownloadURL
			}
		}
	}
	if plan.DownloadURL == "" {
		plan.DownloadURL = releaseDownloadURL(cfg.ReleaseBaseURL, plan.ReleaseTag, plan.AssetName)
	}
	if plan.ChecksumsURL == "" {
		plan.ChecksumsURL = releaseDownloadURL(cfg.ReleaseBaseURL, plan.ReleaseTag, "checksums.txt")
	}
	return plan, nil
}

func resolveMethod(version string, requested UpdateMethod) (UpdateMethod, error) {
	if requested != "" {
		switch requested {
		case MethodReleaseAssets, MethodSource:
			return requested, nil
		default:
			return "", fmt.Errorf("update tool: unknown update method %q; use %q or %q", requested, MethodReleaseAssets, MethodSource)
		}
	}
	if version == "main" {
		return MethodSource, nil
	}
	return MethodReleaseAssets, nil
}

func releaseAssetName(binary, goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin":
	default:
		return "", fmt.Errorf("update tool: release assets are available for linux/darwin on amd64/arm64; current platform is %s/%s. Use --source from a Go-enabled source-development environment.",
			goos, goarch)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("update tool: release assets are available for linux/darwin on amd64/arm64; current platform is %s/%s. Use --source from a Go-enabled source-development environment.",
			goos, goarch)
	}
	return fmt.Sprintf("%s-%s-%s", binary, goos, goarch), nil
}

func displayVersion(version string) string {
	if version == DefaultVersion || version == "main" {
		return version
	}
	return NormalizeVersion(version)
}

func releaseTag(version string) string {
	v := strings.TrimSpace(version)
	v = strings.TrimPrefix(v, "@")
	if v == "" || v == DefaultVersion {
		return DefaultVersion
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	if _, ok := parseSemver(v); ok {
		return "v" + v
	}
	return v
}

func releaseDownloadURL(baseURL, tag, asset string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = DefaultReleaseDownloadURL
	}
	return base + "/" + tag + "/" + asset
}

func releaseIdentity(release ReleaseInfo) string {
	if release.TagName != "" {
		return release.TagName
	}
	if release.Name != "" {
		return release.Name
	}
	return "latest"
}

func downloadFile(ctx context.Context, client *http.Client, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	setGitHubDownloadHeaders(req, "mars-harness-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s%s", url, resp.Status, githubAuthHint(resp.StatusCode))
	}
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func checksumForAsset(checksumsPath, assetName string) (string, error) {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == assetName || strings.TrimPrefix(fields[1], "*") == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("%s not found in checksums.txt", assetName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
