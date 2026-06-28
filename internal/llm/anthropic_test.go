/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
*/
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnthropicChatCompletionMapsMessagesToolsAndAuth(t *testing.T) {
	var gotHeader string
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("x-api-key")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg_1",
			"model":       "claude-test",
			"stop_reason": "tool_use",
			"content": []map[string]any{
				{"type": "text", "text": "Checking."},
				{"type": "tool_use", "id": "toolu_1", "name": "file_read", "input": map[string]any{"path": "README.md"}},
			},
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:    srv.URL,
		Provider:   "anthropic",
		APIKey:     "secret-value",
		Model:      "claude-test",
		HTTPClient: srv.Client(),
		MaxRetries: 1,
	})
	require.NoError(t, err)

	resp, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "system prompt"},
			{Role: "user", Content: "read"},
		},
		Tools: []ToolDefinition{{
			Type: "function",
			Function: FunctionSpec{
				Name:        "file_read",
				Description: "Read a file",
				Parameters:  map[string]any{"type": "object"},
			},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "/messages", gotPath)
	require.Equal(t, "secret-value", gotHeader)
	require.Equal(t, "claude-test", gotBody["model"])
	require.Equal(t, "system prompt", gotBody["system"])
	require.NotEmpty(t, gotBody["tools"])

	require.Equal(t, "Checking.", resp.Choices[0].Message.Content)
	require.Len(t, resp.Choices[0].Message.ToolCalls, 1)
	require.Equal(t, "file_read", resp.Choices[0].Message.ToolCalls[0].Function.Name)
	require.JSONEq(t, `{"path":"README.md"}`, resp.Choices[0].Message.ToolCalls[0].Function.Arguments)
	require.Equal(t, 15, resp.Usage.TotalTokens)
}
