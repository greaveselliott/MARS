/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-013-board-driven-integrations.md
*/
package serve

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/greaveselliott/mars/internal/integrations"
	"github.com/greaveselliott/mars/internal/tools"
	"github.com/stretchr/testify/require"
)

func TestEffectiveToolAllowlist_noConfigUnchanged(t *testing.T) {
	t.Parallel()
	reg := tools.NewRegistry()
	base := []string{"file_read", "git_status"}
	got := effectiveToolAllowlist(base, integrations.Defaults(), reg)
	require.Equal(t, base, got)
	require.NotSame(t, &base[0], &got[0])
}

func TestEffectiveToolAllowlist_appendsOnlyRegisteredGatedTools(t *testing.T) {
	t.Parallel()
	reg := tools.NewRegistry()
	require.NoError(t, reg.Register("figma_fetch", "fetch figma", json.RawMessage(`{"type":"object"}`), noopToolHandler))

	cfg := integrations.Defaults()
	cfg.FlowProfile = integrations.FlowProfileBoardDriven
	cfg.DesignSources.Figma.Enabled = true
	cfg.Delivery.Mode = integrations.DeliveryModePullRequest

	got := effectiveToolAllowlist([]string{"file_read"}, cfg, reg)
	require.Equal(t, []string{"file_read", "figma_fetch"}, got)
}

func noopToolHandler(context.Context, tools.Root, json.RawMessage) (tools.ToolResult, error) {
	return tools.ToolResult{}, nil
}
