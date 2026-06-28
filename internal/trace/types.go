/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package trace

import (
	"time"

	"github.com/greaveselliott/mars/internal/llm"
)

// Header is the first JSON line in a trace file (MH-005).
type Header struct {
	Type    string `json:"type"` // always "header"
	TraceID string `json:"trace_id"`
	JobID   string `json:"job_id"`
	Model   string `json:"model,omitempty"`
	Started string `json:"started"` // RFC3339Nano
}

// Turn is one conversation message for the trace (MH-005).
type Turn struct {
	Type          string         `json:"type"` // always "turn"
	Role          string         `json:"role"`
	Content       string         `json:"content,omitempty"`
	ToolCalls     []llm.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID    string         `json:"tool_call_id,omitempty"`
	Timestamp     time.Time      `json:"ts"`
	TokenEstimate int            `json:"token_est,omitempty"`
	Truncated     bool           `json:"truncated,omitempty"`
}

// Summary is persisted alongside JSONL for quick queries (MH-005).
type Summary struct {
	TraceID          string            `json:"trace_id"`
	JobID            string            `json:"job_id"`
	Outcome          string            `json:"outcome"`
	WallMs           int64             `json:"wall_ms"`
	TotalTokens      int               `json:"total_tokens"`
	ToolsCalled      []string          `json:"tools_called"`
	TurnCount        int               `json:"turn_count"`
	ToolCallMessages int               `json:"tool_call_messages"` // tool_calls blocks on assistant turns
	ToolInvocations  int               `json:"tool_invocations"`   // executor runs (from runtime)
	LLMCalls         int               `json:"llm_calls,omitempty"`
	ToolCounts       map[string]int    `json:"tool_counts,omitempty"`
	CodeIntel        *CodeIntelSummary `json:"code_intel,omitempty"`
	Error            string            `json:"error,omitempty"`
	CreatedAt        string            `json:"created_at"` // RFC3339Nano
}

// CodeIntelSummary records whether automatic graph assistance was active for a job.
type CodeIntelSummary struct {
	Mode   string `json:"mode"`
	Source string `json:"source,omitempty"`
}

// Record is a trace row loaded from SQLite.
type Record struct {
	TraceID     string
	JobID       string
	TurnsJSONL  string
	SummaryJSON string
	CreatedAt   time.Time
}
