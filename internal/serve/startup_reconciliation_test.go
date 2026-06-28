/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartupReconciliationSummaryIncludesDefaultActionAndEvidence(t *testing.T) {
	t.Parallel()

	summary := StartupReconciliation{
		Role:     "engineer",
		JobID:    "job-123",
		Evidence: []string{"recovered stale job", "active engineer job already queued"},
	}.Summary()

	require.Contains(t, summary, "startup_action=refused_ambiguous_state")
	require.Contains(t, summary, "role=engineer")
	require.Contains(t, summary, "job=job-123")
	require.Contains(t, summary, "evidence=recovered stale job; active engineer job already queued")
}
