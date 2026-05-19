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
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/pkg/testutil"

	"github.com/stretchr/testify/require"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	requireGit(t)
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, string(out))
	}
	run("init")
	run("config", "user.email", "harness@test.local")
	run("config", "user.name", "Harness Test")
}

func TestGitStatus_cleanRepo(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleGitStatus(context.Background(), root, []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Equal(t, "", strings.TrimSpace(res.Output))
}

func TestGitCommit_andStatus(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "a.txt"), "v1\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "a.txt"))
	_, err = handleGitCommit(context.Background(), root, []byte(`{"message":"first","paths":["a.txt"]}`))
	require.NoError(t, err)
	res, err := handleGitStatus(context.Background(), root, []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Equal(t, "", strings.TrimSpace(res.Output))
}

func TestGitBranch_create(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "f.txt"), "x")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "f.txt"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "init"))
	res, err := handleGitBranch(context.Background(), root, []byte(`{"name":"topic","create":true}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "topic")
}

func TestGitDiff_workingTree(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "a.txt"), "v1\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "a.txt"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "init"))
	testutil.WriteFile(t, filepath.Join(dir, "a.txt"), "v2\n")
	res, err := handleGitDiff(context.Background(), root, []byte(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "a.txt")
}

func TestGitPush_noRemote(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "f.txt"), "x")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "f.txt"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "init"))
	res, err := handleGitPush(context.Background(), root, []byte(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "remote \"origin\" is not configured")
}

func TestGitPushPolicyAllowsOnlyMain(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	err = preToolPolicy(context.Background(), root, "git_push", json.RawMessage(`{"branch":"feature/recovery"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "strict trunk")

	err = preToolPolicy(context.Background(), root, "git_push", json.RawMessage(`{"branch":"main"}`))
	require.NoError(t, err)
}

func TestGitCommitPolicyBlocksSecretsInDirtyDiff(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "README.md"), "initial\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "README.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "init"))

	testutil.WriteFile(t, filepath.Join(dir, "secret.txt"), "token = \"github-token-placeholder\"\n")

	err = preToolPolicy(context.Background(), root, "git_commit", json.RawMessage(`{"message":"commit secret"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "secret scanner")
}

func TestGitCommitBlocksGeneratedWorkspaceOutput(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "README.md"), "initial\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "README.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "init"))

	testutil.WriteFile(t, filepath.Join(dir, "dist", "bundle.js"), "compiled\n")
	testutil.WriteFile(t, filepath.Join(dir, "src", "index.js"), "console.log('ship')\n")

	_, err = handleGitCommit(context.Background(), root, []byte(`{"message":"stage all"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated workspace paths would be staged: dist")

	_, err = handleGitCommit(context.Background(), root, []byte(`{"message":"explicit dist","paths":["dist/bundle.js"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated workspace paths cannot be committed: dist")

	_, err = handleGitCommit(context.Background(), root, []byte(`{"message":"source only","paths":["src/index.js"]}`))
	require.NoError(t, err)

	ls, err := runGit(context.Background(), root, "ls-files", "dist")
	require.NoError(t, err)
	require.Equal(t, "", strings.TrimSpace(ls.Output))
}

func TestGitCommitBlocksAlreadyStagedGeneratedWorkspaceOutput(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "README.md"), "initial\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, runGitExit0(context.Background(), root, "add", "README.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "init"))

	testutil.WriteFile(t, filepath.Join(dir, "dist", "bundle.js"), "compiled\n")
	require.NoError(t, runGitExit0(context.Background(), root, "add", "dist/bundle.js"))

	_, err = handleGitCommit(context.Background(), root, []byte(`{"message":"normal file","paths":["README.md"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated workspace paths are staged: dist")
}
