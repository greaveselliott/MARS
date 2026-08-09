/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
- docs/tickets/done/MH-025-mars-prompt-port.md
- docs/validation/reports/2026-08-09-rights-media-and-name-review.md
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestExampleRolePromptLineage(t *testing.T) {
	root := repoRoot(t)
	roleDir := filepath.Join(root, "examples", "roles")
	entries, err := os.ReadDir(roleDir)
	if err != nil {
		t.Fatalf("read example role directory: %v", err)
	}

	var roles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			roles = append(roles, entry.Name())
		}
	}
	sort.Strings(roles)
	wantRoles := []string{
		"code-reviewer.md", "dependency-updater.md", "docs-writer.md",
		"engineer.md", "incident-responder.md", "performance-optimizer.md",
		"pipeline-fixer.md", "qa.md", "refactorer.md", "release-manager.md",
		"security-auditor.md",
	}
	if !reflect.DeepEqual(roles, wantRoles) {
		t.Fatalf("example role inventory changed: got %v want %v", roles, wantRoles)
	}

	lineage := []string{
		"mars_introduction_commit: c854b28ce9b5c22a7b9cce926ecfa6e080016553",
		"predecessor_comparison_snapshot: 56afa3a84225988c2bcc18073ee839eeba09645e",
		"textual_port_evidence: not_established",
		"owner_disposition: pending",
	}
	for _, name := range roles {
		body, err := os.ReadFile(filepath.Join(roleDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(body)
		if strings.Contains(text, "source_mars_commit") {
			t.Errorf("%s retains an unsupported symbolic source claim", name)
		}
		for _, fact := range lineage {
			if count := strings.Count(text, fact); count != 1 {
				t.Errorf("%s contains lineage fact %q %d times", name, fact, count)
			}
		}
	}

	manifest := readPromptLineageFile(t, filepath.Join(root, "examples", "roles", "manifest.yaml"))
	if !strings.Contains(manifest, "description: Eleven MARS example automation roles") {
		t.Error("example manifest retains textual-port wording")
	}
	for _, fact := range lineage {
		if count := strings.Count(manifest, fact); count != 1 {
			t.Errorf("manifest contains lineage fact %q %d times", fact, count)
		}
	}

	history := readPromptLineageFile(t, filepath.Join(root, "docs", "tickets", "done", "MH-025-mars-prompt-port.md"))
	for _, required := range []string{
		"T-073 provenance correction — 2026-08-09",
		"textual-port evidence was not established",
		"Owner rights and final disposition remain pending",
		"# MH-025: Port all 11 Mars automation prompts",
		"All eleven prompts exist in repo",
	} {
		if !strings.Contains(history, required) {
			t.Errorf("historical ticket is missing %q", required)
		}
	}
}

func TestCurrentPublicationMediaIsSourceNative(t *testing.T) {
	root := repoRoot(t)
	asset := filepath.Join(root, "docs", "harness-ecosystem", "assets", "harness-network.png")
	if _, err := os.Stat(asset); err == nil || !os.IsNotExist(err) {
		t.Fatalf("retired binary publication asset remains accessible: %v", err)
	}

	surfaces := []string{
		"docs/index.html",
		"docs/site.css",
		"docs/harness-ecosystem/index.html",
		"docs/harness-ecosystem/styles.css",
	}
	for _, rel := range surfaces {
		body := readPromptLineageFile(t, filepath.Join(root, filepath.FromSlash(rel)))
		if strings.Contains(body, "harness-network.png") {
			t.Errorf("%s retains the retired binary publication asset", rel)
		}
	}

	index := readPromptLineageFile(t, filepath.Join(root, "docs", "index.html"))
	if !strings.Contains(index, `class="hero-flow"`) || !strings.Contains(index, "MARS foundation runtime, deployed harness, target repository, and feedback loop") {
		t.Error("docs landing page is missing the semantic source-native system map")
	}
}

func readPromptLineageFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
