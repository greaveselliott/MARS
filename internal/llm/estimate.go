/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/local-inference.md
- docs/design-docs/context-efficiency.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
*/
package llm

// EstimateTokens returns a conservative token count for budgeting.
//
// It uses a character-length heuristic of ~3 characters per token (AD-288).
// The previous ~4 chars/token ratio matched English prose but under-counted
// code-heavy harness prompts: the 2026-06-12 demo-12 engineer wedge measured
// 33,281 served tokens for content the heuristic estimated at 26,188
// (~3.15 chars/token actual on package.json/tool-JSON/source content), so the
// context pruner never fired before llama.cpp rejected the request. Budget
// math must over-estimate, never under-estimate: 3 chars/token bounds the
// measured worst case while staying CGO- and table-free.
func EstimateTokens(messages []Message, tools []ToolDefinition) int {
	n := 0
	for _, m := range messages {
		n += estimateString(m.Role)
		n += estimateString(m.Content)
		n += estimateString(m.Name)
		n += estimateString(m.ToolCallID)
		for _, tc := range m.ToolCalls {
			n += estimateString(tc.ID)
			n += estimateString(tc.Type)
			n += estimateString(tc.Function.Name)
			n += estimateString(tc.Function.Arguments)
		}
	}
	for _, td := range tools {
		n += estimateString(td.Type)
		n += estimateString(td.Function.Name)
		n += estimateString(td.Function.Description)
		n += estimateJSONValue(td.Function.Parameters)
	}
	return n
}

func estimateString(s string) int {
	if s == "" {
		return 0
	}
	// ceil(len/3) — minimum 1 token for non-empty strings (AD-288 calibration).
	chars := len(s)
	tokens := (chars + 2) / 3
	if tokens < 1 {
		return 1
	}
	return tokens
}

func estimateJSONValue(v any) int {
	if v == nil {
		return 0
	}
	// Without importing encoding/json for every estimate call, approximate via fmt would allocate.
	// For parameters we only need a coarse bound; use a type switch for common cases.
	switch t := v.(type) {
	case string:
		return estimateString(t)
	case map[string]any:
		sum := 2 // braces overhead
		for k, val := range t {
			sum += estimateString(k) + estimateJSONValue(val)
		}
		return sum
	case []any:
		sum := 2
		for _, val := range t {
			sum += estimateJSONValue(val)
		}
		return sum
	default:
		return 4
	}
}
