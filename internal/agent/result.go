/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"fmt"

	"github.com/greaveselliott/mars-harness/internal/scoring"
)

// SuccessfulEnd reports whether a loop terminal reason represents completed work.
func SuccessfulEnd(reason EndReason) bool {
	return reason == EndCompleted
}

// OutcomeForEndReason maps loop termination to trunk-native scoring outcomes.
func OutcomeForEndReason(reason EndReason) scoring.OutcomeType {
	switch reason {
	case EndCompleted:
		return scoring.OutcomePassed
	case EndTimeout:
		return scoring.OutcomeTimeout
	case EndBudgetExceeded, EndMaxTurns, EndMaxToolCalls, EndEmptyResponse, EndLLMUnreachable, EndCircleDetected:
		return scoring.OutcomeFailed
	default:
		return scoring.OutcomeFailed
	}
}

// NonSuccessError returns nil only for a completed run.
func NonSuccessError(res LoopResult) error {
	if SuccessfulEnd(res.EndReason) {
		return nil
	}
	if res.Err != nil {
		return fmt.Errorf("agent ended with %s: %w", res.EndReason, res.Err)
	}
	return fmt.Errorf("agent ended with %s", res.EndReason)
}
