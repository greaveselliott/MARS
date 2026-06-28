/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/dashboard.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/local-inference.md
- docs/design-docs/release-versioning.md
- docs/design-docs/self-reflective-telemetry.md
- docs/product-specs/product-surface.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-012-self-improvement-loop.md
*/
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/foundationtelemetry"
	"github.com/greaveselliott/mars/internal/queue"
	"github.com/greaveselliott/mars/internal/release"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/scoring"
	"github.com/greaveselliott/mars/internal/selfupdate"
	"github.com/greaveselliott/mars/internal/serve"
	harnesstools "github.com/greaveselliott/mars/internal/tools"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestToolsListCommandIncludesUniversalTools(t *testing.T) {
	t.Parallel()
	cmd := toolsListCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Contains(t, lines, "mars_cli")
	require.Contains(t, lines, "github_auth_check")
	require.Contains(t, lines, "tool_create")
	require.Contains(t, lines, "tool_creation_guard")
}

func TestCodeIntelMetricsCommandReportsEmptyLocalEvidence(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	cmd := codeIntelMetricsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--repo", repo, "--db", dbPath})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Code-intel metrics:")
	require.Contains(t, out.String(), "Jobs: 0")
}

func TestCodeIntelBenchmarkCommandRunsLocalControlTreatment(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeToolRunRepoFile(t, repo, "go.mod", "module example.test/app\n\ngo 1.22\n")
	writeToolRunRepoFile(t, repo, "internal/app/app.go", "package app\n\nfunc Run() string { return \"ok\" }\n")
	writeToolRunRepoFile(t, repo, "internal/app/app_test.go", "package app\n\nfunc TestRun(t *testing.T) {}\n")
	writeToolRunRepoFile(t, repo, "docs/design-docs/app.md", "Run is documented by internal/app/app.go.\n")
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	cmd := codeIntelBenchmarkCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repo,
		"--db", dbPath,
		"--trials", "1",
		"--changed-paths", "internal/app/app.go",
		"--expected-tests", "internal/app/app_test.go",
		"--expected-docs", "docs/design-docs/app.md",
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Code-intel benchmark:")
	require.Contains(t, out.String(), "Control:")
	require.Contains(t, out.String(), "Treatment:")
	require.Contains(t, out.String(), "Expected tests hit rate: 1.00")
	require.Contains(t, out.String(), "Expected docs hit rate: 1.00")
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
	writeToolRunRepoFile(t, dir, "go.mod", "module github.com/greaveselliott/mars\n\ngo 1.24\n")
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

func TestVersionEntrypointsPrintSameVersionLine(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"version"},
		{"--version"},
		{"-v"},
	} {
		args := args
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			cmd := newRootCommand()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(args)

			require.NoError(t, cmd.Execute())
			require.Equal(t, versionLine()+"\n", out.String())
		})
	}
}

func TestAuthGitHubCheckCommandReportsMissingAuthWithoutSecrets(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("MARS_GITHUB_TOKEN", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	cmd := authGitHubCheckCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, out.String(), `"auth_source": "none"`)
	require.Contains(t, out.String(), "mars auth github setup")
	require.NotContains(t, out.String(), "Bearer")
	require.NotContains(t, out.String(), "ghs_")
}

func TestMarsCLIToolReferenceTracksCommandTree(t *testing.T) {
	t.Parallel()
	reference := harnesstools.MarsCLIReference()
	for _, path := range runnableCommandPaths(newRootCommand()) {
		require.Truef(t, commandReferenceContainsPath(reference, path), "mars_cli reference must document CLI command %q", path)
	}
}

