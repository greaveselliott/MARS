/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func decodeStringSliceArg(raw json.RawMessage, field string) ([]string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	var out []string
	if err := json.Unmarshal(trimmed, &out); err == nil {
		return out, nil
	}

	var encoded string
	if err := json.Unmarshal(trimmed, &encoded); err == nil {
		return parseStringSliceText(encoded, field)
	}

	if bytes.HasPrefix(trimmed, []byte("[")) {
		return parseStringSliceText(string(trimmed), field)
	}
	return nil, fmt.Errorf("%s must be an array of strings or a string containing one", field)
}

func parseStringSliceText(text, field string) ([]string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, nil
	}

	var out []string
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &out); err == nil {
			return out, nil
		}
		parsed, ok := parsePythonStyleStringList(trimmed)
		if !ok {
			return nil, fmt.Errorf("%s string must contain a JSON array or quoted string list", field)
		}
		return parsed, nil
	}
	return []string{trimmed}, nil
}

func parsePythonStyleStringList(text string) ([]string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return nil, false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
	if inner == "" {
		return []string{}, true
	}

	var out []string
	for i := 0; i < len(inner); {
		for i < len(inner) && isArgListSpace(inner[i]) {
			i++
		}
		if i >= len(inner) {
			break
		}
		quote := inner[i]
		if quote != '\'' && quote != '"' {
			return nil, false
		}
		i++
		var b strings.Builder
		closed := false
		for i < len(inner) {
			ch := inner[i]
			if ch == '\\' && i+1 < len(inner) {
				b.WriteByte(inner[i+1])
				i += 2
				continue
			}
			if ch == quote {
				i++
				closed = true
				break
			}
			b.WriteByte(ch)
			i++
		}
		if !closed {
			return nil, false
		}
		out = append(out, b.String())

		for i < len(inner) && isArgListSpace(inner[i]) {
			i++
		}
		if i >= len(inner) {
			break
		}
		if inner[i] != ',' {
			return nil, false
		}
		i++
	}
	return out, true
}

func isArgListSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
