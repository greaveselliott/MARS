/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/design-docs/dashboard.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-012-self-improvement-loop.md
*/
package serve

import (
	"testing"

	"github.com/greaveselliott/mars/internal/telemetry"
	"github.com/greaveselliott/mars/internal/tools"
	"github.com/stretchr/testify/require"
)

func TestPolicyBlockLoopTrackerThresholdsIdenticalBlocks(t *testing.T) {
	t.Parallel()
	tracker := &policyBlockLoopTracker{}
	evt := tools.PolicyEvent{
		Stage:    "post",
		ToolName: "job_disposition_record",
		Message:  "successful disposition requires feature contract coverage",
	}

	for i := 1; i < telemetry.PatternThreshold; i++ {
		count, threshold := tracker.record(evt)
		require.Equal(t, i, count)
		require.False(t, threshold)
	}

	count, threshold := tracker.record(evt)
	require.Equal(t, telemetry.PatternThreshold, count)
	require.True(t, threshold)

	count, threshold = tracker.record(evt)
	require.Equal(t, telemetry.PatternThreshold+1, count)
	require.False(t, threshold)
}

func TestPolicyBlockLoopTrackerNormalizesEquivalentMessages(t *testing.T) {
	t.Parallel()
	tracker := &policyBlockLoopTracker{}

	count, threshold := tracker.record(tools.PolicyEvent{
		Stage:    "POST",
		ToolName: " job_disposition_record ",
		Message:  "Successful disposition requires   feature contract coverage",
	})
	require.Equal(t, 1, count)
	require.False(t, threshold)

	count, threshold = tracker.record(tools.PolicyEvent{
		Stage:    "post",
		ToolName: "job_disposition_record",
		Message:  "successful disposition requires feature contract coverage",
	})
	require.Equal(t, 2, count)
	require.False(t, threshold)
}

func TestPolicyBlockLoopTrackerSeparatesDifferentTools(t *testing.T) {
	t.Parallel()
	tracker := &policyBlockLoopTracker{}

	count, threshold := tracker.record(tools.PolicyEvent{
		Stage:    "post",
		ToolName: "job_disposition_record",
		Message:  "blocked",
	})
	require.Equal(t, 1, count)
	require.False(t, threshold)

	count, threshold = tracker.record(tools.PolicyEvent{
		Stage:    "post",
		ToolName: "shell_exec",
		Message:  "blocked",
	})
	require.Equal(t, 1, count)
	require.False(t, threshold)
}
