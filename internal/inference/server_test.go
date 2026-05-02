package inference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/stretchr/testify/require"
)

func TestServer_configBuildsCorrectArgs(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		BinaryPath:    "/opt/llama-server",
		ModelPath:     "/models/model.gguf",
		Port:          9000,
		ContextLength: 8192,
		GPULayers:     -1,
		ServerTuning: ServerTuning{
			Threads:        8,
			ThreadsBatch:   6,
			Parallel:       1,
			BatchSize:      1024,
			UBatchSize:     256,
			FlashAttention: "auto",
			MLock:          true,
		},
	}
	require.Equal(t, []string{
		"--model", "/models/model.gguf",
		"--port", "9000",
		"--ctx-size", "8192",
		"--n-gpu-layers", "-1",
		"-t", "8",
		"--threads-batch", "6",
		"--parallel", "1",
		"--batch-size", "1024",
		"--ubatch-size", "256",
		"--flash-attn", "auto",
		"--mlock",
	}, llamaServerArgs(cfg))

	cfg.ServerTuning = ServerTuning{}
	require.Equal(t, []string{
		"--model", "/models/model.gguf",
		"--port", "9000",
		"--ctx-size", "8192",
		"--n-gpu-layers", "-1",
	}, llamaServerArgs(cfg))

	cfg.ContextLength = 0
	require.Equal(t, []string{
		"--model", "/models/model.gguf",
		"--port", "9000",
		"--ctx-size", "4096",
		"--n-gpu-layers", "-1",
	}, llamaServerArgs(cfg))

	cfg.FlashAttention = "not-valid"
	require.NotContains(t, llamaServerArgs(cfg), "--flash-attn")
}

func TestServer_baseURL(t *testing.T) {
	t.Parallel()

	s := NewServer(ServerConfig{Port: 8080})
	require.Equal(t, "http://localhost:8080", s.BaseURL())
}

func TestServer_stateTransitions(t *testing.T) {
	t.Parallel()

	s := NewServer(ServerConfig{Port: 8080})
	require.Equal(t, StateStopped, s.State())
}

func TestServer_healthCheck(t *testing.T) {
	t.Parallel()

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)

	u, err := url.Parse(okSrv.URL)
	require.NoError(t, err)
	port := u.Port()
	require.NotEmpty(t, port)

	p, err := strconv.Atoi(port)
	require.NoError(t, err)
	s := NewServer(ServerConfig{Port: p})
	require.True(t, s.Healthy())

	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(badSrv.Close)

	u2, err := url.Parse(badSrv.URL)
	require.NoError(t, err)
	p2, err := strconv.Atoi(u2.Port())
	require.NoError(t, err)
	s2 := NewServer(ServerConfig{Port: p2})
	require.False(t, s2.Healthy())
}

func TestRouter_defaultsToCodingTier(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{
		RoleMapping: map[string]hardware.Tier{
			"known": hardware.TierReasoning,
		},
	})
	require.Equal(t, hardware.TierReasoning, r.tierForRole("known"))
	require.Equal(t, hardware.TierCoding, r.tierForRole("unknown-role"))
}

func TestTierForRoleModel_usesManifestTierBeforeRoleFallback(t *testing.T) {
	t.Parallel()

	require.Equal(t, hardware.TierReasoning, TierForRoleModel("ceo", ""))
	require.Equal(t, hardware.TierFast, TierForRoleModel("ceo", "fast"))
	require.Equal(t, hardware.TierCoding, TierForRoleModel("qa", "coding"))
	require.Equal(t, hardware.TierCoding, TierForRoleModel("custom-role", "unknown-model"))
}

func TestDefaultRoleTierMapping_coversStarterRoles(t *testing.T) {
	t.Parallel()

	mapping := DefaultRoleTierMapping()
	require.Equal(t, hardware.TierReasoning, mapping["ceo"])
	require.Equal(t, hardware.TierReasoning, mapping["coo"])
	require.Equal(t, hardware.TierReasoning, mapping["cto-weekly"])
	require.Equal(t, hardware.TierFast, mapping["qa"])
	require.Equal(t, hardware.TierReasoning, mapping["security"])
	require.Equal(t, hardware.TierFast, mapping["dependency-manager"])
	require.Equal(t, hardware.TierCoding, mapping["dogfood"])
}

func TestRouter_stopAllSafe(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{})
	require.NotPanics(t, func() { r.StopAll() })
}

func TestRouter_serverForRoleUsesFallbackWhenModelMissing(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{
		Models: map[hardware.Tier]hardware.ModelSpec{
			hardware.TierCoding: {File: "missing.gguf", ContextLen: 4096},
		},
		ModelsDir:   t.TempDir(),
		FallbackURL: "https://example.com/v1/",
	})

	base, err := r.ServerForRole(context.Background(), "any-role")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", base)
}

func TestRouter_serverForRoleErrorsWithoutModelOrFallback(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{
		Models: map[hardware.Tier]hardware.ModelSpec{
			hardware.TierReasoning: {File: "model.gguf", ContextLen: 4096},
		},
		ModelsDir: t.TempDir(),
	})

	_, err := r.ServerForRole(context.Background(), "unknown-role")
	require.Error(t, err)
	require.ErrorContains(t, err, "run `mars-harness setup`")
}

func TestRouter_serverForRoleModelUsesManifestTierInError(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{
		Models: map[hardware.Tier]hardware.ModelSpec{
			hardware.TierFast: {File: "fast.gguf", ContextLen: 4096},
		},
		ModelsDir: t.TempDir(),
	})

	_, err := r.ServerForRoleModel(context.Background(), "ceo", "fast")
	require.ErrorContains(t, err, `tier "fast"`)
	require.ErrorContains(t, err, "fast.gguf")
}
