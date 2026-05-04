package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/serve"
	"github.com/stretchr/testify/require"
)

func TestToolsListCommandIncludesUniversalTools(t *testing.T) {
	t.Parallel()
	cmd := toolsListCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Contains(t, lines, "mars_harness_cli")
	require.Contains(t, lines, "tool_create")
	require.Contains(t, lines, "tool_creation_guard")
}

func TestToolsRunCommandExecutesRegisteredTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeToolRunRepoFile(t, dir, "docs/design-docs/tools-glossary.md", "New built-in tools must originate through `tool_create`\n`record_decision`\n`example_tool`\n")
	writeToolRunRepoFile(t, dir, "docs/design-docs/delivery-operating-model.md", "Built-in tool creation must dogfood the meta-tool path\n")
	writeToolRunRepoFile(t, dir, "docs/design-docs/dogfood-and-decisions.md", "bypassing `tool_create` breaks the doctrine it represents\n")
	writeToolRunRepoFile(t, dir, "internal/scanner/init.go", "Tool creation path\nexample_tool\n")
	writeToolRunRepoFile(t, dir, "internal/docsconsistency/operating_rules_test.go", "TestToolCreationPathIsDocumented\n")
	writeToolRunRepoFile(t, dir, "internal/tools/example_tool.go", "package tools\n")
	writeToolRunRepoFile(t, dir, "internal/tools/example_tool_test.go", "package tools\n")

	cmd := toolsRunCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"tool_creation_guard",
		"--repo", dir,
		"--args-json", `{"tool_name":"example_tool"}`,
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "status: ok")
	require.Contains(t, out.String(), "PASS: internal/tools/example_tool.go exists")
}

func TestToolsRunCommandDogfoodsToolCreate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())
	writeToolRunRepoFile(t, dir, "go.mod", "module github.com/greaveselliott/mars-harness\n\ngo 1.24\n")
	writeToolRunRepoFile(t, dir, "internal/tools/registry.go", "package tools\n")
	writeToolRunRepoFile(t, dir, "internal/tools/register_default.go", "package tools\n")

	cmd := toolsRunCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"tool_create",
		"--repo", dir,
		"--trust", "contributor",
		"--args-json", `{"name":"universal_probe","description":"Probe universal tool invocation.","fields":[{"name":"topic","type":"string","description":"Topic to inspect."}]}`,
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "created internal/tools/universal_probe.go")
	require.FileExists(t, filepath.Join(dir, "internal", "tools", "universal_probe.go"))
	require.FileExists(t, filepath.Join(dir, "internal", "tools", "universal_probe_test.go"))
}

func TestToolsRunObserverBlocksMutatingTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cmd := toolsRunCmd()
	cmd.SetArgs([]string{
		"file_write",
		"--repo", dir,
		"--trust", "observer",
		"--args-json", `{"path":"x.txt","content":"x"}`,
	})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "observer cannot run mutating tool")
	require.NoFileExists(t, filepath.Join(dir, "x.txt"))
}

func TestMCPServeCommand(t *testing.T) {
	t.Parallel()
	cmd := mcpServeCmd()
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--repo", t.TempDir()})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), `"tools"`)
	require.Contains(t, out.String(), `"tool_create"`)
}

func TestScoresCommandMissingDBDirectoryIsActionable(t *testing.T) {
	t.Parallel()
	cmd := scoresCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--db", filepath.Join(t.TempDir(), "missing", "mars.db"),
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "No scores recorded yet.")
	require.Contains(t, out.String(), "database directory")
	require.Contains(t, out.String(), "mars-harness register --repo")
}

func TestScoresCommandUnavailableDBIsActionable(t *testing.T) {
	t.Parallel()
	cmd := scoresCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--db", t.TempDir(),
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "No scores recorded yet.")
	require.Contains(t, out.String(), "database is unavailable")
	require.Contains(t, out.String(), "mars-harness register --repo")
	require.NotContains(t, out.String(), "unable to open database file")
	require.NotContains(t, out.String(), "(14)")
}

func TestScoresCommandFormatsWindowColumn(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	store, err := scoring.OpenStore(dbPath)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		require.NoError(t, store.RecordOutcome(context.Background(), scoring.Outcome{
			JobID:  fmt.Sprintf("job-%d", i),
			RepoID: "repo",
			Role:   "engineer",
			Type:   scoring.OutcomeChecksPassed,
		}))
	}
	_, err = store.ComputeScore(context.Background(), "engineer", "repo", 30)
	require.NoError(t, err)
	require.NoError(t, store.Close())

	cmd := scoresCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--db", dbPath})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "engineer")
	require.NotContains(t, out.String(), "30      d")
}

func TestTrustCommandMissingDBDirectoryIsActionable(t *testing.T) {
	t.Parallel()
	cmd := trustCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--db", filepath.Join(t.TempDir(), "missing", "mars.db"),
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "No trust entries recorded yet.")
	require.Contains(t, out.String(), "database directory")
	require.Contains(t, out.String(), "mars-harness register --repo")
}

func TestTrustCommandUnavailableDBIsActionable(t *testing.T) {
	t.Parallel()
	cmd := trustCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--db", t.TempDir(),
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "No trust entries recorded yet.")
	require.Contains(t, out.String(), "database is unavailable")
	require.Contains(t, out.String(), "mars-harness register --repo")
	require.NotContains(t, out.String(), "unable to open database file")
	require.NotContains(t, out.String(), "(14)")
}

