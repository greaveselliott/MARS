package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShellExec_argv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","echo hello"],"timeout_seconds":5}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Contains(t, strings.TrimSpace(res.Output), "hello")
}

func TestShellExec_shellCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"shell_command":"echo ok","timeout_seconds":5}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
}

func TestShellExec_timeout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = handleShellExec(ctx, root, []byte(`{"shell_command":"sleep 5","timeout_seconds":1}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
}

func TestShellExec_mutexArgs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","echo x"],"shell_command":"echo y"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one")
}