func TestMarsCLIRepoShortcutTracksRepoPathFlags(t *testing.T) {
	t.Parallel()
	for _, cmd := range runnableCommands(newRootCommand()) {
		path := commandPathWithoutRoot(cmd)
		if hasWorkspaceRepoFlag(cmd) {
			require.Truef(t, harnesstools.MarsCommandSupportsRepo(strings.Fields(path)), "mars_cli repo shortcut must support %q because it has a workspace --repo flag", path)
			continue
		}
		require.Falsef(t, harnesstools.MarsCommandSupportsRepo(strings.Fields(path)), "mars_cli repo shortcut should not append a workspace --repo path to %q", path)
	}
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

func TestReleaseBackfillNotesCommandChecksAndWrites(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	dir := initReleaseBackfillCommandRepo(t)

	checkCmd := releaseBackfillNotesCmd()
	var checkOut bytes.Buffer
	checkCmd.SetOut(&checkOut)
	checkCmd.SetArgs([]string{"--repo", dir, "--check"})
	err := checkCmd.Execute()
	require.Error(t, err)
	require.Contains(t, checkOut.String(), "Release backfill-notes: checked 1 entries, changed 1")

	writeCmd := releaseBackfillNotesCmd()
	var writeOut bytes.Buffer
	writeCmd.SetOut(&writeOut)
	writeCmd.SetArgs([]string{"--repo", dir, "--min-version", "0.1.0", "--max-version", "0.1.0"})
	require.NoError(t, writeCmd.Execute())
	require.Contains(t, writeOut.String(), "Updated files:")

	changelog, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	require.NoError(t, err)
	require.Contains(t, string(changelog), "### Impact")
	require.Contains(t, string(changelog), "### Why")
	require.Contains(t, string(changelog), "### What Changed")
	require.NotContains(t, string(changelog), "### Why This Release Matters")
}

func TestChecksRunRecordsFailedOutcome(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	repoDir := t.TempDir()

	cmd := checksRunCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--name", "unit-failure",
		"--", "sh", "-c", "exit 7",
	})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, out.String(), `Recorded check "unit-failure" as checks_failed`)

	store, err := scoring.OpenStore(dbPath)
	require.NoError(t, err)
	defer store.Close()
	counts, err := store.OutcomeCounts(context.Background(), repoDir, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, counts, 1)
	require.Equal(t, scoring.OutcomeChecksFailed, counts[0].Type)
	require.Equal(t, "engineer", counts[0].Role)
}

func TestReleasePublishAssetsCommandBuildsLocalDist(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not in PATH")
	}
	repoDir := initReleaseAssetRepo(t)
	distDir := filepath.Join(t.TempDir(), "release")

	cmd := releasePublishAssetsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--version", "v1.2.3",
		"--dist", distDir,
		"--upload", "none",
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Status: ok")
	for _, name := range selfupdate.ExpectedReleaseAssetNames() {
		require.FileExists(t, filepath.Join(distDir, name))
	}
}

func TestReleaseVerifyAssetsCommandChecksLocalDist(t *testing.T) {
	t.Parallel()
	distDir := t.TempDir()
	var checksumLines []string
	for _, name := range selfupdate.ExpectedReleaseAssetNames() {
		if name == "checksums.txt" {
			continue
		}
		path := filepath.Join(distDir, name)
		require.NoError(t, os.WriteFile(path, []byte(name+"\n"), 0o644))
		sum := sha256.Sum256([]byte(name + "\n"))
		checksumLines = append(checksumLines, fmt.Sprintf("%x  %s", sum, name))
	}
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "checksums.txt"), []byte(strings.Join(checksumLines, "\n")+"\n"), 0o644))

	cmd := releaseVerifyAssetsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--dist", distDir, "--version", "v1.2.3"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Status: ok")
}

