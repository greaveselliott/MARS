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
	"errors"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testReleaseTag    = "v0.69.0"
	testReleaseCommit = "0123456789abcdef0123456789abcdef01234567"
)

func TestVerifyMARSReleaseArchiveExactContract(t *testing.T) {
	binary := []byte("synthetic Go binary")
	archive := canonicalMARSReleaseArchive(t, binary, nil)

	for _, platform := range []struct {
		goos, goarch, archSetting, archValue string
	}{
		{"darwin", "amd64", "GOAMD64", "v1"},
		{"darwin", "arm64", "GOARM64", "v8.0"},
		{"linux", "amd64", "GOAMD64", "v1"},
		{"linux", "arm64", "GOARM64", "v8.0"},
	} {
		t.Run(platform.goos+"_"+platform.goarch, func(t *testing.T) {
			name, _, _, ok := marsReleaseArchiveIdentity(testReleaseTag, testReleaseCommit, platform.goos, platform.goarch)
			require.True(t, ok)
			checksums := signedChecksumsForArchive(name, archive, testReleaseTag, testReleaseCommit)
			calls := 0
			got, err := verifyMARSReleaseArchive(archive, checksums, testReleaseTag, testReleaseCommit, platform.goos, platform.goarch, func(io.ReaderAt) (*debug.BuildInfo, error) {
				calls++
				return validMARSReleaseBuildInfo(platform.goos, platform.goarch, platform.archSetting, platform.archValue), nil
			})
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Equal(t, binary, got.Binary())

			archive[0] ^= 1
			first := got.Binary()
			first[0] ^= 1
			require.Equal(t, binary, got.Binary(), "verified output must not alias input or prior accessors")
			archive[0] ^= 1
		})
	}
}

func TestVerifyMARSReleaseArchiveAdmission(t *testing.T) {
	archive := canonicalMARSReleaseArchive(t, []byte("binary"), nil)
	name, _, _, ok := marsReleaseArchiveIdentity(testReleaseTag, testReleaseCommit, "darwin", "arm64")
	require.True(t, ok)
	valid := signedChecksumsForArchive(name, archive, testReleaseTag, testReleaseCommit)
	readerCalls := 0
	reader := func(io.ReaderAt) (*debug.BuildInfo, error) {
		readerCalls++
		return validMARSReleaseBuildInfo("darwin", "arm64", "GOARM64", "v8.0"), nil
	}

	for testName, invoke := range map[string]func() error{
		"invalid tag": func() error {
			_, err := verifyMARSReleaseArchive(archive, valid, "0.69.0", testReleaseCommit, "darwin", "arm64", reader)
			return err
		},
		"invalid commit": func() error {
			_, err := verifyMARSReleaseArchive(archive, valid, testReleaseTag, "ABC", "darwin", "arm64", reader)
			return err
		},
		"unsupported platform": func() error {
			_, err := verifyMARSReleaseArchive(archive, valid, testReleaseTag, testReleaseCommit, "windows", "amd64", reader)
			return err
		},
	} {
		t.Run(testName, func(t *testing.T) { require.ErrorIs(t, invoke(), ErrReleaseArchiveIdentity) })
	}

	boundToOtherCommit := signedChecksumsForArchive(name, archive, testReleaseTag, strings.Repeat("a", 40))
	_, err := verifyMARSReleaseArchive(archive, boundToOtherCommit, testReleaseTag, testReleaseCommit, "darwin", "arm64", reader)
	require.ErrorIs(t, err, ErrReleaseArchiveIdentity)
	require.Zero(t, readerCalls, "authenticated identity mismatch must fail before parsing build metadata")

	missingCount := signedChecksumsForArchive(name, archive, testReleaseTag, testReleaseCommit)
	delete(missingCount.digests, expectedMARSArchiveChecksumNames("0.69.0")[0])
	_, err = verifyMARSReleaseArchive(archive, missingCount, testReleaseTag, testReleaseCommit, "darwin", "arm64", reader)
	require.ErrorIs(t, err, ErrReleaseArchiveDigest)

	missingSelected := signedChecksumsForArchive(name, archive, testReleaseTag, testReleaseCommit)
	delete(missingSelected.digests, name)
	missingSelected.digests["untrusted-extra"] = sha256.Sum256([]byte("extra"))
	_, err = verifyMARSReleaseArchive(archive, missingSelected, testReleaseTag, testReleaseCommit, "darwin", "arm64", reader)
	require.ErrorIs(t, err, ErrReleaseArchiveDigest)

	malformed := []byte("not a gzip stream")
	badDigest := signedChecksumsForArchive(name, []byte("different bytes"), testReleaseTag, testReleaseCommit)
	_, err = verifyMARSReleaseArchive(malformed, badDigest, testReleaseTag, testReleaseCommit, "darwin", "arm64", reader)
	require.ErrorIs(t, err, ErrReleaseArchiveDigest, "digest admission must fail before parsing")
	require.Zero(t, readerCalls)

	oversizedArchive := make([]byte, maxReleaseArchiveBytes+1)
	_, err = verifyMARSReleaseArchive(oversizedArchive, signedChecksumsForArchive(name, oversizedArchive, testReleaseTag, testReleaseCommit), testReleaseTag, testReleaseCommit, "darwin", "arm64", reader)
	require.ErrorIs(t, err, ErrReleaseArchiveDigest)
}

