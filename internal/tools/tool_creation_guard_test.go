package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToolCreationGuard_reportsGovernedPath(t *testing.T) {
	t.Parallel()
	root := newToolCreationGuardRoot(t)
	res, err := handleToolCreationGuard(context.Background(), root, []byte(`{"tool_name":"example_tool"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "status: ok")
	require.Contains(t, res.Output, "PASS: docs/design-docs/tools-glossary.md contains New built-in tools must originate through `tool_create`")
	require.Contains(t, res.Output, "PASS: internal/tools/example_tool.go exists")
	require.Contains(t, res.Output, "PASS: docs/design-docs/tools-glossary.md contains `example_tool`")
}

func TestToolCreationGuard_flagsMissingToolArtifacts(t *testing.T) {
	t.Parallel()
	root := newToolCreationGuardRoot(t)
	res, err := handleToolCreationGuard(context.Background(), root, []byte(`{"tool_name":"missing_tool"}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "status: needs_attention")
	require.Contains(t, res.Output, "FAIL: internal/tools/missing_tool.go missing")
}

func newToolCreationGuardRoot(t *testing.T) Root {
	t.Helper()
	dir := t.TempDir()
	writeGuardFile(t, dir, "docs/design-docs/tools-glossary.md", "New built-in tools must originate through `tool_create`\n`record_decision`\n`example_tool`\n")
	writeGuardFile(t, dir, "docs/design-docs/delivery-operating-model.md", "Built-in tool creation must dogfood the meta-tool path\n")
	writeGuardFile(t, dir, "docs/design-docs/dogfood-and-decisions.md", "bypassing `tool_create` breaks the doctrine it represents\nBypassing `tool_create`\n")
	writeGuardFile(t, dir, "internal/scanner/init.go", "Tool creation path\nexample_tool\n")
	writeGuardFile(t, dir, "internal/docsconsistency/operating_rules_test.go", "TestToolCreationPathIsDocumented\n")
	writeGuardFile(t, dir, "internal/tools/example_tool.go", "package tools\n")
	writeGuardFile(t, dir, "internal/tools/example_tool_test.go", "package tools\n")
	root, err := NewRoot(dir)
	require.NoError(t, err)
	return root
}

func writeGuardFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
