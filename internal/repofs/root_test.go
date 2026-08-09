/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-017-open-source-publication.md
*/
package repofs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootAtomicWriteAndInventory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, root.MkdirAll("docs/reports", 0o755))
	require.NoError(t, root.AtomicWrite("docs/reports/result.md", []byte("result\n"), 0o640))
	resolved, err := root.Resolve("docs/reports/result.md")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root.Abs(), "docs/reports/result.md"), resolved)

	data, err := root.ReadFile("docs/reports/result.md")
	require.NoError(t, err)
	require.Equal(t, "result\n", string(data))
	info, err := root.Stat("docs/reports/result.md")
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
	lstat, err := root.Lstat("docs/reports/result.md")
	require.NoError(t, err)
	require.True(t, lstat.Mode().IsRegular())
	require.NoError(t, root.Chmod("docs/reports/result.md", 0o600))
	info, err = root.Stat("docs/reports/result.md")
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	file, err := root.OpenFile("docs/reports/result.md")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	matches, err := root.Glob("docs/reports/*.md")
	require.NoError(t, err)
	require.Equal(t, []string{"docs/reports/result.md"}, matches)

	exclusive, err := root.CreateExclusive("docs/reports/exclusive.txt", 0o600)
	require.NoError(t, err)
	_, err = exclusive.WriteString("exclusive\n")
	require.NoError(t, err)
	require.NoError(t, exclusive.Close())
	_, err = root.CreateExclusive("docs/reports/exclusive.txt", 0o600)
	require.Error(t, err)
	require.NoError(t, root.Rename("docs/reports/exclusive.txt", "docs/reports/moved.txt"))
	require.NoError(t, root.Remove("docs/reports/moved.txt"))
	_, err = root.Stat("docs/reports/moved.txt")
	require.ErrorIs(t, err, os.ErrNotExist)

	require.NoError(t, root.MkdirAll("scratch/nested", 0o755))
	require.NoError(t, root.AtomicWrite("scratch/nested/data.txt", []byte("data\n"), 0o600))
	require.NoError(t, root.RemoveAll("scratch"))
	_, err = root.Stat("scratch")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRootRejectsInvalidAndSymlinkedPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel.txt")
	require.NoError(t, os.WriteFile(sentinel, []byte("outside\n"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "linked-parent")))
	require.NoError(t, os.Symlink(sentinel, filepath.Join(dir, "linked-leaf")))
	root, err := Open(dir)
	require.NoError(t, err)

	for _, name := range []string{"", "../outside", sentinel, "bad\x00path"} {
		require.Error(t, root.AtomicWrite(name, []byte("mutated"), 0o600), name)
	}
	for _, name := range []string{"linked-parent/sentinel.txt", "linked-leaf"} {
		_, err := root.ReadFile(name)
		require.Error(t, err, name)
		require.Error(t, root.AtomicWrite(name, []byte("mutated"), 0o600), name)
	}
	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "outside\n", string(data))
}

func TestRootRenameAndRemoveDoNotFollowSymlinks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("outside\n"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "linked")))
	root, err := Open(dir)
	require.NoError(t, err)
	require.Error(t, root.Remove("linked"))
	require.Error(t, root.Rename("linked", "renamed"))
	data, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.Equal(t, "outside\n", string(data))
}

func TestRootRemainsBoundWhenOriginalPathIsReplaced(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	repository := filepath.Join(base, "repository")
	moved := filepath.Join(base, "moved-repository")
	outside := filepath.Join(base, "outside")
	require.NoError(t, os.Mkdir(repository, 0o700))
	require.NoError(t, os.Mkdir(outside, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(repository, "sentinel.txt"), []byte("original\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "sentinel.txt"), []byte("outside\n"), 0o600))

	root, err := Open(repository)
	require.NoError(t, err)
	require.NoError(t, os.Rename(repository, moved))
	require.NoError(t, os.Symlink(outside, repository))

	data, err := root.ReadFile("sentinel.txt")
	require.NoError(t, err)
	require.Equal(t, "original\n", string(data))
	require.NoError(t, root.AtomicWrite("sentinel.txt", []byte("updated\n"), 0o600))

	outsideData, err := os.ReadFile(filepath.Join(outside, "sentinel.txt"))
	require.NoError(t, err)
	require.Equal(t, "outside\n", string(outsideData))
	movedData, err := os.ReadFile(filepath.Join(moved, "sentinel.txt"))
	require.NoError(t, err)
	require.Equal(t, "updated\n", string(movedData))
}
