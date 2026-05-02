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
	// Tool timeout must match before context overflow because "context deadline exceeded"
	// contains both "context" and "timeout".
	case strings.Contains(lower, "timed out"):
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

	default:
		return CategoryUnknown
	}
}

// Retryable returns true if the failure category is worth retrying
// (possibly with adjusted parameters).
func (c FailureCategory) Retryable() bool {
	switch c {
	case CategoryContextOverflow,
		CategoryLLMUnreachable,
		CategoryInferenceCrash,
		CategoryToolTimeout:
		return true
	default:
		return false
	}
}
