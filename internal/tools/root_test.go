/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoot_ResolvePath_allowsNested(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, err := NewRoot(dir)
	require.NoError(t, err)
	got, err := r.ResolvePath(filepath.Join("a", "b.txt"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(r.Abs(), "a", "b.txt"), got)
}

func TestRoot_ResolvePath_rejectsEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = r.ResolvePath(filepath.Join("..", "..", "etc", "passwd"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes")
}

func TestRoot_ResolvePath_rejectsAbsolute(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = r.ResolvePath("/tmp/outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), "relative")
}
