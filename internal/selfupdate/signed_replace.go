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
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"io"
	"runtime"
)

var (
	ErrSignedReplaceAdmission = errors.New("signed release replacement: verified candidate admission failed; reacquire the release before retrying")
	ErrSignedReplaceUnsafe    = errors.New("signed release replacement: the install directory or current binary is unsafe; repair its ownership, permissions, or file type before retrying")
	ErrSignedReplaceBusy      = errors.New("signed release replacement: another update transaction is active; wait for it to finish before retrying")
	ErrSignedReplaceCancelled = errors.New("signed release replacement: cancelled before commit; the current binary was not replaced")
	ErrSignedReplaceFailed    = errors.New("signed release replacement: the transaction failed and the prior binary state was preserved; retry the update")
	ErrSignedReplaceRecovery  = errors.New("signed release replacement: local recovery is required; do not run mars, preserve .mars-update.transaction, and run make install from a trusted source checkout before retrying")
	ErrSignedReplacePlatform  = errors.New("signed release replacement: durable replacement is unsupported on this platform; install from a supported Darwin or Linux host")
)

type signedReplaceCheckpoint uint8

const (
	signedReplaceAfterCandidate signedReplaceCheckpoint = iota + 1
	signedReplaceAfterBackup
	signedReplaceBeforeCommit
	signedReplaceAfterCommit
	signedReplaceAfterCommitSync
	signedReplaceAfterCommitVerify
	signedReplaceBeforeCompensate
	signedReplaceBeforeCompensateCommit
	signedReplaceAfterCompensateCommit
	signedReplaceBeforeCleanup
)

type signedReplaceDependencies struct {
	readBuildInfo func(io.ReaderAt) (*buildinfo.BuildInfo, error)
	checkpoint    func(signedReplaceCheckpoint) error
	syncFD        func(int) error
	renameAt      func(int, string, int, string) error
	linkAt        func(int, string, int, string, int) error
	unlinkAt      func(int, string, int) error
}

type signedReplacementCandidate struct {
	tag        string
	fullCommit string
	binary     []byte
}

type signedReplaceResult struct {
	releaseID        int64
	tag              string
	fullCommit       string
	replacedExisting bool
}

func replaceVerifiedMARSRelease(ctx context.Context, installDir string, download verifiedMARSReleaseDownload) (signedReplaceResult, error) {
	return replaceVerifiedMARSReleaseWithDependencies(ctx, installDir, download, signedReplaceDependencies{readBuildInfo: buildinfo.Read})
}

func replaceVerifiedMARSReleaseWithDependencies(ctx context.Context, installDir string, download verifiedMARSReleaseDownload, deps signedReplaceDependencies) (signedReplaceResult, error) {
	if ctx == nil || deps.readBuildInfo == nil {
		return signedReplaceResult{}, ErrSignedReplaceAdmission
	}
	candidate, err := admitSignedReplacement(download, deps.readBuildInfo)
	if err != nil {
		return signedReplaceResult{}, err
	}
	result, err := replaceVerifiedMARSReleasePlatform(ctx, installDir, candidate, deps)
	if err != nil {
		return signedReplaceResult{}, err
	}
	result.releaseID = download.releaseID
	return result, nil
}

func admitSignedReplacement(download verifiedMARSReleaseDownload, readBuildInfo func(io.ReaderAt) (*buildinfo.BuildInfo, error)) (signedReplacementCandidate, error) {
	archiveName, archSetting, archValue, ok := marsReleaseArchiveIdentity(download.tag, download.fullCommit, runtime.GOOS, runtime.GOARCH)
	if !ok || download.releaseID <= 0 || download.archiveName != archiveName || len(download.candidate) == 0 || len(download.candidate) > maxReleaseBinaryBytes {
		return signedReplacementCandidate{}, ErrSignedReplaceAdmission
	}
	binary := bytes.Clone(download.candidate)
	if err := verifyMARSReleaseBuildInfo(binary, download.fullCommit, runtime.GOOS, runtime.GOARCH, archSetting, archValue, readBuildInfo); err != nil {
		return signedReplacementCandidate{}, ErrSignedReplaceAdmission
	}
	return signedReplacementCandidate{tag: download.tag, fullCommit: download.fullCommit, binary: bytes.Clone(binary)}, nil
}

func canonicalExistingMARSBinary(binary []byte, readBuildInfo func(io.ReaderAt) (*buildinfo.BuildInfo, error)) bool {
	if len(binary) == 0 || len(binary) > maxReleaseBinaryBytes || readBuildInfo == nil {
		return false
	}
	info, err := readBuildInfo(bytes.NewReader(binary))
	return err == nil && info != nil && info.Path == DefaultPackage && info.Main.Path == releaseModulePath && info.Main.Replace == nil
}

func runSignedReplaceCheckpoint(deps signedReplaceDependencies, checkpoint signedReplaceCheckpoint) error {
	if deps.checkpoint == nil {
		return nil
	}
	return deps.checkpoint(checkpoint)
}
