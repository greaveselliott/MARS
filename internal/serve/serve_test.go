/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/design-docs/local-inference.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/board-driven-integrations.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-013-board-driven-integrations.md
- docs/features/F-010-dashboard-control-plane.md
*/
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/bundle"
	jiraintegration "github.com/greaveselliott/mars/internal/jira"
	"github.com/greaveselliott/mars/internal/queue"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/scheduler"
	"github.com/greaveselliott/mars/internal/scoring"
	ticketstate "github.com/greaveselliott/mars/internal/tickets"
	"github.com/stretchr/testify/require"
)

type flushSignalWriter struct {
	header  http.Header
	flushed chan struct{}
	once    sync.Once
}

func (w *flushSignalWriter) Header() http.Header         { return w.header }
func (w *flushSignalWriter) WriteHeader(int)             {}
func (w *flushSignalWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *flushSignalWriter) Flush()                      { w.once.Do(func() { close(w.flushed) }) }

func testDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free TCP addr: %v", err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

func waitForHTTPStatus(t *testing.T, url string, want int) {
	t.Helper()
	client := http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == want {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to return %d: %v", url, want, lastErr)
}

func TestServer_healthHandler_healthy(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	srv.health.Store(true)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %q", body["status"])
	}
}

func TestServerNewModelEndpointSkipsLocalModelPreflight(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelsDir:     filepath.Join(t.TempDir(), "missing-models"),
		BinDir:        filepath.Join(t.TempDir(), "missing-bin"),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	if err != nil {
		t.Fatalf("New returned error with model endpoint override: %v", err)
	}
	if srv.router == nil {
		t.Fatal("expected inference router")
	}
	got, err := srv.router.ServerForRole(context.Background(), "ceo")
	if err != nil {
		t.Fatalf("expected endpoint-only router to return fallback endpoint: %v", err)
	}
	if got != "http://127.0.0.1:9999" {
		t.Fatalf("fallback endpoint = %q, want normalized endpoint", got)
	}
}

func TestServerNewRequiresLocalModelPreflightWhenConfigured(t *testing.T) {
	_, err := New(Config{
		WebhookAddr:           "127.0.0.1:0",
		DashboardAddr:         "127.0.0.1:0",
		DBPath:                testDBPath(t),
		ModelsDir:             filepath.Join(t.TempDir(), "missing-models"),
		BinDir:                filepath.Join(t.TempDir(), "missing-bin"),
		RequireModelPreflight: true,
	})
	if err == nil {
		t.Fatal("expected missing local model preflight error")
	}
	if !strings.Contains(err.Error(), "missing model file") {
		t.Fatalf("expected missing model file error, got %v", err)
	}
}

