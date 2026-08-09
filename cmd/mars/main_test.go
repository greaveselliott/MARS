/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/dashboard.md
- docs/design-docs/github-app-integration.md
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
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
- docs/features/F-012-self-improvement-loop.md
*/
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/foundationtelemetry"
	"github.com/greaveselliott/mars/internal/queue"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/scoring"
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

func TestToolsRunCommandCleansBackgroundProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("background cleanup test is unix-specific")
	}
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())
	pidFile := filepath.Join(dir, "child.pid")
	argsJSON, err := json.Marshal(map[string]any{
		"shell_command": fmt.Sprintf("echo $$ > %q; sleep 30", pidFile),
		"background":    true,
	})
	require.NoError(t, err)

	cmd := toolsRunCmd()
	cmd.SetArgs([]string{
		"shell_exec",
		"--repo", dir,
		"--trust", "contributor",
		"--execution-profile", "host",
		"--acknowledge-host-execution",
		"--args-json", string(argsJSON),
	})
	require.NoError(t, cmd.Execute())

	pid := readMainTestPID(t, pidFile)
	require.Eventually(t, func() bool { return syscall.Kill(pid, 0) != nil }, 2*time.Second, 25*time.Millisecond)
}

func TestRunCommandCleansBackgroundProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("background cleanup test is unix-specific")
	}
	t.Setenv("HOME", t.TempDir())
	repoDir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", repoDir).Run())
	pidFile := filepath.Join(repoDir, "child.pid")
	writeToolRunRepoFile(t, repoDir, ".harness/manifest.yaml", `name: cleanup-run
roles:
  worker:
    prompt: roles/worker.md
    model: test
    trust_level: contributor
    tools: [shell_exec]
`)
	writeToolRunRepoFile(t, repoDir, ".harness/roles/worker.md", "Run one bounded validation command.\n")
	toolArgs, err := json.Marshal(map[string]any{
		"shell_command": fmt.Sprintf("echo $$ > %q; sleep 30", pidFile),
		"background":    true,
	})
	require.NoError(t, err)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if requests.Add(1) == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
				"message": map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{
					"id": "call_1", "type": "function", "function": map[string]any{"name": "shell_exec", "arguments": string(toolArgs)},
				}}},
				"finish_reason": "tool_calls",
			}}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"message": map[string]any{"role": "assistant", "content": "done"}, "finish_reason": "stop",
		}}})
	}))
	t.Cleanup(server.Close)

	cmd := runCmd()
	cmd.SetArgs([]string{
		"worker",
		"--repo", repoDir,
		"--model-endpoint", server.URL,
		"--log-file", filepath.Join(t.TempDir(), "run.log"),
		"--code-intel", "false",
		"--max-turns", "2",
		"--no-init",
		"--execution-profile", "host",
		"--acknowledge-host-execution",
	})
	require.NoError(t, cmd.Execute())

	pid := readMainTestPID(t, pidFile)
	require.Eventually(t, func() bool { return syscall.Kill(pid, 0) != nil }, 2*time.Second, 25*time.Millisecond)
}

func readMainTestPID(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	require.NoError(t, err)
	return pid
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
		"--execution-profile", "host",
		"--acknowledge-host-execution",
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
		"--trust", "contributor",
		"--args-json", `{"path":"x.txt","content":"x"}`,
	})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution profile observer")
	require.NoFileExists(t, filepath.Join(dir, "x.txt"))
}

func TestToolsRunHostAcknowledgementDoesNotUpgradeTrust(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cmd := toolsRunCmd()
	cmd.SetArgs([]string{
		"file_write",
		"--repo", dir,
		"--trust", "observer",
		"--execution-profile", "host",
		"--acknowledge-host-execution",
		"--args-json", `{"path":"x.txt","content":"x"}`,
	})

	err := cmd.Execute()
	require.ErrorContains(t, err, "observer cannot run mutating tool")
	require.NoFileExists(t, filepath.Join(dir, "x.txt"))
}

