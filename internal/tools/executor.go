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
	"log/slog"
	"strings"
	"time"
)

// Executor dispatches tool calls against a registry with allowlist enforcement.
type Executor struct {
	Registry   *Registry
	MaxOutput  int // defaults to DefaultMaxToolOutputBytes when zero
	DefaultTTL time.Duration
	Session    *Session

	// StopAfterTool lets callers make specific tool calls terminal for an agent
	// loop. It is checked after a successful tool result has been traced.
	StopAfterTool func() bool
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
	if len(allowlist) == 0 {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: no tools are allowed for this role; configure an explicit tools list in .harness/manifest.yaml")
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

	if e.Session != nil {
		if e.Session.ToolCounts == nil {
			e.Session.ToolCounts = make(map[string]int)
		}
		runCtx = WithSession(runCtx, *e.Session)
	}
	if err := preToolPolicy(runCtx, root, name, raw); err != nil {
		recordPolicyEvent(runCtx, "pre", name, err)
		return ToolResult{Duration: time.Since(start)}, err
	}
	res, err := executeHandlerWithTimeout(runCtx, start, ttl, name, root, raw, h)
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

type handlerResult struct {
	result ToolResult
	err    error
}

func executeHandlerWithTimeout(ctx context.Context, start time.Time, ttl time.Duration, name string, root Root, raw json.RawMessage, h Handler) (ToolResult, error) {
	done := make(chan handlerResult, 1)
	slog.Debug("tools: executing tool", "tool", name, "ttl", ttl)
	go func() {
		res, err := h(ctx, root, raw)
		res.Duration = time.Since(start)
		if err == nil {
			if postErr := postToolPolicy(ctx, root, name, raw); postErr != nil {
				recordPolicyEvent(ctx, "post", name, postErr)
				err = postErr
			}
		}
		done <- handlerResult{result: res, err: err}
	}()

	select {
	case out := <-done:
		slog.Debug("tools: tool finished", "tool", name, "duration", out.result.Duration, "err", out.err != nil)
		return out.result, out.err
	case <-ctx.Done():
		duration := time.Since(start)
		slog.Warn("tools: tool timed out", "tool", name, "duration", duration, "ttl", ttl, "err", ctx.Err())
		return ToolResult{
			Duration: duration,
			ExitCode: -1,
		}, fmt.Errorf("tools: tool %q timed out after %s; the harness stopped waiting so the agent can record a blocker instead of hanging", name, ttl.Round(time.Second))
	}
}

func recordPolicyEvent(ctx context.Context, stage, toolName string, err error) {
	if err == nil {
		return
	}
	session, ok := SessionFromContext(ctx)
	if !ok || session.PolicyRecorder == nil {
		return
	}
	session.PolicyRecorder(PolicyEvent{
		Stage:    stage,
		ToolName: toolName,
		Message:  err.Error(),
	})
}
