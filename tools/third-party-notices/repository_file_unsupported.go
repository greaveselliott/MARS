//go:build !darwin && !linux

package notices

import (
	"errors"
	"os"
)

func openRepositoryFileNoFollow(_, _ string) (*os.File, error) {
	return nil, errors.New("dependency notice inputs are supported only on Darwin and Linux")
}
