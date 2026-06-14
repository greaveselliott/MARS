/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

// EndReason is a stable machine-readable loop termination code.
type EndReason string

const (
	EndCompleted      EndReason = "completed"
	EndBudgetExceeded EndReason = "budget_exceeded"
	EndTimeout        EndReason = "timeout"
	EndMaxTurns       EndReason = "max_turns"
	EndMaxToolCalls   EndReason = "max_tool_calls"
	EndEmptyResponse  EndReason = "empty_response"
	EndLLMUnreachable EndReason = "llm_unreachable"
	EndCircleDetected EndReason = "circle_detected"
)

// LoopConfig caps resource usage for a single agent job.
type LoopConfig struct {
	Model         string
	MaxTurns      int           // LLM round-trips; default 50 when zero
	TokenBudget   int           // estimated tokens across messages+tools; 0 = unlimited
	WallTime      time.Duration // 0 = unlimited
	MaxToolCalls  int           // total tool executions; 0 = unlimited
	LLMMaxRetries int           // retries per completion on transport errors; default 3 when zero
	ContextSize   int           // context window in tokens; default 32768 when zero
}

// PreflightToolCall is a deterministic tool invocation inserted into the loop
// before the first model turn. It uses the same executor, allowlist, trace, and
// session counters as model-selected tool calls.
type PreflightToolCall struct {
	Name      string
	ArgsJSON  string
	Rationale string
}

// LoopResult is the final transcript and why the loop stopped.
type LoopResult struct {
	Messages         []llm.Message
	EndReason        EndReason
	LLMCalls         int
	ToolInvocations  int
	TokenEstimate    int
	WallTime         time.Duration
	CircleDiagnostic string // set when EndReason is EndCircleDetected
	Err              error  // transport / programming errors only
}

func (c LoopConfig) effectiveMaxTurns() int {
	if c.MaxTurns <= 0 {
		return 50
	}
	return c.MaxTurns
}

func (c LoopConfig) effectiveLLMRetries() int {
	if c.LLMMaxRetries <= 0 {
		return 3
	}
	return c.LLMMaxRetries
}

func (c LoopConfig) effectiveContextSize() int {
	if c.ContextSize <= 0 {
		return 32768
	}
	return c.ContextSize
}