func TestRunHostAcknowledgementPreservesObserverRoleTrust(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repoDir := t.TempDir()
	writeToolRunRepoFile(t, repoDir, ".harness/manifest.yaml", `name: observer-run
roles:
  observer-role:
    prompt: roles/observer.md
    model: test
    trust_level: observer
    tools: [file_write]
`)
	writeToolRunRepoFile(t, repoDir, ".harness/roles/observer.md", "Remain read-only.\n")

	var requests atomic.Int32
	var sawObserverBlock atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if bytes.Contains(body, []byte("observer cannot run mutating tool")) {
			sawObserverBlock.Store(true)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"file_write","arguments":"{\"path\":\"blocked.txt\",\"content\":\"blocked\"}"}}]},"finish_reason":"tool_calls"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(server.Close)

	cmd := runCmd()
	cmd.SetArgs([]string{
		"observer-role",
		"--repo", repoDir,
		"--model-endpoint", server.URL,
		"--log-file", filepath.Join(t.TempDir(), "run.log"),
		"--code-intel", "false",
		"--max-turns", "2",
		"--no-init",
		"--execution-profile", "host",
		"--acknowledge-host-execution",
	})

	require.NoError(t, cmd.Execute())
	require.True(t, sawObserverBlock.Load(), "model should receive the observer trust policy rejection")
	require.NoFileExists(t, filepath.Join(repoDir, "blocked.txt"))
}

func TestAgentEntrypointsExposeExecutionProfileAdmission(t *testing.T) {
	t.Parallel()
	for name, cmd := range map[string]*cobra.Command{
		"run":       runCmd(),
		"start":     startCmd(),
		"serve":     serveCmd(),
		"tools run": toolsRunCmd(),
		"mcp serve": mcpServeCmd(),
	} {
		profile := cmd.Flags().Lookup("execution-profile")
		require.NotNil(t, profile, "%s missing --execution-profile", name)
		require.Equal(t, "observer", profile.DefValue, "%s must default to observer", name)
		require.NotNil(t, cmd.Flags().Lookup("acknowledge-host-execution"), "%s missing host acknowledgement", name)
	}
}

func TestAgentEntrypointsRejectUnadmittedProfilesBeforeState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		name string
		new  func() *cobra.Command
		args func(string) []string
	}{
		{name: "run", new: runCmd, args: func(repo string) []string { return []string{"engineer", "--repo", repo, "--dry-run"} }},
		{name: "start", new: startCmd, args: func(repo string) []string { return []string{"--repo", repo, "--exit-after-seed"} }},
		{name: "serve", new: serveCmd, args: func(string) []string { return nil }},
		{name: "tools run", new: toolsRunCmd, args: func(repo string) []string { return []string{"git_status", "--repo", repo} }},
		{name: "mcp serve", new: mcpServeCmd, args: func(repo string) []string { return []string{"--repo", repo} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, profile := range []string{"isolated", "host"} {
				repo := t.TempDir()
				cmd := tc.new()
				args := append(tc.args(repo), "--execution-profile", profile)
				cmd.SetArgs(args)
				err := cmd.Execute()
				require.Error(t, err)
				if profile == "isolated" {
					require.ErrorContains(t, err, "no enforceable isolation adapter")
				} else {
					require.ErrorContains(t, err, "--acknowledge-host-execution")
				}
				require.NoDirExists(t, filepath.Join(repo, ".harness"))
			}
		})
	}
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

