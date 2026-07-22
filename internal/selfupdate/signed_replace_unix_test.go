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
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

const testSignedReplaceCommit = "0123456789abcdef0123456789abcdef01234567"

const testSignedRollbackCommit = "89abcdef0123456789abcdef0123456789abcdef"

func TestReplaceVerifiedMARSReleaseCommitsDurably(t *testing.T) {
	for _, test := range []struct {
		name     string
		existing bool
		mode     os.FileMode
	}{
		{name: "existing", existing: true, mode: 0o750},
		{name: "absent", mode: 0o755},
	} {
		t.Run(test.name, func(t *testing.T) {
			installDir := signedReplaceTestDir(t)
			prior := []byte("prior-mars-binary")
			if test.existing {
				require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, test.mode))
			}
			download := signedReplaceTestDownload(t, []byte("verified-release-candidate"))
			ctx, cancel := context.WithCancel(context.Background())
			candidateObserved := false
			backupObserved := false
			deferredCancellationReached := false
			deps := signedReplaceTestDependencies()
			deps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
				switch checkpoint {
				case signedReplaceAfterCandidate:
					candidateObserved = true
					assertSignedReplacePathMode(t, filepath.Join(installDir, signedReplaceTransactionName), os.ModeDir|0o700)
					assertSignedReplacePathMode(t, filepath.Join(installDir, signedReplaceTransactionName, signedReplaceCandidateName), 0o600)
				case signedReplaceAfterBackup:
					backupObserved = true
					assertSignedReplacePathMode(t, filepath.Join(installDir, signedReplaceTransactionName, signedReplaceBackupName), 0o600)
				case signedReplaceAfterCommit:
					deferredCancellationReached = true
					cancel()
				}
				return nil
			}

			result, err := replaceVerifiedMARSReleaseWithDependencies(ctx, installDir, download, deps)
			require.NoError(t, err)
			require.True(t, candidateObserved, "the candidate permission checkpoint must run")
			require.Equal(t, test.existing, backupObserved, "only replacement of an existing binary may stage a backup")
			require.True(t, deferredCancellationReached, "the test must cancel only after the commit barrier")
			require.Equal(t, download.releaseID, result.releaseID)
			require.Equal(t, download.tag, result.tag)
			require.Equal(t, download.fullCommit, result.fullCommit)
			require.Equal(t, test.existing, result.replacedExisting)
			require.Equal(t, download.candidate, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
			assertSignedReplacePathMode(t, filepath.Join(installDir, DefaultBinary), test.mode)
			assertSignedReplacePathMode(t, filepath.Join(installDir, signedReplaceLockName), 0o600)
			require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
			assertSignedReplaceNlink(t, filepath.Join(installDir, DefaultBinary), 1)
		})
	}
}

func TestReplaceVerifiedMARSReleasePreverifiedUpdateAndRollbackLifecycle(t *testing.T) {
	installDir := signedReplaceTestDir(t)
	binaryPath := filepath.Join(installDir, DefaultBinary)
	prior := []byte("preverified-v0.69.0-candidate")
	require.NoError(t, os.WriteFile(binaryPath, prior, 0o755))

	newer := signedReplaceTestDownloadFor(t, "v0.69.1", testSignedReplaceCommit, 691, []byte("preverified-v0.69.1-candidate"))
	_, err := replaceVerifiedMARSReleaseExpectedWithDependencies(
		context.Background(), installDir, newer,
		signedPriorExpectation{required: true, digest: sha256.Sum256(prior)},
		signedReplaceTestDependenciesFor(newer.tag, testSignedRollbackCommit),
	)
	require.ErrorIs(t, err, ErrSignedReplaceAdmission)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, binaryPath))
	require.NoFileExists(t, filepath.Join(installDir, signedReplaceLockName))
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))

	updated, err := replaceVerifiedMARSReleaseExpectedWithDependencies(
		context.Background(), installDir, newer,
		signedPriorExpectation{required: true, digest: sha256.Sum256(prior)},
		signedReplaceTestDependenciesFor(newer.tag, newer.fullCommit),
	)
	require.NoError(t, err)
	require.Equal(t, newer.releaseID, updated.releaseID)
	require.Equal(t, newer.tag, updated.tag)
	require.Equal(t, newer.fullCommit, updated.fullCommit)
	require.Equal(t, newer.candidate, mustReadSignedReplaceFile(t, binaryPath))
	require.NotEqual(t, sha256.Sum256(prior), sha256.Sum256(newer.candidate))
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))

	rollback := signedReplaceTestDownloadFor(t, "v0.69.0", testSignedRollbackCommit, 690, prior)
	rolledBack, err := replaceVerifiedMARSReleaseExpectedWithDependencies(
		context.Background(), installDir, rollback,
		signedPriorExpectation{required: true, digest: sha256.Sum256(newer.candidate)},
		signedReplaceTestDependenciesFor(rollback.tag, rollback.fullCommit),
	)
	require.NoError(t, err)
	require.Equal(t, rollback.releaseID, rolledBack.releaseID)
	require.Equal(t, rollback.tag, rolledBack.tag)
	require.Equal(t, rollback.fullCommit, rolledBack.fullCommit)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, binaryPath))
	assertSignedReplacePathMode(t, installDir, os.ModeDir|0o700)
	assertSignedReplacePathMode(t, binaryPath, 0o755)
	assertSignedReplaceNlink(t, binaryPath, 1)
	assertSignedReplacePathMode(t, filepath.Join(installDir, signedReplaceLockName), 0o600)
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
}

