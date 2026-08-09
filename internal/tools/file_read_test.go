/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/pkg/testutil"

	"github.com/stretchr/testify/require"
)

func TestFileRead_readsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "a.txt"), "line1\nline2\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleFileRead(context.Background(), root, []byte(`{"path":"a.txt"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "line1")
}

func TestFileRead_lineRange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "a.txt"), "a\nb\nc\nd\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleFileRead(context.Background(), root, []byte(`{"path":"a.txt","start_line":2,"end_line":3}`))
	require.NoError(t, err)
	require.Equal(t, "b\nc", res.Output)
}

func TestFileRead_missingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleFileRead(context.Background(), root, []byte(`{"path":"missing.txt"}`))
	require.Error(t, err)
	require.True(t, errors.Is(err, os.ErrNotExist), "expected ErrNotExist in chain: %v", err)
}

func TestFileReadPolicyBlocksGeneratedWorkspaceOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "dist", "assets", "index.js"), "minified bundle")
	root, err := NewRoot(dir)
	require.NoError(t, err)

	err = preToolPolicy(context.Background(), root, "file_read", []byte(`{"path":"dist/assets/index.js"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated dependency/build output")
	require.Contains(t, err.Error(), "dist/assets/index.js")
}

func TestFileRead_rejectsSymlinkParentAndLeaf(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outsideDir := t.TempDir()
	sentinel := filepath.Join(outsideDir, "sentinel.txt")
	require.NoError(t, os.WriteFile(sentinel, []byte("outside-secret\n"), 0o600))
	require.NoError(t, os.Symlink(outsideDir, filepath.Join(dir, "parent")))
	require.NoError(t, os.Symlink(sentinel, filepath.Join(dir, "leaf")))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	for _, path := range []string{"parent/sentinel.txt", "leaf"} {
		result, err := handleFileRead(context.Background(), root, []byte(`{"path":"`+path+`"}`))
		require.Error(t, err)
		require.NotContains(t, result.Output, "outside-secret")
	}
}