func TestPrintReleaseAuditResultReportsFindingsAndSkips(t *testing.T) {
	t.Parallel()

	var clean bytes.Buffer
	require.NoError(t, printReleaseAuditResult(&clean, release.AuditResult{
		RepoFullName: "owner/repo",
		Checked:      []string{"v0.2.0", "v0.1.0"},
	}, false))
	require.Contains(t, clean.String(), "Status: ok")

	var findings bytes.Buffer
	err := printReleaseAuditResult(&findings, release.AuditResult{
		RepoFullName: "owner/repo",
		Checked:      []string{"v0.2.0"},
		Findings: []release.AuditFinding{{
			TagName:     "v0.2.0",
			Class:       release.AuditNotesOnly,
			Missing:     []string{"checksums.txt"},
			Remediation: "mars release publish-assets --repo . --version v0.2.0 --upload github",
		}},
	}, false)
	require.Error(t, err)
	require.Contains(t, findings.String(), "FINDING (notes_only): v0.2.0")
	require.Contains(t, findings.String(), "checksums.txt")
	require.Contains(t, findings.String(), "publish-assets --repo . --version v0.2.0")

	var skipped bytes.Buffer
	require.NoError(t, printReleaseAuditResult(&skipped, release.AuditResult{
		RepoFullName: "owner/repo",
		Skipped:      true,
		SkipReason:   "cannot list GitHub releases",
	}, false))
	require.Contains(t, skipped.String(), "Skipped: cannot list GitHub releases")

	var jsonOut bytes.Buffer
	require.NoError(t, printReleaseAuditResult(&jsonOut, release.AuditResult{
		RepoFullName: "owner/repo",
		Checked:      []string{"v0.1.0"},
	}, true))
	require.Contains(t, jsonOut.String(), `"repo_full_name": "owner/repo"`)
}

func TestDocSyncAuditCommandReportsStatus(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeToolRunRepoFile(t, dir, "docs/design-docs/code-documentation-map.md", "map")
	writeToolRunRepoFile(t, dir, "docs/design-docs/release-versioning.md", "release")
	writeToolRunRepoFile(t, dir, "docs/features/F-009-release-update-lifecycle.md", "feature")
	writeToolRunRepoFile(t, dir, "internal/release/notes.go", `/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release
`)

	cmd := docsyncAuditCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--repo", dir})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "docsync: checked 1 files, findings 0")
	require.Contains(t, out.String(), "Status: ok")
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
	require.Contains(t, out.String(), "mars register --repo")
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
	require.Contains(t, out.String(), "mars register --repo")
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

func TestTelemetryTriageFoundationRequiresDistinctInstallOrVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "intake.db")
	store, err := foundationtelemetry.OpenSQLiteStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	now := time.Now().UTC()
	upsert := func(hash, version string) {
		t.Helper()
		require.NoError(t, store.UpsertPattern(ctx, foundationtelemetry.AggregatedPattern{
			Signature:       "sig-foundation",
			ReportHash:      hash,
			ReportKey:       "rk-one-install",
			FirstSeen:       now.Add(-time.Hour),
			LastSeen:        now,
			ReportCount:     1,
			HarnessVersions: []string{version},
			Category:        "max_turns",
			Target:          "skill",
			Severity:        "medium",
		}))
	}
	upsert("hash-one", "0.30.1")
	upsert("hash-two", "0.30.1")
	upsert("hash-three", "0.30.1")

	repoDir := t.TempDir()
	cmd := telemetryTriageFoundationCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--db", dbPath, "--repo", repoDir})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Triaged 0 foundation telemetry pattern(s).")
	require.NoDirExists(t, filepath.Join(repoDir, "docs", "tickets", "backlog"))

	upsert("hash-four", "0.30.2")
	cmd = telemetryTriageFoundationCmd()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--db", dbPath, "--repo", repoDir})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Triaged 1 foundation telemetry pattern(s).")
	entries, err := os.ReadDir(filepath.Join(repoDir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(repoDir, "docs", "tickets", "backlog", entries[0].Name()))
	require.NoError(t, err)
	require.Contains(t, string(data), "foundation-telemetry:sig-foundation")
	require.Contains(t, string(data), "kind: intervention-debt")
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
	require.Contains(t, out.String(), "mars register --repo")
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
	require.Contains(t, out.String(), "mars register --repo")
	require.NotContains(t, out.String(), "unable to open database file")
	require.NotContains(t, out.String(), "(14)")
}

func TestRunStartServeExposeDebugAndLogFileFlags(t *testing.T) {
	for name, cmd := range map[string]*cobra.Command{
		"run":   runCmd(),
		"start": startCmd(),
		"serve": serveCmd(),
	} {
		require.NotNil(t, cmd.Flags().Lookup("debug"), "%s missing --debug", name)
		require.NotNil(t, cmd.Flags().Lookup("log-file"), "%s missing --log-file", name)
	}
}

