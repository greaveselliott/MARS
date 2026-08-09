/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/product-specs/product-surface.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/design-docs/github-app-integration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
*/
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
	cfg.WebhookAllowedActorIDs = []int64{42, 84}
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
	if len(loaded.WebhookAllowedActorIDs) != 2 || loaded.WebhookAllowedActorIDs[0] != 42 {
		t.Fatalf("webhook actor policy did not round-trip: %+v", loaded.WebhookAllowedActorIDs)
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
	if err := os.WriteFile(legacyPath, []byte("models_dir: /tmp/legacy-models\n"), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	if err := os.Chmod(legacyPath, 0o644); err != nil {
		t.Fatalf("loosen legacy config: %v", err)
	}

	cfg, err := Load(DefaultPath())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsDir != "/tmp/legacy-models" {
		t.Fatalf("expected legacy config fallback, got %+v", cfg)
	}
	assertConfigMode(t, legacyPath, 0o600)
}

func TestLoadTightensExistingConfigWithoutChangingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	want := []byte("models_dir: /preserved\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("loosen config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ModelsDir != "/preserved" {
		t.Fatalf("unexpected loaded config: %+v", cfg)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("config content changed: got %q want %q", got, want)
	}
	assertConfigMode(t, path, 0o600)
}

func TestSaveTightensConfigAndPreservesCustomParentMode(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "operator-config")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatalf("mkdir custom parent: %v", err)
	}
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatalf("set custom parent mode: %v", err)
	}
	path := filepath.Join(parent, "config.yaml")
	if err := os.WriteFile(path, []byte("models_dir: /old\n"), 0o644); err != nil {
		t.Fatalf("write old config: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("loosen old config: %v", err)
	}

	if err := Save(path, Defaults()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertConfigMode(t, parent, 0o755)
	assertConfigMode(t, path, 0o600)
}

func TestConfigLoadAndSaveRejectSymlinkWithoutTouchingTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.yaml")
	want := []byte("models_dir: /target\n")
	if err := os.WriteFile(target, want, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatalf("set target mode: %v", err)
	}
	link := filepath.Join(dir, "config.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink config: %v", err)
	}

	if _, err := Load(link); err == nil {
		t.Fatal("expected Load to reject config symlink")
	}
	if err := Save(link, Defaults()); err == nil {
		t.Fatal("expected Save to reject config symlink")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("symlink target changed: got %q want %q", got, want)
	}
	assertConfigMode(t, target, 0o644)
}

func TestClearStoredGitHubTokenPreservesOtherFieldsModeAndIgnoresEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	wantField := "models_dir: /stored/models"
	wantUnknown := "future_setting:"
	if err := os.WriteFile(path, []byte("# preserved config\ngithub_token: stored-secret\n"+wantField+"\n"+wantUnknown+"\n  enabled: true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("MARS_GITHUB_TOKEN", "env-secret")
	t.Setenv("MARS_MODELS_DIR", "/env/models")

	cleared, err := ClearStoredGitHubToken(path)
	if err != nil {
		t.Fatalf("ClearStoredGitHubToken: %v", err)
	}
	if !cleared {
		t.Fatal("expected stored GitHub token to be cleared")
	}
	assertConfigMode(t, path, 0o600)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cleared config: %v", err)
	}
	for _, forbidden := range []string{"github_token", "stored-secret", "env-secret", "/env/models"} {
		if strings.Contains(string(got), forbidden) {
			t.Fatalf("cleared config contains %q: %s", forbidden, got)
		}
	}
	for _, preserved := range []string{wantField, wantUnknown, "enabled: true"} {
		if !strings.Contains(string(got), preserved) {
			t.Fatalf("cleared config lost %q: %s", preserved, got)
		}
	}

	beforeSecondClear := append([]byte(nil), got...)
	cleared, err = ClearStoredGitHubToken(path)
	if err != nil {
		t.Fatalf("second ClearStoredGitHubToken: %v", err)
	}
	if cleared {
		t.Fatal("second clear should be idempotent")
	}
	afterSecondClear, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read idempotently cleared config: %v", err)
	}
	if !bytes.Equal(afterSecondClear, beforeSecondClear) {
		t.Fatalf("idempotent clear changed config: got %q want %q", afterSecondClear, beforeSecondClear)
	}
}

func TestClearStoredGitHubTokenUsesSelectedLegacyFileWithoutCopyingOrContamination(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MARS_GITHUB_TOKEN", "env-secret")
	legacyPath := LegacyPath()
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("github_token: legacy-secret\nmodels_dir: /legacy/models\n"), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cleared, err := ClearStoredGitHubToken("")
	if err != nil {
		t.Fatalf("ClearStoredGitHubToken: %v", err)
	}
	if !cleared {
		t.Fatal("expected legacy token to be cleared")
	}
	got, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy config: %v", err)
	}
	if strings.Contains(string(got), "github_token") || strings.Contains(string(got), "legacy-secret") || strings.Contains(string(got), "env-secret") {
		t.Fatalf("legacy token was not isolated from environment: %s", got)
	}
	if !strings.Contains(string(got), "models_dir: /legacy/models") {
		t.Fatalf("legacy config lost unrelated field: %s", got)
	}
	assertConfigMode(t, legacyPath, 0o600)
	if _, err := os.Lstat(DefaultPath()); !os.IsNotExist(err) {
		t.Fatalf("clear copied legacy config into canonical path: %v", err)
	}
}

func assertConfigMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode %s = %04o, want %04o", path, got, want)
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
