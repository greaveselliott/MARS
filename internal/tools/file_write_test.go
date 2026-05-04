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

	"github.com/stretchr/testify/require"
)

func TestFileWrite_createsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleFileWrite(context.Background(), root, []byte(`{"path":"x/y.txt","content":"hi"}`))
	require.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(dir, "x", "y.txt"))
	require.NoError(t, err)
	require.Equal(t, "hi", string(b))
}

func TestFileWrite_rejectsEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleFileWrite(context.Background(), root, []byte(`{"path":"../outside.txt","content":"x"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes")
}
