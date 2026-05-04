/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/hardware"
	"gopkg.in/yaml.v3"
)

const modelOverridesPath = ".harness/model-overrides.yaml"

// ModelOverride is an explicit repo-owned model provider selection.
type ModelOverride struct {
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Reason   string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// ModelOverrides is the .harness/model-overrides.yaml shape.
type ModelOverrides struct {
	Version int                      `yaml:"version" json:"version"`
	Tiers   map[string]ModelOverride `yaml:"tiers,omitempty" json:"tiers,omitempty"`
	Roles   map[string]ModelOverride `yaml:"roles,omitempty" json:"roles,omitempty"`
}

// SetModelOverride writes a tier or role override into the target repo harness.
func SetModelOverride(repoRoot, tier, role string, override ModelOverride) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("models override: --repo is required")
	}
	tier = strings.ToLower(strings.TrimSpace(tier))
	role = strings.TrimSpace(role)
	if (tier == "") == (role == "") {
		return "", fmt.Errorf("models override: set exactly one of --tier or --role")
	}
	if tier != "" && !isModelTier(tier) {
		return "", fmt.Errorf("models override: unsupported tier %q — use fast, reasoning, or coding", tier)
	}
	normalized, err := normalizeOverride(override)
	if err != nil {
		return "", err
	}

	harnessDir := filepath.Join(repoRoot, ".harness")
	if info, err := os.Stat(harnessDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("models override: %s is missing — run `mars-harness init --repo %s` first", harnessDir, repoRoot)
	}

	path := filepath.Join(repoRoot, modelOverridesPath)
	overrides, err := LoadModelOverrides(repoRoot)
	if err != nil {
		return "", err
	}
	if overrides.Version == 0 {
		overrides.Version = 1
	}
	if tier != "" {
		if overrides.Tiers == nil {
			overrides.Tiers = map[string]ModelOverride{}
		}
		overrides.Tiers[tier] = normalized
	} else {
		if overrides.Roles == nil {
			overrides.Roles = map[string]ModelOverride{}
		}
		overrides.Roles[role] = normalized
	}

	data, err := yaml.Marshal(overrides)
	if err != nil {
		return "", fmt.Errorf("models override: marshal overrides: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("models override: write %s: %w", path, err)
	}
	return path, nil
}

// LoadModelOverrides reads model overrides from a target repo. Missing file is empty state.
func LoadModelOverrides(repoRoot string) (ModelOverrides, error) {
	path := filepath.Join(strings.TrimSpace(repoRoot), modelOverridesPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ModelOverrides{Version: 1}, nil
	}
	if err != nil {
		return ModelOverrides{}, fmt.Errorf("models override: read %s: %w", path, err)
	}
	var overrides ModelOverrides
	if err := yaml.Unmarshal(data, &overrides); err != nil {
		return ModelOverrides{}, fmt.Errorf("models override: parse %s: %w", path, err)
	}
	if overrides.Version == 0 {
		overrides.Version = 1
	}
	return overrides, nil
}

// ResolveModelOverride returns the role override, then tier override, for a run.
func ResolveModelOverride(repoRoot, roleName, roleModel string) (ModelOverride, bool, error) {
	overrides, err := LoadModelOverrides(repoRoot)
	if err != nil {
		return ModelOverride{}, false, err
	}
	if override, ok := overrides.Roles[strings.TrimSpace(roleName)]; ok {
		normalized, err := normalizeOverride(override)
		return normalized, err == nil, err
	}
	tier := strings.ToLower(strings.TrimSpace(roleModel))
	if !isModelTier(tier) {
		tier = string(hardware.TierCoding)
	}
	if override, ok := overrides.Tiers[tier]; ok {
		normalized, err := normalizeOverride(override)
		return normalized, err == nil, err
	}
	return ModelOverride{}, false, nil
}

func normalizeOverride(override ModelOverride) (ModelOverride, error) {
	override.Provider = NormalizeProvider(override.Provider)
	override.Model = strings.TrimSpace(override.Model)
	override.Endpoint = strings.TrimSpace(override.Endpoint)
	override.Reason = strings.TrimSpace(override.Reason)
	if override.Model == "" {
		return ModelOverride{}, fmt.Errorf("models override: --model is required")
	}
	switch override.Provider {
	case ProviderOllama:
		if override.Endpoint == "" {
			override.Endpoint = DefaultOllamaEndpoint
		}
	case ProviderOpenAICompatible:
		if override.Endpoint == "" {
			return ModelOverride{}, fmt.Errorf("models override: --endpoint is required for provider %s", ProviderOpenAICompatible)
		}
	default:
		return ModelOverride{}, fmt.Errorf("models override: unsupported provider %q — use ollama or openai-compatible", override.Provider)
	}
	return override, nil
}

func isModelTier(value string) bool {
	switch hardware.Tier(value) {
	case hardware.TierFast, hardware.TierReasoning, hardware.TierCoding:
		return true
	default:
		return false
	}
}
