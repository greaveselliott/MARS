/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDogfoodToolCreationGuardCurrentRepo(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	root, err := NewRoot(filepath.Join(wd, "..", ".."))
	require.NoError(t, err)
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	exec := NewExecutor(reg)
	res, err := exec.Execute(context.Background(), root, []string{"tool_creation_guard"}, "tool_creation_guard", `{"tool_name":"tool_creation_guard"}`)
	require.NoError(t, err)
	require.Contains(t, res.Output, "status: ok")
}
