package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan_emptyRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	hasType := func(typ string) bool {
		for _, f := range result.Findings {
			if f.Type == typ {
				return true
			}
		}
		return false
	}
	assert.True(t, hasType("no_ci"), "expected no_ci finding")
	assert.True(t, hasType("no_readme"), "expected no_readme finding")
	assert.True(t, hasType("no_license"), "expected no_license finding")
	assert.True(t, hasType("no_gitignore"), "expected no_gitignore finding")
	assert.False(t, result.HasCI)
	assert.False(t, result.HasReadme)
	assert.False(t, result.HasLicense)
}

func TestScan_detectsGoLanguage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)
	assert.Equal(t, "Go", result.Language)
}

func TestScan_detectsCI(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	wfDir := filepath.Join(dir, ".github", "workflows")
	os.MkdirAll(wfDir, 0o755)
	os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)
	assert.True(t, result.HasCI)
}

func TestScan_detectsMissingTests(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	pkgDir := filepath.Join(dir, "pkg", "foo")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte("package foo\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "missing_tests" && f.Path == filepath.Join("pkg", "foo") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_tests finding for pkg/foo")
}

func TestScan_noMissingTestsWhenTestExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	pkgDir := filepath.Join(dir, "pkg", "bar")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "bar.go"), []byte("package bar\n"), 0o644)
	os.WriteFile(filepath.Join(pkgDir, "bar_test.go"), []byte("package bar\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "missing_tests" && f.Path == filepath.Join("pkg", "bar") {
			t.Fatal("should not report missing_tests for pkg with test files")
		}
	}
}

func TestScan_detectsTodos(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// TODO: fix this\n// FIXME: broken\n// HACK: workaround\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	todoCount := 0
	for _, f := range result.Findings {
		if f.Type == "todo" {
			todoCount++
		}
	}
	assert.Equal(t, 3, todoCount, "expected 3 todo findings (TODO, FIXME, HACK)")
}

func TestScan_detectsLargeFunction(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	var src string
	src += "package main\n\n"
	src += "func bigFunc() {\n"
	for i := 0; i < 55; i++ {
		src += "\t_ = 0\n"
	}
	src += "}\n"
	os.WriteFile(filepath.Join(dir, "big.go"), []byte(src), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "large_function" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected large_function finding")
}

func TestScan_skipsDefaultDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	nmDir := filepath.Join(dir, "node_modules", "pkg")
	os.MkdirAll(nmDir, 0o755)
	os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("// TODO: never see this\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "todo" {
			t.Fatal("should not find TODOs in node_modules")
		}
	}
}

