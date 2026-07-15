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
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

func TestVerifyLocalAssetsAcceptsCanonicalAndLegacyBinaryNames(t *testing.T) {
	t.Parallel()
	dist := t.TempDir()
	assets := make([]string, 0, len(AssetTargets)*2)

	for _, target := range AssetTargets {
		name := fmt.Sprintf("mars-%s-%s", target.GOOS, target.GOARCH)
		canonical := filepath.Join(dist, name)
		require.NoError(t, os.WriteFile(canonical, []byte("binary "+name), 0o755))
		assets = append(assets, canonical)

		legacyName := fmt.Sprintf("mars-harness-%s-%s", target.GOOS, target.GOARCH)
		legacy := filepath.Join(dist, legacyName)
		require.NoError(t, copyFile(canonical, legacy))
		assets = append(assets, legacy)
	}
	require.NoError(t, writeChecksums(filepath.Join(dist, "checksums.txt"), assets))

	report, err := VerifyLocalAssets(dist, "v1.2.3")
	require.NoError(t, err)
	require.True(t, report.OK, "missing: %v", report.Missing)
	require.Equal(t, "v1.2.3", report.TagName)
	require.Equal(t, "1.2.3", report.Version)
	require.Empty(t, report.Missing)
	require.ElementsMatch(t, selfupdate.ExpectedReleaseAssetNames(), report.Found)

	require.NoError(t, os.WriteFile(filepath.Join(dist, "mars-darwin-arm64"), []byte("tampered"), 0o755))
	report, err = VerifyLocalAssets(dist, "1.2.3")
	require.NoError(t, err)
	require.False(t, report.OK)
	require.Contains(t, report.Missing, "valid checksums.txt")
}

func TestLocalReleaseAssetContractBindsExactNamesSizesAndDigests(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	require.Len(t, contract, 9)
	require.True(t, sort.SliceIsSorted(contract, func(i, j int) bool { return contract[i].Name < contract[j].Name }))
	for _, asset := range contract {
		data, err := os.ReadFile(asset.Path)
		require.NoError(t, err)
		sum := sha256.Sum256(data)
		require.Equal(t, int64(len(data)), asset.Size)
		require.Equal(t, fmt.Sprintf("%x", sum), asset.SHA256)
	}

	_, err := localReleaseAssetContract(contractPaths(contract[:8]))
	require.ErrorContains(t, err, "mirror_incomplete")
	require.ErrorContains(t, err, contract[8].Name)
}

func TestLocalReleaseAssetContractRejectsSymlinkedAsset(t *testing.T) {
	contract := testLocalReleaseAssetContract(t)
	linked := contract[0].Path
	real := linked + ".real"
	require.NoError(t, os.Rename(linked, real))
	require.NoError(t, os.Symlink(real, linked))

	_, err := localReleaseAssetContract(contractPaths(contract))
	require.ErrorContains(t, err, "asset must be a non-empty regular file")
}

func TestLocalReleaseAssetContractRejectsSymlinkedParent(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "release-link")
	require.NoError(t, os.Symlink(realDir, linkDir))
	var paths []string
	for _, name := range selfupdate.ExpectedReleaseAssetNames() {
		require.NoError(t, os.WriteFile(filepath.Join(realDir, name), []byte(name), 0o600))
		paths = append(paths, filepath.Join(linkDir, name))
	}
	_, err := localReleaseAssetContract(paths)
	require.ErrorContains(t, err, "asset parent must be a real directory")
}

func TestRevalidateLocalReleaseAssetRejectsPathSwap(t *testing.T) {
	contract := testLocalReleaseAssetContract(t)
	asset := contract[0]
	replacement := asset.Path + ".replacement"
	require.NoError(t, os.WriteFile(replacement, []byte("different bytes"), 0o600))
	require.NoError(t, os.Rename(replacement, asset.Path))
	require.ErrorContains(t, revalidateLocalReleaseAsset(asset), "changed after contract creation")
}

func TestSnapshotReleaseAssetContractSealsAndRevalidatesBytes(t *testing.T) {
	contract := testLocalReleaseAssetContract(t)
	snapshot, dir, err := snapshotReleaseAssetContract(contract)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	require.Len(t, snapshot, len(contract))
	for i, asset := range snapshot {
		require.NotEqual(t, contract[i].Path, asset.Path)
		info, err := os.Stat(asset.Path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o400), info.Mode().Perm())
		require.NoError(t, revalidateLocalReleaseAsset(asset))
	}
}

