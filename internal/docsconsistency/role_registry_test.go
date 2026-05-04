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

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/roleregistry"
	"github.com/greaveselliott/mars-harness/internal/scanner"
)

func TestSourceRoleRegistryCoversGeneratedManifestRoles(t *testing.T) {
	root := repoRoot(t)
	sourcePath := filepath.Join(root, filepath.FromSlash(roleregistry.RegistryPath))
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", roleregistry.RegistryPath, err)
	}
	sourceEntries, err := roleregistry.ParseMarkdown(sourceData)
	if err != nil {
		t.Fatalf("parse source role registry: %v", err)
	}

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target repo: %v", err)
	}
	manifest, err := bundle.Load(repo)
	if err != nil {
		t.Fatalf("load generated manifest: %v", err)
	}

	for roleName, role := range manifest.Roles {
		entry, ok := sourceEntries[roleName]
		if !ok {
			t.Fatalf("%s is in generated manifest but missing from %s", roleName, roleregistry.RegistryPath)
		}
		if entry.Domain != role.Domain {
			t.Fatalf("%s domain mismatch: registry %q manifest %q", roleName, entry.Domain, role.Domain)
		}
		if entry.Mode != role.Mode {
			t.Fatalf("%s mode mismatch: registry %q manifest %q", roleName, entry.Mode, role.Mode)
		}
		if entry.ModelRouting != role.Model {
			t.Fatalf("%s model mismatch: registry %q manifest %q", roleName, entry.ModelRouting, role.Model)
		}
		for _, tool := range role.Tools {
			if !registryCellContainsToken(entry.Tools, tool) {
				t.Fatalf("%s registry tools missing manifest tool %q", roleName, tool)
			}
		}
	}
}

func TestGeneratedRoleRegistryMatchesGeneratedManifest(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target repo: %v", err)
	}

	report, err := roleregistry.CheckRepo(repo)
	if err != nil {
		t.Fatalf("check generated role registry: %v", err)
	}
	if !report.OK() {
		t.Fatalf("generated role registry drift: %+v", report.Issues)
	}
}

func TestSampleBundleRoleRegistryMatchesManifest(t *testing.T) {
	root := repoRoot(t)
	sample := filepath.Join(root, "examples", "sample-bundle")

	report, err := roleregistry.CheckRepo(sample)
	if err != nil {
		t.Fatalf("check sample-bundle role registry: %v", err)
	}
	if !report.OK() {
		t.Fatalf("sample-bundle role registry drift: %+v", report.Issues)
	}
}

func registryCellContainsToken(cell, token string) bool {
	for _, part := range strings.FieldsFunc(cell, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		if part == token {
			return true
		}
	}
	return false
}
