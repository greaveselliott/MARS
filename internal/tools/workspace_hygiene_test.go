/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceHygieneDetectsMissingNodeModulesIgnore(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"node test.js"}}`), 0o644))

	report, err := AuditWorkspaceHygiene(context.Background(), root, WorkspaceHygieneOptions{Mode: "pre_dependency"})
	require.NoError(t, err)
	require.True(t, report.Blocking)
	require.Equal(t, workspaceRecipeAddIgnore, report.RecipeID)
	require.Contains(t, report.NextAction, "node_modules/")
}

func TestWorkspaceHygieneNormalizesStringPaths(t *testing.T) {
	_, root := setupWorkspaceHygieneRepo(t)
	res, err := handleWorkspaceHygiene(context.Background(), root, []byte(`{
		"mode": "audit",
		"paths": "[\"docs/tickets/backlog\"]"
	}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, `"status": "clean"`)

	args, err := decodeWorkspaceHygieneArgs([]byte(`{"mode":"audit","paths":"['docs/tickets/backlog']"}`))
	require.NoError(t, err)
	require.Equal(t, []string{"docs/tickets/backlog"}, args.Paths)
}

func TestWorkspaceHygieneDetectsTrackedGeneratedDirectory(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("module.exports = {}\n"), 0o644))
	runTestGit(t, dir, "add", "node_modules/pkg/index.js")
	runTestGit(t, dir, "commit", "-m", "track generated")

	report, err := AuditWorkspaceHygiene(context.Background(), root, WorkspaceHygieneOptions{Mode: "pre_job"})
	require.NoError(t, err)
	require.True(t, report.Blocking)
	require.Equal(t, workspaceRecipeTrackGenerated, report.RecipeID)
	require.Contains(t, strings.Join(report.Findings[0].Paths, ","), "node_modules")
}

func TestWorkspaceHygieneDetectsUntrackedGeneratedChurn(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("dist/\n"), 0o644))
	runTestGit(t, dir, "add", ".gitignore")
	runTestGit(t, dir, "commit", "-m", "ignore dist")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("module.exports = {}\n"), 0o644))

	report, err := AuditWorkspaceHygiene(context.Background(), root, WorkspaceHygieneOptions{Mode: "post_dependency"})
	require.NoError(t, err)
	require.True(t, report.Blocking)
	require.Equal(t, workspaceRecipeGeneratedDirty, report.RecipeID)
}

func TestWorkspaceHygieneDetectsForbiddenDeletion(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "README.md")))

	report, err := AuditWorkspaceHygiene(context.Background(), root, WorkspaceHygieneOptions{Mode: "pre_job"})
	require.NoError(t, err)
	require.True(t, report.Blocking)
	require.Equal(t, workspaceRecipeForbiddenDelete, report.RecipeID)
}

func TestWorkspaceHygieneDetectsLargeGeneratedDiff(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "dist"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dist", "bundle.js"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", "dist/bundle.js")
	runTestGit(t, dir, "commit", "-m", "track dist")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dist", "bundle.js"), []byte(strings.Repeat("x\n", 600)), 0o644))

	report, err := AuditWorkspaceHygiene(context.Background(), root, WorkspaceHygieneOptions{Mode: "pre_job"})
	require.NoError(t, err)
	require.True(t, report.Blocking)
	foundLarge := false
	for _, finding := range report.Findings {
		if finding.Type == "large_generated_diff" {
			foundLarge = true
		}
	}
	require.True(t, foundLarge)
}

func TestWorkspaceHygieneRepairCommitsMissingGeneratedIgnorePolicy(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"node test.js"}}`), 0o644))
	runTestGit(t, dir, "add", "package.json")
	runTestGit(t, dir, "commit", "-m", "add package manifest")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("module.exports = {}\n"), 0o644))

	repair, err := RepairWorkspaceHygieneIgnorePolicy(context.Background(), root)
	require.NoError(t, err)
	require.True(t, repair.Committed)
	require.Contains(t, repair.MissingIgnores, "node_modules")
	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	require.Contains(t, string(gitignore), "node_modules/")
	require.Contains(t, gitLog(t, dir, "-1", "--pretty=%s"), "chore(hygiene): ignore generated workspace output")
	require.Empty(t, strings.TrimSpace(testGitOutput(t, dir, "status", "--porcelain", "-uall")))

	report, err := AuditWorkspaceHygiene(context.Background(), root, WorkspaceHygieneOptions{Mode: "pre_job"})
	require.NoError(t, err)
	require.False(t, report.Blocking)
}

func TestWorkspaceHygieneRepairSkipsTrackedGeneratedPaths(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"node test.js"}}`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("module.exports = {}\n"), 0o644))
	runTestGit(t, dir, "add", "package.json", "node_modules/pkg/index.js")
	runTestGit(t, dir, "commit", "-m", "track generated")

	repair, err := RepairWorkspaceHygieneIgnorePolicy(context.Background(), root)
	require.NoError(t, err)
	require.False(t, repair.Committed)
	require.Contains(t, repair.Message, "generated paths are already tracked")
	require.NoFileExists(t, filepath.Join(dir, ".gitignore"))
}

func setupWorkspaceHygieneRepo(t *testing.T) (string, Root) {
	t.Helper()
	dir := t.TempDir()
	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "test@example.com")
	runTestGit(t, dir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", ".")
	runTestGit(t, dir, "commit", "-m", "initial")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	return dir, root
}

func gitLog(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return testGitOutput(t, dir, append([]string{"log"}, args...)...)
}

func testGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}