func TestReconcileGitHubAssetsConvergesFromZeroThroughFourToNine(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	sequence := []githubReleaseMetadata{
		testGitHubMetadata("v1.2.3", nil),
		testGitHubMetadata("v1.2.3", contract[:4]),
		testGitHubMetadata("v1.2.3", contract),
	}
	fetch, calls := testGitHubMetadataSequence(sequence)
	var uploads []string
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 3, 0, fetch,
		func(_ context.Context, asset localReleaseAsset, clobber bool) error {
			require.False(t, clobber)
			uploads = append(uploads, asset.Name)
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, 9, verified)
	require.Equal(t, 3, *calls)
	require.Equal(t, []string{contract[0].Name, contract[4].Name}, uploads)
}

func TestReconcileGitHubAssetsFailsPermanentFourOfNine(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	partial := testGitHubMetadata("v1.2.3", contract[:4])
	fetch, calls := testGitHubMetadataSequence([]githubReleaseMetadata{partial})
	var uploads []string
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 8, 0, fetch,
		func(_ context.Context, asset localReleaseAsset, clobber bool) error {
			require.False(t, clobber)
			uploads = append(uploads, asset.Name)
			return nil
		})
	require.Zero(t, verified)
	require.ErrorContains(t, err, "mirror_incomplete")
	require.ErrorContains(t, err, "metadata check 8/8")
	require.ErrorContains(t, err, "missing=")
	require.Equal(t, 8, *calls)
	require.ElementsMatch(t, sortedContractNames(contract[4:]), uploads)
}

func TestReconcileGitHubAssetsIsIdempotentForVerifiedNineOfNine(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	fetch, calls := testGitHubMetadataSequence([]githubReleaseMetadata{testGitHubMetadata("v1.2.3", contract)})
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 3, 0, fetch,
		func(context.Context, localReleaseAsset, bool) error {
			t.Fatal("verified remote assets must not be uploaded again")
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, 9, verified)
	require.Equal(t, 1, *calls)
}

func TestReconcileGitHubAssetsClobbersOnlyProvenMismatch(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*githubReleaseAssetState)
	}{
		{name: "wrong digest", mutate: func(asset *githubReleaseAssetState) { asset.Digest = "sha256:" + strings.Repeat("0", 64) }},
		{name: "wrong size", mutate: func(asset *githubReleaseAssetState) { asset.Size++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			contract := testLocalReleaseAssetContract(t)
			initial := testGitHubMetadata("v1.2.3", contract)
			tc.mutate(&initial.Assets[3])
			fetch, _ := testGitHubMetadataSequence([]githubReleaseMetadata{initial, testGitHubMetadata("v1.2.3", contract)})
			var uploads []string
			verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 2, 0, fetch,
				func(_ context.Context, asset localReleaseAsset, clobber bool) error {
					require.True(t, clobber)
					uploads = append(uploads, asset.Name)
					return nil
				})
			require.NoError(t, err)
			require.Equal(t, 9, verified)
			require.Equal(t, []string{contract[3].Name}, uploads)
		})
	}
}

func TestReconcileGitHubAssetsRejectsUnsafeOrUnverifiableRemoteState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		mutate     func(*githubReleaseMetadata)
		wantDetail string
	}{
		{name: "pending", mutate: func(metadata *githubReleaseMetadata) { metadata.Assets[0].State = "new" }, wantDetail: "pending="},
		{name: "missing digest", mutate: func(metadata *githubReleaseMetadata) { metadata.Assets[0].Digest = "" }, wantDetail: "unverifiable="},
		{name: "duplicate", mutate: func(metadata *githubReleaseMetadata) { metadata.Assets = append(metadata.Assets, metadata.Assets[0]) }, wantDetail: "duplicate="},
		{name: "extra", mutate: func(metadata *githubReleaseMetadata) {
			metadata.Assets = append(metadata.Assets, githubReleaseAssetState{Name: "unexpected", State: "uploaded", Size: 1, Digest: "sha256:" + strings.Repeat("a", 64)})
		}, wantDetail: "extra=<redacted-asset-name>"},
		{name: "non-exact asset name", mutate: func(metadata *githubReleaseMetadata) {
			metadata.Assets[0].Name += " "
		}, wantDetail: "extra="},
		{name: "non-exact uploaded state", mutate: func(metadata *githubReleaseMetadata) {
			metadata.Assets[0].State = " uploaded"
		}, wantDetail: "pending="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			contract := testLocalReleaseAssetContract(t)
			metadata := testGitHubMetadata("v1.2.3", contract)
			tc.mutate(&metadata)
			fetch, _ := testGitHubMetadataSequence([]githubReleaseMetadata{metadata, metadata})
			verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 2, 0, fetch,
				func(context.Context, localReleaseAsset, bool) error {
					t.Fatal("unsafe or unverifiable assets must not be overwritten")
					return nil
				})
			require.Zero(t, verified)
			require.ErrorContains(t, err, "mirror_incomplete")
			require.ErrorContains(t, err, tc.wantDetail)
		})
	}
}

