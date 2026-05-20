/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

func TestShellExec_normalizesModelMalformedArgv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "json encoded argv string",
			raw:  `{"argv":"[\"echo\",\"hello\"]","timeout_seconds":5}`,
		},
		{
			name: "python style argv string",
			raw:  `{"argv":"['echo','hello']","timeout_seconds":5}`,
		},
		{
			name: "single simple command string in argv",
			raw:  `{"argv":["echo hello"],"timeout_seconds":5}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			res, err := handleShellExec(context.Background(), root, []byte(tt.raw))
			require.NoError(t, err)
			require.Equal(t, 0, res.ExitCode)
			require.Equal(t, "hello", strings.TrimSpace(res.Output))
		})
	}
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

func TestShellExecRejectsShellCommandBackgroundOperator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "standalone background",
			raw:  `{"shell_command":"go run src/main.go & PID=$!"}`,
		},
		{
			name: "compact background",
			raw:  `{"shell_command":"go run src/main.go& PID=$!"}`,
		},
		{
			name: "unquoted URL ampersand",
			raw:  `{"shell_command":"curl http://localhost:8080/health?ready=1&verbose=1"}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "background:true")
			require.Contains(t, err.Error(), "leak child processes")
		})
	}
}

func TestShellExecAllowsShellCommandNonBackgroundAmpersands(t *testing.T) {
	t.Parallel()
	tests := []string{
		`{"shell_command":"printf ok && printf done","timeout_seconds":5}`,
		`{"shell_command":"printf ok 2>&1","timeout_seconds":5}`,
		`{"shell_command":"printf 'a&b'","timeout_seconds":5}`,
		`{"shell_command":"printf \"a&b\"","timeout_seconds":5}`,
	}
	for _, raw := range tests {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(raw))
			require.NoError(t, err)
		})
	}
}

func TestShellExecRejectsBarePortCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "argv",
			raw:  `{"argv":[":8080"]}`,
		},
		{
			name: "shell command",
			raw:  `{"shell_command":":8080"}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)

			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "is a port, not an executable command")
			require.Contains(t, err.Error(), "background:true")
			require.Contains(t, err.Error(), "curl http://localhost:8080/health")
		})
	}
}

func TestShellExecRejectsExternalTimeoutCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "argv timeout",
			raw:  `{"argv":["timeout","5","go","run","main.go"]}`,
		},
		{
			name: "shell command timeout",
			raw:  `{"shell_command":"timeout 5 go run main.go"}`,
		},
		{
			name: "gnu timeout alias",
			raw:  `{"argv":["gtimeout","5","go","test","./..."]}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)

			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "external timeout command")
			require.Contains(t, err.Error(), "timeout_seconds")
			require.Contains(t, err.Error(), "background:true")
		})
	}
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

func TestShellExecBackgroundReportsEarlyExit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"shell_command":"echo boom >&2; exit 7","background":true}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "background process exited during startup")
	require.Equal(t, 7, res.ExitCode)
	require.Contains(t, res.Stderr, "boom")
}

func TestShellExecBackgroundReturnsPIDForLongRunningProcess(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	defer KillBackgroundProcs()
	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","sleep 5"],"background":true}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "Started in background (PID")
}

func TestKillBackgroundProcsKillsEscapedChildProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group cleanup test is unix-specific")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "leaker.go")
	bin := filepath.Join(dir, "leaker")
	pidFile := filepath.Join(dir, "child.pid")
	require.NoError(t, os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	cmd := exec.Command("/bin/sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}
`), 0o644))
	build := exec.Command("go", "build", "-o", bin, src)
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleShellExec(context.Background(), root, []byte(fmt.Sprintf(`{"argv":[%q,%q],"background":true}`, bin, pidFile)))
	require.NoError(t, err)
	t.Cleanup(KillBackgroundProcs)

	var childPID int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return false
		}
		childPID = pid
		return syscall.Kill(childPID, 0) == nil
	}, 3*time.Second, 50*time.Millisecond)

	KillBackgroundProcs()
	require.Eventually(t, func() bool {
		return syscall.Kill(childPID, 0) != nil
	}, 3*time.Second, 50*time.Millisecond)
}

func TestShellExecKillTrackedBackgroundPIDKillsDescendant(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group cleanup test is unix-specific")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "leaker.go")
	bin := filepath.Join(dir, "leaker")
	pidFile := filepath.Join(dir, "child.pid")
	require.NoError(t, os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	cmd := exec.Command("/bin/sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}
`), 0o644))
	build := exec.Command("go", "build", "-o", bin, src)
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(fmt.Sprintf(`{"argv":[%q,%q],"background":true}`, bin, pidFile)))
	require.NoError(t, err)
	t.Cleanup(KillBackgroundProcs)
	parentPID := backgroundPIDFromOutput(t, res.Output)

	var childPID int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return false
		}
		childPID = pid
		return syscall.Kill(childPID, 0) == nil
	}, 3*time.Second, 50*time.Millisecond)

	res, err = handleShellExec(context.Background(), root, []byte(fmt.Sprintf(`{"argv":["kill","-TERM","%d"]}`, parentPID)))
	require.NoError(t, err)
	require.Contains(t, res.Output, "Killed background process tree")
	require.Eventually(t, func() bool {
		return syscall.Kill(childPID, 0) != nil
	}, 3*time.Second, 50*time.Millisecond)
}

