/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-017-open-source-publication.md
- docs/features/F-018-goreleaser-distribution.md
*/
package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/greaveselliott/mars/internal/githubauth"
	"github.com/greaveselliott/mars/internal/shellpath"
	"github.com/stretchr/testify/require"
)

const (
	testRunCurrentCommit = "abcdef0123456789abcdef0123456789abcdef01"
	testRunReleaseCommit = "0123456789abcdef0123456789abcdef01234567"
	testExactModuleSum   = "h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="
)

func TestResolvePlanDefaultsToSignedReleaseInCurrentExecutableDir(t *testing.T) {
	plan, err := ResolvePlan(Config{DryRun: true})
	require.NoError(t, err)

	require.Equal(t, MethodReleaseAssets, plan.Method)
	require.Equal(t, DefaultVersion, plan.Version)
	require.Equal(t, DefaultVersion, plan.ReleaseTag)
	require.Empty(t, plan.Command)
	require.Empty(t, plan.AssetName, "planning must not invent a resolved archive")
	require.Empty(t, plan.AuthSource)
	require.True(t, filepath.IsAbs(plan.InstallDir))
	require.Equal(t, filepath.Join(plan.InstallDir, DefaultBinary), plan.BinaryPath)
	require.True(t, plan.DryRun)
}

func TestResolvePlanAcceptsSourceMethodVersionAndInstallDir(t *testing.T) {
	installDir := t.TempDir()
	plan, err := ResolvePlan(Config{
		Version:    "@main",
		InstallDir: installDir,
		BinaryName: "mars-dev",
		Method:     MethodSource,
	})
	require.NoError(t, err)

	require.Equal(t, MethodSource, plan.Method)
	require.Equal(t, "main", plan.Version)
	require.Equal(t, []string{"go", "install", DefaultPackage + "@main"}, plan.Command)
	require.Equal(t, installDir, plan.InstallDir)
	require.Equal(t, filepath.Join(installDir, "mars-dev"), plan.BinaryPath)
}

func TestResolvePlanSelectsSourceForMain(t *testing.T) {
	plan, err := ResolvePlan(Config{Version: "main"})
	require.NoError(t, err)

	require.Equal(t, MethodSource, plan.Method)
	require.Equal(t, []string{"go", "install", DefaultPackage + "@main"}, plan.Command)
}

func TestRunExactModuleBootstrapValidatesRunningIdentityAndSkipsShellPath(t *testing.T) {
	installDir := t.TempDir()
	download := testRunVerifiedDownload(t, "v0.69.1")
	events := make([]string, 0, 3)
	deps := runReleaseDependencies{
		readRuntimeBuildInfo: func() (*debug.BuildInfo, bool) {
			events = append(events, "identity")
			return testExactModuleBuildInfo("v0.69.1", testExactModuleSum), true
		},
		captureCurrent: func(string, string) (signedPriorExpectation, error) {
			t.Fatal("an exact bootstrap must not borrow the staged command as prior signed-release identity")
			return signedPriorExpectation{}, nil
		},
		acquire: func(_ context.Context, _ *http.Client, tag, currentVersion, currentCommit, _, _ string) (verifiedMARSReleaseDownload, error) {
			events = append(events, "acquire")
			require.Equal(t, "v0.69.1", tag)
			require.Empty(t, currentVersion, "validated module bootstrap is not a prior signed release")
			require.Empty(t, currentCommit)
			return download, nil
		},
		replace: func(_ context.Context, gotDir string, got verifiedMARSReleaseDownload, prior signedPriorExpectation) (signedReplaceResult, error) {
			events = append(events, "replace")
			require.Equal(t, installDir, gotDir)
			require.Equal(t, download, got)
			require.False(t, prior.required)
			return testRunReplaceResult(download), nil
		},
		ensurePath: func(shellpath.Config) (shellpath.Result, error) {
			t.Fatal("bootstrap skip-shell-path must not call shellpath.Ensure")
			return shellpath.Result{}, nil
		},
	}

	plan, err := runWithReleaseDependencies(context.Background(), Config{
		Version: "v0.69.1", CurrentVersion: "0.69.1", CurrentCommit: "unknown",
		InstallDir: installDir, ExactBootstrap: true, SkipShellPath: true,
	}, deps)
	require.NoError(t, err)
	require.Equal(t, []string{"identity", "acquire", "replace"}, events)
	require.Equal(t, "v0.69.1", plan.ReleaseTag)
	require.Empty(t, plan.ShellPath.InstallDir)
}

