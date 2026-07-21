/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	maxSnapshotArchiveBytes  = 64 << 20
	maxSnapshotExpandedBytes = 64 << 20
	maxSnapshotBinaryBytes   = 48 << 20
	maxSnapshotDocumentBytes = 2 << 20
	maxSnapshotSBOMBytes     = 4 << 20
	maxSnapshotChecksumBytes = 16 << 10
)

var (
	snapshotVersionPattern  = regexp.MustCompile(`^0\.69\.0-dev\.[0-9a-f]{7,12}$`)
	snapshotCommitPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	snapshotChecksumPattern = regexp.MustCompile(`^([0-9a-f]{64})  ([A-Za-z0-9._-]+)$`)
)

type goReleaserSnapshotPlatform struct {
	GOOS             string
	GOARCH           string
	ArchSetting      string
	ArchSettingValue string
	WorkingDir       string
}

var goReleaserSnapshotPlatforms = []goReleaserSnapshotPlatform{
	{GOOS: "darwin", GOARCH: "amd64", ArchSetting: "GOAMD64", ArchSettingValue: "v1", WorkingDir: "mars_darwin_amd64_v1"},
	{GOOS: "darwin", GOARCH: "arm64", ArchSetting: "GOARM64", ArchSettingValue: "v8.0", WorkingDir: "mars_darwin_arm64_v8.0"},
	{GOOS: "linux", GOARCH: "amd64", ArchSetting: "GOAMD64", ArchSettingValue: "v1", WorkingDir: "mars_linux_amd64_v1"},
	{GOOS: "linux", GOARCH: "arm64", ArchSetting: "GOARM64", ArchSettingValue: "v8.0", WorkingDir: "mars_linux_arm64_v8.0"},
}

type goReleaserSnapshotExpectation struct {
	Version    string
	FullCommit string
	CommitTime time.Time
	GoVersion  string
	Documents  map[string][]byte
}

type goReleaserSnapshotReport struct {
	ArchiveSHA256        map[string]string
	ArchiveChecksumLines map[string]string
	NormalizedSBOMSHA256 map[string]string
	nativeBinary         []byte
}

func TestVerifyGoReleaserSnapshotDistFromEnvironment(t *testing.T) {
	if os.Getenv("MARS_GORELEASER_VERIFY") != "1" {
		t.Skip("set MARS_GORELEASER_VERIFY=1 after producing a GoReleaser snapshot")
	}
	want := goReleaserSnapshotExpectationFromEnvironment(t)
	report, err := verifyGoReleaserSnapshotDist(os.Getenv("MARS_GORELEASER_DIST"), want)
	if err != nil {
		t.Fatalf("snapshot contract: %v", err)
	}
	if err := executeNativeSnapshotBinary(t, report.nativeBinary, want); err != nil {
		t.Fatalf("native snapshot identity: %v", err)
	}

	if second := strings.TrimSpace(os.Getenv("MARS_GORELEASER_COMPARE_DIST")); second != "" {
		if err := requireDistinctGoReleaserSnapshotRoots(os.Getenv("MARS_GORELEASER_DIST"), second); err != nil {
			t.Fatalf("comparison snapshot roots: %v", err)
		}
		other, err := verifyGoReleaserSnapshotDist(second, want)
		if err != nil {
			t.Fatalf("comparison snapshot contract: %v", err)
		}
		if err := compareGoReleaserSnapshotReports(report, other); err != nil {
			t.Fatalf("snapshot reproducibility: %v", err)
		}
	}
}

func TestGoReleaserSnapshotComparisonRejectsSameRootAndDifferentReports(t *testing.T) {
	dir := t.TempDir()
	if err := requireDistinctGoReleaserSnapshotRoots(dir, dir); err == nil {
		t.Fatal("same-root comparison must fail closed")
	}
	first := goReleaserSnapshotReport{
		ArchiveSHA256:        map[string]string{"a.tar.gz": "one"},
		ArchiveChecksumLines: map[string]string{"a.tar.gz": "one  a.tar.gz"},
		NormalizedSBOMSHA256: map[string]string{"a.tar.gz.sbom.json": "sbom"},
	}
	second := first
	second.ArchiveSHA256 = map[string]string{"a.tar.gz": "different"}
	if err := compareGoReleaserSnapshotReports(first, second); err == nil {
		t.Fatal("different archive reports must fail reproducibility")
	}
}