func TestScan_detectsLicense(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("MIT\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)
	assert.True(t, result.HasLicense)
}

func TestScan_invalidRoot(t *testing.T) {
	t.Parallel()
	_, err := Scan(context.Background(), Config{RepoRoot: "/nonexistent/path"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access")
}

func TestScan_emptyRoot(t *testing.T) {
	t.Parallel()
	_, err := Scan(context.Background(), Config{RepoRoot: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestScan_contextCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Scan(ctx, Config{RepoRoot: dir})
	require.Error(t, err)
}

func TestGenerateTickets(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "backlog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "in-progress"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))

	findings := []Finding{
		{Type: "no_ci", Description: "No CI found", Severity: "high"},
		{Type: "missing_tests", Path: "pkg/foo", Description: "No tests", Severity: "medium"},
		{Type: "todo", Path: "main.go:5", Description: "// TODO: fix", Severity: "low"},
	}

	err := GenerateTickets(findings, dir)
	require.NoError(t, err)

	outputDir := filepath.Join(dir, "docs", "tickets", "backlog")
	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.Equal(t, 3, len(entries), "expected TODO findings to become backlog tickets")

	data, err := os.ReadFile(filepath.Join(outputDir, entries[0].Name()))
	require.NoError(t, err)
	assert.Contains(t, string(data), "priority:")
	assert.Contains(t, string(data), "source: scanner")
}

func TestInit_success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	err := Init(dir, false)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(dir, ".harness"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "roles"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "skills"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "guardrails"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "knowledge"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))

	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "backlog"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "in-progress"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "done"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "active"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "completed"))
	assert.DirExists(t, filepath.Join(dir, "docs", "design-docs"))
	assert.DirExists(t, filepath.Join(dir, "docs", "references"))
	assert.FileExists(t, filepath.Join(dir, "AGENTS.md"))
	assert.FileExists(t, filepath.Join(dir, "VERSION"))
	assert.FileExists(t, filepath.Join(dir, "CHANGELOG.md"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "knowledge", "context-glossary.yaml"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "skills", "self-improvement", "SKILL.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "tickets", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "exec-plans", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "index.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "context-glossary.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "release-versioning.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "skill-evolution.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "references", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "references", "harness-engineering-agent-first.md"))

	expectedPrompts := []string{
		"ceo", "coo", "cto", "engineer", "qa", "security",
		"dependency-manager", "release-manager", "dogfood",
		"pipeline-fixer", "janitor",
	}
	for _, role := range expectedPrompts {
		assert.FileExists(t, filepath.Join(dir, ".harness", "roles", role+".md"),
			"expected default prompt for role %s", role)
	}

	manifest, err := os.ReadFile(filepath.Join(dir, ".harness", "manifest.yaml"))
	require.NoError(t, err)
	manifestStr := string(manifest)
	assert.Contains(t, manifestStr, filepath.Base(dir))
	for _, key := range []string{
		"ceo:", "coo:", "cto-weekly:",
		"engineer:", "qa:", "security:",
		"dependency-manager:", "release-manager:",
		"dogfood:", "pipeline-fixer:", "janitor:",
	} {
		assert.Contains(t, manifestStr, key, "manifest missing role %s", key)
	}

	for _, chain := range []string{
		"then: [cto-weekly]",
		"then: [coo]",
		"then: [engineer]",
		"then: [qa, engineer, dogfood]",
		"then: [qa]",
		"then: [security]",
		"then: [dependency-manager]",
		"idle_then: [ceo, janitor]",
	} {
		assert.Contains(t, manifestStr, chain, "manifest missing chain %s", chain)
	}

	assert.Contains(t, manifestStr, "record_decision", "manifest should include record_decision in tool lists")
	assert.Contains(t, manifestStr, "max_turns: 40", "dogfood role should have max_turns: 40")
	assert.Contains(t, manifestStr, "knowledge/context-glossary.yaml", "manifest should include default glossary knowledge route")

	glossary, err := os.ReadFile(filepath.Join(dir, ".harness", "knowledge", "context-glossary.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(glossary), "docs/design-docs/context-glossary.md")
	assert.Contains(t, string(glossary), "docs/design-docs/release-versioning.md")
	assert.Contains(t, string(glossary), "docs/design-docs/skill-evolution.md")

	skill, err := os.ReadFile(filepath.Join(dir, ".harness", "skills", "self-improvement", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(skill), "Create or update a skill when the fix is reusable procedure")

	version, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	require.NoError(t, err)
	assert.Equal(t, "0.1.0\n", string(version))

	changelog, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	require.NoError(t, err)
	assert.Contains(t, string(changelog), "mars-harness release notes")

	agentGuide, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(agentGuide), "After every non-release semantic commit")
	assert.Contains(t, string(agentGuide), "release: notes X.Y.Z")
	assert.Contains(t, string(agentGuide), "Operating rules inherited from Mars Harness apply here")

	releaseDoc, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "release-versioning.md"))
	require.NoError(t, err)
	assert.Contains(t, string(releaseDoc), "Every non-release semantic commit")
	assert.Contains(t, string(releaseDoc), "release: notes X.Y.Z")

	releasePrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "release-manager.md"))
	require.NoError(t, err)
	assert.Contains(t, string(releasePrompt), "Treat every non-release semantic commit")
	assert.Contains(t, string(releasePrompt), "Do not generate another version")
}

func TestInit_alreadyExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".harness"), 0o755)

	err := Init(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInit_forceOverwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".harness"), 0o755)

	err := Init(dir, true)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))
}

func TestInit_forcePreservesExistingContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	require.NoError(t, Init(dir, false))

	ticketDir := filepath.Join(dir, "docs", "tickets", "in-progress")
	ticketPath := filepath.Join(ticketDir, "T-001-user-work.md")
	os.MkdirAll(ticketDir, 0o755)
	require.NoError(t, os.WriteFile(ticketPath, []byte("# T-001: User created ticket\nThis is real work."), 0o644))

	customPrompt := filepath.Join(dir, ".harness", "roles", "engineer.md")
	require.NoError(t, os.WriteFile(customPrompt, []byte("# Custom Engineer Prompt"), 0o644))

	readmePath := filepath.Join(dir, "docs", "tickets", "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("# Custom README"), 0o644))

	agentsPath := filepath.Join(dir, "AGENTS.md")
	require.NoError(t, os.WriteFile(agentsPath, []byte("# Custom Agent Guide"), 0o644))

	glossaryRoute := filepath.Join(dir, ".harness", "knowledge", "context-glossary.yaml")
	require.NoError(t, os.WriteFile(glossaryRoute, []byte("routes: []\n# custom"), 0o644))

	require.NoError(t, Init(dir, true))

	ticketContent, err := os.ReadFile(ticketPath)
	require.NoError(t, err)
	assert.Contains(t, string(ticketContent), "User created ticket", "ticket content must be preserved on --force")

	promptContent, err := os.ReadFile(customPrompt)
	require.NoError(t, err)
	assert.Equal(t, "# Custom Engineer Prompt", string(promptContent), "custom role prompt must be preserved on --force")

	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	assert.Equal(t, "# Custom README", string(readmeContent), "custom docs must be preserved on --force")

	agentsContent, err := os.ReadFile(agentsPath)
	require.NoError(t, err)
	assert.Equal(t, "# Custom Agent Guide", string(agentsContent), "custom AGENTS.md must be preserved on --force")

	glossaryContent, err := os.ReadFile(glossaryRoute)
	require.NoError(t, err)
	assert.Equal(t, "routes: []\n# custom", string(glossaryContent), "custom harness knowledge routes must be preserved on --force")

	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"), "manifest must still be created")
}

