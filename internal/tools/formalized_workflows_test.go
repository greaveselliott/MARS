/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/release-versioning.md
- docs/design-docs/tools-glossary.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormalizedWorkflowTools_registered(t *testing.T) {
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	for _, name := range []string{
		"release_orchestrate",
		"github_release_status",
		"architecture_audit",
		"harness_doctrine_sync",
		"docsync_audit",
		"git_release_guard",
		"tool_creation_guard",
		"tool_inventory_audit",
		"task_trace_summarize",
	} {
		_, _, ok := reg.Lookup(name)
		require.True(t, ok, "%s should be registered", name)
	}
}

func TestDocSyncAudit_reportsMetadataStatus(t *testing.T) {
	root := newWorkflowToolRoot(t)
	writeWorkflowFile(t, root.Abs(), "docs/design-docs/code-documentation-map.md", "map")
	writeWorkflowFile(t, root.Abs(), "docs/design-docs/delivery-operating-model.md", "delivery")
	writeWorkflowFile(t, root.Abs(), "docs/design-docs/release-versioning.md", "release")
	writeWorkflowFile(t, root.Abs(), "docs/features/F-009-release-update-lifecycle.md", "release feature")
	writeWorkflowFile(t, root.Abs(), "docs/features/F-004-target-harness-lifecycle.md", "target feature")
	writeWorkflowFile(t, root.Abs(), "internal/scanner/init.go", `/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-004-target-harness-lifecycle.md
*/
package scanner
`)
	writeWorkflowFile(t, root.Abs(), "internal/release/notes.go", `/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release
`)

	res, err := handleDocSyncAudit(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "docsync: checked")
	require.Contains(t, res.Output, "Status: ok")
}