func backgroundPIDFromOutput(t *testing.T, output string) int {
	t.Helper()
	start := strings.Index(output, "PID ")
	require.NotEqual(t, -1, start, output)
	start += len("PID ")
	end := strings.Index(output[start:], ")")
	require.NotEqual(t, -1, end, output)
	pid, err := strconv.Atoi(output[start : start+end])
	require.NoError(t, err)
	return pid
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

func TestShellExecArgvRejectsShellSyntax(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{name: "shell builtin truncate", raw: `{"argv":[":",">","src/test.html"]}`},
		{name: "redirection as executable", raw: `{"argv":[">","/dev/null"]}`},
		{name: "redirection argument", raw: `{"argv":["echo","ok",">","out.txt"]}`},
		{name: "file descriptor redirection", raw: `{"argv":["echo","ok","2>/dev/null"]}`},
		{name: "control syntax in single argv string", raw: `{"argv":["echo ok && rm nope"]}`},
		{name: "pipeline argument", raw: `{"argv":["cat","README.md","|","wc","-l"]}`},
		{name: "command substitution", raw: `{"argv":["echo","$(pwd)"]}`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "argv mode cannot run shell syntax")
			require.Contains(t, err.Error(), "Use shell_command")
		})
	}
}

func TestShellExecBlocksGitRemoteMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{name: "argv remote add", raw: `{"argv":["git","remote","add","origin","https://example.invalid/repo.git"]}`},
		{name: "shell command remote set url", raw: `{"shell_command":"git remote set-url origin https://example.invalid/repo.git"}`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "git remote")
			require.Contains(t, err.Error(), "blocked")
		})
	}
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

func TestShellPolicyBlocksRawDependencyMutationCommands(t *testing.T) {
	t.Parallel()
	cases := []string{
		`{"argv":["npm","install"]}`,
		`{"argv":["npm","ci"]}`,
		`{"argv":["pnpm","install"]}`,
		`{"argv":["yarn","install"]}`,
		`{"argv":["bun","install"]}`,
		`{"argv":["go","mod","download"]}`,
		`{"argv":["cargo","fetch"]}`,
		`{"argv":["pip","install","-r","requirements.txt"]}`,
		`{"argv":["python","-m","pip","install","-r","requirements.txt"]}`,
		`{"argv":["bundle","install"]}`,
		`{"argv":["composer","install"]}`,
	}
	for _, raw := range cases {
		err := checkShellPolicy(json.RawMessage(raw))
		require.Error(t, err)
		require.Contains(t, err.Error(), "dependency_sync")
	}
}

func TestShellPolicyBlocksBroadFindWithoutGeneratedExcludes(t *testing.T) {
	t.Parallel()
	err := checkShellPolicy(json.RawMessage(`{"argv":["find",".","-name","*.js"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated-directory excludes")

	err = checkShellPolicy(json.RawMessage(`{"argv":["find",".","-path","./node_modules","-prune","-o","-name","*.js"]}`))
	require.NoError(t, err)
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

func TestShellExecAllowsUntrackedRootBuildArtifactCleanup(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	artifact := filepath.Base(dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, artifact), append([]byte{0}, bytes.Repeat([]byte("binary\n"), 600)...), 0o755))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), artifact)
	require.Contains(t, err.Error(), "rm "+artifact)

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["rm","%s"]}`, artifact))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.NoFileExists(t, filepath.Join(dir, artifact))
}

