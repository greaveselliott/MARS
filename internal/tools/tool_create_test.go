package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToolCreate_scaffoldsToolFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal", "tools"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/greaveselliott/mars-harness\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "tools", "registry.go"), []byte("package tools\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "tools", "register_default.go"), []byte("package tools\n"), 0o644))

	root, err := NewRoot(dir)
	require.NoError(t, err)

	res, err := handleToolCreate(context.Background(), root, []byte(`{
		"name": "cli_reference",
		"description": "Return curated CLI reference snippets.",
		"fields": [
			{"name": "topic", "type": "string", "description": "CLI topic to retrieve", "required": true},
			{"name": "json_output", "type": "boolean", "description": "Whether JSON examples are needed"}
		]
	}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "internal/tools/cli_reference.go")
	require.Contains(t, res.Output, "registerCliReference")
	require.Contains(t, res.Output, "docs/design-docs/tools-glossary.md")

	toolData, err := os.ReadFile(filepath.Join(dir, "internal", "tools", "cli_reference.go"))
	require.NoError(t, err)
	toolText := string(toolData)
	require.Contains(t, toolText, "const cliReferenceSchema")
	require.Contains(t, toolText, "type cliReferenceArgs struct")
	require.Contains(t, toolText, "Topic string `json:\"topic\"`")
	require.Contains(t, toolText, "JsonOutput bool `json:\"json_output\"`")
	require.Contains(t, toolText, "func registerCliReference")
	require.Contains(t, toolText, "handler not implemented yet")

	testData, err := os.ReadFile(filepath.Join(dir, "internal", "tools", "cli_reference_test.go"))
	require.NoError(t, err)
	require.Contains(t, string(testData), "TestCliReference_scaffoldRequiresImplementation")
}

func TestToolCreate_rejectsInvalidName(t *testing.T) {
	t.Parallel()
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	_, err = handleToolCreate(context.Background(), root, []byte(`{"name":"Bad-Tool","description":"bad"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "snake_case")
}

func TestToolCreate_allowsSingleCharacterName(t *testing.T) {
	t.Parallel()
	spec, err := normalizeToolSpec(toolCreateArgs{
		Name:        "x",
		Description: "Tiny tool.",
	})
	require.NoError(t, err)
	require.Equal(t, "X", spec.TypeName)
}

func TestToolCreate_refusesOverwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal", "tools"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/greaveselliott/mars-harness\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "tools", "registry.go"), []byte("package tools\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "tools", "register_default.go"), []byte("package tools\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "tools", "cli_reference.go"), []byte("package tools\n"), 0o644))

	root, err := NewRoot(dir)
	require.NoError(t, err)

	_, err = handleToolCreate(context.Background(), root, []byte(`{"name":"cli_reference","description":"Return CLI docs."}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestDefaultRegistry_includesToolCreate(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	require.Contains(t, reg.Names(), "tool_create")
}