func TestAuthGitHubCheckCommandUsesAnonymousReleaseAccessWithoutSecrets(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-must-not-be-read")
	t.Setenv("GITHUB_TOKEN", "github-token-must-not-be-read")
	t.Setenv("MARS_GITHUB_TOKEN", "config-env-token-must-not-be-read")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	requests := 0
	http.DefaultTransport = mainTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "https://api.github.com/repos/greaveselliott/MARS/releases/latest", req.URL.String())
		require.Empty(t, req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"name":"public release"}`)),
		}, nil
	})

	cmd := authGitHubCheckCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, 1, requests)
	require.Contains(t, out.String(), `"access_class": "anonymous"`)
	require.Contains(t, out.String(), `"auth_source": "none"`)
	require.NotContains(t, out.String(), "Bearer")
	require.NotContains(t, out.String(), "must-not-be-read")
}

func TestAuthGitHubClearLocalCommandTouchesOnlySelectedStoredFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GH_TOKEN", "env-gh-token")
	t.Setenv("GITHUB_TOKEN", "env-github-token")
	t.Setenv("MARS_GITHUB_TOKEN", "env-config-token")
	t.Setenv("PATH", t.TempDir())

	legacyPath := filepath.Join(home, ".mars-harness", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o700))
	legacyContent := []byte("github_token: legacy-token\nmodels_dir: /legacy/models\n")
	require.NoError(t, os.WriteFile(legacyPath, legacyContent, 0o600))
	selectedPath := filepath.Join(t.TempDir(), "selected.yaml")
	require.NoError(t, os.WriteFile(selectedPath, []byte("github_token: selected-token\nmodels_dir: /selected/models\n"), 0o600))

	cmd := authGitHubClearLocalCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--config", selectedPath, "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), `"cleared": true`)
	selected, err := os.ReadFile(selectedPath)
	require.NoError(t, err)
	require.NotContains(t, string(selected), "github_token")
	require.NotContains(t, string(selected), "selected-token")
	require.Contains(t, string(selected), "models_dir: /selected/models")
	info, err := os.Stat(selectedPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	legacyAfter, err := os.ReadFile(legacyPath)
	require.NoError(t, err)
	require.Equal(t, legacyContent, legacyAfter)
	require.Equal(t, "env-gh-token", os.Getenv("GH_TOKEN"))
	require.Equal(t, "env-github-token", os.Getenv("GITHUB_TOKEN"))
	require.Equal(t, "env-config-token", os.Getenv("MARS_GITHUB_TOKEN"))

	second := authGitHubClearLocalCmd()
	var secondOut bytes.Buffer
	second.SetOut(&secondOut)
	second.SetArgs([]string{"--config", selectedPath, "--json"})
	require.NoError(t, second.Execute())
	require.Contains(t, secondOut.String(), `"cleared": false`)
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

func TestMCPServeObserverProfileCapsContributorTrust(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	cmd := mcpServeCmd()
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"file_write","arguments":{"path":"x.txt","content":"x"}}}` + "\n"))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--repo", repo, "--trust", "contributor"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "execution profile observer")
	require.NoFileExists(t, filepath.Join(repo, "x.txt"))
}

func TestToolEntrypointsRejectInvalidTrustBeforeRepoAccess(t *testing.T) {
	t.Parallel()
	for name, cmd := range map[string]*cobra.Command{
		"tools run": toolsRunCmd(),
		"mcp serve": mcpServeCmd(),
	} {
		repo := filepath.Join(t.TempDir(), "missing")
		args := []string{"--repo", repo, "--trust", "root"}
		if name == "tools run" {
			args = append([]string{"git_status"}, args...)
		}
		cmd.SetArgs(args)
		err := cmd.Execute()
		require.ErrorContains(t, err, "trust level \"root\" is invalid")
		require.NoDirExists(t, repo)
	}
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

func TestReleaseLegacyConsumerCommandsAreRetired(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"verify-assets", "audit"} {
		name := name
		t.Run(name, func(t *testing.T) {
			cmd := releaseCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{name})
			err := cmd.Execute()
			require.ErrorContains(t, err, "unknown command")

			root := newRootCommand()
			out.Reset()
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs([]string{"release", name})
			err = root.Execute()
			require.ErrorContains(t, err, "unknown command")
		})
	}
}

func TestReleasePublishAssetsCommandIsRetired(t *testing.T) {
	t.Parallel()
	cmd := releaseCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"publish-assets"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "unknown command")
	require.NotContains(t, out.String(), "Build and optionally mirror release assets")
}