func TestReplaceVerifiedMARSReleaseRejectsConcurrentUpdateBeforeMutation(t *testing.T) {
	installDir := signedReplaceTestDir(t)
	prior := []byte("prior-mars-binary")
	require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, 0o755))
	installFD, _, err := openSignedInstallDirectory(installDir)
	require.NoError(t, err)
	defer unix.Close(installFD)
	deps := completeSignedReplaceDependencies(signedReplaceTestDependencies())
	lockFD, _, err := lockSignedInstallDirectory(installFD, deps)
	require.NoError(t, err)
	defer func() {
		_ = unix.Flock(lockFD, unix.LOCK_UN)
		_ = unix.Close(lockFD)
	}()

	result, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), signedReplaceTestDependencies())
	require.ErrorIs(t, err, ErrSignedReplaceBusy)
	require.Zero(t, result)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
}

func TestReplaceVerifiedMARSReleaseRejectsPriorDriftBeforeTransaction(t *testing.T) {
	installDir := signedReplaceTestDir(t)
	priorA := []byte("canonical-prior-a")
	priorB := []byte("canonical-prior-b")
	binaryPath := filepath.Join(installDir, DefaultBinary)
	require.NoError(t, os.WriteFile(binaryPath, priorA, 0o755))
	expected := signedPriorExpectation{required: true, digest: sha256.Sum256(priorA)}
	require.NoError(t, os.WriteFile(binaryPath, priorB, 0o755), "simulate a canonical binary change during acquisition")

	_, err := replaceVerifiedMARSReleaseExpectedWithDependencies(
		context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), expected, signedReplaceTestDependencies(),
	)
	require.ErrorIs(t, err, ErrSignedReplacePriorDrift)
	require.Equal(t, priorB, mustReadSignedReplaceFile(t, binaryPath))
	assertSignedReplacePathMode(t, filepath.Join(installDir, signedReplaceLockName), 0o600)
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
}

