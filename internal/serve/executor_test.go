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

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/stretchr/testify/assert"
)

func TestRoleDefaultTrustLevel(t *testing.T) {
	tests := []struct {
		name string
		role bundle.RoleConfig
		want trust.Level
	}{
		{
			name: "missing trust defaults to observer",
			role: bundle.RoleConfig{},
			want: trust.LevelObserver,
		},
		{
			name: "contributor trust is honored",
			role: bundle.RoleConfig{TrustLevel: "contributor"},
			want: trust.LevelContributor,
		},
		{
			name: "operator alias maps to contributor",
			role: bundle.RoleConfig{TrustLevel: "operator"},
			want: trust.LevelContributor,
		},
		{
			name: "autonomous trust is honored",
			role: bundle.RoleConfig{TrustLevel: "autonomous"},
			want: trust.LevelAutonomous,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, roleDefaultTrustLevel(tt.role))
		})
	}
}
