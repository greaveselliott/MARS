/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/design-docs/context-efficiency.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package inference

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/llm"
)

// Router maps role names to running inference servers.
type Router struct {
	mu             sync.RWMutex
	servers        map[hardware.Tier]*Server
	reservations   map[hardware.Tier]*portReservation
	mapping        map[string]hardware.Tier // role name → tier
	models         map[hardware.Tier]hardware.ModelSpec
	modelsDir      string
	binaryPath     string
	tuning         ServerTuning
	singleTier     hardware.Tier
	portBases      map[hardware.Tier]int
	fallback       *llm.Client // optional remote API fallback
	remoteBaseURL  string      // normalized base URL (no trailing /v1)
	remoteAPIKey   string
	remoteProvider string
	remoteModel    string
}

// RouterConfig configures the router.
type RouterConfig struct {
	BinaryPath  string // path to llama-server binary
	Models      map[hardware.Tier]hardware.ModelSpec
	RoleMapping map[string]hardware.Tier // role → tier
	ModelsDir   string                   // where model files live
	// SingleServerTier forces every role and manifest model hint onto one tier.
	// It is used by validation lanes that need parallel role execution through a
	// single local llama-server instead of starting one server per model tier.
	SingleServerTier hardware.Tier
	PortBases        map[hardware.Tier]int // optional test/runtime overrides for tier base ports
	FallbackURL      string                // optional remote API
	FallbackKey      string
	FallbackProvider string
	FallbackModel    string
	Tuning           ServerTuning
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
		servers:        make(map[hardware.Tier]*Server),
		reservations:   make(map[hardware.Tier]*portReservation),
		mapping:        mapping,
		models:         models,
		modelsDir:      cfg.ModelsDir,
		binaryPath:     cfg.BinaryPath,
		tuning:         cfg.Tuning,
		singleTier:     cfg.SingleServerTier,
		portBases:      cfg.PortBases,
		fallback:       fb,
		remoteBaseURL:  fbBase,
		remoteAPIKey:   strings.TrimSpace(cfg.FallbackKey),
		remoteProvider: strings.TrimSpace(cfg.FallbackProvider),
		remoteModel:    strings.TrimSpace(cfg.FallbackModel),
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
	if r.singleTier != "" {
		return r.singleTier
	}
	if t, ok := r.mapping[role]; ok {
		return t
	}
	return TierForRoleModel(role, "")
}

func (r *Router) tierForRoleModel(role, modelHint string) hardware.Tier {
	if r.singleTier != "" {
		return r.singleTier
	}
	return TierForRoleModel(role, modelHint)
}

func (r *Router) tierPort(tier hardware.Tier) int {
	if r != nil && r.portBases != nil {
		if port, ok := r.portBases[tier]; ok && port > 0 {
			return port
		}
	}
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
	return r.serverForTier(ctx, role, r.tierForRoleModel(role, modelHint))
}

// ClientConfigForRoleModel resolves the endpoint and auth metadata for a role.
func (r *Router) ClientConfigForRoleModel(ctx context.Context, role, modelHint string) (llm.Config, error) {
	endpoint, err := r.ServerForRoleModel(ctx, role, modelHint)
	if err != nil {
		return llm.Config{}, err
	}
	cfg := llm.Config{BaseURL: endpoint, Model: modelHint}
	if r.remoteBaseURL != "" && strings.TrimRight(endpoint, "/") == strings.TrimRight(r.remoteBaseURL, "/") {
		cfg.APIKey = r.remoteAPIKey
		cfg.Provider = r.remoteProvider
		if r.remoteModel != "" {
			cfg.Model = r.remoteModel
		}
	}
	return cfg, nil
}

// ContextWindowForRoleModel returns the context window (tokens) actually
// served for the tier this role resolves to, so agent-loop budgeting can
// clamp to the real serving window instead of assuming a default (AD-288).
// Returns 0 when unknown (no local model spec, e.g. remote fallback).
func (r *Router) ContextWindowForRoleModel(role, modelHint string) int {
	if r == nil {
		return 0
	}
	tier := r.tierForRoleModel(role, modelHint)
	spec, ok := r.models[tier]
	if !ok || spec.ContextLen <= 0 {
		return 0
	}
	return spec.ContextLen
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
				return "", fmt.Errorf("inference: local model for tier %q is missing at %s and no remote fallback configured.%s Run `mars setup` to download the %s model, set `performance_profile: quality` in ~/.mars/config.yaml to keep using an installed larger local model, or configure a remote fallback", tier, expectedPath, detail, tier)
			}
			return "", fmt.Errorf("inference: no local model configured for tier %q and no remote fallback configured — run `mars setup` or configure a remote fallback", tier)
		}
		return base, nil
	}

	r.mu.Lock()
	srv := r.servers[tier]
	if srv == nil {
		contextLen := serverContextLength(spec, r.tuning)
		port, reservation, reuse, err := r.resolveServerPort(ctx, tier, role, portReservationMetadata{
			Tier:       tier,
			ModelPath:  modelPath,
			ModelName:  spec.Name,
			ContextLen: contextLen,
			Parallel:   r.tuning.Parallel,
		})
		if err != nil {
			r.mu.Unlock()
			return "", err
		}
		if reuse {
			r.mu.Unlock()
			return fmt.Sprintf("http://localhost:%d", port), nil
		}
		srv = NewServer(ServerConfig{
			BinaryPath:    r.binaryPath,
			ModelPath:     modelPath,
			Port:          port,
			ContextLength: contextLen,
			GPULayers:     -1,
			ServerTuning:  r.tuning,
		})
		r.servers[tier] = srv
		r.reservations[tier] = reservation
	}
	r.mu.Unlock()

	if err := srv.Start(ctx); err != nil {
		r.releaseReservation(tier)
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
			"tier", string(tier), "role", role, "port", srv.cfg.Port)
		r.mu.Lock()
		delete(r.servers, tier)
		r.mu.Unlock()
		r.releaseReservation(tier)
		_ = srv.Stop()
		return r.serverForTier(ctx, role, tier)
	}

	return srv.BaseURL(), nil
}

