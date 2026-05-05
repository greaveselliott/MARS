/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/release-versioning.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGithubAuthCheck_reportsMissingAuthWithoutSecrets(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("MARS_HARNESS_GITHUB_TOKEN", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	res, err := handleGithubAuthCheck(context.Background(), root, []byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "private release auth is not configured")
	require.Contains(t, res.Output, `"status": "fail"`)
	require.Contains(t, res.Output, `"auth_source": "none"`)
	require.Contains(t, res.Output, "mars-harness auth github setup")
	require.NotContains(t, res.Output, "Bearer")
	require.NotContains(t, res.Output, "ghs_")
}

func TestDefaultRegistry_includesGithubAuthCheck(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	require.Contains(t, reg.Names(), "github_auth_check")
}
