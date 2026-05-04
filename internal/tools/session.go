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

	"github.com/greaveselliott/mars-harness/internal/guardrails"
	"github.com/greaveselliott/mars-harness/internal/safety"
)

type sessionKey struct{}

// PolicyEvent describes a trust, guardrail, or repository policy block during
// tool execution. Callers can record it as telemetry without coupling tools to
// the telemetry package.
type PolicyEvent struct {
	Stage    string
	ToolName string
	Message  string
}

// Session carries job-specific policy data through tool execution without making
// the registry process-global stateful.
type Session struct {
	Role                string
	JobID               string
	RepoID              string
	TrustLevel          string
	BaselineCommit      string
	Guardrails          *guardrails.Engine
	SafetyLimits        safety.Limits
	ToolCounts          map[string]int
	PolicyRecorder      func(PolicyEvent)
	DispositionRecorder func(context.Context, json.RawMessage) error
}

// WithSession stores a tool execution session on ctx.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// SessionFromContext returns the current tool execution session, if any.
func SessionFromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(Session)
	return s, ok
}
