/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import (
	"strings"
	"time"
)

// FailureCategory is a machine-readable classification of why a job failed.
type FailureCategory string

const (
	CategoryContextOverflow  FailureCategory = "context_overflow"
	CategoryLLMUnreachable   FailureCategory = "llm_unreachable"
	CategoryInferenceCrash   FailureCategory = "inference_crash"
	CategoryModelUnavailable FailureCategory = "model_unavailable"
	CategoryToolTimeout      FailureCategory = "tool_timeout"
	CategoryCircleDetected   FailureCategory = "circle_detected"
	CategoryMaxTurns         FailureCategory = "max_turns"
	CategoryBudgetExceeded   FailureCategory = "budget_exceeded"
	CategoryManifestError    FailureCategory = "manifest_error"
	CategoryTicketGate       FailureCategory = "ticket_gate"
	CategoryDispatchProtocol FailureCategory = "dispatch_protocol"
	CategoryGuardrailBlock   FailureCategory = "guardrail_block"
	CategoryGuardrailLoop    FailureCategory = "guardrail_loop"
	CategoryWorkspaceHygiene FailureCategory = "workspace_hygiene"
	CategoryHumanFollowup    FailureCategory = "human_followup"
	CategoryRevertedCommit   FailureCategory = "reverted_commit"
	CategoryStaleTicket      FailureCategory = "stale_in_progress_ticket"
	CategoryManualStop       FailureCategory = "manual_stop"
	CategoryUnknown          FailureCategory = "unknown"
)

// Event is a single telemetry observation from the pipeline.
type Event struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	JobID     string          `json:"job_id"`
	RepoID    string          `json:"repo_id"`
	Role      string          `json:"role"`
	Category  FailureCategory `json:"category"`
	Message   string          `json:"message"`
	Remedied  bool            `json:"remedied"`
	Action    string          `json:"action,omitempty"`
}

