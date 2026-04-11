package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_Definitions_sortedAndAllowlisted(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	defs, err := reg.Definitions([]string{"file_read", "grep"})
	require.NoError(t, err)
	require.Len(t, defs, 2)
	require.Equal(t, "function", defs[0].Type)
	require.Equal(t, "file_read", defs[0].Function.Name)
	require.Equal(t, "grep", defs[1].Function.Name)
}

func TestRegistry_Definitions_unknownTool(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	_, err = reg.Definitions([]string{"file_read", "does_not_exist"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown tool")
	require.Contains(t, err.Error(), "does_not_exist")
}

func TestRegistry_Definitions_emptyAllowlist(t *testing.T) {
	t.Parallel()
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	_, err = reg.Definitions(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowlist")
}