func TestExtractMARSReleaseArchiveRejectsHostileStructure(t *testing.T) {
	canonical := canonicalMARSReleaseEntries([]byte("binary"))
	sparse := canonical[0].header
	sparse.Format = tar.FormatGNU
	sparse.Typeflag = tar.TypeGNUSparse
	require.False(t, safeMARSReleaseHeader(&sparse, releaseArchiveMembers[0]), "GNU sparse metadata must never be admissible")
	mutations := map[string]func([]releaseTarEntry) []releaseTarEntry{
		"absolute name": func(entries []releaseTarEntry) []releaseTarEntry { entries[0].header.Name = "/LICENSE"; return entries },
		"traversal name": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Name = "../LICENSE"
			return entries
		},
		"backslash name": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Name = `dir\LICENSE`
			return entries
		},
		"duplicate": func(entries []releaseTarEntry) []releaseTarEntry { entries[1].header.Name = "LICENSE"; return entries },
		"missing":   func(entries []releaseTarEntry) []releaseTarEntry { return entries[:3] },
		"extra": func(entries []releaseTarEntry) []releaseTarEntry {
			return append(entries, regularReleaseTarEntry("extra", 0o644, []byte("x")))
		},
		"symlink": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Typeflag = tar.TypeSymlink
			entries[0].header.Linkname = "mars"
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
		"hardlink": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Typeflag = tar.TypeLink
			entries[0].header.Linkname = "mars"
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
		"directory": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Typeflag = tar.TypeDir
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
		"character device": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Typeflag = tar.TypeChar
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
		"block device": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Typeflag = tar.TypeBlock
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
		"fifo": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Typeflag = tar.TypeFifo
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
		"GNU format": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Format = tar.FormatGNU
			return entries
		},
		"PAX metadata": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Format = tar.FormatPAX
			entries[0].header.PAXRecords = map[string]string{"comment": "hidden"}
			return entries
		},
		"wrong document mode": func(entries []releaseTarEntry) []releaseTarEntry { entries[0].header.Mode = 0o600; return entries },
		"wrong binary mode":   func(entries []releaseTarEntry) []releaseTarEntry { entries[3].header.Mode = 0o777; return entries },
		"wrong owner":         func(entries []releaseTarEntry) []releaseTarEntry { entries[0].header.Uid = 1; return entries },
		"empty member": func(entries []releaseTarEntry) []releaseTarEntry {
			entries[0].header.Size = 0
			entries[0].body = nil
			return entries
		},
	}

	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			archive := releaseTarGzip(t, mutate(cloneReleaseTarEntries(canonical)), nil)
			tarBytes, err := decompressMARSReleaseArchive(archive)
			require.NoError(t, err)
			binary, err := extractMARSReleaseArchive(tarBytes)
			require.ErrorIs(t, err, ErrReleaseArchiveUnsafe)
			require.Nil(t, binary)
		})
	}

	oversized := cloneReleaseTarEntries(canonical)
	oversized[0].body = bytes.Repeat([]byte{'x'}, maxReleaseDocumentBytes+1)
	oversized[0].header.Size = int64(len(oversized[0].body))
	tarBytes, err := decompressMARSReleaseArchive(releaseTarGzip(t, oversized, nil))
	require.NoError(t, err)
	binary, err := extractMARSReleaseArchive(tarBytes)
	require.ErrorIs(t, err, ErrReleaseArchiveUnsafe)
	require.Nil(t, binary)

	binary, err = extractMARSReleaseArchive(tarWithDeclaredOversizedBinary(t))
	require.ErrorIs(t, err, ErrReleaseArchiveUnsafe)
	require.Nil(t, binary)
}

