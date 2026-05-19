/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
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

func TestRemoteTrunkOperatingModelIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"AGENTS.md": {
			"Start from remote trunk",
			"git fetch origin main",
			"origin/main",
			"Push ready commits immediately",
			"origin main",
		},
		"docs/design-docs/delivery-operating-model.md": {
			"AD-108",
			"Remote Trunk Freshness And Immediate Publishing",
			"origin/main",
			"origin main",
			"offline or local-only work",
		},
		"docs/features/F-001-delivery-operating-model.md": {
			"F-001-S012",
			"Remote Trunk Freshness And Immediate Publishing",
			"origin/main",
			"pushes ready commits to `origin main`",
		},
		"docs/product-specs/product-surface.md": {
			"remote-trunk freshness",
			"push timing",
			"work starting from `origin/main`",
		},
		"internal/scanner/init.go": {
			"git fetch origin main",
			"origin/main",
			"origin main",
			"Remote Trunk Freshness And Immediate Publishing",
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
				t.Fatalf("%s must document remote-trunk freshness; missing %q", rel, needle)
			}
		}
	}
}

func TestLiveDemoImprovementLoopIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"AGENTS.md": {
			"Source improvements use the live demo loop",
			"run a representative clean target such as `demo-123`",
			"claim improvement only from the rerun evidence",
		},
		"docs/design-docs/delivery-operating-model.md": {
			"AD-138",
			"Live Demo Improvement Loop",
			"run a clean representative target",
			"review the findings",
			"one or two bounded source actions",
			"claim improvement only from rerun evidence",
		},
		"docs/design-docs/harness-operating-model.md": {
			"live demo improvement loop",
			"`demo-123`",
			"rerun before claiming",
		},
		"docs/design-docs/harness-glossary.md": {
			"Live demo improvement loop",
			"run a clean representative target such as `demo-123`",
			"claim improvement only from rerun evidence",
		},
		"docs/features/F-001-delivery-operating-model.md": {
			"F-001-S014",
			"Continuous Live Demo Improvement Loop",
			"selects one or two bounded source actions",
			"claims improvement only when the rerun shows better product progress",
		},
		"internal/scanner/init.go": {
			"Product lifecycle improvements use a live evidence loop",
			"claim improvement only from rerun evidence",
			"source-only shorthand",
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
				t.Fatalf("%s must document live demo improvement loop; missing %q", rel, needle)
			}
		}
	}
}

func TestCLIToolSkillSyncIsDocumented(t *testing.T) {
	root := repoRoot(t)
	required := map[string][]string{
		"AGENTS.md": {
			"CLI tool/skill sync",
			"mars_harness_cli",
			"repo-shortcut map",
		},
		"docs/design-docs/cli-tool-skill-sync.md": {
			"AD-103",
			"mars_harness_cli",
			"repo shortcut map",
			"generated target doctrine",
			"skills",
			"go test ./cmd/mars-harness -run TestMarsHarnessCLI",
		},
		"docs/design-docs/tools-glossary.md": {
			"cli-tool-skill-sync.md",
			"mars_harness_cli",
			"repo-shortcut map",
		},
		"docs/design-docs/skill-evolution.md": {
			"cli-tool-skill-sync.md",
			"Skills remain compact",
		},
		"internal/scanner/init.go": {
			"cli-tool-skill-sync.md",
			"CLI tool/skill sync",
			"mars_harness_cli",
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
				t.Fatalf("%s must document CLI tool/skill sync; missing %q", rel, needle)
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
