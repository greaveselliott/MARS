package selfupdate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePlan_defaultsToReleaseAssetInCurrentExecutableDir(t *testing.T) {
	plan, err := ResolvePlan(Config{DryRun: true})
	require.NoError(t, err)

	require.Equal(t, MethodReleaseAssets, plan.Method)
	require.Equal(t, DefaultVersion, plan.Version)
	require.Empty(t, plan.Command)
	require.Equal(t, "mars-harness-"+runtime.GOOS+"-"+runtime.GOARCH, plan.AssetName)
	require.Contains(t, plan.DownloadURL, "/latest/"+plan.AssetName)
	require.Contains(t, plan.ChecksumsURL, "/latest/checksums.txt")
	require.True(t, filepath.IsAbs(plan.InstallDir))
	require.Equal(t, filepath.Join(plan.InstallDir, DefaultBinary), plan.BinaryPath)
	require.True(t, plan.DryRun)
}

func TestResolvePlan_acceptsSourceMethodVersionAndInstallDir(t *testing.T) {
	installDir := t.TempDir()
	plan, err := ResolvePlan(Config{
		Version:    "@main",
		InstallDir: installDir,
		BinaryName: "mars-harness-dev",
		Method:     MethodSource,
	})
	require.NoError(t, err)

	require.Equal(t, MethodSource, plan.Method)
	require.Equal(t, "main", plan.Version)
	require.Equal(t, []string{"go", "install", DefaultPackage + "@main"}, plan.Command)
	require.Equal(t, installDir, plan.InstallDir)
	require.Equal(t, filepath.Join(installDir, "mars-harness-dev"), plan.BinaryPath)
}

func TestResolvePlan_selectsSourceForMain(t *testing.T) {
	plan, err := ResolvePlan(Config{Version: "main"})
	require.NoError(t, err)

	require.Equal(t, MethodSource, plan.Method)
	require.Equal(t, []string{"go", "install", DefaultPackage + "@main"}, plan.Command)
}

func TestRunReleaseAssetsVerifiesChecksumAndInstalls(t *testing.T) {
	t.Parallel()
	installDir := t.TempDir()
	asset := "mars-harness-" + runtime.GOOS + "-" + runtime.GOARCH
	payload := []byte("#!/bin/sh\necho updated\n")
	sum := sha256.Sum256(payload)

	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/download/v1.2.3/" + asset:
			return textResponse(http.StatusOK, string(payload)), nil
		case "/download/v1.2.3/checksums.txt":
			return textResponse(http.StatusOK, fmt.Sprintf("%x  %s\n", sum, asset)), nil
		default:
			return textResponse(http.StatusNotFound, "not found"), nil
		}
	})

	plan, err := Run(context.Background(), Config{
		Version:        "v1.2.3",
		InstallDir:     installDir,
		SkipShellPath:  true,
		ReleaseBaseURL: "https://example.test/download",
		HTTPClient:     client,
	})
	require.NoError(t, err)

	require.Equal(t, MethodReleaseAssets, plan.Method)
	require.Equal(t, "1.2.3", plan.Version)
	require.Equal(t, "v1.2.3", plan.ReleaseTag)
	got, err := os.ReadFile(filepath.Join(installDir, DefaultBinary))
	require.NoError(t, err)
	require.Equal(t, payload, got)
	info, err := os.Stat(filepath.Join(installDir, DefaultBinary))
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111)
}

func TestRunReleaseAssetsRejectsChecksumMismatchWithoutReplacingBinary(t *testing.T) {
	t.Parallel()
	installDir := t.TempDir()
	asset := "mars-harness-" + runtime.GOOS + "-" + runtime.GOARCH
	existing := []byte("existing")
	require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), existing, 0o755))

	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/download/v1.2.3/" + asset:
			return textResponse(http.StatusOK, "new"), nil
		case "/download/v1.2.3/checksums.txt":
			return textResponse(http.StatusOK, fmt.Sprintf("%064x  %s\n", 0, asset)), nil
		default:
			return textResponse(http.StatusNotFound, "not found"), nil
		}
	})

	_, err := Run(context.Background(), Config{
		Version:        "v1.2.3",
		InstallDir:     installDir,
		SkipShellPath:  true,
		ReleaseBaseURL: "https://example.test/download",
		HTTPClient:     client,
	})
	require.ErrorContains(t, err, "checksum mismatch")

	got, readErr := os.ReadFile(filepath.Join(installDir, DefaultBinary))
	require.NoError(t, readErr)
	require.Equal(t, existing, got)
}
