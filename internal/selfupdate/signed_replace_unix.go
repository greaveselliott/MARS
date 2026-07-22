//go:build darwin || linux

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
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	signedReplaceLockName        = ".mars-update.lock"
	signedReplaceTransactionName = ".mars-update.transaction"
	signedReplaceCandidateName   = "candidate"
	signedReplaceBackupName      = "backup"
	signedReplaceRestoreName     = "restore"
)

type signedUnixIdentity struct {
	dev   uint64
	ino   uint64
	mode  uint32
	uid   uint32
	nlink uint64
	size  int64
}

type signedExistingBinary struct {
	exists   bool
	bytes    []byte
	identity signedUnixIdentity
}

type signedUnixTransaction struct {
	installPath     string
	installFD       int
	installIdentity signedUnixIdentity
	lockFD          int
	lockIdentity    signedUnixIdentity
	txFD            int
	txIdentity      signedUnixIdentity
	deps            signedReplaceDependencies
}

func replaceVerifiedMARSReleasePlatform(ctx context.Context, installDir string, candidate signedReplacementCandidate, deps signedReplaceDependencies) (signedReplaceResult, error) {
	deps = completeSignedReplaceDependencies(deps)
	if ctx.Err() != nil {
		return signedReplaceResult{}, ErrSignedReplaceCancelled
	}
	installFD, installIdentity, err := openSignedInstallDirectory(installDir)
	if err != nil {
		return signedReplaceResult{}, ErrSignedReplaceUnsafe
	}
	tx := &signedUnixTransaction{installPath: installDir, installFD: installFD, installIdentity: installIdentity, lockFD: -1, txFD: -1, deps: deps}
	defer tx.closeDescriptors()

	lockFD, lockIdentity, err := lockSignedInstallDirectory(installFD, deps)
	if err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return signedReplaceResult{}, ErrSignedReplaceBusy
		}
		return signedReplaceResult{}, ErrSignedReplaceUnsafe
	}
	tx.lockFD, tx.lockIdentity = lockFD, lockIdentity
	if ctx.Err() != nil {
		return signedReplaceResult{}, ErrSignedReplaceCancelled
	}

	prior, err := readSignedExistingBinary(installFD, deps)
	if err != nil {
		return signedReplaceResult{}, ErrSignedReplaceUnsafe
	}
	txFD, txIdentity, err := createSignedTransactionDirectory(installFD, installIdentity, deps)
	if err != nil {
		if errors.Is(err, unix.EEXIST) || errors.Is(err, errSignedReplaceResidue) {
			return signedReplaceResult{}, ErrSignedReplaceRecovery
		}
		return signedReplaceResult{}, ErrSignedReplaceFailed
	}
	tx.txFD, tx.txIdentity = txFD, txIdentity

	failBeforeCommit := func(reason error) (signedReplaceResult, error) {
		if !tx.priorStateStillExact(prior) || !tx.cleanup() || !tx.priorStateStillExact(prior) {
			return signedReplaceResult{}, ErrSignedReplaceRecovery
		}
		return signedReplaceResult{}, reason
	}

	stagedBytes, stagedIdentity, err := writeAndVerifySignedStageFile(txFD, signedReplaceCandidateName, candidate.binary, deps)
	if err != nil || !bytes.Equal(stagedBytes, candidate.binary) || !verifySignedCandidateBuildInfo(stagedBytes, candidate, deps) {
		return failBeforeCommit(ErrSignedReplaceFailed)
	}
	if err := runSignedReplaceCheckpoint(deps, signedReplaceAfterCandidate); err != nil {
		return failBeforeCommit(ErrSignedReplaceFailed)
	}
	if prior.exists {
		backup, _, backupErr := writeAndVerifySignedStageFile(txFD, signedReplaceBackupName, prior.bytes, deps)
		if backupErr != nil || !bytes.Equal(backup, prior.bytes) {
			return failBeforeCommit(ErrSignedReplaceFailed)
		}
		if err := runSignedReplaceCheckpoint(deps, signedReplaceAfterBackup); err != nil {
			return failBeforeCommit(ErrSignedReplaceFailed)
		}
	}

	current, currentErr := readSignedExistingBinary(installFD, deps)
	if currentErr != nil || !sameSignedExistingBinary(prior, current) || !tx.bindingsStillExact() {
		return failBeforeCommit(ErrSignedReplaceRecovery)
	}
	finalMode := uint32(0o755)
	if prior.exists {
		finalMode = prior.identity.mode & 0o7777
	}
	stagedIdentity, err = promoteSignedStageFile(txFD, signedReplaceCandidateName, stagedIdentity, finalMode, deps)
	if err != nil {
		return failBeforeCommit(ErrSignedReplaceFailed)
	}
	if ctx.Err() != nil {
		return failBeforeCommit(ErrSignedReplaceCancelled)
	}
	if err := runSignedReplaceCheckpoint(deps, signedReplaceBeforeCommit); err != nil {
		if ctx.Err() != nil {
			return failBeforeCommit(ErrSignedReplaceCancelled)
		}
		return failBeforeCommit(ErrSignedReplaceFailed)
	}
	if ctx.Err() != nil {
		return failBeforeCommit(ErrSignedReplaceCancelled)
	}
	if !verifyStagedSignedCandidate(txFD, candidate, stagedIdentity, finalMode, deps) || !tx.priorStateStillExact(prior) || !tx.bindingsStillExact() {
		return failBeforeCommit(ErrSignedReplaceRecovery)
	}

	if prior.exists {
		err = deps.renameAt(txFD, signedReplaceCandidateName, installFD, DefaultBinary)
	} else {
		err = deps.linkAt(txFD, signedReplaceCandidateName, installFD, DefaultBinary, 0)
	}
	if err != nil {
		return failBeforeCommit(ErrSignedReplaceFailed)
	}
	expectedLinks := uint64(1)
	if !prior.exists {
		expectedLinks = 2
	}
	compensate := func() (signedReplaceResult, error) {
		return compensateSignedReplacement(tx, prior, candidate, stagedIdentity, finalMode, expectedLinks)
	}
	if err := runSignedReplaceCheckpoint(deps, signedReplaceAfterCommit); err != nil {
		return compensate()
	}
	if err := deps.syncFD(installFD); err != nil {
		return compensate()
	}
	if err := runSignedReplaceCheckpoint(deps, signedReplaceAfterCommitSync); err != nil {
		return compensate()
	}
	if !verifyInstalledSignedCandidate(installFD, candidate, stagedIdentity, finalMode, expectedLinks, deps) {
		return compensate()
	}
	if err := runSignedReplaceCheckpoint(deps, signedReplaceAfterCommitVerify); err != nil {
		return compensate()
	}
	if err := runSignedReplaceCheckpoint(deps, signedReplaceBeforeCleanup); err != nil {
		return compensate()
	}
	if !verifyInstalledSignedCandidate(installFD, candidate, stagedIdentity, finalMode, expectedLinks, deps) || !tx.cleanup() || !rebindSignedInstallDirectory(installDir, installIdentity) ||
		!verifyInstalledSignedCandidate(installFD, candidate, stagedIdentity, finalMode, 1, deps) {
		return signedReplaceResult{}, ErrSignedReplaceRecovery
	}
	return signedReplaceResult{tag: candidate.tag, fullCommit: candidate.fullCommit, replacedExisting: prior.exists}, nil
}

