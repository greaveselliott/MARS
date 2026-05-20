/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileWriteSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "path": { "type": "string", "description": "Path relative to repository root" },
    "content": { "type": "string", "description": "File contents to write" }
  },
  "required": ["path", "content"]
}`

type fileWriteArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func registerFileWrite(r *Registry) error {
	return r.Register("file_write", "Create or overwrite a file under the repository root.", json.RawMessage(fileWriteSchema), handleFileWrite)
}

func handleFileWrite(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	args, err := decodeFileWriteArgs(raw)
	if err != nil {
		return ToolResult{}, fmt.Errorf("file_write: parse arguments: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return ToolResult{}, fmt.Errorf("file_write: field path is required")
	}
	path, err := root.ResolvePath(args.Path)
	if err != nil {
		return ToolResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ToolResult{}, fmt.Errorf("file_write: mkdir %q: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return ToolResult{}, fmt.Errorf("file_write: write %q: %w", args.Path, err)
	}
	return ToolResult{Output: fmt.Sprintf("wrote %d bytes to %s", len(args.Content), args.Path)}, nil
}

func decodeFileWriteArgs(raw json.RawMessage) (fileWriteArgs, error) {
	var args fileWriteArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return fileWriteArgs{}, err
	}
	args = normalizeFileWriteParameterMarker(args)
	return args, nil
}

func normalizeFileWriteParameterMarker(args fileWriteArgs) fileWriteArgs {
	if strings.TrimSpace(args.Content) != "" {
		return args
	}
	const marker = "\n<parameter=content>\n"
	path, content, ok := strings.Cut(args.Path, marker)
	if !ok {
		return args
	}
	args.Path = strings.TrimSpace(path)
	args.Content = content
	return args
}
