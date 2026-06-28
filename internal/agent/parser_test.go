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

	"github.com/greaveselliott/mars/internal/llm"

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

func TestToolCallsFromAssistantMessage_functionTags(t *testing.T) {
	t.Parallel()
	raw := `I'll inspect the repo.

<function=file_read>
<parameter=path>
README.md
</parameter>
</function>
</tool_call>`
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: raw})
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "file_read", calls[0].Function.Name)
	require.JSONEq(t, `{"path":"README.md"}`, calls[0].Function.Arguments)
}

func TestToolCallsFromAssistantMessage_multipleFunctionTags(t *testing.T) {
	t.Parallel()
	raw := `<function=file_read><parameter=path>README.md</parameter></function>
<function=grep><parameter=pattern>F-001</parameter><parameter=glob>docs/features/*.md</parameter></function>`
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: raw})
	require.NoError(t, err)
	require.Len(t, calls, 2)
	require.Equal(t, "file_read", calls[0].Function.Name)
	require.JSONEq(t, `{"path":"README.md"}`, calls[0].Function.Arguments)
	require.Equal(t, "grep", calls[1].Function.Name)
	require.JSONEq(t, `{"pattern":"F-001","glob":"docs/features/*.md"}`, calls[1].Function.Arguments)
}

func TestToolCallsFromAssistantMessage_inlineToolCallTags(t *testing.T) {
	t.Parallel()
	raw := `I will read the ticket.

<tool_call>file_read{path:<|"|>docs/tickets/done/T-001.md<|"|>}</tool_call>`
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: raw})
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "file_read", calls[0].Function.Name)
	require.JSONEq(t, `{"path":"docs/tickets/done/T-001.md"}`, calls[0].Function.Arguments)
}

func TestToolCallsFromAssistantMessage_inlineDispositionTag(t *testing.T) {
	t.Parallel()
	raw := `<tool_call>job_disposition_record{blocked_by:[<|"|>T-001<|"|>],evidence_links:[<|"|>docs/tickets/done/T-001.md<|"|>,<|"|>src/game.js<|"|>],next_need:<|"|>liveness<|"|>,reason:<|"|>Cannot begin QA review until the ticket is readable.<|"|>,status:<|"|>blocked<|"|>}</tool_call>`
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: raw})
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "job_disposition_record", calls[0].Function.Name)
	require.JSONEq(t, `{
		"blocked_by":["T-001"],
		"evidence_links":["docs/tickets/done/T-001.md","src/game.js"],
		"next_need":"liveness",
		"reason":"Cannot begin QA review until the ticket is readable.",
		"status":"blocked"
	}`, calls[0].Function.Arguments)
}

func TestToolCallsFromAssistantMessage_inlineDispositionNestedFeedback(t *testing.T) {
	t.Parallel()
	raw := `<tool_call>job_disposition_record{status:<|"|>changes_requested<|"|>,ticket_id:<|"|>T-001<|"|>,feedback:{for_role:<|"|>engineer<|"|>,requested_change:<|"|>Add browser evidence, then rerun QA.<|"|>,severity:<|"|>medium<|"|>},evidence_links:[]}</tool_call>`
	calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: raw})
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "job_disposition_record", calls[0].Function.Name)
	require.JSONEq(t, `{
		"status":"changes_requested",
		"ticket_id":"T-001",
		"feedback":{
			"for_role":"engineer",
			"requested_change":"Add browser evidence, then rerun QA.",
			"severity":"medium"
		},
		"evidence_links":[]
	}`, calls[0].Function.Arguments)
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