func TestInit_autoGitInit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := Init(dir, false)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(dir, ".git"), "git should be auto-initialised")
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))
}

func TestInit_emptyRoot(t *testing.T) {
	t.Parallel()
	err := Init("", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestEnsureHarness_noManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	did, err := EnsureHarness(dir, false)
	require.NoError(t, err)
	assert.True(t, did)
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))

	did2, err := EnsureHarness(dir, false)
	require.NoError(t, err)
	assert.False(t, did2)
}

func TestEnsureHarness_invalidManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".harness"), 0o755)
	err := os.WriteFile(filepath.Join(dir, ".harness", "manifest.yaml"),
		[]byte("name: bad\n"), 0o644)
	require.NoError(t, err)

	_, err = EnsureHarness(dir, false)
	require.Error(t, err)
}

func TestDetectFramework_goMod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)

	fw := detectFramework(dir, []string{"go.mod"})
	assert.Equal(t, "Go Module", fw)
}

func TestDetectFramework_cargoToml(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644)

	fw := detectFramework(dir, []string{"Cargo.toml"})
	assert.Equal(t, "Rust/Cargo", fw)
}

// --- Bootability check tests ---

func TestBootability_missingDevScript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"test": "jest"}
	}`), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "missing_dev_script" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_dev_script finding when package.json has no dev/start script")
}

func TestBootability_hasDevScript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build", "test": "jest"}
	}`), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "missing_dev_script" {
			t.Fatal("should not report missing_dev_script when dev script exists")
		}
	}
}

func TestBootability_missingRootLayout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)
	appDir := filepath.Join(dir, "src", "app", "(dashboard)")
	os.MkdirAll(appDir, 0o755)
	os.WriteFile(filepath.Join(appDir, "page.tsx"), []byte("export default function Page() { return <div/>; }"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "missing_root_layout" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_root_layout when src/app/layout.tsx doesn't exist")
}

func TestBootability_hasRootLayout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)
	appDir := filepath.Join(dir, "src", "app")
	os.MkdirAll(appDir, 0o755)
	os.WriteFile(filepath.Join(appDir, "layout.tsx"), []byte("export default function Layout({children}) { return <html><body>{children}</body></html>; }"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "missing_root_layout" {
			t.Fatal("should not report missing_root_layout when src/app/layout.tsx exists")
		}
	}
}

func TestBootability_conflictingAppAndPages(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)

	// app/ at root, pages/ under src/ — conflict
	rootApp := filepath.Join(dir, "app")
	os.MkdirAll(rootApp, 0o755)
	os.WriteFile(filepath.Join(rootApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644)
	os.WriteFile(filepath.Join(rootApp, "page.tsx"), []byte("export default function P() { return <div/>; }"), 0o644)

	srcPages := filepath.Join(dir, "src", "pages", "auth")
	os.MkdirAll(srcPages, 0o755)
	os.WriteFile(filepath.Join(srcPages, "login.tsx"), []byte("export default function Login() { return <div/>; }"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "conflicting_app_pages" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected conflicting_app_pages when app/ is at root and pages/ is under src/")
}

func TestBootability_noConflictWhenSameRoot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)

	srcApp := filepath.Join(dir, "src", "app")
	os.MkdirAll(srcApp, 0o755)
	os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644)

	srcPages := filepath.Join(dir, "src", "pages", "api")
	os.MkdirAll(srcPages, 0o755)
	os.WriteFile(filepath.Join(srcPages, "health.ts"), []byte("export default function handler() {}"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "conflicting_app_pages" {
			t.Fatal("should not report conflict when app/ and pages/ are both under src/")
		}
	}
}

