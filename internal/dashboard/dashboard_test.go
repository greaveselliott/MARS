/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/features/F-010-dashboard-control-plane.md
*/
package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestDashboard(t *testing.T) *Dashboard {
	t.Helper()
	d, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

func TestDashboard_allPagesReturn200(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	pages := []struct {
		path       string
		wantStatus int
	}{
		{"/pipeline", http.StatusOK},
		{"/roles", http.StatusOK},
		{"/throughput", http.StatusOK},
		{"/debug", http.StatusOK},
		{"/evolution", http.StatusOK},
	}

	for _, tc := range pages {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("GET %s: got status %d, want %d", tc.path, resp.StatusCode, tc.wantStatus)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "text/html") {
				t.Errorf("GET %s: got Content-Type %q, want text/html", tc.path, ct)
			}
		})
	}
}

func TestDashboard_indexRedirects(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp, err := client.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("GET /: got status %d, want %d", resp.StatusCode, http.StatusFound)
	}
	loc := resp.Header.Get("Location")
	if loc != "/pipeline" {
		t.Errorf("GET /: redirect to %q, want /pipeline", loc)
	}
}

func TestDashboard_notFound(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /nonexistent: got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestDashboard_sseConnection(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/events: got status %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("GET /api/events: Content-Type = %q, want text/event-stream", ct)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Read the initial keepalive comment.
	if !scanner.Scan() {
		t.Fatal("expected keepalive line")
	}
	if got := scanner.Text(); !strings.HasPrefix(got, ":") {
		t.Errorf("expected keepalive comment, got %q", got)
	}

	// Broadcast an event and verify it arrives.
	var received string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				received = strings.TrimPrefix(line, "data: ")
				return
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)
	d.BroadcastEvent("test_event", "hello")

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("timed out waiting for SSE event")
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(received), &parsed); err != nil {
		t.Fatalf("unmarshal SSE data: %v (raw: %q)", err, received)
	}
	if parsed["type"] != "test_event" {
		t.Errorf("event type = %q, want test_event", parsed["type"])
	}
	if parsed["data"] != "hello" {
		t.Errorf("event data = %q, want hello", parsed["data"])
	}
}

func TestDashboard_emergencyStop(t *testing.T) {
	var called bool
	d, err := New(Config{
		EmergencyStop: func() []error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/emergency-stop", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/emergency-stop: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	var result emergencyStopResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.OK {
		t.Error("expected ok=true")
	}
	if !called {
		t.Error("EmergencyStop callback was not invoked")
	}
}

func TestDashboard_emergencyStopWithErrors(t *testing.T) {
	d, err := New(Config{
		EmergencyStop: func() []error {
			return []error{fmt.Errorf("agent stuck"), fmt.Errorf("queue locked")}
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/emergency-stop", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	var result emergencyStopResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.OK {
		t.Error("expected ok=false when errors returned")
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestDashboard_emergencyStopMethodNotAllowed(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/emergency-stop")
	if err != nil {
		t.Fatalf("GET /api/emergency-stop: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want 405", resp.StatusCode)
	}
}

func TestDashboard_staticAssets(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	assets := []struct {
		path        string
		contentType string
	}{
		{"/static/style.css", "text/css"},
		{"/static/app.js", "text/javascript"},
	}

	for _, tc := range assets {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("GET %s: got status %d, want 200", tc.path, resp.StatusCode)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, tc.contentType) {
				t.Errorf("GET %s: got Content-Type %q, want %s", tc.path, ct, tc.contentType)
			}
		})
	}
}

func TestDashboard_themeAvoidsLegacyBluePalette(t *testing.T) {
	files := []string{
		"static/style.css",
		"templates/evolution.html",
		"templates/throughput.html",
	}
	legacyTokens := []string{
		"#0f172a",
		"#1e293b",
		"#334155",
		"#e2e8f0",
		"#94a3b8",
		"#3b82f6",
		"#2563eb",
		"15, 23, 42",
		"51, 65, 85",
		"59, 130, 246",
	}

	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			data, err := content.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			body := string(data)
			for _, token := range legacyTokens {
				if strings.Contains(body, token) {
					t.Fatalf("%s still contains legacy dashboard palette token %q", path, token)
				}
			}
		})
	}

	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/static/style.css")
	if err != nil {
		t.Fatalf("GET /static/style.css: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /static/style.css: %v", err)
	}
	css := string(body)
	for _, token := range []string{"--primary: #f97316;", "--accent: #14b8a6;", "--surface-raised: #23241f;"} {
		if !strings.Contains(css, token) {
			t.Fatalf("style.css missing current dashboard theme token %q", token)
		}
	}
}

