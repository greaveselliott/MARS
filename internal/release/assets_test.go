/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

func TestVerifyLocalAssetsAcceptsCanonicalAndLegacyBinaryNames(t *testing.T) {
	t.Parallel()
	dist := t.TempDir()
	assets := make([]string, 0, len(AssetTargets)*2)

	for _, target := range AssetTargets {
		name := fmt.Sprintf("mars-%s-%s", target.GOOS, target.GOARCH)
		canonical := filepath.Join(dist, name)
		require.NoError(t, os.WriteFile(canonical, []byte("binary "+name), 0o755))
		assets = append(assets, canonical)

		legacyName := fmt.Sprintf("mars-harness-%s-%s", target.GOOS, target.GOARCH)
		legacy := filepath.Join(dist, legacyName)
		require.NoError(t, copyFile(canonical, legacy))
		assets = append(assets, legacy)
	}
	require.NoError(t, writeChecksums(filepath.Join(dist, "checksums.txt"), assets))

	report, err := VerifyLocalAssets(dist, "v1.2.3")
	require.NoError(t, err)
	require.True(t, report.OK, "missing: %v", report.Missing)
	require.Equal(t, "v1.2.3", report.TagName)
	require.Equal(t, "1.2.3", report.Version)
	require.Empty(t, report.Missing)
	require.ElementsMatch(t, selfupdate.ExpectedReleaseAssetNames(), report.Found)

	require.NoError(t, os.WriteFile(filepath.Join(dist, "mars-darwin-arm64"), []byte("tampered"), 0o755))
	report, err = VerifyLocalAssets(dist, "1.2.3")
	require.NoError(t, err)
	require.False(t, report.OK)
	require.Contains(t, report.Missing, "valid checksums.txt")
}