func TestReconcileGitHubAssetsRejectsNonExactTagWithoutTranscribingControls(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	metadata := testGitHubMetadata("v1.2.3\nghp_SECRET", contract)
	fetch, _ := testGitHubMetadataSequence([]githubReleaseMetadata{metadata})
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 1, 0, fetch,
		func(context.Context, localReleaseAsset, bool) error {
			t.Fatal("non-exact tag must not upload")
			return nil
		})
	require.Zero(t, verified)
	require.ErrorContains(t, err, "does not exactly match")
	require.NotContains(t, err.Error(), "\n")
	require.NotContains(t, err.Error(), "ghp_SECRET")
}

func TestReconcileGitHubAssetsHonorsCancellation(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetch, _ := testGitHubMetadataSequence([]githubReleaseMetadata{testGitHubMetadata("v1.2.3", nil)})
	var uploads int
	verified, err := reconcileGitHubAssets(ctx, "v1.2.3", contract, 3, 0, fetch,
		func(context.Context, localReleaseAsset, bool) error {
			uploads++
			cancel()
			return nil
		})
	require.Zero(t, verified)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "verification canceled")
	require.Equal(t, 1, uploads)
}

func TestReconcileGitHubAssetsAttributesUploadFailure(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	fetch, _ := testGitHubMetadataSequence([]githubReleaseMetadata{testGitHubMetadata("v1.2.3", nil)})
	wantName := sortedContractNames(contract)[0]
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 2, 0, fetch,
		func(_ context.Context, asset localReleaseAsset, _ bool) error {
			return errors.New("simulated upload failure for " + asset.Name)
		})
	require.Zero(t, verified)
	require.ErrorContains(t, err, wantName)
	require.NotContains(t, err.Error(), "simulated upload failure")
}

func TestReconcileGitHubAssetsRejectsCancellationDuringCompleteFetch(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	ctx, cancel := context.WithCancel(context.Background())
	verified, err := reconcileGitHubAssets(ctx, "v1.2.3", contract, 1, 0,
		func(context.Context) (githubReleaseMetadata, error) {
			cancel()
			return testGitHubMetadata("v1.2.3", contract), nil
		},
		func(context.Context, localReleaseAsset, bool) error {
			t.Fatal("canceled complete fetch must not upload")
			return nil
		})
	require.Zero(t, verified)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "canceled after metadata fetch")
}

func TestReconcileGitHubAssetsUsesFirstSuccessfulSnapshotAfterMetadataError(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	partial := testGitHubMetadata("v1.2.3", contract[:4])
	complete := testGitHubMetadata("v1.2.3", contract)
	call := 0
	fetch := func(context.Context) (githubReleaseMetadata, error) {
		call++
		switch call {
		case 1:
			return githubReleaseMetadata{}, errors.New("transient metadata failure")
		case 2:
			return partial, nil
		default:
			return complete, nil
		}
	}
	var uploaded []string
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 3, 0, fetch,
		func(_ context.Context, asset localReleaseAsset, clobber bool) error {
			require.False(t, clobber)
			uploaded = append(uploaded, asset.Name)
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, 9, verified)
	require.Equal(t, []string{contract[4].Name}, uploaded)
}

func TestReconcileGitHubAssetsAcceptsVerifiedPostconditionAfterUploadTransportError(t *testing.T) {
	t.Parallel()
	contract := testLocalReleaseAssetContract(t)
	initial := testGitHubMetadata("v1.2.3", contract[1:])
	complete := testGitHubMetadata("v1.2.3", contract)
	fetch, _ := testGitHubMetadataSequence([]githubReleaseMetadata{initial, complete})
	verified, err := reconcileGitHubAssets(context.Background(), "v1.2.3", contract, 2, 0, fetch,
		func(_ context.Context, asset localReleaseAsset, clobber bool) error {
			require.Equal(t, contract[0].Name, asset.Name)
			require.False(t, clobber)
			return errors.New("connection closed after request body")
		})
	require.NoError(t, err)
	require.Equal(t, 9, verified)
}

