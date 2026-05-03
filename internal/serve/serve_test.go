package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/scanner"
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

func TestServer_qualityScoreAPIServesRepoArtifact(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "QUALITY_SCORE.md"), []byte("# Quality Score\n\nGenerated.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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

func TestHandleJobFailedEnqueuesSingleRecovery(t *testing.T) {
	srv, repoID := newRecoveryTestServer(t)
	ctx := context.Background()

	job := &queue.Job{
		ID:     "job-1",
		RepoID: repoID,
		Role:   "engineer",
	}

	srv.handleJobFailed(ctx, job, errTest("agent failed unexpectedly"))
	srv.handleJobFailed(ctx, job, errTest("agent failed unexpectedly"))

	if got := countJobsByStatus(t, srv, "pending"); got != 1 {
		t.Fatalf("expected one active recovery job, got %d", got)
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
	if err := os.WriteFile(filepath.Join(repo, ".harness", "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".harness", "roles", "engineer.md"), []byte("# Engineer\n"), 0o644); err != nil {
		t.Fatalf("write role prompt: %v", err)
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

func TestBuildTicketIndex_findsTickets(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	os.WriteFile(filepath.Join(dir, "docs/tickets/done/T-001-alpha.md"), []byte("# Alpha\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/backlog/T-002-beta.md"), []byte("# Beta\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/in-progress/T-003-gamma.md"), []byte("# Gamma\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/done/README.md"), []byte("# Tickets\n"), 0o644)

	idx := BuildTicketIndex(dir)
	if !strings.Contains(idx, "3 total") {
		t.Errorf("expected 3 total, got: %s", idx)
	}
	if !strings.Contains(idx, "Eligible in-progress tickets are the Engineer front of queue") {
		t.Errorf("expected in-progress priority guidance, got: %s", idx)
	}
	if !strings.Contains(idx, "[backlog] T-002-beta.md") {
		t.Errorf("expected backlog ticket, got: %s", idx)
	}
	if !strings.Contains(idx, "[in-progress] T-003-gamma.md") {
		t.Errorf("expected in-progress ticket, got: %s", idx)
	}
	if strings.Index(idx, "[in-progress] T-003-gamma.md") > strings.Index(idx, "[backlog] T-002-beta.md") {
		t.Errorf("expected in-progress tickets before backlog, got: %s", idx)
	}
	if !strings.Contains(idx, "[done] T-001-alpha.md") {
		t.Errorf("expected done ticket, got: %s", idx)
	}
	if strings.Contains(idx, "README.md") {
		t.Errorf("README.md should be excluded")
	}
}

func TestBuildTicketIndex_prioritizesInterventionDebtAheadOfOrdinaryBacklog(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	os.WriteFile(filepath.Join(dir, "docs/tickets/backlog/MH-010-ordinary.md"), []byte("# Ordinary\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/backlog/MH-011-intervention.md"), []byte("---\nkind: intervention-debt\n---\n# Intervention\n"), 0o644)

	idx := BuildTicketIndex(dir)
	if !strings.Contains(idx, "intervention-debt is prioritised") {
		t.Errorf("expected intervention-debt guidance, got: %s", idx)
	}
	interventionPos := strings.Index(idx, "[backlog][intervention-debt] MH-011-intervention.md")
	ordinaryPos := strings.Index(idx, "[backlog] MH-010-ordinary.md")
	if interventionPos < 0 || ordinaryPos < 0 {
		t.Fatalf("expected both backlog entries, got: %s", idx)
	}
	if interventionPos > ordinaryPos {
		t.Fatalf("expected intervention-debt backlog before ordinary backlog, got: %s", idx)
	}
}

func TestBuildTicketIndex_movesBlockedInProgressBehindEligibleBacklog(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	os.WriteFile(filepath.Join(dir, "docs/tickets/in-progress/T-001-blocked.md"), []byte("---\nid: T-001\nblocker: \"waiting\"\nblocked_by: [\"T-003\"]\nnext_action: \"complete T-003\"\n---\n# Blocked\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/in-progress/T-002-ready.md"), []byte("---\nid: T-002\nblocker: none\nblocked_by: []\n---\n# Ready\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs/tickets/backlog/T-003-dependency.md"), []byte("---\nid: T-003\n---\n# Dependency\n"), 0o644)

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