func TestExactModuleBootstrapIdentityRejectsMalformedOrSubstitutedBuilds(t *testing.T) {
	valid := testExactModuleBuildInfo("v0.69.1", testExactModuleSum)
	require.True(t, validExactModuleBootstrapIdentity(valid, "v0.69.1"))

	tests := map[string]func(*debug.BuildInfo){
		"wrong command":          func(info *debug.BuildInfo) { info.Path = "example.com/not-mars/cmd/mars" },
		"wrong module":           func(info *debug.BuildInfo) { info.Main.Path = "example.com/not-mars" },
		"wrong version":          func(info *debug.BuildInfo) { info.Main.Version = "v0.69.0" },
		"main replacement":       func(info *debug.BuildInfo) { info.Main.Replace = &debug.Module{Path: "./local"} },
		"dependency replacement": func(info *debug.BuildInfo) { info.Deps[0].Replace = &debug.Module{Path: "./local"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			info := testExactModuleBuildInfo("v0.69.1", testExactModuleSum)
			mutate(info)
			require.False(t, validExactModuleBootstrapIdentity(info, "v0.69.1"))
		})
	}
	for _, requested := range []string{"", DefaultVersion, "v0.69.0", "v0.69.1-rc.1"} {
		t.Run("request "+requested, func(t *testing.T) {
			require.False(t, validExactModuleBootstrapIdentity(valid, requested))
		})
	}
}

func TestExactModuleBootstrapSumRequiresCanonicalSHA256H1(t *testing.T) {
	require.True(t, validExactModuleSum(testExactModuleSum))
	for _, sum := range []string{
		"", "h1:", "h1:testsum", "sha256:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=",
		"h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU",                  // missing padding
		"h1:47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",                 // URL alphabet
		"h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFV=",                 // non-canonical pad bits
		"h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=\n",               // whitespace
		"h1:QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFB", // wrong digest size
	} {
		t.Run(sum, func(t *testing.T) { require.False(t, validExactModuleSum(sum)) })
	}
}

func TestExactModuleBootstrapDoesNotWeakenNormalIdentityAdmission(t *testing.T) {
	download := testRunVerifiedDownload(t, "v0.69.1")
	deps := testRunReleaseDependencies(t, &[]string{}, download)
	deps.acquire = func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error) {
		t.Fatal("normal exact release admission must reject an unknown stable commit before acquisition")
		return verifiedMARSReleaseDownload{}, nil
	}
	_, err := runWithReleaseDependencies(context.Background(), Config{
		Version: "v0.69.1", CurrentVersion: "0.69.1", CurrentCommit: "unknown",
		InstallDir: t.TempDir(), SkipShellPath: true,
	}, deps)
	require.ErrorIs(t, err, ErrSignedUpdateIdentity)

	_, err = runWithReleaseDependencies(context.Background(), Config{
		Version: "main", Method: MethodSource, InstallDir: t.TempDir(), ExactBootstrap: true,
	}, deps)
	require.ErrorIs(t, err, ErrExactModuleBootstrap)

	deps.readRuntimeBuildInfo = func() (*debug.BuildInfo, bool) {
		return testExactModuleBuildInfo("v0.69.1", testExactModuleSum), true
	}
	_, err = runWithReleaseDependencies(context.Background(), Config{
		Version: "0.69.1", InstallDir: t.TempDir(), ExactBootstrap: true, SkipShellPath: true,
	}, deps)
	require.ErrorIs(t, err, ErrExactModuleBootstrap, "bootstrap input must be the exact canonical release tag")
}

