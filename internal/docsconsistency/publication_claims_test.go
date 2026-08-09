/*
MarsDocSync:
docs:
- AGENTS.md
- README.md
- ARCHITECTURE.md
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
- docs/product-specs/product-surface.md
- docs/security-governance-guide.html
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLivePublicationClaimsDescribeDataAndHostBoundaries(t *testing.T) {
	root := repoRoot(t)
	live := []string{
		"AGENTS.md",
		"ARCHITECTURE.md",
		"README.md",
		"docs/auth-credentials-reference.html",
		"docs/cli-reference.html",
		"docs/configuration-reference.html",
		"docs/design-docs/index.md",
		"docs/design-docs/self-reflective-telemetry.md",
		"docs/features/F-012-self-improvement-loop.md",
		"docs/harness-ecosystem/index.html",
		"docs/integrations-validation-guide.html",
		"docs/models-guide.html",
		"docs/observability-guide.html",
		"docs/product-specs/product-surface.md",
		"docs/product-specs/vision.md",
		"docs/safety-quality-guide.html",
		"docs/security-governance-guide.html",
		"docs/tools-mcp-guide.html",
		"internal/scanner/init.go",
	}
	forbidden := []string{
		"all inference runs locally",
		"no cloud api costs",
		"no data exfiltration",
		"anonymous foundation telemetry",
	}

	for _, rel := range live {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		normalized := strings.Join(strings.Fields(string(body)), " ")
		lower := strings.ToLower(normalized)
		for _, claim := range forbidden {
			if strings.Contains(lower, claim) {
				t.Errorf("%s retains universal or ambiguous claim %q", rel, claim)
			}
		}
	}
	targetedForbidden := map[string][]string{
		"docs/features/F-012-self-improvement-loop.md": {
			"anonymous report envelopes",
			"anonymous signatures",
		},
		"docs/observability-guide.html": {
			"private unless anonymous reporting",
			"--addr :9092",
		},
		"internal/scanner/init.go": {
			"never leave this machine",
		},
	}
	for rel, claims := range targetedForbidden {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		lower := strings.ToLower(strings.Join(strings.Fields(string(body)), " "))
		for _, claim := range claims {
			if strings.Contains(lower, claim) {
				t.Errorf("%s retains stale boundary claim %q", rel, claim)
			}
		}
	}

	required := map[string][]string{
		"AGENTS.md": {
			"local-first by default",
			"Configured cloud model routes",
			"opt-in aggregate reporting",
		},
		"README.md": {
			"Local-first is a default, not a universal no-network claim",
			"Configured GitHub, JIRA, remote MCP, update, and model-download workflows transmit",
			"GitHub API requests and webhook deliveries exchange the repository, actor, release, and configured event content required for the operation",
		},
		"ARCHITECTURE.md": {
			"current operating-system user's full authority",
			"not a security sandbox",
		},
		"docs/security-governance-guide.html": {
			"the transport is not anonymous",
			"current operating-system user's full authority",
		},
		"docs/auth-credentials-reference.html": {
			"authentication material according to its protocol",
			"Local storage boundary and use",
		},
		"docs/configuration-reference.html": {
			"does not make the network transport anonymous",
		},
		"docs/features/F-012-self-improvement-loop.md": {
			"minimized aggregate report envelopes under the mode named `anonymous`",
		},
		"docs/integrations-validation-guide.html": {
			"GitHub, JIRA, and remote MCP exchange",
		},
		"docs/models-guide.html": {
			"Enabling a cloud route sends",
		},
		"docs/observability-guide.html": {
			"payload minimization, not network anonymity",
			"--addr 127.0.0.1:9092",
		},
		"docs/safety-quality-guide.html": {
			"rejects every non-loopback or DNS address before opening the collector database",
		},
		"docs/tools-mcp-guide.html": {
			"current operating-system user's full authority",
			"not a security sandbox",
		},
		"internal/scanner/init.go": {
			"telemetry payload excludes raw traces",
			"network transport is not anonymous",
		},
	}
	for rel, needles := range required {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		normalized := strings.Join(strings.Fields(string(body)), " ")
		for _, needle := range needles {
			if !strings.Contains(normalized, needle) {
				t.Errorf("%s missing boundary statement %q", rel, needle)
			}
		}
	}
}
