package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEjectDryRunDoesNotRemoveHarnessArtifacts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))

	result, err := Eject(dir, EjectOptions{})
	require.NoError(t, err)
	require.Contains(t, result.Removed, ".harness")
	require.Contains(t, result.Removed, "docs/tickets")
	require.Contains(t, result.Removed, "docs/exec-plans")
	require.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))
	require.FileExists(t, filepath.Join(dir, "docs", "tickets", "README.md"))
}

func TestEjectApplyRemovesHarnessArtifactsAndPreservesAppFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("app\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src.txt"), []byte("source\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "app.md"), []byte("app docs\n"), 0o644))

	result, err := Eject(dir, EjectOptions{Apply: true})
	require.NoError(t, err)
	require.Contains(t, result.Removed, ".harness")
	require.Contains(t, result.Removed, "AGENTS.md")
	require.Contains(t, result.Removed, "docs/features")

	require.NoDirExists(t, filepath.Join(dir, ".harness"))
	require.NoFileExists(t, filepath.Join(dir, "AGENTS.md"))
	require.NoDirExists(t, filepath.Join(dir, "docs", "tickets"))
	require.NoDirExists(t, filepath.Join(dir, "docs", "features"))
	require.FileExists(t, filepath.Join(dir, "README.md"))
	require.FileExists(t, filepath.Join(dir, "src.txt"))
	require.FileExists(t, filepath.Join(dir, "docs", "app.md"))
}

func TestEjectApplyPrunesEmptyDocsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))

	result, err := Eject(dir, EjectOptions{Apply: true})
	require.NoError(t, err)
	require.Contains(t, result.Pruned, "docs")
	require.NoDirExists(t, filepath.Join(dir, "docs"))
}