func TestServerNewRepoDefaultCloudRoutingSkipsLocalPreflight(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", "model-overrides.yaml"), []byte(`version: 2
default:
  routing: cloud
  provider: openai-compatible
  endpoint: http://127.0.0.1:9999/v1
  model: route-test
  api_key_env: TEST_PROVIDER_KEY
`), 0o644))
	t.Setenv("TEST_PROVIDER_KEY", "secret-value")

	srv, err := New(Config{
		WebhookAddr:           "127.0.0.1:0",
		DashboardAddr:         "127.0.0.1:0",
		DBPath:                testDBPath(t),
		RepoScope:             repo,
		ModelsDir:             filepath.Join(t.TempDir(), "missing-models"),
		BinDir:                filepath.Join(t.TempDir(), "missing-bin"),
		RequireModelPreflight: true,
	})
	require.NoError(t, err)
	got, err := srv.router.ServerForRole(context.Background(), "ceo")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9999", got)
	clientCfg, err := srv.router.ClientConfigForRoleModel(context.Background(), "ceo", "")
	require.NoError(t, err)
	require.Equal(t, "openai-compatible", clientCfg.Provider)
	require.Equal(t, "route-test", clientCfg.Model)
	require.Equal(t, "secret-value", clientCfg.APIKey)
}

func TestServerNewRepoDefaultDeferredRoutingBlocksRuntime(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", "model-overrides.yaml"), []byte(`version: 2
default:
  routing: defer
`), 0o644))

	_, err := New(Config{
		WebhookAddr:           "127.0.0.1:0",
		DashboardAddr:         "127.0.0.1:0",
		DBPath:                testDBPath(t),
		RepoScope:             repo,
		ModelsDir:             filepath.Join(t.TempDir(), "missing-models"),
		BinDir:                filepath.Join(t.TempDir(), "missing-bin"),
		RequireModelPreflight: true,
	})
	require.ErrorContains(t, err, "model routing is deferred")
	require.ErrorContains(t, err, "configure cloud routing")
}

func TestServer_healthHandler_unhealthy(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	srv.health.Store(false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "unhealthy" {
		t.Errorf("expected status=unhealthy, got %q", body["status"])
	}
}

func TestServerPauseResumeAndStatus(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if srv.IsPaused() {
		t.Fatal("expected new server to start unpaused")
	}
	srv.Pause()
	if !srv.IsPaused() {
		t.Fatal("expected Pause to mark worker pool paused")
	}
	srv.Resume()
	if srv.IsPaused() {
		t.Fatal("expected Resume to clear worker pool pause")
	}

	status := srv.Status()
	if status.Paused {
		t.Fatalf("expected status paused=false after resume: %+v", status)
	}
	if len(status.Repos) != 0 {
		t.Fatalf("expected no repos for fresh server: %+v", status.Repos)
	}

	view := srv.buildPipelineView()
	if view.Mode != "none" || !strings.Contains(view.Description, "No repositories") {
		t.Fatalf("expected no-repo pipeline view, got %+v", view)
	}
}

func TestDashboardRoleNodesOrdersAndFiltersRoles(t *testing.T) {
	manifest := &bundle.Manifest{Roles: map[string]bundle.RoleConfig{
		"orchestrator": {Domain: "orchestrator", Mode: "routing"},
		"custom":       {Domain: "planner", Mode: "custom"},
		"qa":           {Domain: "reviewer", Mode: "quality"},
		"engineer":     {Domain: "engineer", Mode: "ticket-delivery"},
	}}

	if got := dashboardRoleNodes(nil, true); got != nil {
		t.Fatalf("expected nil nodes for nil manifest, got %+v", got)
	}
	nodes := dashboardRoleNodes(manifest, true)
	if len(nodes) != 3 {
		t.Fatalf("expected orchestrator filtered out, got %+v", nodes)
	}
	names := []string{nodes[0].Name, nodes[1].Name, nodes[2].Name}
	if strings.Join(names, ",") != "engineer,qa,custom" {
		t.Fatalf("unexpected dashboard role order: %+v", nodes)
	}
	if nodes[0].Domain != "engineer" || nodes[0].Mode != "ticket-delivery" {
		t.Fatalf("expected role metadata copied into node: %+v", nodes[0])
	}
	if roleDashboardOrder("unknown-role") <= roleDashboardOrder("janitor") {
		t.Fatal("expected unknown roles after known dashboard roles")
	}
}

func TestServer_qualityScoreAPIServesRepoArtifact(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(repo, "docs", "QUALITY_SCORE.md"), []byte("# Quality Score\n\nGenerated.\n"))
	srv := &Server{cfg: Config{RepoScope: repo}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/quality-score", nil)
	srv.handleQualityScoreAPI(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "# Quality Score") {
		t.Fatalf("expected quality score body, got %q", rec.Body.String())
	}
}

func TestNew_missingAddr(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for missing WebhookAddr")
	}
}

func TestServer_startStop(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	require.Eventually(t, srv.Healthy, 3*time.Second, 25*time.Millisecond, "expected server to be healthy after start")

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected clean shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}

	if srv.Healthy() {
		t.Error("expected server to be unhealthy after stop")
	}
}

func TestServer_startUsesEphemeralHTTPFallbackWhenDefaultPortsBusy(t *testing.T) {
	busyWebhook, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("busy webhook listen: %v", err)
	}
	defer busyWebhook.Close()
	busyDashboard, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("busy dashboard listen: %v", err)
	}
	defer busyDashboard.Close()

	webhookAddr := busyWebhook.Addr().String()
	dashboardAddr := busyDashboard.Addr().String()
	srv, err := New(Config{
		WebhookAddr:           webhookAddr,
		DashboardAddr:         dashboardAddr,
		DBPath:                testDBPath(t),
		EphemeralHTTPFallback: true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = srv.Stop(context.Background())
	})
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if srv.Healthy() {
			break
		}
		select {
		case err := <-errCh:
			t.Fatalf("Start returned before fallback became healthy: %v", err)
		default:
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !srv.Healthy() {
		t.Fatal("expected server to be healthy on ephemeral fallback ports")
	}
	if srv.http.Addr == webhookAddr {
		t.Fatalf("expected webhook addr to fall back from busy %s", webhookAddr)
	}
	if srv.dashHTTP.Addr == dashboardAddr {
		t.Fatalf("expected dashboard addr to fall back from busy %s", dashboardAddr)
	}
	waitForHTTPStatus(t, "http://"+srv.dashHTTP.Addr+"/api/status", http.StatusOK)

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected clean shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}
}

func TestServer_dashboardStopEndpointStopsStart(t *testing.T) {
	const controlSecret = "0123456789abcdef0123456789abcdef"
	webhookAddr := freeTCPAddr(t)
	dashboardAddr := freeTCPAddr(t)
	srv, err := New(Config{
		WebhookAddr:            webhookAddr,
		DashboardAddr:          dashboardAddr,
		DashboardControlSecret: controlSecret,
		DBPath:                 testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = srv.Stop(context.Background())
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	waitForHTTPStatus(t, "http://"+dashboardAddr+"/api/status", http.StatusOK)

	origin := "http://" + dashboardAddr
	loginReq, _ := http.NewRequest(http.MethodPost, origin+"/api/login", strings.NewReader(`{"secret":"`+controlSecret+`"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set("Origin", origin)
	loginResp, err := http.DefaultClient.Do(loginReq)
	if err != nil {
		t.Fatalf("POST /api/login: %v", err)
	}
	var login struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&login); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	_ = loginResp.Body.Close()
	stopReq, _ := http.NewRequest(http.MethodPost, origin+"/api/stop", nil)
	stopReq.Header.Set("Origin", origin)
	stopReq.Header.Set("X-MARS-CSRF-Token", login.CSRF)
	for _, cookie := range loginResp.Cookies() {
		stopReq.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(stopReq)
	if err != nil {
		t.Fatalf("POST /api/stop: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/stop status = %d, want 200", resp.StatusCode)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected dashboard stop to end Start cleanly, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("dashboard stop did not end Start within 5 seconds")
	}
}

func TestServerDirectRestartInvalidatesDashboardSessionAndSSE(t *testing.T) {
	const controlSecret = "0123456789abcdef0123456789abcdef"
	srv, err := New(Config{
		WebhookAddr: "127.0.0.1:0", DashboardAddr: "127.0.0.1:0",
		DashboardControlSecret: controlSecret, DBPath: testDBPath(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	srv.startCtx = context.Background()
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	handler := srv.dash.Handler()
	login := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9090/api/login", strings.NewReader(`{"secret":"`+controlSecret+`"}`))
	login.Host = "127.0.0.1:9090"
	login.Header.Set("Origin", "http://127.0.0.1:9090")
	login.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, login)
	if loginRec.Code != http.StatusOK || len(loginRec.Result().Cookies()) != 1 {
		t.Fatalf("login status=%d cookies=%d", loginRec.Code, len(loginRec.Result().Cookies()))
	}
	cookie := loginRec.Result().Cookies()[0]
	eventReq := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9090/api/events", nil)
	eventReq.Host = "127.0.0.1:9090"
	eventReq.AddCookie(cookie)
	w := &flushSignalWriter{header: make(http.Header), flushed: make(chan struct{})}
	done := make(chan struct{})
	go func() { handler.ServeHTTP(w, eventReq); close(done) }()
	select {
	case <-w.flushed:
	case <-time.After(time.Second):
		t.Fatal("SSE did not start")
	}
	if err := srv.Restart(context.Background()); err != nil {
		t.Fatalf("direct Restart: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("direct Restart did not close SSE")
	}
	read := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9090/api/repos", nil)
	read.Host = "127.0.0.1:9090"
	read.AddCookie(cookie)
	readRec := httptest.NewRecorder()
	handler.ServeHTTP(readRec, read)
	if readRec.Code != http.StatusUnauthorized {
		t.Fatalf("old session after direct Restart status=%d", readRec.Code)
	}
}

func TestServer_doubleStart(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	err = srv.Start(ctx)
	if err == nil {
		t.Fatal("expected error on double start")
	}
}

func TestServer_defaultConcurrency(t *testing.T) {
	cfg := Config{WebhookAddr: "127.0.0.1:0"}
	if cfg.concurrency() != 2 {
		t.Errorf("expected default concurrency=2, got %d", cfg.concurrency())
	}

	cfg.Concurrency = 5
	if cfg.concurrency() != 5 {
		t.Errorf("expected concurrency=5, got %d", cfg.concurrency())
	}
}

func TestFilterReposByPath(t *testing.T) {
	repos := []RepoRecord{
		{ID: "aaa", Path: "workspace/repo-a"},
		{ID: "bbb", Path: "workspace/repo-b"},
		{ID: "ccc", Path: "workspace/repo-c"},
	}

	filtered := filterReposByPath(repos, "workspace/repo-a")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(filtered))
	}
	if filtered[0].ID != "aaa" {
		t.Errorf("expected repo aaa, got %s", filtered[0].ID)
	}
}

func TestFilterReposByPath_noMatch(t *testing.T) {
	repos := []RepoRecord{
		{ID: "aaa", Path: "workspace/repo-a"},
	}
	filtered := filterReposByPath(repos, "workspace/repo-b")
	if len(filtered) != 0 {
		t.Fatalf("expected 0 repos, got %d", len(filtered))
	}
}

func TestFilterReposByPath_empty(t *testing.T) {
	filtered := filterReposByPath(nil, "/any/path")
	if len(filtered) != 0 {
		t.Fatalf("expected 0 repos for nil input, got %d", len(filtered))
	}
}

func TestServer_registerCronSchedulesNoConfigPreservesPlanningSchedules(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	require.NoError(t, err)

	repo := writeIntegrationScheduleRepo(t, "")
	srv.registerCronSchedules([]RepoRecord{{ID: "repo-1", Path: repo}})

	require.Equal(t, []string{"repo-1:ceo", "repo-1:coo", "repo-1:cto-weekly", "repo-1:engineer"}, scheduleNames(srv.scheduler.Schedules()))
	status := srv.Status()
	require.Len(t, status.Repos, 0)
}

func TestServer_registerCronSchedulesBoardDrivenSuppressesPlanningSchedules(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	require.NoError(t, err)

	repo := writeIntegrationScheduleRepo(t, `version: 1
flow_profile: board-driven
`)
	_, err = srv.Repos().Register(context.Background(), repo, "owner/repo", "main")
	require.NoError(t, err)
	srv.registerCronSchedules([]RepoRecord{{ID: "repo-1", Path: repo}})

	require.Equal(t, []string{"repo-1:engineer"}, scheduleNames(srv.scheduler.Schedules()))
	status := srv.Status()
	require.Len(t, status.Repos, 1)
	require.Equal(t, "board-driven", status.Repos[0].FlowProfile)
}

func TestServer_registerCronSchedulesReplacesStaleSchedules(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	require.NoError(t, err)

	repo := writeIntegrationScheduleRepo(t, "")
	srv.registerCronSchedules([]RepoRecord{{ID: "repo-1", Path: repo}})
	require.Len(t, srv.scheduler.Schedules(), 4)

	mustWriteFile(t, filepath.Join(repo, ".harness", "integrations.yaml"), []byte("version: 1\nflow_profile: board-driven\n"))
	srv.registerCronSchedules([]RepoRecord{{ID: "repo-1", Path: repo}})
	require.Equal(t, []string{"repo-1:engineer"}, scheduleNames(srv.scheduler.Schedules()))
}

func TestServerJIRAWebhookNoConfigReturnsNotFoundAndNoQueueJobs(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	require.NoError(t, err)
	repo := writeIntegrationScheduleRepo(t, "")
	_, err = srv.Repos().Register(context.Background(), repo, "owner/repo", "main")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", strings.NewReader(`{"issue":{"key":"DEMO-1"}}`))
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	jobs, err := srv.queue.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, jobs)
}

func TestServerJIRAWebhookMirrorsScopedIssueWithoutQueueJob(t *testing.T) {
	t.Setenv("JIRA_WEBHOOK_SECRET", "serve-secret")
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	require.NoError(t, err)
	repo := writeIntegrationScheduleRepo(t, `version: 1
flow_profile: board-driven
ingestion:
  jira:
    enabled: true
    base_url: https://jira.example
    webhook_secret_env: JIRA_WEBHOOK_SECRET
    project_repo_map:
      - { project: DEMO, repo: owner/repo }
    scope:
      allowed_workspaces:
        - https://jira.example/jira/software/c/projects/DEMO/boards/board-example/backlog
      required_labels:
        - allowed-intake
`)
	_, err = srv.Repos().Register(context.Background(), repo, "owner/repo", "main")
	require.NoError(t, err)
	body := []byte(`{"issue":{"key":"DEMO-99","self":"https://jira.example/rest/api/3/issue/DEMO-99","fields":{"project":{"key":"DEMO"},"summary":"Scoped opportunity","description":"Mirror this scoped issue.","priority":{"name":"P1"},"status":{"name":"Ready for Dev"},"labels":["allowed-intake"]}}}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", strings.NewReader(string(body)))
	req.Header.Set("X-Atlassian-Webhook-Identifier", "serve-delivery-1")
	req.Header.Set("X-Mars-Harness-Jira-Signature", jiraintegration.SignWebhookPayloadForTest("serve-secret", body))
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	jobs, err := srv.queue.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, jobs)
	tickets, err := ticketstate.List(repo)
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	require.Equal(t, "backlog", tickets[0].Status)
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(tickets[0].RelPath)))
	require.NoError(t, err)
	require.Contains(t, string(content), `jira_key: "DEMO-99"`)
	require.Contains(t, string(content), "allowed-intake")
}

func writeIntegrationScheduleRepo(t *testing.T, integrationsYAML string) string {
	t.Helper()
	repo := t.TempDir()
	harnessDir := filepath.Join(repo, ".harness")
	require.NoError(t, os.MkdirAll(harnessDir, 0o755))
	manifest := `name: test
roles:
  ceo:
    prompt: roles/ceo.md
    model: fast
    schedule: "0 20 * * 0"
    tools: [file_read]
  coo:
    prompt: roles/coo.md
    model: fast
    schedule: "0 21 * * 0"
    tools: [file_read]
  cto-weekly:
    prompt: roles/cto.md
    model: fast
    schedule: "0 22 * * 0"
    tools: [file_read]
  engineer:
    prompt: roles/engineer.md
    model: fast
    schedule: "0 23 * * 0"
    tools: [file_read]
`
	mustWriteFile(t, filepath.Join(harnessDir, "manifest.yaml"), []byte(manifest))
	if strings.TrimSpace(integrationsYAML) != "" {
		mustWriteFile(t, filepath.Join(harnessDir, "integrations.yaml"), []byte(integrationsYAML))
	}
	return repo
}

func scheduleNames(schedules []scheduler.Schedule) []string {
	names := make([]string, 0, len(schedules))
	for _, sched := range schedules {
		names = append(names, sched.Name)
	}
	sort.Strings(names)
	return names
}

func TestRepoScope_isolatesStartup(t *testing.T) {
	dbPath := testDBPath(t)

	repoA := filepath.Join(t.TempDir(), "repo-a")
	repoB := filepath.Join(t.TempDir(), "repo-b")
	for _, p := range []string{repoA, repoB} {
		harnessDir := filepath.Join(p, ".harness")
		if err := os.MkdirAll(harnessDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		manifest := "name: test\nroles:\n  ceo:\n    prompt: roles/ceo.md\n    model: fast\n    tools: [file_read]\n"
		mustWriteFile(t, filepath.Join(harnessDir, "manifest.yaml"), []byte(manifest))
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	bgCtx := context.Background()

	_, err = srv.Repos().Register(bgCtx, repoA, "", "main")
	if err != nil {
		t.Fatalf("register repo-a: %v", err)
	}
	_, err = srv.Repos().Register(bgCtx, repoB, "", "main")
	if err != nil {
		t.Fatalf("register repo-b: %v", err)
	}

	repos, err := srv.Repos().List(bgCtx)
	if err != nil {
		t.Fatalf("list repos: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos in unscoped DB, got %d", len(repos))
	}

	scopedSrv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        dbPath,
		RepoScope:     repoA,
	})
	if err != nil {
		t.Fatalf("New scoped: %v", err)
	}

	allRepos, _ := scopedSrv.Repos().List(bgCtx)
	scoped := filterReposByPath(allRepos, scopedSrv.cfg.RepoScope)
	if len(scoped) != 1 {
		t.Fatalf("expected 1 repo in scoped view, got %d", len(scoped))
	}
	if scoped[0].Path != repoA {
		t.Errorf("expected repo-a path, got %s", scoped[0].Path)
	}
}

func TestHandleJobFailedEnqueuesSingleRecovery(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	job := &queue.Job{
		ID:     "job-1",
		RepoID: repoID,
		Role:   "engineer",
	}

	srv.handleJobFailed(ctx, job, errTest("executor: agent loop error (llm_unreachable): connection refused"))
	srv.handleJobFailed(ctx, job, errTest("executor: agent loop error (llm_unreachable): connection refused"))

	if got := countJobsByStatus(t, srv, "pending"); got != 1 {
		t.Fatalf("expected one active recovery job, got %d", got)
	}
}

func TestHandleJobFailedDoesNotRecoverDeterministicFailures(t *testing.T) {
	tests := []struct {
		name string
		err  errTest
	}{
		{name: "guardrail", err: "executor: dirty worktree containment: blast radius exceeded: 12 files changed (limit 10)"},
		{name: "workspace hygiene", err: "executor: workspace_hygiene_blocked before role \"engineer\" run: generated dependency output is dirty"},
		{name: "context overflow", err: "executor: agent loop error (llm_unreachable): llm: context size exceeded (non-retryable)"},
		{name: "model unavailable", err: "inference: local model for tier reasoning is missing"},
		{name: "max turns", err: "executor: agent ended with max_turns"},
		{name: "circle", err: "executor: agent ended with circle_detected"},
		{name: "unknown", err: "agent failed unexpectedly"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, repoID := newRecoveryTestServer(t)
			ctx := context.Background()
			job := &queue.Job{
				ID:     "job-1",
				RepoID: repoID,
				Role:   "engineer",
			}

			srv.handleJobFailed(ctx, job, tt.err)

			if got := countJobsByStatus(t, srv, "pending"); got != 0 {
				t.Fatalf("expected no deterministic recovery job, got %d", got)
			}
		})
	}
}

func TestHandleJobFailedDoesNotRecoverTicketGateFailure(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	job := &queue.Job{
		ID:     "job-1",
		RepoID: repoID,
		Role:   "engineer",
	}

	srv.handleJobFailed(ctx, job, errTest("executor: ticket gate: engineer ended without completing any existing in-progress ticket"))

	if got := countJobsByStatus(t, srv, "pending"); got != 0 {
		t.Fatalf("expected no ticket-gate recovery job, got %d", got)
	}
}

func TestHandleJobFailedDoesNotRecoverRecoveryJob(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	job := &queue.Job{
		ID:      "job-1",
		RepoID:  repoID,
		Role:    "engineer",
		Trigger: `{"type":"auto_recover","source_job":"previous","reason":"ticket gate failed"}`,
	}

	srv.handleJobFailed(ctx, job, errTest("ticket gate failed again"))

	if got := countJobsByStatus(t, srv, "pending"); got != 0 {
		t.Fatalf("expected no recursive recovery job, got %d", got)
	}
}

func TestIsAutoRecoverTrigger(t *testing.T) {
	if !isAutoRecoverTrigger(`{"type":"auto_recover"}`) {
		t.Fatal("expected auto_recover trigger to be detected")
	}
	if isAutoRecoverTrigger(`{"type":"chain"}`) {
		t.Fatal("expected non-recovery trigger to be false")
	}
	if isAutoRecoverTrigger(`not-json`) {
		t.Fatal("expected malformed trigger to be false")
	}
}

func TestSelfHealRecoveryQueueCancelsDuplicates(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, err := srv.queue.Enqueue(ctx, queue.Job{
			RepoID:         repoID,
			Role:           "engineer",
			Trigger:        fmt.Sprintf(`{"type":"auto_recover","source_job":"job-%d"}`, i),
			IdempotencyKey: fmt.Sprintf("recover:%s:engineer:%d", repoID, i),
		})
		if err != nil {
			t.Fatalf("enqueue recovery job: %v", err)
		}
	}

	srv.selfHealRecoveryQueue(ctx, "test")

	if got := countJobsByStatus(t, srv, "pending"); got != 1 {
		t.Fatalf("expected one recovery job left pending, got %d", got)
	}
	if got := countJobsByStatus(t, srv, "cancelled"); got != 1 {
		t.Fatalf("expected one duplicate recovery job cancelled, got %d", got)
	}
}

func TestBuildTicketIndex_empty(t *testing.T) {
	dir := t.TempDir()
	idx := BuildTicketIndex(dir)
	if idx != "No existing tickets found in docs/tickets/." {
		t.Errorf("unexpected index for empty dir: %s", idx)
	}
}

type errTest string

func (e errTest) Error() string {
	return string(e)
}

func newRecoveryTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".harness", "roles"), 0o755); err != nil {
		t.Fatalf("mkdir harness: %v", err)
	}
	manifest := "name: test\nroles:\n  engineer:\n    prompt: roles/engineer.md\n    model: fast\n    then: [engineer]\n    tools: [file_read]\n"
	mustWriteFile(t, filepath.Join(repo, ".harness", "manifest.yaml"), []byte(manifest))
	mustWriteFile(t, filepath.Join(repo, ".harness", "roles", "engineer.md"), []byte("# Engineer\n"))

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })

	repoID, err := srv.Repos().Register(context.Background(), repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}
	return srv, repoID
}

func countJobsByStatus(t *testing.T, srv *Server, status string) int {
	t.Helper()
	var count int
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE status = ?`, status).Scan(&count); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	return count
}

func countJobsByStatusAndRole(t *testing.T, srv *Server, status, role string) int {
	t.Helper()
	var count int
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE status = ? AND role = ?`, status, role).Scan(&count); err != nil {
		t.Fatalf("count jobs by status and role: %v", err)
	}
	return count
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestBuildTicketIndex_findsTickets(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		mustMkdirAll(t, filepath.Join(dir, sub))
	}
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/done/T-001-alpha.md"), []byte("# Alpha\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/backlog/T-002-beta.md"), []byte("# Beta\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/in-progress/T-003-gamma.md"), []byte("# Gamma\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/done/README.md"), []byte("# Tickets\n"))

	idx := BuildTicketIndex(dir)
	if !strings.Contains(idx, "3 total") {
		t.Errorf("expected 3 total, got: %s", idx)
	}
	if !strings.Contains(idx, "Eligible product in-progress tickets are the Engineer front of queue") {
		t.Errorf("expected in-progress priority guidance, got: %s", idx)
	}
	if !strings.Contains(idx, "[backlog] T-002-beta.md") {
		t.Errorf("expected backlog ticket, got: %s", idx)
	}
	if !strings.Contains(idx, "path: docs/tickets/backlog/T-002-beta.md") {
		t.Errorf("expected backlog ticket path, got: %s", idx)
	}
	if !strings.Contains(idx, "[in-progress] T-003-gamma.md") {
		t.Errorf("expected in-progress ticket, got: %s", idx)
	}
	if !strings.Contains(idx, "path: docs/tickets/in-progress/T-003-gamma.md") {
		t.Errorf("expected in-progress ticket path, got: %s", idx)
	}
	if strings.Index(idx, "[in-progress] T-003-gamma.md") > strings.Index(idx, "[backlog] T-002-beta.md") {
		t.Errorf("expected in-progress tickets before backlog, got: %s", idx)
	}
	if !strings.Contains(idx, "[done] T-001-alpha.md") {
		t.Errorf("expected done ticket, got: %s", idx)
	}
	if !strings.Contains(idx, "path: docs/tickets/done/T-001-alpha.md") {
		t.Errorf("expected done ticket path, got: %s", idx)
	}
	if strings.Contains(idx, "README.md") {
		t.Errorf("README.md should be excluded")
	}
}

func TestBuildTicketIndex_interventionDebtDoesNotPreemptOrdinaryBacklog(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		mustMkdirAll(t, filepath.Join(dir, sub))
	}
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/backlog/MH-010-ordinary.md"), []byte("# Ordinary\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/backlog/MH-011-medium-intervention.md"), []byte("---\nkind: intervention-debt\npriority: medium\n---\n# Medium Intervention\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/backlog/MH-012-high-intervention.md"), []byte("---\nkind: intervention-debt\npriority: high\n---\n# High Intervention\n"))

	idx := BuildTicketIndex(dir)
	if !strings.Contains(idx, "intervention-debt tickets stay visible") {
		t.Errorf("expected intervention-debt priority guidance, got: %s", idx)
	}
	if !strings.Contains(idx, "2 hidden") {
		t.Errorf("expected hidden deferred intervention-debt count, got: %s", idx)
	}
	highInterventionPos := strings.Index(idx, "[backlog][intervention-debt] MH-012-high-intervention.md")
	ordinaryPos := strings.Index(idx, "[backlog] MH-010-ordinary.md")
	mediumInterventionPos := strings.Index(idx, "[backlog][intervention-debt] MH-011-medium-intervention.md")
	if ordinaryPos < 0 {
		t.Fatalf("expected ordinary backlog entry, got: %s", idx)
	}
	if highInterventionPos >= 0 || mediumInterventionPos >= 0 {
		t.Fatalf("expected intervention-debt backlog to be deferred behind product work, got: %s", idx)
	}
}

func TestFirstBacklogInterventionDebtDoesNotPreemptProductBacklog(t *testing.T) {
	tickets := []ticketstate.Ticket{
		{ID: "T-001", Status: ticketstate.StatusBacklog, Kind: "intervention-debt", Priority: "medium"},
		{ID: "T-002", Status: ticketstate.StatusBacklog, Kind: "standard", Priority: "high"},
		{ID: "T-003", Status: ticketstate.StatusBacklog, Kind: "intervention-debt", Priority: "high"},
	}

	_, ok := firstBacklogInterventionDebt(tickets)
	if ok {
		t.Fatal("expected intervention-debt backlog to stay non-preemptive")
	}
}

func TestBuildTicketIndex_movesBlockedInProgressBehindEligibleBacklog(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		mustMkdirAll(t, filepath.Join(dir, sub))
	}
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/in-progress/T-001-blocked.md"), []byte("---\nid: T-001\nblocker: \"waiting\"\nblocked_by: [\"T-003\"]\nnext_action: \"complete T-003\"\n---\n# Blocked\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/in-progress/T-002-ready.md"), []byte("---\nid: T-002\nblocker: none\nblocked_by: []\n---\n# Ready\n"))
	mustWriteFile(t, filepath.Join(dir, "docs/tickets/backlog/T-003-dependency.md"), []byte("---\nid: T-003\n---\n# Dependency\n"))

	idx := BuildTicketIndex(dir)
	readyPos := strings.Index(idx, "[in-progress] T-002-ready.md")
	backlogPos := strings.Index(idx, "[backlog] T-003-dependency.md")
	blockedPos := strings.Index(idx, "[in-progress][blocked] T-001-blocked.md")
	if readyPos < 0 || backlogPos < 0 || blockedPos < 0 {
		t.Fatalf("expected ready, backlog, and blocked lines, got: %s", idx)
	}
	if readyPos > backlogPos || backlogPos > blockedPos {
		t.Fatalf("expected eligible in-progress, backlog, blocked in-progress ordering, got: %s", idx)
	}
	if !strings.Contains(idx, "next: complete T-003") {
		t.Fatalf("expected next action in blocked line, got: %s", idx)
	}
}

func TestScanRepoEnqueuesJanitorForStaleInProgressTicket(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}
	writeTicketGateContent(t, repo, "in-progress", "T-001-stale.md", `---
id: T-001
title: Stale
last_attempt: "2026-04-01"
blocker: none
blocked_by: []
---

# T-001
`)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	repoID, err := srv.Repos().Register(ctx, repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}

	if err := srv.ScanRepo(ctx, repoID); err != nil {
		t.Fatalf("ScanRepo: %v", err)
	}

	var count int
	if err := srv.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE role = 'janitor' AND idempotency_key = ?`, "ticket:stale-in-progress:"+repoID).Scan(&count); err != nil {
		t.Fatalf("count janitor jobs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one janitor stale-ticket job, got %d", count)
	}
}

