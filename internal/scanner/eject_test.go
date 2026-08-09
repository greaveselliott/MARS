/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-004-target-harness-lifecycle.md
*/
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

func TestEjectRejectsSymlinkedHarnessLeaf(t *testing.T) {
	for _, apply := range []bool{false, true} {
		t.Run(map[bool]string{false: "dry-run", true: "apply"}[apply], func(t *testing.T) {
			dir := t.TempDir()
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "sentinel.txt")
			require.NoError(t, os.WriteFile(sentinel, []byte("outside\n"), 0o600))
			require.NoError(t, os.Symlink(outside, filepath.Join(dir, ".harness")))

			_, err := Eject(dir, EjectOptions{Apply: apply})
			require.Error(t, err)
			data, readErr := os.ReadFile(sentinel)
			require.NoError(t, readErr)
			require.Equal(t, "outside\n", string(data))
			require.FileExists(t, filepath.Join(dir, ".harness", "sentinel.txt"))
		})
	}
}

func TestEjectApplyRejectsSymlinkedDocsParentBeforeRemoval(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "docs")))
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel.txt")
	require.NoError(t, os.WriteFile(sentinel, []byte("outside\n"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "docs")))

	_, err := Eject(dir, EjectOptions{Apply: true})
	require.Error(t, err)
	data, readErr := os.ReadFile(sentinel)
	require.NoError(t, readErr)
	require.Equal(t, "outside\n", string(data))
	require.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))
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
