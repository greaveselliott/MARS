/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/buildinfo"
	"errors"
	"io"
	"runtime/debug"
	"time"
)

const (
	maxReleaseArchiveBytes  = 64 << 20
	maxReleaseTarBytes      = 64 << 20
	maxReleaseBinaryBytes   = 48 << 20
	maxReleaseDocumentBytes = 2 << 20
	releaseTarBlockBytes    = 512
	releaseGoVersion        = "go1.26.5"
	releaseModulePath       = "github.com/greaveselliott/mars"
)

var (
	// These fixed errors deliberately exclude archive names, paths, header
	// values, build settings, and downloaded content.
	ErrReleaseArchiveIdentity = errors.New("release archive verification: release or platform identity is invalid; do not install or replace the current binary")
	ErrReleaseArchiveDigest   = errors.New("release archive verification: archive digest is not authenticated; discard the download and do not replace the current binary")
	ErrReleaseArchiveUnsafe   = errors.New("release archive verification: archive structure is unsafe; discard the download and do not replace the current binary")
	ErrReleaseBinaryMetadata  = errors.New("release archive verification: binary metadata does not match the release; discard the download and do not replace the current binary")
)

// VerifiedMARSReleaseArchive contains the authenticated and inspected MARS
// binary. The bytes remain private so callers cannot mutate the verified value.
type VerifiedMARSReleaseArchive struct {
	binary []byte
}

// Binary returns a new copy of the verified binary.
func (v VerifiedMARSReleaseArchive) Binary() []byte {
	return bytes.Clone(v.binary)
}

// VerifyMARSReleaseArchive authenticates and inspects one canonical MARS
// release archive entirely in memory. It performs no filesystem, network, or
// process operations.
func VerifyMARSReleaseArchive(archive []byte, checksums SignedChecksums, tag, fullCommit, goos, goarch string) (VerifiedMARSReleaseArchive, error) {
	return verifyMARSReleaseArchive(archive, checksums, tag, fullCommit, goos, goarch, buildinfo.Read)
}

func verifyMARSReleaseArchive(archive []byte, checksums SignedChecksums, tag, fullCommit, goos, goarch string, readBuildInfo func(io.ReaderAt) (*debug.BuildInfo, error)) (VerifiedMARSReleaseArchive, error) {
	archiveName, archSetting, archValue, ok := marsReleaseArchiveIdentity(tag, fullCommit, goos, goarch)
	if !ok || !checksums.matchesIdentity(tag, fullCommit) {
		return VerifiedMARSReleaseArchive{}, ErrReleaseArchiveIdentity
	}
	if len(archive) == 0 || len(archive) > maxReleaseArchiveBytes || checksums.Len() != expectedSignedChecksumCount {
		return VerifiedMARSReleaseArchive{}, ErrReleaseArchiveDigest
	}

	// Snapshot before hashing so every later check consumes the authenticated
	// bytes rather than caller-owned storage.
	snapshot := bytes.Clone(archive)
	wantDigest, ok := checksums.Digest(archiveName)
	if !ok || sha256.Sum256(snapshot) != wantDigest {
		return VerifiedMARSReleaseArchive{}, ErrReleaseArchiveDigest
	}

	tarBytes, err := decompressMARSReleaseArchive(snapshot)
	if err != nil {
		return VerifiedMARSReleaseArchive{}, ErrReleaseArchiveUnsafe
	}
	binary, err := extractMARSReleaseArchive(tarBytes)
	if err != nil {
		return VerifiedMARSReleaseArchive{}, ErrReleaseArchiveUnsafe
	}
	if err := verifyMARSReleaseBuildInfo(binary, fullCommit, goos, goarch, archSetting, archValue, readBuildInfo); err != nil {
		return VerifiedMARSReleaseArchive{}, ErrReleaseBinaryMetadata
	}
	return VerifiedMARSReleaseArchive{binary: bytes.Clone(binary)}, nil
}

func marsReleaseArchiveIdentity(tag, fullCommit, goos, goarch string) (name, archSetting, archValue string, ok bool) {
	if !exactReleaseTagPattern.MatchString(tag) || !exactCommitPattern.MatchString(fullCommit) {
		return "", "", "", false
	}
	switch {
	case (goos == "darwin" || goos == "linux") && goarch == "amd64":
		archSetting, archValue = "GOAMD64", "v1"
	case (goos == "darwin" || goos == "linux") && goarch == "arm64":
		archSetting, archValue = "GOARM64", "v8.0"
	default:
		return "", "", "", false
	}
	version := tag[1:]
	return "mars_" + version + "_" + goos + "_" + goarch + ".tar.gz", archSetting, archValue, true
}

