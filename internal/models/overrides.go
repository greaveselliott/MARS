/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package models

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/repofs"
	"gopkg.in/yaml.v3"
)

const modelOverridesPath = ".harness/model-overrides.yaml"

// ModelOverride is an explicit repo-owned model provider selection.
type ModelOverride struct {
	Routing     string `yaml:"routing,omitempty" json:"routing,omitempty"`
	LocalBundle string `yaml:"local_bundle,omitempty" json:"local_bundle,omitempty"`
	Provider    string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Model       string `yaml:"model,omitempty" json:"model,omitempty"`
	Endpoint    string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	APIKeyEnv   string `yaml:"api_key_env,omitempty" json:"api_key_env,omitempty"`
	Reason      string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// ModelOverrides is the .harness/model-overrides.yaml shape.
type ModelOverrides struct {
	Version int                      `yaml:"version" json:"version"`
	Default *ModelOverride           `yaml:"default,omitempty" json:"default,omitempty"`
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

	root, err := openModelRepository(repoRoot, "models override")
	if err != nil {
		return "", err
	}
	if err := requireModelHarness(root, "models override"); err != nil {
		return "", err
	}

	overrides, err := loadModelOverrides(root)
	if err != nil {
		return "", err
	}
	if overrides.Version == 0 {
		overrides.Version = 2
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
	if err := writeModelRepositoryFile(root, modelOverridesPath, data, 0o644, true); err != nil {
		return "", fmt.Errorf("models override: write %s: %w", modelOverridesPath, err)
	}
	return filepath.Join(root.Abs(), modelOverridesPath), nil
}

// SetDefaultModelRouting writes the repo default model route used when no role
// or tier override exists.
func SetDefaultModelRouting(repoRoot string, override ModelOverride) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("models routing: --repo is required")
	}
	normalized, err := normalizeOverride(override)
	if err != nil {
		return "", err
	}
	root, err := openModelRepository(repoRoot, "models routing")
	if err != nil {
		return "", err
	}
	if err := requireModelHarness(root, "models routing"); err != nil {
		return "", err
	}
	overrides, err := loadModelOverrides(root)
	if err != nil {
		return "", err
	}
	overrides.Version = 2
	overrides.Default = &normalized
	data, err := yaml.Marshal(overrides)
	if err != nil {
		return "", fmt.Errorf("models routing: marshal overrides: %w", err)
	}
	if err := writeModelRepositoryFile(root, modelOverridesPath, data, 0o644, true); err != nil {
		return "", fmt.Errorf("models routing: write %s: %w", modelOverridesPath, err)
	}
	return filepath.Join(root.Abs(), modelOverridesPath), nil
}

// LoadModelOverrides reads model overrides from a target repo. Missing file is empty state.
func LoadModelOverrides(repoRoot string) (ModelOverrides, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return ModelOverrides{}, fmt.Errorf("models override: --repo is required")
	}
	root, err := openModelRepository(repoRoot, "models override")
	if err != nil {
		return ModelOverrides{}, err
	}
	return loadModelOverrides(root)
}

func loadModelOverrides(root *repofs.Root) (ModelOverrides, error) {
	data, err := root.ReadFile(modelOverridesPath)
	if errors.Is(err, fs.ErrNotExist) {
		return ModelOverrides{Version: 1}, nil
	}
	if err != nil {
		return ModelOverrides{}, fmt.Errorf("models override: read %s: %w", modelOverridesPath, err)
	}
	var overrides ModelOverrides
	if err := yaml.Unmarshal(data, &overrides); err != nil {
		return ModelOverrides{}, fmt.Errorf("models override: parse %s: invalid YAML", modelOverridesPath)
	}
	if overrides.Version == 0 {
		overrides.Version = 1
	}
	return overrides, nil
}

func openModelRepository(repoRoot, operation string) (*repofs.Root, error) {
	root, err := repofs.Open(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("%s: repository root is unavailable — choose an existing repository directory", operation)
	}
	return root, nil
}

