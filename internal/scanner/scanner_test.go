/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/harness-glossary.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/release-versioning.md
- docs/design-docs/tools-glossary.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
- docs/roles/ROLES.md
*/
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan_emptyRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644))

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)
	assert.Equal(t, "Go", result.Language)
}

func TestScan_detectsCI(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	wfDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(wfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI\n"), 0o644))

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)
	assert.True(t, result.HasCI)
}

func TestScan_detectsMissingTests(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	pkgDir := filepath.Join(dir, "pkg", "foo")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte("package foo\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	pkgDir := filepath.Join(dir, "pkg", "bar")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "bar.go"), []byte("package bar\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "bar_test.go"), []byte("package bar\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// TODO: fix this\n// FIXME: broken\n// HACK: workaround\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	var src string
	src += "package main\n\n"
	src += "func bigFunc() {\n"
	for i := 0; i < 55; i++ {
		src += "\t_ = 0\n"
	}
	src += "}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.go"), []byte(src), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	nmDir := filepath.Join(dir, "node_modules", "pkg")
	require.NoError(t, os.MkdirAll(nmDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("// TODO: never see this\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("MIT\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

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

func TestFindStaleInProgressTicketsSkipsBlockedTickets(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "backlog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "in-progress"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "tickets", "in-progress", "T-001-stale.md"), []byte(`---
id: T-001
title: Stale
last_attempt: "2026-04-01"
blocker: none
blocked_by: []
---

# Stale
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "tickets", "in-progress", "T-002-blocked.md"), []byte(`---
id: T-002
title: Blocked
last_attempt: "2026-04-01"
blocker: "waiting for dependency"
blocked_by: ["T-003"]
---

# Blocked
`), 0o644))

	findings := findStaleInProgressTickets(dir, time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC))
	require.Len(t, findings, 1)
	assert.Equal(t, "stale_in_progress_ticket", findings[0].Type)
	assert.Contains(t, findings[0].Path, "T-001-stale.md")
}

func TestGenerateTicketsCreatesInterventionDebtForStaleInProgress(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "backlog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "in-progress"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))

	err := GenerateTickets([]Finding{{
		Type:        "stale_in_progress_ticket",
		Path:        "docs/tickets/in-progress/T-001-stale.md",
		Description: "stale ticket",
		Severity:    "high",
	}}, dir)
	require.NoError(t, err)

	entries, err := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", entries[0].Name()))
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, "kind: intervention-debt")
	assert.Contains(t, text, "work_type: intervention-debt")
	assert.Contains(t, text, "dedupe_key:")
	assert.Contains(t, text, "category: \"stale_in_progress_ticket\"")
}

