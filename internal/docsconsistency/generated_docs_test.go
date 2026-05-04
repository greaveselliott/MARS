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
	"strings"
	"testing"
)

func TestGeneratedDocsReadmeHasRequiredMetadata(t *testing.T) {
	root := repoRoot(t)
	readme := filepath.Join(root, "docs", "generated", "README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read docs/generated/README.md: %v", err)
	}
	text := string(data)
	if !productStatusRE.MatchString(text) {
		t.Fatalf("docs/generated/README.md missing **Status:** metadata")
	}
	if !productUpdatedRE.MatchString(text) {
		t.Fatalf("docs/generated/README.md missing **Updated:** YYYY-MM-DD metadata")
	}
	if !productOwnerRE.MatchString(text) {
		t.Fatalf("docs/generated/README.md missing **Owner:** metadata")
	}
}

func TestGeneratedDocsReadmeCatalogsGeneratedMarkdown(t *testing.T) {
	root := repoRoot(t)
	genDir := filepath.Join(root, "docs", "generated")
	readme := filepath.Join(genDir, "README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read docs/generated/README.md: %v", err)
	}
	index := string(data)

	entries, err := os.ReadDir(genDir)
	if err != nil {
		t.Fatalf("read docs/generated: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || entry.Name() == "README.md" {
			continue
		}
		if !strings.Contains(index, "("+entry.Name()+")") {
			t.Fatalf("docs/generated/README.md must catalog %s", entry.Name())
		}
	}
}

func TestGeneratedDocsReadmeLinksExist(t *testing.T) {
	root := repoRoot(t)
	genDir := filepath.Join(root, "docs", "generated")
	readme := filepath.Join(genDir, "README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read docs/generated/README.md: %v", err)
	}
	for _, match := range markdownLinkRE.FindAllStringSubmatch(string(data), -1) {
		link := strings.TrimSpace(match[1])
		if strings.Contains(link, "://") || strings.HasPrefix(link, "#") {
			continue
		}
		target := filepath.Clean(filepath.Join(genDir, link))
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("docs/generated/README.md links to missing file %s", link)
		}
	}
}
