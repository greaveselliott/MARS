/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package selfupdate

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	runtimedebug "runtime/debug"
	"strings"

	"github.com/greaveselliott/mars/internal/childenv"
	"github.com/greaveselliott/mars/internal/githubauth"
	"github.com/greaveselliott/mars/internal/shellpath"
)

const (
	DefaultPackage = "github.com/greaveselliott/mars/cmd/mars"
	DefaultVersion = "latest"
	DefaultBinary  = "mars"
)

var (
	ErrSignedUpdateIdentity    = errors.New("update tool: current release identity is unavailable; select an exact signed version or reinstall from a trusted source checkout")
	ErrExactModuleBootstrap    = errors.New("update tool: exact-module bootstrap identity is invalid; run only the reviewed install script from an independently reviewed exact release checkout")
	ErrSignedUpdateDestination = errors.New("update tool: implicit latest updates may replace only the running mars binary; omit --install-dir or select an exact signed version")
	ErrSignedUpdateConfig      = errors.New("update tool: signed release mode supports only the canonical mars binary and an exact release version")
	ErrSignedUpdateShellPath   = errors.New("update tool: signed binary replacement committed but shell PATH setup failed")
)

// UpdateMethod describes how update tool obtains the replacement binary.
type UpdateMethod string

const (
	MethodReleaseAssets UpdateMethod = "release-assets"
	MethodSource        UpdateMethod = "source"
)

// Config controls an installed-tool update run.
type Config struct {
	Version        string
	CurrentVersion string
	CurrentCommit  string
	InstallDir     string
	BinaryName     string
	Method         UpdateMethod
	DryRun         bool
	SkipShellPath  bool
	ExactBootstrap bool
	HTTPClient     *http.Client
}

// Plan is the resolved update action.
type Plan struct {
	Method        UpdateMethod     `json:"method"`
	Package       string           `json:"package,omitempty"`
	Version       string           `json:"version"`
	ReleaseTag    string           `json:"release_tag,omitempty"`
	ReleaseCommit string           `json:"release_commit,omitempty"`
	InstallDir    string           `json:"install_dir"`
	BinaryPath    string           `json:"binary_path"`
	AssetName     string           `json:"asset_name,omitempty"`
	AuthSource    string           `json:"auth_source,omitempty"`
	Command       []string         `json:"command,omitempty"`
	ShellPath     shellpath.Result `json:"shell_path"`
	DryRun        bool             `json:"dry_run"`
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
		if binary != DefaultBinary {
			return Plan{}, ErrSignedUpdateConfig
		}
		requestedTag, latest, ok := normalizeSignedReleaseRequest(version)
		if !ok {
			return Plan{}, ErrSignedUpdateConfig
		}
		plan.ReleaseTag = requestedTag
		plan.BinaryPath = filepath.Join(absInstallDir, DefaultBinary)
		if latest {
			plan.Version = DefaultVersion
		} else {
			plan.Version = strings.TrimPrefix(requestedTag, "v")
		}
	default:
		return Plan{}, fmt.Errorf("update tool: unknown update method %q", method)
	}
	return plan, nil
}

// Run updates mars into the resolved install directory.
func Run(ctx context.Context, cfg Config) (Plan, error) {
	return runWithReleaseDependencies(ctx, cfg, productionRunReleaseDependencies())
}

func runWithReleaseDependencies(ctx context.Context, cfg Config, deps runReleaseDependencies) (Plan, error) {
	plan, err := ResolvePlan(cfg)
	if err != nil {
		return Plan{}, err
	}
	if plan.Method == MethodSource {
		if cfg.ExactBootstrap {
			return Plan{}, ErrExactModuleBootstrap
		}
		return runSource(ctx, cfg, plan)
	}
	return runReleaseAssetsWithDependencies(ctx, cfg, plan, deps)
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
	if err := childenv.ApplyWith(cmd, "GOBIN="+plan.InstallDir); err != nil {
		return Plan{}, fmt.Errorf("update tool: child environment: %w", err)
	}
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

type runReleaseDependencies struct {
	acquire              func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error)
	replace              func(context.Context, string, verifiedMARSReleaseDownload, signedPriorExpectation) (signedReplaceResult, error)
	ensurePath           func(shellpath.Config) (shellpath.Result, error)
	captureCurrent       func(string, string) (signedPriorExpectation, error)
	readRuntimeBuildInfo func() (*runtimedebug.BuildInfo, bool)
}

