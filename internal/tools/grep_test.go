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

func TestGrep_findsLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "f.go"), "foo\nbar\nfoo2\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleGrep(context.Background(), root, []byte(`{"pattern":"foo","glob":"*.go"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "f.go:1:foo")
	require.Contains(t, res.Output, "f.go:3:foo2")
}

func TestGrep_invalidRegex(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleGrep(context.Background(), root, []byte(`{"pattern":"("}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid regex")
}

func TestGrep_rejectsSymlinkedInput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.go")
	require.NoError(t, os.WriteFile(outside, []byte("outside-secret\n"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "linked.go")))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	result, err := handleGrep(context.Background(), root, []byte(`{"pattern":"outside","glob":"*.go"}`))
	require.Error(t, err)
	require.NotContains(t, result.Output, "outside-secret")
}