func TestInit_success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Space Invaders\n\nBuild a small browser arcade game with player movement, alien waves, and projectiles.\n"), 0o644))

	err := Init(dir, false)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(dir, ".harness"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "roles"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "skills"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "guardrails"))
	assert.DirExists(t, filepath.Join(dir, ".harness", "knowledge"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "metadata.yaml"))

	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "backlog"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "in-progress"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "in-review"))
	assert.DirExists(t, filepath.Join(dir, "docs", "tickets", "done"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "backlog"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "active"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "completed"))
	assert.DirExists(t, filepath.Join(dir, "docs", "exec-plans", "superseded"))
	assert.DirExists(t, filepath.Join(dir, "docs", "design-docs"))
	assert.DirExists(t, filepath.Join(dir, "docs", "goals"))
	assert.DirExists(t, filepath.Join(dir, "docs", "features"))
	assert.DirExists(t, filepath.Join(dir, "docs", "roles"))
	assert.DirExists(t, filepath.Join(dir, "docs", "references"))
	assert.DirExists(t, filepath.Join(dir, "docs", "reports", "qa"))
	assert.DirExists(t, filepath.Join(dir, "docs", "reports", "security"))
	assert.DirExists(t, filepath.Join(dir, "docs", "reports", "dependencies"))
	assert.DirExists(t, filepath.Join(dir, "docs", "reports", "dogfood"))
	assert.DirExists(t, filepath.Join(dir, "docs", "reports", "strategy"))
	assert.FileExists(t, filepath.Join(dir, "AGENTS.md"))
	assert.FileExists(t, filepath.Join(dir, "VERSION"))
	assert.FileExists(t, filepath.Join(dir, "CHANGELOG.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "QUALITY_SCORE.md"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "knowledge", "context-glossary.yaml"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "skills", "self-improvement", "SKILL.md"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "skills", "github-private-release-auth", "SKILL.md"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "skills", "cli-tool-sync", "SKILL.md"))
	assert.FileExists(t, filepath.Join(dir, ".harness", "skills", "persona-design", "SKILL.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "tickets", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "exec-plans", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "exec-plans", "active", "current-operating-plan.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "index.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "delivery-operating-model.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "harness-operating-model.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "conversation-as-system-record.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "context-glossary.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "release-versioning.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "design-docs", "skill-evolution.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "goals", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "goals", "active.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "goals", "observations.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "goals", "superseded.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "features", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "roles", "ROLES.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "roles", "personas", "ceo.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "roles", "personas", "coo.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "roles", "personas", "cto-weekly.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "references", "README.md"))
	assert.FileExists(t, filepath.Join(dir, "docs", "references", "harness-engineering-agent-first.md"))

	expectedPrompts := []string{
		"ceo", "head-of-strategy", "coo", "cto", "engineer", "qa", "security",
		"dependency-manager", "release-manager", "dogfood",
		"pipeline-fixer", "orchestrator", "janitor",
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
		"ceo:", "head-of-strategy:", "coo:", "cto-weekly:",
		"engineer:", "qa:", "security:",
		"dependency-manager:", "release-manager:",
		"dogfood:", "pipeline-fixer:", "orchestrator:", "janitor:",
	} {
		assert.Contains(t, manifestStr, key, "manifest missing role %s", key)
	}

	assert.Contains(t, manifestStr, "orchestration_mode: dispatch", "generated manifest should return terminal dispositions through the orchestrator")
	assert.NotContains(t, manifestStr, "then: [cto-weekly]", "dispatch defaults should not encode fixed role-to-role handoffs")
	assert.NotContains(t, manifestStr, "idle_then:", "dispatch defaults should route idle work through dispositions and the orchestrator")

	assert.Contains(t, manifestStr, "record_decision", "manifest should include record_decision in tool lists")
	assert.Contains(t, manifestStr, "trust_level: contributor", "generated manifest should seed bootstrap mutating roles above observer trust")
	assert.Contains(t, manifestStr, "job_disposition_record", "manifest should expose dispatch disposition recording")
	assert.Contains(t, manifestStr, "domain: planner", "manifest should include canonical domain metadata")
	assert.Contains(t, manifestStr, "mode: ticket-delivery", "manifest should include role mode metadata")
	assert.Contains(t, manifestStr, "qa:\n    prompt: roles/qa.md\n    domain: reviewer\n    mode: quality-review\n    model: reasoning", "QA should use a tool-reliable model tier")
	assert.Contains(t, manifestStr, "tool_inventory_audit, git_status, git_diff]", "QA should have read-only git inspection tools")
	assert.Contains(t, manifestStr, "mars_harness_cli", "manifest should expose mars_harness_cli as a mirrored tool")
	assert.Contains(t, manifestStr, "github_auth_check", "manifest should expose private release auth check as a mirrored tool")
	assert.Contains(t, manifestStr, "tool_create", "manifest should expose tool_create as a mirrored tool")
	assert.Contains(t, manifestStr, "docsync_audit", "manifest should expose docsync_audit as a mirrored documentation tool")
	assert.Contains(t, manifestStr, "record_decision, tool_create, persona_create, task_trace_summarize, docsync_audit, git_status", "implementation roles should allow tool/persona creation and docsync before git tools")
	assert.Contains(t, manifestStr, "record_decision, ticket_create, tool_create, persona_create, task_trace_summarize, docsync_audit", "dogfood should create findings through ticket_create and audit docs")
	assert.Contains(t, manifestStr, "release_orchestrate", "release role should expose release orchestration")
	assert.Contains(t, manifestStr, "architecture_audit", "review roles should expose architecture audit")
	assert.Contains(t, manifestStr, "tool_creation_guard", "review roles should expose tool creation guard")
	assert.Contains(t, manifestStr, "tool_inventory_audit", "review roles should expose tool inventory audit")
	assert.Contains(t, manifestStr, "max_turns: 40", "dogfood role should have max_turns: 40")
	assert.Contains(t, manifestStr, "knowledge/context-glossary.yaml", "manifest should include default glossary knowledge route")

	strategyRoleBlock := manifestRoleBlock(t, manifestStr, "head-of-strategy")
	assert.Contains(t, strategyRoleBlock, "domain: planner")
	assert.Contains(t, strategyRoleBlock, "mode: strategy-advisory")
	assert.NotContains(t, strategyRoleBlock, "schedule:", "strategy advisor should not enter the default scheduled flow")
	assert.NotContains(t, strategyRoleBlock, "triggers:", "strategy advisor should be dispatch/manual only")
	assert.NotContains(t, strategyRoleBlock, "ticket_create", "strategy advisor should not create tickets")
	assert.NotContains(t, strategyRoleBlock, "tool_create", "strategy advisor should not implement tooling")
	assert.NotContains(t, strategyRoleBlock, "shell_exec", "strategy advisor should not get general implementation shell access")

	cooRoleBlock := manifestRoleBlock(t, manifestStr, "coo")
	assert.Contains(t, cooRoleBlock, "mode: execution-planning")
	assert.NotContains(t, cooRoleBlock, "ticket_create", "COO owns plans and BDD contracts, not ticket creation")
	assert.NotContains(t, cooRoleBlock, "shell_exec", "COO should not get shell access that can bypass planning-only ownership")

	ctoRoleBlock := manifestRoleBlock(t, manifestStr, "cto-weekly")
	assert.Contains(t, ctoRoleBlock, "mode: technical-planning")
	assert.Contains(t, ctoRoleBlock, "ticket_create", "CTO owns technical ticket creation")
	assert.Contains(t, ctoRoleBlock, "task_trace_summarize", "CTO can inspect immediate handoff traces without broad audit tools")
	assert.NotContains(t, ctoRoleBlock, "architecture_audit", "fresh CTO bootstrap should not start with broad architecture audit")
	assert.NotContains(t, ctoRoleBlock, "docsync_audit", "fresh CTO bootstrap should not start with broad docsync audit")
	assert.NotContains(t, ctoRoleBlock, "tool_inventory_audit", "fresh CTO bootstrap should not start with broad tool inventory audit")
	assert.NotContains(t, ctoRoleBlock, "shell_exec", "fresh CTO bootstrap should use ticket and doc tools before general shell access")
	assert.NotContains(t, ctoRoleBlock, "mars_harness_cli", "fresh CTO bootstrap should not run source-harness CLI workflows before product tickets exist")

	strategyPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "head-of-strategy.md"))
	require.NoError(t, err)
	strategyPromptStr := string(strategyPrompt)
	assert.Contains(t, strategyPromptStr, "## Personal Guide")
	assert.Contains(t, strategyPromptStr, "### Modus Operandi")
	assert.Contains(t, strategyPromptStr, "### Owns")
	assert.Contains(t, strategyPromptStr, "### Does Not Own")
	assert.Contains(t, strategyPromptStr, "### Best Feedback Format")
	assert.Contains(t, strategyPromptStr, "### Stop Conditions")

	cooPersona, err := os.ReadFile(filepath.Join(dir, "docs", "roles", "personas", "coo.md"))
	require.NoError(t, err)
	assert.Contains(t, string(cooPersona), "## Owns")
	assert.Contains(t, string(cooPersona), "BDD feature contracts and scenario schedule")
	assert.Contains(t, string(cooPersona), "## Feedback I Need")
	assert.Contains(t, string(cooPersona), "## Stop Conditions")

	ctoPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "cto.md"))
	require.NoError(t, err)
	assert.Contains(t, string(ctoPrompt), "DISPATCH-BOOTSTRAP FAST PATH")
	assert.Contains(t, string(ctoPrompt), "Create at most one ordinary feature ticket")
	assert.Contains(t, string(ctoPrompt), "not decompose the same BDD scenario into several independent backlog tickets")

	dogfoodPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "dogfood.md"))
	require.NoError(t, err)
	assert.Contains(t, string(dogfoodPrompt), "You are observation-first")
	assert.Contains(t, string(dogfoodPrompt), "Do not add scripts yourself")
	assert.Contains(t, string(dogfoodPrompt), "docs/reports/dogfood/")

	metadata, err := ReadHarnessMetadata(dir)
	require.NoError(t, err)
	assert.Equal(t, "mars-harness", metadata.Generator)
	assert.NotEmpty(t, metadata.GeneratorVersion)

	glossary, err := os.ReadFile(filepath.Join(dir, ".harness", "knowledge", "context-glossary.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(glossary), "docs/design-docs/context-glossary.md")
	assert.Contains(t, string(glossary), "docs/design-docs/harness-glossary.md")
	assert.Contains(t, string(glossary), "docs/design-docs/harness-operating-model.md")
	assert.Contains(t, string(glossary), "docs/roles/ROLES.md")
	assert.Contains(t, string(glossary), "docs/design-docs/tools-glossary.md")
	assert.Contains(t, string(glossary), "docs/design-docs/code-documentation-map.md")
	assert.Contains(t, string(glossary), "docs/design-docs/documentation-sync-architecture.md")
	assert.Contains(t, string(glossary), "docs/design-docs/cli-tool-skill-sync.md")
	assert.Contains(t, string(glossary), ".harness/skills/cli-tool-sync/SKILL.md")
	assert.Contains(t, string(glossary), "tool availability")
	assert.Contains(t, string(glossary), "docs/design-docs/release-versioning.md")
	assert.Contains(t, string(glossary), "github-private-release-auth")
	assert.NotContains(t, string(glossary), "docs/features/F-009-release-update-lifecycle.md")
	assert.Contains(t, string(glossary), "docs/design-docs/skill-evolution.md")
	assert.Contains(t, string(glossary), "operating model")
	assert.Contains(t, string(glossary), "goals, BDD, feature contracts, planning, feedback, or quality evidence")

	skill, err := os.ReadFile(filepath.Join(dir, ".harness", "skills", "self-improvement", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(skill), "Create or update a skill when the fix is reusable procedure")

	githubAuthSkill, err := os.ReadFile(filepath.Join(dir, ".harness", "skills", "github-private-release-auth", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(githubAuthSkill), "mars-harness auth github setup")
	assert.Contains(t, string(githubAuthSkill), "github_auth_check")
	assert.Contains(t, string(githubAuthSkill), "Never paste token values")

	cliSkill, err := os.ReadFile(filepath.Join(dir, ".harness", "skills", "cli-tool-sync", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(cliSkill), "CLI Tool Sync Skill")
	assert.Contains(t, string(cliSkill), "mars_harness_cli")
	assert.Contains(t, string(cliSkill), "repo shortcut")

	version, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	require.NoError(t, err)
	assert.Equal(t, "0.1.0\n", string(version))

	changelog, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	require.NoError(t, err)
	assert.Contains(t, string(changelog), "mars-harness release notes")

	releaseDoc, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "release-versioning.md"))
	require.NoError(t, err)
	assert.Contains(t, string(releaseDoc), "mars-harness release backfill-notes")
	assert.Contains(t, string(releaseDoc), "--check")
	assert.Contains(t, string(releaseDoc), "Private Release Auth")
	assert.Contains(t, string(releaseDoc), "mars-harness auth github setup")
	assert.Contains(t, string(releaseDoc), "github_auth_check")

	agentGuide, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(agentGuide), "docs/QUALITY_SCORE.md")
	assert.Contains(t, string(agentGuide), "Harness Glossary")
	assert.Contains(t, string(agentGuide), "docs/design-docs/harness-operating-model.md")
	assert.Contains(t, string(agentGuide), "docs/roles/ROLES.md")
	assert.Contains(t, string(agentGuide), "docs/design-docs/documentation-sync-architecture.md")
	assert.Contains(t, string(agentGuide), "docs/design-docs/cli-tool-skill-sync.md")
	assert.Contains(t, string(agentGuide), "CLI tool/skill sync")
	assert.Contains(t, string(agentGuide), "Foundation harness")
	assert.Contains(t, string(agentGuide), "Deployed harness")
	assert.Contains(t, string(agentGuide), "Foundation operating model")
	assert.Contains(t, string(agentGuide), "Deployed operating model")
	assert.Contains(t, string(agentGuide), "Symbiotic operating-model change")
	assert.Contains(t, string(agentGuide), "Conversation system record")
	assert.Contains(t, string(agentGuide), "Meta tool")
	assert.Contains(t, string(agentGuide), "Formalized tool creation trigger")
	assert.Contains(t, string(agentGuide), "Tool creation path")
	assert.Contains(t, string(agentGuide), "Universal tool surface")
	assert.Contains(t, string(agentGuide), "mars-harness mcp serve")
	assert.Contains(t, string(agentGuide), "mars_harness_cli")
	assert.Contains(t, string(agentGuide), "github_auth_check")
	assert.Contains(t, string(agentGuide), "docsync_audit")
	assert.Contains(t, string(agentGuide), "job_disposition_record")
	assert.Contains(t, string(agentGuide), "tool_create")
	assert.Contains(t, string(agentGuide), "Skills")
	assert.Contains(t, string(agentGuide), "Universal skills")
	assert.Contains(t, string(agentGuide), "Foundation skills")
	assert.Contains(t, string(agentGuide), "Deployed skills")
	assert.Contains(t, string(agentGuide), "Contextual harness definition")
	assert.Contains(t, string(agentGuide), "docs/design-docs/harness-glossary.md")
	assert.Contains(t, string(agentGuide), "docs/design-docs/tools-glossary.md")
	assert.Contains(t, string(agentGuide), "docs/goals/active.md")
	assert.Contains(t, string(agentGuide), "BDD feature contracts")
	assert.Contains(t, string(agentGuide), "Business logic is first-class BDD")
	assert.Contains(t, string(agentGuide), "No stale documentation")
	assert.Contains(t, string(agentGuide), "MarsDocSync")
	assert.Contains(t, string(agentGuide), "docs:")
	assert.Contains(t, string(agentGuide), "walking skeleton")
	assert.Contains(t, string(agentGuide), "exactly one active exec plan")
	assert.Contains(t, string(agentGuide), "git fetch origin main")
	assert.Contains(t, string(agentGuide), "origin/main")
	assert.Contains(t, string(agentGuide), "origin main")
	assert.Contains(t, string(agentGuide), "After every non-release semantic commit")
	assert.Contains(t, string(agentGuide), "release: notes X.Y.Z")
	assert.Contains(t, string(agentGuide), "Significant conversations must update the owning repo artifact")
	assert.Contains(t, string(agentGuide), "Chat summaries cannot replace those artifacts")
	assert.Contains(t, string(agentGuide), "Trivial command responses")
	assert.Contains(t, string(agentGuide), "Operating rules inherited from Mars Harness apply here")
	assert.Contains(t, string(agentGuide), "mars-harness eject --repo .")
	assert.Contains(t, string(agentGuide), "--apply --confirm <repo-name>")
	assert.Contains(t, string(agentGuide), "publish or update GitHub Release")
	assert.Contains(t, string(agentGuide), "notes-only GitHub Release")
	assert.Contains(t, string(agentGuide), "mars-harness auth github check")
	assert.Contains(t, string(agentGuide), "mars-harness auth github setup")
	assert.Contains(t, string(agentGuide), "never paste tokens")
	assert.Contains(t, string(agentGuide), "Product features and user-visible behavior changes must be documented with")
	assert.Contains(t, string(agentGuide), "Do not leave architecture or product intent only in chat")

	harnessGlossary, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "harness-glossary.md"))
	require.NoError(t, err)
	assert.Contains(t, string(harnessGlossary), "First-Class Harness Definitions")
	assert.Contains(t, string(harnessGlossary), "Operating model")
	assert.Contains(t, string(harnessGlossary), "BDD feature contract")
	assert.Contains(t, string(harnessGlossary), "Business logic")
	assert.Contains(t, string(harnessGlossary), "No stale documentation")
	assert.Contains(t, string(harnessGlossary), "MarsDocSync")
	assert.Contains(t, string(harnessGlossary), "Role registry")
	assert.Contains(t, string(harnessGlossary), "Foundation operating model")
	assert.Contains(t, string(harnessGlossary), "Deployed operating model")
	assert.Contains(t, string(harnessGlossary), "Symbiotic operating-model change")
	assert.Contains(t, string(harnessGlossary), "Conversation system record")
	assert.Contains(t, string(harnessGlossary), "When turning chat context into durable repo state include this")
	assert.Contains(t, string(harnessGlossary), "docs/design-docs/conversation-as-system-record.md")
	assert.Contains(t, string(harnessGlossary), "Formalized tool creation trigger")
	assert.Contains(t, string(harnessGlossary), "Tool creation path")
	assert.Contains(t, string(harnessGlossary), "Universal tool surface")
	assert.Contains(t, string(harnessGlossary), "mars_harness_cli")
	assert.Contains(t, string(harnessGlossary), "github_auth_check")
	assert.Contains(t, string(harnessGlossary), "docsync_audit")
	assert.Contains(t, string(harnessGlossary), "job_disposition_record")
	assert.Contains(t, string(harnessGlossary), "Skills")
	assert.Contains(t, string(harnessGlossary), "Universal skills")
	assert.Contains(t, string(harnessGlossary), "Foundation skills")
	assert.Contains(t, string(harnessGlossary), "Deployed skills")
	assert.Contains(t, string(harnessGlossary), "CLI tool/skill sync")
	assert.Contains(t, string(harnessGlossary), "When changing operating doctrine include this")
	assert.Contains(t, string(harnessGlossary), "When choosing, creating, or changing tools include this")
	assert.Contains(t, string(harnessGlossary), "`tool_create` is a mirrored tool")
	assert.Contains(t, string(harnessGlossary), "docs/design-docs/tools-glossary.md")
	assert.Contains(t, string(harnessGlossary), "When doing X include this: <path to document.md>")

	toolsGlossary, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "tools-glossary.md"))
	require.NoError(t, err)
	assert.Contains(t, string(toolsGlossary), "Mirrored Built-In Tools")
	assert.Contains(t, string(toolsGlossary), "mars_harness_cli")
	assert.Contains(t, string(toolsGlossary), "github_auth_check")
	assert.Contains(t, string(toolsGlossary), "cli-tool-skill-sync.md")
	assert.Contains(t, string(toolsGlossary), "release_orchestrate")
	assert.Contains(t, string(toolsGlossary), "job_disposition_record")
	assert.Contains(t, string(toolsGlossary), "harness_doctrine_sync")
	assert.Contains(t, string(toolsGlossary), "docsync_audit")
	assert.Contains(t, string(toolsGlossary), "tool_creation_guard")
	assert.Contains(t, string(toolsGlossary), "task_trace_summarize")
	assert.Contains(t, string(toolsGlossary), "mars-harness mcp serve")
	assert.Contains(t, string(toolsGlossary), "New built-in tools must originate through")
	assert.Contains(t, string(toolsGlossary), "Every newly created tool must extend this glossary")

	roleRegistry, err := os.ReadFile(filepath.Join(dir, "docs", "roles", "ROLES.md"))
	require.NoError(t, err)
	assert.Contains(t, string(roleRegistry), "| `engineer` | default | engineer | `ticket-delivery` |")
	assert.Contains(t, string(roleRegistry), "| `orchestrator` | default | orchestrator | `dispatch-routing` |")
	assert.Contains(t, string(roleRegistry), "Optional GitHub webhook triggers are explicit repair inputs")
	assert.Contains(t, string(roleRegistry), "`Origin` set to `custom`")

	qaPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "qa.md"))
	require.NoError(t, err)
	assert.Contains(t, string(qaPrompt), "Default QA is read-only")
	assert.Contains(t, string(qaPrompt), "START with an allowed read-only tool call")
	assert.Contains(t, string(qaPrompt), "QA does not have shell_exec")
	assert.Contains(t, string(qaPrompt), "record exactly one job_disposition_record")
	assert.Contains(t, string(qaPrompt), "A prose response without job_disposition_record fails the dispatch protocol")
	assert.Contains(t, string(qaPrompt), "Do not block only because implementation source or diffs were absent from the")
	assert.Contains(t, string(qaPrompt), "inspect the relevant ticket")
	assert.Contains(t, string(qaPrompt), "Tickets live only under docs/tickets/backlog/")
	assert.Contains(t, string(qaPrompt), "Do not assume a ticket lives")
	assert.Contains(t, string(qaPrompt), "docs/tickets/README.md contains conventions and examples")

	mirroredGlossary, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "mirrored-harness-and-context-glossary.md"))
	require.NoError(t, err)
	assert.Contains(t, string(mirroredGlossary), "AD-076")
	assert.Contains(t, string(mirroredGlossary), "AD-080")
	assert.Contains(t, string(mirroredGlossary), "AD-082")

	designIndex, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "index.md"))
	require.NoError(t, err)
	assert.Contains(t, string(designIndex), "AD-082")
	assert.Contains(t, string(designIndex), "AD-085")
	assert.Contains(t, string(designIndex), "AD-086")
	assert.Contains(t, string(designIndex), "AD-087")
	assert.Contains(t, string(designIndex), "AD-097")
	assert.Contains(t, string(designIndex), "AD-098")
	assert.Contains(t, string(designIndex), "AD-099")
	assert.Contains(t, string(designIndex), "AD-100")
	assert.Contains(t, string(designIndex), "AD-101")
	assert.Contains(t, string(designIndex), "AD-102")
	assert.Contains(t, string(designIndex), "AD-103")
	assert.Contains(t, string(designIndex), "AD-108")

	codeDocMap, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "code-documentation-map.md"))
	require.NoError(t, err)
	assert.Contains(t, string(codeDocMap), "Code Documentation Map")
	assert.Contains(t, string(codeDocMap), "MarsDocSync")
	assert.Contains(t, string(codeDocMap), "docs:")
	assert.Contains(t, string(codeDocMap), "docsync audit")
	assert.Contains(t, string(codeDocMap), "documentation-sync-architecture.md")
	assert.Contains(t, string(codeDocMap), "cli-tool-skill-sync.md")

	cliToolSkillSync, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "cli-tool-skill-sync.md"))
	require.NoError(t, err)
	assert.Contains(t, string(cliToolSkillSync), "CLI Tool And Skill Synchronization")
	assert.Contains(t, string(cliToolSkillSync), "Universal Operating Model")
	assert.Contains(t, string(cliToolSkillSync), "mars_harness_cli")
	assert.Contains(t, string(cliToolSkillSync), "repo shortcut")

	docSyncArchitecture, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "documentation-sync-architecture.md"))
	require.NoError(t, err)
	assert.Contains(t, string(docSyncArchitecture), "Documentation Sync Architecture")
	assert.Contains(t, string(docSyncArchitecture), "Universal Operating Model")
	assert.Contains(t, string(docSyncArchitecture), "docsync_audit")
	assert.Contains(t, string(docSyncArchitecture), "Role Responsibilities")

	conversationRecord, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "conversation-as-system-record.md"))
	require.NoError(t, err)
	assert.Contains(t, string(conversationRecord), "Conversation As System Record")
	assert.Contains(t, string(conversationRecord), "Trivial command responses")
	assert.Contains(t, string(conversationRecord), "active-plan hygiene")

	tenets, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "tenets.md"))
	require.NoError(t, err)
	assert.Contains(t, string(tenets), "Progressive Autonomy")
	assert.Contains(t, string(tenets), "Context Efficiency")

	operatingModel, err := os.ReadFile(filepath.Join(dir, "docs", "design-docs", "delivery-operating-model.md"))
	require.NoError(t, err)
	assert.Contains(t, string(operatingModel), "Operating-model changes must be symbiotic")
	assert.Contains(t, string(operatingModel), "without handoff gaps")
	assert.Contains(t, string(operatingModel), "repeated process promotion to formalized tools")
	assert.Contains(t, string(operatingModel), "Built-in tool creation must dogfood the meta-tool path")
	assert.Contains(t, string(operatingModel), "active exec plan first, then feature contract, then")
	assert.Contains(t, string(operatingModel), "AD-097: Business Logic Is First-Class BDD")
	assert.Contains(t, string(operatingModel), "Business logic is first-class BDD")
	assert.Contains(t, string(operatingModel), "AD-098: No Stale Documentation")
	assert.Contains(t, string(operatingModel), "AD-103: CLI Tool And Skill Synchronization")
	assert.Contains(t, string(operatingModel), "AD-108: Remote Trunk Freshness And Immediate Publishing")
	assert.Contains(t, string(operatingModel), "origin/main")
	assert.Contains(t, string(operatingModel), "MarsDocSync")
	assert.Contains(t, string(operatingModel), "code-documentation-map.md")
	assert.Contains(t, string(operatingModel), "docsync_audit")

	execPlanReadme, err := os.ReadFile(filepath.Join(dir, "docs", "exec-plans", "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(execPlanReadme), "only one active exec plan")
	assert.Contains(t, string(execPlanReadme), "## Planning Order")
	assert.Contains(t, string(execPlanReadme), "exec plan, feature contract, ticket, delivery")
	assert.Contains(t, string(execPlanReadme), "active/current-operating-plan.md")
	assert.Contains(t, string(execPlanReadme), "backlog plans")
	assert.Contains(t, string(execPlanReadme), "**Depends On**")
	assert.Contains(t, string(execPlanReadme), "**Blocks**")
	assert.Contains(t, string(execPlanReadme), "**Goals**")
	assert.Contains(t, string(execPlanReadme), "**BDD Feature**")
	assert.Contains(t, string(execPlanReadme), "**Scenario Schedule**")
	assert.Contains(t, string(execPlanReadme), "All business logic must be documented step by step")
	assert.Contains(t, string(execPlanReadme), "No stale documentation")

	currentPlan, err := os.ReadFile(filepath.Join(dir, "docs", "exec-plans", "active", "current-operating-plan.md"))
	require.NoError(t, err)
	assert.Contains(t, string(currentPlan), "**Status:** Active")
	assert.Contains(t, string(currentPlan), "**Priority:** P0")
	assert.Contains(t, string(currentPlan), "**Depends On:**")
	assert.Contains(t, string(currentPlan), "**Blocks:**")
	assert.Contains(t, string(currentPlan), "**Related Tickets:**")
	assert.Contains(t, string(currentPlan), "**Goals:** G-001")
	assert.Contains(t, string(currentPlan), "**BDD Feature:** F-001")
	assert.Contains(t, string(currentPlan), "Space Invaders")
	assert.Contains(t, string(currentPlan), "browser arcade game")
	assert.Contains(t, string(currentPlan), "**Current Failing Scenario:**")
	assert.Contains(t, string(currentPlan), "product-specific walking skeleton")
	assert.Contains(t, string(currentPlan), "ordinary product ticket")

	qualityScore, err := os.ReadFile(filepath.Join(dir, "docs", "QUALITY_SCORE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(qualityScore), "## Grading Scale")
	assert.Contains(t, string(qualityScore), "Harness readiness")
	assert.Contains(t, string(qualityScore), "shipped feature scenarios")
	assert.Contains(t, string(qualityScore), "enabler work")
	assert.Contains(t, string(qualityScore), "scores export")

	goals, err := os.ReadFile(filepath.Join(dir, "docs", "goals", "active.md"))
	require.NoError(t, err)
	assert.Contains(t, string(goals), "G-001")
	assert.Contains(t, string(goals), "Status: active")
	assert.Contains(t, string(goals), "Space Invaders")
	assert.Contains(t, string(goals), "Category: product")

	featureContract, err := os.ReadFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"))
	require.NoError(t, err)
	assert.Contains(t, string(featureContract), "Feature ID: F-001")
	assert.Contains(t, string(featureContract), "Product Walking Skeleton")
	assert.Contains(t, string(featureContract), "Space Invaders")
	assert.Contains(t, string(featureContract), "browser arcade game")
	assert.Contains(t, string(featureContract), "## Business Logic")
	assert.Contains(t, string(featureContract), "## Step-By-Step Behavior")
	assert.Contains(t, string(featureContract), "Project Brief Becomes A Visible Product Slice")
	assert.Contains(t, string(featureContract), "First Product Behavior Is Runnable Or Inspectable")
	assert.Contains(t, string(featureContract), "Product Evidence Comes Before Governance Expansion")
	assert.NotContains(t, string(featureContract), "Remote Trunk Freshness And Immediate Publishing")
	assert.Contains(t, string(featureContract), "Given")
	assert.Contains(t, string(featureContract), "When")
	assert.Contains(t, string(featureContract), "Then")

	featuresReadme, err := os.ReadFile(filepath.Join(dir, "docs", "features", "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(featuresReadme), "Business Logic Is First-Class BDD")
	assert.Contains(t, string(featuresReadme), "No Stale Documentation")
	assert.Contains(t, string(featuresReadme), "MarsDocSync")
	assert.Contains(t, string(featuresReadme), "docsync audit")
	assert.Contains(t, string(featuresReadme), "documentation-sync-architecture.md")
	assert.Contains(t, string(featuresReadme), "cli-tool-skill-sync.md")
	assert.Contains(t, string(featuresReadme), "Business logic is documented step by step")
	assert.Contains(t, string(featuresReadme), "Feature contracts come after the active exec plan")

	ticketsReadme, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(ticketsReadme), "Tickets come after planning and feature contracts")
	assert.Contains(t, string(ticketsReadme), "MarsDocSync")

	assert.Contains(t, string(agentGuide), "Bootstrap and delivery order is strict")

	ceoPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "ceo.md"))
	require.NoError(t, err)
	assert.Contains(t, string(ceoPrompt), "You own vision, active goals, and final strategy/scope")
	assert.Contains(t, string(ceoPrompt), "Do not write docs/exec-plans/active/current-operating-plan.md")
	assert.Contains(t, string(ceoPrompt), "route to COO")
	assert.Contains(t, string(ceoPrompt), "route to CTO")

	cooPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "coo.md"))
	require.NoError(t, err)
	assert.Contains(t, string(cooPrompt), "Do not use ticket_create")
	assert.Contains(t, string(cooPrompt), "Do not create or edit application source files")
	assert.Contains(t, string(cooPrompt), "docs/features/F-NNN*.md")
	assert.Contains(t, string(cooPrompt), "target_role: cto-weekly")

	orchestratorPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "orchestrator.md"))
	require.NoError(t, err)
	assert.Contains(t, string(orchestratorPrompt), "treat slugged matches")
	assert.Contains(t, string(orchestratorPrompt), "without ticket-state change")
	assert.Contains(t, string(orchestratorPrompt), "ticket_shaping")
	assert.Contains(t, string(orchestratorPrompt), "source_disposition")
	assert.Contains(t, string(orchestratorPrompt), "CEO -> COO -> CTO -> Engineer")

	engineerPrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "engineer.md"))
	require.NoError(t, err)
	assert.Contains(t, string(engineerPrompt), "No stale documentation")
	assert.Contains(t, string(engineerPrompt), "MarsDocSync")

	qaPrompt, err = os.ReadFile(filepath.Join(dir, ".harness", "roles", "qa.md"))
	require.NoError(t, err)
	assert.Contains(t, string(qaPrompt), "MarsDocSync")

	releaseDoc, err = os.ReadFile(filepath.Join(dir, "docs", "design-docs", "release-versioning.md"))
	require.NoError(t, err)
	assert.Contains(t, string(releaseDoc), "Every non-release semantic commit")
	assert.Contains(t, string(releaseDoc), "release: notes X.Y.Z")
	assert.Contains(t, string(releaseDoc), "GitHub Release")
	assert.Contains(t, string(releaseDoc), "vX.Y.Z")
	assert.Contains(t, string(releaseDoc), "notes-only GitHub")
	assert.Contains(t, string(releaseDoc), "Impact")
	assert.Contains(t, string(releaseDoc), "Why")
	assert.Contains(t, string(releaseDoc), "What Changed")

	releasePrompt, err := os.ReadFile(filepath.Join(dir, ".harness", "roles", "release-manager.md"))
	require.NoError(t, err)
	assert.Contains(t, string(releasePrompt), "Treat every non-release semantic commit")
	assert.Contains(t, string(releasePrompt), "Do not generate another version")
	assert.Contains(t, string(releasePrompt), "Separate shipped feature scenarios from enabler work")
	assert.Contains(t, string(releasePrompt), "Impact")
	assert.Contains(t, string(releasePrompt), "What Changed")
	assert.Contains(t, string(releasePrompt), "publish or update GitHub Release")
	assert.Contains(t, string(releasePrompt), "notes-only release is a blocker")
}