func TestRootReleasePublishAssetsCommandIsRetired(t *testing.T) {
	t.Parallel()
	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"release", "publish-assets"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "unknown command")
	require.NotContains(t, out.String(), "Build and optionally mirror release assets")
}

func TestUpdateToolDryRunDoesNotExposeOrResolveReleaseURLs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/zsh")
	for _, test := range []struct {
		name string
		json bool
	}{
		{name: "text"},
		{name: "json", json: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			installDir := t.TempDir()
			args := []string{"--dry-run", "--version", "v0.69.0", "--install-dir", installDir}
			if test.json {
				args = append(args, "--json")
			}
			cmd := updateToolCmd()
			cmd.SetArgs(args)
			var runErr error
			out := captureStdout(t, func() { runErr = cmd.Execute() })
			require.NoError(t, runErr)
			lower := strings.ToLower(out)
			require.NotContains(t, lower, "http")
			require.NotContains(t, out, "Download:")
			require.NotContains(t, out, "Checksums:")
			require.NotContains(t, out, "download_url")
			require.NotContains(t, out, "checksums_url")
			require.NotContains(t, out, "requires_github_auth")
			require.NoFileExists(t, filepath.Join(installDir, ".mars-update.lock"))
			require.NoDirExists(t, filepath.Join(installDir, ".mars-update.transaction"))
			if test.json {
				require.Contains(t, out, `"release_tag": "v0.69.0"`)
			} else {
				require.Contains(t, out, "Dry run: no changes made")
			}
		})
	}
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
	require.NotNil(t, cmd.Flags().Lookup("remote"))
	require.NotNil(t, cmd.Flags().Lookup("branch"))
	require.NotNil(t, cmd.Flags().Lookup("webhook-actor-id"))
	require.NotNil(t, cmd.Flags().Lookup("dashboard-trusted-origin"))
	serve := serveCmd()
	require.NotNil(t, serve.Flags().Lookup("webhook-actor-id"))
	require.NotNil(t, serve.Flags().Lookup("dashboard-trusted-origin"))
}

func TestResolveWebhookActorIDsPrecedenceValidationAndDeduplication(t *testing.T) {
	t.Setenv("MARS_WEBHOOK_ALLOWED_ACTOR_IDS", "20,30,20")
	got, err := resolveWebhookActorIDs([]int64{10, 10}, []int64{40})
	require.NoError(t, err)
	require.Equal(t, []int64{10}, got)
	got, err = resolveWebhookActorIDs(nil, []int64{40})
	require.NoError(t, err)
	require.Equal(t, []int64{20, 30}, got)
	t.Setenv("MARS_WEBHOOK_ALLOWED_ACTOR_IDS", "")
	got, err = resolveWebhookActorIDs(nil, []int64{40, 40})
	require.NoError(t, err)
	require.Equal(t, []int64{40}, got)
	for _, raw := range []string{"not-a-number", "0", "-1"} {
		t.Setenv("MARS_WEBHOOK_ALLOWED_ACTOR_IDS", raw)
		_, err := resolveWebhookActorIDs(nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "actor")
	}
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
		"--remote", "Owner/Repo",
		"--branch", "release/Main",
		"--log-file", logPath,
		"--model-endpoint", "http://127.0.0.1:9999/v1",
		"--execution-profile", "host",
		"--acknowledge-host-execution",
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
	require.Equal(t, "owner/repo", repos[0].Remote)
	require.Equal(t, "release/Main", repos[0].Branch)

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
		"--execution-profile", "host",
		"--acknowledge-host-execution",
		"--exit-after-seed",
	})
	require.NoError(t, second.Execute())
	repos, err = registry.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "owner/repo", repos[0].Remote)
	require.Equal(t, "release/Main", repos[0].Branch)
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
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath, "--execution-profile", "host", "--acknowledge-host-execution", "--exit-after-seed"})

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
	cmd.SetArgs([]string{"--repo", repoDir, "--log-file", logPath, "--execution-profile", "host", "--acknowledge-host-execution", "--exit-after-seed"})

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
	cmd.SetArgs([]string{"ceo", "--repo", repoDir, "--log-file", logPath, "--dry-run", "--execution-profile", "host", "--acknowledge-host-execution"})

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

