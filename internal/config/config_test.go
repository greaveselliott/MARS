/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/product-specs/product-surface.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
*/
package config

import (
	"path/filepath"
	"testing"
)

func TestCodeIntelDefaultsEnabledAndEnvCanDisable(t *testing.T) {
	t.Setenv("MARS_HARNESS_CODE_INTEL_ENABLED", "false")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CodeIntel.Enabled {
		t.Fatalf("expected code intel disabled by env, got %+v", cfg.CodeIntel)
	}

	t.Setenv("MARS_HARNESS_CODE_INTEL_ENABLED", "")
	cfg, err = Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CodeIntel.Enabled {
		t.Fatalf("expected code intel enabled by default, got %+v", cfg.CodeIntel)
	}
}
