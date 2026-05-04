/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package inference

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/llm"
)

// Router maps role names to running inference servers.
type Router struct {
	mu            sync.RWMutex
	servers       map[hardware.Tier]*Server
	mapping       map[string]hardware.Tier // role name → tier
	models        map[hardware.Tier]hardware.ModelSpec
	modelsDir     string
	binaryPath    string
	tuning        ServerTuning
	fallback      *llm.Client // optional remote API fallback
	remoteBaseURL string      // normalized base URL (no trailing /v1)
}

// RouterConfig configures the router.
type RouterConfig struct {
	BinaryPath  string // path to llama-server binary
	Models      map[hardware.Tier]hardware.ModelSpec
	RoleMapping map[string]hardware.Tier // role → tier
	ModelsDir   string                   // where model files live
	FallbackURL string                   // optional remote API
	FallbackKey string
	Tuning      ServerTuning
}

// NewRouter creates a router (does not start servers).
func NewRouter(cfg RouterConfig) *Router {
	models := cfg.Models
	if models == nil {
		models = map[hardware.Tier]hardware.ModelSpec{}
	}
	mapping := cfg.RoleMapping
	if mapping == nil {
		mapping = map[string]hardware.Tier{}
	}

	var fb *llm.Client
	fbBase := normalizeAPIBase(cfg.FallbackURL)
	if fbBase != "" {
		c, err := llm.NewClient(llm.Config{
			BaseURL: fbBase,
			APIKey:  strings.TrimSpace(cfg.FallbackKey),
		})
		if err != nil {
			slog.Warn("inference router: invalid fallback URL; remote fallback disabled", "err", err)
		} else {
			fb = c
		}
	}

	return &Router{
		servers:       make(map[hardware.Tier]*Server),
		mapping:       mapping,
		models:        models,
		modelsDir:     cfg.ModelsDir,
		binaryPath:    cfg.BinaryPath,
		tuning:        cfg.Tuning,
		fallback:      fb,
		remoteBaseURL: fbBase,
	}
}

func normalizeAPIBase(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	u = strings.TrimRight(u, "/")
	u = strings.TrimSuffix(u, "/v1")
	return u
}

// DefaultRoleTierMapping returns conservative defaults for built-in starter roles.
// Manifest role.model remains the source of truth when available; this mapping
// exists for older callers and custom roles without tier hints.
func DefaultRoleTierMapping() map[string]hardware.Tier {
	return map[string]hardware.Tier{
		"ceo":                   hardware.TierReasoning,
		"coo":                   hardware.TierReasoning,
		"cto":                   hardware.TierReasoning,
		"cto-weekly":            hardware.TierReasoning,
		"engineer":              hardware.TierCoding,
		"pipeline-fixer":        hardware.TierCoding,
		"reviewer":              hardware.TierReasoning,
		"code-reviewer":         hardware.TierReasoning,
		"qa":                    hardware.TierFast,
		"documenter":            hardware.TierFast,
		"docs-writer":           hardware.TierFast,
		"release":               hardware.TierFast,
		"release-manager":       hardware.TierReasoning,
		"triager":               hardware.TierFast,
		"onboarder":             hardware.TierFast,
		"auditor":               hardware.TierReasoning,
		"security":              hardware.TierReasoning,
		"security-auditor":      hardware.TierReasoning,
		"backlog":               hardware.TierFast,
		"janitor":               hardware.TierFast,
		"dogfood":               hardware.TierCoding,
		"evolution":             hardware.TierReasoning,
		"dependency-manager":    hardware.TierFast,
		"dependency-updater":    hardware.TierFast,
		"performance-optimizer": hardware.TierCoding,
		"refactorer":            hardware.TierCoding,
		"incident-responder":    hardware.TierCoding,
	}
}

// TierForRoleModel resolves a manifest role.model hint into a hardware tier.
func TierForRoleModel(role, modelHint string) hardware.Tier {
	switch strings.ToLower(strings.TrimSpace(modelHint)) {
	case string(hardware.TierCoding):
		return hardware.TierCoding
	case string(hardware.TierReasoning):
		return hardware.TierReasoning
	case string(hardware.TierFast):
		return hardware.TierFast
	default:
		if tier, ok := DefaultRoleTierMapping()[role]; ok {
			return tier
		}
		return hardware.TierCoding
	}
}

func (r *Router) tierForRole(role string) hardware.Tier {
	if t, ok := r.mapping[role]; ok {
		return t
	}
	return TierForRoleModel(role, "")
}

func (r *Router) tierPort(tier hardware.Tier) int {
	switch tier {
	case hardware.TierCoding:
		return 18080
	case hardware.TierReasoning:
		return 18081
	case hardware.TierFast:
		return 18082
	default:
		return 18089
	}
}

func (r *Router) fallbackBase() string {
	if r.fallback == nil {
		return ""
	}
	return r.remoteBaseURL
}

