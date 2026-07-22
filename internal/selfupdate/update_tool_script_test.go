/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-009-release-update-lifecycle.md
*/
package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateToolScriptFastForwardsAndInstalls(t *testing.T) {
	requireShellCommand(t, "bash")
	local, origin := initUpdateToolRepo(t)
	remoteWork := cloneUpdateToolRepo(t, origin)
	commitUpdateToolFile(t, remoteWork, "remote.txt", "remote\n", "remote update")
	runGit(t, remoteWork, "push", "origin", "main")

	fake := newUpdateToolFakeInstall(t)
	out, err := runUpdateToolScript(t, local, fake)

	require.NoError(t, err, out)
	require.Contains(t, out, "Fast-forwarding to origin/main")
	require.Contains(t, out, "mars test-version")
	require.Equal(t, gitOutput(t, local, "rev-parse", "HEAD"), gitOutput(t, local, "rev-parse", "origin/main"))
	log := readUpdateToolLog(t, fake)
	require.Contains(t, log, "go install install ./cmd/mars")
	require.Contains(t, log, "path setup path setup --install-dir "+fake.installBin)
}

func TestUpdateToolScriptRefusesDirtyCheckout(t *testing.T) {
	requireShellCommand(t, "bash")
	local, _ := initUpdateToolRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(local, "README.md"), []byte("dirty\n"), 0o644))

	fake := newUpdateToolFakeInstall(t)
	out, err := runUpdateToolScript(t, local, fake)

	require.Error(t, err)
	require.Contains(t, out, "worktree has uncommitted changes")
	require.NoFileExists(t, fake.logPath)
}

func TestUpdateToolScriptRefusesMissingOrigin(t *testing.T) {
	requireShellCommand(t, "bash")
	local := t.TempDir()
	runGit(t, local, "init")
	runGit(t, local, "checkout", "-b", "main")
	configureUpdateToolGitUser(t, local)
	commitUpdateToolFile(t, local, "README.md", "initial\n", "initial")

	fake := newUpdateToolFakeInstall(t)
	out, err := runUpdateToolScript(t, local, fake)

	require.Error(t, err)
	require.Contains(t, out, "No origin remote found")
	require.Contains(t, out, "make install")
	require.NoFileExists(t, fake.logPath)
}

func TestUpdateToolScriptRefusesDivergedCheckout(t *testing.T) {
	requireShellCommand(t, "bash")
	local, origin := initUpdateToolRepo(t)
	remoteWork := cloneUpdateToolRepo(t, origin)
	commitUpdateToolFile(t, remoteWork, "remote.txt", "remote\n", "remote update")
	runGit(t, remoteWork, "push", "origin", "main")
	commitUpdateToolFile(t, local, "local.txt", "local\n", "local update")

	fake := newUpdateToolFakeInstall(t)
	out, err := runUpdateToolScript(t, local, fake)

	require.Error(t, err)
	require.Contains(t, out, "cannot fast-forward")
	require.NoFileExists(t, fake.logPath)
}

type updateToolFakeInstall struct {
	binDir     string
	installBin string
	logPath    string
	gopath     string
}

func newUpdateToolFakeInstall(t *testing.T) updateToolFakeInstall {
	t.Helper()
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	installBin := filepath.Join(dir, "gobin")
	gopath := filepath.Join(dir, "gopath")
	logPath := filepath.Join(dir, "calls.log")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	goScript := `#!/bin/sh
set -eu
if [ "$1" = "env" ]; then
	case "$2" in
		GOBIN) printf '%s\n' "$FAKE_GOBIN" ;;
		GOPATH) printf '%s\n' "$FAKE_GOPATH" ;;
		*) exit 1 ;;
	esac
	exit 0
fi
if [ "$1" = "install" ]; then
	mkdir -p "$FAKE_GOBIN"
	{
		printf '%s\n' '#!/bin/sh'
		printf '%s\n' 'set -eu'
		printf '%s\n' 'case "$1" in'
		printf '%s\n' '  path) echo "path setup $*" >> "$FAKE_LOG"; exit 0 ;;'
		printf '%s\n' '  version) echo "mars test-version"; exit 0 ;;'
		printf '%s\n' 'esac'
		printf '%s\n' 'echo "unexpected mars $*" >> "$FAKE_LOG"'
		printf '%s\n' 'exit 1'
	} > "$FAKE_GOBIN/mars"
	chmod +x "$FAKE_GOBIN/mars"
	echo "go install $*" >> "$FAKE_LOG"
	exit 0
fi
echo "unexpected go $*" >> "$FAKE_LOG"
exit 1
`
	goPath := filepath.Join(binDir, "go")
	require.NoError(t, os.WriteFile(goPath, []byte(goScript), 0o755))
	return updateToolFakeInstall{binDir: binDir, installBin: installBin, logPath: logPath, gopath: gopath}
}

func runUpdateToolScript(t *testing.T, repo string, fake updateToolFakeInstall) (string, error) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	script := filepath.Join(wd, "..", "..", "scripts", "update-tool.sh")
	cmd := exec.Command("bash", script)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(),
		"PATH="+fake.binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_GOBIN="+fake.installBin,
		"FAKE_GOPATH="+fake.gopath,
		"FAKE_LOG="+fake.logPath,
		"GO=go",
	)
	out, runErr := cmd.CombinedOutput()
	return string(out), runErr
}

func initUpdateToolRepo(t *testing.T) (string, string) {
	t.Helper()
	local := t.TempDir()
	origin := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, local, "init")
	runGit(t, local, "checkout", "-b", "main")
	configureUpdateToolGitUser(t, local)
	commitUpdateToolFile(t, local, "README.md", "initial\n", "initial")
	runGit(t, "", "init", "--bare", origin)
	runGit(t, origin, "symbolic-ref", "HEAD", "refs/heads/main")
	runGit(t, local, "remote", "add", "origin", origin)
	runGit(t, local, "push", "-u", "origin", "main")
	return local, origin
}

func cloneUpdateToolRepo(t *testing.T, origin string) string {
	t.Helper()
	work := filepath.Join(t.TempDir(), "work")
	runGit(t, "", "clone", origin, work)
	configureUpdateToolGitUser(t, work)
	return work
}

func configureUpdateToolGitUser(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "config", "user.email", "test@example.invalid")
	runGit(t, dir, "config", "user.name", "Update Tool Test")
}

func commitUpdateToolFile(t *testing.T, dir, name, content, message string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", message)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s\n%s", strings.Join(args, " "), string(out))
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s\n%s", strings.Join(args, " "), string(out))
	return strings.TrimSpace(string(out))
}

func readUpdateToolLog(t *testing.T, fake updateToolFakeInstall) string {
	t.Helper()
	data, err := os.ReadFile(fake.logPath)
	require.NoError(t, err)
	return string(data)
}

func requireShellCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available", name)
	}
}
