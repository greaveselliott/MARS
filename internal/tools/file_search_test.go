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
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/pkg/testutil"

	"github.com/stretchr/testify/require"
)

func TestFileSearch_glob(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "a.go"), "x")
	testutil.WriteFile(t, filepath.Join(dir, "b.txt"), "y")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleFileSearch(context.Background(), root, []byte(`{"pattern":"*.go"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "a.go")
	require.NotContains(t, res.Output, "b.txt")
}

func TestFileSearch_rejectsSymlinkedMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.go")
	require.NoError(t, os.WriteFile(outside, []byte("outside\n"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "linked.go")))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	result, err := handleFileSearch(context.Background(), root, []byte(`{"pattern":"*.go"}`))
	require.Error(t, err)
	require.NotContains(t, result.Output, "outside.go")
}
