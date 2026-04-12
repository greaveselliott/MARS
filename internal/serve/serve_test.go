package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestFilterReposByPath(t *testing.T) {
	repos := []RepoRecord{
		{ID: "aaa", Path: "<home-path>"},
		{ID: "bbb", Path: "<home-path>"},
		{ID: "ccc", Path: "<home-path>"},
	}

	filtered := filterReposByPath(repos, "<home-path>")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(filtered))
	}
	if filtered[0].ID != "aaa" {
		t.Errorf("expected repo aaa, got %s", filtered[0].ID)
	}
}

func TestFilterReposByPath_noMatch(t *testing.T) {
	repos := []RepoRecord{
		{ID: "aaa", Path: "<home-path>"},
	}
	filtered := filterReposByPath(repos, "<home-path>")
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
		if err := os.WriteFile(filepath.Join(harnessDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
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

func TestBuildTicketIndex_empty(t *testing.T) {
	dir := t.TempDir()
	idx := BuildTicketIndex(dir)
	if idx != "No existing tickets found in docs/tickets/." {
		t.Errorf("unexpected index for empty dir: %s", idx)
	}
}

func TestBuildTicketIndex_findsTickets(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	os.WriteFile(filepath.Join(dir, "docs/tickets/done/T-001-alpha.md"), []byte("# Alpha\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/backlog/T-002-beta.md"), []byte("# Beta\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/done/README.md"), []byte("# Tickets\n"), 0o644)

	idx := BuildTicketIndex(dir)
	if !strings.Contains(idx, "2 total") {
		t.Errorf("expected 2 total, got: %s", idx)
	}
	if !strings.Contains(idx, "[backlog] T-002-beta.md") {
		t.Errorf("expected backlog ticket, got: %s", idx)
	}
	if !strings.Contains(idx, "[done] T-001-alpha.md") {
		t.Errorf("expected done ticket, got: %s", idx)
	}
	if strings.Contains(idx, "README.md") {
		t.Errorf("README.md should be excluded")
	}
}
