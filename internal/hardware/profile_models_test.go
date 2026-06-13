/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package hardware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMissingRequiredModelFiles_reportsAbsentWeights(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missing, err := MissingRequiredModelFiles(dir, PerformanceBalanced)
	require.NoError(t, err)
	require.NotEmpty(t, missing)
}

func TestMissingRequiredModelFiles_emptyWhenPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	hw := Detect()
	for _, spec := range UniqueModels(DefaultModelsForHardware(hw, PerformanceBalanced)) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, spec.File), []byte("stub"), 0o644))
	}
	missing, err := MissingRequiredModelFiles(dir, PerformanceBalanced)
	require.NoError(t, err)
	require.Empty(t, missing)
}

func TestProfileModelPreflightError_actionable(t *testing.T) {
	t.Parallel()
	err := ProfileModelPreflightError(PerformanceBalanced, []string{"missing.gguf"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mars-harness setup")
	require.Contains(t, err.Error(), "missing.gguf")
}
