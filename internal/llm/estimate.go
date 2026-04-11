package llm

// EstimateTokens returns a rough token count for budgeting.
// It uses a character-length heuristic (~4 characters per token) which matches
// common tiktoken-style estimates for English-ish text without pulling in CGO or large tables.
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
	// ceil(len/4) — minimum 1 token for non-empty strings
	chars := len(s)
	tokens := (chars + 3) / 4
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
