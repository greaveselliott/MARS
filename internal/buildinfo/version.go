/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package buildinfo

// DefaultVersion is the source-tree fallback used when release builds do not
// inject a version with ldflags.
const DefaultVersion = "0.41.28"
