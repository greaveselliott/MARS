/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MissingRequiredModelFiles returns basenames of GGUF artifacts required by the
// effective performance profile that are absent from modelsDir.
func MissingRequiredModelFiles(modelsDir, performanceProfile string) ([]string, error) {
	modelsDir = strings.TrimSpace(modelsDir)
	if modelsDir == "" {
		return nil, fmt.Errorf("hardware: models directory is required to verify profile weights")
	}
	hw := Detect()
	required := UniqueModels(DefaultModelsForHardware(hw, performanceProfile))
	var missing []string
	for _, spec := range required {
		path := filepath.Join(modelsDir, spec.File)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, spec.File)
				continue
			}
			return nil, fmt.Errorf("hardware: stat %s: %w", path, err)
		}
	}
	return missing, nil
}

// EstimatedProfileRAMMiB sums RAMMinMiB for unique models required by the
// effective performance profile (conservative footprint estimate for doctor).
func EstimatedProfileRAMMiB(performanceProfile string) int {
	hw := Detect()
	total := 0
	for _, spec := range UniqueModels(DefaultModelsForHardware(hw, performanceProfile)) {
		total += spec.RAMMinMiB
	}
	return total
}

// ProfileModelPreflightError formats an actionable error when required weights
// are missing for the active performance profile (T-033 / AD-032 extension).
func ProfileModelPreflightError(performanceProfile string, missing []string) error {
	if len(missing) == 0 {
		return nil
	}
	effective := EffectivePerformanceProfile(Detect(), performanceProfile)
	return fmt.Errorf(
		"missing model file(s) for performance_profile %q: %s — run 'mars-harness setup' to download the required weights for this profile",
		effective,
		strings.Join(missing, ", "),
	)
}
