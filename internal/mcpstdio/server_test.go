/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package mcpstdio

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/greaveselliott/mars/internal/tools"
)

func TestServerInitializeAndListTools(t *testing.T) {
	t.Parallel()
	server := newTestServer(t, t.TempDir(), tools.Session{TrustLevel: "observer"})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		"",
	}, "\n")

	var out bytes.Buffer
	require.NoError(t, server.Serve(context.Background(), strings.NewReader(input), &out))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2)
	require.Contains(t, lines[0], `"protocolVersion"`)
	require.Contains(t, lines[1], `"tools"`)
	require.Contains(t, lines[1], `"tool_creation_guard"`)
	require.Contains(t, lines[1], `"inputSchema"`)
}

func TestServerCallTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMCPRepoFile(t, dir, "docs/design-docs/tools-glossary.md", "New built-in tools must originate through `tool_create`\n`record_decision`\n`example_tool`\n")
	writeMCPRepoFile(t, dir, "docs/design-docs/delivery-operating-model.md", "Built-in tool creation must dogfood the meta-tool path\n")
	writeMCPRepoFile(t, dir, "docs/design-docs/dogfood-and-decisions.md", "bypassing `tool_create` breaks the doctrine it represents\n")
	writeMCPRepoFile(t, dir, "internal/scanner/init.go", "Tool creation path\nexample_tool\n")
	writeMCPRepoFile(t, dir, "internal/docsconsistency/operating_rules_test.go", "TestToolCreationPathIsDocumented\n")
	writeMCPRepoFile(t, dir, "internal/tools/example_tool.go", "package tools\n")
	writeMCPRepoFile(t, dir, "internal/tools/example_tool_test.go", "package tools\n")
	server := newTestServer(t, dir, tools.Session{TrustLevel: "observer"})
	input := `{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"tool_creation_guard","arguments":{"tool_name":"example_tool"}}}` + "\n"

	var out bytes.Buffer
	require.NoError(t, server.Serve(context.Background(), strings.NewReader(input), &out))

	var resp map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp))
	require.NotContains(t, resp, "error")
	require.Contains(t, out.String(), "status: ok")
	require.Contains(t, out.String(), "PASS: internal/tools/example_tool.go exists")
}

func TestServerMutatingToolBlockedAtObserverTrust(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	server := newTestServer(t, dir, tools.Session{TrustLevel: "observer"})
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"file_write","arguments":{"path":"x.txt","content":"x"}}}` + "\n"

	var out bytes.Buffer
	require.NoError(t, server.Serve(context.Background(), strings.NewReader(input), &out))

	require.Contains(t, out.String(), `"isError":true`)
	require.Contains(t, out.String(), "observer cannot run mutating tool")
	require.NoFileExists(t, filepath.Join(dir, "x.txt"))
}

func newTestServer(t *testing.T, dir string, session tools.Session) Server {
	t.Helper()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	registry, err := tools.DefaultRegistry()
	require.NoError(t, err)
	executor := tools.NewExecutor(registry)
	executor.Session = &session
	return Server{
		Registry: registry,
		Executor: executor,
		Root:     root,
	}
}

func writeMCPRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
