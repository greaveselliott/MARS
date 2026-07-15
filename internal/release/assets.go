/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-017-open-source-publication.md
*/
package release

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

// AssetTarget is one platform binary produced for a MARS release.
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
	TagName             string
	Version             string
	DistDir             string
	Assets              []string
	ChecksumsPath       string
	Uploaded            bool
	UploadSkipped       bool
	RemoteVerifiedCount int
	RemoteExpectedCount int
}

const (
	githubMirrorMetadataAttempts = 15
	githubMirrorPollInterval     = 2 * time.Second
)

type localReleaseAsset struct {
	Name   string
	Path   string
	Size   int64
	SHA256 string
	Info   os.FileInfo
}

type githubReleaseMetadata struct {
	TagName string                    `json:"tag_name"`
	Assets  []githubReleaseAssetState `json:"assets"`
}

type githubReleaseAssetState struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type githubMirrorAssessment struct {
	Missing      []string
	Mismatched   []string
	Pending      []string
	Unverifiable []string
	Extra        []string
	Duplicate    []string
}

type githubAPIError struct {
	StatusCode int
	Operation  string
}

func (e *githubAPIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("GitHub API %s returned HTTP %d", e.Operation, e.StatusCode)
	}
	return fmt.Sprintf("GitHub API %s failed", e.Operation)
}