func TestDecompressMARSReleaseArchiveRejectsAmbiguousStreams(t *testing.T) {
	valid := canonicalMARSReleaseArchive(t, []byte("binary"), nil)
	badCRC := append([]byte(nil), valid...)
	badCRC[len(badCRC)-8] ^= 1
	concatenated := append(append([]byte(nil), valid...), valid...)
	rawTrailing := append(append([]byte(nil), valid...), []byte("TOKEN-TRAIL")...)
	insideGzipTrailing := canonicalMARSReleaseArchive(t, []byte("binary"), []byte("TOKEN-TRAIL"))
	truncated := append([]byte(nil), valid[:len(valid)-4]...)
	malformedTar := gzipBytes(t, []byte("not a tar stream"))

	for name, archive := range map[string][]byte{
		"malformed gzip":              []byte("not gzip"),
		"truncated":                   truncated,
		"bad CRC":                     badCRC,
		"concatenated gzip":           concatenated,
		"raw trailing bytes":          rawTrailing,
		"decompressed trailing bytes": insideGzipTrailing,
		"malformed tar":               malformedTar,
	} {
		t.Run(name, func(t *testing.T) {
			tarBytes, err := decompressMARSReleaseArchive(archive)
			if err == nil {
				_, err = extractMARSReleaseArchive(tarBytes)
			}
			require.ErrorIs(t, err, ErrReleaseArchiveUnsafe)
		})
	}

	tooLarge := gzipBytes(t, bytes.Repeat([]byte{0}, maxReleaseTarBytes+1))
	_, err := decompressMARSReleaseArchive(tooLarge)
	require.ErrorIs(t, err, ErrReleaseArchiveUnsafe)
}

func TestValidateMARSReleaseBuildInfo(t *testing.T) {
	for _, platform := range []struct {
		goos, goarch, archSetting, archValue string
	}{
		{"darwin", "amd64", "GOAMD64", "v1"},
		{"darwin", "arm64", "GOARM64", "v8.0"},
		{"linux", "amd64", "GOAMD64", "v1"},
		{"linux", "arm64", "GOARM64", "v8.0"},
	} {
		info := validMARSReleaseBuildInfo(platform.goos, platform.goarch, platform.archSetting, platform.archValue)
		require.NoError(t, validateMARSReleaseBuildInfo(info, testReleaseCommit, platform.goos, platform.goarch, platform.archSetting, platform.archValue))
	}

	mutations := map[string]func(*debug.BuildInfo){
		"toolchain":          func(info *debug.BuildInfo) { info.GoVersion = "go1.26.5" },
		"command path":       func(info *debug.BuildInfo) { info.Path = "attacker.invalid/mars" },
		"module path":        func(info *debug.BuildInfo) { info.Main.Path = "attacker.invalid/module" },
		"module replacement": func(info *debug.BuildInfo) { info.Main.Replace = &debug.Module{Path: "local"} },
		"GOOS":               func(info *debug.BuildInfo) { setBuildSetting(info, "GOOS", "linux") },
		"missing GOOS":       func(info *debug.BuildInfo) { removeBuildSetting(info, "GOOS") },
		"GOARCH":             func(info *debug.BuildInfo) { setBuildSetting(info, "GOARCH", "amd64") },
		"architecture level": func(info *debug.BuildInfo) { setBuildSetting(info, "GOARM64", "v9.0") },
		"other architecture level": func(info *debug.BuildInfo) {
			info.Settings = append(info.Settings, debug.BuildSetting{Key: "GOAMD64", Value: "v1"})
		},
		"CGO":            func(info *debug.BuildInfo) { setBuildSetting(info, "CGO_ENABLED", "1") },
		"trimpath":       func(info *debug.BuildInfo) { setBuildSetting(info, "-trimpath", "false") },
		"VCS":            func(info *debug.BuildInfo) { setBuildSetting(info, "vcs", "hg") },
		"revision":       func(info *debug.BuildInfo) { setBuildSetting(info, "vcs.revision", strings.Repeat("f", 40)) },
		"modified":       func(info *debug.BuildInfo) { setBuildSetting(info, "vcs.modified", "true") },
		"missing time":   func(info *debug.BuildInfo) { removeBuildSetting(info, "vcs.time") },
		"malformed time": func(info *debug.BuildInfo) { setBuildSetting(info, "vcs.time", "not-a-time") },
		"zero time":      func(info *debug.BuildInfo) { setBuildSetting(info, "vcs.time", "0001-01-01T00:00:00Z") },
		"duplicate setting": func(info *debug.BuildInfo) {
			info.Settings = append(info.Settings, debug.BuildSetting{Key: "GOOS", Value: "darwin"})
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			info := validMARSReleaseBuildInfo("darwin", "arm64", "GOARM64", "v8.0")
			mutate(info)
			require.ErrorIs(t, validateMARSReleaseBuildInfo(info, testReleaseCommit, "darwin", "arm64", "GOARM64", "v8.0"), ErrReleaseBinaryMetadata)
		})
	}
	require.ErrorIs(t, validateMARSReleaseBuildInfo(nil, testReleaseCommit, "darwin", "arm64", "GOARM64", "v8.0"), ErrReleaseBinaryMetadata)
}

