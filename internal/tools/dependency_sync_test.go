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
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDependencySyncRepairsMissingNodeModulesIgnoreBeforeInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell executable uses POSIX sh")
	}
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo"}`), 0o644))
	runTestGit(t, dir, "add", "package.json")
	runTestGit(t, dir, "commit", "-m", "node project")
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "npm"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > npm-args.txt\n"), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	report, err := RunDependencySync(context.Background(), root, dependencySyncArgs{Action: "install", PackageManager: "npm"})
	require.NoError(t, err)
	require.NotNil(t, report.Repair)
	require.True(t, report.Repair.Committed)
	require.Contains(t, report.Repair.MissingIgnores, "node_modules")
	require.Contains(t, string(mustReadFile(t, filepath.Join(dir, ".gitignore"))), "node_modules/")
	require.False(t, report.Hygiene.Blocking)
}

func TestDependencySyncUsesFrozenNpmInstallWhenLockfileExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell executable uses POSIX sh")
	}
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"lockfileVersion":3}`+"\n"), 0o644))
	runTestGit(t, dir, "add", ".gitignore", "package.json", "package-lock.json")
	runTestGit(t, dir, "commit", "-m", "node project")
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "npm"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > npm-args.txt\n"), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	report, err := RunDependencySync(context.Background(), root, dependencySyncArgs{Action: "install", PackageManager: "npm"})
	require.NoError(t, err)
	require.Equal(t, []string{"npm", "ci"}, report.Command)
	data, err := os.ReadFile(filepath.Join(dir, "npm-args.txt"))
	require.NoError(t, err)
	require.Equal(t, "ci\n", string(data))
}

func TestDependencySyncRequiresReasonWhenFrozenFalse(t *testing.T) {
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo"}`), 0o644))
	frozen := false

	_, err := RunDependencySync(context.Background(), root, dependencySyncArgs{Action: "install", PackageManager: "npm", Frozen: &frozen})
	require.Error(t, err)
	require.Contains(t, err.Error(), "reason is required")
}

func TestDependencySyncBlocksPostInstallGeneratedPollution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell executable uses POSIX sh")
	}
	dir, root := setupWorkspaceHygieneRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo"}`), 0o644))
	runTestGit(t, dir, "add", ".gitignore", "package.json")
	runTestGit(t, dir, "commit", "-m", "node project")
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "npm"), []byte("#!/bin/sh\nmkdir -p dist\nprintf 'x\\n' > dist/bundle.js\n"), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	frozen := false

	report, err := RunDependencySync(context.Background(), root, dependencySyncArgs{
		Action:         "install",
		PackageManager: "npm",
		Frozen:         &frozen,
		Reason:         "test lockfile creation path",
	})
	require.Error(t, err)
	require.True(t, report.Hygiene.Blocking)
	require.Contains(t, strings.ToLower(report.Message), "dist")
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
