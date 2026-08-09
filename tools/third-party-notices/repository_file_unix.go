//go:build darwin || linux

package notices

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// openRepositoryFileNoFollow walks the repository path through directory
// descriptors. Every component is opened with O_NOFOLLOW, so a repository
// symlink or concurrent pathname replacement cannot redirect the read.
func openRepositoryFileNoFollow(repoRoot, rel string) (*os.File, error) {
	current, err := unix.Open(repoRoot, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("open repository root without following links")
	}
	components := strings.Split(rel, "/")
	for i, component := range components {
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW
		if i < len(components)-1 {
			flags |= unix.O_DIRECTORY
		} else {
			flags |= unix.O_NONBLOCK
		}
		next, openErr := unix.Openat(current, component, flags, 0)
		closeErr := unix.Close(current)
		if openErr != nil {
			return nil, errors.New("open repository-relative component without following links")
		}
		if closeErr != nil {
			unix.Close(next)
			return nil, errors.New("close repository-relative parent descriptor")
		}
		current = next
	}
	file := os.NewFile(uintptr(current), rel)
	if file == nil {
		unix.Close(current)
		return nil, errors.New("adopt repository-relative file descriptor")
	}
	return file, nil
}
