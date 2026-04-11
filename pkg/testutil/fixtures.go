package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// FixtureBytes reads a file from testdata/ (or any path) and fails the test on error.
func FixtureBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err, "reading fixture %s", path)
	return b
}

// FixtureString is a convenience wrapper that returns file contents as a string.
func FixtureString(t *testing.T, path string) string {
	return string(FixtureBytes(t, path))
}