func TestValidateGitHubMirrorIdentityRejectsUnsafeRepoAndTag(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateGitHubMirrorIdentity("owner/project", "v1.2.3"))
	for _, repo := range []string{"owner", "owner/project/extra", "../project", "owner/proj ect"} {
		require.Error(t, validateGitHubMirrorIdentity(repo, "v1.2.3"), repo)
	}
	for _, tag := range []string{"1.2.3", "v1.2", "v1.2.3/evil", "v1.2.3\n--clobber"} {
		require.Error(t, validateGitHubMirrorIdentity("owner/project", tag), tag)
	}
}

func TestVerifyRemoteTagCommitUsesExactCommitResolution(t *testing.T) {
	t.Parallel()
	const commit = "0123456789abcdef0123456789abcdef01234567"
	const tagObject = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, kind := range []string{"lightweight", "annotated"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			calls := 0
			runner := scriptedReleaseRunner(t, func(name string, args []string) releaseCommandResult {
				require.Equal(t, "gh", name)
				calls++
				if calls == 1 {
					require.Equal(t, []string{"api", "--method", "GET", "repos/owner/project/git/ref/tags/v1.2.3"}, args)
					if kind == "lightweight" {
						return releaseCommandResult{stdout: `{"object":{"type":"commit","sha":"` + commit + `"}}`}
					}
					return releaseCommandResult{stdout: `{"object":{"type":"tag","sha":"` + tagObject + `"}}`}
				}
				require.Equal(t, []string{"api", "--method", "GET", "repos/owner/project/git/tags/" + tagObject}, args)
				return releaseCommandResult{stdout: `{"object":{"type":"commit","sha":"` + commit + `"}}`}
			})
			require.NoError(t, verifyRemoteTagCommit(context.Background(), runner, t.TempDir(), "owner/project", "v1.2.3", commit))
			if kind == "lightweight" {
				require.Equal(t, 1, calls)
			} else {
				require.Equal(t, 2, calls)
			}
		})
	}
}

func TestVerifyRemoteTagCommitRejectsWrongAndMissingResolution(t *testing.T) {
	t.Parallel()
	const commit = "0123456789abcdef0123456789abcdef01234567"
	t.Run("wrong", func(t *testing.T) {
		runner := scriptedReleaseRunner(t, func(string, []string) releaseCommandResult {
			return releaseCommandResult{stdout: `{"object":{"type":"commit","sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`}
		})
		err := verifyRemoteTagCommit(context.Background(), runner, t.TempDir(), "owner/project", "v1.2.3", commit)
		require.ErrorContains(t, err, "different commit")
	})
	t.Run("missing", func(t *testing.T) {
		runner := scriptedReleaseRunner(t, func(string, []string) releaseCommandResult {
			return releaseCommandResult{stderr: "gh: Not Found (HTTP 404)", exitCode: 1}
		})
		err := verifyRemoteTagCommit(context.Background(), runner, t.TempDir(), "owner/project", "v1.2.3", commit)
		require.ErrorContains(t, err, "HTTP 404")
	})
}

func TestEnsureGitHubReleaseCreatesOnlyOnExact404WithVerifyTag(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "CHANGELOG.md"), []byte("## [1.2.3] - 2026-07-13\n\nNotes.\n"), 0o600))
	var commands [][]string
	runner := scriptedReleaseRunner(t, func(name string, args []string) releaseCommandResult {
		commands = append(commands, append([]string{name}, args...))
		if len(args) > 0 && args[0] == "api" {
			require.Equal(t, "repos/owner/project/releases/tags/v1.2.3", args[3])
			return releaseCommandResult{
				stdout:   `{"message":"Not Found","status":"404"}`,
				stderr:   "gh: Not Found (HTTP 404)",
				exitCode: 1,
			}
		}
		return releaseCommandResult{}
	})
	require.NoError(t, ensureGitHubRelease(context.Background(), runner, repo, "owner/project", "v1.2.3"))
	require.Len(t, commands, 2)
	require.Equal(t, "release", commands[1][1])
	require.Equal(t, "create", commands[1][2])
	require.Contains(t, commands[1], "--verify-tag")
}

