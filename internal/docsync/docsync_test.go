/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/features/F-001-delivery-operating-model.md
*/
package docsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetadataDocsParsesStructuredAndLegacyBlocks(t *testing.T) {
	structured := `/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-001-delivery-operating-model.md
*/
package example
`
	legacy := `/*
MarsDocSync:
- docs/design-docs/release-versioning.md
*/
package example
`

	if got := MetadataDocs(structured); len(got) != 2 || got[0] != "docs/design-docs/code-documentation-map.md" || got[1] != "docs/features/F-001-delivery-operating-model.md" {
		t.Fatalf("structured docs parsed incorrectly: %#v", got)
	}
	if got := MetadataDocs(legacy); len(got) != 1 || got[0] != "docs/design-docs/release-versioning.md" {
		t.Fatalf("legacy docs parsed incorrectly: %#v", got)
	}
}

func TestMetadataDocsParsesCommentsAfterHTMLDoctype(t *testing.T) {
	html := `<!DOCTYPE html>
<!--
MarsDocSync:
docs:
- docs/design-docs/dashboard.md
- docs/features/F-010-dashboard-control-plane.md
-->
<html lang="en">
`
	got := MetadataDocs(html)
	if len(got) != 2 || got[0] != "docs/design-docs/dashboard.md" || got[1] != "docs/features/F-010-dashboard-control-plane.md" {
		t.Fatalf("html docs parsed incorrectly: %#v", got)
	}
}

func TestAuditReportsMissingMetadataAndMissingExpectedDocs(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, "docs/design-docs/code-documentation-map.md", "map")
	writeDocSyncTestFile(t, dir, "docs/design-docs/release-versioning.md", "release")
	writeDocSyncTestFile(t, dir, "docs/features/F-009-release-update-lifecycle.md", "feature")
	writeDocSyncTestFile(t, dir, "internal/release/notes.go", `/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
*/
package release
`)
	writeDocSyncTestFile(t, dir, "internal/release/semver.go", "package release\n")

	report, err := Audit(Config{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK() {
		t.Fatalf("expected findings")
	}
	text := report.Summary()
	if text != "docsync: checked 2 files, findings 3" {
		t.Fatalf("unexpected summary %q", text)
	}
}

func TestAuditPassesWhenMetadataMatchesExpectedDocs(t *testing.T) {
	dir := t.TempDir()
	for _, doc := range []string{
		"docs/design-docs/code-documentation-map.md",
		"docs/design-docs/release-versioning.md",
		"docs/features/F-009-release-update-lifecycle.md",
	} {
		writeDocSyncTestFile(t, dir, doc, "doc")
	}
	writeDocSyncTestFile(t, dir, "internal/release/notes.go", `/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release
`)

	report, err := Audit(Config{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK() {
		t.Fatalf("unexpected findings: %#v", report.Findings)
	}
}

func writeDocSyncTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