func TestRunReleaseWiresAcquisitionReplacementAndShellPath(t *testing.T) {
	installDir := t.TempDir()
	client := &http.Client{}
	download := testRunVerifiedDownload(t, "v0.69.0")
	events := make([]string, 0, 4)
	wantRequestedTag := DefaultVersion
	wantPrior := testRunPriorExpectation()
	deps := runReleaseDependencies{
		captureCurrent: func(path, commit string) (signedPriorExpectation, error) {
			events = append(events, "destination")
			require.Equal(t, filepath.Join(installDir, DefaultBinary), path)
			require.Equal(t, testRunCurrentCommit, commit)
			return wantPrior, nil
		},
		acquire: func(_ context.Context, gotClient *http.Client, requestedTag, currentVersion, currentCommit, goos, goarch string) (verifiedMARSReleaseDownload, error) {
			events = append(events, "acquire")
			require.Same(t, client, gotClient)
			require.Equal(t, wantRequestedTag, requestedTag)
			require.Equal(t, "0.68.49", currentVersion)
			require.Equal(t, testRunCurrentCommit, currentCommit)
			require.Equal(t, runtime.GOOS, goos)
			require.Equal(t, runtime.GOARCH, goarch)
			return download, nil
		},
		replace: func(_ context.Context, gotDir string, got verifiedMARSReleaseDownload, prior signedPriorExpectation) (signedReplaceResult, error) {
			events = append(events, "replace")
			require.Equal(t, installDir, gotDir)
			require.Equal(t, download, got)
			require.Equal(t, wantPrior, prior)
			return testRunReplaceResult(download), nil
		},
		ensurePath: func(cfg shellpath.Config) (shellpath.Result, error) {
			events = append(events, "path")
			require.Equal(t, installDir, cfg.InstallDir)
			require.False(t, cfg.DryRun)
			return shellpath.Result{InstallDir: installDir, Message: "configured shell PATH"}, nil
		},
	}

	plan, err := runWithReleaseDependencies(context.Background(), Config{
		Version:        DefaultVersion,
		CurrentVersion: "0.68.49",
		CurrentCommit:  testRunCurrentCommit,
		InstallDir:     installDir,
		HTTPClient:     client,
	}, deps)
	require.NoError(t, err)
	require.Equal(t, []string{"destination", "acquire", "replace", "path"}, events)
	require.Equal(t, "0.69.0", plan.Version)
	require.Equal(t, download.tag, plan.ReleaseTag)
	require.Equal(t, download.fullCommit, plan.ReleaseCommit)
	require.Equal(t, download.archiveName, plan.AssetName)
	require.Equal(t, githubauth.SourceGHCLI, plan.AuthSource)
	require.Equal(t, filepath.Join(installDir, DefaultBinary), plan.BinaryPath)
	encoded, marshalErr := json.Marshal(plan)
	require.NoError(t, marshalErr)
	require.NotContains(t, strings.ToLower(string(encoded)), "http")
	require.NotContains(t, string(encoded), "download_url")
	require.NotContains(t, string(encoded), "checksums_url")
	require.NotContains(t, string(encoded), "requires_github_auth")

	events = events[:0]
	wantRequestedTag = "v0.69.0"
	wantPrior = signedPriorExpectation{}
	deps.captureCurrent = func(string, string) (signedPriorExpectation, error) {
		t.Fatal("exact versions must not borrow the running binary's identity")
		return signedPriorExpectation{}, nil
	}
	plan, err = runWithReleaseDependencies(context.Background(), Config{
		Version: "v0.69.0", CurrentVersion: "0.68.49", CurrentCommit: testRunCurrentCommit,
		InstallDir: installDir, HTTPClient: client, SkipShellPath: true,
	}, deps)
	require.NoError(t, err)
	require.Equal(t, []string{"acquire", "replace"}, events)
	require.Empty(t, plan.ShellPath.InstallDir)
}

