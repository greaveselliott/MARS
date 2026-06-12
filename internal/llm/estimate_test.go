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

// TestEstimateTokens_calibrationCoversDemo12Wedge encodes the AD-288
// calibration floor: the 2026-06-12 demo-12 engineer overflow served
// 33,281 tokens for ~104,752 characters of code-heavy prompt content
// (job 0b93881f, trace tr-1781225306000294000). The estimator must report
// at least the served count for that character volume, otherwise the agent
// loop's pruner cannot fire before llama.cpp rejects the request.
func TestEstimateTokens_calibrationCoversDemo12Wedge(t *testing.T) {
	t.Parallel()
	const (
		demo12Chars        = 104752
		demo12ServedTokens = 33281
	)
	msgs := []Message{{Role: "user", Content: stringsRepeat("a", demo12Chars)}}
	n := EstimateTokens(msgs, nil)
	require.GreaterOrEqual(t, n, demo12ServedTokens,
		"estimator must over-estimate the measured served token count so budget pruning fires before the serving window")
}

func TestEstimateTokens_threeCharsPerToken(t *testing.T) {
	t.Parallel()
	msgs := []Message{{Role: "user", Content: stringsRepeat("a", 300)}}
	// role "user" (4 chars → 2 tokens) + 300 chars content (→ 100 tokens)
	require.Equal(t, 102, EstimateTokens(msgs, nil))
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