func TestGoReleaserDistClassificationRejectsMissingExtraAndSymlink(t *testing.T) {
	expected := []string{"a.tar.gz", "a.tar.gz.sbom.json", "checksums.txt"}
	write := func(t *testing.T, dir, name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "missing", mutate: func(t *testing.T, dir string) { os.Remove(filepath.Join(dir, "a.tar.gz")) }},
		{name: "extra", mutate: func(t *testing.T, dir string) { write(t, dir, "mars-harness-linux-amd64") }},
		{name: "symlink", mutate: func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "a.tar.gz")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("checksums.txt", filepath.Join(dir, "a.tar.gz")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range expected {
				write(t, dir, name)
			}
			test.mutate(t, dir)
			if err := classifyGoReleaserSnapshotDist(dir, expected); err == nil {
				t.Fatal("expected classification to fail closed")
			}
		})
	}
	t.Run("symlinked root", func(t *testing.T) {
		target := t.TempDir()
		for _, name := range expected {
			write(t, target, name)
		}
		link := filepath.Join(t.TempDir(), "dist")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if err := classifyGoReleaserSnapshotDist(link, expected); err == nil || !strings.Contains(err.Error(), "real directory") {
			t.Fatalf("expected symlinked root rejection, got %v", err)
		}
	})
}

func TestGoReleaserChecksumsRejectHostileRecords(t *testing.T) {
	files := map[string][]byte{"a.tar.gz": []byte("archive"), "a.tar.gz.sbom.json": []byte("sbom")}
	valid := snapshotChecksumFixture(files)
	tests := []struct {
		name string
		text string
	}{
		{name: "empty", text: ""},
		{name: "missing", text: strings.Split(valid, "\n")[0] + "\n"},
		{name: "duplicate", text: valid + strings.Split(valid, "\n")[0] + "\n"},
		{name: "extra", text: valid + strings.Repeat("0", 64) + "  extra\n"},
		{name: "path", text: strings.Replace(valid, "a.tar.gz", "../a.tar.gz", 1)},
		{name: "star", text: strings.Replace(valid, "  a.tar.gz", " *a.tar.gz", 1)},
		{name: "crlf", text: strings.ReplaceAll(valid, "\n", "\r\n")},
		{name: "uppercase hash", text: strings.ToUpper(valid[:64]) + valid[64:]},
		{name: "mismatch", text: strings.Repeat("0", 64) + valid[64:]},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, data := range files {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(test.text), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := verifyGoReleaserSnapshotChecksums(dir, []string{"a.tar.gz", "a.tar.gz.sbom.json"}); err == nil {
				t.Fatal("expected checksum verification to fail closed")
			}
		})
	}
}

func TestGoReleaserSecondReadRequiresChecksumMatch(t *testing.T) {
	data := []byte("sealed bytes")
	digest := sha256.Sum256(data)
	if _, err := verifyGoReleaserReadDigest("asset", data, hex.EncodeToString(digest[:])); err != nil {
		t.Fatalf("exact second read should pass: %v", err)
	}
	if _, err := verifyGoReleaserReadDigest("asset", append(data, '!'), hex.EncodeToString(digest[:])); err == nil {
		t.Fatal("mutated second read must fail closed")
	}
}