func TestReplaceVerifiedMARSReleaseRejectsUnsafeDestinationBeforeStaging(t *testing.T) {
	t.Run("candidate admission", func(t *testing.T) {
		installDir := signedReplaceTestDir(t)
		download := signedReplaceTestDownload(t, []byte("candidate"))
		download.archiveName = "wrong"
		_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, download, signedReplaceTestDependencies())
		require.ErrorIs(t, err, ErrSignedReplaceAdmission)
		require.NoFileExists(t, filepath.Join(installDir, signedReplaceLockName))
		require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
	})

	t.Run("symlink leaf", func(t *testing.T) {
		installDir := signedReplaceTestDir(t)
		target := filepath.Join(installDir, "external")
		require.NoError(t, os.WriteFile(target, []byte("external-sentinel"), 0o755))
		require.NoError(t, os.Symlink(target, filepath.Join(installDir, DefaultBinary)))
		assertSignedReplaceUnsafe(t, installDir)
		require.Equal(t, []byte("external-sentinel"), mustReadSignedReplaceFile(t, target))
	})

	t.Run("directory leaf", func(t *testing.T) {
		installDir := signedReplaceTestDir(t)
		require.NoError(t, os.Mkdir(filepath.Join(installDir, DefaultBinary), 0o700))
		assertSignedReplaceUnsafe(t, installDir)
	})

	t.Run("hard linked leaf", func(t *testing.T) {
		installDir := signedReplaceTestDir(t)
		binary := filepath.Join(installDir, DefaultBinary)
		require.NoError(t, os.WriteFile(binary, []byte("prior"), 0o755))
		require.NoError(t, os.Link(binary, filepath.Join(installDir, "second-link")))
		assertSignedReplaceUnsafe(t, installDir)
		require.Equal(t, []byte("prior"), mustReadSignedReplaceFile(t, binary))
	})

	t.Run("permissive leaf", func(t *testing.T) {
		installDir := signedReplaceTestDir(t)
		binary := filepath.Join(installDir, DefaultBinary)
		require.NoError(t, os.WriteFile(binary, []byte("prior"), 0o755))
		require.NoError(t, os.Chmod(binary, 0o777))
		assertSignedReplaceUnsafe(t, installDir)
		require.Equal(t, []byte("prior"), mustReadSignedReplaceFile(t, binary))
	})

	t.Run("symlink install directory", func(t *testing.T) {
		realDir := signedReplaceTestDir(t)
		linkParent := signedReplaceTestDir(t)
		link := filepath.Join(linkParent, "linked-install")
		require.NoError(t, os.Symlink(realDir, link))
		_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), link, signedReplaceTestDownload(t, []byte("candidate")), signedReplaceTestDependencies())
		require.ErrorIs(t, err, ErrSignedReplaceUnsafe)
		require.NoFileExists(t, filepath.Join(realDir, signedReplaceLockName))
	})

	t.Run("permissive install directory", func(t *testing.T) {
		installDir := signedReplaceTestDir(t)
		require.NoError(t, os.Chmod(installDir, 0o777))
		t.Cleanup(func() { _ = os.Chmod(installDir, 0o700) })
		_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), signedReplaceTestDependencies())
		require.ErrorIs(t, err, ErrSignedReplaceUnsafe)
		require.NoFileExists(t, filepath.Join(installDir, signedReplaceLockName))
	})

	t.Run("world-writable parent", func(t *testing.T) {
		root := signedReplaceTestDir(t)
		unsafeParent := filepath.Join(root, "unsafe-parent")
		require.NoError(t, os.Mkdir(unsafeParent, 0o700))
		require.NoError(t, os.Chmod(unsafeParent, 0o777))
		installDir := filepath.Join(unsafeParent, "install")
		require.NoError(t, os.Mkdir(installDir, 0o700))

		_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), signedReplaceTestDependencies())
		require.ErrorIs(t, err, ErrSignedReplaceUnsafe)
		require.NoFileExists(t, filepath.Join(installDir, signedReplaceLockName))
		require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
	})
}

func TestReplaceVerifiedMARSReleasePreservesPriorOnPreCommitFailure(t *testing.T) {
	installDir := signedReplaceTestDir(t)
	prior := []byte("prior-mars-binary")
	require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, 0o750))
	reached := false
	deps := signedReplaceTestDependencies()
	deps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
		if checkpoint == signedReplaceBeforeCommit {
			reached = true
			return errors.New("injected pre-commit failure")
		}
		return nil
	}

	_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), deps)
	require.ErrorIs(t, err, ErrSignedReplaceFailed)
	require.True(t, reached)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
	assertSignedReplacePathMode(t, filepath.Join(installDir, DefaultBinary), 0o750)
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = replaceVerifiedMARSReleaseWithDependencies(cancelled, installDir, signedReplaceTestDownload(t, []byte("candidate")), signedReplaceTestDependencies())
	require.ErrorIs(t, err, ErrSignedReplaceCancelled)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))

	checkpointContext, checkpointCancel := context.WithCancel(context.Background())
	checkpointReached := false
	checkpointDeps := signedReplaceTestDependencies()
	checkpointDeps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
		if checkpoint == signedReplaceBeforeCommit {
			checkpointReached = true
			checkpointCancel()
		}
		return nil
	}
	_, err = replaceVerifiedMARSReleaseWithDependencies(checkpointContext, installDir, signedReplaceTestDownload(t, []byte("candidate")), checkpointDeps)
	require.ErrorIs(t, err, ErrSignedReplaceCancelled)
	require.True(t, checkpointReached, "the cancellation must occur at the final pre-commit checkpoint")
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
}

func TestReplaceVerifiedMARSReleaseRestoresPriorOnPostCommitFailure(t *testing.T) {
	for _, existing := range []bool{true, false} {
		name := "absent"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			installDir := signedReplaceTestDir(t)
			prior := []byte("prior-mars-binary")
			if existing {
				require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, 0o750))
			}
			commitReached := false
			compensationReached := false
			deps := signedReplaceTestDependencies()
			deps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
				switch checkpoint {
				case signedReplaceAfterCommit:
					commitReached = true
					return errors.New("injected post-commit failure")
				case signedReplaceAfterCompensateCommit:
					compensationReached = true
				}
				return nil
			}

			_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), deps)
			require.ErrorIs(t, err, ErrSignedReplaceFailed)
			require.True(t, commitReached)
			require.True(t, compensationReached)
			if existing {
				require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
				assertSignedReplacePathMode(t, filepath.Join(installDir, DefaultBinary), 0o750)
			} else {
				require.NoFileExists(t, filepath.Join(installDir, DefaultBinary))
			}
			require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))

			_, err = replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("clean-retry")), signedReplaceTestDependencies())
			require.NoError(t, err, "verified compensation must permit a later clean transaction")
		})
	}
}

