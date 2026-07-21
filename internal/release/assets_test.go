/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-017-open-source-publication.md
- docs/features/F-018-goreleaser-distribution.md
*/
package release

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

func TestVerifyLocalAssetsPreservesLegacyConsumerUntilT066(t *testing.T) {
	t.Parallel()
	dist := writeLegacyReleaseAssetFixture(t)

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

func TestVerifyLocalAssetsRejectsExtraProducerOutput(t *testing.T) {
	t.Parallel()
	dist := writeLegacyReleaseAssetFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(dist, "unexpected"), []byte("extra"), 0o644))

	report, err := VerifyLocalAssets(dist, "v1.2.3")
	require.NoError(t, err)
	require.False(t, report.OK)
	require.Equal(t, []string{"unexpected"}, report.Extra)
}

func writeLegacyReleaseAssetFixture(t *testing.T) string {
	t.Helper()
	dist := t.TempDir()
	var checksumLines []string
	for _, name := range selfupdate.ExpectedReleaseAssetNames() {
		if name == "checksums.txt" {
			continue
		}
		data := []byte("binary " + name)
		require.NoError(t, os.WriteFile(filepath.Join(dist, name), data, 0o755))
		sum := sha256.Sum256(data)
		checksumLines = append(checksumLines, fmt.Sprintf("%x  %s", sum, name))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(dist, "checksums.txt"),
		[]byte(strings.Join(checksumLines, "\n")+"\n"),
		0o644,
	))
	return dist
}