func TestBootability_missingTailwindConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)

	srcApp := filepath.Join(dir, "src", "app")
	os.MkdirAll(srcApp, 0o755)
	os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644)

	stylesDir := filepath.Join(dir, "src", "styles")
	os.MkdirAll(stylesDir, 0o755)
	os.WriteFile(filepath.Join(stylesDir, "globals.css"), []byte("@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	foundTailwind := false
	foundPostCSS := false
	for _, f := range result.Findings {
		if f.Type == "missing_tailwind_config" && f.Path == "" {
			foundTailwind = true
		}
		if f.Type == "missing_tailwind_config" && f.Path == "postcss.config.js" {
			foundPostCSS = true
		}
	}
	assert.True(t, foundTailwind, "expected missing_tailwind_config when CSS uses @tailwind directives")
	assert.True(t, foundPostCSS, "expected missing postcss.config finding when CSS uses @tailwind directives")
}

func TestBootability_tailwindConfigPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	stylesDir := filepath.Join(dir, "src", "styles")
	os.MkdirAll(stylesDir, 0o755)
	os.WriteFile(filepath.Join(stylesDir, "globals.css"), []byte("@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "tailwind.config.js"), []byte("module.exports = {};\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "postcss.config.js"), []byte("module.exports = {};\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "missing_tailwind_config" {
			t.Fatal("should not report missing_tailwind_config when config files exist")
		}
	}
}

func TestBootability_deprecatedNextConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(`const nextConfig = { experimental: { appDir: true } }; module.exports = nextConfig;`), 0o644)

	srcApp := filepath.Join(dir, "src", "app")
	os.MkdirAll(srcApp, 0o755)
	os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "deprecated_next_config" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected deprecated_next_config when next.config.js has appDir")
}

func TestBootability_misconfiguredPathAlias(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
		"compilerOptions": {
			"baseUrl": ".",
			"paths": { "@/*": ["./*"] }
		}
	}`), 0o644)

	srcApp := filepath.Join(dir, "src", "app")
	os.MkdirAll(srcApp, 0o755)
	os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644)
	os.WriteFile(filepath.Join(srcApp, "page.tsx"), []byte("export default function P() { return <div/>; }"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "misconfigured_path_alias" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected misconfigured_path_alias when @/* maps to ./* but source is in src/")
}

func TestBootability_correctPathAlias(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644)
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
		"compilerOptions": {
			"baseUrl": ".",
			"paths": { "@/*": ["./src/*"] }
		}
	}`), 0o644)

	srcApp := filepath.Join(dir, "src", "app")
	os.MkdirAll(srcApp, 0o755)
	os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "misconfigured_path_alias" {
			t.Fatal("should not report misconfigured_path_alias when @/* correctly maps to ./src/*")
		}
	}
}

func TestScan_missingGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	hasType := func(typ string) bool {
		for _, f := range result.Findings {
			if f.Type == typ {
				return true
			}
		}
		return false
	}
	assert.True(t, hasType("no_gitignore"), "expected no_gitignore finding when .gitignore is missing")
}

func TestScan_hasGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n.next/\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "no_gitignore" {
			t.Fatal("should not report no_gitignore when .gitignore exists at root")
		}
	}
}

func TestUpgrade_preservesUserConfiguredManifestAndPrompts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)

	require.NoError(t, Init(dir, false))

	manifestPath := filepath.Join(dir, ".harness", "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("name: custom\nroles:\n  custom:\n    prompt: roles/custom.md\n"), 0o644))

	customPrompt := filepath.Join(dir, ".harness", "roles", "coo.md")
	require.NoError(t, os.WriteFile(customPrompt, []byte("# Custom COO Prompt"), 0o644))

	customKnowledge := filepath.Join(dir, ".harness", "knowledge", "context-glossary.yaml")
	require.NoError(t, os.WriteFile(customKnowledge, []byte("routes: []\n# custom knowledge"), 0o644))

	missingPrompt := filepath.Join(dir, ".harness", "roles", "qa.md")
	require.NoError(t, os.Remove(missingPrompt))

	updated, err := Upgrade(dir)
	require.NoError(t, err)
	assert.NotContains(t, updated, "manifest.yaml")
	assert.NotContains(t, updated, "roles/coo.md")
	assert.NotContains(t, updated, "knowledge/context-glossary.yaml")
	assert.Contains(t, updated, "roles/qa.md")

	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(manifest), "custom")

	content, err := os.ReadFile(customPrompt)
	require.NoError(t, err)
	assert.Equal(t, "# Custom COO Prompt", string(content))

	knowledge, err := os.ReadFile(customKnowledge)
	require.NoError(t, err)
	assert.Equal(t, "routes: []\n# custom knowledge", string(knowledge))

	createdPrompt, err := os.ReadFile(missingPrompt)
	require.NoError(t, err)
	assert.Contains(t, string(createdPrompt), "Quality Reviewer")
}

func TestUpgrade_failsWithoutHarness(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	_, err := Upgrade(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}
