/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/selfupdate"
)

// AssetTarget is one platform binary produced for a Mars Harness release.
type AssetTarget struct {
	GOOS   string
	GOARCH string
}

// AssetTargets is the source release binary matrix.
var AssetTargets = []AssetTarget{
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "darwin", GOARCH: "amd64"},
	{GOOS: "darwin", GOARCH: "arm64"},
}

// PublishAssetsConfig controls local release asset publication.
type PublishAssetsConfig struct {
	RepoRoot       string
	Version        string
	DistDir        string
	Upload         string
	GitHubRepo     string
	Stdout         io.Writer
	Stderr         io.Writer
	Now            time.Time
	SkipBuild      bool
	CommandContext func(context.Context, string, ...string) *exec.Cmd
}

// PublishAssetsResult describes the locally produced release assets.
type PublishAssetsResult struct {
	TagName       string
	Version       string
	DistDir       string
	Assets        []string
	ChecksumsPath string
	Uploaded      bool
	UploadSkipped bool
}

// PublishAssets builds, verifies, and optionally mirrors source release assets.
func PublishAssets(ctx context.Context, cfg PublishAssetsConfig) (PublishAssetsResult, error) {
	repoRoot, err := filepath.Abs(strings.TrimSpace(cfg.RepoRoot))
	if err != nil {
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: resolve repo: %w", err)
	}
	if repoRoot == "" {
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: repo path is empty")
	}
	tag := releaseTag(cfg.Version)
	version := strings.TrimPrefix(tag, "v")
	if version == "" {
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: version is required, for example v1.2.3")
	}
	dist := strings.TrimSpace(cfg.DistDir)
	if dist == "" {
		dist = filepath.Join(repoRoot, "dist", "releases")
	}
	if !filepath.IsAbs(dist) {
		dist = filepath.Join(repoRoot, dist)
	}
	dist, err = filepath.Abs(dist)
	if err != nil {
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: resolve dist: %w", err)
	}
	upload := strings.TrimSpace(cfg.Upload)
	if upload == "" {
		upload = "none"
	}
	switch upload {
	case "none", "github", "auto":
	default:
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: --upload must be one of none, github, or auto")
	}
	runner := cfg.CommandContext
	if runner == nil {
		runner = exec.CommandContext
	}
	if err := validateReleaseAssetState(ctx, runner, repoRoot, version, tag); err != nil {
		return PublishAssetsResult{}, err
	}
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: create dist directory %s: %w", dist, err)
	}
	commit := strings.TrimSpace(commandOutput(ctx, runner, repoRoot, "git", "rev-parse", "--short=8", "HEAD"))
	if commit == "" {
		commit = "unknown"
	}
	now := cfg.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	date := now.UTC().Format(time.RFC3339)

	var assets []string
	for _, target := range AssetTargets {
		name := fmt.Sprintf("mars-harness-%s-%s", target.GOOS, target.GOARCH)
		outPath := filepath.Join(dist, name)
		if !cfg.SkipBuild {
			cmd := runner(ctx, "go", "build",
				"-ldflags", fmt.Sprintf("-X main.version=%s -X main.commit=%s -X main.date=%s", tag, commit, date),
				"-o", outPath,
				"./cmd/mars-harness",
			)
			cmd.Dir = repoRoot
			cmd.Env = append(os.Environ(),
				"CGO_ENABLED=0",
				"GOOS="+target.GOOS,
				"GOARCH="+target.GOARCH,
			)
			cmd.Stdout = cfg.Stdout
			cmd.Stderr = cfg.Stderr
			if err := cmd.Run(); err != nil {
				return PublishAssetsResult{}, fmt.Errorf("release publish-assets: build %s/%s: %w", target.GOOS, target.GOARCH, err)
			}
		}
		assets = append(assets, outPath)
	}
	checksumsPath := filepath.Join(dist, "checksums.txt")
	if err := writeChecksums(checksumsPath, assets); err != nil {
		return PublishAssetsResult{}, err
	}
	report, err := VerifyLocalAssets(dist, tag)
	if err != nil {
		return PublishAssetsResult{}, err
	}
	if !report.OK {
		return PublishAssetsResult{}, fmt.Errorf("release publish-assets: local assets incomplete: missing %s", strings.Join(report.Missing, ", "))
	}

	result := PublishAssetsResult{
		TagName:       tag,
		Version:       version,
		DistDir:       dist,
		Assets:        assets,
		ChecksumsPath: checksumsPath,
	}
	switch upload {
	case "github":
		if err := uploadGitHubAssets(ctx, runner, repoRoot, cfg.GitHubRepo, tag, append(assets, checksumsPath)); err != nil {
			return result, err
		}
		result.Uploaded = true
	case "auto":
		if _, err := exec.LookPath("gh"); err != nil {
			result.UploadSkipped = true
			return result, nil
		}
		if err := uploadGitHubAssets(ctx, runner, repoRoot, cfg.GitHubRepo, tag, append(assets, checksumsPath)); err != nil {
			return result, err
		}
		result.Uploaded = true
	}
	return result, nil
}