func TestGoReleaserArchiveRejectsHostileMembers(t *testing.T) {
	want := goReleaserSnapshotExpectation{
		FullCommit: strings.Repeat("a", 40),
		CommitTime: time.Date(2026, 7, 21, 21, 0, 0, 0, time.UTC),
		GoVersion:  "go1.26.5",
		Documents: map[string][]byte{
			"LICENSE":             []byte("license"),
			"NOTICE":              []byte("notice"),
			"THIRD_PARTY_NOTICES": []byte("third party"),
		},
	}
	platform := goReleaserSnapshotPlatforms[0]
	regular := func(name string, mode int64, data string) snapshotTarFixtureEntry {
		return snapshotTarFixtureEntry{Header: tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: mode, Uid: 0, Gid: 0, Uname: "root", Gname: "root", ModTime: want.CommitTime}, Data: []byte(data)}
	}
	base := []snapshotTarFixtureEntry{
		regular("LICENSE", 0o644, "license"),
		regular("NOTICE", 0o644, "notice"),
		regular("THIRD_PARTY_NOTICES", 0o644, "third party"),
		regular("mars", 0o755, "not a Go binary"),
	}

	tests := []struct {
		name   string
		mutate func([]snapshotTarFixtureEntry) []snapshotTarFixtureEntry
		want   string
	}{
		{name: "traversal", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry {
			entries[0].Header.Name = "../LICENSE"
			return entries
		}, want: "archive member 1"},
		{name: "wrong order", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry {
			entries[0], entries[1] = entries[1], entries[0]
			return entries
		}, want: "archive member 1"},
		{name: "symlink", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry {
			entries[0].Header.Typeflag = tar.TypeSymlink
			return entries
		}, want: "not a plain regular file"},
		{name: "device", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry {
			entries[0].Header.Typeflag = tar.TypeChar
			return entries
		}, want: "not a plain regular file"},
		{name: "duplicate", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry {
			return append(entries[:1], append([]snapshotTarFixtureEntry{entries[0]}, entries[1:]...)...)
		}, want: "archive member 2"},
		{name: "missing", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry { return entries[:3] }, want: "archive contains 3 members"},
		{name: "wrong mode", mutate: func(entries []snapshotTarFixtureEntry) []snapshotTarFixtureEntry {
			entries[0].Header.Mode = 0o666
			return entries
		}, want: "unexpected ownership or mode"},
	}
	if _, err := inspectGoReleaserSnapshotArchive(makeSnapshotTarGz(t, base), platform, want); err == nil || !strings.Contains(err.Error(), "read Go build info") {
		t.Fatalf("valid synthetic structure should reach buildinfo validation, got %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]snapshotTarFixtureEntry(nil), base...)
			entries = test.mutate(entries)
			archive := makeSnapshotTarGz(t, entries)
			if _, err := inspectGoReleaserSnapshotArchive(archive, platform, want); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q before buildinfo validation, got %v", test.want, err)
			}
		})
	}
}

func TestGoReleaserSBOMNormalizationDropsOnlyApprovedVolatileFields(t *testing.T) {
	archiveDigest := strings.Repeat("a", 64)
	binaryDigest := strings.Repeat("b", 64)
	fixture := func(namespace, created string) []byte {
		return []byte(fmt.Sprintf(`{"spdxVersion":"SPDX-2.3","dataLicense":"CC0-1.0","SPDXID":"SPDXRef-DOCUMENT","name":"a.tar.gz","documentNamespace":%q,"creationInfo":{"created":%q,"creators":["Tool: syft"]},"packages":[{"name":"a.tar.gz","SPDXID":"SPDXRef-DocumentRoot-File-a","primaryPackagePurpose":"FILE","versionInfo":"sha256:%s","checksums":[{"algorithm":"SHA256","checksumValue":"%s"}]}],"files":[{"fileName":"mars","checksums":[{"algorithm":"SHA256","checksumValue":"%s"}]}]}`, namespace, created, archiveDigest, archiveDigest, binaryDigest))
	}
	first := fixture("https://example.test/one", "2026-07-21T21:00:00Z")
	second := fixture("https://example.test/two", "2026-07-21T21:01:00Z")
	left, err := normalizeGoReleaserSnapshotSBOM(first, "a.tar.gz", archiveDigest, binaryDigest)
	if err != nil {
		t.Fatal(err)
	}
	right, err := normalizeGoReleaserSnapshotSBOM(second, "a.tar.gz", archiveDigest, binaryDigest)
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatal("approved volatile SPDX fields should normalize equally")
	}

	mutated := bytes.Replace(second, []byte(`"Tool: syft"`), []byte(`"Tool: other"`), 1)
	changed, err := normalizeGoReleaserSnapshotSBOM(mutated, "a.tar.gz", archiveDigest, binaryDigest)
	if err != nil {
		t.Fatal(err)
	}
	if left == changed {
		t.Fatal("non-volatile SPDX changes must remain visible")
	}

	for name, raw := range map[string][]byte{
		"trailing JSON":        append(append([]byte(nil), first...), []byte(`{}`)...),
		"wrong archive name":   bytes.Replace(first, []byte(`"a.tar.gz"`), []byte(`"b.tar.gz"`), 1),
		"missing namespace":    bytes.Replace(first, []byte(`"documentNamespace":"https://example.test/one",`), nil, 1),
		"missing created time": bytes.Replace(first, []byte(`"created":"2026-07-21T21:00:00Z",`), nil, 1),
		"wrong archive digest": bytes.Replace(first, []byte(archiveDigest), []byte(strings.Repeat("c", 64)), 1),
		"wrong binary digest":  bytes.Replace(first, []byte(binaryDigest), []byte(strings.Repeat("d", 64)), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeGoReleaserSnapshotSBOM(raw, "a.tar.gz", archiveDigest, binaryDigest); err == nil {
				t.Fatal("expected SPDX normalization to fail closed")
			}
		})
	}
}