func productionRunReleaseDependencies() runReleaseDependencies {
	return runReleaseDependencies{
		acquire:              fetchVerifiedMARSRelease,
		replace:              replaceVerifiedMARSReleaseExpected,
		ensurePath:           shellpath.Ensure,
		captureCurrent:       captureCurrentMARSExecutableDestination,
		readRuntimeBuildInfo: runtimedebug.ReadBuildInfo,
	}
}

func runReleaseAssetsWithDependencies(ctx context.Context, cfg Config, plan Plan, deps runReleaseDependencies) (Plan, error) {
	bootstrap := false
	if cfg.ExactBootstrap {
		if cfg.Version != plan.ReleaseTag || deps.readRuntimeBuildInfo == nil {
			return Plan{}, ErrExactModuleBootstrap
		}
		info, ok := deps.readRuntimeBuildInfo()
		if !ok || !validExactModuleBootstrapIdentity(info, plan.ReleaseTag) {
			return Plan{}, ErrExactModuleBootstrap
		}
		bootstrap = true
	}
	if cfg.DryRun {
		if cfg.SkipShellPath {
			return plan, nil
		}
		if deps.ensurePath == nil {
			return Plan{}, ErrSignedUpdateConfig
		}
		pathResult, pathErr := deps.ensurePath(shellpath.Config{InstallDir: plan.InstallDir, DryRun: true})
		if pathErr == nil {
			plan.ShellPath = pathResult
		}
		return plan, nil
	}
	if ctx == nil || deps.acquire == nil || deps.replace == nil || deps.captureCurrent == nil || (!cfg.SkipShellPath && deps.ensurePath == nil) {
		return Plan{}, ErrSignedUpdateConfig
	}
	_, latest, ok := normalizeSignedReleaseRequest(plan.ReleaseTag)
	if !ok || (!bootstrap && !validSignedCurrentIdentity(cfg.CurrentVersion, cfg.CurrentCommit, latest)) {
		return Plan{}, ErrSignedUpdateIdentity
	}
	expectedPrior := signedPriorExpectation{}
	if latest {
		captured, captureErr := deps.captureCurrent(plan.BinaryPath, cfg.CurrentCommit)
		if captureErr != nil || !captured.required {
			return Plan{}, ErrSignedUpdateDestination
		}
		expectedPrior = captured
	}
	acquisitionCurrentVersion := cfg.CurrentVersion
	acquisitionCurrentCommit := cfg.CurrentCommit
	if bootstrap {
		// An exact go-install bootstrap is not an installed signed release. Its
		// independently checked module identity replaces only the prior-release
		// replay input; the requested release remains the same exact tag.
		acquisitionCurrentVersion = ""
		acquisitionCurrentCommit = ""
	}
	download, err := deps.acquire(ctx, cfg.HTTPClient, plan.ReleaseTag, acquisitionCurrentVersion, acquisitionCurrentCommit, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return Plan{}, err
	}
	if !safeSignedUpdateAuthSource(download.authSource) {
		return Plan{}, ErrSignedReleaseMetadata
	}
	replaced, err := deps.replace(ctx, plan.InstallDir, download, expectedPrior)
	if err != nil {
		return Plan{}, err
	}
	plan.Version = strings.TrimPrefix(download.tag, "v")
	plan.ReleaseTag = download.tag
	plan.ReleaseCommit = download.fullCommit
	plan.AssetName = download.archiveName
	plan.AuthSource = download.authSource
	if replaced.releaseID != download.releaseID || replaced.tag != download.tag || replaced.fullCommit != download.fullCommit {
		return plan, ErrSignedReplaceRecovery
	}
	if cfg.SkipShellPath {
		return plan, nil
	}
	pathResult, err := deps.ensurePath(shellpath.Config{InstallDir: plan.InstallDir})
	if err != nil {
		return plan, fmt.Errorf("%w; run `mars path setup --install-dir %s` before using the new binary", ErrSignedUpdateShellPath, plan.InstallDir)
	}
	plan.ShellPath = pathResult
	return plan, nil
}

