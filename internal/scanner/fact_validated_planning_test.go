/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/mirrored-harness-and-context-glossary.md
- docs/features/F-004-target-harness-lifecycle.md
*/
package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitMirrorsFactValidatedPlanningAndLocalEnvIgnore(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Init(dir, false))

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	require.NoError(t, err)
	require.Contains(t, string(agents), "Fact-Validated Planning")
	require.Contains(t, string(agents), "inspect discoverable repo and system facts directly")

	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	require.Contains(t, string(gitignore), ".harness/.env.local")
}