func TestMARSReleaseArchiveErrorsAreFixedAndRedacted(t *testing.T) {
	const hostile = "../TOKEN-DO-NOT-LOG"
	entries := canonicalMARSReleaseEntries([]byte("binary"))
	entries[0].header.Name = hostile
	archive := releaseTarGzip(t, entries, nil)
	name, _, _, ok := marsReleaseArchiveIdentity(testReleaseTag, testReleaseCommit, "darwin", "arm64")
	require.True(t, ok)
	checksums := signedChecksumsForArchive(name, archive, testReleaseTag, testReleaseCommit)
	result, err := verifyMARSReleaseArchive(archive, checksums, testReleaseTag, testReleaseCommit, "darwin", "arm64", func(io.ReaderAt) (*debug.BuildInfo, error) {
		return nil, errors.New(hostile)
	})
	require.ErrorIs(t, err, ErrReleaseArchiveUnsafe)
	require.NotContains(t, err.Error(), hostile)
	require.Empty(t, result.Binary())

	archive = canonicalMARSReleaseArchive(t, []byte("binary"), nil)
	checksums = signedChecksumsForArchive(name, archive, testReleaseTag, testReleaseCommit)
	result, err = verifyMARSReleaseArchive(archive, checksums, testReleaseTag, testReleaseCommit, "darwin", "arm64", func(io.ReaderAt) (*debug.BuildInfo, error) {
		return nil, errors.New(hostile)
	})
	require.ErrorIs(t, err, ErrReleaseBinaryMetadata)
	require.NotContains(t, err.Error(), hostile)
	require.Empty(t, result.Binary())
}

func TestVerifyMARSReleaseArchiveFromEnvironment(t *testing.T) {
	archivePath := os.Getenv("MARS_TEST_RELEASE_ARCHIVE")
	if archivePath == "" {
		t.Skip("set MARS_TEST_RELEASE_ARCHIVE to run the clean release archive proof")
	}
	tag := os.Getenv("MARS_TEST_RELEASE_TAG")
	commit := os.Getenv("MARS_TEST_RELEASE_COMMIT")
	goos := os.Getenv("MARS_TEST_RELEASE_GOOS")
	goarch := os.Getenv("MARS_TEST_RELEASE_GOARCH")
	archive, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	name, _, _, ok := marsReleaseArchiveIdentity(tag, commit, goos, goarch)
	require.True(t, ok)
	result, err := VerifyMARSReleaseArchive(archive, signedChecksumsForArchive(name, archive, tag, commit), tag, commit, goos, goarch)
	require.NoError(t, err)
	require.NotEmpty(t, result.Binary())
}

type releaseTarEntry struct {
	header tar.Header
	body   []byte
}

func canonicalMARSReleaseArchive(t *testing.T, binary, trailing []byte) []byte {
	t.Helper()
	return releaseTarGzip(t, canonicalMARSReleaseEntries(binary), trailing)
}

func canonicalMARSReleaseEntries(binary []byte) []releaseTarEntry {
	return []releaseTarEntry{
		regularReleaseTarEntry("LICENSE", 0o644, []byte("license")),
		regularReleaseTarEntry("NOTICE", 0o644, []byte("notice")),
		regularReleaseTarEntry("THIRD_PARTY_NOTICES", 0o644, []byte("third-party notices")),
		regularReleaseTarEntry("mars", 0o755, binary),
	}
}

func regularReleaseTarEntry(name string, mode int64, body []byte) releaseTarEntry {
	return releaseTarEntry{header: tar.Header{
		Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg,
		Uid: 0, Gid: 0, Uname: "root", Gname: "root", ModTime: time.Unix(1, 0).UTC(), Format: tar.FormatUSTAR,
	}, body: append([]byte(nil), body...)}
}

