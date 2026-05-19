/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExecutor_notAllowlisted(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"file_read"}, "shell_exec", `{"argv":["echo","hi"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestExecutor_unknownTool(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	require.NoError(t, RegisterBuiltinTools(reg))
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"not_a_real_tool"}, "not_a_real_tool", `{}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}

func TestExecutor_invalidJSON(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"file_read"}, "file_read", `{`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "valid JSON")
}

func TestExecutor_emptyAllowlistFailsClosed(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, nil, "file_read", `{"path":"README.md"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no tools are allowed")
}

func TestExecutor_toolHandlerHardTimeout(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	require.NoError(t, reg.Register("slow_tool", "test slow tool", json.RawMessage(`{"type":"object"}`), func(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
		time.Sleep(250 * time.Millisecond)
		return ToolResult{Output: "late"}, nil
	}))
	ex := NewExecutor(reg)
	ex.DefaultTTL = 20 * time.Millisecond
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)

	start := time.Now()
	res, err := ex.Execute(context.Background(), root, []string{"slow_tool"}, "slow_tool", `{}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
	require.Equal(t, -1, res.ExitCode)
	require.Less(t, time.Since(start), 150*time.Millisecond)
}

func TestExecutor_observerCannotMutate(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{TrustLevel: "observer"}
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"x.txt","content":"x"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "observer")
}

func TestExecutor_secretScannerBlocksFileWrite(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{TrustLevel: "contributor"}
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	_, err = ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"secret.txt","content":"token = \"github-token-placeholder\""}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "secret scanner")
}
