package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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
		{"/static/app.css", "text/css"},
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
		})
	}
}

func TestDashboard_missingModuleEmptyState(t *testing.T) {
	d := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()

	pages := []string{"/pipeline", "/roles", "/throughput", "/debug", "/evolution"}

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
