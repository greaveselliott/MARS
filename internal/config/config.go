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
	ModelsDir    string `yaml:"models_dir"`
	BinDir       string `yaml:"bin_dir"`
	TracesDir    string `yaml:"traces_dir"`
	LogFormat    string `yaml:"log_format"`
	GitHubToken  string `yaml:"github_token"`
	WebhookPort  int    `yaml:"webhook_port"`
	DashboardPort int   `yaml:"dashboard_port"`
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

func defaults() Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".mars-harness")
	return Config{
		ModelsDir:     filepath.Join(base, "models"),
		BinDir:        filepath.Join(base, "bin"),
		TracesDir:     filepath.Join(base, "traces"),
		LogFormat:     "text",
		WebhookPort:   9091,
		DashboardPort: 9090,
	}
}

func applyEnv(cfg *Config) {
	envMap := map[string]*string{
		"MARS_HARNESS_MODELS_DIR":   &cfg.ModelsDir,
		"MARS_HARNESS_BIN_DIR":      &cfg.BinDir,
		"MARS_HARNESS_TRACES_DIR":   &cfg.TracesDir,
		"MARS_HARNESS_LOG_FORMAT":   &cfg.LogFormat,
		"MARS_HARNESS_GITHUB_TOKEN": &cfg.GitHubToken,
	}
	for k, ptr := range envMap {
		if v := os.Getenv(k); strings.TrimSpace(v) != "" {
			*ptr = v
		}
	}
	envInt := map[string]*int{
		"MARS_HARNESS_WEBHOOK_PORT":   &cfg.WebhookPort,
		"MARS_HARNESS_DASHBOARD_PORT": &cfg.DashboardPort,
	}
	for k, ptr := range envInt {
		if v := os.Getenv(k); strings.TrimSpace(v) != "" {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				*ptr = n
			}
		}
	}
}