func newTestDashboardWithControls(t *testing.T) (*Dashboard, *controlState) {
	t.Helper()
	state := &controlState{}
	d, err := New(Config{
		Controls: ControlCallbacks{
			Pause:   func() { state.paused = true },
			Resume:  func() { state.paused = false },
			Restart: func(_ context.Context) error { state.restarted = true; return nil },
			Stop:    func(_ context.Context) error { state.stopped = true; return nil },
			Scan: func(_ context.Context, repoID string) error {
				state.scannedRepo = repoID
				return nil
			},
			RunRole: func(_ context.Context, repoID, role string) error {
				state.runRepo = repoID
				state.runRole = role
				return nil
			},
			Status: func() interface{} {
				return map[string]interface{}{
					"healthy":     true,
					"paused":      state.paused,
					"active_jobs": 0,
					"uptime_secs": 42.0,
				}
			},
			IsPaused: func() bool { return state.paused },
			ListRepos: func() []RepoInfoDTO {
				return []RepoInfoDTO{{ID: "r1", Path: "/tmp/repo"}}
			},
			ListRoles: func(_ string) []string {
				return []string{"ceo", "engineer"}
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d, state
}

type controlState struct {
	paused      bool
	restarted   bool
	stopped     bool
	scannedRepo string
	runRepo     string
	runRole     string
}

func TestDashboard_pauseEndpoint(t *testing.T) {
	d, state := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/pause", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/pause: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if !state.paused {
		t.Error("Pause callback was not invoked")
	}
}

func TestDashboard_resumeEndpoint(t *testing.T) {
	d, state := newTestDashboardWithControls(t)
	state.paused = true
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/resume", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/resume: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if state.paused {
		t.Error("Resume callback did not clear paused state")
	}
}

func TestDashboard_restartEndpoint(t *testing.T) {
	d, state := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/restart", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/restart: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if !state.restarted {
		t.Error("Restart callback was not invoked")
	}
}

func TestDashboard_stopEndpoint(t *testing.T) {
	d, state := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/stop", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/stop: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if !state.stopped {
		t.Error("Stop callback was not invoked")
	}
}

func TestDashboard_scanEndpoint(t *testing.T) {
	d, state := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/scan", "application/json",
		strings.NewReader(`{"repo_id":"r1"}`))
	if err != nil {
		t.Fatalf("POST /api/scan: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if state.scannedRepo != "r1" {
		t.Errorf("expected scanned repo r1, got %q", state.scannedRepo)
	}
}

func TestDashboard_scanEndpoint_missingRepoID(t *testing.T) {
	d, _ := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/scan", "application/json",
		strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /api/scan: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", resp.StatusCode)
	}
}

func TestDashboard_runRoleEndpoint(t *testing.T) {
	d, state := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/run-role", "application/json",
		strings.NewReader(`{"repo_id":"r1","role":"engineer"}`))
	if err != nil {
		t.Fatalf("POST /api/run-role: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if state.runRepo != "r1" || state.runRole != "engineer" {
		t.Errorf("expected r1/engineer, got %s/%s", state.runRepo, state.runRole)
	}
}

func TestDashboard_statusEndpoint(t *testing.T) {
	d, _ := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["healthy"] != true {
		t.Errorf("expected healthy=true, got %v", result["healthy"])
	}
}

func TestDashboard_reposEndpoint(t *testing.T) {
	d, _ := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/repos")
	if err != nil {
		t.Fatalf("GET /api/repos: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	var repos []RepoInfoDTO
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(repos) != 1 || repos[0].ID != "r1" {
		t.Errorf("unexpected repos: %+v", repos)
	}
}

func TestDashboard_controlEndpoints_methodNotAllowed(t *testing.T) {
	d, _ := newTestDashboardWithControls(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	postOnlyEndpoints := []string{"/api/pause", "/api/resume", "/api/stop", "/api/restart", "/api/scan", "/api/run-role"}
	for _, ep := range postOnlyEndpoints {
		t.Run(ep, func(t *testing.T) {
			resp, err := http.Get(srv.URL + ep)
			if err != nil {
				t.Fatalf("GET %s: %v", ep, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("GET %s: got status %d, want 405", ep, resp.StatusCode)
			}
		})
	}
}

func TestDashboard_controlEndpoints_nilCallbacks(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	endpoints := []string{"/api/pause", "/api/resume", "/api/stop", "/api/restart"}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			resp, err := http.Post(srv.URL+ep, "application/json", nil)
			if err != nil {
				t.Fatalf("POST %s: %v", ep, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotImplemented {
				t.Errorf("POST %s: got status %d, want 501", ep, resp.StatusCode)
			}
		})
	}
}

func TestDashboard_missingModuleEmptyState(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	pages := []string{"/pipeline", "/roles", "/throughput", "/debug", "/evolution", "/orchestration"}

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			resp, err := http.Get(srv.URL + page)
			if err != nil {
				t.Fatalf("GET %s: %v", page, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("GET %s: status %d, want 200 (empty-state render)", page, resp.StatusCode)
			}
		})
	}
}
