/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-017-open-source-publication.md
*/
package executionprofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdmit(t *testing.T) {
	t.Parallel()

	profile, err := Admit("run", "", false)
	require.NoError(t, err)
	require.Equal(t, Observer, profile)

	_, err = Admit("run", "host", false)
	require.ErrorContains(t, err, "--acknowledge-host-execution")
	require.ErrorContains(t, err, "is not containment")

	profile, err = Admit("run", "host", true)
	require.NoError(t, err)
	require.Equal(t, Host, profile)

	_, err = Admit("run", "isolated", true)
	require.ErrorContains(t, err, "no enforceable isolation adapter")

	_, err = Admit("run", "unknown", false)
	require.ErrorContains(t, err, "choose observer, host, or isolated")
}

func TestRequireTargetMutation(t *testing.T) {
	t.Parallel()
	require.NoError(t, Host.RequireTargetMutation("write target files"))
	require.ErrorContains(t, Observer.RequireTargetMutation("write target files"), "blocks write target files")
}