func (r *Router) resolveServerPort(ctx context.Context, tier hardware.Tier, role string, metas ...portReservationMetadata) (int, *portReservation, bool, error) {
	start := r.tierPort(tier)
	var meta portReservationMetadata
	if len(metas) > 0 {
		meta = metas[0]
	}
	var lastConflict *PortConflictError
	for i := 0; i < defaultTierPortRange; i++ {
		port := start + (i * 10)
		reservation, err := acquirePortReservation(port, meta)
		if err != nil {
			var conflict *PortConflictError
			if errors.As(err, &conflict) {
				conflict.Tier = tier
				conflict.Role = role
				if compatiblePortLock(conflict, meta) && healthyLocalEndpoint(ctx, port) {
					return port, nil, true, nil
				}
				lastConflict = conflict
				continue
			}
			return 0, nil, false, err
		}
		available, pid := portAvailable(port)
		if available {
			return port, reservation, false, nil
		}
		reservation.Release()
		lastConflict = &PortConflictError{Port: port, PID: pid, Tier: tier, Role: role}
	}
	if lastConflict != nil {
		return 0, nil, false, lastConflict
	}
	return 0, nil, false, &PortConflictError{Port: start, Tier: tier, Role: role}
}

func compatiblePortLock(conflict *PortConflictError, meta portReservationMetadata) bool {
	if conflict == nil {
		return false
	}
	if strings.TrimSpace(conflict.Owner) != "mars" {
		return false
	}
	if conflict.LockedTier != meta.Tier {
		return false
	}
	if strings.TrimSpace(conflict.ModelPath) == "" || strings.TrimSpace(meta.ModelPath) == "" {
		return false
	}
	if filepath.Clean(conflict.ModelPath) != filepath.Clean(meta.ModelPath) {
		return false
	}
	if conflict.ContextLen > 0 && meta.ContextLen > 0 && conflict.ContextLen != meta.ContextLen {
		return false
	}
	if conflict.Parallel > 0 && meta.Parallel > 0 && conflict.Parallel != meta.Parallel {
		return false
	}
	return true
}

func (r *Router) releaseReservation(tier hardware.Tier) {
	r.mu.Lock()
	reservation := r.reservations[tier]
	delete(r.reservations, tier)
	delete(r.servers, tier)
	r.mu.Unlock()
	if reservation != nil {
		reservation.Release()
	}
}

func serverContextLength(spec hardware.ModelSpec, tuning ServerTuning) int {
	ctxLen := spec.ContextLen
	if ctxLen <= 0 {
		ctxLen = effectiveContextLength(0)
	}
	if tuning.Parallel > 1 {
		return ctxLen * tuning.Parallel
	}
	return ctxLen
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
	reservations := make([]*portReservation, 0, len(r.reservations))
	for _, reservation := range r.reservations {
		if reservation != nil {
			reservations = append(reservations, reservation)
		}
	}
	r.reservations = make(map[hardware.Tier]*portReservation)
	r.mu.Unlock()

	for _, srv := range list {
		if err := srv.Stop(); err != nil {
			slog.Warn("inference router: stop server failed", "err", err)
		}
	}
	for _, reservation := range reservations {
		reservation.Release()
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
	reservations := make([]*portReservation, 0, len(r.reservations))
	for _, reservation := range r.reservations {
		if reservation != nil {
			reservations = append(reservations, reservation)
		}
	}
	r.servers = make(map[hardware.Tier]*Server, len(r.servers))
	r.reservations = make(map[hardware.Tier]*portReservation)
	r.mu.Unlock()

	for _, srv := range list {
		if err := srv.Stop(); err != nil {
			slog.Warn("inference router: restart stop failed", "err", err)
		}
	}
	for _, reservation := range reservations {
		reservation.Release()
	}
	slog.Info("inference router: restarted — servers will re-launch on next request", "stopped", len(list))
}