func TestOrchestratorSurveyRoutesStaleTicketAndOwnership(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}
	writeTicketGateContent(t, repo, "in-progress", "T-001-stale.md", `---
id: T-001
title: Stale
last_attempt: "2026-04-01"
blocker: none
blocked_by: []
---

# T-001
`)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	repoID, err := srv.Repos().Register(ctx, repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}

	report, err := srv.surveyOrchestrator(ctx, "test")
	if err != nil {
		t.Fatalf("surveyOrchestrator: %v", err)
	}
	if report.JobsRouted < 2 {
		t.Fatalf("expected ticket owner and stale janitor jobs, got %+v", report)
	}

	assertSurveyJob(t, srv, repoID, "engineer", "ticket_delivery", "eligible_in_progress_ticket")
	assertSurveyJob(t, srv, repoID, "janitor", "ticket_hygiene", "stale_in_progress_ticket")
}

func TestOrchestratorSurveyPausesTicketOwnerAfterRecentRuntimeFailure(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}
	writeTicketGateContent(t, repo, "in-progress", "T-001-active.md", `---
id: T-001
title: Active
last_attempt: "`+time.Now().UTC().Format("2006-01-02")+`"
blocker: none
blocked_by: []
---

# T-001
`)

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	repoID, err := srv.Repos().Register(ctx, repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}

	jobID, err := srv.queue.Enqueue(ctx, queue.Job{RepoID: repoID, Role: "engineer"})
	if err != nil {
		t.Fatalf("enqueue failed engineer: %v", err)
	}
	claimed, err := srv.queue.Claim(ctx, "test-worker")
	if err != nil {
		t.Fatalf("claim failed engineer: %v", err)
	}
	if claimed == nil || claimed.ID != jobID {
		t.Fatalf("expected to claim failed engineer seed job, got %+v", claimed)
	}
	if err := srv.queue.MarkRunning(ctx, claimed.ID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := srv.queue.Fail(ctx, claimed.ID, "executor: agent ended with max_turns"); err != nil {
		t.Fatalf("fail engineer: %v", err)
	}

	report, err := srv.surveyOrchestrator(ctx, "test")
	if err != nil {
		t.Fatalf("surveyOrchestrator: %v", err)
	}
	if report.JobsRouted != 0 {
		t.Fatalf("expected recent runtime failure to pause ticket-owner routing, got %+v", report)
	}
	if got := countJobsByStatusAndRole(t, srv, "pending", "engineer"); got != 0 {
		t.Fatalf("expected no immediate engineer retry after max_turns, got %d", got)
	}
}