func TestTaskTraceSummarize_suggestsFormalTools(t *testing.T) {
	root := newWorkflowToolRoot(t)
	res, err := handleTaskTraceSummarize(context.Background(), root, json.RawMessage(`{"notes":"release, tag, verify assets, then audit docs"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "release_orchestrate")
	require.Contains(t, res.Output, "architecture_audit")
	require.Contains(t, res.Output, "formal tools")
}

func TestReleaseWorkflowsUseRepositoryOwnedProducer(t *testing.T) {
	root := newWorkflowToolRoot(t)
	result, err := handleReleaseOrchestrate(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, result.Output, "repository's approved release producer")
	require.NotContains(t, result.Output, "publish-assets")
	require.NotContains(t, result.Output, "release verify-assets")
	require.NotContains(t, result.Output, "release audit")

	status, err := handleGithubReleaseStatus(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, status.Output, "repository's approved release workflow")
	require.NotContains(t, status.Output, "publish-assets")
	require.NotContains(t, status.Output, "release verify-assets")
	require.NotContains(t, status.Output, "release audit")
}

func TestReleaseWorkflowBlocksFoundationSourcePublication(t *testing.T) {
	root := newWorkflowToolRoot(t)
	writeWorkflowFile(t, root.Abs(), "cmd/mars/main.go", "package main\n")
	writeWorkflowFile(t, root.Abs(), "docs/roles/personas/foundation-maintainer.md", "# Foundation Maintainer\n")

	result, err := handleReleaseOrchestrate(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, result.Output, "active F-018 plan")
	require.Contains(t, result.Output, "Do not create or move a tag, upload, sign, announce, publish")
	require.Contains(t, result.Output, ".github/workflows/release-snapshot.yml")
	require.NotContains(t, result.Output, "Tag the release-note commit")
	require.NotContains(t, result.Output, "release notes")
	require.NotContains(t, result.Output, "release: notes")
	require.NotContains(t, result.Output, "vX.Y.Z")
	require.NotContains(t, result.Output, "publish-assets")
}

func TestToolInventoryAudit_reportsRegistryAndGlossary(t *testing.T) {
	root := newWorkflowToolRoot(t)
	res, err := handleToolInventoryAudit(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "Registered tools:")
	require.Contains(t, res.Output, "release_orchestrate")
	require.Contains(t, res.Output, "PASS: glossary includes release_orchestrate")
}

func TestGitReleaseGuardReportsStaleReleaseTag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	require.NoError(t, err)
	writeWorkflowFile(t, dir, "VERSION", "0.1.0\n")
	writeWorkflowFile(t, dir, "CHANGELOG.md", "# Changelog\n")
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "feat: seed product"))
	previous := strings.TrimSpace(gitOutput(context.Background(), root, "rev-parse", "HEAD"))
	writeWorkflowFile(t, dir, "VERSION", "0.2.0\n")
	writeWorkflowFile(t, dir, "CHANGELOG.md", "# Changelog\n\n## [0.2.0]\n")
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "release: notes 0.2.0"))
	require.NoError(t, runGitExit0(context.Background(), root, "tag", "v0.2.0", previous))

	res, err := handleGitReleaseGuard(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "FAIL: tag v0.2.0 exists but does not point at release-note HEAD")
}

func TestArchitectureAudit_detectsCurrentTerms(t *testing.T) {
	root := newWorkflowToolRoot(t)
	res, err := handleArchitectureAudit(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "PASS: ARCHITECTURE.md contains mars_cli")
	require.Contains(t, res.Output, "PASS: ARCHITECTURE.md omits stale bundle.lock.json")
}

func newWorkflowToolRoot(t *testing.T) Root {
	t.Helper()
	dir := t.TempDir()
	writeWorkflowFile(t, dir, "VERSION", "1.2.3\n")
	writeWorkflowFile(t, dir, "ARCHITECTURE.md", `
mars update tool
mars_cli
.harness/metadata.yaml
docs/QUALITY_SCORE.md
BDD-led
it does not clone a fresh working directory per job
`)
	writeWorkflowFile(t, dir, "docs/design-docs/tools-glossary.md", `
`+"`release_orchestrate`"+`
`+"`github_release_status`"+`
`+"`architecture_audit`"+`
`+"`harness_doctrine_sync`"+`
`+"`docsync_audit`"+`
`+"`git_release_guard`"+`
`+"`tool_creation_guard`"+`
`+"`tool_inventory_audit`"+`
`+"`task_trace_summarize`"+`
`+"`code_index`"+`
`+"`code_search`"+`
`+"`code_snippet`"+`
`+"`code_trace`"+`
`+"`code_impact`"+`
`+"`file_read`"+`
`+"`file_write`"+`
`+"`file_search`"+`
`+"`grep`"+`
`+"`shell_exec`"+`
`+"`mars_cli`"+`
`+"`record_decision`"+`
`+"`ticket_create`"+`
`+"`tool_create`"+`
`+"`git_status`"+`
`+"`git_diff`"+`
`+"`git_commit`"+`
`+"`git_push`"+`
`)
	writeWorkflowFile(t, dir, "AGENTS.md", "Operating model\nMirrored tools\ndocs/design-docs/tools-glossary.md\ndocs/design-docs/documentation-sync-architecture.md\ndocs/design-docs/cli-tool-skill-sync.md\n")
	writeWorkflowFile(t, dir, "docs/design-docs/harness-glossary.md", "Symbiotic operating-model change\nFormalized tool creation trigger\n")
	writeWorkflowFile(t, dir, "docs/design-docs/delivery-operating-model.md", "formalized tools\nrepeated process\ndocsync_audit\ndocumentation-sync-architecture.md\ncli-tool-skill-sync.md\n")
	writeWorkflowFile(t, dir, "docs/design-docs/documentation-sync-architecture.md", "AD-102\nUniversal Operating Model\ndocsync_audit\n")
	writeWorkflowFile(t, dir, "docs/design-docs/cli-tool-skill-sync.md", "AD-103\nmars_cli\nrepo shortcut map\nskills\n")
	writeWorkflowFile(t, dir, "internal/scanner/init.go", "release_orchestrate\ndocsync_audit\ndocumentation-sync-architecture.md\ncli-tool-skill-sync.md\nFormalized tool creation trigger\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	return root
}

func writeWorkflowFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
