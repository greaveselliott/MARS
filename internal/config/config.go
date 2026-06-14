/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/product-specs/product-surface.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
*/
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the top-level user configuration (AD-015).
// Loaded from ~/.mars-harness/config.yaml, with MARS_HARNESS_ env overrides.
type Config struct {
	ModelsDir           string          `yaml:"models_dir"`
	BinDir              string          `yaml:"bin_dir"`
	TracesDir           string          `yaml:"traces_dir"`
	LogFormat           string          `yaml:"log_format"`
	GitHubToken         string          `yaml:"github_token"`
	WebhookPort         int             `yaml:"webhook_port"`
	DashboardPort       int             `yaml:"dashboard_port"`
	PerformanceProfile  string          `yaml:"performance_profile"`
	LlamaParallel       int             `yaml:"llama_parallel"`
	LlamaThreads        int             `yaml:"llama_threads"`
	LlamaThreadsBatch   int             `yaml:"llama_threads_batch"`
	LlamaBatchSize      int             `yaml:"llama_batch_size"`
	LlamaUBatchSize     int             `yaml:"llama_ubatch_size"`
	LlamaFlashAttention string          `yaml:"llama_flash_attention"`
	LlamaMLock          bool            `yaml:"llama_mlock"`
	Telemetry           TelemetryConfig `yaml:"telemetry"`
	CodeIntel           CodeIntelConfig `yaml:"code_intel"`
}

// TelemetryConfig controls optional anonymous foundation telemetry reporting.
type TelemetryConfig struct {
	Reporting      string `yaml:"reporting"`
	Endpoint       string `yaml:"endpoint"`
	Token          string `yaml:"token"`
	ReportInterval string `yaml:"report_interval"`
}

// CodeIntelConfig controls automatic Mars code graph assistance.
type CodeIntelConfig struct {
	Enabled bool `yaml:"enabled"`
}

// DefaultPath returns the conventional config file location.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mars-harness", "config.yaml")
}

// Load reads the YAML config file and applies environment variable overrides.
// Missing file is not an error — returns defaults.
func Load(path string) (Config, error) {
	cfg := defaults()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(&cfg)
			return cfg, nil
		}
		return cfg, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("config: parse %s: %w", path, err)
	}
	applyEnv(&cfg)
	return cfg, nil
}

// Save writes the YAML config file with owner-only permissions because the
// config may contain optional local credentials.
func Save(path string, cfg Config) error {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: create %s: %w", filepath.Dir(path), err)
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("config: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

func defaults() Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".mars-harness")
	return Config{
		ModelsDir:           filepath.Join(base, "models"),
		BinDir:              filepath.Join(base, "bin"),
		TracesDir:           filepath.Join(base, "traces"),
		LogFormat:           "text",
		WebhookPort:         9091,
		DashboardPort:       9090,
		PerformanceProfile:  "auto",
		LlamaParallel:       1,
		LlamaFlashAttention: "auto",
		Telemetry: TelemetryConfig{
			Reporting:      "off",
			ReportInterval: "24h",
		},
		CodeIntel: CodeIntelConfig{
			Enabled: true,
		},
	}
}

// Defaults returns the built-in configuration without reading disk or env.
func Defaults() Config {
	return defaults()
}

func applyEnv(cfg *Config) {
	envMap := map[string]*string{
		"MARS_HARNESS_MODELS_DIR":            &cfg.ModelsDir,
		"MARS_HARNESS_BIN_DIR":               &cfg.BinDir,
		"MARS_HARNESS_TRACES_DIR":            &cfg.TracesDir,
		"MARS_HARNESS_LOG_FORMAT":            &cfg.LogFormat,
		"MARS_HARNESS_GITHUB_TOKEN":          &cfg.GitHubToken,
		"MARS_HARNESS_PERFORMANCE_PROFILE":   &cfg.PerformanceProfile,
		"MARS_HARNESS_LLAMA_FLASH_ATTENTION": &cfg.LlamaFlashAttention,
		"MARS_HARNESS_TELEMETRY_REPORTING":   &cfg.Telemetry.Reporting,
		"MARS_HARNESS_TELEMETRY_ENDPOINT":    &cfg.Telemetry.Endpoint,
		"MARS_HARNESS_TELEMETRY_TOKEN":       &cfg.Telemetry.Token,
		"MARS_HARNESS_TELEMETRY_INTERVAL":    &cfg.Telemetry.ReportInterval,
	}
	for k, ptr := range envMap {
		if v := os.Getenv(k); strings.TrimSpace(v) != "" {
			*ptr = v
		}
	}
	envInt := map[string]*int{
		"MARS_HARNESS_WEBHOOK_PORT":        &cfg.WebhookPort,
		"MARS_HARNESS_DASHBOARD_PORT":      &cfg.DashboardPort,
		"MARS_HARNESS_LLAMA_PARALLEL":      &cfg.LlamaParallel,
		"MARS_HARNESS_LLAMA_THREADS":       &cfg.LlamaThreads,
		"MARS_HARNESS_LLAMA_THREADS_BATCH": &cfg.LlamaThreadsBatch,
		"MARS_HARNESS_LLAMA_BATCH_SIZE":    &cfg.LlamaBatchSize,
		"MARS_HARNESS_LLAMA_UBATCH_SIZE":   &cfg.LlamaUBatchSize,
	}
	for k, ptr := range envInt {
		if v := os.Getenv(k); strings.TrimSpace(v) != "" {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				*ptr = n
			}
		}
	}
	if v := os.Getenv("MARS_HARNESS_LLAMA_MLOCK"); strings.TrimSpace(v) != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.LlamaMLock = true
		case "0", "false", "no", "off":
			cfg.LlamaMLock = false
		}
	}
	if v := os.Getenv("MARS_HARNESS_CODE_INTEL_ENABLED"); strings.TrimSpace(v) != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on", "enabled":
			cfg.CodeIntel.Enabled = true
		case "0", "false", "no", "off", "disabled":
			cfg.CodeIntel.Enabled = false
		}
	}
}
