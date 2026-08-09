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
	"sort"
	"strings"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/repofs"
)

const (
	RoutingLocal = "local"
	RoutingCloud = "cloud"
	RoutingDefer = "defer"

	LocalBundleAuto     = "auto"
	LocalBundleCPU      = "local-cpu-q3"
	LocalBundleBalanced = "local-balanced-q4"
	LocalBundleQuality  = "local-quality-q8"

	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
	ProviderMistral   = "mistral"
	ProviderXAI       = "xai"
	ProviderDeepSeek  = "deepseek"
	ProviderGroq      = "groq"
	ProviderCohere    = "cohere"
)

// BundleRequirements describes the minimum machine shape for a local bundle.
type BundleRequirements struct {
	RAMMinMiB           int      `json:"ram_min_mib" yaml:"ram_min_mib"`
	DedicatedVRAMMinMiB int      `json:"dedicated_vram_min_mib,omitempty" yaml:"dedicated_vram_min_mib,omitempty"`
	UnifiedMemoryMinMiB int      `json:"unified_memory_min_mib,omitempty" yaml:"unified_memory_min_mib,omitempty"`
	DiskEstimateMiB     int      `json:"disk_estimate_mib,omitempty" yaml:"disk_estimate_mib,omitempty"`
	OS                  []string `json:"os,omitempty" yaml:"os,omitempty"`
	Arch                []string `json:"arch,omitempty" yaml:"arch,omitempty"`
	Backends            []string `json:"backends,omitempty" yaml:"backends,omitempty"`
}

// LocalBundle is a selectable local model bundle.
type LocalBundle struct {
	ID           string                               `json:"id"`
	Name         string                               `json:"name"`
	Profile      string                               `json:"profile"`
	Rank         int                                  `json:"rank"`
	Requirements BundleRequirements                   `json:"requirements"`
	Models       map[hardware.Tier]hardware.ModelSpec `json:"models"`
}

// BundleEligibility is a user-facing eligibility row.
type BundleEligibility struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Profile        string             `json:"profile"`
	Eligible       bool               `json:"eligible"`
	Selected       bool               `json:"selected,omitempty"`
	DisabledReason string             `json:"disabled_reason,omitempty"`
	Reasons        []string           `json:"reasons,omitempty"`
	Requirements   BundleRequirements `json:"requirements"`
}

// EligibilityReport summarizes hardware and local-bundle eligibility.
type EligibilityReport struct {
	Hardware       hardware.Summary    `json:"hardware"`
	SelectedBundle string              `json:"selected_bundle,omitempty"`
	Bundles        []BundleEligibility `json:"bundles"`
}