func TestGoReleaserBuildInfoContractAcceptsExactMetadataBeforeRejectingDirty(t *testing.T) {
	when := time.Date(2026, 7, 21, 21, 0, 0, 0, time.UTC)
	platform := goReleaserSnapshotPlatforms[0]
	want := goReleaserSnapshotExpectation{FullCommit: strings.Repeat("a", 40), CommitTime: when, GoVersion: "go1.26.5"}
	info := &debug.BuildInfo{
		GoVersion: "go1.26.5",
		Path:      "github.com/greaveselliott/mars/cmd/mars",
		Main:      debug.Module{Path: "github.com/greaveselliott/mars"},
		Settings: []debug.BuildSetting{
			{Key: "GOOS", Value: "darwin"}, {Key: "GOARCH", Value: "amd64"}, {Key: "GOAMD64", Value: "v1"},
			{Key: "CGO_ENABLED", Value: "0"}, {Key: "-trimpath", Value: "true"}, {Key: "vcs", Value: "git"},
			{Key: "vcs.revision", Value: want.FullCommit}, {Key: "vcs.time", Value: when.Format(time.RFC3339)}, {Key: "vcs.modified", Value: "false"},
		},
	}
	if err := validateGoReleaserSnapshotBuildInfo(info, platform, want); err != nil {
		t.Fatalf("exact synthetic build metadata should pass: %v", err)
	}
	info.Settings[len(info.Settings)-1].Value = "true"
	if err := validateGoReleaserSnapshotBuildInfo(info, platform, want); err == nil {
		t.Fatal("dirty build metadata must fail closed")
	}
}

func verifyGoReleaserSnapshotDist(distDir string, want goReleaserSnapshotExpectation) (goReleaserSnapshotReport, error) {
	if err := validateGoReleaserSnapshotExpectation(want); err != nil {
		return goReleaserSnapshotReport{}, err
	}
	archives, sboms, publishable := goReleaserSnapshotNames(want.Version)
	if err := classifyGoReleaserSnapshotDist(distDir, publishable); err != nil {
		return goReleaserSnapshotReport{}, err
	}
	checksumNames := append(append([]string(nil), archives...), sboms...)
	sort.Strings(checksumNames)
	checksums, err := verifyGoReleaserSnapshotChecksums(distDir, checksumNames)
	if err != nil {
		return goReleaserSnapshotReport{}, err
	}

	report := goReleaserSnapshotReport{
		ArchiveSHA256:        make(map[string]string, len(archives)),
		ArchiveChecksumLines: make(map[string]string, len(archives)),
		NormalizedSBOMSHA256: make(map[string]string, len(sboms)),
	}
	for i, platform := range goReleaserSnapshotPlatforms {
		archiveName := archives[i]
		archive, err := readBoundedRegularFile(filepath.Join(distDir, archiveName), maxSnapshotArchiveBytes)
		if err != nil {
			return goReleaserSnapshotReport{}, fmt.Errorf("archive %s: %w", archiveName, err)
		}
		digestText, err := verifyGoReleaserReadDigest(archiveName, archive, checksums[archiveName])
		if err != nil {
			return goReleaserSnapshotReport{}, err
		}
		report.ArchiveSHA256[archiveName] = digestText
		report.ArchiveChecksumLines[archiveName] = checksums[archiveName] + "  " + archiveName
		binary, err := inspectGoReleaserSnapshotArchive(archive, platform, want)
		if err != nil {
			return goReleaserSnapshotReport{}, fmt.Errorf("archive %s: %w", archiveName, err)
		}
		if platform.GOOS == runtime.GOOS && platform.GOARCH == runtime.GOARCH {
			if report.nativeBinary != nil {
				return goReleaserSnapshotReport{}, errors.New("multiple native binaries in snapshot")
			}
			report.nativeBinary = binary
		}

		sbomName := sboms[i]
		rawSBOM, err := readBoundedRegularFile(filepath.Join(distDir, sbomName), maxSnapshotSBOMBytes)
		if err != nil {
			return goReleaserSnapshotReport{}, fmt.Errorf("SBOM %s: %w", sbomName, err)
		}
		if _, err := verifyGoReleaserReadDigest(sbomName, rawSBOM, checksums[sbomName]); err != nil {
			return goReleaserSnapshotReport{}, err
		}
		binaryDigest := sha256.Sum256(binary)
		normalized, err := normalizeGoReleaserSnapshotSBOM(rawSBOM, archiveName, digestText, hex.EncodeToString(binaryDigest[:]))
		if err != nil {
			return goReleaserSnapshotReport{}, fmt.Errorf("SBOM %s: %w", sbomName, err)
		}
		report.NormalizedSBOMSHA256[sbomName] = hex.EncodeToString(normalized[:])
	}
	if report.nativeBinary == nil {
		return goReleaserSnapshotReport{}, fmt.Errorf("snapshot has no native %s/%s artifact", runtime.GOOS, runtime.GOARCH)
	}
	return report, nil
}

