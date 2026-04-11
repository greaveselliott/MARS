package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

// ToolCallsFromAssistantMessage returns tool calls from structured fields or from
// JSON embedded in markdown / prose (common with local models).
func ToolCallsFromAssistantMessage(msg llm.Message) ([]llm.ToolCall, error) {
	if len(msg.ToolCalls) > 0 {
		out := make([]llm.ToolCall, len(msg.ToolCalls))
		copy(out, msg.ToolCalls)
		return out, nil
	}
	raw := strings.TrimSpace(msg.Content)
	if raw == "" {
		return nil, nil
	}
	calls, err := parseToolCallsFromText(raw)
	if err != nil && looksLikeToolJSON(raw) {
		return nil, err
	}
	return calls, nil
}

// looksLikeToolJSON is true when the assistant text plausibly attempted tool JSON (vs plain prose).
func looksLikeToolJSON(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return true
	}
	return strings.Contains(s, "```")
}

// normalizeToolArgumentsJSON applies small repairs before json.Valid / execution.
func normalizeToolArgumentsJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "{}"
	}
	// Python-style booleans in JSON-like tool args (quoted keys assumed elsewhere).
	s = regexp.MustCompile(`:\s*True\b`).ReplaceAllString(s, ": true")
	s = regexp.MustCompile(`:\s*False\b`).ReplaceAllString(s, ": false")
	s = regexp.MustCompile(`:\s*None\b`).ReplaceAllString(s, ": null")
	return s
}

func parseToolCallsFromText(src string) ([]llm.ToolCall, error) {
	src = stripMarkdownFences(src)
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, nil
	}
	// Single object or array of tool call objects (informal schema).
	if !json.Valid([]byte(src)) {
		// Try to locate first JSON array or object in text.
		if i := strings.Index(src, "["); i >= 0 {
			if j := findMatchingBracket(src, i, '[', ']'); j > i {
				chunk := src[i : j+1]
				if json.Valid([]byte(chunk)) {
					src = chunk
				}
			}
		} else if i := strings.Index(src, "{"); i >= 0 {
			if j := findMatchingBrace(src, i); j > i {
				chunk := src[i : j+1]
				if json.Valid([]byte(chunk)) {
					src = chunk
				}
			}
		}
	}
	if !json.Valid([]byte(src)) {
		return nil, fmt.Errorf("agent: no valid JSON tool call payload in assistant text")
	}
	src = normalizeToolArgumentsJSON(src)
	trim := strings.TrimSpace(src)
	if strings.HasPrefix(trim, "[") {
		var raw []struct {
			ID        string          `json:"id"`
			Type      string          `json:"type"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(trim), &raw); err != nil {
			return nil, fmt.Errorf("agent: parse tool call array: %w", err)
		}
		var out []llm.ToolCall
		for i, r := range raw {
			id := r.ID
			if id == "" {
				id = fmt.Sprintf("call_%d", i)
			}
			args := string(r.Arguments)
			if args == "" {
				args = "{}"
			}
			args = normalizeToolArgumentsJSON(args)
			if !json.Valid([]byte(args)) {
				return nil, fmt.Errorf("agent: tool %q arguments are not valid JSON", r.Name)
			}
			out = append(out, llm.ToolCall{
				ID:   id,
				Type: nonEmpty(r.Type, "function"),
				Function: llm.FunctionCall{
					Name:      r.Name,
					Arguments: args,
				},
			})
		}
		return out, nil
	}
	var one struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(trim), &one); err != nil {
		return nil, fmt.Errorf("agent: parse tool call object: %w", err)
	}
	if one.Name == "" {
		return nil, fmt.Errorf("agent: tool call missing name")
	}
	args := string(one.Arguments)
	if args == "" {
		args = "{}"
	}
	args = normalizeToolArgumentsJSON(args)
	if !json.Valid([]byte(args)) {
		return nil, fmt.Errorf("agent: tool %q arguments are not valid JSON", one.Name)
	}
	id := one.ID
	if id == "" {
		id = "call_0"
	}
	return []llm.ToolCall{{
		ID:   id,
		Type: nonEmpty(one.Type, "function"),
		Function: llm.FunctionCall{
			Name:      one.Name,
			Arguments: args,
		},
	}}, nil
}

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	// ```json ... ``` or ``` ... ```
	re := regexp.MustCompile("(?s)```(?:json|JSON)?\\s*\\n(.*?)```")
	if m := re.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return s
}

func findMatchingBracket(s string, start int, open, close rune) int {
	if start < 0 || start >= len(s) || rune(s[start]) != open {
		return -1
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		r := rune(s[i])
		if inStr {
			if esc {
				esc = false
				continue
			}
			if r == '\\' {
				esc = true
				continue
			}
			if r == '"' {
				inStr = false
			}
			continue
		}
		if r == '"' {
			inStr = true
			continue
		}
		if r == open {
			depth++
		} else if r == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findMatchingBrace(s string, start int) int {
	return findMatchingBracket(s, start, '{', '}')
}

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// fingerprintToolCalls builds a stable string for circle detection (AD-005 order preserved).
func fingerprintToolCalls(calls []llm.ToolCall) string {
	var b bytes.Buffer
	for i, c := range calls {
		if i > 0 {
			b.WriteByte('\x1e')
		}
		args := strings.TrimSpace(c.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		var compact bytes.Buffer
		if json.Valid([]byte(args)) {
			_ = json.Compact(&compact, []byte(args))
			args = compact.String()
		}
		fmt.Fprintf(&b, "%s|%s", c.Function.Name, args)
	}
	return b.String()
}