var errSignedReplaceResidue = errors.New("signed replacement residue")

func completeSignedReplaceDependencies(deps signedReplaceDependencies) signedReplaceDependencies {
	if deps.syncFD == nil {
		deps.syncFD = unix.Fsync
	}
	if deps.renameAt == nil {
		deps.renameAt = unix.Renameat
	}
	if deps.linkAt == nil {
		deps.linkAt = unix.Linkat
	}
	if deps.unlinkAt == nil {
		deps.unlinkAt = unix.Unlinkat
	}
	return deps
}

func openSignedInstallDirectory(path string) (int, signedUnixIdentity, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.IndexByte(path, 0) >= 0 || path == string(filepath.Separator) {
		return -1, signedUnixIdentity{}, unix.EINVAL
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, signedUnixIdentity{}, err
	}
	components := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for index, component := range components {
		if component == "" || component == "." || component == ".." {
			_ = unix.Close(fd)
			return -1, signedUnixIdentity{}, unix.EINVAL
		}
		next, openErr := unix.Openat(fd, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if openErr != nil {
			return -1, signedUnixIdentity{}, openErr
		}
		identity, statErr := signedIdentityForFD(next)
		final := index == len(components)-1
		if statErr != nil || identity.mode&unix.S_IFMT != unix.S_IFDIR || identity.mode&0o022 != 0 ||
			(identity.uid != 0 && identity.uid != uint32(unix.Geteuid())) ||
			(final && (identity.uid != uint32(unix.Geteuid()) || identity.mode&0o700 != 0o700)) {
			_ = unix.Close(next)
			return -1, signedUnixIdentity{}, unix.EPERM
		}
		fd = next
	}
	identity, err := signedIdentityForFD(fd)
	if err != nil {
		_ = unix.Close(fd)
		return -1, signedUnixIdentity{}, err
	}
	return fd, identity, nil
}

func rebindSignedInstallDirectory(path string, expected signedUnixIdentity) bool {
	fd, identity, err := openSignedInstallDirectory(path)
	if fd >= 0 {
		_ = unix.Close(fd)
	}
	return err == nil && sameSignedObject(identity, expected)
}

func lockSignedInstallDirectory(installFD int, deps signedReplaceDependencies) (int, signedUnixIdentity, error) {
	fd, err := unix.Openat(installFD, signedReplaceLockName, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	created := err == nil
	if errors.Is(err, unix.EEXIST) {
		fd, err = unix.Openat(installFD, signedReplaceLockName, unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	}
	if err != nil {
		return -1, signedUnixIdentity{}, err
	}
	identity, statErr := signedIdentityForFD(fd)
	if statErr != nil || !safeSignedRegular(identity, 0o600, 1) || identity.size != 0 {
		_ = unix.Close(fd)
		return -1, signedUnixIdentity{}, unix.EPERM
	}
	if created && (deps.syncFD(fd) != nil || deps.syncFD(installFD) != nil) {
		_ = unix.Close(fd)
		return -1, signedUnixIdentity{}, unix.EIO
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = unix.Close(fd)
		return -1, signedUnixIdentity{}, err
	}
	if !signedEntryMatchesAt(installFD, signedReplaceLockName, identity) {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
		return -1, signedUnixIdentity{}, unix.EPERM
	}
	return fd, identity, nil
}

func createSignedTransactionDirectory(installFD int, installIdentity signedUnixIdentity, deps signedReplaceDependencies) (int, signedUnixIdentity, error) {
	if err := unix.Mkdirat(installFD, signedReplaceTransactionName, 0o700); err != nil {
		return -1, signedUnixIdentity{}, err
	}
	txFD, err := unix.Openat(installFD, signedReplaceTransactionName, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = deps.unlinkAt(installFD, signedReplaceTransactionName, unix.AT_REMOVEDIR)
		_ = deps.syncFD(installFD)
		return -1, signedUnixIdentity{}, errSignedReplaceResidue
	}
	identity, statErr := signedIdentityForFD(txFD)
	if statErr != nil || identity.mode&unix.S_IFMT != unix.S_IFDIR || identity.mode&0o7777 != 0o700 ||
		identity.uid != uint32(unix.Geteuid()) || identity.dev != installIdentity.dev || deps.syncFD(txFD) != nil || deps.syncFD(installFD) != nil {
		_ = unix.Close(txFD)
		_ = deps.unlinkAt(installFD, signedReplaceTransactionName, unix.AT_REMOVEDIR)
		_ = deps.syncFD(installFD)
		return -1, signedUnixIdentity{}, errSignedReplaceResidue
	}
	return txFD, identity, nil
}

func writeAndVerifySignedStageFile(dirFD int, name string, content []byte, deps signedReplaceDependencies) ([]byte, signedUnixIdentity, error) {
	if len(content) == 0 || len(content) > maxReleaseBinaryBytes {
		return nil, signedUnixIdentity{}, unix.EFBIG
	}
	fd, err := unix.Openat(dirFD, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, signedUnixIdentity{}, err
	}
	if err = writeAllSignedFD(fd, content); err == nil {
		err = deps.syncFD(fd)
	}
	if closeErr := unix.Close(fd); err == nil {
		err = closeErr
	}
	if err != nil || deps.syncFD(dirFD) != nil {
		return nil, signedUnixIdentity{}, unix.EIO
	}
	read, identity, err := readSignedRegularAt(dirFD, name, maxReleaseBinaryBytes)
	if err != nil || !safeSignedRegular(identity, 0o600, 1) {
		return nil, signedUnixIdentity{}, unix.EPERM
	}
	return read, identity, nil
}

func promoteSignedStageFile(dirFD int, name string, expected signedUnixIdentity, mode uint32, deps signedReplaceDependencies) (signedUnixIdentity, error) {
	fd, err := unix.Openat(dirFD, name, unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return signedUnixIdentity{}, err
	}
	defer unix.Close(fd)
	identity, err := signedIdentityForFD(fd)
	if err != nil || !sameSignedObject(identity, expected) || unix.Fchmod(fd, mode) != nil || deps.syncFD(fd) != nil || deps.syncFD(dirFD) != nil {
		return signedUnixIdentity{}, unix.EIO
	}
	identity, err = signedIdentityForFD(fd)
	if err != nil || !safeSignedRegular(identity, mode, 1) {
		return signedUnixIdentity{}, unix.EPERM
	}
	return identity, nil
}

func readSignedExistingBinary(installFD int, deps signedReplaceDependencies) (signedExistingBinary, error) {
	fd, err := unix.Openat(installFD, DefaultBinary, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		if signedDestinationAbsent(installFD) {
			return signedExistingBinary{}, nil
		}
		return signedExistingBinary{}, unix.EPERM
	}
	if err != nil {
		return signedExistingBinary{}, err
	}
	defer unix.Close(fd)
	identity, err := signedIdentityForFD(fd)
	if err != nil || !safeSignedExisting(identity) || !signedEntryMatchesAt(installFD, DefaultBinary, identity) {
		return signedExistingBinary{}, unix.EPERM
	}
	content, err := readAllSignedFD(fd, int(identity.size))
	if err != nil || !canonicalExistingMARSBinary(content, deps.readBuildInfo) {
		return signedExistingBinary{}, unix.EPERM
	}
	return signedExistingBinary{exists: true, bytes: content, identity: identity}, nil
}

func readSignedRegularAt(dirFD int, name string, limit int) ([]byte, signedUnixIdentity, error) {
	fd, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, signedUnixIdentity{}, err
	}
	defer unix.Close(fd)
	identity, err := signedIdentityForFD(fd)
	if err != nil || identity.mode&unix.S_IFMT != unix.S_IFREG || identity.size <= 0 || identity.size > int64(limit) || !signedEntryMatchesAt(dirFD, name, identity) {
		return nil, signedUnixIdentity{}, unix.EPERM
	}
	content, err := readAllSignedFD(fd, int(identity.size))
	return content, identity, err
}

func verifyInstalledSignedCandidate(installFD int, candidate signedReplacementCandidate, expected signedUnixIdentity, mode uint32, nlink uint64, deps signedReplaceDependencies) bool {
	content, identity, err := readSignedRegularAt(installFD, DefaultBinary, maxReleaseBinaryBytes)
	return err == nil && identity.dev == expected.dev && identity.ino == expected.ino && safeSignedRegular(identity, mode, nlink) &&
		bytes.Equal(content, candidate.binary) && verifySignedCandidateBuildInfo(content, candidate, deps)
}

func verifyStagedSignedCandidate(txFD int, candidate signedReplacementCandidate, expected signedUnixIdentity, mode uint32, deps signedReplaceDependencies) bool {
	content, identity, err := readSignedRegularAt(txFD, signedReplaceCandidateName, maxReleaseBinaryBytes)
	return err == nil && identity.dev == expected.dev && identity.ino == expected.ino && safeSignedRegular(identity, mode, 1) &&
		bytes.Equal(content, candidate.binary) && verifySignedCandidateBuildInfo(content, candidate, deps)
}

func verifySignedCandidateBuildInfo(content []byte, candidate signedReplacementCandidate, deps signedReplaceDependencies) bool {
	_, archSetting, archValue, ok := marsReleaseArchiveIdentity(candidate.tag, candidate.fullCommit, runtime.GOOS, runtime.GOARCH)
	return ok && verifyMARSReleaseBuildInfo(content, candidate.fullCommit, runtime.GOOS, runtime.GOARCH, archSetting, archValue, deps.readBuildInfo) == nil
}

func compensateSignedReplacement(tx *signedUnixTransaction, prior signedExistingBinary, candidate signedReplacementCandidate, candidateIdentity signedUnixIdentity, finalMode uint32, candidateLinks uint64) (signedReplaceResult, error) {
	if runSignedReplaceCheckpoint(tx.deps, signedReplaceBeforeCompensate) != nil ||
		!verifyInstalledSignedCandidate(tx.installFD, candidate, candidateIdentity, finalMode, candidateLinks, tx.deps) || !tx.bindingsStillExact() {
		return signedReplaceResult{}, ErrSignedReplaceRecovery
	}
	if prior.exists {
		backup, backupIdentity, err := readSignedRegularAt(tx.txFD, signedReplaceBackupName, maxReleaseBinaryBytes)
		if err != nil || !safeSignedRegular(backupIdentity, 0o600, 1) || !bytes.Equal(backup, prior.bytes) {
			return signedReplaceResult{}, ErrSignedReplaceRecovery
		}
		_, restoreIdentity, err := writeAndVerifySignedStageFile(tx.txFD, signedReplaceRestoreName, backup, tx.deps)
		if err != nil {
			return signedReplaceResult{}, ErrSignedReplaceRecovery
		}
		restoreIdentity, err = promoteSignedStageFile(tx.txFD, signedReplaceRestoreName, restoreIdentity, prior.identity.mode&0o7777, tx.deps)
		if err != nil ||
			runSignedReplaceCheckpoint(tx.deps, signedReplaceBeforeCompensateCommit) != nil ||
			!verifyStagedSignedRestore(tx.txFD, prior, restoreIdentity) ||
			!verifyInstalledSignedCandidate(tx.installFD, candidate, candidateIdentity, finalMode, candidateLinks, tx.deps) || !tx.bindingsStillExact() ||
			tx.deps.renameAt(tx.txFD, signedReplaceRestoreName, tx.installFD, DefaultBinary) != nil || tx.deps.syncFD(tx.installFD) != nil {
			return signedReplaceResult{}, ErrSignedReplaceRecovery
		}
	} else {
		if runSignedReplaceCheckpoint(tx.deps, signedReplaceBeforeCompensateCommit) != nil ||
			!verifyInstalledSignedCandidate(tx.installFD, candidate, candidateIdentity, finalMode, candidateLinks, tx.deps) || !tx.bindingsStillExact() ||
			tx.deps.unlinkAt(tx.installFD, DefaultBinary, 0) != nil || tx.deps.syncFD(tx.installFD) != nil || !signedDestinationAbsent(tx.installFD) {
			return signedReplaceResult{}, ErrSignedReplaceRecovery
		}
	}
	if runSignedReplaceCheckpoint(tx.deps, signedReplaceAfterCompensateCommit) != nil || !tx.priorContentRestored(prior) {
		return signedReplaceResult{}, ErrSignedReplaceRecovery
	}
	if !tx.cleanup() || !tx.priorContentRestored(prior) {
		return signedReplaceResult{}, ErrSignedReplaceRecovery
	}
	return signedReplaceResult{}, ErrSignedReplaceFailed
}

func verifyStagedSignedRestore(txFD int, prior signedExistingBinary, expected signedUnixIdentity) bool {
	content, identity, err := readSignedRegularAt(txFD, signedReplaceRestoreName, maxReleaseBinaryBytes)
	return prior.exists && err == nil && identity.dev == expected.dev && identity.ino == expected.ino &&
		safeSignedRegular(identity, prior.identity.mode&0o7777, 1) && bytes.Equal(content, prior.bytes)
}

func (tx *signedUnixTransaction) bindingsStillExact() bool {
	return rebindSignedInstallDirectory(tx.installPath, tx.installIdentity) &&
		signedLockEntryStillExact(tx.installFD, tx.lockIdentity) &&
		signedTransactionEntryStillExact(tx.installFD, tx.txIdentity, tx.installIdentity)
}

func (tx *signedUnixTransaction) priorStateStillExact(prior signedExistingBinary) bool {
	if !rebindSignedInstallDirectory(tx.installPath, tx.installIdentity) {
		return false
	}
	current, err := readSignedExistingBinary(tx.installFD, tx.deps)
	return err == nil && sameSignedExistingBinary(prior, current)
}

func (tx *signedUnixTransaction) priorContentRestored(prior signedExistingBinary) bool {
	if !rebindSignedInstallDirectory(tx.installPath, tx.installIdentity) {
		return false
	}
	current, err := readSignedExistingBinary(tx.installFD, tx.deps)
	if err != nil || prior.exists != current.exists {
		return false
	}
	return !prior.exists || (prior.identity.mode == current.identity.mode && bytes.Equal(prior.bytes, current.bytes))
}

func (tx *signedUnixTransaction) cleanup() bool {
	if tx.txFD < 0 || !signedTransactionEntryStillExact(tx.installFD, tx.txIdentity, tx.installIdentity) {
		return false
	}
	for _, name := range []string{signedReplaceCandidateName, signedReplaceRestoreName, signedReplaceBackupName} {
		if err := tx.deps.unlinkAt(tx.txFD, name, 0); err != nil && !errors.Is(err, unix.ENOENT) {
			return false
		}
	}
	if tx.deps.syncFD(tx.txFD) != nil || unix.Close(tx.txFD) != nil {
		tx.txFD = -1
		return false
	}
	tx.txFD = -1
	if tx.deps.unlinkAt(tx.installFD, signedReplaceTransactionName, unix.AT_REMOVEDIR) != nil || tx.deps.syncFD(tx.installFD) != nil {
		return false
	}
	return true
}

func (tx *signedUnixTransaction) closeDescriptors() {
	if tx.txFD >= 0 {
		_ = unix.Close(tx.txFD)
		tx.txFD = -1
	}
	if tx.lockFD >= 0 {
		_ = unix.Flock(tx.lockFD, unix.LOCK_UN)
		_ = unix.Close(tx.lockFD)
		tx.lockFD = -1
	}
	if tx.installFD >= 0 {
		_ = unix.Close(tx.installFD)
		tx.installFD = -1
	}
}

func signedIdentityForFD(fd int) (signedUnixIdentity, error) {
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return signedUnixIdentity{}, err
	}
	return signedIdentityFromStat(&stat), nil
}

func signedIdentityFromStat(stat *unix.Stat_t) signedUnixIdentity {
	return signedUnixIdentity{dev: uint64(stat.Dev), ino: uint64(stat.Ino), mode: uint32(stat.Mode), uid: stat.Uid, nlink: uint64(stat.Nlink), size: stat.Size}
}

func signedEntryMatchesAt(dirFD int, name string, expected signedUnixIdentity) bool {
	var stat unix.Stat_t
	return unix.Fstatat(dirFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW) == nil && sameSignedObject(signedIdentityFromStat(&stat), expected)
}

func signedLockEntryStillExact(installFD int, expected signedUnixIdentity) bool {
	identity, ok := signedIdentityAt(installFD, signedReplaceLockName)
	return ok && sameSignedObject(identity, expected) && safeSignedRegular(identity, 0o600, 1) && identity.size == 0
}

func signedTransactionEntryStillExact(installFD int, expected, install signedUnixIdentity) bool {
	identity, ok := signedIdentityAt(installFD, signedReplaceTransactionName)
	return ok && sameSignedObject(identity, expected) && identity.mode&unix.S_IFMT == unix.S_IFDIR &&
		identity.mode&0o7777 == 0o700 && identity.uid == uint32(unix.Geteuid()) && identity.dev == install.dev
}

func signedIdentityAt(dirFD int, name string) (signedUnixIdentity, bool) {
	var stat unix.Stat_t
	if unix.Fstatat(dirFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW) != nil {
		return signedUnixIdentity{}, false
	}
	return signedIdentityFromStat(&stat), true
}

func sameSignedObject(left, right signedUnixIdentity) bool {
	return left.dev == right.dev && left.ino == right.ino && left.mode&unix.S_IFMT == right.mode&unix.S_IFMT && left.uid == right.uid
}

func safeSignedRegular(identity signedUnixIdentity, exactMode uint32, nlink uint64) bool {
	return identity.mode&unix.S_IFMT == unix.S_IFREG && identity.mode&0o7777 == exactMode && identity.uid == uint32(unix.Geteuid()) && identity.nlink == nlink
}

func safeSignedExisting(identity signedUnixIdentity) bool {
	mode := identity.mode & 0o7777
	return identity.mode&unix.S_IFMT == unix.S_IFREG && identity.uid == uint32(unix.Geteuid()) && identity.nlink == 1 &&
		identity.size > 0 && identity.size <= maxReleaseBinaryBytes && mode&0o7000 == 0 && mode&0o022 == 0 && mode&0o100 != 0
}

func sameSignedExistingBinary(left, right signedExistingBinary) bool {
	return left.exists == right.exists && (!left.exists ||
		(left.identity.dev == right.identity.dev && left.identity.ino == right.identity.ino && left.identity.mode == right.identity.mode && bytes.Equal(left.bytes, right.bytes)))
}

func signedDestinationAbsent(installFD int) bool {
	var stat unix.Stat_t
	return errors.Is(unix.Fstatat(installFD, DefaultBinary, &stat, unix.AT_SYMLINK_NOFOLLOW), unix.ENOENT)
}

func writeAllSignedFD(fd int, content []byte) error {
	for len(content) != 0 {
		n, err := unix.Write(fd, content)
		if err != nil {
			return err
		}
		if n <= 0 {
			return unix.EIO
		}
		content = content[n:]
	}
	return nil
}

func readAllSignedFD(fd, size int) ([]byte, error) {
	content := make([]byte, size)
	for offset := 0; offset < size; {
		n, err := unix.Pread(fd, content[offset:], int64(offset))
		if err != nil {
			return nil, err
		}
		if n <= 0 {
			return nil, unix.EIO
		}
		offset += n
	}
	return content, nil
}
