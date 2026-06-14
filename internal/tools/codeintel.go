/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/codeintel"
)

const codeIndexSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "full": { "type": "boolean", "description": "Force a full reindex instead of incremental refresh" },
    "max_files": { "type": "integer", "description": "Maximum files to inspect; default 20000" }
  }
}`

const codeSearchSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "query": { "type": "string", "description": "Keyword search across indexed symbols, signatures, paths, and docs/config text" },
    "kind": { "type": "string", "description": "Optional symbol kind filter such as function, method, class, heading, config" },
    "language": { "type": "string", "description": "Optional language filter such as go, typescript, javascript, markdown" },
    "path": { "type": "string", "description": "Optional path substring filter" },
    "limit": { "type": "integer", "description": "Maximum results; default 20, max 100" }
  },
  "required": ["query"]
}`

const codeSnippetSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "symbol": { "type": "string", "description": "Indexed symbol name or qualified_name from code_search" }
  },
  "required": ["symbol"]
}`

const codeTraceSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "symbol": { "type": "string", "description": "Indexed symbol name or qualified_name from code_search" },
    "direction": { "type": "string", "enum": ["inbound", "outbound", "both"], "description": "Trace direction; default both" }
  },
  "required": ["symbol"]
}`

const codeImpactSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "paths": { "type": "array", "items": { "type": "string" }, "description": "Optional changed paths relative to repo root. If omitted, git status is used." },
    "base_ref": { "type": "string", "description": "Optional git ref for diff mode, e.g. origin/main" }
  }
}`

type codeIndexArgs struct {
	Full     bool `json:"full"`
	MaxFiles int  `json:"max_files"`
}

type codeSearchArgs struct {
	Query    string `json:"query"`
	Kind     string `json:"kind"`
	Language string `json:"language"`
	Path     string `json:"path"`
	Limit    int    `json:"limit"`
}

type codeSymbolArgs struct {
	Symbol string `json:"symbol"`
}

type codeTraceArgs struct {
	Symbol    string `json:"symbol"`
	Direction string `json:"direction"`
}

type rawCodeImpactArgs struct {
	Paths   json.RawMessage `json:"paths"`
	BaseRef string          `json:"base_ref"`
}

func registerCodeIntelTools(r *Registry) error {
	if err := r.Register("code_index", "Refresh the Mars-native code intelligence index for this repo. Writes only Mars DB state, never repo files.", json.RawMessage(codeIndexSchema), handleCodeIndex); err != nil {
		return err
	}
	if err := r.Register("code_search", "Search the Mars-native code intelligence graph before broad grep or bulk file reads.", json.RawMessage(codeSearchSchema), handleCodeSearch); err != nil {
		return err
	}
	if err := r.Register("code_snippet", "Read exact bounded source for an indexed symbol from the Mars code intelligence graph.", json.RawMessage(codeSnippetSchema), handleCodeSnippet); err != nil {
		return err
	}
	if err := r.Register("code_trace", "Trace known import and call relationships for an indexed symbol.", json.RawMessage(codeTraceSchema), handleCodeTrace); err != nil {
		return err
	}
	return r.Register("code_impact", "Map changed files to indexed symbols, likely tests, MarsDocSync docs, feature scenarios, and tickets.", json.RawMessage(codeImpactSchema), handleCodeImpact)
}

func handleCodeIndex(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args codeIndexArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("code_index: parse arguments: %w", err)
	}
	store, err := codeintel.Open(root.Abs(), root.DBPath())
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	res, err := store.Index(ctx, codeintel.IndexOptions{Full: args.Full, MaxFiles: args.MaxFiles})
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: codeintel.Marshal(res)}, nil
}

func handleCodeSearch(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args codeSearchArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("code_search: parse arguments: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return ToolResult{}, fmt.Errorf("code_search: field query is required")
	}
	store, err := codeintel.Open(root.Abs(), root.DBPath())
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	res, err := store.Search(ctx, codeintel.SearchOptions{Query: args.Query, Kind: args.Kind, Language: args.Language, Path: args.Path, Limit: args.Limit})
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: codeintel.Marshal(res)}, nil
}

func handleCodeSnippet(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args codeSymbolArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("code_snippet: parse arguments: %w", err)
	}
	store, err := codeintel.Open(root.Abs(), root.DBPath())
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	res, err := store.Snippet(ctx, args.Symbol)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: codeintel.Marshal(res)}, nil
}

func handleCodeTrace(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args codeTraceArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("code_trace: parse arguments: %w", err)
	}
	store, err := codeintel.Open(root.Abs(), root.DBPath())
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	res, err := store.Trace(ctx, args.Symbol, args.Direction)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: codeintel.Marshal(res)}, nil
}

func handleCodeImpact(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var rawArgs rawCodeImpactArgs
	if err := json.Unmarshal(raw, &rawArgs); err != nil {
		return ToolResult{}, fmt.Errorf("code_impact: parse arguments: %w", err)
	}
	paths, err := decodeStringSliceArg(rawArgs.Paths, "code_impact.paths")
	if err != nil {
		return ToolResult{}, err
	}
	store, err := codeintel.Open(root.Abs(), root.DBPath())
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	res, err := store.Impact(ctx, paths, rawArgs.BaseRef)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: codeintel.Marshal(res)}, nil
}
