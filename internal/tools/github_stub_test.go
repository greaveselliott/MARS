package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubStubs_notImplemented(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	root, err := NewRoot(t.TempDir())
	require.NoError(t, err)
	h, _, ok := reg.Lookup("github_pr_create")
	require.True(t, ok)
	_, err = h(context.Background(), root, []byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "M4")
}
