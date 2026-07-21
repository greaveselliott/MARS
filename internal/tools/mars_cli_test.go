/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/local-inference.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/tools-glossary.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-017-open-source-publication.md
- docs/features/F-012-self-improvement-loop.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarsCLI_reference(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	res, err := handleMarsCLI(context.Background(), root, []byte(`{"mode":"reference"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "mars_cli reference")
	for _, command := range []string{
		"setup", "init", "eject", "upgrade", "start", "serve", "register", "run <role>",
		"scan", "doctor", "auth github check", "auth github setup", "update check", "update tool", "update harness",
		"path setup", "release notes", "release backfill-notes", "release verify-assets", "checks run", "scores", "scores export",
		"telemetry status", "telemetry preview", "telemetry export", "telemetry send", "telemetry collect", "telemetry triage-foundation",
		"docsync audit", "trust", "trust set", "models evaluate", "models eligible", "models list", "models override", "models credentials write-local-env", "guardrails secret-scan", "guardrails install-hooks",
		"tools list", "tools run <name>", "mcp serve",
	} {
		require.Contains(t, res.Output, command)
	}
	require.NotContains(t, res.Output, "release publish-assets")
	require.Contains(t, res.Output, "--model-endpoint <real-url>")
	require.Contains(t, res.Output, "--dashboard-addr <host:port>")
	require.Contains(t, res.Output, "--dashboard-trusted-origin <exact-https-origin>")
	require.Contains(t, res.Output, "MARS_DASHBOARD_CONTROL_SECRET")
	require.Contains(t, res.Output, "Anonymous loopback dashboard access is limited")
	require.Contains(t, res.Output, "rich reads, SSE, and mutation require")
	require.Contains(t, res.Output, "Default scoped starts fall back to ephemeral local control/dashboard ports")
}

func TestMarsCLI_repoShortcutAppendsRepoFlagForSyncedCommands(t *testing.T) {
	for _, args := range [][]string{
		{"release", "backfill-notes", "--dry-run"},
		{"checks", "run", "--name", "unit", "--", "go", "test", "./..."},
		{"docsync", "audit"},
		{"models", "evaluate", "--json"},
		{"models", "override", "--tier", "coding", "--provider", "ollama", "--model", "qwen"},
		{"models", "credentials", "write-local-env", "--api-key-env", "ANTHROPIC_API_KEY", "--yes", "--json"},
		{"guardrails", "secret-scan", "--staged", "--json"},
		{"guardrails", "install-hooks", "--json"},
		{"scores"},
		{"telemetry", "preview"},
		{"telemetry", "triage-foundation", "--db", "intake.db"},
		{"trust"},
	} {
		t.Run(strings.Join(args[:min(2, len(args))], "_"), func(t *testing.T) {
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			bin := writeFakeMarsBinary(t, dir)
			t.Setenv("MARS_CLI_BIN", bin)

			raw, err := json.Marshal(marsCLIArgs{
				Mode:           "run",
				Args:           args,
				Repo:           ".",
				TimeoutSeconds: 5,
			})
			require.NoError(t, err)
			res, err := handleMarsCLI(context.Background(), root, raw)
			require.NoError(t, err)
			require.Contains(t, res.Output, "--repo "+root.Abs())
		})
	}
}

func TestMarsCLI_runUsesStructuredArgv(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsBinary(t, dir)
	t.Setenv("MARS_CLI_BIN", bin)

	res, err := handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["version"],
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Equal(t, "fake-mars version", strings.TrimSpace(res.Output))
}

func TestMarsCLI_normalizesModelMalformedArgs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "json encoded args string",
			raw: `{
				"mode": "run",
				"args": "[\"version\"]",
				"timeout_seconds": 5
			}`,
		},
		{
			name: "python style args string",
			raw: `{
				"mode": "run",
				"args": "['version']",
				"timeout_seconds": 5
			}`,
		},
		{
			name: "single simple command string",
			raw: `{
				"mode": "run",
				"args": "version",
				"timeout_seconds": 5
			}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			bin := writeFakeMarsBinary(t, dir)
			t.Setenv("MARS_CLI_BIN", bin)

			res, err := handleMarsCLI(context.Background(), root, []byte(tt.raw))
			require.NoError(t, err)
			require.Equal(t, 0, res.ExitCode)
			require.Equal(t, "fake-mars version", strings.TrimSpace(res.Output))
		})
	}
}

