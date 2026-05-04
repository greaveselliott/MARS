package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/safety"
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
		{
			name: "repo delete shell command",
			raw:  `{"shell_command":"rm -rf src docs/tickets"}`,
			want: "rm",
		},
		{
			name: "git rm shell command",
			raw:  `{"shell_command":"git rm -r src"}`,
			want: "git rm",
		},
		{
			name: "find delete shell command",
			raw:  `{"shell_command":"find . -name '*.tmp' -delete"}`,
			want: "find -delete",
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

func TestShellExecReadOnlyClassification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "ls argv", raw: `{"argv":["ls","docs/tickets"]}`, want: true},
		{name: "find without actions", raw: `{"argv":["find","docs/tickets","-name","*.md","-type","f"]}`, want: true},
		{name: "safe git status", raw: `{"argv":["git","status","--short"]}`, want: true},
		{name: "safe branch current", raw: `{"argv":["git","branch","--show-current"]}`, want: true},
		{name: "sed no print", raw: `{"shell_command":"sed -n '1,20p' docs/tickets/README.md"}`, want: true},
		{name: "sed in place", raw: `{"argv":["sed","-i","s/a/b/","file.txt"]}`, want: false},
		{name: "find exec", raw: `{"argv":["find",".","-exec","rm","{}",";"]}`, want: false},
		{name: "shell control", raw: `{"shell_command":"ls docs | wc -l"}`, want: false},
		{name: "touch", raw: `{"argv":["touch","x.txt"]}`, want: false},
		{name: "background", raw: `{"argv":["ls"],"background":true}`, want: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, shellExecReadOnly(json.RawMessage(tt.raw)))
		})
	}
}

func TestShellExecReadOnlyAllowedInDirtyRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 12)
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	var policyEvents []PolicyEvent
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
		PolicyRecorder: func(evt PolicyEvent) {
			policyEvents = append(policyEvents, evt)
		},
	}

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["ls","."]}`)
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Empty(t, policyEvents, "read-only inspection should not emit blast-radius policy noise")
	require.NoFileExists(t, filepath.Join(dir, "should-not-exist"))
}

func TestShellExecUnknownCommandBlockedBeforeExecutionInDirtyRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty-00.txt"), []byte(strings.Repeat("dirty\n", 600)), 0o644))
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	var policyEvents []PolicyEvent
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
		PolicyRecorder: func(evt PolicyEvent) {
			policyEvents = append(policyEvents, evt)
		},
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["touch","should-not-exist"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "may mutate")
	require.Contains(t, err.Error(), "blast radius exceeded")
	require.NoFileExists(t, filepath.Join(dir, "should-not-exist"))
	require.Len(t, policyEvents, 1)
	require.Equal(t, "pre", policyEvents[0].Stage)
}

func setupDirtyGitRepo(t *testing.T, changedFiles int) (string, Root) {
	t.Helper()
	dir := t.TempDir()
	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "test@example.com")
	runTestGit(t, dir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", ".")
	runTestGit(t, dir, "commit", "-m", "initial")
	for i := 0; i < changedFiles; i++ {
		name := fmt.Sprintf("dirty-%02d.txt", i)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("dirty\n"), 0o644))
	}
	root, err := NewRoot(dir)
	require.NoError(t, err)
	return dir, root
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