func validateGoReleaserSnapshotExpectation(want goReleaserSnapshotExpectation) error {
	if !snapshotVersionPattern.MatchString(want.Version) {
		return fmt.Errorf("invalid snapshot version %q", want.Version)
	}
	if !snapshotCommitPattern.MatchString(want.FullCommit) {
		return errors.New("full commit must be 40 lowercase hexadecimal characters")
	}
	if want.CommitTime.IsZero() {
		return errors.New("commit time is required")
	}
	if want.GoVersion != "go1.26.5" {
		return fmt.Errorf("unexpected Go toolchain %q", want.GoVersion)
	}
	for _, name := range []string{"LICENSE", "NOTICE", "THIRD_PARTY_NOTICES"} {
		if len(want.Documents[name]) == 0 {
			return fmt.Errorf("expected document %s is empty", name)
		}
	}
	return nil
}

func goReleaserSnapshotNames(version string) (archives, sboms, publishable []string) {
	for _, platform := range goReleaserSnapshotPlatforms {
		archive := fmt.Sprintf("mars_%s_%s_%s.tar.gz", version, platform.GOOS, platform.GOARCH)
		archives = append(archives, archive)
		sboms = append(sboms, archive+".sbom.json")
	}
	publishable = append(append(append([]string(nil), archives...), sboms...), "checksums.txt")
	sort.Strings(publishable)
	return archives, sboms, publishable
}

func classifyGoReleaserSnapshotDist(distDir string, expected []string) error {
	if strings.TrimSpace(distDir) == "" {
		return errors.New("dist directory is required")
	}
	rootInfo, err := os.Lstat(distDir)
	if err != nil {
		return fmt.Errorf("inspect dist root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return errors.New("dist root must be a real directory, not a symlink")
	}
	expectedSet := make(map[string]struct{}, len(expected))
	for _, name := range expected {
		expectedSet[name] = struct{}{}
	}
	workingFiles := map[string]struct{}{"artifacts.json": {}, "config.yaml": {}, "metadata.json": {}}
	workingDirs := make(map[string]struct{}, len(goReleaserSnapshotPlatforms))
	for _, platform := range goReleaserSnapshotPlatforms {
		workingDirs[platform.WorkingDir] = struct{}{}
	}
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return fmt.Errorf("read dist: %w", err)
	}
	var found []string
	for _, entry := range entries {
		name := entry.Name()
		info, err := os.Lstat(filepath.Join(distDir, name))
		if err != nil {
			return fmt.Errorf("inspect top-level entry %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("top-level entry %s is a symlink", name)
		}
		if _, ok := expectedSet[name]; ok {
			if !info.Mode().IsRegular() || info.Size() <= 0 {
				return fmt.Errorf("publishable entry %s is not a non-empty regular file", name)
			}
			found = append(found, name)
			continue
		}
		if _, ok := workingFiles[name]; ok {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("working metadata %s is not a regular file", name)
			}
			continue
		}
		if _, ok := workingDirs[name]; ok {
			if !info.IsDir() {
				return fmt.Errorf("working output %s is not a directory", name)
			}
			continue
		}
		return fmt.Errorf("unexpected top-level dist entry %s", name)
	}
	sort.Strings(found)
	if !equalStrings(found, expected) {
		return fmt.Errorf("publishable set mismatch: got %v want %v", found, expected)
	}
	return nil
}

func verifyGoReleaserReadDigest(name string, data []byte, expected string) (string, error) {
	digest := sha256.Sum256(data)
	actual := hex.EncodeToString(digest[:])
	if actual != expected {
		return "", fmt.Errorf("checksummed bytes changed before inspection for %s", name)
	}
	return actual, nil
}

