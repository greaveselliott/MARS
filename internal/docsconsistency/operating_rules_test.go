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
		lower := strings.ToLower(text)
		for _, needle := range []string{
			"Operating rules",
			"source-only",
			"target",
		} {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document operating-rule inheritance; missing %q", rel, needle)
			}
		}
		for _, needle := range []string{
			"architecture",
			"product",
			"why",
		} {
			if !strings.Contains(lower, needle) {
				t.Fatalf("%s must document rationale-bearing architecture/product changes; missing %q", rel, needle)
			}
		}
	}
}

func TestToolCreationPathIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"AGENTS.md": {
			"Tool creation path",
			"tool_create",
			"record_decision",
		},
		"docs/design-docs/tools-glossary.md": {
			"New built-in tools must originate through `tool_create`",
			"record_decision",
			"design-doc rationale",
		},
		"docs/design-docs/delivery-operating-model.md": {
			"Built-in tool creation must dogfood the meta-tool path",
			"tool_create",
			"record_decision",
		},
		"docs/design-docs/dogfood-and-decisions.md": {
			"bypassing `tool_create` breaks the doctrine it represents",
			"record_decision",
			"design-doc rationale",
		},
		"internal/scanner/init.go": {
			"Tool creation path",
			"tool_create",
			"record_decision",
		},
	}

	for rel, needles := range required {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s must document governed tool creation path; missing %q", rel, needle)
			}
		}
	}
}
