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
	outputDir := filepath.Join(dir, "tickets")

	findings := []Finding{
		{Type: "no_ci", Description: "No CI found", Severity: "high"},
		{Type: "missing_tests", Path: "pkg/foo", Description: "No tests", Severity: "medium"},
		{Type: "todo", Path: "main.go:5", Description: "// TODO: fix", Severity: "low"},
	}

	err := GenerateTickets(findings, outputDir)
	require.NoError(t, err)

	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.Equal(t, 2, len(entries), "expected 2 tickets (todo findings are skipped)")

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
	assert.DirExists(t, filepath.Join(dir, ".harness", "guardrails"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "knowledge"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))

	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "backlog"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "in-progress"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "done"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "active"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "completed"))
	assert.DirExists(t, filepath.Join(dir, "docs", "design-docs"))
	assert.FileExists(t, filepath.Join(dir, "docs", "tickets", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "exec-plans", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "index.md"))

	expectedPrompts := []string{
		"ceo", "coo", "cto", "engineer", "qa", "security",
		"dependency-manager", "release-manager", "dogfood",
		"pipeline-fixer", "pr-comment-fixer",
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
		"ceo:", "coo:", "cto-pr-merge:", "cto-weekly:",
		"engineer:", "qa:", "security-pr:", "security-weekly:",
		"dependency-manager:", "release-pr:", "release-weekly:",
		"dogfood:", "pipeline-fixer:", "pr-comment-fixer:",
	} {
		assert.Contains(t, manifestStr, key, "manifest missing role %s", key)
	}

	for _, chain := range []string{
		"then: [cto-weekly]",
		"then: [coo]",
		"then: [engineer]",
		"then: [qa, engineer, dogfood]",
		"then: [qa]",
		"then: [security-pr]",
		"then: [dependency-manager]",
		"idle_then: [ceo, janitor]",
	} {
		assert.Contains(t, manifestStr, chain, "manifest missing chain %s", chain)
	}

	assert.Contains(t, manifestStr, "record_decision", "manifest should include record_decision in tool lists")
	assert.Contains(t, manifestStr, "max_turns: 40", "dogfood role should have max_turns: 40")
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