func TestStartCommandExposesRealModelEndpointOverride(t *testing.T) {
	cmd := startCmd()
	require.NotNil(t, cmd.Flags().Lookup("model-endpoint"))
}

func TestStartCommandExposesParallelAddressControls(t *testing.T) {
	cmd := startCmd()
	require.NotNil(t, cmd.Flags().Lookup("addr"))
	require.NotNil(t, cmd.Flags().Lookup("dashboard-addr"))
}

func TestScoresExportExposesCreateInterventionDebtFlag(t *testing.T) {
	cmd := scoresExportCmd()
	require.NotNil(t, cmd.Flags().Lookup("create-intervention-debt"))
}

func TestStartCommandInitializesRegistersSeedsAndStops(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	logPath := filepath.Join(t.TempDir(), "start.log")
	t.Setenv("MARS_WEBHOOK_PORT", "0")
	t.Setenv("MARS_DASHBOARD_PORT", "0")
	t.Setenv("MARS_SKIP_START_CLEANUP", "1")

	cmd := startCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--log-file", logPath,
		"--model-endpoint", "http://127.0.0.1:9999/v1",
		"--exit-after-seed",
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Registered repo")
	require.Contains(t, out.String(), "Seeded CEO agent")
	require.Contains(t, out.String(), "Committed generated harness baseline")
	require.FileExists(t, filepath.Join(repoDir, ".harness", "manifest.yaml"))
	require.FileExists(t, dbPath)
	require.FileExists(t, logPath)

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

	second := startCmd()
	var secondOut bytes.Buffer
	second.SetOut(&secondOut)
	second.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--log-file", logPath,
		"--model-endpoint", "http://127.0.0.1:9999/v1",
		"--exit-after-seed",
	})
	require.NoError(t, second.Execute())
	jobs, err = q.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "restarting bootstrap should reuse the active CEO job")

	log := runMainTestGit(t, repoDir, "log", "--oneline", "-1")
	require.Contains(t, log, "chore(harness): initialize mars harness")
	status := runMainTestGit(t, repoDir, "status", "--short")
	require.Empty(t, strings.TrimSpace(status))
}

func TestStartCommandRoutesExistingInProgressTicketInsteadOfSeedingCEO(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	logPath := filepath.Join(t.TempDir(), "start.log")
	t.Setenv("MARS_WEBHOOK_PORT", "0")
	t.Setenv("MARS_DASHBOARD_PORT", "0")
	t.Setenv("MARS_SKIP_START_CLEANUP", "1")

	preInitChanges, err := gitChangedPaths(repoDir)
	require.NoError(t, err)
	didInit, err := scanner.EnsureHarness(repoDir, false)
	require.NoError(t, err)
	require.True(t, didInit)
	committed, err := commitGeneratedHarnessBaseline(repoDir, preInitChanges)
	require.NoError(t, err)
	require.True(t, committed)

	ticketDir := filepath.Join(repoDir, "docs", "tickets", "in-progress")
	require.NoError(t, os.MkdirAll(ticketDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ticketDir, "T-001-first-slice.md"), []byte(`---
id: T-001
title: First slice
blocker: none
blocked_by: []
---

# First slice
`), 0o644))

	cmd := startCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--log-file", logPath,
		"--model-endpoint", "http://127.0.0.1:9999/v1",
		"--exit-after-seed",
	})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "startup_action=routed_existing_ticket")
	require.Contains(t, out.String(), "role=engineer")
	require.Contains(t, out.String(), "Resuming lifecycle with engineer job")
	require.NotContains(t, out.String(), "Seeded CEO agent")

	q, err := queue.Open(dbPath)
	require.NoError(t, err)
	defer q.Close()
	jobs, err := q.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, "engineer", jobs[0].Role)
	require.Contains(t, jobs[0].Trigger, `"type":"startup_reconciliation"`)
	require.Contains(t, jobs[0].Trigger, `"ticket_id":"T-001"`)
}