// ServerForRole returns the base URL for the server handling this role.
// Starts the server if not already running. Verifies the server is actually
// responsive before returning — catches stale "healthy" state from crashed servers.
func (r *Router) ServerForRole(ctx context.Context, role string) (string, error) {
	return r.serverForTier(ctx, role, r.tierForRole(role))
}

// ServerForRoleModel returns the base URL for a role using its manifest model tier.
func (r *Router) ServerForRoleModel(ctx context.Context, role, modelHint string) (string, error) {
	return r.serverForTier(ctx, role, TierForRoleModel(role, modelHint))
}

func (r *Router) serverForTier(ctx context.Context, role string, tier hardware.Tier) (string, error) {
	spec, modelOK := r.models[tier]
	modelPath := ""
	expectedPath := ""
	if modelOK && strings.TrimSpace(spec.File) != "" {
		candidate := filepath.Join(r.modelsDir, spec.File)
		expectedPath = candidate
		if _, err := os.Stat(candidate); err == nil {
			modelPath = candidate
		}
	}

	if modelPath == "" {
		base := r.fallbackBase()
		if base == "" {
			if expectedPath != "" {
				detail := r.installedModelVariantHint(spec)
				if detail != "" {
					detail = " " + detail
				}
				return "", fmt.Errorf("inference: local model for tier %q is missing at %s and no remote fallback configured.%s Run `mars-harness setup` to download the %s model, set `performance_profile: quality` in ~/.mars-harness/config.yaml to keep using an installed larger local model, or configure a remote fallback", tier, expectedPath, detail, tier)
			}
			return "", fmt.Errorf("inference: no local model configured for tier %q and no remote fallback configured — run `mars-harness setup` or configure a remote fallback", tier)
		}
		return base, nil
	}

	r.mu.Lock()
	srv := r.servers[tier]
	if srv == nil {
		ctxLen := spec.ContextLen
		if ctxLen <= 0 {
			ctxLen = effectiveContextLength(0)
		}
		srv = NewServer(ServerConfig{
			BinaryPath:    r.binaryPath,
			ModelPath:     modelPath,
			Port:          r.tierPort(tier),
			ContextLength: ctxLen,
			GPULayers:     -1,
			ServerTuning:  r.tuning,
		})
		r.servers[tier] = srv
	}
	r.mu.Unlock()

	if err := srv.Start(ctx); err != nil {
		if base := r.fallbackBase(); base != "" {
			slog.Warn("inference router: local server start failed; using remote fallback",
				"tier", string(tier),
				"role", role,
				"err", err,
			)
			return base, nil
		}
		return "", err
	}

	if !srv.Healthy() {
		slog.Warn("inference router: server reports healthy but failed spot check; restarting",
			"tier", string(tier), "role", role, "port", r.tierPort(tier))
		r.mu.Lock()
		delete(r.servers, tier)
		r.mu.Unlock()
		_ = srv.Stop()
		return r.serverForTier(ctx, role, tier)
	}

	return srv.BaseURL(), nil
}

func (r *Router) installedModelVariantHint(spec hardware.ModelSpec) string {
	if strings.TrimSpace(r.modelsDir) == "" || strings.TrimSpace(spec.Name) == "" {
		return ""
	}

	entries, err := os.ReadDir(r.modelsDir)
	if err != nil {
		return ""
	}

	prefix := strings.TrimSuffix(spec.Name, "-Instruct")
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == spec.File {
			continue
		}
		if !strings.HasSuffix(name, ".gguf") {
			continue
		}
		if strings.Contains(name, spec.Name) || (prefix != "" && strings.Contains(name, prefix)) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	return fmt.Sprintf("Installed variant(s) for the same model are present: %s.", strings.Join(matches, ", "))
}

// StopAll gracefully stops all managed servers.
func (r *Router) StopAll() {
	r.mu.Lock()
	list := make([]*Server, 0, len(r.servers))
	for _, srv := range r.servers {
		if srv != nil {
			list = append(list, srv)
		}
	}
	r.mu.Unlock()

	for _, srv := range list {
		if err := srv.Stop(); err != nil {
			slog.Warn("inference router: stop server failed", "err", err)
		}
	}
	slog.Info("inference router: stopped managed servers", "count", len(list))
}

// RestartAll kills every running inference server and clears the
// server map so they are re-launched on the next request. Use after
// the machine wakes from sleep when connections are stale.
func (r *Router) RestartAll() {
	r.mu.Lock()
	list := make([]*Server, 0, len(r.servers))
	for _, srv := range r.servers {
		if srv != nil {
			list = append(list, srv)
		}
	}
	r.servers = make(map[hardware.Tier]*Server, len(r.servers))
	r.mu.Unlock()

	for _, srv := range list {
		if err := srv.Stop(); err != nil {
			slog.Warn("inference router: restart stop failed", "err", err)
		}
	}
	slog.Info("inference router: restarted — servers will re-launch on next request", "stopped", len(list))
}
