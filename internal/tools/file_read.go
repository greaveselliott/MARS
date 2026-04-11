package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const fileReadSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "path": { "type": "string", "description": "Path relative to repository root" },
    "start_line": { "type": "integer", "description": "1-based inclusive start line (optional)" },
    "end_line": { "type": "integer", "description": "1-based inclusive end line (optional)" }
  },
  "required": ["path"]
}`

type fileReadArgs struct {
	Path      string `json:"path"`
	StartLine *int   `json:"start_line"`
	EndLine   *int   `json:"end_line"`
}

func registerFileRead(r *Registry) error {
	return r.Register("file_read", "Read a UTF-8 text file under the repository root, optionally by line range.", json.RawMessage(fileReadSchema), handleFileRead)
}

func handleFileRead(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args fileReadArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("file_read: parse arguments: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return ToolResult{}, fmt.Errorf("file_read: field path is required")
	}
	path, err := root.ResolvePath(args.Path)
	if err != nil {
		return ToolResult{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ToolResult{}, fmt.Errorf("file_read: open %q: %w", args.Path, err)
		}
		return ToolResult{}, fmt.Errorf("file_read: open %q: %w", args.Path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, DefaultMaxToolOutputBytes+1))
	if err != nil {
		return ToolResult{}, fmt.Errorf("file_read: read %q: %w", args.Path, err)
	}
	truncated := false
	if len(data) > DefaultMaxToolOutputBytes {
		data = data[:DefaultMaxToolOutputBytes]
		truncated = true
	}
	if IsProbablyBinary(data) {
		return ToolResult{Output: "", IsBinary: true, Truncated: truncated}, nil
	}

	text := string(data)
	if args.StartLine != nil || args.EndLine != nil {
		text, err = sliceLines(text, args.StartLine, args.EndLine)
		if err != nil {
			return ToolResult{}, err
		}
	}
	out, trunc := TruncateUTF8(text, DefaultMaxToolOutputBytes)
	return ToolResult{Output: out, Truncated: truncated || trunc}, nil
}

func sliceLines(text string, start, end *int) (string, error) {
	lines := strings.Split(text, "\n")
	n := len(lines)
	s := 1
	e := n
	if start != nil {
		if *start < 1 {
			return "", fmt.Errorf("file_read: start_line must be >= 1")
		}
		s = *start
	}
	if end != nil {
		if *end < 1 {
			return "", fmt.Errorf("file_read: end_line must be >= 1")
		}
		e = *end
	}
	if s > n {
		return "", fmt.Errorf("file_read: start_line %d is beyond file length (%d lines)", s, n)
	}
	if e < s {
		return "", fmt.Errorf("file_read: end_line %d is before start_line %d", e, s)
	}
	if e > n {
		e = n
	}
	// 1-based inclusive slice
	selected := lines[s-1 : e]
	return strings.Join(selected, "\n"), nil
}
