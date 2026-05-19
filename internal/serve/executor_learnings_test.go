/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/dogfood-and-decisions.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/design-docs/pipeline-engine.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommitRuntimeLearningsIfOnlyDirtyCommitsLearningFile(t *testing.T) {
	t.Parallel()
	dir := initRuntimeLearningsGitRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harness", "learnings.yaml"), []byte("decisions:\n- summary: test\n"), 0o644))

	committed, commit, err := commitRuntimeLearningsIfOnlyDirty(context.Background(), dir, "ceo")
	require.NoError(t, err)
	require.True(t, committed)
	require.NotEmpty(t, commit)
	require.Equal(t, "", strings.TrimSpace(testGit(t, dir, "status", "--short")))
	require.Contains(t, testGit(t, dir, "log", "-1", "--pretty=%s"), "chore(learnings): update runtime learnings for ceo")
}

func TestCommitRuntimeLearningsIfOnlyDirtyLeavesMixedDirtyTreeAlone(t *testing.T) {
	t.Parallel()
	dir := initRuntimeLearningsGitRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harness", "learnings.yaml"), []byte("decisions:\n- summary: test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "product.txt"), []byte("dirty\n"), 0o644))

	committed, commit, err := commitRuntimeLearningsIfOnlyDirty(context.Background(), dir, "engineer")
	require.NoError(t, err)
	require.False(t, committed)
	require.Empty(t, commit)
	status := testGit(t, dir, "status", "--short")
	require.Contains(t, status, "?? .harness/")
	require.Contains(t, status, "?? product.txt")
}

func initRuntimeLearningsGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testGit(t, dir, "init")
	testGit(t, dir, "config", "user.email", "test@example.com")
	testGit(t, dir, "config", "user.name", "Mars Harness Test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644))
	testGit(t, dir, "add", "README.md")
	testGit(t, dir, "commit", "-m", "chore: seed")
	return dir
}

func testGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%v\n%s", args, out)
	return string(out)
}