func requireModelHarness(root *repofs.Root, operation string) error {
	info, err := root.Stat(".harness")
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%s: .harness is missing or unavailable — run `mars init --repo <path>` first using the same --repo path", operation)
	}
	return nil
}

func writeModelRepositoryFile(root *repofs.Root, rel string, data []byte, defaultMode os.FileMode, preserveExistingMode bool) error {
	mode := defaultMode
	info, err := root.Stat(rel)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("repository replacement target %s is not a regular file", rel)
		}
		if preserveExistingMode {
			mode = info.Mode().Perm()
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return root.AtomicWrite(rel, data, mode)
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
	if overrides.Default != nil {
		normalized, err := normalizeOverride(*overrides.Default)
		return normalized, err == nil, err
	}
	return ModelOverride{}, false, nil
}

// ResolveDefaultModelRouting returns only the repo default route, if present.
func ResolveDefaultModelRouting(repoRoot string) (ModelOverride, bool, error) {
	overrides, err := LoadModelOverrides(repoRoot)
	if err != nil {
		return ModelOverride{}, false, err
	}
	if overrides.Default == nil {
		return ModelOverride{}, false, nil
	}
	normalized, err := normalizeOverride(*overrides.Default)
	return normalized, err == nil, err
}

func normalizeOverride(override ModelOverride) (ModelOverride, error) {
	override.Routing = normalizeRouting(override.Routing)
	override.LocalBundle = strings.ToLower(strings.TrimSpace(override.LocalBundle))
	override.Provider = NormalizeProvider(override.Provider)
	override.Model = strings.TrimSpace(override.Model)
	override.Endpoint = strings.TrimSpace(override.Endpoint)
	override.APIKeyEnv = strings.TrimSpace(override.APIKeyEnv)
	override.Reason = strings.TrimSpace(override.Reason)

	if override.Routing == "" {
		if override.LocalBundle != "" {
			override.Routing = RoutingLocal
		} else {
			override.Routing = RoutingCloud
		}
	}
	if override.Routing == RoutingLocal {
		if override.LocalBundle == "" {
			override.LocalBundle = LocalBundleAuto
		}
		if override.LocalBundle != LocalBundleAuto {
			if _, ok := BundleByID(override.LocalBundle); !ok {
				return ModelOverride{}, fmt.Errorf("models override: unsupported local bundle %q — use auto, %s, %s, or %s", override.LocalBundle, LocalBundleCPU, LocalBundleBalanced, LocalBundleQuality)
			}
		}
		override.Provider = ""
		override.Endpoint = ""
		override.APIKeyEnv = ""
		return override, nil
	}
	if override.Routing == RoutingDefer {
		return override, nil
	}
	if override.Routing != RoutingCloud {
		return ModelOverride{}, fmt.Errorf("models override: unsupported routing %q — use local, cloud, or defer", override.Routing)
	}
	if override.Model == "" {
		return ModelOverride{}, fmt.Errorf("models override: --model is required for cloud/provider routing")
	}
	spec, ok := ProviderSpecByName(override.Provider)
	if !ok || override.Provider == ProviderRegistry {
		return ModelOverride{}, fmt.Errorf("models override: unsupported provider %q", override.Provider)
	}
	if !spec.Selectable {
		return ModelOverride{}, fmt.Errorf("models override: provider %q is not selectable: %s", spec.Name, spec.UnavailableReason)
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
		if override.Endpoint == "" {
			override.Endpoint = spec.DefaultEndpoint
		}
		if override.APIKeyEnv == "" {
			override.APIKeyEnv = spec.DefaultAPIKeyEnv
		}
		if override.APIKeyEnv == "" {
			return ModelOverride{}, fmt.Errorf("models override: --api-key-env is required for provider %s", override.Provider)
		}
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
