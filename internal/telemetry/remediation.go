/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

// RemediationAction is the auto-fix strategy for a failure category.
type RemediationAction string

const (
	ActionNone             RemediationAction = "none"
	ActionRetryHalfContext RemediationAction = "retry_half_context"
	ActionRestartInference RemediationAction = "restart_inference"
	ActionRetryLonger      RemediationAction = "retry_longer_timeout"
	ActionRetryPlain       RemediationAction = "retry"
)

// Remediate returns the recommended auto-fix action for a failure category.
func Remediate(cat FailureCategory) RemediationAction {
	switch cat {
	case CategoryLLMUnreachable:
		return ActionRestartInference

	case CategoryInferenceCrash:
		return ActionRestartInference

	case CategoryToolTimeout:
		return ActionRetryLonger

	case CategoryModelUnavailable:
		return ActionNone

	case CategoryCircleDetected,
		CategoryContextOverflow,
		CategoryMaxTurns,
		CategoryBudgetExceeded,
		CategoryManifestError,
		CategoryTicketGate,
		CategoryUnknown:
		return ActionNone

	default:
		return ActionNone
	}
}
