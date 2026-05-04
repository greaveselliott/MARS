/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/features/F-005-agent-execution-runtime.md
*/
package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadHarnessIgnore_missingFileReturnsNil(t *testing.T) {
	t.Parallel()
	pats, err := LoadHarnessIgnore(t.TempDir())
	require.NoError(t, err)
	require.Nil(t, pats)
}

func TestLoadHarnessIgnore_parsesPatterns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harnessignore"), []byte("*.log\n# comment\nnode_modules\n\n"), 0o644))

	pats, err := LoadHarnessIgnore(dir)
	require.NoError(t, err)
	require.Equal(t, []string{"*.log", "node_modules"}, pats)
}

func TestFileFilter_excludesIgnored(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.log"), []byte("log line"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644))

	ff := &FileFilter{IgnorePatterns: []string{"*.log"}}
	require.False(t, ff.ShouldInclude(dir, "app.log"))
	require.True(t, ff.ShouldInclude(dir, "main.go"))
}

func TestFileFilter_excludesLargeFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Repeat("x", 2000)), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "small.txt"), []byte("ok"), 0o644))

	ff := &FileFilter{MaxFileSize: 100}
	require.False(t, ff.ShouldInclude(dir, "big.txt"))
	require.True(t, ff.ShouldInclude(dir, "small.txt"))
}

func TestFileFilter_excludesBinaryFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	binContent := make([]byte, 100)
	for i := range binContent {
		binContent[i] = 0
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.dat"), binContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "text.txt"), []byte("hello world"), 0o644))

	ff := &FileFilter{}
	require.False(t, ff.ShouldInclude(dir, "binary.dat"))
	require.True(t, ff.ShouldInclude(dir, "text.txt"))
}

func TestFileFilter_missingFileExcluded(t *testing.T) {
	t.Parallel()
	ff := &FileFilter{}
	require.False(t, ff.ShouldInclude(t.TempDir(), "no-such-file.txt"))
}
