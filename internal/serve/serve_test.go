package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func testDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func TestServer_healthHandler_healthy(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   ":0",
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

func TestServer_healthHandler_unhealthy(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   ":0",
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
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	if !srv.Healthy() {
		t.Error("expected server to be healthy after start")
	}

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

func TestServer_doubleStart(t *testing.T) {
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
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
	cfg := Config{WebhookAddr: ":0"}
	if cfg.concurrency() != 2 {
		t.Errorf("expected default concurrency=2, got %d", cfg.concurrency())
	}

	cfg.Concurrency = 5
	if cfg.concurrency() != 5 {
		t.Errorf("expected concurrency=5, got %d", cfg.concurrency())
	}
}
