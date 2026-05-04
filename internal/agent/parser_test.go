/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/llm"

	"github.com/stretchr/testify/require"
)

func TestToolCallsFromAssistantMessage_invalidJSONObjectReturnsError(t *testing.T) {
	t.Parallel()
	_, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: `{not json`})
	require.Error(t, err)
}

func TestToolCallsFromAssistantMessage_plainProseIsNotToolJSON(t *testing.T) {
	t.Parallel()
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: "done."})
	require.NoError(t, err)
	require.Nil(t, calls)
}

func TestToolCallsFromAssistantMessage_structured(t *testing.T) {
	t.Parallel()
	msg := llm.Message{
		Role: "assistant",
		ToolCalls: []llm.ToolCall{{
			ID:   "c1",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "demo",
				Arguments: `{"x":1}`,
			},
		}},
	}
	calls, err := ToolCallsFromAssistantMessage(msg)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "demo", calls[0].Function.Name)
}

func TestToolCallsFromAssistantMessage_markdownFence(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join("testdata", "assistant_fence.txt"))
	require.NoError(t, err)
	msg := llm.Message{Role: "assistant", Content: string(b)}
	calls, err := ToolCallsFromAssistantMessage(msg)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "file_read", calls[0].Function.Name)
}

func TestToolCallsFromAssistantMessage_pythonBooleans(t *testing.T) {
	t.Parallel()
	raw := `[{"name":"noop","id":"c1","arguments":"{\"active\": True, \"ok\": False}"}]`
	msg := llm.Message{Role: "assistant", Content: raw}
	calls, err := ToolCallsFromAssistantMessage(msg)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.True(t, json.Valid([]byte(calls[0].Function.Arguments)))
}

func TestToolCallsFromRecordedLlamaFixture(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join("testdata", "recorded_llama.txt"))
	require.NoError(t, err)
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: string(b)})
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "file_read", calls[0].Function.Name)
}

func TestToolCallsFromAssistantMessage_internalQuotes(t *testing.T) {
	t.Parallel()
	raw := `[{"name":"echo","id":"c2","arguments":"{\"msg\": \"say \\\"hi\\\"\"}"}]`
	msg := llm.Message{Role: "assistant", Content: raw}
	calls, err := ToolCallsFromAssistantMessage(msg)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Contains(t, calls[0].Function.Arguments, "hi")
}