func verifyGoReleaserSnapshotChecksums(distDir string, expected []string) (map[string]string, error) {
	raw, err := readBoundedRegularFile(filepath.Join(distDir, "checksums.txt"), maxSnapshotChecksumBytes)
	if err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' || bytes.Contains(raw, []byte{'\r'}) {
		return nil, errors.New("checksums must be non-empty LF-terminated text")
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != len(expected) {
		return nil, fmt.Errorf("checksums contain %d entries, expected %d", len(lines), len(expected))
	}
	records := make(map[string]string, len(lines))
	for i, line := range lines {
		match := snapshotChecksumPattern.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("checksum line %d is not canonical", i+1)
		}
		if _, duplicate := records[match[2]]; duplicate {
			return nil, fmt.Errorf("duplicate checksum for %s", match[2])
		}
		records[match[2]] = match[1]
	}
	sorted := append([]string(nil), expected...)
	sort.Strings(sorted)
	for i, name := range sorted {
		if lines[i] != records[name]+"  "+name {
			return nil, errors.New("checksums are not the exact sorted publishable set")
		}
		limit := int64(maxSnapshotSBOMBytes)
		if strings.HasSuffix(name, ".tar.gz") {
			limit = maxSnapshotArchiveBytes
		}
		data, err := readBoundedRegularFile(filepath.Join(distDir, name), limit)
		if err != nil {
			return nil, fmt.Errorf("checksum target %s: %w", name, err)
		}
		digest := sha256.Sum256(data)
		if records[name] != hex.EncodeToString(digest[:]) {
			return nil, fmt.Errorf("checksum mismatch for %s", name)
		}
	}
	return records, nil
}

func inspectGoReleaserSnapshotArchive(data []byte, platform goReleaserSnapshotPlatform, want goReleaserSnapshotExpectation) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()
	if !gz.Header.ModTime.IsZero() || gz.Header.Name != "" || gz.Header.Comment != "" {
		return nil, errors.New("gzip header contains non-deterministic metadata")
	}
	tr := tar.NewReader(gz)
	members := []string{"LICENSE", "NOTICE", "THIRD_PARTY_NOTICES", "mars"}
	var binary []byte
	var expanded int64
	for index := 0; ; index++ {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			if index != len(members) {
				return nil, fmt.Errorf("archive contains %d members, expected %d", index, len(members))
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar header: %w", err)
		}
		if index >= len(members) {
			return nil, errors.New("archive contains an extra member")
		}
		name := members[index]
		if header.Name != name {
			return nil, fmt.Errorf("archive member %d is %q, expected %q", index+1, header.Name, name)
		}
		if header.Typeflag != tar.TypeReg || header.Linkname != "" || header.Devmajor != 0 || header.Devminor != 0 {
			return nil, fmt.Errorf("archive member %s is not a plain regular file", name)
		}
		mode := int64(0o644)
		limit := int64(maxSnapshotDocumentBytes)
		if name == "mars" {
			mode = 0o755
			limit = maxSnapshotBinaryBytes
		}
		if header.Mode != mode || header.Uid != 0 || header.Gid != 0 || header.Uname != "root" || header.Gname != "root" {
			return nil, fmt.Errorf("archive member %s has unexpected ownership or mode", name)
		}
		if !header.ModTime.Equal(want.CommitTime) {
			return nil, fmt.Errorf("archive member %s does not use commit-derived mtime", name)
		}
		if header.Size <= 0 || header.Size > limit || expanded+header.Size > maxSnapshotExpandedBytes {
			return nil, fmt.Errorf("archive member %s exceeds its size contract", name)
		}
		content, err := io.ReadAll(io.LimitReader(tr, limit+1))
		if err != nil {
			return nil, fmt.Errorf("read archive member %s: %w", name, err)
		}
		if int64(len(content)) != header.Size {
			return nil, fmt.Errorf("archive member %s size mismatch", name)
		}
		expanded += header.Size
		if name == "mars" {
			binary = content
		} else if !bytes.Equal(content, want.Documents[name]) {
			return nil, fmt.Errorf("archive member %s differs from the repository document", name)
		}
	}
	info, err := buildinfo.Read(bytes.NewReader(binary))
	if err != nil {
		return nil, fmt.Errorf("read Go build info: %w", err)
	}
	if err := validateGoReleaserSnapshotBuildInfo(info, platform, want); err != nil {
		return nil, err
	}
	return binary, nil
}

