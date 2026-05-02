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
	case CategoryContextOverflow:
		return ActionRetryHalfContext

	case CategoryLLMUnreachable:
		return ActionRestartInference

	case CategoryInferenceCrash:
		return ActionRestartInference

	case CategoryToolTimeout:
		return ActionRetryLonger

	case CategoryModelUnavailable:
		return ActionNone

	case CategoryCircleDetected,
		CategoryMaxTurns,
		CategoryBudgetExceeded,
		CategoryManifestError,
		CategoryUnknown:
		return ActionNone

	default:
		return ActionNone
	}
}
