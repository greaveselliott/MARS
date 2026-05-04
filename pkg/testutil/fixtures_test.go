/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package testutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixtureBytes_readsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	WriteFile(t, path, "hello fixture")

	b := FixtureBytes(t, path)
	require.Equal(t, "hello fixture", string(b))
}

func TestFixtureString_readsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	WriteFile(t, path, "string fixture")

	s := FixtureString(t, path)
	require.Equal(t, "string fixture", s)
}