func validateGoReleaserSnapshotBuildInfo(info *debug.BuildInfo, platform goReleaserSnapshotPlatform, want goReleaserSnapshotExpectation) error {
	if info.GoVersion != want.GoVersion || info.Path != "github.com/greaveselliott/mars/cmd/mars" || info.Main.Path != "github.com/greaveselliott/mars" {
		return errors.New("binary module or toolchain identity mismatch")
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		if _, duplicate := settings[setting.Key]; duplicate {
			return fmt.Errorf("duplicate Go build setting %s", setting.Key)
		}
		settings[setting.Key] = setting.Value
	}
	required := map[string]string{
		"GOOS": platform.GOOS, "GOARCH": platform.GOARCH, platform.ArchSetting: platform.ArchSettingValue,
		"CGO_ENABLED": "0", "-trimpath": "true", "vcs": "git", "vcs.revision": want.FullCommit, "vcs.modified": "false",
	}
	for key, value := range required {
		if settings[key] != value {
			return fmt.Errorf("Go build setting %s is %q, expected %q", key, settings[key], value)
		}
	}
	built, err := time.Parse(time.RFC3339, settings["vcs.time"])
	if err != nil || !built.Equal(want.CommitTime) {
		return errors.New("Go build vcs.time does not match the commit time")
	}
	return nil
}

func normalizeGoReleaserSnapshotSBOM(raw []byte, archiveName, archiveDigest, binaryDigest string) ([32]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return [32]byte{}, fmt.Errorf("decode SPDX JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return [32]byte{}, errors.New("SPDX JSON must contain exactly one document")
	}
	for key, value := range map[string]string{
		"spdxVersion": "SPDX-2.3", "dataLicense": "CC0-1.0", "SPDXID": "SPDXRef-DOCUMENT", "name": archiveName,
	} {
		if document[key] != value {
			return [32]byte{}, fmt.Errorf("SPDX field %s does not match the snapshot contract", key)
		}
	}
	namespace, ok := document["documentNamespace"].(string)
	if !ok || strings.TrimSpace(namespace) == "" {
		return [32]byte{}, errors.New("SPDX documentNamespace is missing")
	}
	creation, ok := document["creationInfo"].(map[string]any)
	if !ok {
		return [32]byte{}, errors.New("SPDX creationInfo is missing")
	}
	created, ok := creation["created"].(string)
	if !ok {
		return [32]byte{}, errors.New("SPDX creationInfo.created is missing")
	}
	if _, err := time.Parse(time.RFC3339, created); err != nil {
		return [32]byte{}, errors.New("SPDX creationInfo.created is invalid")
	}
	if err := verifyGoReleaserSBOMArtifactBindings(document, archiveName, archiveDigest, binaryDigest); err != nil {
		return [32]byte{}, err
	}
	delete(document, "documentNamespace")
	delete(creation, "created")
	normalized, err := json.Marshal(document)
	if err != nil {
		return [32]byte{}, fmt.Errorf("marshal normalized SPDX JSON: %w", err)
	}
	return sha256.Sum256(normalized), nil
}

func verifyGoReleaserSBOMArtifactBindings(document map[string]any, archiveName, archiveDigest, binaryDigest string) error {
	packages, ok := document["packages"].([]any)
	if !ok {
		return errors.New("SPDX packages are missing")
	}
	rootCount := 0
	for _, value := range packages {
		pkg, ok := value.(map[string]any)
		if !ok || pkg["name"] != archiveName || pkg["primaryPackagePurpose"] != "FILE" {
			continue
		}
		id, _ := pkg["SPDXID"].(string)
		if !strings.HasPrefix(id, "SPDXRef-DocumentRoot-File-") || pkg["versionInfo"] != "sha256:"+archiveDigest || !hasGoReleaserSPDXChecksum(pkg["checksums"], archiveDigest) {
			return errors.New("SPDX document-root package does not bind the archive SHA-256")
		}
		rootCount++
	}
	if rootCount != 1 {
		return errors.New("SPDX must contain exactly one archive document-root package")
	}
	files, ok := document["files"].([]any)
	if !ok {
		return errors.New("SPDX files are missing")
	}
	marsCount := 0
	for _, value := range files {
		file, ok := value.(map[string]any)
		if !ok || file["fileName"] != "mars" {
			continue
		}
		if !hasGoReleaserSPDXChecksum(file["checksums"], binaryDigest) {
			return errors.New("SPDX mars file does not bind the binary SHA-256")
		}
		marsCount++
	}
	if marsCount != 1 {
		return errors.New("SPDX must contain exactly one mars file record")
	}
	return nil
}

