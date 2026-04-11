package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Executor dispatches tool calls against a registry with allowlist enforcement.
type Executor struct {
	Registry   *Registry
	MaxOutput  int // defaults to DefaultMaxToolOutputBytes when zero
	DefaultTTL time.Duration
}

// NewExecutor returns an executor backed by reg.
func NewExecutor(reg *Registry) *Executor {
	if reg == nil {
		panic("tools: NewExecutor registry is nil")
	}
	return &Executor{
		Registry:   reg,
		DefaultTTL: 2 * time.Minute,
	}
}

func (e *Executor) maxOut() int {
	if e.MaxOutput > 0 {
		return e.MaxOutput
	}
	return DefaultMaxToolOutputBytes
}

// Execute runs a single tool if it is allowlisted.
func (e *Executor) Execute(ctx context.Context, root Root, allowlist []string, name, argsJSON string) (ToolResult, error) {
	start := time.Now()
	name = strings.TrimSpace(name)
	if name == "" {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: tool name is empty")
	}
	if !Allowlisted(name, allowlist) {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: tool %q is not allowed for this role; allowlist: %s", name, strings.Join(allowlist, ", "))
	}
	h, _, ok := e.Registry.Lookup(name)
	if !ok {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: tool %q is not registered", name)
	}
	ttl := e.DefaultTTL
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, ttl)
	defer cancel()

	raw := json.RawMessage(strings.TrimSpace(argsJSON))
	if len(strings.TrimSpace(argsJSON)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: arguments for %q are not valid JSON", name)
	}

	res, err := h(runCtx, root, raw)
	res.Duration = time.Since(start)
	if err != nil {
		return res, err
	}
	if res.Output != "" || res.Stderr != "" {
		combined := len(res.Output) + len(res.Stderr)
		if combined > e.maxOut() {
			trunc := e.maxOut() / 2
			out, _ := TruncateUTF8(res.Output, trunc)
			errOut, _ := TruncateUTF8(res.Stderr, trunc)
			res.Output = out
			res.Stderr = errOut
			res.Truncated = true
		}
	}
	return res, nil
}