func TestRunReleaseStopsAtFailedStage(t *testing.T) {
	installDir := t.TempDir()
	download := testRunVerifiedDownload(t, "v0.69.0")
	baseConfig := Config{
		Version: DefaultVersion, CurrentVersion: "0.68.49", CurrentCommit: testRunCurrentCommit,
		InstallDir: installDir,
	}

	t.Run("acquisition", func(t *testing.T) {
		events := make([]string, 0, 2)
		deps := testRunReleaseDependencies(t, &events, download)
		deps.acquire = func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error) {
			events = append(events, "acquire")
			return verifiedMARSReleaseDownload{}, ErrSignedReleaseDownloadEvidence
		}
		_, err := runWithReleaseDependencies(context.Background(), baseConfig, deps)
		require.ErrorIs(t, err, ErrSignedReleaseDownloadEvidence)
		require.Equal(t, []string{"destination", "acquire"}, events)
	})

	t.Run("replacement", func(t *testing.T) {
		events := make([]string, 0, 3)
		deps := testRunReleaseDependencies(t, &events, download)
		deps.replace = func(context.Context, string, verifiedMARSReleaseDownload, signedPriorExpectation) (signedReplaceResult, error) {
			events = append(events, "replace")
			return signedReplaceResult{}, ErrSignedReplaceFailed
		}
		_, err := runWithReleaseDependencies(context.Background(), baseConfig, deps)
		require.ErrorIs(t, err, ErrSignedReplaceFailed)
		require.Equal(t, []string{"destination", "acquire", "replace"}, events)
	})

	t.Run("replacement identity mismatch", func(t *testing.T) {
		events := make([]string, 0, 3)
		deps := testRunReleaseDependencies(t, &events, download)
		deps.replace = func(context.Context, string, verifiedMARSReleaseDownload, signedPriorExpectation) (signedReplaceResult, error) {
			events = append(events, "replace")
			mismatched := testRunReplaceResult(download)
			mismatched.fullCommit = testRunCurrentCommit
			return mismatched, nil
		}
		plan, err := runWithReleaseDependencies(context.Background(), baseConfig, deps)
		require.ErrorIs(t, err, ErrSignedReplaceRecovery)
		require.Equal(t, download.tag, plan.ReleaseTag)
		require.Equal(t, download.fullCommit, plan.ReleaseCommit)
		require.Equal(t, download.archiveName, plan.AssetName)
		require.Equal(t, []string{"destination", "acquire", "replace"}, events)
	})

	t.Run("shell path after commit", func(t *testing.T) {
		events := make([]string, 0, 4)
		deps := testRunReleaseDependencies(t, &events, download)
		deps.ensurePath = func(shellpath.Config) (shellpath.Result, error) {
			events = append(events, "path")
			return shellpath.Result{}, errors.New("injected profile failure")
		}
		plan, err := runWithReleaseDependencies(context.Background(), baseConfig, deps)
		require.ErrorIs(t, err, ErrSignedUpdateShellPath)
		require.Contains(t, err.Error(), "replacement committed")
		require.Contains(t, err.Error(), "mars path setup --install-dir")
		require.Equal(t, download.tag, plan.ReleaseTag, "the committed replacement must remain visible in the returned plan")
		require.Equal(t, []string{"destination", "acquire", "replace", "path"}, events)
	})
}