var (
	githubRepoComponentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)
	releaseTagPattern          = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z][0-9A-Za-z.-]*)?(?:\+[0-9A-Za-z][0-9A-Za-z.-]*)?$`)
	fullGitSHA1Pattern         = regexp.MustCompile(`^[0-9a-f]{40}$`)
	httpStatusPattern          = regexp.MustCompile(`(?i)\(HTTP ([1-5][0-9]{2})\)$`)
)

func (a githubMirrorAssessment) complete() bool {
	return len(a.Missing) == 0 && len(a.Mismatched) == 0 && len(a.Pending) == 0 &&
		len(a.Unverifiable) == 0 && len(a.Extra) == 0 && len(a.Duplicate) == 0
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
		name := fmt.Sprintf("mars-%s-%s", target.GOOS, target.GOARCH)
		outPath := filepath.Join(dist, name)
		if !cfg.SkipBuild {
			cmd := runner(ctx, "go", "build",
				"-ldflags", fmt.Sprintf("-X main.version=%s -X main.commit=%s -X main.date=%s", tag, commit, date),
				"-o", outPath,
				"./cmd/mars",
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
		legacyName := fmt.Sprintf("mars-harness-%s-%s", target.GOOS, target.GOARCH)
		legacyPath := filepath.Join(dist, legacyName)
		if err := copyFile(outPath, legacyPath); err != nil {
			return PublishAssetsResult{}, fmt.Errorf("release publish-assets: create legacy asset alias %s: %w", legacyName, err)
		}
		assets = append(assets, legacyPath)
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
		TagName:             tag,
		Version:             version,
		DistDir:             dist,
		Assets:              assets,
		ChecksumsPath:       checksumsPath,
		RemoteExpectedCount: len(selfupdate.ExpectedReleaseAssetNames()),
	}
	switch upload {
	case "github":
		verified, err := uploadGitHubAssets(ctx, runner, repoRoot, cfg.GitHubRepo, tag, append(assets, checksumsPath))
		if err != nil {
			return result, err
		}
		result.RemoteVerifiedCount = verified
		result.Uploaded = true
	case "auto":
		if _, err := exec.LookPath("gh"); err != nil {
			result.UploadSkipped = true
			return result, nil
		}
		verified, err := uploadGitHubAssets(ctx, runner, repoRoot, cfg.GitHubRepo, tag, append(assets, checksumsPath))
		if err != nil {
			return result, err
		}
		result.RemoteVerifiedCount = verified
		result.Uploaded = true
	}
	return result, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
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
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	var found, missing, extra []string
	for _, name := range required {
		if foundSet[name] {
			found = append(found, name)
		} else {
			missing = append(missing, name)
		}
	}
	for name := range foundSet {
		if !requiredSet[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(found)
	sort.Strings(extra)
	if len(missing) == 0 && len(extra) == 0 {
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
		Extra:    extra,
		OK:       len(missing) == 0 && len(extra) == 0,
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

func uploadGitHubAssets(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, repoFullName, tag string, files []string) (int, error) {
	repoFullName = strings.TrimSpace(repoFullName)
	if repoFullName == "" {
		repoFullName = selfupdate.DefaultRepoFullName
	}
	if err := validateGitHubMirrorIdentity(repoFullName, tag); err != nil {
		return 0, err
	}
	localHead, err := commandOutputErr(ctx, runner, repoRoot, "git", "rev-parse", "HEAD")
	if err != nil {
		return 0, fmt.Errorf("release publish-assets: resolve full local HEAD before GitHub mirroring: %w", err)
	}
	localHead = strings.ToLower(strings.TrimSpace(localHead))
	if !fullGitSHA1Pattern.MatchString(localHead) {
		return 0, fmt.Errorf("release publish-assets: local HEAD is not a full Git SHA-1; run `git rev-parse HEAD` and repair the checkout before retrying")
	}
	contract, err := localReleaseAssetContract(files)
	if err != nil {
		return 0, err
	}
	contract, snapshotDir, err := snapshotReleaseAssetContract(contract)
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(snapshotDir)
	if err := verifyRemoteTagCommit(ctx, runner, repoRoot, repoFullName, tag, localHead); err != nil {
		return 0, err
	}
	if err := ensureGitHubRelease(ctx, runner, repoRoot, repoFullName, tag); err != nil {
		return 0, err
	}
	fetch := func(fetchCtx context.Context) (githubReleaseMetadata, error) {
		return fetchGitHubReleaseMetadata(fetchCtx, runner, repoRoot, repoFullName, tag)
	}
	upload := func(uploadCtx context.Context, asset localReleaseAsset, clobber bool) error {
		if err := revalidateLocalReleaseAsset(asset); err != nil {
			return err
		}
		args := []string{"release", "upload", tag, "--repo", repoFullName}
		if clobber {
			args = append(args, "--clobber")
		}
		args = append(args, asset.Path)
		if err := runCommand(uploadCtx, runner, repoRoot, nil, nil, "gh", args...); err != nil {
			if ctxErr := uploadCtx.Err(); ctxErr != nil {
				return fmt.Errorf("upload canceled: %w", ctxErr)
			}
			return fmt.Errorf("GitHub CLI upload failed")
		}
		return nil
	}
	verified, err := reconcileGitHubAssets(ctx, tag, contract, githubMirrorMetadataAttempts, githubMirrorPollInterval, fetch, upload)
	if err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled before final postcondition: %w", err)
	}
	for _, asset := range contract {
		if err := revalidateLocalReleaseAsset(asset); err != nil {
			return 0, err
		}
	}
	if err := verifyRemoteTagCommit(ctx, runner, repoRoot, repoFullName, tag, localHead); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled before success: %w", err)
	}
	return verified, nil
}

func ensureGitHubRelease(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, repoFullName, tag string) error {
	metadata, err := fetchGitHubReleaseMetadata(ctx, runner, repoRoot, repoFullName, tag)
	if err == nil {
		if metadata.TagName != tag {
			return fmt.Errorf("release publish-assets: mirror_incomplete: exact release tag mismatch before mutation")
		}
		return nil
	}
	var apiErr *githubAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		return fmt.Errorf("release publish-assets: mirror_incomplete: cannot establish exact release state before mutation: %w", err)
	}
	body, bodyErr := changelogEntryFile(repoRoot, strings.TrimPrefix(tag, "v"))
	if bodyErr != nil {
		return fmt.Errorf("release publish-assets: GitHub release %s is missing and changelog body could not be prepared: %w", tag, bodyErr)
	}
	defer os.Remove(body)
	if err := runCommand(ctx, runner, repoRoot, nil, nil, "gh", "release", "create", tag, "--repo", repoFullName, "--verify-tag", "--title", tag, "--notes-file", body); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("release publish-assets: mirror_incomplete: GitHub release creation canceled: %w", ctxErr)
		}
		return fmt.Errorf("release publish-assets: mirror_incomplete: GitHub CLI could not create exact-tag release %s with --verify-tag", tag)
	}
	return nil
}

func validateGitHubMirrorIdentity(repoFullName, tag string) error {
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 || !githubRepoComponentPattern.MatchString(parts[0]) || !githubRepoComponentPattern.MatchString(parts[1]) || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return fmt.Errorf("release publish-assets: --github-repo must be a safe owner/repository name")
	}
	if !releaseTagPattern.MatchString(tag) {
		return fmt.Errorf("release publish-assets: version must resolve to a safe semantic release tag such as v1.2.3")
	}
	return nil
}

func snapshotReleaseAssetContract(contract []localReleaseAsset) ([]localReleaseAsset, string, error) {
	if len(contract) == 0 {
		return nil, "", fmt.Errorf("release publish-assets: mirror_incomplete: local asset contract is empty")
	}
	parent := filepath.Dir(contract[0].Path)
	dir, err := os.MkdirTemp(parent, ".mars-release-upload-")
	if err != nil {
		return nil, "", fmt.Errorf("release publish-assets: create owner-only upload snapshot: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := os.Chmod(dir, 0o700); err != nil {
		cleanup()
		return nil, "", fmt.Errorf("release publish-assets: secure upload snapshot: %w", err)
	}
	var snapshot []localReleaseAsset
	for _, asset := range contract {
		if err := revalidateLocalReleaseAsset(asset); err != nil {
			cleanup()
			return nil, "", err
		}
		src, err := os.Open(asset.Path)
		if err != nil {
			cleanup()
			return nil, "", fmt.Errorf("release publish-assets: local asset %s changed before snapshot", asset.Name)
		}
		srcInfo, statErr := src.Stat()
		if statErr != nil || !os.SameFile(asset.Info, srcInfo) {
			_ = src.Close()
			cleanup()
			return nil, "", fmt.Errorf("release publish-assets: local asset %s changed before snapshot", asset.Name)
		}
		dstPath := filepath.Join(dir, asset.Name)
		dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			_ = src.Close()
			cleanup()
			return nil, "", fmt.Errorf("release publish-assets: create immutable snapshot for %s: %w", asset.Name, err)
		}
		hash := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(dst, hash), src)
		closeDstErr := dst.Close()
		closeSrcErr := src.Close()
		if copyErr != nil || closeDstErr != nil || closeSrcErr != nil || written != asset.Size || hex.EncodeToString(hash.Sum(nil)) != asset.SHA256 {
			cleanup()
			return nil, "", fmt.Errorf("release publish-assets: local asset %s changed while creating upload snapshot", asset.Name)
		}
		if err := os.Chmod(dstPath, 0o400); err != nil {
			cleanup()
			return nil, "", fmt.Errorf("release publish-assets: seal immutable snapshot for %s: %w", asset.Name, err)
		}
		info, size, digest, err := inspectLocalReleaseAsset(dstPath)
		if err != nil || size != asset.Size || digest != asset.SHA256 {
			cleanup()
			return nil, "", fmt.Errorf("release publish-assets: immutable snapshot verification failed for %s", asset.Name)
		}
		snapshot = append(snapshot, localReleaseAsset{Name: asset.Name, Path: dstPath, Size: size, SHA256: digest, Info: info})
	}
	return snapshot, dir, nil
}

func revalidateLocalReleaseAsset(asset localReleaseAsset) error {
	info, size, digest, err := inspectLocalReleaseAsset(asset.Path)
	if err != nil || asset.Info == nil || !os.SameFile(asset.Info, info) || size != asset.Size || digest != asset.SHA256 {
		return fmt.Errorf("release publish-assets: mirror_incomplete: local asset %s changed after contract creation; rebuild into a fresh directory and retry", asset.Name)
	}
	return nil
}

func inspectLocalReleaseAsset(path string) (os.FileInfo, int64, string, error) {
	parentInfo, err := os.Lstat(filepath.Dir(path))
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return nil, 0, "", fmt.Errorf("asset parent must be a real directory")
	}
	pathInfo, err := os.Lstat(path)
	if err != nil || !pathInfo.Mode().IsRegular() || pathInfo.Size() <= 0 {
		return nil, 0, "", fmt.Errorf("asset must be a non-empty regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, "", err
	}
	defer f.Close()
	openInfo, err := f.Stat()
	if err != nil || !openInfo.Mode().IsRegular() || !os.SameFile(pathInfo, openInfo) {
		return nil, 0, "", fmt.Errorf("asset identity changed while opening")
	}
	hash := sha256.New()
	size, err := io.Copy(hash, f)
	if err != nil {
		return nil, 0, "", err
	}
	afterInfo, err := os.Lstat(path)
	if err != nil || !os.SameFile(openInfo, afterInfo) || size != openInfo.Size() {
		return nil, 0, "", fmt.Errorf("asset identity changed while hashing")
	}
	return openInfo, size, hex.EncodeToString(hash.Sum(nil)), nil
}

func localReleaseAssetContract(files []string) ([]localReleaseAsset, error) {
	required := selfupdate.ExpectedReleaseAssetNames()
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	byName := make(map[string]localReleaseAsset, len(files))
	var extra, duplicate []string
	var assetDir string
	for _, path := range files {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("release publish-assets: resolve local asset path: %w", err)
		}
		path = absPath
		if assetDir == "" {
			assetDir = filepath.Dir(path)
		} else if filepath.Dir(path) != assetDir {
			return nil, fmt.Errorf("release publish-assets: mirror_incomplete: all local release assets must share one real parent directory")
		}
		name := filepath.Base(path)
		if !requiredSet[name] {
			extra = append(extra, name)
			continue
		}
		if _, exists := byName[name]; exists {
			duplicate = append(duplicate, name)
			continue
		}
		info, size, digest, err := inspectLocalReleaseAsset(path)
		if err != nil {
			return nil, fmt.Errorf("release publish-assets: build local mirror contract for %s: %w", name, err)
		}
		byName[name] = localReleaseAsset{Name: name, Path: path, Size: size, SHA256: digest, Info: info}
	}
	var missing []string
	for _, name := range required {
		if _, ok := byName[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(extra)
	sort.Strings(duplicate)
	if len(missing) > 0 || len(extra) > 0 || len(duplicate) > 0 {
		return nil, fmt.Errorf("release publish-assets: mirror_incomplete: local asset contract is not the exact nine-file set (%s)",
			formatMirrorIssues(missing, nil, nil, nil, extra, duplicate))
	}
	contract := make([]localReleaseAsset, 0, len(required))
	for _, name := range required {
		contract = append(contract, byName[name])
	}
	sort.Slice(contract, func(i, j int) bool { return contract[i].Name < contract[j].Name })
	return contract, nil
}

func fetchGitHubReleaseMetadata(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, repoFullName, tag string) (githubReleaseMetadata, error) {
	if err := ctx.Err(); err != nil {
		return githubReleaseMetadata{}, err
	}
	endpoint := fmt.Sprintf("repos/%s/releases/tags/%s", repoFullName, url.PathEscape(tag))
	out, err := githubAPIOutput(ctx, runner, repoRoot, "read exact-tag release metadata", endpoint)
	if err != nil {
		return githubReleaseMetadata{}, err
	}
	if err := ctx.Err(); err != nil {
		return githubReleaseMetadata{}, err
	}
	var metadata githubReleaseMetadata
	if err := json.Unmarshal([]byte(out), &metadata); err != nil {
		return githubReleaseMetadata{}, fmt.Errorf("decode exact-tag metadata: response was not valid bounded JSON")
	}
	if len(metadata.Assets) > 64 {
		return githubReleaseMetadata{}, fmt.Errorf("decode exact-tag metadata: response exceeded the 64-asset safety bound")
	}
	return metadata, nil
}

func verifyRemoteTagCommit(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, repoFullName, tag, expectedCommit string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("release publish-assets: mirror_incomplete: remote tag verification canceled: %w", err)
	}
	remoteCommit, err := resolveRemoteTagCommit(ctx, runner, repoRoot, repoFullName, tag)
	if err != nil {
		return fmt.Errorf("release publish-assets: mirror_incomplete: cannot resolve remote tag %s to an exact commit: %w", tag, err)
	}
	if !fullGitSHA1Pattern.MatchString(remoteCommit) {
		return fmt.Errorf("release publish-assets: mirror_incomplete: remote tag did not resolve to a full Git commit SHA")
	}
	if remoteCommit != expectedCommit {
		return fmt.Errorf("release publish-assets: mirror_incomplete: remote tag %s resolves to a different commit than local HEAD; do not move the tag", tag)
	}
	return nil
}

func resolveRemoteTagCommit(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, repoFullName, tag string) (string, error) {
	type gitObject struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
	}
	var payload struct {
		Object gitObject `json:"object"`
	}
	endpoint := fmt.Sprintf("repos/%s/git/ref/tags/%s", repoFullName, url.PathEscape(tag))
	for depth := 0; depth < 8; depth++ {
		out, err := githubAPIOutput(ctx, runner, repoRoot, "resolve remote release tag ref", endpoint)
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		payload = struct {
			Object gitObject `json:"object"`
		}{}
		if err := json.Unmarshal([]byte(out), &payload); err != nil {
			return "", fmt.Errorf("remote tag ref response was not valid bounded JSON")
		}
		objectType := strings.ToLower(strings.TrimSpace(payload.Object.Type))
		objectSHA := strings.ToLower(strings.TrimSpace(payload.Object.SHA))
		if !fullGitSHA1Pattern.MatchString(objectSHA) {
			return "", fmt.Errorf("remote tag object did not contain a full Git SHA-1")
		}
		switch objectType {
		case "commit":
			return objectSHA, nil
		case "tag":
			endpoint = fmt.Sprintf("repos/%s/git/tags/%s", repoFullName, objectSHA)
		default:
			return "", fmt.Errorf("remote tag resolved to unsupported Git object type")
		}
	}
	return "", fmt.Errorf("remote annotated tag chain exceeded the depth limit")
}

func githubAPIOutput(ctx context.Context, runner func(context.Context, string, ...string) *exec.Cmd, repoRoot, operation, endpoint string) (string, error) {
	cmd := runner(ctx, "gh", "api", "--method", "GET", endpoint)
	cmd.Dir = repoRoot
	stdout := &boundedOutput{limit: 1 << 20}
	stderr := &boundedOutput{limit: 8 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if stdout.truncated || stderr.truncated {
		return "", &githubAPIError{Operation: operation}
	}
	if err != nil {
		// gh writes its synthesized `(HTTP NNN)` status to stderr while the
		// server response body remains on stdout. Classify only the trusted CLI
		// status line: appending the response body both hid a real trailing 404
		// and would let hostile body content influence mutation authority.
		status := parseHTTPStatus(stderr.String())
		return "", &githubAPIError{StatusCode: status, Operation: operation}
	}
	return stdout.String(), nil
}

type boundedOutput struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return original, nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return original, nil
	}
	_, _ = b.buf.Write(p)
	return original, nil
}

func (b *boundedOutput) String() string { return b.buf.String() }

func parseHTTPStatus(output string) int {
	match := httpStatusPattern.FindStringSubmatch(strings.TrimSpace(output))
	if len(match) != 2 {
		return 0
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return status
}

func sanitizeRemoteAssetName(value string) string {
	return "<redacted-asset-name>"
}

func reconcileGitHubAssets(
	ctx context.Context,
	tag string,
	contract []localReleaseAsset,
	maxMetadataAttempts int,
	pollInterval time.Duration,
	fetch func(context.Context) (githubReleaseMetadata, error),
	upload func(context.Context, localReleaseAsset, bool) error,
) (int, error) {
	if maxMetadataAttempts < 1 {
		maxMetadataAttempts = 1
	}
	var lastAssessment githubMirrorAssessment
	var lastMetadataErr error
	attempted := make(map[string]bool, len(contract))
	var uploadFailures []string
	for attempt := 1; attempt <= maxMetadataAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled: %w", err)
		}
		metadata, err := fetch(ctx)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled after metadata fetch: %w", ctxErr)
		}
		if err != nil {
			lastMetadataErr = err
		} else {
			lastMetadataErr = nil
			if metadata.TagName != tag {
				return 0, fmt.Errorf("release publish-assets: mirror_incomplete: remote release tag does not exactly match expected tag %q", tag)
			}
			lastAssessment = assessGitHubMirror(contract, metadata.Assets)
			if len(lastAssessment.Extra) > 0 || len(lastAssessment.Duplicate) > 0 {
				return 0, mirrorIncompleteError(tag, attempt, maxMetadataAttempts, lastAssessment, nil, uploadFailures)
			}
			if lastAssessment.complete() {
				if err := ctx.Err(); err != nil {
					return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled before success: %w", err)
				}
				return len(contract), nil
			}
			byName := make(map[string]localReleaseAsset, len(contract))
			for _, asset := range contract {
				byName[asset.Name] = asset
			}
			name, clobber := nextGitHubMirrorAction(lastAssessment, attempted)
			if name != "" {
				if err := ctx.Err(); err != nil {
					return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled before reconciling %s: %w", name, err)
				}
				attempted[name] = true
				if err := upload(ctx, byName[name], clobber); err != nil {
					uploadFailures = append(uploadFailures, name)
				}
				if err := ctx.Err(); err != nil {
					return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled after reconciling %s: %w", name, err)
				}
				// Always fetch a fresh exact-tag snapshot after an individual upload,
				// including when the transport reports an error. GitHub may have
				// accepted the bytes before the client observed that error.
				continue
			}
		}
		if attempt == maxMetadataAttempts {
			break
		}
		if err := waitForGitHubMetadata(ctx, pollInterval); err != nil {
			return 0, fmt.Errorf("release publish-assets: mirror_incomplete: verification canceled: %w", err)
		}
	}
	return 0, mirrorIncompleteError(tag, maxMetadataAttempts, maxMetadataAttempts, lastAssessment, lastMetadataErr, uploadFailures)
}

func nextGitHubMirrorAction(assessment githubMirrorAssessment, attempted map[string]bool) (string, bool) {
	for _, name := range assessment.Missing {
		if !attempted[name] {
			return name, false
		}
	}
	for _, name := range assessment.Mismatched {
		if !attempted[name] {
			return name, true
		}
	}
	return "", false
}

func assessGitHubMirror(contract []localReleaseAsset, remote []githubReleaseAssetState) githubMirrorAssessment {
	localByName := make(map[string]localReleaseAsset, len(contract))
	for _, asset := range contract {
		localByName[asset.Name] = asset
	}
	remoteByName := make(map[string][]githubReleaseAssetState, len(remote))
	var assessment githubMirrorAssessment
	for _, asset := range remote {
		if _, ok := localByName[asset.Name]; !ok {
			assessment.Extra = append(assessment.Extra, sanitizeRemoteAssetName(asset.Name))
			continue
		}
		remoteByName[asset.Name] = append(remoteByName[asset.Name], asset)
	}
	for _, local := range contract {
		matches := remoteByName[local.Name]
		switch len(matches) {
		case 0:
			assessment.Missing = append(assessment.Missing, local.Name)
		case 1:
			remoteAsset := matches[0]
			if remoteAsset.State != "uploaded" {
				assessment.Pending = append(assessment.Pending, local.Name+" (state=not-uploaded)")
				continue
			}
			if !validRemoteSHA256(remoteAsset.Digest) || remoteAsset.Size <= 0 {
				assessment.Unverifiable = append(assessment.Unverifiable, local.Name)
				continue
			}
			if remoteAsset.Size != local.Size || remoteAsset.Digest != "sha256:"+local.SHA256 {
				assessment.Mismatched = append(assessment.Mismatched, local.Name)
			}
		default:
			assessment.Duplicate = append(assessment.Duplicate, fmt.Sprintf("%s (%d copies)", local.Name, len(matches)))
		}
	}
	sort.Strings(assessment.Missing)
	sort.Strings(assessment.Mismatched)
	sort.Strings(assessment.Pending)
	sort.Strings(assessment.Unverifiable)
	sort.Strings(assessment.Extra)
	sort.Strings(assessment.Duplicate)
	return assessment
}

func validRemoteSHA256(digest string) bool {
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	return err == nil && strings.ToLower(digest) == digest
}

func waitForGitHubMetadata(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func mirrorIncompleteError(tag string, attempt, maxAttempts int, assessment githubMirrorAssessment, metadataErr error, uploadFailures []string) error {
	detail := formatMirrorIssues(assessment.Missing, assessment.Mismatched, assessment.Pending, assessment.Unverifiable, assessment.Extra, assessment.Duplicate)
	if metadataErr != nil {
		detail += "; metadata_error=" + safeOperationalError(metadataErr)
	}
	if len(uploadFailures) > 0 {
		detail += "; upload_error=" + strings.Join(uploadFailures, " | ")
	}
	return fmt.Errorf("release publish-assets: mirror_incomplete: GitHub release %s did not reach the exact asset contract after metadata check %d/%d (%s); rerun `mars release publish-assets --repo . --version %s --upload github`",
		tag, attempt, maxAttempts, detail, tag)
}

func safeOperationalError(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "request canceled"
	}
	var apiErr *githubAPIError
	if errors.As(err, &apiErr) {
		return apiErr.Error()
	}
	return "request failed"
}

func formatMirrorIssues(missing, mismatched, pending, unverifiable, extra, duplicate []string) string {
	var parts []string
	for _, issue := range []struct {
		label  string
		values []string
	}{
		{label: "missing", values: missing},
		{label: "mismatched", values: mismatched},
		{label: "pending", values: pending},
		{label: "unverifiable", values: unverifiable},
		{label: "extra", values: extra},
		{label: "duplicate", values: duplicate},
	} {
		if len(issue.values) > 0 {
			parts = append(parts, issue.label+"="+strings.Join(issue.values, ","))
		}
	}
	if len(parts) == 0 {
		return "remote metadata remained unavailable"
	}
	return strings.Join(parts, "; ")
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
	tmp, err := os.CreateTemp("", "mars-release-body-*.md")
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
