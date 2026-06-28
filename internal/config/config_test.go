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
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsDefaultPathAndSaveRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := DefaultPath(); got != filepath.Join(home, ".mars", "config.yaml") {
		t.Fatalf("DefaultPath = %q", got)
	}

	cfg := Defaults()
	if cfg.ModelsDir != filepath.Join(home, ".mars", "models") {
		t.Fatalf("unexpected default models dir: %q", cfg.ModelsDir)
	}
	cfg.GitHubToken = "local-token"
	cfg.Telemetry.Endpoint = "http://127.0.0.1:9099/telemetry"
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("saved config perms = %v, want 0600", info.Mode().Perm())
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load saved config: %v", err)
	}
	if loaded.GitHubToken != "local-token" || loaded.Telemetry.Endpoint != cfg.Telemetry.Endpoint {
		t.Fatalf("loaded config did not round-trip: %+v", loaded)
	}
}

func TestCodeIntelDefaultsEnabledAndEnvCanDisable(t *testing.T) {
	t.Setenv("MARS_CODE_INTEL_ENABLED", "false")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CodeIntel.Enabled {
		t.Fatalf("expected code intel disabled by env, got %+v", cfg.CodeIntel)
	}

	t.Setenv("MARS_CODE_INTEL_ENABLED", "")
	cfg, err = Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CodeIntel.Enabled {
		t.Fatalf("expected code intel enabled by default, got %+v", cfg.CodeIntel)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("MARS_MODELS_DIR", "/tmp/models")
	t.Setenv("MARS_BIN_DIR", "/tmp/bin")
	t.Setenv("MARS_TRACES_DIR", "/tmp/traces")
	t.Setenv("MARS_LOG_FORMAT", "json")
	t.Setenv("MARS_GITHUB_TOKEN", "env-token")
	t.Setenv("MARS_PERFORMANCE_PROFILE", "battery")
	t.Setenv("MARS_LLAMA_FLASH_ATTENTION", "off")
	t.Setenv("MARS_TELEMETRY_REPORTING", "on")
	t.Setenv("MARS_TELEMETRY_ENDPOINT", "https://telemetry.example")
	t.Setenv("MARS_TELEMETRY_TOKEN", "telemetry-token")
	t.Setenv("MARS_TELEMETRY_INTERVAL", "1h")
	t.Setenv("MARS_WEBHOOK_PORT", "19091")
	t.Setenv("MARS_DASHBOARD_PORT", "19090")
	t.Setenv("MARS_LLAMA_PARALLEL", "2")
	t.Setenv("MARS_LLAMA_THREADS", "8")
	t.Setenv("MARS_LLAMA_THREADS_BATCH", "4")
	t.Setenv("MARS_LLAMA_BATCH_SIZE", "512")
	t.Setenv("MARS_LLAMA_UBATCH_SIZE", "128")
	t.Setenv("MARS_LLAMA_MLOCK", "yes")
	t.Setenv("MARS_CODE_INTEL_ENABLED", "disabled")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsDir != "/tmp/models" || cfg.BinDir != "/tmp/bin" || cfg.TracesDir != "/tmp/traces" {
		t.Fatalf("string env overrides not applied: %+v", cfg)
	}
	if cfg.WebhookPort != 19091 || cfg.DashboardPort != 19090 || cfg.LlamaParallel != 2 ||
		cfg.LlamaThreads != 8 || cfg.LlamaThreadsBatch != 4 || cfg.LlamaBatchSize != 512 || cfg.LlamaUBatchSize != 128 {
		t.Fatalf("int env overrides not applied: %+v", cfg)
	}
	if !cfg.LlamaMLock || cfg.CodeIntel.Enabled {
		t.Fatalf("bool env overrides not applied: %+v", cfg)
	}
	if cfg.Telemetry.Reporting != "on" || cfg.Telemetry.Endpoint == "" || cfg.Telemetry.Token != "telemetry-token" || cfg.Telemetry.ReportInterval != "1h" {
		t.Fatalf("telemetry env overrides not applied: %+v", cfg.Telemetry)
	}
}

func TestLoadAppliesLegacyEnvironmentFallbacks(t *testing.T) {
	t.Setenv("MARS_HARNESS_MODELS_DIR", "/tmp/legacy-models")
	t.Setenv("MARS_HARNESS_WEBHOOK_PORT", "19092")
	t.Setenv("MARS_HARNESS_CODE_INTEL_ENABLED", "disabled")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsDir != "/tmp/legacy-models" || cfg.WebhookPort != 19092 || cfg.CodeIntel.Enabled {
		t.Fatalf("legacy env fallbacks not applied: %+v", cfg)
	}
}

func TestLoadCanonicalEnvironmentBeatsLegacy(t *testing.T) {
	t.Setenv("MARS_MODELS_DIR", "/tmp/new-models")
	t.Setenv("MARS_HARNESS_MODELS_DIR", "/tmp/legacy-models")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsDir != "/tmp/new-models" {
		t.Fatalf("canonical env should win, got %q", cfg.ModelsDir)
	}
}

func TestLoadDefaultPathReadsLegacyConfigWhenNewConfigMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	legacyPath := filepath.Join(home, ".mars-harness", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("mkdir legacy config: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("models_dir: /tmp/legacy-models\n"), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := Load(DefaultPath())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsDir != "/tmp/legacy-models" {
		t.Fatalf("expected legacy config fallback, got %+v", cfg)
	}
}

func TestLoadRejectsInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("models_dir: [unterminated\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid YAML error")
	}
}
