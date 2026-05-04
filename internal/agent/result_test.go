/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"testing"

	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/stretchr/testify/require"
)

func TestSuccessfulEnd_onlyCompleted(t *testing.T) {
	t.Parallel()

	reasons := []EndReason{
		EndCompleted,
		EndBudgetExceeded,
		EndTimeout,
		EndMaxTurns,
		EndMaxToolCalls,
		EndEmptyResponse,
		EndLLMUnreachable,
		EndCircleDetected,
	}

	for _, reason := range reasons {
		got := SuccessfulEnd(reason)
		require.Equal(t, reason == EndCompleted, got, "reason %s", reason)
	}
}

func TestOutcomeForEndReason(t *testing.T) {
	t.Parallel()

	require.Equal(t, scoring.OutcomePassed, OutcomeForEndReason(EndCompleted))
	require.Equal(t, scoring.OutcomeTimeout, OutcomeForEndReason(EndTimeout))
	require.Equal(t, scoring.OutcomeFailed, OutcomeForEndReason(EndMaxTurns))
	require.Equal(t, scoring.OutcomeFailed, OutcomeForEndReason(EndBudgetExceeded))
	require.Equal(t, scoring.OutcomeFailed, OutcomeForEndReason(EndCircleDetected))
}

func TestNonSuccessError(t *testing.T) {
	t.Parallel()

	require.NoError(t, NonSuccessError(LoopResult{EndReason: EndCompleted}))
	require.ErrorContains(t, NonSuccessError(LoopResult{EndReason: EndMaxTurns}), "max_turns")
}
