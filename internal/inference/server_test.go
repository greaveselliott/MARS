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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/childenv"
	"github.com/greaveselliott/mars/internal/hardware"
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

func TestServer_startCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewServer(ServerConfig{BinaryPath: "/missing/llama-server", Port: 8080})
	err := s.Start(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, StateStopped, s.State())
}

func TestServer_startMissingBinaryMarksFailed(t *testing.T) {
	t.Parallel()

	s := NewServer(ServerConfig{BinaryPath: filepath.Join(t.TempDir(), "missing-llama-server"), Port: 8080})
	err := s.Start(context.Background())
	require.Error(t, err)
	require.Equal(t, StateFailed, s.State())
}

func TestServerManagedChildPreservesOrdinaryEnvironmentAndDropsCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake llama-server executable uses POSIX sh")
	}
	home := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "inference-env.txt")
	bin := filepath.Join(t.TempDir(), "llama-server")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nenv > \"$B1_INFERENCE_ENV_LOG\"\n"), 0o755))
	t.Setenv("HOME", home)
	t.Setenv("B1_INFERENCE_ENV_LOG", logPath)
	t.Setenv("GOCACHE", "/safe/go-cache")
	t.Setenv("GITHUB_TOKEN", "poison")
	t.Setenv(childenv.AllowlistVariable, "")

	s := NewServer(ServerConfig{BinaryPath: bin, ModelPath: "/models/test.gguf", Port: 28080})
	cmd, err := s.startCmdLocked()
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())
	child, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(child), "GOCACHE=/safe/go-cache")
	require.NotContains(t, string(child), "GITHUB_TOKEN=poison")
	require.NotContains(t, string(child), childenv.AllowlistVariable+"=")
}

func TestServer_restartBackoffAndContextLength(t *testing.T) {
	t.Parallel()

	require.Equal(t, time.Second, restartBackoff(1))
	require.Equal(t, 2*time.Second, restartBackoff(2))
	require.Equal(t, 30*time.Second, restartBackoff(10))
	require.Equal(t, 4096, effectiveContextLength(0))
	require.Equal(t, 4096, effectiveContextLength(-1))
	require.Equal(t, 8192, effectiveContextLength(8192))
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

func TestPortReservationDetectsLiveAndStaleLocks(t *testing.T) {
	t.Parallel()

	reservation, err := acquirePortReservation(25301)
	require.NoError(t, err)
	t.Cleanup(reservation.Release)

	_, err = acquirePortReservation(25301)
	var conflict *PortConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, 25301, conflict.Port)
	require.Equal(t, os.Getpid(), conflict.PID)
	require.Contains(t, conflict.Error(), "inference_port_conflict")

	stalePath := filepath.Join(os.TempDir(), "mars-inference-ports", "25302.lock")
	require.NoError(t, os.MkdirAll(filepath.Dir(stalePath), 0o755))
	require.NoError(t, os.WriteFile(stalePath, []byte("-1\n"), 0o644))
	staleReservation, err := acquirePortReservation(25302)
	require.NoError(t, err)
	t.Cleanup(staleReservation.Release)
	info := readPortLockInfo(stalePath)
	require.Equal(t, os.Getpid(), info.PID)
	require.Equal(t, "mars", info.Owner)
}

func TestRouterResolveServerPortAllocatesNextPortWhenLockIsUnhealthy(t *testing.T) {
	t.Parallel()

	first, err := acquirePortReservation(25310)
	require.NoError(t, err)
	t.Cleanup(first.Release)

	r := NewRouter(RouterConfig{PortBases: map[hardware.Tier]int{hardware.TierCoding: 25310}})
	port, reservation, reuse, err := r.resolveServerPort(context.Background(), hardware.TierCoding, "engineer")
	require.NoError(t, err)
	t.Cleanup(reservation.Release)
	require.Equal(t, 25320, port)
	require.False(t, reuse)
	require.NotNil(t, reservation)
}

func TestRouterResolveServerPortTreatsFreshInvalidLockAsActive(t *testing.T) {
	port := 25410
	lockPath := filepath.Join(os.TempDir(), "mars-inference-ports", fmt.Sprintf("%d.lock", port))
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPath), 0o755))
	require.NoError(t, os.WriteFile(lockPath, nil, 0o644))
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	r := NewRouter(RouterConfig{PortBases: map[hardware.Tier]int{hardware.TierCoding: port}})
	got, reservation, reuse, err := r.resolveServerPort(context.Background(), hardware.TierCoding, "engineer")
	require.NoError(t, err)
	t.Cleanup(reservation.Release)
	require.Equal(t, port+10, got)
	require.False(t, reuse)
	require.NotNil(t, reservation)
	require.FileExists(t, lockPath)
}

func TestRouterResolveServerPortReusesHealthyLockedEndpoint(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	meta := portReservationMetadata{
		Tier:       hardware.TierCoding,
		ModelPath:  "/models/coding.gguf",
		ModelName:  "coding",
		ContextLen: 4096,
		Parallel:   1,
	}
	lock, err := acquirePortReservation(port, meta)
	require.NoError(t, err)
	t.Cleanup(lock.Release)

	healthSrv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	healthLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	healthSrv.Listener = healthLn
	healthSrv.Start()
	t.Cleanup(healthSrv.Close)

	r := NewRouter(RouterConfig{PortBases: map[hardware.Tier]int{hardware.TierCoding: port}})
	got, reservation, reuse, err := r.resolveServerPort(context.Background(), hardware.TierCoding, "engineer", meta)
	require.NoError(t, err)
	require.Equal(t, port, got)
	require.True(t, reuse)
	require.Nil(t, reservation)
}