func decompressMARSReleaseArchive(snapshot []byte) ([]byte, error) {
	compressed := bytes.NewReader(snapshot)
	gz, err := gzip.NewReader(compressed)
	if err != nil {
		return nil, ErrReleaseArchiveUnsafe
	}
	gz.Multistream(false)
	if !gz.Header.ModTime.IsZero() || gz.Header.Name != "" || gz.Header.Comment != "" || len(gz.Header.Extra) != 0 {
		return nil, ErrReleaseArchiveUnsafe
	}
	tarBytes, err := io.ReadAll(io.LimitReader(gz, maxReleaseTarBytes+1))
	if err != nil || len(tarBytes) == 0 || len(tarBytes) > maxReleaseTarBytes {
		return nil, ErrReleaseArchiveUnsafe
	}
	if err := gz.Close(); err != nil || compressed.Len() != 0 {
		return nil, ErrReleaseArchiveUnsafe
	}
	return tarBytes, nil
}

type releaseArchiveMember struct {
	name  string
	mode  int64
	limit int64
}

var releaseArchiveMembers = [...]releaseArchiveMember{
	{name: "LICENSE", mode: 0o644, limit: maxReleaseDocumentBytes},
	{name: "NOTICE", mode: 0o644, limit: maxReleaseDocumentBytes},
	{name: "THIRD_PARTY_NOTICES", mode: 0o644, limit: maxReleaseDocumentBytes},
	{name: "mars", mode: 0o755, limit: maxReleaseBinaryBytes},
}

func extractMARSReleaseArchive(tarBytes []byte) ([]byte, error) {
	reader := bytes.NewReader(tarBytes)
	tr := tar.NewReader(reader)
	expectedTarBytes := int64(2 * releaseTarBlockBytes)
	var binary []byte
	var expanded int64

	for _, member := range releaseArchiveMembers {
		header, err := tr.Next()
		if err != nil || !safeMARSReleaseHeader(header, member) {
			return nil, ErrReleaseArchiveUnsafe
		}
		if header.Size <= 0 || header.Size > member.limit || expanded+header.Size > maxReleaseTarBytes {
			return nil, ErrReleaseArchiveUnsafe
		}
		content := make([]byte, header.Size)
		if _, err := io.ReadFull(tr, content); err != nil {
			return nil, ErrReleaseArchiveUnsafe
		}
		expanded += header.Size
		expectedTarBytes += releaseTarBlockBytes + paddedReleaseTarSize(header.Size)
		if member.name == "mars" {
			binary = content
		}
	}
	if header, err := tr.Next(); !errors.Is(err, io.EOF) || header != nil || expectedTarBytes != int64(len(tarBytes)) {
		return nil, ErrReleaseArchiveUnsafe
	}
	return binary, nil
}

func safeMARSReleaseHeader(header *tar.Header, member releaseArchiveMember) bool {
	return header != nil &&
		header.Format == tar.FormatUSTAR &&
		header.Name == member.name &&
		header.Typeflag == tar.TypeReg &&
		header.Mode == member.mode &&
		header.Uid == 0 && header.Gid == 0 &&
		header.Uname == "root" && header.Gname == "root" &&
		header.Linkname == "" && header.Devmajor == 0 && header.Devminor == 0 &&
		len(header.PAXRecords) == 0 && len(header.Xattrs) == 0 &&
		header.AccessTime.IsZero() && header.ChangeTime.IsZero()
}

func paddedReleaseTarSize(size int64) int64 {
	return (size + releaseTarBlockBytes - 1) &^ (releaseTarBlockBytes - 1)
}

func verifyMARSReleaseBuildInfo(binary []byte, fullCommit, goos, goarch, archSetting, archValue string, readBuildInfo func(io.ReaderAt) (*debug.BuildInfo, error)) error {
	if readBuildInfo == nil {
		return ErrReleaseBinaryMetadata
	}
	info, err := readBuildInfo(bytes.NewReader(binary))
	if err != nil || info == nil {
		return ErrReleaseBinaryMetadata
	}
	return validateMARSReleaseBuildInfo(info, fullCommit, goos, goarch, archSetting, archValue)
}

func validateMARSReleaseBuildInfo(info *debug.BuildInfo, fullCommit, goos, goarch, archSetting, archValue string) error {
	if info == nil || info.GoVersion != releaseGoVersion || info.Path != DefaultPackage || info.Main.Path != releaseModulePath || info.Main.Replace != nil {
		return ErrReleaseBinaryMetadata
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		if _, duplicate := settings[setting.Key]; duplicate {
			return ErrReleaseBinaryMetadata
		}
		settings[setting.Key] = setting.Value
	}
	required := map[string]string{
		"GOOS": goos, "GOARCH": goarch, archSetting: archValue,
		"CGO_ENABLED": "0", "-trimpath": "true", "vcs": "git",
		"vcs.revision": fullCommit, "vcs.modified": "false",
	}
	for key, value := range required {
		if settings[key] != value {
			return ErrReleaseBinaryMetadata
		}
	}
	if (goarch == "amd64" && settings["GOARM64"] != "") || (goarch == "arm64" && settings["GOAMD64"] != "") {
		return ErrReleaseBinaryMetadata
	}
	commitTime, err := time.Parse(time.RFC3339, settings["vcs.time"])
	if err != nil || commitTime.IsZero() {
		return ErrReleaseBinaryMetadata
	}
	return nil
}