func TestReplaceVerifiedMARSReleaseRetainsEvidenceWhenCompensationFails(t *testing.T) {
	installDir := signedReplaceTestDir(t)
	prior := []byte("prior-mars-binary")
	require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, 0o755))
	deps := completeSignedReplaceDependencies(signedReplaceTestDependencies())
	realRename := deps.renameAt
	restoreRenameReached := false
	deps.renameAt = func(fromFD int, from string, toFD int, to string) error {
		if from == signedReplaceRestoreName {
			restoreRenameReached = true
			return unix.EIO
		}
		return realRename(fromFD, from, toFD, to)
	}
	deps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
		if checkpoint == signedReplaceAfterCommit {
			return errors.New("force compensation")
		}
		return nil
	}

	_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), deps)
	require.ErrorIs(t, err, ErrSignedReplaceRecovery)
	require.True(t, restoreRenameReached, "the injected compensation rename failure must be reached")
	stateDir := filepath.Join(installDir, signedReplaceTransactionName)
	require.DirExists(t, stateDir)
	assertSignedReplacePathMode(t, stateDir, os.ModeDir|0o700)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(stateDir, signedReplaceBackupName)))
	assertSignedReplacePathMode(t, filepath.Join(stateDir, signedReplaceBackupName), 0o600)
	require.Equal(t, []byte("candidate"), mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))

	_, retryErr := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("retry")), signedReplaceTestDependencies())
	require.ErrorIs(t, retryErr, ErrSignedReplaceRecovery)
	require.Equal(t, []byte("candidate"), mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(stateDir, signedReplaceBackupName)))
	assertSignedReplacePathMode(t, filepath.Join(stateDir, signedReplaceBackupName), 0o600)
}

func TestReplaceVerifiedMARSReleaseDoesNotOverwriteUnknownCompensationDestination(t *testing.T) {
	for _, existing := range []bool{true, false} {
		name := "absent"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			installDir := signedReplaceTestDir(t)
			prior := []byte("prior-mars-binary")
			if existing {
				require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, 0o755))
			}
			unknown := []byte("unknown-same-user-replacement")
			compensationCommitReached := false
			deps := signedReplaceTestDependencies()
			deps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
				switch checkpoint {
				case signedReplaceAfterCommit:
					return errors.New("force compensation")
				case signedReplaceBeforeCompensateCommit:
					compensationCommitReached = true
					destination := filepath.Join(installDir, DefaultBinary)
					require.NoError(t, os.Remove(destination))
					require.NoError(t, os.WriteFile(destination, unknown, 0o755))
				}
				return nil
			}

			_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), deps)
			require.ErrorIs(t, err, ErrSignedReplaceRecovery)
			require.True(t, compensationCommitReached, "the destination swap must occur immediately before compensation")
			require.Equal(t, unknown, mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
			stateDir := filepath.Join(installDir, signedReplaceTransactionName)
			require.DirExists(t, stateDir)
			assertSignedReplacePathMode(t, stateDir, os.ModeDir|0o700)
			if existing {
				require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(stateDir, signedReplaceBackupName)))
				assertSignedReplacePathMode(t, filepath.Join(stateDir, signedReplaceBackupName), 0o600)
			} else {
				require.NoFileExists(t, filepath.Join(stateDir, signedReplaceBackupName))
			}
		})
	}
}

