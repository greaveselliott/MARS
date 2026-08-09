/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dogfood-and-decisions.md
- docs/features/F-012-self-improvement-loop.md
*/
package learnings

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDetectConventions_doesNotFollowSymlinkInput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "package.json")
	require.NoError(t, os.WriteFile(outside, []byte(`{"scripts":{"test":"outside"}}`), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "package.json")))

	conv := DetectConventions(root)
	require.Empty(t, conv.Language)
	require.Empty(t, conv.TestCommand)
	require.FileExists(t, outside)
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

func TestStore_AddDecision_dedup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))

	store := NewStore(dir)

	added, err := store.AddDecision("engineer", "Use yarn over npm", "project has yarn.lock")
	require.NoError(t, err)
	require.True(t, added)

	added, err = store.AddDecision("cto", "Use yarn over npm", "different rationale")
	require.NoError(t, err)
	require.False(t, added, "duplicate summary should be rejected")

	l, err := store.Load()
	require.NoError(t, err)
	require.Len(t, l.Decisions, 1)
	require.Equal(t, "engineer", l.Decisions[0].Role)
	require.Equal(t, "Use yarn over npm", l.Decisions[0].Summary)
	require.Equal(t, "project has yarn.lock", l.Decisions[0].Rationale)
}

func TestFormatForContext_withDecisions(t *testing.T) {
	t.Parallel()
	l := &Learnings{
		Conventions: Conventions{
			PackageManager: "yarn",
			Language:       "typescript",
		},
		Decisions: []Decision{
			{Role: "engineer", Summary: "Use yarn over npm", Rationale: "project has yarn.lock"},
			{Role: "dogfood", Summary: "App requires Node 22"},
		},
	}

	output := l.FormatForContext()
	require.Contains(t, output, "### Decisions from past runs")
	require.Contains(t, output, "(engineer) Use yarn over npm — project has yarn.lock")
	require.Contains(t, output, "(dogfood) App requires Node 22")
	require.NotContains(t, output, "App requires Node 22 —")
}

func TestFormatForContext_withStartCommandAndDevPort(t *testing.T) {
	t.Parallel()
	l := &Learnings{
		Conventions: Conventions{
			PackageManager: "yarn",
			StartCommand:   "yarn run dev",
			DevPort:        "3000",
		},
	}

	output := l.FormatForContext()
	require.Contains(t, output, "Start command: yarn run dev")
	require.Contains(t, output, "Dev port: 3000")
}

func TestDetectConventions_nodeStartAndPort(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"scripts": {"dev": "next dev", "build": "next build", "test": "vitest"}
	}`), 0o644))

	conv := DetectConventions(dir)
	require.Equal(t, "yarn run dev", conv.StartCommand)
	require.Equal(t, "3000", conv.DevPort)
}

func TestDetectConventions_goStartAndPort(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), []byte(""), 0o644))

	conv := DetectConventions(dir)
	require.Equal(t, "go run .", conv.StartCommand)
	require.Equal(t, "8080", conv.DevPort)
}

func TestDetectConventions_staticHTML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644))

	conv := DetectConventions(dir)
	require.Equal(t, "python -m http.server 8080", conv.StartCommand)
	require.Equal(t, "8080", conv.DevPort)
}

func TestContainerfileTemplate_node(t *testing.T) {
	t.Parallel()
	conv := Conventions{
		Language:       "typescript",
		Framework:      "next.js",
		PackageManager: "yarn",
		BuildCommand:   "yarn run build",
		StartCommand:   "yarn run start",
		DevPort:        "3000",
	}

	cf := ContainerfileTemplate(conv)
	require.Contains(t, cf, "FROM node:22-alpine")
	require.Contains(t, cf, "yarn install --frozen-lockfile")
	require.Contains(t, cf, "EXPOSE 3000")
	require.Contains(t, cf, "USER node")
}

func TestContainerfileTemplate_go(t *testing.T) {
	t.Parallel()
	conv := Conventions{
		Language: "go",
		DevPort:  "8080",
	}

	cf := ContainerfileTemplate(conv)
	require.Contains(t, cf, "FROM golang:1.24-alpine")
	require.Contains(t, cf, "CGO_ENABLED=0")
	require.Contains(t, cf, "EXPOSE 8080")
	require.Contains(t, cf, "USER nobody")
}

func TestContainerfileTemplate_python(t *testing.T) {
	t.Parallel()
	conv := Conventions{
		Language:     "python",
		StartCommand: "python manage.py runserver",
		DevPort:      "8000",
	}

	cf := ContainerfileTemplate(conv)
	require.Contains(t, cf, "FROM python:3.12-slim")
	require.Contains(t, cf, "pip install")
	require.Contains(t, cf, "EXPOSE 8000")
}

func TestContainerfileTemplate_static(t *testing.T) {
	t.Parallel()
	conv := Conventions{}

	cf := ContainerfileTemplate(conv)
	require.Contains(t, cf, "FROM nginx:alpine")
	require.Contains(t, cf, "EXPOSE 80")
}

func TestContainerfileTemplate_fallback(t *testing.T) {
	t.Parallel()
	conv := Conventions{
		StartCommand: "./run.sh",
		DevPort:      "9000",
	}

	cf := ContainerfileTemplate(conv)
	require.Contains(t, cf, "FROM alpine:3.20")
	require.Contains(t, cf, "EXPOSE 9000")
	require.Contains(t, cf, "./run.sh")
}

func TestDecisionsSurviveRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))

	store := NewStore(dir)

	_, err := store.AddDecision("cto", "Adopt gRPC for service communication", "Lower latency than REST for internal services")
	require.NoError(t, err)
	_, err = store.AddDecision("engineer", "Pin Node to v22", "crypto.subtle requires v22+")
	require.NoError(t, err)

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded.Decisions, 2)
	require.Equal(t, "decision-001", loaded.Decisions[0].ID)
	require.Equal(t, "decision-002", loaded.Decisions[1].ID)

	ctx := loaded.FormatForContext()
	require.True(t, strings.Contains(ctx, "Adopt gRPC"))
	require.True(t, strings.Contains(ctx, "Pin Node to v22"))
}
