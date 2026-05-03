package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceRepoVersioningRuleIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := []string{
		"AGENTS.md",
		"CONTRIBUTING.md",
		".cursor/rules/strict-trunk-commits.mdc",
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
		} {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document automatic source versioning; missing %q", rel, needle)
			}
		}
	}
}

func TestReleaseWorkflowBackfillsNotesOnlyReleases(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	text := string(data)
	for _, needle := range []string{
		"tags: ['v*']",
		"release:",
		"types: [published]",
		"workflow_dispatch:",
		"version:",
		"RELEASE_TAG:",
		"ref: ${{ env.RELEASE_TAG }}",
		"tag_name: ${{ env.RELEASE_TAG }}",
		"overwrite_files: true",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("release workflow must support asset backfills; missing %q", needle)
		}
	}
}
