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
	"slices"
	"sort"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

// Handler executes a tool with parsed JSON arguments under root.
type Handler func(ctx context.Context, root Root, rawArgs json.RawMessage) (ToolResult, error)

// Registry maps tool names to JSON Schema and handlers.
type Registry struct {
	tools map[string]toolEntry
}

type toolEntry struct {
	description string
	schema      json.RawMessage
	handle      Handler
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]toolEntry)}
}

// Register adds or replaces a tool definition.
func (r *Registry) Register(name, description string, parameters json.RawMessage, handle Handler) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("tools: tool name is empty")
	}
	if handle == nil {
		return fmt.Errorf("tools: handler for %q is nil", name)
	}
	if len(parameters) == 0 {
		return fmt.Errorf("tools: parameters schema for %q is empty", name)
	}
	if !json.Valid(parameters) {
		return fmt.Errorf("tools: parameters schema for %q is not valid JSON", name)
	}
	r.tools[name] = toolEntry{
		description: strings.TrimSpace(description),
		schema:      parameters,
		handle:      handle,
	}
	return nil
}

// Names returns all registered tool names sorted lexicographically.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for n := range r.tools {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Definitions returns OpenAI-style tool definitions for allowlisted tools only.
// allowlist order is ignored; output is sorted by tool name for stability.
func (r *Registry) Definitions(allowlist []string) ([]llm.ToolDefinition, error) {
	if len(allowlist) == 0 {
		return nil, fmt.Errorf("tools: allowlist is empty; register at least one tool for this role")
	}
	seen := make(map[string]struct{}, len(allowlist))
	var unknown []string
	for _, name := range allowlist {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := r.tools[name]; !ok {
			unknown = append(unknown, name)
		}
		seen[name] = struct{}{}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("tools: unknown tool(s) in allowlist: %s; registered tools: %s",
			strings.Join(unknown, ", "),
			strings.Join(r.Names(), ", "))
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]llm.ToolDefinition, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
		out = append(out, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionSpec{
				Name:        name,
				Description: t.description,
				Parameters:  json.RawMessage(t.schema),
			},
		})
	}
	return out, nil
}

// Lookup returns the handler and schema for a tool name.
func (r *Registry) Lookup(name string) (Handler, json.RawMessage, bool) {
	t, ok := r.tools[name]
	if !ok {
		return nil, nil, false
	}
	return t.handle, t.schema, true
}

// Allowlisted reports whether name appears in allowlist.
func Allowlisted(name string, allowlist []string) bool {
	return slices.Contains(allowlist, name)
}