func TestRunReleaseDryRunHasNoAuthority(t *testing.T) {
	for _, version := range []string{DefaultVersion, "v0.69.0"} {
		t.Run(version, func(t *testing.T) {
			installDir := t.TempDir()
			events := make([]string, 0, 1)
			deps := runReleaseDependencies{
				acquire: func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error) {
					t.Fatal("dry-run must not acquire a release")
					return verifiedMARSReleaseDownload{}, nil
				},
				replace: func(context.Context, string, verifiedMARSReleaseDownload, signedPriorExpectation) (signedReplaceResult, error) {
					t.Fatal("dry-run must not replace a binary")
					return signedReplaceResult{}, nil
				},
				captureCurrent: func(string, string) (signedPriorExpectation, error) {
					t.Fatal("dry-run must not inspect the current executable")
					return signedPriorExpectation{}, nil
				},
				ensurePath: func(cfg shellpath.Config) (shellpath.Result, error) {
					events = append(events, "path-plan")
					require.True(t, cfg.DryRun)
					return shellpath.Result{InstallDir: installDir, DryRun: true}, nil
				},
			}
			plan, err := runWithReleaseDependencies(context.Background(), Config{
				Version: version, InstallDir: installDir, DryRun: true,
				HTTPClient: &http.Client{Transport: failRoundTripper{t: t}},
			}, deps)
			require.NoError(t, err)
			require.Equal(t, []string{"path-plan"}, events)
			require.Empty(t, plan.AssetName)
			require.Empty(t, plan.ReleaseCommit)
			require.Empty(t, plan.AuthSource)
			require.NoFileExists(t, filepath.Join(installDir, ".mars-update.lock"))
			require.NoDirExists(t, filepath.Join(installDir, ".mars-update.transaction"))
			encoded, marshalErr := json.Marshal(plan)
			require.NoError(t, marshalErr)
			require.NotContains(t, strings.ToLower(string(encoded)), "http")
		})
	}
}

func TestRunSourceAndMainBypassSignedPipeline(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/zsh")
	for _, cfg := range []Config{
		{Version: "main", InstallDir: t.TempDir(), BinaryName: "mars-main", DryRun: true},
		{Version: "v0.69.0", InstallDir: t.TempDir(), BinaryName: "mars-source", Method: MethodSource, DryRun: true},
	} {
		plan, err := runWithReleaseDependencies(context.Background(), cfg, runReleaseDependencies{
			acquire: func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error) {
				t.Fatal("source mode must not acquire a release")
				return verifiedMARSReleaseDownload{}, nil
			},
			replace: func(context.Context, string, verifiedMARSReleaseDownload, signedPriorExpectation) (signedReplaceResult, error) {
				t.Fatal("source mode must not replace through the signed release path")
				return signedReplaceResult{}, nil
			},
		})
		require.NoError(t, err)
		require.Equal(t, MethodSource, plan.Method)
		require.Equal(t, []string{"go", "install", DefaultPackage + "@" + strings.TrimPrefix(cfg.Version, "@")}, plan.Command)
		require.Empty(t, plan.AssetName)
		require.Empty(t, plan.AuthSource)
	}
}

func TestRunReleaseIdentityAndDestinationAdmission(t *testing.T) {
	installDir := t.TempDir()
	for _, test := range []struct {
		name        string
		cfg         Config
		self        bool
		wantErr     error
		wantAcquire bool
	}{
		{name: "latest dev identity", cfg: Config{Version: DefaultVersion, CurrentVersion: "0.69.0-dev", CurrentCommit: "unknown"}, self: true, wantErr: ErrSignedUpdateIdentity},
		{name: "latest malformed commit", cfg: Config{Version: DefaultVersion, CurrentVersion: "0.68.49", CurrentCommit: "unknown"}, self: true, wantErr: ErrSignedUpdateIdentity},
		{name: "latest foreign destination", cfg: Config{Version: DefaultVersion, CurrentVersion: "0.68.49", CurrentCommit: testRunCurrentCommit}, wantErr: ErrSignedUpdateDestination},
		{name: "exact stable malformed commit", cfg: Config{Version: "v0.69.0", CurrentVersion: "0.68.49", CurrentCommit: "unknown"}, self: true, wantErr: ErrSignedUpdateIdentity},
		{name: "exact tag from dev build", cfg: Config{Version: "v0.69.0", CurrentVersion: "0.69.0-dev", CurrentCommit: "unknown"}, wantAcquire: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := make([]string, 0, 3)
			download := testRunVerifiedDownload(t, "v0.69.0")
			deps := testRunReleaseDependencies(t, &events, download)
			deps.captureCurrent = func(string, string) (signedPriorExpectation, error) {
				events = append(events, "destination")
				if !test.self {
					return signedPriorExpectation{}, ErrSignedUpdateDestination
				}
				return testRunPriorExpectation(), nil
			}
			test.cfg.InstallDir = installDir
			test.cfg.SkipShellPath = true
			_, err := runWithReleaseDependencies(context.Background(), test.cfg, deps)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.NotContains(t, events, "replace")
				require.NotContains(t, events, "acquire")
				return
			}
			require.NoError(t, err)
			require.True(t, test.wantAcquire)
			require.Equal(t, []string{"acquire", "replace"}, events)
		})
	}
}

