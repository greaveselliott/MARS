/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-012-self-improvement-loop.md
- docs/product-specs/product-surface.md
*/
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/remediation"
)

func checkDeterministicRemediationHealth(cfg Config) CheckResult {
	start := time.Now()
	name := "deterministic-remediation"
	registry := remediation.DefaultRegistry()
	recipes := registry.List()
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  fmt.Sprintf("%d deterministic remediation recipes available; repo-specific remediation check skipped", len(recipes)),
			Duration: nonZeroDurationSince(start),
		}
	}

	harnessDir := filepath.Join(cfg.RepoPath, ".harness")
	if _, err := os.Stat(harnessDir); err != nil {
		if os.IsNotExist(err) {
			recipe, _ := registry.Find("manifest:validate-or-init")
			return CheckResult{
				Name:     name,
				Status:   statusWarn,
				Message:  fmt.Sprintf("recipe %s applies: target harness scaffold is missing", recipe.ID),
				Duration: nonZeroDurationSince(start),
				Fix:      fmt.Sprintf("run 'mars init --repo %s' before starting autonomous work", cfg.RepoPath),
			}
		}
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("check target harness scaffold: %v", err),
			Duration: nonZeroDurationSince(start),
			Fix:      "run doctor with --repo pointing at a readable repository root",
		}
	}
	if _, err := os.Stat(filepath.Join(harnessDir, "manifest.yaml")); err != nil {
		if os.IsNotExist(err) {
			recipe, _ := registry.Find("manifest:validate-or-init")
			return CheckResult{
				Name:     name,
				Status:   statusWarn,
				Message:  fmt.Sprintf("recipe %s applies: .harness/manifest.yaml is missing", recipe.ID),
				Duration: nonZeroDurationSince(start),
				Fix:      fmt.Sprintf("run 'mars init --repo %s' or 'mars upgrade --repo %s'", cfg.RepoPath, cfg.RepoPath),
			}
		}
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("check target harness manifest: %v", err),
			Duration: nonZeroDurationSince(start),
			Fix:      "repair .harness permissions, then rerun doctor",
		}
	}
	if _, err := os.Stat(filepath.Join(harnessDir, "metadata.yaml")); err != nil {
		if os.IsNotExist(err) {
			recipe, _ := registry.Find("generated-docs:update-missing-defaults")
			return CheckResult{
				Name:     name,
				Status:   statusWarn,
				Message:  fmt.Sprintf("recipe %s applies: generated harness metadata is missing", recipe.ID),
				Duration: nonZeroDurationSince(start),
				Fix:      fmt.Sprintf("run 'mars update harness --repo %s'", cfg.RepoPath),
			}
		}
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("check generated harness metadata: %v", err),
			Duration: nonZeroDurationSince(start),
			Fix:      "repair .harness permissions, then rerun doctor",
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  fmt.Sprintf("%d deterministic remediation recipes available; target harness scaffold has manifest and metadata", len(recipes)),
		Duration: nonZeroDurationSince(start),
	}
}
