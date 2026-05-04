/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
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

func TestMarsHarnessCLI_reference(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	res, err := handleMarsHarnessCLI(context.Background(), root, []byte(`{"mode":"reference"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "mars_harness_cli reference")
	for _, command := range []string{
		"setup", "init", "eject", "upgrade", "start", "serve", "register", "run <role>",
		"scan", "doctor", "update check", "update tool", "update harness",
		"path setup", "release notes", "release backfill-notes", "release verify-assets", "scores", "scores export",
		"telemetry status", "telemetry preview", "telemetry export", "telemetry send", "telemetry collect", "telemetry triage-foundation",
		"docsync audit", "trust", "trust set", "models evaluate", "models list", "models override",
		"tools list", "tools run <name>", "mcp serve",
	} {
		require.Contains(t, res.Output, command)
	}
}

func TestMarsHarnessCLI_repoShortcutAppendsRepoFlagForSyncedCommands(t *testing.T) {
	for _, args := range [][]string{
		{"release", "backfill-notes", "--dry-run"},
		{"docsync", "audit"},
		{"models", "evaluate", "--json"},
		{"models", "override", "--tier", "coding", "--provider", "ollama", "--model", "qwen"},
		{"scores"},
		{"telemetry", "preview"},
		{"telemetry", "triage-foundation", "--db", "intake.db"},
		{"trust"},
	} {
		t.Run(strings.Join(args[:min(2, len(args))], "_"), func(t *testing.T) {
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			bin := writeFakeMarsHarnessBinary(t, dir)
			t.Setenv("MARS_HARNESS_CLI_BIN", bin)

			raw, err := json.Marshal(marsHarnessCLIArgs{
				Mode:           "run",
				Args:           args,
				Repo:           ".",
				TimeoutSeconds: 5,
			})
			require.NoError(t, err)
			res, err := handleMarsHarnessCLI(context.Background(), root, raw)
			require.NoError(t, err)
			require.Contains(t, res.Output, "--repo "+root.Abs())
		})
	}
}

func TestMarsHarnessCLI_runUsesStructuredArgv(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsHarnessBinary(t, dir)
	t.Setenv("MARS_HARNESS_CLI_BIN", bin)

	res, err := handleMarsHarnessCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["version"],
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Equal(t, "fake-mars version", strings.TrimSpace(res.Output))
}

func TestMarsHarnessCLI_repoShortcutAppendsRepoFlag(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsHarnessBinary(t, dir)
	t.Setenv("MARS_HARNESS_CLI_BIN", bin)

	res, err := handleMarsHarnessCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["scan"],
		"repo": ".",
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "args: scan --repo "+root.Abs())
}

func TestMarsHarnessCLI_repoShortcutAppendsRepoFlagForToolsRun(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	bin := writeFakeMarsHarnessBinary(t, dir)
	t.Setenv("MARS_HARNESS_CLI_BIN", bin)

	res, err := handleMarsHarnessCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["tools", "run", "tool_creation_guard", "--args-json", "{}"],
		"repo": ".",
		"timeout_seconds": 5
	}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "args: tools run tool_creation_guard --args-json {} --repo "+root.Abs())
}

func TestMarsHarnessCLI_repoShortcutRejectsUnsupportedCommand(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	_, err = handleMarsHarnessCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["version"],
		"repo": "."
	}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support --repo")
}

func TestMarsHarnessCLI_rejectsEmptyArg(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	_, err = handleMarsHarnessCLI(context.Background(), root, []byte(`{
		"mode": "run",
		"args": ["doctor", ""]
	}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be non-empty")
}

func TestDefaultRegistry_includesMarsHarnessCLI(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	require.Contains(t, reg.Names(), "mars_harness_cli")
}

func writeFakeMarsHarnessBinary(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake binary is POSIX-only")
	}
	path := filepath.Join(dir, "mars-harness")
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
