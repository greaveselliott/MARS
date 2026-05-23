/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var changelogHeadingRE = regexp.MustCompile(`(?m)^## \[[^\]]+\] - [^\n]+`)

func TestSourceRepoVersioningRuleIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := []string{
		"AGENTS.md",
		"CONTRIBUTING.md",
		"docs/design-docs/release-versioning.md",
	}

	for _, rel := range required {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, needle := range []string{
			"mars-harness release notes --repo . --bump auto",
			"non-release semantic commit",
			"release: notes X.Y.Z",
			"GitHub Release",
			"vX.Y.Z",
			"gh release view vX.Y.Z",
		} {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document automatic source versioning; missing %q", rel, needle)
			}
		}
		if rel == "AGENTS.md" || rel == "docs/design-docs/release-versioning.md" {
			for _, needle := range []string{"Impact", "Why", "What Changed"} {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s must document detailed release note narrative; missing %q", rel, needle)
				}
			}
		}
	}

	cursorRule, err := os.ReadFile(filepath.Join(root, ".cursor", "rules", "strict-trunk-commits.mdc"))
	if err != nil {
		t.Fatalf("read Cursor strict trunk adapter: %v", err)
	}
	cursorText := string(cursorRule)
	for _, needle := range []string{
		"AGENTS.md",
		"docs/roles/personas/foundation-maintainer.md",
		"release",
	} {
		if !strings.Contains(cursorText, needle) {
			t.Fatalf(".cursor/rules/strict-trunk-commits.mdc must stay a thin release adapter; missing %q", needle)
		}
	}
}

func TestReleasePublicationIsLocalFirst(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "design-docs", "release-versioning.md"))
	if err != nil {
		t.Fatalf("read release versioning doc: %v", err)
	}
	text := string(data)
	for _, needle := range []string{
		"mars-harness release publish-assets",
		"--upload none|github|auto",
		"local release assets",
		"Optional GitHub mirror",
		"mars-harness release verify-assets --dist",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("release docs must describe local-first asset publication; missing %q", needle)
		}
	}
}

func TestChangelogEntriesUseDetailedNarrativeSections(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	text := string(data)
	matches := changelogHeadingRE.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		t.Fatal("CHANGELOG.md must contain release entries")
	}

	buckets := []string{
		"### Breaking Changes",
		"### Features",
		"### Fixes",
		"### Documentation",
		"### Maintenance",
		"### Tests",
		"### Other",
		"### Delivery Evidence",
	}
	for i, match := range matches {
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		entry := text[match[0]:end]
		version := strings.TrimSpace(entry[:strings.Index(entry, "\n")])
		if strings.Contains(entry, "### Why This Release Matters") {
			t.Fatalf("%s must use detailed narrative sections, not legacy Why This Release Matters", version)
		}
		impact := strings.Index(entry, "### Impact")
		why := strings.Index(entry, "### Why")
		what := strings.Index(entry, "### What Changed")
		if impact < 0 || why < 0 || what < 0 {
			t.Fatalf("%s must include Impact, Why, and What Changed sections", version)
		}
		if !(impact < why && why < what) {
			t.Fatalf("%s must order narrative sections as Impact, Why, then What Changed", version)
		}
		firstBucket := -1
		for _, bucket := range buckets {
			if idx := strings.Index(entry, bucket); idx >= 0 && (firstBucket == -1 || idx < firstBucket) {
				firstBucket = idx
			}
		}
		if firstBucket == -1 {
			t.Fatalf("%s must retain semantic commit buckets or delivery evidence after narrative sections", version)
		}
		if what > firstBucket {
			t.Fatalf("%s must place Impact, Why, and What Changed before semantic commit buckets", version)
		}
	}
}
