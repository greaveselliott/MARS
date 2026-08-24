/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/release-versioning.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-018-goreleaser-distribution.md
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
var htmlExecutableSurfaceRE = regexp.MustCompile(`(?is)<pre><code>.*?</code></pre>|<td>\s*<code>[^<]*</code>\s*</td>`)

func TestSourceRepoVersioningRuleIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := []struct {
		rel     string
		needles []string
	}{
		{
			rel: "AGENTS.md",
			needles: []string{
				"mars release notes --repo . --bump auto",
				"non-release semantic commit",
				"release: notes X.Y.Z",
				"repository-owned release producer",
				"GitHub `actions/attest`",
			},
		},
		{
			rel: "CONTRIBUTING.md",
			needles: []string{
				"active execution plan",
				"T-071 through T-079",
				"no-publish producer",
				"Do not create or move a tag",
			},
		},
		{
			rel: "docs/design-docs/release-versioning.md",
			needles: []string{
				"mars release notes --repo . --bump auto",
				"non-release semantic commit",
				"release: notes X.Y.Z",
				"AD-313: Source MARS Uses Pinned GoReleaser; Targets Own Their Producer",
			},
		},
	}

	for _, item := range required {
		data, err := os.ReadFile(filepath.Join(root, item.rel))
		if err != nil {
			t.Fatalf("read %s: %v", item.rel, err)
		}
		text := string(data)
		for _, needle := range item.needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document source versioning authority; missing %q", item.rel, needle)
			}
		}
		if item.rel == "AGENTS.md" || item.rel == "docs/design-docs/release-versioning.md" {
			for _, needle := range []string{"Impact", "Why", "What Changed"} {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s must document detailed release note narrative; missing %q", item.rel, needle)
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

func TestReleaseProductionUsesStandardSourceAndRepositoryOwnedTargetContracts(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "design-docs", "release-versioning.md"))
	if err != nil {
		t.Fatalf("read release versioning doc: %v", err)
	}
	text := strings.Join(strings.Fields(string(data)), " ")
	for _, needle := range []string{
		"AD-313: Source MARS Uses Pinned GoReleaser; Targets Own Their Producer",
		"AD-315: Launch Uses Conventional Go Production And GitHub Attestations",
		"publication-disabled private snapshots",
		"Generated target repositories do not inherit this Go-specific producer",
		"signed archive consumer",
		"Fully superseded 2026-07-22 by T-066 D1/F-018",
		"Superseded 2026-07-21 by AD-313/T-065/F-018",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("release docs must describe the current source/target producer boundary; missing %q", needle)
		}
	}
	for _, retired := range []string{
		"- Add `mars release verify-assets",
		"- Audit recent tags for notes-only or missing GitHub releases with `mars release audit`",
	} {
		if strings.Contains(text, retired) {
			t.Fatalf("current release implementation requirements must not retain retired command %q", retired)
		}
	}
}

func TestLiveReleaseGuidesDoNotInvokeRetiredConsumerCommands(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"docs/cli-reference.html",
		"docs/auth-credentials-reference.html",
		"docs/files-state-reference.html",
		"docs/planning-delivery-guide.html",
		"docs/release-update-guide.html",
		"docs/troubleshooting-guide.html",
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		executableSurfaces := htmlExecutableSurfaceRE.FindAllString(string(data), -1)
		for _, retired := range []string{
			"mars release verify-assets",
			"mars release audit",
		} {
			for _, surface := range executableSurfaces {
				if strings.Contains(surface, retired) {
					t.Fatalf("%s must not invoke retired command %q in an executable surface", rel, retired)
				}
			}
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