func hasGoReleaserSPDXChecksum(value any, digest string) bool {
	checksums, ok := value.([]any)
	if !ok {
		return false
	}
	for _, value := range checksums {
		checksum, ok := value.(map[string]any)
		if ok && checksum["algorithm"] == "SHA256" && checksum["checksumValue"] == digest {
			return true
		}
	}
	return false
}

func compareGoReleaserSnapshotReports(first, second goReleaserSnapshotReport) error {
	for name, digest := range first.ArchiveSHA256 {
		if second.ArchiveSHA256[name] != digest {
			return fmt.Errorf("archive bytes differ for %s", name)
		}
		if second.ArchiveChecksumLines[name] != first.ArchiveChecksumLines[name] {
			return fmt.Errorf("archive checksum record differs for %s", name)
		}
	}
	for name, digest := range first.NormalizedSBOMSHA256 {
		if second.NormalizedSBOMSHA256[name] != digest {
			return fmt.Errorf("normalized SBOM differs for %s", name)
		}
	}
	return nil
}

func requireDistinctGoReleaserSnapshotRoots(first, second string) error {
	firstInfo, err := os.Stat(first)
	if err != nil {
		return fmt.Errorf("inspect first snapshot root: %w", err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		return fmt.Errorf("inspect second snapshot root: %w", err)
	}
	if !firstInfo.IsDir() || !secondInfo.IsDir() || os.SameFile(firstInfo, secondInfo) {
		return errors.New("snapshot comparison requires two distinct directories")
	}
	return nil
}

func readBoundedRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > limit {
		return nil, errors.New("file is not a non-empty bounded regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != info.Size() || int64(len(data)) > limit {
		return nil, errors.New("file changed or exceeded its size bound while reading")
	}
	return data, nil
}

func executeNativeSnapshotBinary(t *testing.T, binary []byte, want goReleaserSnapshotExpectation) error {
	t.Helper()
	if len(binary) == 0 {
		return errors.New("native binary is missing")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "mars")
	if err := os.WriteFile(path, binary, 0o700); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Env = []string{"HOME=" + dir, "PATH=/usr/bin:/bin", "TMPDIR=" + dir}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("native --version failed: %w", err)
	}
	wantLine := fmt.Sprintf("mars %s %s/%s commit=%s built=%s", want.Version, runtime.GOOS, runtime.GOARCH, want.FullCommit, want.CommitTime.UTC().Format(time.RFC3339))
	if strings.TrimSpace(string(output)) != wantLine {
		return errors.New("native --version output does not bind the expected snapshot identity")
	}
	return nil
}

func goReleaserSnapshotExpectationFromEnvironment(t *testing.T) goReleaserSnapshotExpectation {
	t.Helper()
	commitTime, err := time.Parse(time.RFC3339, strings.TrimSpace(os.Getenv("MARS_GORELEASER_COMMIT_TIME")))
	if err != nil {
		t.Fatalf("parse MARS_GORELEASER_COMMIT_TIME: %v", err)
	}
	documents := make(map[string][]byte, 3)
	for _, name := range []string{"LICENSE", "NOTICE", "THIRD_PARTY_NOTICES"} {
		data, err := os.ReadFile(filepath.Join(releaseRepoRoot(t), name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		documents[name] = data
	}
	return goReleaserSnapshotExpectation{
		Version:    strings.TrimSpace(os.Getenv("MARS_GORELEASER_VERSION")),
		FullCommit: strings.TrimSpace(os.Getenv("MARS_GORELEASER_COMMIT")),
		CommitTime: commitTime,
		GoVersion:  strings.TrimSpace(os.Getenv("MARS_GORELEASER_GO_VERSION")),
		Documents:  documents,
	}
}

type snapshotTarFixtureEntry struct {
	Header tar.Header
	Data   []byte
}

func makeSnapshotTarGz(t *testing.T, entries []snapshotTarFixtureEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	gz := gzip.NewWriter(&output)
	gz.Header.ModTime = time.Time{}
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		header := entry.Header
		data := entry.Data
		if header.Typeflag != tar.TypeReg {
			data = nil
		}
		header.Size = int64(len(data))
		if err := tw.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func snapshotChecksumFixture(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var output strings.Builder
	for _, name := range names {
		digest := sha256.Sum256(files[name])
		fmt.Fprintf(&output, "%x  %s\n", digest, name)
	}
	return output.String()
}

func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}
