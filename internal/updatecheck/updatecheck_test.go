package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/buildinfo"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/stretchr/testify/require"
)

func TestRun_reportsBehindToolAndCurrentHarness(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v999.0.0"}`))
	}))
	defer server.Close()

	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0o755))
	require.NoError(t, scanner.Init(repo, false))

	report, err := Run(context.Background(), Config{
		CurrentVersion:   buildinfo.DefaultVersion,
		RepoPath:         repo,
		LatestReleaseURL: server.URL,
		HTTPClient:       server.Client(),
	})
	require.NoError(t, err)
	require.Equal(t, StatusBehind, report.Tool.Status)
	require.Contains(t, report.Tool.Command, "update tool")
	require.Equal(t, StatusUpToDate, report.Harness.Status)
	require.Contains(t, report.Actions, "mars-harness update tool --version v999.0.0")
}

func TestRun_recommendsHarnessUpdateWhenMetadataMissing(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))

	report, err := Run(context.Background(), Config{
		CurrentVersion: buildinfo.DefaultVersion,
		RepoPath:       repo,
		SkipRemote:     true,
	})
	require.NoError(t, err)
	require.Equal(t, StatusUnknown, report.Tool.Status)
	require.Equal(t, StatusUnknown, report.Harness.Status)
	require.Contains(t, report.Harness.Command, "update harness")
	require.Contains(t, report.Actions, report.Harness.Command)
}

func TestRun_reportsHarnessBehindInstalledTool(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0o755))
	require.NoError(t, scanner.Init(repo, false))
	metadataPath := filepath.Join(repo, ".harness", "metadata.yaml")
	require.NoError(t, os.WriteFile(metadataPath, []byte("schema_version: 1\ngenerator: mars-harness\ngenerator_version: 0.5.0\n"), 0o644))

	report, err := Run(context.Background(), Config{
		CurrentVersion: buildinfo.DefaultVersion,
		RepoPath:       repo,
		SkipRemote:     true,
	})
	require.NoError(t, err)
	require.Equal(t, StatusBehind, report.Harness.Status)
	require.Contains(t, report.Harness.Command, "update harness")
}

func TestRun_treatsFoundationRepoAsSourceHarness(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module github.com/greaveselliott/mars-harness\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "internal", "scanner"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "internal", "scanner", "init.go"), []byte("package scanner\n"), 0o644))

	report, err := Run(context.Background(), Config{
		CurrentVersion: buildinfo.DefaultVersion,
		RepoPath:       repo,
		SkipRemote:     true,
	})
	require.NoError(t, err)
	require.Equal(t, StatusUpToDate, report.Harness.Status)
	require.Empty(t, report.Harness.Command)
	require.NotContains(t, report.Actions, "mars-harness init --repo "+repo)
	require.Contains(t, report.Harness.Message, "foundation harness source repo")
}

func TestRun_reportsOperatingModelDrift(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", "metadata.yaml"), []byte("schema_version: 1\ngenerator: mars-harness\ngenerator_version: "+buildinfo.DefaultVersion+"\n"), 0o644))

	report, err := Run(context.Background(), Config{
		CurrentVersion: buildinfo.DefaultVersion,
		RepoPath:       repo,
		SkipRemote:     true,
	})
	require.NoError(t, err)
	require.Equal(t, StatusBehind, report.Harness.Status)
	require.Contains(t, report.Harness.Message, "operating model drift")
	require.Contains(t, report.Harness.Command, "update harness")
	require.Contains(t, report.Actions, report.Harness.Command)
}