// Classify inspects an error message string and returns the most specific
// FailureCategory. The classifier uses substring matching because the
// system's errors are unstructured fmt.Errorf strings.
func Classify(errMsg string) FailureCategory {
	lower := strings.ToLower(errMsg)

	switch {
	case strings.Contains(lower, "guardrail_loop"):
		return CategoryGuardrailLoop
	case strings.Contains(lower, "repeated policy block"):
		return CategoryGuardrailLoop
	case strings.Contains(lower, "guardrails: blocked"):
		return CategoryGuardrailBlock
	case strings.Contains(lower, "guardrail block"):
		return CategoryGuardrailBlock
	case strings.Contains(lower, "tool policy blocked"):
		return CategoryGuardrailBlock
	case strings.Contains(lower, "blast radius exceeded"):
		return CategoryGuardrailBlock
	case strings.Contains(lower, "policy: secret scanner blocked"):
		return CategoryGuardrailBlock
	case strings.Contains(lower, "policy: trust level observer cannot run mutating tool"):
		return CategoryGuardrailBlock
	case strings.Contains(lower, "policy: strict trunk only allows"):
		return CategoryGuardrailBlock

	// Tool timeout must match before context overflow because "context deadline exceeded"
	// contains both "context" and "timeout".
	case strings.Contains(lower, "timed out"):
		return CategoryToolTimeout
	case strings.Contains(lower, "timeout"):
		return CategoryToolTimeout
	case strings.Contains(lower, "context deadline exceeded"):
		return CategoryToolTimeout

	case strings.Contains(lower, "exceed") && strings.Contains(lower, "context size"):
		return CategoryContextOverflow
	case strings.Contains(lower, "exceed_context_size"):
		return CategoryContextOverflow
	case strings.Contains(lower, "context_overflow"):
		return CategoryContextOverflow

	case strings.Contains(lower, "llm_unreachable"):
		return CategoryLLMUnreachable
	case strings.Contains(lower, "unexpected status 5"):
		return CategoryLLMUnreachable
	case strings.Contains(lower, "connection refused"):
		return CategoryLLMUnreachable

	case strings.Contains(lower, "inference") && (strings.Contains(lower, "crash") || strings.Contains(lower, "health check")):
		return CategoryInferenceCrash
	case strings.Contains(lower, "inference") && strings.Contains(lower, "oom"):
		return CategoryInferenceCrash
	case strings.Contains(lower, "signal: killed"):
		return CategoryInferenceCrash
	case strings.Contains(lower, "no local model"):
		return CategoryModelUnavailable
	case strings.Contains(lower, "local model") && strings.Contains(lower, "missing"):
		return CategoryModelUnavailable
	case strings.Contains(lower, "remote fallback configured") && strings.Contains(lower, "model"):
		return CategoryModelUnavailable

	case strings.Contains(lower, "circle_detected"):
		return CategoryCircleDetected

	case strings.Contains(lower, "max_turns"):
		return CategoryMaxTurns

	case strings.Contains(lower, "budget_exceeded"):
		return CategoryBudgetExceeded

	case strings.Contains(lower, "manifest") || strings.Contains(lower, "bundle"):
		return CategoryManifestError

	case strings.Contains(lower, "ticket gate"):
		return CategoryTicketGate
	case strings.Contains(lower, "ended without completing any existing in-progress ticket"):
		return CategoryTicketGate
	case strings.Contains(lower, "cannot hand off") && strings.Contains(lower, "docs/tickets/in-progress"):
		return CategoryTicketGate

	case strings.Contains(lower, "dispatch mode requires") && strings.Contains(lower, "job_disposition_record"):
		return CategoryDispatchProtocol
	case strings.Contains(lower, "suggested route rejected"):
		return CategoryDispatchProtocol

	case strings.Contains(lower, "workspace_hygiene_blocked"):
		return CategoryWorkspaceHygiene
	case strings.Contains(lower, "workspace hygiene") && strings.Contains(lower, "blocked"):
		return CategoryWorkspaceHygiene
	case strings.Contains(lower, "dependency_sync") && strings.Contains(lower, "workspace hygiene"):
		return CategoryWorkspaceHygiene

	case strings.Contains(lower, "human_followup"):
		return CategoryHumanFollowup
	case strings.Contains(lower, "human follow-up"):
		return CategoryHumanFollowup
	case strings.Contains(lower, "human followup"):
		return CategoryHumanFollowup

	case strings.Contains(lower, "reverted_commit"):
		return CategoryRevertedCommit
	case strings.Contains(lower, "reverted agent commit"):
		return CategoryRevertedCommit
	case strings.Contains(lower, "agent commit reverted"):
		return CategoryRevertedCommit
	case strings.Contains(lower, "revert") && strings.Contains(lower, "agent"):
		return CategoryRevertedCommit

	case strings.Contains(lower, "stale_in_progress_ticket"):
		return CategoryStaleTicket
	case strings.Contains(lower, "stale in-progress ticket"):
		return CategoryStaleTicket
	case strings.Contains(lower, "stale ticket"):
		return CategoryStaleTicket

	case strings.Contains(lower, "manual_stop"):
		return CategoryManualStop
	case strings.Contains(lower, "manual stop"):
		return CategoryManualStop
	case strings.Contains(lower, "operator stopped"):
		return CategoryManualStop
	case strings.Contains(lower, "interrupted by operator"):
		return CategoryManualStop
	case strings.Contains(lower, "context canceled") || strings.Contains(lower, "context cancelled"):
		return CategoryManualStop

	default:
		return CategoryUnknown
	}
}

// Retryable returns true if the failure category is worth retrying
// (possibly with adjusted parameters).
func (c FailureCategory) Retryable() bool {
	switch c {
	case CategoryLLMUnreachable,
		CategoryInferenceCrash,
		CategoryToolTimeout:
		return true
	default:
		return false
	}
}
