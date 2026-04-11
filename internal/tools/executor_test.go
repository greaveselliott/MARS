package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutor_notAllowlisted(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"file_read"}, "shell_exec", `{"argv":["echo","hi"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestExecutor_unknownTool(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	require.NoError(t, RegisterBuiltinTools(reg))
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"not_a_real_tool"}, "not_a_real_tool", `{}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}

func TestExecutor_invalidJSON(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"file_read"}, "file_read", `{`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "valid JSON")
}