func TestShellExecAllowsUntrackedGoModuleBuildArtifactCleanup(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	artifact := "task-notes-api"
	require.NoError(t, os.WriteFile(filepath.Join(dir, artifact), append([]byte{0}, bytes.Repeat([]byte("binary\n"), 600)...), 0o755))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), artifact)
	require.Contains(t, err.Error(), "rm "+artifact)

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["rm","%s"]}`, artifact))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.NoFileExists(t, filepath.Join(dir, artifact))
	require.FileExists(t, filepath.Join(dir, "go.mod"))
}

func TestShellExecBlocksGoBuildOutputInsideRepoBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["go","build","-o","task-notes-api","main.go"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build output")
	require.Contains(t, err.Error(), "inside the target repo")
	require.Contains(t, err.Error(), "/tmp/task-notes-api-validation")
	require.NoFileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestShellExecBlocksDefaultGoBuildInsideRepoBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "dogfood",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["go","build","./..."]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build without -o")
	require.Contains(t, err.Error(), "/tmp/task-notes-api-validation")
	require.NoFileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestShellExecBlocksDefaultGoBuildInShellCommandBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "dogfood",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"shell_command":"go build ./... && go test ./..."}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build without -o")
	require.Contains(t, err.Error(), "/tmp/task-notes-api-validation")
	require.NoFileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestShellExecAllowsGoBuildOutputOutsideRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	output := filepath.Join(t.TempDir(), "task-notes-api")

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["go","build","-o",%q,"main.go"]}`, output))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.FileExists(t, output)
}

func TestShellExecMalformedArgsNotMaskedByDirtyArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	artifact := "task-notes-api"
	require.NoError(t, os.WriteFile(filepath.Join(dir, artifact), append([]byte{0}, bytes.Repeat([]byte("binary\n"), 600)...), 0o755))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":[]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "provide exactly one of argv")
	require.NotContains(t, err.Error(), "blast radius exceeded")
	require.FileExists(t, filepath.Join(dir, artifact))
}

func TestShellExecStillBlocksRemovalOfOrdinaryFiles(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep me\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["rm","notes.txt"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "forbidden operation")
	require.FileExists(t, filepath.Join(dir, "notes.txt"))
}

func TestShellExecStillBlocksGoModuleNamedTextFileRemoval(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task-notes-api"), []byte("keep me\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["rm","task-notes-api"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "forbidden operation")
	require.FileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestFileWriteBlocksNewRootValidationScript(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"validate.sh","content":"#!/bin/sh\ngo test ./...\n"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "repo-root validation script")
	require.Contains(t, err.Error(), "direct shell_exec build/run/curl evidence")
	require.NoFileExists(t, filepath.Join(dir, "validate.sh"))
}

func TestFileWriteAllowsExistingRootValidationScriptUpdate(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "validate.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	runTestGit(t, dir, "add", "validate.sh")
	runTestGit(t, dir, "commit", "-m", "add validation script")

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	res, err := ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"validate.sh","content":"#!/bin/sh\ngo test ./...\n"}`)
	require.NoError(t, err)
	require.Contains(t, res.Output, "wrote")
	content, err := os.ReadFile(filepath.Join(dir, "validate.sh"))
	require.NoError(t, err)
	require.Contains(t, string(content), "go test")
}

func TestValidateRepoDiffIgnoresGeneratedUntrackedFiles(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "huge"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "huge", "index.js"), []byte(strings.Repeat("generated\n", 1200)), 0o644))

	err := ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "implementation.js"), []byte(strings.Repeat("source\n", 1200)), 0o644))
	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "implementation.js")
}

func TestValidateRepoDiffIgnoresGeneratedDependencyMetadataLineChurn(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", "package-lock.json")
	runTestGit(t, dir, "commit", "-m", "add lockfile")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(strings.Repeat("lock\n", 1200)), 0o644))
	err := ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "implementation.js"), []byte(strings.Repeat("source\n", 1200)), 0o644))
	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "implementation.js")
	require.NotContains(t, err.Error(), "package-lock.json")
}

func TestValidateRepoDiffIgnoresUntrackedGeneratedDependencyMetadataLineChurn(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(strings.Repeat("lock\n", 1200)), 0o644))

	err := ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "implementation.js"), []byte(strings.Repeat("source\n", 1200)), 0o644))
	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "implementation.js")
	require.NotContains(t, err.Error(), "package-lock.json")
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
