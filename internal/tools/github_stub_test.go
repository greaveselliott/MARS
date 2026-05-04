/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultRegistry_omitsPRWorkflowTools(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	_, _, ok := reg.Lookup("github_pr_create")
	require.False(t, ok)
	_, _, ok = reg.Lookup("github_pr_comment")
	require.False(t, ok)
}
