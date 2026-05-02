package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperatingRulesMirrorTargetHarnesses(t *testing.T) {
	root := repoRoot(t)
	required := []string{
		"AGENTS.md",
		"docs/design-docs/mirrored-harness-and-context-glossary.md",
		"docs/product-specs/product-surface.md",
		"internal/scanner/init.go",
	}

	for _, rel := range required {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, needle := range []string{
			"Operating rules",
			"source-only",
			"target",
		} {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document operating-rule inheritance; missing %q", rel, needle)
			}
		}
	}
}
