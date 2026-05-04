/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"

	"github.com/stretchr/testify/require"
)

func TestRecorder_headerAndTurns(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	rec := NewRecorder(&buf)
	require.NoError(t, rec.WriteHeader("job-1", "tr-1", "m1"))
	require.NoError(t, rec.WriteTurn(llm.Message{Role: "user", Content: "hi"}, 5))
	require.Contains(t, rec.JSONL(), `"type":"header"`)
	require.Contains(t, rec.JSONL(), `"type":"turn"`)
}

func TestRecorder_truncatesLargeContent(t *testing.T) {
	t.Parallel()
	rec := NewRecorder(nil)
	rec.SetMaxBody(100)
	require.NoError(t, rec.WriteHeader("j", "t", ""))
	big := strings.Repeat("x", 500)
	require.NoError(t, rec.WriteTurn(llm.Message{Role: "tool", Content: big}, 1))
	require.Contains(t, rec.JSONL(), "truncated")
	require.Less(t, len(rec.JSONL()), len(big)+200)
}

func TestRecorder_finalizeFiveTurns(t *testing.T) {
	t.Parallel()
	rec := NewRecorder(nil)
	require.NoError(t, rec.WriteHeader("job", "tid", "model-x"))

	require.NoError(t, rec.WriteTurn(llm.Message{Role: "system", Content: "You are a coding agent."}, 10))
	require.NoError(t, rec.WriteTurn(llm.Message{Role: "user", Content: "Fix the bug in main.go."}, 8))
	require.NoError(t, rec.WriteTurn(llm.Message{
		Role: "assistant",
		ToolCalls: []llm.ToolCall{{
			ID: "c1", Type: "function",
			Function: llm.FunctionCall{Name: "file_read", Arguments: `{"path":"main.go"}`},
		}},
	}, 15))
	require.NoError(t, rec.WriteTurn(llm.Message{
		Role: "tool", ToolCallID: "c1", Content: "package main\nfunc main() { x := 1 }\n",
	}, 20))
	require.NoError(t, rec.WriteTurn(llm.Message{
		Role: "assistant", Content: "The variable x is declared but not used. I'll fix it.",
	}, 12))

	s := rec.Finalize("job", "completed", 12*time.Millisecond, 3, 1, nil)
	require.Equal(t, 5, s.TurnCount)
	require.Equal(t, int64(12), s.WallMs)
	require.Equal(t, "completed", s.Outcome)
	require.Equal(t, 65, s.TotalTokens) // 10+8+15+20+12
	require.Equal(t, 3, s.LLMCalls)
	require.Equal(t, 1, s.ToolInvocations)
}

func TestRecorder_toolNamesInSummary(t *testing.T) {
	t.Parallel()
	rec := NewRecorder(nil)
	require.NoError(t, rec.WriteHeader("j", "t", ""))
	require.NoError(t, rec.WriteTurn(llm.Message{
		Role: "assistant",
		ToolCalls: []llm.ToolCall{{
			ID:   "1",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "demo_tool",
				Arguments: `{}`,
			},
		}},
	}, 10))
	s := rec.Finalize("j", "completed", time.Second, 1, 1, nil)
	require.Contains(t, s.ToolsCalled, "demo_tool")
}

func TestRecorder_fileExportValidJSONL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	f, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	rec := NewRecorder(f)
	require.NoError(t, rec.WriteHeader("j", "t", ""))
	require.NoError(t, rec.WriteTurn(llm.Message{Role: "system", Content: "s"}, 1))
	_ = f.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	require.Len(t, lines, 2)
	var h Header
	require.NoError(t, json.Unmarshal(lines[0], &h))
	require.Equal(t, "header", h.Type)
	var turn Turn
	require.NoError(t, json.Unmarshal(lines[1], &turn))
	require.Equal(t, "turn", turn.Type)
}