func TestStartCommandRefusesDirtyWorkspaceWithoutDeterministicRoute(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	logPath := filepath.Join(t.TempDir(), "start.log")
	t.Setenv("MARS_WEBHOOK_PORT", "0")
	t.Setenv("MARS_DASHBOARD_PORT", "0")
	t.Setenv("MARS_SKIP_START_CLEANUP", "1")

	preInitChanges, err := gitChangedPaths(repoDir)
	require.NoError(t, err)
	didInit, err := scanner.EnsureHarness(repoDir, false)
	require.NoError(t, err)
	require.True(t, didInit)
	committed, err := commitGeneratedHarnessBaseline(repoDir, preInitChanges)
	require.NoError(t, err)
	require.True(t, committed)
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "index.html"), []byte("<main>dirty product</main>\n"), 0o644))

	cmd := startCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--db", dbPath,
		"--log-file", logPath,
		"--model-endpoint", "http://127.0.0.1:9999/v1",
		"--exit-after-seed",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "refused_ambiguous_state")
	require.Contains(t, out.String(), "dirty workspace without deterministic ticket route")
	require.NotContains(t, out.String(), "Seeded CEO agent")
}

func TestStartCommandRejectsRepoLocalDBPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(repoDir, "mars.db")
	t.Setenv("MARS_WEBHOOK_PORT", "0")
	t.Setenv("MARS_DASHBOARD_PORT", "0")
	t.Setenv("MARS_SKIP_START_CLEANUP", "1")

	cmd := startCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath, "--exit-after-seed"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--db path")
	require.Contains(t, err.Error(), "inside target repo")
	require.NoFileExists(t, dbPath)
	require.NoFileExists(t, dbPath+"-wal")
	require.NoFileExists(t, dbPath+"-shm")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
}

func TestRegisterCommandRejectsRepoLocalDBPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	dbPath := filepath.Join(repoDir, "mars.db")

	cmd := registerCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--db path")
	require.Contains(t, err.Error(), "inside target repo")
	require.NoFileExists(t, dbPath)
	require.NoFileExists(t, dbPath+"-wal")
	require.NoFileExists(t, dbPath+"-shm")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
}

func TestStartCommandRejectsRepoLocalLogFile(t *testing.T) {
	repoDir := t.TempDir()
	logPath := filepath.Join(repoDir, "start.log")

	cmd := startCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--log-file", logPath, "--exit-after-seed"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--log-file path")
	require.Contains(t, err.Error(), "inside target repo")
	require.NoFileExists(t, logPath)
}

