package tools

import (
	"context"

	"github.com/greaveselliott/mars-harness/internal/guardrails"
	"github.com/greaveselliott/mars-harness/internal/safety"
)

type sessionKey struct{}

// Session carries job-specific policy data through tool execution without making
// the registry process-global stateful.
type Session struct {
	Role           string
	JobID          string
	RepoID         string
	TrustLevel     string
	BaselineCommit string
	Guardrails     *guardrails.Engine
	SafetyLimits   safety.Limits
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
