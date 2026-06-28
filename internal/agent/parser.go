/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/greaveselliott/mars/internal/llm"
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
	if calls, ok, err := parseFunctionTagToolCalls(raw); ok || err != nil {
		return calls, err
	}
	if calls, ok, err := parseInlineToolCallTags(raw); ok || err != nil {
		return calls, err
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
			if r.Name == "" {
				return nil, fmt.Errorf("agent: tool call at index %d missing name", i)
			}
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

func parseFunctionTagToolCalls(src string) ([]llm.ToolCall, bool, error) {
	functionRe := regexp.MustCompile(`(?s)<function=([A-Za-z0-9_.-]+)>(.*?)</function>`)
	paramRe := regexp.MustCompile(`(?s)<parameter=([A-Za-z0-9_.-]+)>(.*?)</parameter>`)
	matches := functionRe.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		return nil, false, nil
	}
	out := make([]llm.ToolCall, 0, len(matches))
	for i, match := range matches {
		name := strings.TrimSpace(match[1])
		if name == "" {
			return nil, true, fmt.Errorf("agent: function tag missing name")
		}
		args := map[string]string{}
		for _, param := range paramRe.FindAllStringSubmatch(match[2], -1) {
			key := strings.TrimSpace(param[1])
			if key == "" {
				return nil, true, fmt.Errorf("agent: function tag parameter missing name")
			}
			args[key] = strings.TrimSpace(param[2])
		}
		argBytes, err := json.Marshal(args)
		if err != nil {
			return nil, true, fmt.Errorf("agent: encode function tag arguments for %q: %w", name, err)
		}
		out = append(out, llm.ToolCall{
			ID:   fmt.Sprintf("call_tag_%d", i),
			Type: "function",
			Function: llm.FunctionCall{
				Name:      name,
				Arguments: string(argBytes),
			},
		})
	}
	return out, true, nil
}

func parseInlineToolCallTags(src string) ([]llm.ToolCall, bool, error) {
	const (
		openTag  = "<tool_call>"
		closeTag = "</tool_call>"
	)
	var out []llm.ToolCall
	rest := src
	for {
		start := strings.Index(rest, openTag)
		if start < 0 {
			break
		}
		afterOpen := rest[start+len(openTag):]
		end := strings.Index(afterOpen, closeTag)
		if end < 0 {
			return nil, true, fmt.Errorf("agent: inline tool_call tag missing closing tag")
		}
		inner := strings.TrimSpace(afterOpen[:end])
		rest = afterOpen[end+len(closeTag):]
		if inner == "" {
			return nil, true, fmt.Errorf("agent: inline tool_call tag is empty")
		}
		brace := strings.Index(inner, "{")
		if brace <= 0 {
			return nil, true, fmt.Errorf("agent: inline tool_call missing name or arguments")
		}
		name := strings.TrimSpace(inner[:brace])
		if name == "" {
			return nil, true, fmt.Errorf("agent: inline tool_call missing name")
		}
		argEnd := findMatchingBrace(inner, brace)
		if argEnd <= brace {
			return nil, true, fmt.Errorf("agent: inline tool_call arguments are not balanced for %q", name)
		}
		args, err := parseInlineToolArgs(inner[brace+1 : argEnd])
		if err != nil {
			return nil, true, fmt.Errorf("agent: parse inline tool_call arguments for %q: %w", name, err)
		}
		argBytes, err := json.Marshal(args)
		if err != nil {
			return nil, true, fmt.Errorf("agent: encode inline tool_call arguments for %q: %w", name, err)
		}
		out = append(out, llm.ToolCall{
			ID:   fmt.Sprintf("call_inline_%d", len(out)),
			Type: "function",
			Function: llm.FunctionCall{
				Name:      name,
				Arguments: string(argBytes),
			},
		})
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func parseInlineToolArgs(src string) (map[string]any, error) {
	args := map[string]any{}
	for _, part := range splitInlineTopLevel(src, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := indexInlineTopLevel(part, ':')
		if idx <= 0 {
			return nil, fmt.Errorf("argument %q is missing ':'", part)
		}
		key := strings.Trim(strings.TrimSpace(part[:idx]), `"'`)
		if key == "" {
			return nil, fmt.Errorf("argument %q has empty key", part)
		}
		value, err := parseInlineToolValue(part[idx+1:])
		if err != nil {
			return nil, fmt.Errorf("argument %q: %w", key, err)
		}
		args[key] = value
	}
	return args, nil
}

func parseInlineToolValue(src string) (any, error) {
	s := normalizeInlineSentinel(strings.TrimSpace(src))
	if s == "" {
		return "", nil
	}
	if strings.HasPrefix(s, `<|"|>`) {
		if !strings.HasSuffix(s, `<|"|>`) || len(s) < len(`<|"|>`)*2 {
			return nil, fmt.Errorf("unterminated sentinel string")
		}
		return strings.TrimSuffix(strings.TrimPrefix(s, `<|"|>`), `<|"|>`), nil
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return []any{}, nil
		}
		items := splitInlineTopLevel(inner, ',')
		out := make([]any, 0, len(items))
		for _, item := range items {
			value, err := parseInlineToolValue(item)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	}
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return parseInlineToolArgs(s[1 : len(s)-1])
	}
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		var out string
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	switch s {
	case "true", "True":
		return true, nil
	case "false", "False":
		return false, nil
	case "null", "None":
		return nil, nil
	}
	var decoded any
	if json.Unmarshal([]byte(s), &decoded) == nil {
		return decoded, nil
	}
	return strings.Trim(s, `"'`), nil
}

func splitInlineTopLevel(src string, sep rune) []string {
	var out []string
	start := 0
	depthSquare := 0
	depthCurly := 0
	inString := false
	for i := 0; i < len(src); {
		if strings.HasPrefix(src[i:], `<|"|>`) || strings.HasPrefix(src[i:], `<|\"|>`) {
			if strings.HasPrefix(src[i:], `<|\"|>`) {
				i += len(`<|\"|>`)
			} else {
				i += len(`<|"|>`)
			}
			inString = !inString
			continue
		}
		r := rune(src[i])
		if !inString {
			switch r {
			case '[':
				depthSquare++
			case ']':
				if depthSquare > 0 {
					depthSquare--
				}
			case '{':
				depthCurly++
			case '}':
				if depthCurly > 0 {
					depthCurly--
				}
			default:
				if r == sep && depthSquare == 0 && depthCurly == 0 {
					out = append(out, src[start:i])
					start = i + 1
				}
			}
		}
		i++
	}
	out = append(out, src[start:])
	return out
}

func indexInlineTopLevel(src string, sep rune) int {
	depthSquare := 0
	depthCurly := 0
	inString := false
	for i := 0; i < len(src); {
		if strings.HasPrefix(src[i:], `<|"|>`) || strings.HasPrefix(src[i:], `<|\"|>`) {
			if strings.HasPrefix(src[i:], `<|\"|>`) {
				i += len(`<|\"|>`)
			} else {
				i += len(`<|"|>`)
			}
			inString = !inString
			continue
		}
		r := rune(src[i])
		if !inString {
			switch r {
			case '[':
				depthSquare++
			case ']':
				if depthSquare > 0 {
					depthSquare--
				}
			case '{':
				depthCurly++
			case '}':
				if depthCurly > 0 {
					depthCurly--
				}
			default:
				if r == sep && depthSquare == 0 && depthCurly == 0 {
					return i
				}
			}
		}
		i++
	}
	return -1
}

func normalizeInlineSentinel(s string) string {
	return strings.ReplaceAll(s, `<|\"|>`, `<|"|>`)
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
