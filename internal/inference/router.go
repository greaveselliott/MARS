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

func (r *Router) tierForRole(role string) hardware.Tier {
	if t, ok := r.mapping[role]; ok {
		return t
	}
	return hardware.TierCoding
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
	tier := r.tierForRole(role)

	spec, modelOK := r.models[tier]
	modelPath := ""
	if modelOK && strings.TrimSpace(spec.File) != "" {
		candidate := filepath.Join(r.modelsDir, spec.File)
		if _, err := os.Stat(candidate); err == nil {
			modelPath = candidate
		}
	}

	if modelPath == "" {
		base := r.fallbackBase()
		if base == "" {
			return "", fmt.Errorf("inference: no local model for tier %q and no remote fallback configured", tier)
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
		return r.ServerForRole(ctx, role)
	}

	return srv.BaseURL(), nil
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
