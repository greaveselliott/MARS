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
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/greaveselliott/mars/internal/selfupdate"
)

// VerifyLocalAssets checks the retained legacy local release contract while
// T-066 migrates consumers to the GoReleaser archive contract.
func VerifyLocalAssets(distDir, version string) (selfupdate.ReleaseAssetReport, error) {
	distDir = strings.TrimSpace(distDir)
	if distDir == "" {
		return selfupdate.ReleaseAssetReport{}, fmt.Errorf("release verify-assets: --dist path is empty")
	}
	abs, err := filepath.Abs(distDir)
	if err != nil {
		return selfupdate.ReleaseAssetReport{}, fmt.Errorf("release verify-assets: resolve dist: %w", err)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return selfupdate.ReleaseAssetReport{}, fmt.Errorf("release verify-assets: read local dist %s: %w", abs, err)
	}
	foundSet := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			foundSet[entry.Name()] = true
		}
	}
	required := selfupdate.ExpectedReleaseAssetNames()
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	var found, missing, extra []string
	for _, name := range required {
		if foundSet[name] {
			found = append(found, name)
		} else {
			missing = append(missing, name)
		}
	}
	for name := range foundSet {
		if !requiredSet[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(found)
	sort.Strings(extra)
	if len(missing) == 0 && len(extra) == 0 {
		if err := verifyChecksums(filepath.Join(abs, "checksums.txt"), abs); err != nil {
			missing = append(missing, "valid checksums.txt")
		}
	}
	tag := releaseTag(version)
	return selfupdate.ReleaseAssetReport{
		TagName:  tag,
		Version:  strings.TrimPrefix(tag, "v"),
		URL:      abs,
		Required: required,
		Found:    found,
		Missing:  missing,
		Extra:    extra,
		OK:       len(missing) == 0 && len(extra) == 0,
	}, nil
}

func verifyChecksums(path, dir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			return fmt.Errorf("invalid checksum line %q", scanner.Text())
		}
		got, err := fileSHA256(filepath.Join(dir, fields[len(fields)-1]))
		if err != nil {
			return err
		}
		if got != fields[0] {
			return fmt.Errorf("checksum mismatch for %s", fields[len(fields)-1])
		}
	}
	return scanner.Err()
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("release assets: open %s: %w", path, err)
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("release assets: hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func releaseTag(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