func TestMarsCLI_prefersCurrentExecutableBeforePath(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	current := filepath.Join(dir, "mars-current")
	require.NoError(t, os.WriteFile(current, []byte("#!/bin/sh\n"), 0o755))
	pathDir := filepath.Join(dir, "path-bin")
	require.NoError(t, os.MkdirAll(pathDir, 0o755))
	pathBin := writeFakeMarsBinary(t, pathDir)
	t.Setenv("PATH", filepath.Dir(pathBin))

	argv, err := marsCommandArgvWithExecutable(root, []string{"release", "notes"}, current, nil)
	require.NoError(t, err)
	require.Equal(t, current, argv[0])
	require.Equal(t, []string{current, "release", "notes"}, argv)
}

func TestMarsCLI_prefersSourceCheckoutBeforePathMars(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "mars"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "mars", "main.go"), []byte("package main\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/greaveselliott/mars\n"), 0o644))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	pathDir := filepath.Join(dir, "path-bin")
	require.NoError(t, os.MkdirAll(pathDir, 0o755))
	_ = writeFakeMarsBinary(t, pathDir)
	t.Setenv("PATH", pathDir)

	argv, err := marsCommandArgvWithExecutable(root, []string{"version"}, "", os.ErrNotExist)
	require.NoError(t, err)
	require.Equal(t, []string{"go", "run", "./cmd/mars", "version"}, argv)
}

func TestMarsCLI_usesLegacyEnvFallback(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsBinary(t, dir)
	t.Setenv("MARS_HARNESS_CLI_BIN", bin)

	res, err := handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["version"],
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Equal(t, "fake-mars version", strings.TrimSpace(res.Output))
}

func TestMarsCLI_usesLegacyPathBinaryFallback(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	legacyBin := filepath.Join(binDir, "mars-harness")
	require.NoError(t, os.WriteFile(legacyBin, []byte("#!/bin/sh\necho legacy-mars \"$@\"\n"), 0o755))
	t.Setenv("PATH", binDir)

	argv, err := marsCommandArgvWithExecutable(root, []string{"version"}, "", os.ErrNotExist)
	require.NoError(t, err)
	require.Equal(t, []string{legacyBin, "version"}, argv)
}

func TestMarsCLI_stalePathBinaryAddsActionableGuidance(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeStaleMarsBinary(t, dir)
	t.Setenv("PATH", filepath.Dir(bin))

	res, err := handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["release", "notes", "--repo", ".", "--bump", "auto"],
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.NotEqual(t, 0, res.ExitCode)
	require.Contains(t, res.Stderr, "unknown command \"release\"")
	require.Contains(t, res.Stderr, "resolved binary")
	require.Contains(t, res.Stderr, "MARS_CLI_BIN")
	require.Contains(t, res.Stderr, "mars update tool")
}

func TestMarsCLI_repoShortcutAppendsRepoFlag(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsBinary(t, dir)
	t.Setenv("MARS_CLI_BIN", bin)

	res, err := handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["scan"],
		"repo": ".",
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "args: scan --repo "+root.Abs())
}

func TestMarsCLI_repoShortcutAppendsRepoFlagForToolsRun(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsBinary(t, dir)
	t.Setenv("MARS_CLI_BIN", bin)

	res, err := handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["tools", "run", "tool_creation_guard", "--args-json", "{}"],
		"repo": ".",
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "args: tools run tool_creation_guard --args-json {} --repo "+root.Abs())
}

func TestMarsCLI_repoShortcutRejectsUnsupportedCommand(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	_, err = handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["version"],
		"repo": "."
	}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support --repo")
}

func TestMarsCLI_rejectsEmptyArg(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	_, err = handleMarsCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["doctor", ""]
	}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be non-empty")
}

func TestDefaultRegistry_includesMarsCLI(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	require.Contains(t, reg.Names(), "mars_cli")
	require.Contains(t, reg.Names(), "mars_harness_cli")
	_, _, ok := reg.Lookup("mars_harness_cli")
	require.True(t, ok)
}

func writeFakeMarsBinary(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake binary is POSIX-only")
	}
	path := filepath.Join(dir, "mars")
	script := `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "fake-mars version"
  exit 0
fi
printf 'args:'
for arg in "$@"; do
  printf ' %s' "$arg"
done
printf '\n'
`
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func writeStaleMarsBinary(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake binary is POSIX-only")
	}
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, "mars")
	script := `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "mars 0.0.1-dev darwin/arm64 commit=unknown built=unknown"
  exit 0
fi
echo "Error: unknown command \"$1\" for \"mars\"" >&2
exit 1
`
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}