func TestRouterResolveServerPortDoesNotReuseMismatchedHealthyEndpoint(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	lock, err := acquirePortReservation(port, portReservationMetadata{
		Tier:       hardware.TierReasoning,
		ModelPath:  "/models/reasoning.gguf",
		ContextLen: 8192,
		Parallel:   1,
	})
	require.NoError(t, err)
	t.Cleanup(lock.Release)

	healthSrv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	healthLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	healthSrv.Listener = healthLn
	healthSrv.Start()
	t.Cleanup(healthSrv.Close)

	r := NewRouter(RouterConfig{PortBases: map[hardware.Tier]int{hardware.TierCoding: port}})
	got, reservation, reuse, err := r.resolveServerPort(context.Background(), hardware.TierCoding, "engineer", portReservationMetadata{
		Tier:       hardware.TierCoding,
		ModelPath:  "/models/coding.gguf",
		ContextLen: 4096,
		Parallel:   1,
	})
	require.NoError(t, err)
	t.Cleanup(reservation.Release)
	require.NotEqual(t, port, got)
	require.False(t, reuse)
	require.NotNil(t, reservation)
}

func TestPortConflictErrorIsTypedAndActionable(t *testing.T) {
	t.Parallel()

	err := &PortConflictError{Port: 18081, PID: 1234, Tier: hardware.TierReasoning, Role: "ceo"}
	var conflict *PortConflictError
	require.True(t, errors.As(err, &conflict))
	require.Contains(t, err.Error(), "inference_port_conflict")
	require.Contains(t, err.Error(), "port=18081")
	require.Contains(t, err.Error(), "owning_pid=1234")
	require.Contains(t, err.Error(), "tier=reasoning")
	require.Contains(t, err.Error(), "--model-endpoint")
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

func TestRouter_singleServerTierOverridesRoleAndManifestHints(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{
		SingleServerTier: hardware.TierCoding,
		RoleMapping: map[string]hardware.Tier{
			"ceo": hardware.TierReasoning,
		},
		Models: map[hardware.Tier]hardware.ModelSpec{
			hardware.TierCoding:    {Name: "c", File: "c.gguf", ContextLen: 32768},
			hardware.TierReasoning: {Name: "r", File: "r.gguf", ContextLen: 131072},
		},
		ModelsDir: t.TempDir(),
	})

	require.Equal(t, hardware.TierCoding, r.tierForRole("ceo"))
	require.Equal(t, hardware.TierCoding, r.tierForRoleModel("ceo", "reasoning"))
	require.Equal(t, 32768, r.ContextWindowForRoleModel("ceo", "reasoning"))
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
	require.ErrorContains(t, err, "run `mars setup`")
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

// TestRouter_contextWindowForRoleModel: the agent loop budgets against the
// window the tier actually serves (AD-288), so the router must report it per
// role/tier and return 0 when no local model spec is known.
func TestRouter_contextWindowForRoleModel(t *testing.T) {
	t.Parallel()

	r := NewRouter(RouterConfig{
		Models: map[hardware.Tier]hardware.ModelSpec{
			hardware.TierCoding:    {Name: "c", File: "c.gguf", ContextLen: 32768},
			hardware.TierReasoning: {Name: "r", File: "r.gguf", ContextLen: 131072},
		},
		ModelsDir: t.TempDir(),
	})

	require.Equal(t, 32768, r.ContextWindowForRoleModel("engineer", ""), "engineer defaults to the coding tier")
	require.Equal(t, 131072, r.ContextWindowForRoleModel("ceo", ""), "ceo defaults to the reasoning tier")
	require.Equal(t, 131072, r.ContextWindowForRoleModel("engineer", "reasoning"), "manifest model hint overrides the default tier")
	require.Equal(t, 0, r.ContextWindowForRoleModel("qa", ""), "unknown tier spec reports 0 (caller falls back to default)")

	var nilRouter *Router
	require.Equal(t, 0, nilRouter.ContextWindowForRoleModel("engineer", ""))
}

func TestServerContextLengthPreservesPerSlotWindow(t *testing.T) {
	t.Parallel()

	spec := hardware.ModelSpec{ContextLen: 32768}
	require.Equal(t, 32768, serverContextLength(spec, ServerTuning{Parallel: 1}))
	require.Equal(t, 65536, serverContextLength(spec, ServerTuning{Parallel: 2}))
	require.Equal(t, 4096, serverContextLength(hardware.ModelSpec{}, ServerTuning{}))
}

func TestRouter_serverForRoleModelMentionsInstalledVariant(t *testing.T) {
	t.Parallel()

	modelsDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(modelsDir, "Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf"), []byte("model"), 0o644))

	r := NewRouter(RouterConfig{
		Models: map[hardware.Tier]hardware.ModelSpec{
			hardware.TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf",
				ContextLen: 4096,
			},
		},
		ModelsDir: modelsDir,
	})

	_, err := r.ServerForRoleModel(context.Background(), "ceo", "reasoning")
	require.ErrorContains(t, err, "Installed variant(s) for the same model are present")
	require.ErrorContains(t, err, "Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf")
	require.ErrorContains(t, err, "performance_profile: quality")
}
