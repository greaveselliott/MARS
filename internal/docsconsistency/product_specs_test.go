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

var (
	productStatusRE  = regexp.MustCompile(`(?m)^\*\*Status:\*\*\s+\S`)
	productUpdatedRE = regexp.MustCompile(`(?m)^\*\*Updated:\*\*\s+\d{4}-\d{2}-\d{2}`)
	productOwnerRE   = regexp.MustCompile(`(?m)^\*\*Owner:\*\*\s+\S`)
	markdownLinkRE   = regexp.MustCompile(`\[[^\]]+\]\(([^)]+\.md)(?:#[^)]+)?\)`)
)

func TestProductSpecsHaveRequiredMetadata(t *testing.T) {
	root := repoRoot(t)
	specDir := filepath.Join(root, "docs", "product-specs")
	for _, path := range productSpecFiles(t, specDir) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		rel := mustRel(t, root, path)
		if !productStatusRE.MatchString(text) {
			t.Fatalf("%s missing **Status:** metadata", rel)
		}
		if !productUpdatedRE.MatchString(text) {
			t.Fatalf("%s missing **Updated:** YYYY-MM-DD metadata", rel)
		}
		if !productOwnerRE.MatchString(text) {
			t.Fatalf("%s missing **Owner:** metadata", rel)
		}
	}
}

func TestProductSpecsIndexCoversEverySpec(t *testing.T) {
	root := repoRoot(t)
	specDir := filepath.Join(root, "docs", "product-specs")
	indexPath := filepath.Join(specDir, "index.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read product spec index: %v", err)
	}
	index := string(indexBytes)
	for _, path := range productSpecFiles(t, specDir) {
		name := filepath.Base(path)
		if name == "index.md" {
			continue
		}
		if !strings.Contains(index, "("+name+")") {
			t.Fatalf("docs/product-specs/index.md must catalog %s", name)
		}
	}
}

func TestProductSpecsIndexLinksExist(t *testing.T) {
	root := repoRoot(t)
	specDir := filepath.Join(root, "docs", "product-specs")
	indexPath := filepath.Join(specDir, "index.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read product spec index: %v", err)
	}
	for _, match := range markdownLinkRE.FindAllStringSubmatch(string(indexBytes), -1) {
		link := strings.TrimSpace(match[1])
		if strings.Contains(link, "://") || strings.HasPrefix(link, "#") {
			continue
		}
		target := filepath.Clean(filepath.Join(specDir, link))
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("docs/product-specs/index.md links to missing file %s", link)
		}
	}
}

func productSpecFiles(t *testing.T, specDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(specDir)
	if err != nil {
		t.Fatalf("read %s: %v", specDir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		files = append(files, filepath.Join(specDir, entry.Name()))
	}
	return files
}

func mustRel(t *testing.T, root, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("rel %s: %v", path, err)
	}
	return rel
}
