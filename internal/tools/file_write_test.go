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

func TestFileWrite_normalizesParameterMarkerInPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	raw := []byte("{\"path\":\"docs/reports/security/report.md\\n<parameter=content>\\n# Report\\n\\nValidated static smoke path.\\n\"}")
	_, err = handleFileWrite(context.Background(), root, raw)
	require.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(dir, "docs", "reports", "security", "report.md"))
	require.NoError(t, err)
	require.Equal(t, "# Report\n\nValidated static smoke path.\n", string(b))
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

func TestFileWrite_rejectsSymlinkParentAndLeaf(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outsideDir := t.TempDir()
	sentinel := filepath.Join(outsideDir, "sentinel.txt")
	require.NoError(t, os.WriteFile(sentinel, []byte("outside\n"), 0o600))
	require.NoError(t, os.Symlink(outsideDir, filepath.Join(dir, "parent")))
	require.NoError(t, os.Symlink(sentinel, filepath.Join(dir, "leaf")))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	for _, path := range []string{"parent/sentinel.txt", "leaf"} {
		_, err := handleFileWrite(context.Background(), root, []byte(`{"path":"`+path+`","content":"mutated"}`))
		require.Error(t, err)
	}
	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "outside\n", string(data))
}