func TestOrchestratorSurveyPausesDirtyTargetAfterDogfoodFailure(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if _, err := runGitCommand(ctx, repo, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}
	if _, err := runGitCommand(ctx, repo, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := runGitCommand(ctx, repo, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "test: baseline"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	mustWriteFile(t, filepath.Join(repo, "docs/tickets/backlog/T-002-dogfood-finding.md"), []byte(`---
id: T-002
title: Dogfood finding
---

# T-002
`))

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	repoID, err := srv.Repos().Register(ctx, repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}
	if err := srv.scoreStore.RecordOutcome(ctx, scoring.Outcome{JobID: "dogfood-1", RepoID: repoID, Role: "dogfood", Type: scoring.OutcomeFailed}); err != nil {
		t.Fatalf("record dogfood outcome: %v", err)
	}

	report, err := srv.surveyOrchestrator(ctx, "test")
	if err != nil {
		t.Fatalf("surveyOrchestrator: %v", err)
	}
	if report.JobsRouted != 0 {
		t.Fatalf("expected dirty target to pause dogfood recovery routing, got %+v", report)
	}
	if got := countJobsByStatusAndRole(t, srv, "pending", "engineer"); got != 0 {
		t.Fatalf("expected no engineer survey job while dogfood ticket is uncommitted, got %d", got)
	}
}

func TestOrchestratorSurveyRoutesFailedChecksAndNoops(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	repoID, err := srv.Repos().Register(ctx, repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}

	if err := srv.scoreStore.RecordOutcome(ctx, scoring.Outcome{JobID: "check-1", RepoID: repoID, Role: "engineer", Type: scoring.OutcomeChecksFailed}); err != nil {
		t.Fatalf("record failed check: %v", err)
	}
	if err := srv.scoreStore.RecordOutcome(ctx, scoring.Outcome{JobID: "noop-1", RepoID: repoID, Role: "engineer", Type: scoring.OutcomeNoop}); err != nil {
		t.Fatalf("record noop: %v", err)
	}

	report, err := srv.surveyOrchestrator(ctx, "test")
	if err != nil {
		t.Fatalf("surveyOrchestrator: %v", err)
	}
	if report.JobsRouted < 2 {
		t.Fatalf("expected failed-check and no-op routing, got %+v", report)
	}

	assertSurveyJob(t, srv, repoID, "pipeline-fixer", "pipeline_repair", "failed_check")
	assertSurveyJob(t, srv, repoID, "janitor", "ticket_hygiene", "silent_noop")
}

func TestOrchestratorSurveyTriagesTelemetryAndLowScores(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := scanner.Init(repo, false); err != nil {
		t.Fatalf("init target harness: %v", err)
	}

	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	repoID, err := srv.Repos().Register(ctx, repo, "", "main")
	if err != nil {
		t.Fatalf("register repo: %v", err)
	}

	for i := 0; i < 3; i++ {
		srv.telemetry.Record(fmt.Sprintf("job-%d", i), repoID, "engineer", "context overflow while assembling prompt")
	}
	for i := 0; i < 5; i++ {
		if err := srv.scoreStore.RecordOutcome(ctx, scoring.Outcome{JobID: fmt.Sprintf("score-%d", i), RepoID: repoID, Role: "qa", Type: scoring.OutcomeFailed}); err != nil {
			t.Fatalf("record outcome: %v", err)
		}
	}
	if _, err := srv.scoreStore.ComputeScore(ctx, "qa", repoID, 30); err != nil {
		t.Fatalf("compute score: %v", err)
	}

	report, err := srv.surveyOrchestrator(ctx, "test")
	if err != nil {
		t.Fatalf("surveyOrchestrator: %v", err)
	}
	if report.TicketsTriaged < 2 {
		t.Fatalf("expected telemetry and score triage signals, got %+v", report)
	}

	entries, err := os.ReadDir(filepath.Join(repo, "docs", "tickets", "backlog"))
	if err != nil {
		t.Fatalf("read backlog: %v", err)
	}
	var interventionDebt int
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(repo, "docs", "tickets", "backlog", entry.Name()))
		if err != nil {
			t.Fatalf("read ticket: %v", err)
		}
		if strings.Contains(string(data), "kind: intervention-debt") {
			interventionDebt++
		}
	}
	if interventionDebt != 0 {
		t.Fatalf("expected telemetry and score survey to stay out of target backlog by default, got %d intervention-debt ticket(s)", interventionDebt)
	}
}

func assertSurveyJob(t *testing.T, srv *Server, repoID, role, payloadMode, signal string) {
	t.Helper()
	var trigger, gotPayloadMode, group string
	var cap int
	err := srv.db.QueryRow(`
SELECT trigger_payload, payload_mode, concurrency_group, daily_cap
FROM jobs
WHERE repo_id = ? AND role = ? AND payload_mode = ?
ORDER BY created_at DESC
LIMIT 1`, repoID, role, payloadMode).Scan(&trigger, &gotPayloadMode, &group, &cap)
	if err != nil {
		t.Fatalf("query survey job %s/%s/%s: %v", role, payloadMode, signal, err)
	}
	if gotPayloadMode != payloadMode {
		t.Fatalf("payload mode = %q, want %q", gotPayloadMode, payloadMode)
	}
	if group == "" {
		t.Fatalf("expected concurrency group for %s/%s", role, payloadMode)
	}
	if cap <= 0 {
		t.Fatalf("expected positive daily cap for %s/%s", role, payloadMode)
	}
	if !strings.Contains(trigger, signal) {
		t.Fatalf("expected trigger %q to contain signal %q", trigger, signal)
	}
}