func TestReplaceVerifiedMARSReleaseRejectsUnknownCompensationRestore(t *testing.T) {
	installDir := signedReplaceTestDir(t)
	prior := []byte("prior-mars-binary")
	require.NoError(t, os.WriteFile(filepath.Join(installDir, DefaultBinary), prior, 0o750))
	unknownRestore := []byte("unknown-restore")
	checkpointReached := false
	deps := signedReplaceTestDependencies()
	deps.checkpoint = func(checkpoint signedReplaceCheckpoint) error {
		switch checkpoint {
		case signedReplaceAfterCommit:
			return errors.New("force compensation")
		case signedReplaceBeforeCompensateCommit:
			checkpointReached = true
			restore := filepath.Join(installDir, signedReplaceTransactionName, signedReplaceRestoreName)
			require.NoError(t, os.Remove(restore))
			require.NoError(t, os.WriteFile(restore, unknownRestore, 0o750))
		}
		return nil
	}

	_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), deps)
	require.ErrorIs(t, err, ErrSignedReplaceRecovery)
	require.True(t, checkpointReached, "the restore substitution must occur immediately before compensation")
	require.Equal(t, []byte("candidate"), mustReadSignedReplaceFile(t, filepath.Join(installDir, DefaultBinary)))
	stateDir := filepath.Join(installDir, signedReplaceTransactionName)
	require.DirExists(t, stateDir)
	require.Equal(t, prior, mustReadSignedReplaceFile(t, filepath.Join(stateDir, signedReplaceBackupName)))
	assertSignedReplacePathMode(t, filepath.Join(stateDir, signedReplaceBackupName), 0o600)
	require.Equal(t, unknownRestore, mustReadSignedReplaceFile(t, filepath.Join(stateDir, signedReplaceRestoreName)))
}

func signedReplaceTestDependencies() signedReplaceDependencies {
	return signedReplaceTestDependenciesFor("v0.69.0", testSignedReplaceCommit)
}

func signedReplaceTestDependenciesFor(tag, commit string) signedReplaceDependencies {
	return signedReplaceDependencies{readBuildInfo: func(io.ReaderAt) (*buildinfo.BuildInfo, error) {
		_, archSetting, archValue, ok := marsReleaseArchiveIdentity(tag, commit, runtime.GOOS, runtime.GOARCH)
		if !ok {
			return nil, errors.New("unsupported test platform")
		}
		return &buildinfo.BuildInfo{
			GoVersion: releaseGoVersion,
			Path:      DefaultPackage,
			Main:      debug.Module{Path: releaseModulePath},
			Settings: []debug.BuildSetting{
				{Key: "GOOS", Value: runtime.GOOS}, {Key: "GOARCH", Value: runtime.GOARCH},
				{Key: archSetting, Value: archValue}, {Key: "CGO_ENABLED", Value: "0"},
				{Key: "-trimpath", Value: "true"}, {Key: "vcs", Value: "git"},
				{Key: "vcs.revision", Value: commit}, {Key: "vcs.modified", Value: "false"},
				{Key: "vcs.time", Value: "2026-07-22T00:00:00Z"},
			},
		}, nil
	}}
}

func signedReplaceTestDownload(t *testing.T, binary []byte) verifiedMARSReleaseDownload {
	return signedReplaceTestDownloadFor(t, "v0.69.0", testSignedReplaceCommit, 690, binary)
}

func signedReplaceTestDownloadFor(t *testing.T, tag, commit string, releaseID int64, binary []byte) verifiedMARSReleaseDownload {
	t.Helper()
	archiveName, _, _, ok := marsReleaseArchiveIdentity(tag, commit, runtime.GOOS, runtime.GOARCH)
	require.True(t, ok)
	return verifiedMARSReleaseDownload{releaseID: releaseID, tag: tag, fullCommit: commit, archiveName: archiveName, candidate: append([]byte(nil), binary...)}
}

func signedReplaceTestDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	home, err = filepath.EvalSymlinks(home)
	require.NoError(t, err)
	dir, err := os.MkdirTemp(home, ".mars-signed-replace-test-")
	require.NoError(t, err)
	require.NoError(t, os.Chmod(dir, 0o700))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})
	return dir
}

func assertSignedReplaceUnsafe(t *testing.T, installDir string) {
	t.Helper()
	_, err := replaceVerifiedMARSReleaseWithDependencies(context.Background(), installDir, signedReplaceTestDownload(t, []byte("candidate")), signedReplaceTestDependencies())
	require.ErrorIs(t, err, ErrSignedReplaceUnsafe)
	require.NoDirExists(t, filepath.Join(installDir, signedReplaceTransactionName))
}

func assertSignedReplacePathMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	require.NoError(t, err)
	if expected&os.ModeDir != 0 {
		require.True(t, info.IsDir())
		expected &^= os.ModeDir
	}
	require.Equal(t, expected.Perm(), info.Mode().Perm())
}

func assertSignedReplaceNlink(t *testing.T, path string, expected uint64) {
	t.Helper()
	var stat unix.Stat_t
	require.NoError(t, unix.Lstat(path, &stat))
	require.Equal(t, expected, uint64(stat.Nlink))
}

func mustReadSignedReplaceFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}