// ProviderSpec describes a supported model provider.
type ProviderSpec struct {
	Name              string `json:"name"`
	DefaultEndpoint   string `json:"default_endpoint,omitempty"`
	DefaultAPIKeyEnv  string `json:"default_api_key_env,omitempty"`
	Adapter           string `json:"adapter"`
	Selectable        bool   `json:"selectable"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	OfficialDocs      string `json:"official_docs,omitempty"`
}

// ProviderRoute is a normalized runtime route.
type ProviderRoute struct {
	Routing   string `json:"routing" yaml:"routing"`
	Provider  string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model     string `json:"model,omitempty" yaml:"model,omitempty"`
	Endpoint  string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	APIKeyEnv string `json:"api_key_env,omitempty" yaml:"api_key_env,omitempty"`
	APIKey    string `json:"-" yaml:"-"`
}

// LocalBundles returns the built-in local bundles sorted by increasing rank.
func LocalBundles() []LocalBundle {
	bundles := []LocalBundle{
		{
			ID:      LocalBundleCPU,
			Name:    "CPU / low-memory local bundle",
			Profile: string(hardware.ProfileCPU),
			Rank:    10,
			Requirements: BundleRequirements{
				RAMMinMiB:       8192,
				DiskEstimateMiB: 22000,
				Backends:        []string{"cpu", "metal", "cuda"},
			},
			Models: hardware.DefaultModels(hardware.ProfileCPU),
		},
		{
			ID:      LocalBundleBalanced,
			Name:    "Balanced local bundle",
			Profile: string(hardware.ProfileMedium),
			Rank:    20,
			Requirements: BundleRequirements{
				RAMMinMiB:           24576,
				DedicatedVRAMMinMiB: 12288,
				UnifiedMemoryMinMiB: 32768,
				DiskEstimateMiB:     42000,
				Backends:            []string{"metal", "cuda"},
			},
			Models: hardware.DefaultModels(hardware.ProfileMedium),
		},
		{
			ID:      LocalBundleQuality,
			Name:    "Quality local bundle",
			Profile: string(hardware.ProfileHigh),
			Rank:    30,
			Requirements: BundleRequirements{
				RAMMinMiB:           49152,
				DedicatedVRAMMinMiB: 24576,
				UnifiedMemoryMinMiB: 98304,
				DiskEstimateMiB:     72000,
				Backends:            []string{"metal", "cuda"},
			},
			Models: hardware.DefaultModels(hardware.ProfileHigh),
		},
	}
	sort.Slice(bundles, func(i, j int) bool { return bundles[i].Rank < bundles[j].Rank })
	return bundles
}

// BundleByID returns a local bundle by ID.
func BundleByID(id string) (LocalBundle, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, b := range LocalBundles() {
		if b.ID == id {
			return b, true
		}
	}
	return LocalBundle{}, false
}

// EvaluateLocalBundles evaluates all local bundles for a hardware summary.
func EvaluateLocalBundles(hw hardware.Summary) EligibilityReport {
	rows := make([]BundleEligibility, 0, len(LocalBundles()))
	selected := ""
	for _, bundle := range LocalBundles() {
		reasons := localBundleIneligibility(hw, bundle)
		row := BundleEligibility{
			ID:           bundle.ID,
			Name:         bundle.Name,
			Profile:      bundle.Profile,
			Eligible:     len(reasons) == 0,
			Reasons:      reasons,
			Requirements: bundle.Requirements,
		}
		if len(reasons) > 0 {
			row.DisabledReason = strings.Join(reasons, "; ")
		}
		rows = append(rows, row)
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Eligible {
			rows[i].Selected = true
			selected = rows[i].ID
			break
		}
	}
	return EligibilityReport{Hardware: hw, SelectedBundle: selected, Bundles: rows}
}

// ResolveLocalBundle resolves "auto" or a concrete bundle ID and rejects
// unsupported local bundles with actionable reasons.
func ResolveLocalBundle(hw hardware.Summary, requested string) (LocalBundle, EligibilityReport, error) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" {
		requested = LocalBundleAuto
	}
	report := EvaluateLocalBundles(hw)
	if requested == LocalBundleAuto {
		if report.SelectedBundle == "" {
			return LocalBundle{}, report, fmt.Errorf("models: no eligible local bundle for detected hardware — use --model-routing cloud, --inference cloud, or defer local setup until hardware is available")
		}
		b, _ := BundleByID(report.SelectedBundle)
		return b, report, nil
	}
	bundle, ok := BundleByID(requested)
	if !ok {
		return LocalBundle{}, report, fmt.Errorf("models: unsupported local bundle %q — use auto, %s, %s, or %s", requested, LocalBundleCPU, LocalBundleBalanced, LocalBundleQuality)
	}
	reasons := localBundleIneligibility(hw, bundle)
	if len(reasons) > 0 {
		return LocalBundle{}, report, fmt.Errorf("models: local bundle %q is not eligible for detected hardware: %s — use --local-bundle auto, choose cloud routing, or retry on a larger machine", bundle.ID, strings.Join(reasons, "; "))
	}
	return bundle, report, nil
}

// MissingLocalBundleFiles returns missing GGUF artifact basenames for a bundle.
func MissingLocalBundleFiles(modelsDir string, bundle LocalBundle) ([]string, error) {
	modelsDir = strings.TrimSpace(modelsDir)
	if modelsDir == "" {
		return nil, fmt.Errorf("models: models directory is required to verify bundle weights")
	}
	var missing []string
	for _, spec := range hardware.UniqueModels(bundle.Models) {
		path := filepath.Join(modelsDir, spec.File)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, spec.File)
				continue
			}
			return nil, fmt.Errorf("models: stat %s: %w", path, err)
		}
	}
	return missing, nil
}

// LocalBundlePreflightError formats an actionable local-bundle preflight error.
func LocalBundlePreflightError(bundleID string, missing []string) error {
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing model file(s) for local bundle %q: %s — run `mars setup --inference local --local-bundle %s --download` to download the required weights", bundleID, strings.Join(missing, ", "), bundleID)
}

func localBundleIneligibility(hw hardware.Summary, bundle LocalBundle) []string {
	req := bundle.Requirements
	var reasons []string
	if hw.RAMMiB <= 0 {
		reasons = append(reasons, "system RAM is unknown")
	} else if req.RAMMinMiB > 0 && hw.RAMMiB < req.RAMMinMiB {
		reasons = append(reasons, fmt.Sprintf("requires at least %d MiB RAM, detected %d MiB", req.RAMMinMiB, hw.RAMMiB))
	}
	if len(req.OS) > 0 && !stringIn(strings.ToLower(hw.OS), req.OS) {
		reasons = append(reasons, fmt.Sprintf("requires OS %s, detected %s", strings.Join(req.OS, "/"), hw.OS))
	}
	if len(req.Arch) > 0 && !stringIn(strings.ToLower(hw.Arch), req.Arch) {
		reasons = append(reasons, fmt.Sprintf("requires arch %s, detected %s", strings.Join(req.Arch, "/"), hw.Arch))
	}
	if len(req.Backends) > 0 {
		backend := detectedBackend(hw)
		if backend == "" {
			reasons = append(reasons, "GPU/backend is unknown")
		} else if !stringIn(backend, req.Backends) {
			reasons = append(reasons, fmt.Sprintf("requires backend %s, detected %s", strings.Join(req.Backends, "/"), backend))
		}
	}
	if req.DedicatedVRAMMinMiB > 0 || req.UnifiedMemoryMinMiB > 0 {
		dedicated := dedicatedVRAMMiB(hw)
		unified := unifiedMemoryMiB(hw)
		dedicatedOK := req.DedicatedVRAMMinMiB > 0 && dedicated >= req.DedicatedVRAMMinMiB
		unifiedOK := req.UnifiedMemoryMinMiB > 0 && unified >= req.UnifiedMemoryMinMiB
		if !dedicatedOK && !unifiedOK {
			switch {
			case unified > 0:
				reasons = append(reasons, fmt.Sprintf("requires at least %d MiB unified memory, detected %d MiB", req.UnifiedMemoryMinMiB, unified))
			case dedicated > 0:
				reasons = append(reasons, fmt.Sprintf("requires at least %d MiB dedicated VRAM, detected %d MiB", req.DedicatedVRAMMinMiB, dedicated))
			default:
				reasons = append(reasons, "GPU memory is unknown")
			}
		}
	}
	return reasons
}

func stringIn(value string, values []string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range values {
		if value == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func detectedBackend(hw hardware.Summary) string {
	for _, gpu := range hw.GPUs {
		name := strings.ToLower(gpu.Name)
		driver := strings.ToLower(gpu.Driver)
		switch {
		case strings.Contains(driver, "metal") || strings.Contains(name, "apple silicon"):
			return "metal"
		case strings.Contains(driver, "cuda") || strings.Contains(driver, "nvidia") || strings.Contains(name, "nvidia") || strings.Contains(name, "rtx"):
			return "cuda"
		}
	}
	if len(hw.GPUs) == 0 {
		return "cpu"
	}
	return ""
}

func dedicatedVRAMMiB(hw hardware.Summary) int {
	var largest int
	for _, gpu := range hw.GPUs {
		name := strings.ToLower(gpu.Name)
		driver := strings.ToLower(gpu.Driver)
		if strings.Contains(driver, "metal") || strings.Contains(name, "apple silicon") {
			continue
		}
		if gpu.VRAMMiB > largest {
			largest = gpu.VRAMMiB
		}
	}
	return largest
}

func unifiedMemoryMiB(hw hardware.Summary) int {
	for _, gpu := range hw.GPUs {
		name := strings.ToLower(gpu.Name)
		driver := strings.ToLower(gpu.Driver)
		if strings.Contains(driver, "metal") || strings.Contains(name, "apple silicon") {
			if gpu.VRAMMiB > 0 {
				return gpu.VRAMMiB
			}
			return hw.RAMMiB
		}
	}
	return 0
}

// ProviderSpecs returns all known provider routes.
func ProviderSpecs() []ProviderSpec {
	return []ProviderSpec{
		{Name: ProviderOpenAI, DefaultEndpoint: "https://api.openai.com/v1", DefaultAPIKeyEnv: "OPENAI_API_KEY", Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://platform.openai.com/docs/api-reference/chat/create"},
		{Name: ProviderAnthropic, DefaultEndpoint: "https://api.anthropic.com/v1", DefaultAPIKeyEnv: "ANTHROPIC_API_KEY", Adapter: "anthropic_messages", Selectable: true, OfficialDocs: "https://docs.anthropic.com/en/api/messages"},
		{Name: ProviderGemini, DefaultEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai", DefaultAPIKeyEnv: "GEMINI_API_KEY", Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://ai.google.dev/gemini-api/docs/openai"},
		{Name: ProviderMistral, DefaultEndpoint: "https://api.mistral.ai/v1", DefaultAPIKeyEnv: "MISTRAL_API_KEY", Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://docs.mistral.ai/capabilities/completion/"},
		{Name: ProviderXAI, DefaultEndpoint: "https://api.x.ai/v1", DefaultAPIKeyEnv: "XAI_API_KEY", Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://docs.x.ai/docs/api-reference"},
		{Name: ProviderDeepSeek, DefaultEndpoint: "https://api.deepseek.com/v1", DefaultAPIKeyEnv: "DEEPSEEK_API_KEY", Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://api-docs.deepseek.com/api/create-chat-completion"},
		{Name: ProviderGroq, DefaultEndpoint: "https://api.groq.com/openai/v1", DefaultAPIKeyEnv: "GROQ_API_KEY", Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://console.groq.com/docs/api-reference"},
		{Name: ProviderCohere, DefaultEndpoint: "https://api.cohere.com/v2", DefaultAPIKeyEnv: "COHERE_API_KEY", Adapter: "cohere_chat", Selectable: false, UnavailableReason: "native Cohere tool-call adapter requires request-capture fixtures before runtime selection", OfficialDocs: "https://docs.cohere.com/reference/chat"},
		{Name: ProviderOpenAICompatible, Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://platform.openai.com/docs/api-reference/chat/create"},
		{Name: ProviderOllama, DefaultEndpoint: DefaultOllamaEndpoint, Adapter: "openai_chat", Selectable: true, OfficialDocs: "https://docs.ollama.com/api/openai-compatibility"},
	}
}

// ProviderSpecByName returns a provider spec by normalized provider name.
func ProviderSpecByName(provider string) (ProviderSpec, bool) {
	provider = NormalizeProvider(provider)
	for _, spec := range ProviderSpecs() {
		if spec.Name == provider {
			return spec, true
		}
	}
	return ProviderSpec{}, false
}

// ResolveProviderRoute validates a cloud/provider route and loads the API key
// from environment or .harness/.env.local without exposing it in the returned
// serializable fields.
func ResolveProviderRoute(repoRoot string, route ProviderRoute) (ProviderRoute, error) {
	route.Routing = normalizeRouting(route.Routing)
	if route.Routing == "" {
		route.Routing = RoutingCloud
	}
	if route.Routing != RoutingCloud {
		return route, nil
	}
	route.Provider = NormalizeProvider(route.Provider)
	spec, ok := ProviderSpecByName(route.Provider)
	if !ok || route.Provider == ProviderRegistry {
		return ProviderRoute{}, fmt.Errorf("models: unsupported cloud provider %q", route.Provider)
	}
	if !spec.Selectable {
		return ProviderRoute{}, fmt.Errorf("models: provider %q is not selectable: %s", spec.Name, spec.UnavailableReason)
	}
	route.Model = strings.TrimSpace(route.Model)
	if route.Model == "" {
		return ProviderRoute{}, fmt.Errorf("models: --cloud-model or --model is required for provider %s", route.Provider)
	}
	route.Endpoint = strings.TrimSpace(route.Endpoint)
	if route.Endpoint == "" {
		route.Endpoint = spec.DefaultEndpoint
	}
	if route.Endpoint == "" && route.Provider == ProviderOpenAICompatible {
		return ProviderRoute{}, fmt.Errorf("models: --endpoint is required for provider %s", ProviderOpenAICompatible)
	}
	route.APIKeyEnv = strings.TrimSpace(route.APIKeyEnv)
	if route.APIKeyEnv == "" {
		route.APIKeyEnv = spec.DefaultAPIKeyEnv
	}
	if route.Provider != ProviderOllama && route.APIKeyEnv == "" {
		return ProviderRoute{}, fmt.Errorf("models: --api-key-env is required for provider %s", route.Provider)
	}
	if route.APIKeyEnv != "" {
		value, ok, err := LookupCredential(repoRoot, route.APIKeyEnv)
		if err != nil {
			return ProviderRoute{}, fmt.Errorf("models: read credential env %s: %w", route.APIKeyEnv, err)
		}
		if !ok {
			return ProviderRoute{}, fmt.Errorf("models: credential env %s is not set — export %s=<secret> or run `mars models credentials write-local-env --repo <path> --api-key-env %s --yes --json` with the same --repo path after exporting it", route.APIKeyEnv, route.APIKeyEnv, route.APIKeyEnv)
		}
		route.APIKey = value
	}
	return route, nil
}

func normalizeRouting(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case RoutingLocal:
		return RoutingLocal
	case RoutingCloud:
		return RoutingCloud
	case RoutingDefer:
		return RoutingDefer
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// LookupCredential reads an env var first, then .harness/.env.local.
func LookupCredential(repoRoot, envName string) (string, bool, error) {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return "", false, nil
	}
	if value := os.Getenv(envName); strings.TrimSpace(value) != "" {
		return value, true, nil
	}
	if strings.TrimSpace(repoRoot) == "" {
		return "", false, fmt.Errorf("models credentials: repository root is required for local credential fallback")
	}
	root, err := openModelRepository(repoRoot, "models credentials")
	if err != nil {
		return "", false, err
	}
	values, err := readLocalEnv(root, ".harness/.env.local")
	if err != nil {
		return "", false, err
	}
	value := strings.TrimSpace(values[envName])
	return value, value != "", nil
}

// WriteLocalCredential reads envName from the process environment and writes it
// to .harness/.env.local while recording only the env name in .env.example.
func WriteLocalCredential(repoRoot, envName string) (string, string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	envName = strings.TrimSpace(envName)
	if repoRoot == "" {
		return "", "", fmt.Errorf("models credentials: --repo is required")
	}
	if envName == "" {
		return "", "", fmt.Errorf("models credentials: --api-key-env is required")
	}
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return "", "", fmt.Errorf("models credentials: environment variable %s is not set — export %s=<secret> and retry", envName, envName)
	}
	root, err := openModelRepository(repoRoot, "models credentials")
	if err != nil {
		return "", "", err
	}
	if err := requireModelHarness(root, "models credentials"); err != nil {
		return "", "", err
	}
	const localRel = ".harness/.env.local"
	const exampleRel = ".harness/.env.example"

	localValues, err := readLocalEnv(root, localRel)
	if err != nil {
		return "", "", err
	}
	exampleValues, err := readLocalEnv(root, exampleRel)
	if err != nil {
		return "", "", err
	}
	localValues[envName] = value
	if err := writeEnvFile(root, localRel, localValues, 0o600, false); err != nil {
		return "", "", err
	}
	for key := range exampleValues {
		exampleValues[key] = ""
	}
	exampleValues[envName] = ""
	if err := writeEnvFile(root, exampleRel, exampleValues, 0o644, true); err != nil {
		return "", "", err
	}
	return filepath.Join(root.Abs(), filepath.FromSlash(localRel)), filepath.Join(root.Abs(), filepath.FromSlash(exampleRel)), nil
}

// EnsureEnvExample records a credential env var name without writing a value.
func EnsureEnvExample(repoRoot, envName string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	envName = strings.TrimSpace(envName)
	if repoRoot == "" || envName == "" {
		return "", nil
	}
	root, err := openModelRepository(repoRoot, "models credentials")
	if err != nil {
		return "", err
	}
	if err := requireModelHarness(root, "models credentials"); err != nil {
		return "", err
	}
	const exampleRel = ".harness/.env.example"
	exampleValues, err := readLocalEnv(root, exampleRel)
	if err != nil {
		return "", err
	}
	for key := range exampleValues {
		exampleValues[key] = ""
	}
	exampleValues[envName] = ""
	if err := writeEnvFile(root, exampleRel, exampleValues, 0o644, true); err != nil {
		return "", err
	}
	return filepath.Join(root.Abs(), filepath.FromSlash(exampleRel)), nil
}

func readLocalEnv(root *repofs.Root, rel string) (map[string]string, error) {
	values := map[string]string{}
	data, err := root.ReadFile(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return values, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models credentials: read %s: %w", rel, err)
	}
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("models credentials: parse %s line %d: expected KEY=VALUE", rel, lineNumber+1)
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values, nil
}

func writeEnvFile(root *repofs.Root, rel string, values map[string]string, perm os.FileMode, preserveExistingMode bool) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(values[key])
		b.WriteString("\n")
	}
	if err := writeModelRepositoryFile(root, rel, []byte(b.String()), perm, preserveExistingMode); err != nil {
		return fmt.Errorf("models credentials: write %s: %w", rel, err)
	}
	return nil
}
