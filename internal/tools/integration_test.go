package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration_fileWriteReadGrep(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	allow := []string{"file_write", "file_read", "grep"}

	_, err = ex.Execute(context.Background(), root, allow, "file_write", `{"path":"note.txt","content":"alpha\nbeta\n"}`)
	require.NoError(t, err)

	res, err := ex.Execute(context.Background(), root, allow, "file_read", `{"path":"note.txt"}`)
	require.NoError(t, err)
	require.Contains(t, res.Output, "alpha")

	res, err = ex.Execute(context.Background(), root, allow, "grep", `{"pattern":"beta","glob":"*.txt"}`)
	require.NoError(t, err)
	require.Contains(t, res.Output, "beta")
}