func TestEnsureGitHubReleaseDoesNotCreateOnAuthServerOrNetworkFailure(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{name: "unauthorized", stderr: "gh: token ghp_SECRET rejected (HTTP 401)", want: "HTTP 401"},
		{name: "forbidden", stderr: "gh: forbidden (HTTP 403)", want: "HTTP 403"},
		{name: "server", stderr: "gh: upstream body (HTTP 503)", want: "HTTP 503"},
		{name: "hostile_body_cannot_grant_create", stdout: `{"message":"(HTTP 404)"}`, stderr: "gh: upstream body (HTTP 503)", want: "HTTP 503"},
		{name: "network", stderr: "dial tcp: token ghp_SECRET", want: "GitHub API"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			created := false
			runner := scriptedReleaseRunner(t, func(_ string, args []string) releaseCommandResult {
				if len(args) > 1 && args[0] == "release" && args[1] == "create" {
					created = true
				}
				return releaseCommandResult{stdout: tc.stdout, stderr: tc.stderr, exitCode: 1}
			})
			err := ensureGitHubRelease(context.Background(), runner, t.TempDir(), "owner/project", "v1.2.3")
			require.ErrorContains(t, err, tc.want)
			require.NotContains(t, err.Error(), "ghp_SECRET")
			require.False(t, created)
		})
	}
}

func TestEnsureGitHubReleaseRejectsMetadataTagMismatchWithoutMutation(t *testing.T) {
	t.Parallel()
	created := false
	runner := scriptedReleaseRunner(t, func(_ string, args []string) releaseCommandResult {
		if len(args) > 1 && args[0] == "release" && args[1] == "create" {
			created = true
		}
		return releaseCommandResult{stdout: `{"tag_name":"v9.9.9","assets":[]}`}
	})
	err := ensureGitHubRelease(context.Background(), runner, t.TempDir(), "owner/project", "v1.2.3")
	require.ErrorContains(t, err, "tag mismatch")
	require.False(t, created)
}

type releaseCommandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func scriptedReleaseRunner(t *testing.T, handler func(string, []string) releaseCommandResult) func(context.Context, string, ...string) *exec.Cmd {
	t.Helper()
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		result := handler(name, args)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestReleaseCommandHelperProcess", "--")
		cmd.Env = append(os.Environ(),
			"MARS_RELEASE_COMMAND_HELPER=1",
			"MARS_RELEASE_COMMAND_STDOUT="+base64.StdEncoding.EncodeToString([]byte(result.stdout)),
			"MARS_RELEASE_COMMAND_STDERR="+base64.StdEncoding.EncodeToString([]byte(result.stderr)),
			"MARS_RELEASE_COMMAND_EXIT="+strconv.Itoa(result.exitCode),
		)
		return cmd
	}
}

func TestReleaseCommandHelperProcess(t *testing.T) {
	if os.Getenv("MARS_RELEASE_COMMAND_HELPER") != "1" {
		return
	}
	stdout, _ := base64.StdEncoding.DecodeString(os.Getenv("MARS_RELEASE_COMMAND_STDOUT"))
	stderr, _ := base64.StdEncoding.DecodeString(os.Getenv("MARS_RELEASE_COMMAND_STDERR"))
	_, _ = os.Stdout.Write(stdout)
	_, _ = os.Stderr.Write(stderr)
	exitCode, _ := strconv.Atoi(os.Getenv("MARS_RELEASE_COMMAND_EXIT"))
	os.Exit(exitCode)
}

func testLocalReleaseAssetContract(t *testing.T) []localReleaseAsset {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, 0, len(selfupdate.ExpectedReleaseAssetNames()))
	for _, name := range selfupdate.ExpectedReleaseAssetNames() {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte("fixture:"+name+"\n"), 0o600))
		paths = append(paths, path)
	}
	contract, err := localReleaseAssetContract(paths)
	require.NoError(t, err)
	return contract
}

func contractPaths(contract []localReleaseAsset) []string {
	paths := make([]string, 0, len(contract))
	for _, asset := range contract {
		paths = append(paths, asset.Path)
	}
	return paths
}

func sortedContractNames(contract []localReleaseAsset) []string {
	names := make([]string, 0, len(contract))
	for _, asset := range contract {
		names = append(names, asset.Name)
	}
	sort.Strings(names)
	return names
}

func testGitHubMetadata(tag string, contract []localReleaseAsset) githubReleaseMetadata {
	metadata := githubReleaseMetadata{TagName: tag}
	for _, asset := range contract {
		metadata.Assets = append(metadata.Assets, githubReleaseAssetState{
			Name:   asset.Name,
			State:  "uploaded",
			Size:   asset.Size,
			Digest: "sha256:" + asset.SHA256,
		})
	}
	return metadata
}

func testGitHubMetadataSequence(sequence []githubReleaseMetadata) (func(context.Context) (githubReleaseMetadata, error), *int) {
	calls := 0
	return func(ctx context.Context) (githubReleaseMetadata, error) {
		if err := ctx.Err(); err != nil {
			return githubReleaseMetadata{}, err
		}
		index := calls
		if index >= len(sequence) {
			index = len(sequence) - 1
		}
		calls++
		return sequence[index], nil
	}, &calls
}
