package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSemVer(t *testing.T) {
	t.Parallel()
	got, err := ParseSemVer("v1.2.3")
	require.NoError(t, err)
	require.Equal(t, SemVer{Major: 1, Minor: 2, Patch: 3}, got)

	_, err = ParseSemVer("1.2")
	require.Error(t, err)
}

func TestSemVerBump(t *testing.T) {
	t.Parallel()
	base := SemVer{Major: 1, Minor: 2, Patch: 3}
	require.Equal(t, "2.0.0", base.Bump(BumpMajor).String())
	require.Equal(t, "1.3.0", base.Bump(BumpMinor).String())
	require.Equal(t, "1.2.4", base.Bump(BumpPatch).String())
}