func TestRunObserverRejectsUninitializedTargetBeforeState(t *testing.T) {
	repoDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "run.log")

	cmd := runCmd()
	cmd.SetArgs([]string{"engineer", "--repo", repoDir, "--log-file", logPath, "--dry-run"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "execution profile observer requires an initialized target")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
	require.NoFileExists(t, logPath)
}

func TestStartObserverRejectsUninitializedTargetBeforeState(t *testing.T) {
	repoDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	logPath := filepath.Join(t.TempDir(), "start.log")

	cmd := startCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath, "--log-file", logPath, "--exit-after-seed"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "execution profile observer requires an initialized target")
	require.NoDirExists(t, filepath.Join(repoDir, ".harness"))
	require.NoFileExists(t, dbPath)
	require.NoFileExists(t, logPath)
}

func TestStartObserverRejectsForceBeforeState(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, scanner.Init(repoDir, false))
	manifestPath := filepath.Join(repoDir, ".harness", "manifest.yaml")
	before, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	logPath := filepath.Join(t.TempDir(), "start.log")

	cmd := startCmd()
	cmd.SetArgs([]string{"--repo", repoDir, "--db", dbPath, "--log-file", logPath, "--force", "--exit-after-seed"})

	err = cmd.Execute()
	require.ErrorContains(t, err, "execution profile observer blocks --force")
	after, readErr := os.ReadFile(manifestPath)
	require.NoError(t, readErr)
	require.Equal(t, before, after)
	require.NoFileExists(t, dbPath)
	require.NoFileExists(t, logPath)
}

func TestRunCommandFoundationMaintainerDryRunUsesSourceProfileWithoutInit(t *testing.T) {
	root := mainTestSourceRoot(t)
	require.NoFileExists(t, filepath.Join(root, ".harness", "manifest.yaml"))

	var runErr error
	out := captureStdout(t, func() {
		runErr = executeRun(runOpts{
			roleName: foundationMaintainerRoleName,
			repoPath: root,
			logFile:  filepath.Join(t.TempDir(), "run.log"),
			dryRun:   true,
			noInit:   true,
			budget:   4000,
		})
	})

	require.NoError(t, runErr)
	require.Contains(t, out, "active F-018 plan")
	require.Contains(t, out, "publication-disabled snapshot")
	require.Contains(t, out, "do not tag, upload, sign, announce, or publish")
	require.NotContains(t, out, "publish local release assets")
	require.NotContains(t, out, "optionally mirror them to GitHub")
	require.NoFileExists(t, filepath.Join(root, ".harness", "manifest.yaml"))
}

func TestSourceFoundationProfileRejectsSymlinkPrompt(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	promptDir := filepath.Join(root, "docs", "roles", "personas")
	require.NoError(t, os.MkdirAll(promptDir, 0o755))
	outside := filepath.Join(t.TempDir(), "foundation-maintainer.md")
	require.NoError(t, os.WriteFile(outside, []byte("outside foundation instructions"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(promptDir, "foundation-maintainer.md")))

	_, _, prompt, _, _, _, err := loadSourceFoundationRunProfile(root)
	require.ErrorContains(t, err, "symbolic links are not allowed")
	require.Empty(t, prompt)
	require.FileExists(t, outside)
}

func TestMarsSourceAdmissionRejectsSymlinkMarker(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "mars"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "scanner"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/greaveselliott/mars\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "internal", "scanner", "init.go"), []byte("package scanner\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Agent guide\n"), 0o644))
	outside := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(outside, []byte("package main\n"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "cmd", "mars", "main.go")))

	require.False(t, isMarsSourceRepo(root))
	require.FileExists(t, outside)
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
	cmd.SetArgs([]string{"ceo", "--repo", repoDir, "--dry-run", "--execution-profile", "host", "--acknowledge-host-execution"})

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

type mainTestRoundTripFunc func(*http.Request) (*http.Response, error)

func (f mainTestRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