func TestRunReleaseRejectsCustomBinaryBeforeAcquisition(t *testing.T) {
	called := false
	_, err := runWithReleaseDependencies(context.Background(), Config{
		Version: "v0.69.0", CurrentVersion: "0.68.49", CurrentCommit: testRunCurrentCommit,
		InstallDir: t.TempDir(), BinaryName: "mars-dev",
	}, runReleaseDependencies{acquire: func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error) {
		called = true
		return verifiedMARSReleaseDownload{}, nil
	}})
	require.ErrorIs(t, err, ErrSignedUpdateConfig)
	require.False(t, called)
}

func testRunReleaseDependencies(t *testing.T, events *[]string, download verifiedMARSReleaseDownload) runReleaseDependencies {
	t.Helper()
	return runReleaseDependencies{
		captureCurrent: func(string, string) (signedPriorExpectation, error) {
			*events = append(*events, "destination")
			return testRunPriorExpectation(), nil
		},
		acquire: func(context.Context, *http.Client, string, string, string, string, string) (verifiedMARSReleaseDownload, error) {
			*events = append(*events, "acquire")
			return download, nil
		},
		replace: func(_ context.Context, _ string, got verifiedMARSReleaseDownload, prior signedPriorExpectation) (signedReplaceResult, error) {
			*events = append(*events, "replace")
			require.Equal(t, download, got)
			if prior.required {
				require.Equal(t, testRunPriorExpectation(), prior)
			}
			return testRunReplaceResult(download), nil
		},
		ensurePath: func(shellpath.Config) (shellpath.Result, error) {
			*events = append(*events, "path")
			return shellpath.Result{}, nil
		},
	}
}

func testRunVerifiedDownload(t *testing.T, tag string) verifiedMARSReleaseDownload {
	t.Helper()
	archiveName, _, _, ok := marsReleaseArchiveIdentity(tag, testRunReleaseCommit, runtime.GOOS, runtime.GOARCH)
	require.True(t, ok)
	return verifiedMARSReleaseDownload{
		releaseID: 690, tag: tag, fullCommit: testRunReleaseCommit, archiveName: archiveName,
		authSource: githubauth.SourceGHCLI, candidate: []byte("opaque-verified-candidate"),
	}
}

func testRunReplaceResult(download verifiedMARSReleaseDownload) signedReplaceResult {
	return signedReplaceResult{releaseID: download.releaseID, tag: download.tag, fullCommit: download.fullCommit, replacedExisting: true}
}

func testRunPriorExpectation() signedPriorExpectation {
	return signedPriorExpectation{required: true, digest: [32]byte{1}}
}

func testExactModuleBuildInfo(version, sum string) *debug.BuildInfo {
	return &debug.BuildInfo{
		Path: DefaultPackage,
		Main: debug.Module{Path: releaseModulePath, Version: version, Sum: sum},
		Deps: []*debug.Module{{
			Path: "github.com/spf13/cobra", Version: "v1.10.1",
			Sum: "h1:7SmJGaEXJhHeu7nFTnT/PQmN4hIPpLqZlXGmhpRQaok=",
		}},
	}
}

type failRoundTripper struct{ t *testing.T }

func (f failRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	f.t.Fatal("dry-run must not perform HTTP")
	return nil, errors.New("unexpected HTTP")
}

func TestCaptureCurrentMARSExecutableDestinationRejectsAliases(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	aliasDir := t.TempDir()
	alias := filepath.Join(aliasDir, DefaultBinary)
	require.NoError(t, os.Link(executable, alias))
	_, captureErr := captureCurrentMARSExecutableDestination(alias, testRunCurrentCommit)
	require.ErrorIs(t, captureErr, ErrSignedUpdateDestination, "a sibling hard link must not borrow the running path's identity")
}