func cloneReleaseTarEntries(entries []releaseTarEntry) []releaseTarEntry {
	cloned := make([]releaseTarEntry, len(entries))
	for index := range entries {
		cloned[index] = releaseTarEntry{header: entries[index].header, body: append([]byte(nil), entries[index].body...)}
		if entries[index].header.PAXRecords != nil {
			cloned[index].header.PAXRecords = map[string]string{}
			for key, value := range entries[index].header.PAXRecords {
				cloned[index].header.PAXRecords[key] = value
			}
		}
	}
	return cloned
}

func releaseTarGzip(t *testing.T, entries []releaseTarEntry, trailing []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	gz := gzip.NewWriter(&output)
	gz.Header.ModTime = time.Time{}
	tw := tar.NewWriter(gz)
	for index := range entries {
		header := entries[index].header
		require.NoError(t, tw.WriteHeader(&header))
		if len(entries[index].body) != 0 {
			_, err := tw.Write(entries[index].body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	if len(trailing) != 0 {
		_, err := gz.Write(trailing)
		require.NoError(t, err)
	}
	require.NoError(t, gz.Close())
	return output.Bytes()
}

func tarWithDeclaredOversizedBinary(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	tw := tar.NewWriter(&output)
	entries := canonicalMARSReleaseEntries([]byte("binary"))
	for index := 0; index < len(entries)-1; index++ {
		header := entries[index].header
		require.NoError(t, tw.WriteHeader(&header))
		_, err := tw.Write(entries[index].body)
		require.NoError(t, err)
	}
	header := entries[len(entries)-1].header
	header.Size = maxReleaseBinaryBytes + 1
	require.NoError(t, tw.WriteHeader(&header))
	return output.Bytes()
}

func gzipBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	gz := gzip.NewWriter(&output)
	_, err := gz.Write(raw)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return output.Bytes()
}

func signedChecksumsForArchive(name string, archive []byte, tag, fullCommit string) SignedChecksums {
	digests := make(map[string][sha256.Size]byte, expectedSignedChecksumCount)
	for _, expected := range expectedMARSArchiveChecksumNames(strings.TrimPrefix(tag, "v")) {
		digests[expected] = sha256.Sum256([]byte(expected))
	}
	digests[name] = sha256.Sum256(archive)
	return SignedChecksums{digests: digests, tag: tag, fullCommit: fullCommit}
}

func TestSignedChecksumsForArchiveUsesRequestedTagVersion(t *testing.T) {
	tag := "v0.69.1"
	name, _, _, ok := marsReleaseArchiveIdentity(tag, testReleaseCommit, "darwin", "arm64")
	require.True(t, ok)
	archive := []byte("archive")
	checksums := signedChecksumsForArchive(name, archive, tag, testReleaseCommit)
	require.Equal(t, expectedSignedChecksumCount, checksums.Len())
	require.NotContains(t, checksums.digests, "mars_0.69.0_darwin_arm64.tar.gz")
	require.Equal(t, sha256.Sum256(archive), checksums.digests[name])
}

func validMARSReleaseBuildInfo(goos, goarch, archSetting, archValue string) *debug.BuildInfo {
	return &debug.BuildInfo{
		GoVersion: releaseGoVersion,
		Path:      DefaultPackage,
		Main:      debug.Module{Path: releaseModulePath},
		Settings: []debug.BuildSetting{
			{Key: "GOOS", Value: goos}, {Key: "GOARCH", Value: goarch}, {Key: archSetting, Value: archValue},
			{Key: "CGO_ENABLED", Value: "0"}, {Key: "-trimpath", Value: "true"}, {Key: "vcs", Value: "git"},
			{Key: "vcs.revision", Value: testReleaseCommit}, {Key: "vcs.modified", Value: "false"},
			{Key: "vcs.time", Value: "2026-07-22T00:00:00Z"},
		},
	}
}

func setBuildSetting(info *debug.BuildInfo, key, value string) {
	for index := range info.Settings {
		if info.Settings[index].Key == key {
			info.Settings[index].Value = value
			return
		}
	}
	info.Settings = append(info.Settings, debug.BuildSetting{Key: key, Value: value})
}

func removeBuildSetting(info *debug.BuildInfo, key string) {
	for index := range info.Settings {
		if info.Settings[index].Key == key {
			info.Settings = append(info.Settings[:index], info.Settings[index+1:]...)
			return
		}
	}
}