// VerifyLocalAssets checks local release assets and their checksums.
func VerifyLocalAssets(distDir, version string) (selfupdate.ReleaseAssetReport, error) {
	distDir = strings.TrimSpace(distDir)
	if distDir == "" {
		return selfupdate.ReleaseAssetReport{}, fmt.Errorf("release verify-assets: --dist path is empty")
	}
	abs, err := filepath.Abs(distDir)
	if err != nil {
		return selfupdate.ReleaseAssetReport{}, fmt.Errorf("release verify-assets: resolve dist: %w", err)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return selfupdate.ReleaseAssetReport{}, fmt.Errorf("release verify-assets: read local dist %s: %w", abs, err)
	}
	foundSet := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			foundSet[entry.Name()] = true
		}
	}
	required := selfupdate.ExpectedReleaseAssetNames()
	var found, missing []string
	for _, name := range required {
		if foundSet[name] {
			found = append(found, name)
		} else {
			missing = append(missing, name)
		}
	}
	sort.Strings(found)
	if len(missing) == 0 {
		if err := verifyChecksums(filepath.Join(abs, "checksums.txt"), abs); err != nil {
			missing = append(missing, "valid checksums.txt")
		}
	}
	tag := releaseTag(version)
	return selfupdate.ReleaseAssetReport{
		TagName:  tag,
		Version:  strings.TrimPrefix(tag, "v"),
		URL:      abs,
		Required: required,
		Found:    found,
		Missing:  missing,
		OK:       len(missing) == 0,
	}, nil
}

func validateReleaseAssetState(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, version, tag string) error {
	fileVersion, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		return fmt.Errorf("release publish-assets: read VERSION: %w", err)
	}
	if strings.TrimSpace(string(fileVersion)) != version {
		return fmt.Errorf("release publish-assets: VERSION is %q, expected %q for %s", strings.TrimSpace(string(fileVersion)), version, tag)
	}
	status := commandOutput(ctx, runner, repoRoot, "git", "status", "--short")
	if strings.TrimSpace(status) != "" {
		return fmt.Errorf("release publish-assets: worktree must be clean before publishing assets:\n%s", strings.TrimSpace(status))
	}
	headSubject := commandOutput(ctx, runner, repoRoot, "git", "log", "-1", "--pretty=%s")
	if strings.TrimSpace(headSubject) != "release: notes "+version {
		return fmt.Errorf("release publish-assets: HEAD must be release-note commit %q, got %q", "release: notes "+version, strings.TrimSpace(headSubject))
	}
	headSHA := commandOutput(ctx, runner, repoRoot, "git", "rev-parse", "HEAD")
	tagSHA, err := commandOutputErr(ctx, runner, repoRoot, "git", "rev-list", "-n", "1", tag)
	if err != nil {
		return fmt.Errorf("release publish-assets: tag %s must exist and point at the release-note commit: %w", tag, err)
	}
	if strings.TrimSpace(tagSHA) != strings.TrimSpace(headSHA) {
		return fmt.Errorf("release publish-assets: tag %s points at %s, expected release-note HEAD %s", tag, strings.TrimSpace(tagSHA), strings.TrimSpace(headSHA))
	}
	return nil
}

func writeChecksums(path string, assets []string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("release publish-assets: create checksums.txt: %w", err)
	}
	defer f.Close()
	for _, asset := range assets {
		sum, err := fileSHA256(asset)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(f, "%s  %s\n", sum, filepath.Base(asset)); err != nil {
			return fmt.Errorf("release publish-assets: write checksums.txt: %w", err)
		}
	}
	return nil
}

func verifyChecksums(path, dir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			return fmt.Errorf("invalid checksum line %q", scanner.Text())
		}
		got, err := fileSHA256(filepath.Join(dir, fields[len(fields)-1]))
		if err != nil {
			return err
		}
		if got != fields[0] {
			return fmt.Errorf("checksum mismatch for %s", fields[len(fields)-1])
		}
	}
	return scanner.Err()
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("release assets: open %s: %w", path, err)
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("release assets: hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func uploadGitHubAssets(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, repoFullName, tag string, files []string) error {
	repoFullName = strings.TrimSpace(repoFullName)
	if repoFullName == "" {
		repoFullName = selfupdate.DefaultRepoFullName
	}
	if err := runCommand(ctx, runner, repoRoot, nil, nil, "gh", "release", "view", tag, "--repo", repoFullName); err != nil {
		body, bodyErr := changelogEntryFile(repoRoot, strings.TrimPrefix(tag, "v"))
		if bodyErr != nil {
			return fmt.Errorf("release publish-assets: GitHub release %s is missing and changelog body could not be prepared: %w", tag, bodyErr)
		}
		defer os.Remove(body)
		if err := runCommand(ctx, runner, repoRoot, nil, nil, "gh", "release", "create", tag, "--repo", repoFullName, "--title", tag, "--notes-file", body); err != nil {
			return fmt.Errorf("release publish-assets: create GitHub release %s: %w", tag, err)
		}
	}
	args := []string{"release", "upload", tag, "--repo", repoFullName, "--clobber"}
	args = append(args, files...)
	if err := runCommand(ctx, runner, repoRoot, nil, nil, "gh", args...); err != nil {
		return fmt.Errorf("release publish-assets: upload GitHub assets for %s: %w", tag, err)
	}
	return nil
}

func changelogEntryFile(repoRoot, version string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "CHANGELOG.md"))
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ["+version+"]") {
			start = i
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("CHANGELOG.md has no entry for %s", version)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## [") {
			end = i
			break
		}
	}
	tmp, err := os.CreateTemp("", "mars-harness-release-body-*.md")
	if err != nil {
		return "", err
	}
	if _, err := tmp.WriteString(strings.Join(lines[start:end], "\n")); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func releaseTag(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func commandOutput(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, dir, name string, args ...string) string {
	out, _ := commandOutputErr(ctx, runner, dir, name, args...)
	return out
}

func commandOutputErr(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, dir, name string, args ...string) (string, error) {
	cmd := runner(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func runCommand(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := runner(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