func TestDefaultHeadOfStrategyPromptIncludesPersonalGuide(t *testing.T) {
	t.Parallel()

	prompt := defaultRolePrompt("head-of-strategy", defaultRolePrompts["head-of-strategy"])
	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "## Personal Guide")
	assert.Contains(t, prompt, "### Modus Operandi")
	assert.Contains(t, prompt, "### Priorities")
	assert.Contains(t, prompt, "### Owns")
	assert.Contains(t, prompt, "### Does Not Own")
	assert.Contains(t, prompt, "### Best Feedback Format")
	assert.Contains(t, prompt, "### How I Like To Receive Feedback")
	assert.Contains(t, prompt, "### Stop Conditions")
	assert.Contains(t, prompt, "Final CEO decision")
	assert.Contains(t, prompt, "Decision needed")
	assert.Contains(t, prompt, "The request needs CEO authority rather than strategy advice")
}

func manifestRoleBlock(t *testing.T, manifest, role string) string {
	t.Helper()
	marker := "\n  " + role + ":"
	start := strings.Index(manifest, marker)
	require.NotEqual(t, -1, start, "manifest missing role block %s", role)
	block := manifest[start+1:]
	if end := strings.Index(block[1:], "\n\n  "); end >= 0 {
		block = block[:end+1]
	}
	return block
}

