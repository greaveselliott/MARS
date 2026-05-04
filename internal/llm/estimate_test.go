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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEstimateTokens_nonEmpty(t *testing.T) {
	t.Parallel()
	msgs := []Message{
		{Role: "system", Content: stringsRepeat("a", 40)},
		{Role: "user", Content: "hello world"},
	}
	n := EstimateTokens(msgs, nil)
	require.Greater(t, n, 0)
}

func TestEstimateTokens_tools(t *testing.T) {
	t.Parallel()
	tools := []ToolDefinition{{
		Type: "function",
		Function: FunctionSpec{
			Name:        "demo",
			Description: "d",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"x": map[string]any{"type": "string"},
				},
			},
		},
	}}
	n := EstimateTokens([]Message{{Role: "user", Content: "hi"}}, tools)
	require.Greater(t, n, 4)
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
