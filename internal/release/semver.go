/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverRE = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)$`)

// SemVer is the strict major.minor.patch version used by release notes.
type SemVer struct {
	Major int
	Minor int
	Patch int
}

func ParseSemVer(value string) (SemVer, error) {
	value = strings.TrimSpace(value)
	match := semverRE.FindStringSubmatch(value)
	if match == nil {
		return SemVer{}, fmt.Errorf("release: invalid semantic version %q, expected MAJOR.MINOR.PATCH", value)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	return SemVer{Major: major, Minor: minor, Patch: patch}, nil
}

func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v SemVer) Bump(kind Bump) SemVer {
	switch kind {
	case BumpMajor:
		return SemVer{Major: v.Major + 1}
	case BumpMinor:
		return SemVer{Major: v.Major, Minor: v.Minor + 1}
	default:
		return SemVer{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
}
