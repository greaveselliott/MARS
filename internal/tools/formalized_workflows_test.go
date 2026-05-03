package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
		"git_release_guard",
		"tool_creation_guard",
		"tool_inventory_audit",
		"task_trace_summarize",
	} {
		_, _, ok := reg.Lookup(name)
		require.True(t, ok, "%s should be registered", name)
	}
}

func TestTaskTraceSummarize_suggestsFormalTools(t *testing.T) {
	root := newWorkflowToolRoot(t)
	res, err := handleTaskTraceSummarize(context.Background(), root, json.RawMessage(`{"notes":"release, tag, verify assets, then audit docs"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "release_orchestrate")
	require.Contains(t, res.Output, "architecture_audit")
	require.Contains(t, res.Output, "formal tools")
}

func TestToolInventoryAudit_reportsRegistryAndGlossary(t *testing.T) {
	root := newWorkflowToolRoot(t)
	res, err := handleToolInventoryAudit(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "Registered tools:")
	require.Contains(t, res.Output, "release_orchestrate")
	require.Contains(t, res.Output, "PASS: glossary includes release_orchestrate")
}

func TestArchitectureAudit_detectsCurrentTerms(t *testing.T) {
	root := newWorkflowToolRoot(t)
	res, err := handleArchitectureAudit(context.Background(), root, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "PASS: ARCHITECTURE.md contains mars_harness_cli")
	require.Contains(t, res.Output, "PASS: ARCHITECTURE.md omits stale bundle.lock.json")
}

func newWorkflowToolRoot(t *testing.T) Root {
	t.Helper()
	dir := t.TempDir()
	writeWorkflowFile(t, dir, "VERSION", "1.2.3\n")
	writeWorkflowFile(t, dir, "ARCHITECTURE.md", `
mars-harness update tool
mars_harness_cli
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
`+"`git_release_guard`"+`
`+"`tool_creation_guard`"+`
`+"`tool_inventory_audit`"+`
`+"`task_trace_summarize`"+`
`+"`file_read`"+`
`+"`file_write`"+`
`+"`file_search`"+`
`+"`grep`"+`
`+"`shell_exec`"+`
`+"`mars_harness_cli`"+`
`+"`record_decision`"+`
`+"`ticket_create`"+`
`+"`tool_create`"+`
`+"`git_status`"+`
`+"`git_diff`"+`
`+"`git_commit`"+`
`+"`git_push`"+`
`)
	writeWorkflowFile(t, dir, "docs/design-docs/harness-glossary.md", "Symbiotic operating-model change\nFormalized tool creation trigger\n")
	writeWorkflowFile(t, dir, "docs/design-docs/delivery-operating-model.md", "formalized tools\nrepeated process\n")
	writeWorkflowFile(t, dir, "internal/scanner/init.go", "release_orchestrate\nFormalized tool creation trigger\n")
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
