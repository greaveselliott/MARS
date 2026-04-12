package learnings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStore_LoadSave(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	harnessDir := filepath.Join(dir, ".harness")
	require.NoError(t, os.MkdirAll(harnessDir, 0o755))

	store := NewStore(dir)

	l, err := store.Load()
	require.NoError(t, err)
	require.NotNil(t, l)
	require.Empty(t, l.Lessons)

	l.Conventions.PackageManager = "yarn"
	l.Excludes = []string{"node_modules", ".git"}
	require.NoError(t, store.Save(l))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, "yarn", loaded.Conventions.PackageManager)
	require.Equal(t, []string{"node_modules", ".git"}, loaded.Excludes)
}

func TestStore_AddLesson_dedup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))

	store := NewStore(dir)

	added, err := store.AddLesson("engineer", "failure_avoidance", "use --legacy-peer-deps")
	require.NoError(t, err)
	require.True(t, added)

	added, err = store.AddLesson("engineer", "failure_avoidance", "use --legacy-peer-deps")
	require.NoError(t, err)
	require.False(t, added, "duplicate should be rejected")

	l, err := store.Load()
	require.NoError(t, err)
	require.Len(t, l.Lessons, 1)
}

func TestDetectConventions_node(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"scripts": {"test": "vitest", "lint": "eslint .", "build": "next build"}
	}`), 0o644))

	conv := DetectConventions(dir)
	require.Equal(t, "yarn", conv.PackageManager)
	require.Equal(t, "typescript", conv.Language)
	require.Equal(t, "next.js", conv.Framework)
	require.Equal(t, "yarn run test", conv.TestCommand)
	require.Equal(t, "yarn run lint", conv.LintCommand)
	require.Equal(t, "yarn run build", conv.BuildCommand)
}

func TestDetectConventions_go(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), []byte(""), 0o644))

	conv := DetectConventions(dir)
	require.Equal(t, "go", conv.PackageManager)
	require.Equal(t, "go", conv.Language)
	require.Equal(t, "go test ./...", conv.TestCommand)
	require.Equal(t, "go build ./...", conv.BuildCommand)
}

func TestExtractFailureLesson(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		errMsg string
		want   string
	}{
		{
			name:   "npm eresolve",
			errMsg: "npm ERR! ERESOLVE unable to resolve dependency tree",
			want:   "npm install fails with ERESOLVE — use --legacy-peer-deps",
		},
		{
			name:   "context overflow",
			errMsg: "request exceeds the available context size",
			want:   "Context window overflow — use targeted file reads instead of broad find/grep commands",
		},
		{
			name:   "dev server timeout",
			errMsg: "shell_exec: next dev timed out after 30s",
			want:   "Long-running dev server timed out — use shell_exec with background:true",
		},
		{
			name:   "unknown error",
			errMsg: "something unexpected happened",
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractFailureLesson("engineer", tc.errMsg, "")
			require.Equal(t, tc.want, got)
		})
	}
}

func TestExtractSuccessLesson(t *testing.T) {
	t.Parallel()
	lesson := ExtractSuccessLesson("engineer", []string{
		"npm install --legacy-peer-deps",
		"npm run build",
	})
	require.Equal(t, "This repo requires --legacy-peer-deps for npm install", lesson)

	lesson = ExtractSuccessLesson("engineer", []string{"yarn build"})
	require.Empty(t, lesson)
}

func TestDetectExcludes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	excludes := DetectExcludes(dir)
	require.Contains(t, excludes, "node_modules")
	require.Contains(t, excludes, ".git")
}