func TestStartCommandInitializesRegistersSeedsAndStops(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	t.Setenv("MARS_HARNESS_WEBHOOK_PORT", "0")
	t.Setenv("MARS_HARNESS_DASHBOARD_PORT", "0")
	t.Setenv("MARS_HARNESS_SKIP_START_CLEANUP", "1")

	cmd := startCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--exit-after-seed",
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Registered repo")
	require.Contains(t, out.String(), "Seeded CEO agent")
	require.Contains(t, out.String(), "Committed generated harness baseline")
	require.FileExists(t, filepath.Join(repoDir, ".harness", "manifest.yaml"))
	require.FileExists(t, dbPath)

	db, err := openDB(dbPath)
	require.NoError(t, err)
	defer db.Close()
	registry, err := serve.NewRepoRegistry(db)
	require.NoError(t, err)
	repos, err := registry.List(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, repoDir, repos[0].Path)

	q, err := queue.Open(dbPath)
	require.NoError(t, err)
	defer q.Close()
	jobs, err := q.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, "ceo", jobs[0].Role)
	require.Equal(t, repos[0].ID, jobs[0].RepoID)
	require.Contains(t, jobs[0].Trigger, `"type":"bootstrap"`)

	log := runMainTestGit(t, repoDir, "log", "--oneline", "-1")
	require.Contains(t, log, "chore(harness): initialize mars harness")
	status := runMainTestGit(t, repoDir, "status", "--short")
	require.Empty(t, strings.TrimSpace(status))
}

func TestInitCommandCommitsGeneratedHarnessBaseline(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	cmd := initCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--repo", repoDir})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Initialized .harness/")
	require.Contains(t, out.String(), "Committed generated harness baseline")
	require.Contains(t, runMainTestGit(t, repoDir, "log", "--oneline", "-1"), "chore(harness): initialize mars harness")
	require.Empty(t, strings.TrimSpace(runMainTestGit(t, repoDir, "status", "--short")))
}

func TestEjectCommandDryRunLeavesRepoAndDBUntouched(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	require.NoError(t, scanner.Init(repoDir, false))
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("db\n"), 0o644))

	cmd := ejectCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Mars Harness eject dry-run")
	require.Contains(t, out.String(), "Run with --apply --confirm")
	require.FileExists(t, filepath.Join(repoDir, ".harness", "manifest.yaml"))
	require.FileExists(t, dbPath)
}

func TestEjectCommandApplyRequiresConfirmation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	require.NoError(t, scanner.Init(repoDir, false))

	cmd := ejectCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--apply", "--confirm", "wrong"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "destructive apply requires --confirm")
	require.FileExists(t, filepath.Join(repoDir, ".harness", "manifest.yaml"))
}

func TestEjectCommandApplyRemovesRepoAndDBArtifacts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	require.NoError(t, scanner.Init(repoDir, false))
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	for _, path := range []string{dbPath, dbPath + "-shm", dbPath + "-wal"} {
		require.NoError(t, os.WriteFile(path, []byte("db\n"), 0o644))
	}

	cmd := ejectCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--apply",
		"--confirm", filepath.Base(repoDir),
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Mars Harness eject applied")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
	require.NoFileExists(t, filepath.Join(repoDir, "AGENTS.md"))
	require.NoDirExists(t, filepath.Join(repoDir, "docs", "tickets"))
	require.NoFileExists(t, dbPath)
	require.NoFileExists(t, dbPath+"-shm")
	require.NoFileExists(t, dbPath+"-wal")
}

func TestRunCommandAutoInitCommitsGeneratedHarnessBaseline(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	cmd := runCmd()
	cmd.SetArgs([]string{"ceo", "--repo", repoDir, "--dry-run"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, runMainTestGit(t, repoDir, "log", "--oneline", "-1"), "chore(harness): initialize mars harness")
	require.Empty(t, strings.TrimSpace(runMainTestGit(t, repoDir, "status", "--short")))
}

func TestRegisterCommandAutoInitCommitsGeneratedHarnessBaseline(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	cmd := registerCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath})

	require.NoError(t, cmd.Execute())
	require.Contains(t, runMainTestGit(t, repoDir, "log", "--oneline", "-1"), "chore(harness): initialize mars harness")
	require.Empty(t, strings.TrimSpace(runMainTestGit(t, repoDir, "status", "--short")))
}

func TestScanCommandAutoInitCommitsGeneratedHarnessBaseline(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	cmd := scanCmd()
	cmd.SetArgs([]string{"--repo", repoDir})

	require.NoError(t, cmd.Execute())
	require.Contains(t, runMainTestGit(t, repoDir, "log", "--oneline", "-1"), "chore(harness): initialize mars harness")
	require.Empty(t, strings.TrimSpace(runMainTestGit(t, repoDir, "status", "--short")))
}

func TestCommitGeneratedHarnessBaselineLeavesExistingChangesUnstaged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "scratch.txt"), []byte("user work\n"), 0o644))
	preInitChanges, err := gitChangedPaths(repoDir)
	require.NoError(t, err)

	didInit, err := scanner.EnsureHarness(repoDir, false)
	require.NoError(t, err)
	require.True(t, didInit)
	committed, err := commitGeneratedHarnessBaseline(repoDir, preInitChanges)
	require.NoError(t, err)
	require.True(t, committed)

	log := runMainTestGit(t, repoDir, "log", "--oneline", "-1")
	require.Contains(t, log, "chore(harness): initialize mars harness")
	status := runMainTestGit(t, repoDir, "status", "--short")
	require.Equal(t, "?? scratch.txt", strings.TrimSpace(status))
	tracked := runMainTestGit(t, repoDir, "ls-files")
	require.Contains(t, tracked, ".harness/manifest.yaml")
	require.NotContains(t, tracked, "scratch.txt")
}

func writeToolRunRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func runMainTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}