func TestInit_alreadyExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))

	err := Init(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInit_forceOverwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))

	err := Init(dir, true)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, ".harness", "manifest.yaml"))
}

func TestInit_forcePreservesExistingContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	require.NoError(t, Init(dir, false))

	ticketDir := filepath.Join(dir, "docs", "tickets", "in-progress")
	ticketPath := filepath.Join(ticketDir, "T-001-user-work.md")
	require.NoError(t, os.MkdirAll(ticketDir, 0o755))
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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))
	err := os.WriteFile(filepath.Join(dir, ".harness", "manifest.yaml"),
		[]byte("name: bad\n"), 0o644)
	require.NoError(t, err)

	_, err = EnsureHarness(dir, false)
	require.Error(t, err)
}

func TestDetectFramework_goMod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644))

	fw := detectFramework(dir, []string{"go.mod"})
	assert.Equal(t, "Go Module", fw)
}

func TestDetectFramework_cargoToml(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644))

	fw := detectFramework(dir, []string{"Cargo.toml"})
	assert.Equal(t, "Rust/Cargo", fw)
}

// --- Bootability check tests ---

func TestBootability_missingDevScript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"test": "jest"}
	}`), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build", "test": "jest"}
	}`), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))
	appDir := filepath.Join(dir, "src", "app", "(dashboard)")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "page.tsx"), []byte("export default function Page() { return <div/>; }"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))
	appDir := filepath.Join(dir, "src", "app")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "layout.tsx"), []byte("export default function Layout({children}) { return <html><body>{children}</body></html>; }"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))

	// app/ at root, pages/ under src/ — conflict
	rootApp := filepath.Join(dir, "app")
	require.NoError(t, os.MkdirAll(rootApp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootApp, "page.tsx"), []byte("export default function P() { return <div/>; }"), 0o644))

	srcPages := filepath.Join(dir, "src", "pages", "auth")
	require.NoError(t, os.MkdirAll(srcPages, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcPages, "login.tsx"), []byte("export default function Login() { return <div/>; }"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))

	srcApp := filepath.Join(dir, "src", "app")
	require.NoError(t, os.MkdirAll(srcApp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644))

	srcPages := filepath.Join(dir, "src", "pages", "api")
	require.NoError(t, os.MkdirAll(srcPages, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcPages, "health.ts"), []byte("export default function handler() {}"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))

	srcApp := filepath.Join(dir, "src", "app")
	require.NoError(t, os.MkdirAll(srcApp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644))

	stylesDir := filepath.Join(dir, "src", "styles")
	require.NoError(t, os.MkdirAll(stylesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stylesDir, "globals.css"), []byte("@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	stylesDir := filepath.Join(dir, "src", "styles")
	require.NoError(t, os.MkdirAll(stylesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stylesDir, "globals.css"), []byte("@tailwind base;\n@tailwind components;\n@tailwind utilities;\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tailwind.config.js"), []byte("module.exports = {};\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "postcss.config.js"), []byte("module.exports = {};\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(`const nextConfig = { experimental: { appDir: true } }; module.exports = nextConfig;`), 0o644))

	srcApp := filepath.Join(dir, "src", "app")
	require.NoError(t, os.MkdirAll(srcApp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
		"compilerOptions": {
			"baseUrl": ".",
			"paths": { "@/*": ["./*"] }
		}
	}`), 0o644))

	srcApp := filepath.Join(dir, "src", "app")
	require.NoError(t, os.MkdirAll(srcApp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcApp, "page.tsx"), []byte("export default function P() { return <div/>; }"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name": "test",
		"dependencies": {"next": "^16.0.0"},
		"scripts": {"dev": "next dev", "build": "next build"}
	}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
		"compilerOptions": {
			"baseUrl": ".",
			"paths": { "@/*": ["./src/*"] }
		}
	}`), 0o644))

	srcApp := filepath.Join(dir, "src", "app")
	require.NoError(t, os.MkdirAll(srcApp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcApp, "layout.tsx"), []byte("export default function L({children}) { return <html><body>{children}</body></html>; }"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644))

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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n.next/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644))

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	for _, f := range result.Findings {
		if f.Type == "no_gitignore" {
			t.Fatal("should not report no_gitignore when .gitignore exists at root")
		}
	}
}

func TestScan_reportsWorkspaceHygieneMissingGeneratedIgnore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("dist/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo"}`), 0o644))

	result, err := Scan(context.Background(), Config{RepoRoot: dir})
	require.NoError(t, err)

	found := false
	for _, f := range result.Findings {
		if f.Type == "workspace_hygiene" && strings.Contains(f.Description, "node_modules") {
			found = true
			break
		}
	}
	require.True(t, found, "expected workspace hygiene finding for missing node_modules ignore")
}

func TestUpgrade_preservesUserConfiguredManifestAndPrompts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

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

	metadata, err := ReadHarnessMetadata(dir)
	require.NoError(t, err)
	assert.Equal(t, "mars-harness", metadata.Generator)

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