func validExactModuleBootstrapIdentity(info *runtimedebug.BuildInfo, requestedTag string) bool {
	tag, latest, ok := normalizeSignedReleaseRequest(requestedTag)
	if !ok || latest || tag != requestedTag || info == nil || info.Path != DefaultPackage ||
		info.Main.Path != releaseModulePath || info.Main.Version != tag || info.Main.Replace != nil ||
		!validExactModuleSum(info.Main.Sum) {
		return false
	}
	for _, dependency := range info.Deps {
		if dependency == nil || dependency.Replace != nil {
			return false
		}
	}
	return true
}

func validExactModuleSum(sum string) bool {
	const prefix = "h1:"
	if !strings.HasPrefix(sum, prefix) || strings.TrimSpace(sum) != sum {
		return false
	}
	encoded := strings.TrimPrefix(sum, prefix)
	digest, err := base64.StdEncoding.Strict().DecodeString(encoded)
	return err == nil && len(digest) == sha256.Size && base64.StdEncoding.EncodeToString(digest) == encoded
}

func validSignedCurrentIdentity(version, commit string, latest bool) bool {
	_, stable := exactStableReleaseVersion(version)
	if latest && !stable {
		return false
	}
	return !stable || exactCommitPattern.MatchString(commit)
}

func safeSignedUpdateAuthSource(source string) bool {
	switch source {
	case githubauth.SourceNone, githubauth.SourceEnvGHToken, githubauth.SourceEnvGitHubToken, githubauth.SourceGHCLI, githubauth.SourceConfig:
		return true
	default:
		return false
	}
}

func captureCurrentMARSExecutableDestination(destination, currentCommit string) (signedPriorExpectation, error) {
	if filepath.Base(destination) != DefaultBinary || !exactCommitPattern.MatchString(currentCommit) {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	pathInfo, err := os.Lstat(destination)
	if err != nil || !pathInfo.Mode().IsRegular() {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	executable, err := os.Executable()
	if err != nil {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	resolvedDestination, err := filepath.EvalSymlinks(destination)
	if err != nil || filepath.Clean(executable) != filepath.Clean(resolvedDestination) {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	executableInfo, err := os.Stat(executable)
	if err != nil || !os.SameFile(executableInfo, pathInfo) {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	file, err := os.Open(destination)
	if err != nil {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || openedInfo.Size() <= 0 || openedInfo.Size() > maxReleaseBinaryBytes ||
		!os.SameFile(openedInfo, pathInfo) || !os.SameFile(openedInfo, executableInfo) {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	info, err := buildinfo.Read(file)
	if err != nil || !currentMARSBuildInfoMatchesCommit(info, currentCommit) {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(maxReleaseBinaryBytes)+1))
	if err != nil || int64(len(content)) != openedInfo.Size() || len(content) > maxReleaseBinaryBytes {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	finalOpenedInfo, openedErr := file.Stat()
	finalPathInfo, pathErr := os.Lstat(destination)
	finalExecutableInfo, executableErr := os.Stat(executable)
	if openedErr != nil || pathErr != nil || executableErr != nil || !finalPathInfo.Mode().IsRegular() ||
		!os.SameFile(openedInfo, finalOpenedInfo) || !os.SameFile(openedInfo, finalPathInfo) || !os.SameFile(openedInfo, finalExecutableInfo) {
		return signedPriorExpectation{}, ErrSignedUpdateDestination
	}
	return signedPriorExpectation{required: true, digest: sha256.Sum256(content)}, nil
}

func currentMARSBuildInfoMatchesCommit(info *buildinfo.BuildInfo, commit string) bool {
	if info == nil || info.Path != DefaultPackage || info.Main.Path != releaseModulePath || info.Main.Replace != nil {
		return false
	}
	want := map[string]string{"vcs": "git", "vcs.revision": commit, "vcs.modified": "false"}
	seen := make(map[string]bool, len(want))
	for _, setting := range info.Settings {
		value, required := want[setting.Key]
		if !required {
			continue
		}
		if seen[setting.Key] || setting.Value != value {
			return false
		}
		seen[setting.Key] = true
	}
	return len(seen) == len(want)
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

func displayVersion(version string) string {
	if version == DefaultVersion || version == "main" {
		return version
	}
	return NormalizeVersion(version)
}
