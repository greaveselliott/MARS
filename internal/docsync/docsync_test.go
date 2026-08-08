/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-019-typescript-monorepo-docsync.md
*/
package docsync

import (
	"os"
	"path/filepath"
	"strings"
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

func TestMetadataDocsParsesInlineStaticAssetComment(t *testing.T) {
	css := `/* MarsDocSync: ["docs/features/F-001-product-walking-skeleton.md"] */

body { margin: 0; }
`
	got := MetadataDocs(css)
	if len(got) != 1 || got[0] != "docs/features/F-001-product-walking-skeleton.md" {
		t.Fatalf("inline docs parsed incorrectly: %#v", got)
	}
}

func TestAuditReportsMissingMetadataAndMissingExpectedDocs(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, "go.mod", "module github.com/greaveselliott/mars\n")
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

func TestAuditIncludesDeployedSourceRoots(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, "docs/features/F-001-product-walking-skeleton.md", "feature")
	writeDocSyncTestFile(t, dir, "main.go", `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`)
	writeDocSyncTestFile(t, dir, "cmd/task-notes-api/main.go", `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`)
	writeDocSyncTestFile(t, dir, "src/style.css", `/* MarsDocSync: ["docs/features/F-001-product-walking-skeleton.md"] */
body { margin: 0; }
`)
	writeDocSyncTestFile(t, dir, "src/game.js", `console.log("missing metadata")
`)

	report, err := Audit(Config{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if len(report.Files) != 4 {
		t.Fatalf("expected src files to be audited, got %#v", report.Files)
	}
	if report.OK() {
		t.Fatalf("expected missing metadata finding for src/game.js")
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Path == "src/game.js" && finding.Message == "missing MarsDocSync docs metadata" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing src/game.js finding: %#v", report.Findings)
	}
}

func TestAuditIncludesTypeScriptMonorepoDefaultsAndSkipsGeneratedOutput(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, "docs/features/F-019-typescript-monorepo-docsync.md", "feature")
	metadata := `/*
MarsDocSync:
docs:
- docs/features/F-019-typescript-monorepo-docsync.md
*/
export const covered = true
`
	for _, rel := range []string{
		"apps/web/src/page.tsx",
		"packages/game/src/rules.ts",
		"workers/app.ts",
		"tests/integration/room.test.ts",
	} {
		writeDocSyncTestFile(t, dir, rel, metadata)
	}
	for _, rel := range []string{
		"apps/web/node_modules/pkg/index.ts",
		"apps/web/dist/index.js",
		"apps/mobile/.expo/router.ts",
		"apps/web/.react-router/types.ts",
		"packages/game/src/schema.generated.ts",
	} {
		writeDocSyncTestFile(t, dir, rel, "export const generated = true\n")
	}

	report, err := Audit(Config{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK() {
		t.Fatalf("unexpected findings: %#v", report.Findings)
	}
	if len(report.Files) != 4 {
		t.Fatalf("expected four authored TypeScript files, got %#v", report.Files)
	}
}

func TestAuditUsesManifestDocSyncOverridesPerField(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, "docs/features/F-019-typescript-monorepo-docsync.md", "feature")
	writeDocSyncTestFile(t, dir, ".harness/manifest.yaml", `name: test
docsync:
  include_roots: [modules]
  include_extensions: [.ts]
  exclude_globs: ["**/ignored/**"]
roles:
  engineer:
    prompt: roles/engineer.md
`)
	writeDocSyncTestFile(t, dir, "modules/core/index.ts", `/* MarsDocSync: ["docs/features/F-019-typescript-monorepo-docsync.md"] */
export const covered = true
`)
	writeDocSyncTestFile(t, dir, "modules/core/view.tsx", "export const notSelected = true\n")
	writeDocSyncTestFile(t, dir, "modules/ignored/missing.ts", "export const ignored = true\n")
	writeDocSyncTestFile(t, dir, "apps/web/missing.ts", "export const defaultRootWasReplaced = true\n")

	report, err := Audit(Config{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK() || len(report.Files) != 1 || report.Files[0].Path != "modules/core/index.ts" {
		t.Fatalf("unexpected override report: %#v", report)
	}
}

func TestAuditFallsBackPerEmptyManifestField(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, "docs/features/F-019-typescript-monorepo-docsync.md", "feature")
	writeDocSyncTestFile(t, dir, ".harness/manifest.yaml", `name: test
docsync:
  exclude_globs: ["**/private/**"]
roles:
  engineer:
    prompt: roles/engineer.md
`)
	writeDocSyncTestFile(t, dir, "apps/web/page.tsx", `/* MarsDocSync: ["docs/features/F-019-typescript-monorepo-docsync.md"] */
export default function Page() { return null }
`)
	writeDocSyncTestFile(t, dir, "apps/private/page.tsx", "export default function Hidden() { return null }\n")

	report, err := Audit(Config{RepoRoot: dir})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK() || len(report.Files) != 1 || report.Files[0].Path != "apps/web/page.tsx" {
		t.Fatalf("unexpected per-field fallback report: %#v", report)
	}
}

func TestAuditRejectsUnsafeOrMalformedManifestDocSyncConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		contains string
	}{
		{name: "absolute root", config: "  include_roots: [/tmp/source]\n", contains: "must be repository-relative"},
		{name: "parent root", config: "  include_roots: [../source]\n", contains: "escapes or selects the repository root"},
		{name: "malformed extension", config: "  include_extensions: [ts]\n", contains: "dot-prefixed suffix"},
		{name: "absolute glob", config: "  exclude_globs: [/tmp/**]\n", contains: "must be repository-relative"},
		{name: "parent glob", config: "  exclude_globs: [../private/**]\n", contains: "contains parent traversal"},
		{name: "unsupported glob", config: "  exclude_globs: ['apps/[ab]/**']\n", contains: "character classes are not supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeDocSyncTestFile(t, dir, ".harness/manifest.yaml", "name: test\ndocsync:\n"+tt.config+"roles:\n  engineer:\n    prompt: roles/engineer.md\n")
			_, err := Audit(Config{RepoRoot: dir})
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("expected error containing %q, got %v", tt.contains, err)
			}
		})
	}
}

func TestRequiresMetadataUsesEffectiveManifestSelection(t *testing.T) {
	dir := t.TempDir()
	writeDocSyncTestFile(t, dir, ".harness/manifest.yaml", `name: test
docsync:
  include_roots: [modules]
  include_extensions: [.tsx]
  exclude_globs: ["**/generated/**"]
roles:
  engineer:
    prompt: roles/engineer.md
`)
	for _, tc := range []struct {
		path string
		want bool
	}{
		{path: "modules/ui/card.tsx", want: true},
		{path: "modules/ui/card.ts", want: false},
		{path: "modules/generated/card.tsx", want: false},
		{path: "apps/web/card.tsx", want: false},
		{path: "root.tsx", want: true},
	} {
		got, err := RequiresMetadata(dir, tc.path)
		if err != nil {
			t.Fatalf("RequiresMetadata(%s): %v", tc.path, err)
		}
		if got != tc.want {
			t.Fatalf("RequiresMetadata(%s) = %v, want %v", tc.path, got, tc.want)
		}
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
