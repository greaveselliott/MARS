package tools

import (
	"context"
	"encoding/json"
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

func TestShellPolicyBlocksDestructiveVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "force push reordered shell command",
			raw:  `{"shell_command":"git push origin main --force"}`,
			want: "git push --force",
		},
		{
			name: "force with lease shell command",
			raw:  `{"shell_command":"git push origin main --force-with-lease"}`,
			want: "git push --force",
		},
		{
			name: "short force push argv",
			raw:  `{"argv":["git","push","origin","main","-f"]}`,
			want: "git push --force",
		},
		{
			name: "reset hard",
			raw:  `{"shell_command":"git reset --hard HEAD~1"}`,
			want: "git reset --hard",
		},
		{
			name: "clean combined flags",
			raw:  `{"shell_command":"git clean -dfx"}`,
			want: "git clean -fd",
		},
		{
			name: "branch delete uppercase",
			raw:  `{"shell_command":"git branch -D topic"}`,
			want: "git branch -d",
		},
		{
			name: "root delete reordered flags",
			raw:  `{"argv":["rm","-fr","--","/"]}`,
			want: "rm -rf /",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkShellPolicy(json.RawMessage(tc.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}
