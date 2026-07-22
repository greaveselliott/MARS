//go:build !darwin && !linux

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package selfupdate

import "context"

func replaceVerifiedMARSReleasePlatform(context.Context, string, signedReplacementCandidate, signedReplaceDependencies) (signedReplaceResult, error) {
	return signedReplaceResult{}, ErrSignedReplacePlatform
}
