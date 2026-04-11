package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFile_createsParents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "a.txt")

	WriteFile(t, path, "hello")

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hello", string(b))
}
