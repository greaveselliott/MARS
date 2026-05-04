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
	"path/filepath"
	"strings"
)

const fileSearchSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "pattern": { "type": "string", "description": "Glob pattern relative to repository root (e.g. **/*.go)" }
  },
  "required": ["pattern"]
}`

type fileSearchArgs struct {
	Pattern string `json:"pattern"`
}

func registerFileSearch(r *Registry) error {
	return r.Register("file_search", "List files matching a glob pattern under the repository root.", json.RawMessage(fileSearchSchema), handleFileSearch)
}

func handleFileSearch(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args fileSearchArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("file_search: parse arguments: %w", err)
	}
	pat := strings.TrimSpace(args.Pattern)
	if pat == "" {
		return ToolResult{}, fmt.Errorf("file_search: field pattern is required")
	}
	if filepath.IsAbs(pat) {
		return ToolResult{}, fmt.Errorf("file_search: pattern must be relative to repository root")
	}
	base := root.Abs()
	matches, err := filepath.Glob(filepath.Join(base, pat))
	if err != nil {
		return ToolResult{}, fmt.Errorf("file_search: glob %q: %w", pat, err)
	}
	var rels []string
	for _, m := range matches {
		rel, err := filepath.Rel(base, m)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rels = append(rels, filepath.ToSlash(rel))
	}
	if len(rels) == 0 {
		return ToolResult{Output: "(no matches)"}, nil
	}
	return ToolResult{Output: strings.Join(rels, "\n")}, nil
}