func TestRunCommandRejectsRepoLocalLogFile(t *testing.T) {
	repoDir := t.TempDir()
	logPath := filepath.Join(repoDir, "run.log")

	cmd := runCmd()
	cmd.SetArgs([]string{"ceo", "--repo", repoDir, "--log-file", logPath, "--dry-run"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--log-file path")
	require.Contains(t, err.Error(), "inside target repo")
	require.NoFileExists(t, logPath)
}

func TestRunCommandNoInitDryRunDoesNotWriteUninitializedTarget(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Observer target\n"), 0o644))
	logPath := filepath.Join(t.TempDir(), "run.log")

	cmd := runCmd()
	cmd.SetArgs([]string{"engineer", "--repo", repoDir, "--log-file", logPath, "--dry-run", "--no-init"})

	require.NoError(t, cmd.Execute())
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
	require.NoFileExists(t, logPath)
}

func TestRunCommandFoundationMaintainerDryRunUsesSourceProfileWithoutInit(t *testing.T) {
	root := mainTestSourceRoot(t)
	require.NoFileExists(t, filepath.Join(root, ".harness", "manifest.yaml"))

	err := executeRun(runOpts{
		roleName: foundationMaintainerRoleName,
		repoPath: root,
		logFile:  filepath.Join(t.TempDir(), "run.log"),
		dryRun:   true,
		noInit:   true,
		budget:   4000,
	})

	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(root, ".harness", "manifest.yaml"))
}

func TestRunCommandDryRunInjectsCodeGraphContextForGeneratedEngineer(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeToolRunRepoFile(t, repoDir, "go.mod", "module example.test/target\n\ngo 1.22\n")
	writeToolRunRepoFile(t, repoDir, "internal/app/app.go", "package app\n\nfunc Run() string { return \"ok\" }\n")
	writeToolRunRepoFile(t, repoDir, "docs/features/F-001-demo.md", "Scenario covers internal/app/app.go and Run.\n")
	runMainTestGit(t, repoDir, "init")
	didInit, err := scanner.EnsureHarness(repoDir, false)
	require.NoError(t, err)
	require.True(t, didInit)

	var runErr error
	out := captureStdout(t, func() {
		runErr = executeRun(runOpts{
			roleName: "engineer",
			repoPath: repoDir,
			logFile:  filepath.Join(t.TempDir(), "run.log"),
			dryRun:   true,
			noInit:   true,
		})
	})

	require.NoError(t, runErr)
	require.Contains(t, out, "## CODE GRAPH CONTEXT")
	require.Contains(t, out, "freshness: fresh")
	require.Contains(t, out, "index_refresh:")
	require.Contains(t, out, "internal/app/app.go")
	require.Contains(t, out, "app.Run")
	require.Contains(t, out, "operator_hint: use code_search")
}

func TestRunCommandDryRunSkipsCodeGraphContextWhenDisabled(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeToolRunRepoFile(t, repoDir, "go.mod", "module example.test/target\n\ngo 1.22\n")
	writeToolRunRepoFile(t, repoDir, "internal/app/app.go", "package app\n\nfunc Run() string { return \"ok\" }\n")
	runMainTestGit(t, repoDir, "init")
	didInit, err := scanner.EnsureHarness(repoDir, false)
	require.NoError(t, err)
	require.True(t, didInit)

	var runErr error
	out := captureStdout(t, func() {
		runErr = executeRun(runOpts{
			roleName:      "engineer",
			repoPath:      repoDir,
			logFile:       filepath.Join(t.TempDir(), "run.log"),
			dryRun:        true,
			noInit:        true,
			codeIntelFlag: "false",
		})
	})

	require.NoError(t, runErr)
	require.NotContains(t, out, "## CODE GRAPH CONTEXT")
	require.NotContains(t, out, "operator_hint: use code_search")
}

func TestRunCommandFoundationMaintainerRejectsNonSourceRepo(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module example.com/not-mars\n"), 0o644))
	logPath := filepath.Join(t.TempDir(), "run.log")

	err := executeRun(runOpts{
		roleName: foundationMaintainerRoleName,
		repoPath: repoDir,
		logFile:  logPath,
		dryRun:   true,
		noInit:   true,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "source-only")
	require.Contains(t, err.Error(), "not a mars source checkout")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
	require.NoFileExists(t, logPath)
}

func TestRunCommandNoInitWithoutDryRunFailsClosed(t *testing.T) {
	repoDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "run.log")

	cmd := runCmd()
	cmd.SetArgs([]string{"engineer", "--repo", repoDir, "--log-file", logPath, "--no-init"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), ".harness/manifest.yaml is missing")
	require.Contains(t, err.Error(), "--no-init")
	require.Contains(t, err.Error(), "no files were written")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
	require.NoFileExists(t, logPath)
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

func TestInitCommandWritesCustomCloudEndpoint(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repoDir := t.TempDir()
	cmd := initCmd()
	cmd.SetArgs([]string{
		"--repo", repoDir,
		"--model-routing", "cloud",
		"--cloud-provider", "openai-compatible",
		"--cloud-model", "openai/gpt-4.1-mini",
		"--cloud-endpoint", "https://models.example.test/inference/v1",
		"--api-key-env", "GITHUB_TOKEN",
		"--yes",
		"--json",
	})

	require.NoError(t, cmd.Execute())
	data, err := os.ReadFile(filepath.Join(repoDir, ".harness", "model-overrides.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(data), "provider: openai-compatible")
	require.Contains(t, string(data), "model: openai/gpt-4.1-mini")
	require.Contains(t, string(data), "endpoint: https://models.example.test/inference/v1")
	require.Contains(t, string(data), "api_key_env: GITHUB_TOKEN")
	example, err := os.ReadFile(filepath.Join(repoDir, ".harness", ".env.example"))
	require.NoError(t, err)
	require.Contains(t, string(example), "GITHUB_TOKEN=")
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
	require.Contains(t, out.String(), "MARS eject dry-run")
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
	require.Contains(t, out.String(), "MARS eject applied")
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

func initReleaseBackfillCommandRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runMainTestGit(t, dir, "init")
	runMainTestGit(t, dir, "config", "user.email", "test@example.com")
	runMainTestGit(t, dir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0o644))
	runMainTestGit(t, dir, "add", "-A")
	runMainTestGit(t, dir, "commit", "-m", "feat(cli): add command backfill")
	head := strings.TrimSpace(runMainTestGit(t, dir, "rev-parse", "--short=12", "HEAD"))
	short := strings.TrimSpace(runMainTestGit(t, dir, "rev-parse", "--short=7", "HEAD"))
	changelog := `# Changelog

## [0.1.0] - 2026-05-01
<!-- mars-release: version=0.1.0 commit=` + head + ` -->

### Why This Release Matters
Old narrative.

### Features
- **cli:** Add command backfill (` + short + `)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(changelog), 0o644))
	return dir
}

func initReleaseAssetRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runMainTestGit(t, dir, "init")
	runMainTestGit(t, dir, "config", "user.email", "test@example.com")
	runMainTestGit(t, dir, "config", "user.name", "Test User")
	writeToolRunRepoFile(t, dir, "go.mod", "module example.com/release-asset-test\n\ngo 1.22\n")
	writeToolRunRepoFile(t, dir, "cmd/mars/main.go", `package main

import "fmt"

var version = "dev"
var commit = "unknown"
var date = "unknown"

func main() {
	fmt.Println(version, commit, date)
}
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(`# Changelog

## [1.2.3] - 2026-05-23
<!-- mars-release: version=1.2.3 commit=test -->

### Impact
Local release assets can be verified.

### Why
The release path is tested without GitHub Actions.

### What Changed
Built local assets.
`), 0o644))
	runMainTestGit(t, dir, "add", "-A")
	runMainTestGit(t, dir, "commit", "-m", "release: notes 1.2.3")
	runMainTestGit(t, dir, "tag", "v1.2.3")
	return dir
}

func writeToolRunRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()
	fn()
	require.NoError(t, w.Close())
	out := <-done
	require.NoError(t, r.Close())
	return out
}

func mainTestSourceRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func runnableCommands(root *cobra.Command) []*cobra.Command {
	var commands []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Name() == "help" || child.Hidden {
				continue
			}
			if child.Runnable() {
				commands = append(commands, child)
			}
			walk(child)
		}
	}
	walk(root)
	sort.Slice(commands, func(i, j int) bool {
		return commandPathWithoutRoot(commands[i]) < commandPathWithoutRoot(commands[j])
	})
	return commands
}

func runnableCommandPaths(root *cobra.Command) []string {
	commands := runnableCommands(root)
	paths := make([]string, 0, len(commands))
	for _, cmd := range commands {
		paths = append(paths, commandPathWithoutRoot(cmd))
	}
	return paths
}

func commandPathWithoutRoot(cmd *cobra.Command) string {
	return strings.TrimPrefix(cmd.CommandPath(), "mars ")
}

func commandReferenceContainsPath(reference, path string) bool {
	for _, line := range strings.Split(reference, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == path || strings.HasPrefix(trimmed, path+" <") {
			return true
		}
	}
	return false
}

func hasWorkspaceRepoFlag(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("repo")
	if flag == nil {
		return false
	}
	usage := strings.ToLower(flag.Usage)
	if strings.Contains(usage, "owner/name") {
		return false
	}
	return strings.Contains(usage, "path") ||
		strings.Contains(usage, "repo root") ||
		strings.Contains(usage, "repository") ||
		strings.Contains(usage, "target repo")
}

func runMainTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}
