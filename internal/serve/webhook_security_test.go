/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/design-docs/github-app-integration.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/design-docs/pipeline-engine.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
*/
package serve

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	gh "github.com/greaveselliott/mars/internal/github"
	"github.com/stretchr/testify/require"
)

const integrationWebhookSecret = "0123456789abcdef0123456789abcdef"

func signedGitHubRequest(eventType, delivery string, body []byte) *http.Request {
	mac := hmac.New(sha256.New, []byte(integrationWebhookSecret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", eventType)
	req.Header.Set("X-GitHub-Delivery", delivery)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return req
}

func webhookIntegrationRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", "manifest.yaml"), []byte("name: webhook-test\nroles:\n  engineer:\n    prompt: roles/engineer.md\n    triggers: [push]\n"), 0o644))
	return repo
}

func newWebhookIntegrationServer(t *testing.T, dbPath string) *Server {
	t.Helper()
	srv, err := New(Config{
		WebhookAddr: "127.0.0.1:0", DashboardAddr: "127.0.0.1:0", DBPath: dbPath,
		ModelEndpoint: "http://127.0.0.1:9999/v1", WebhookSecret: integrationWebhookSecret,
		WebhookAllowedActorIDs: []int64{42},
	})
	require.NoError(t, err)
	return srv
}

func TestServerWebhookUsesRealQueueAndDurableReplay(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	repoPath := webhookIntegrationRepo(t)
	srv := newWebhookIntegrationServer(t, dbPath)
	_, err := srv.Repos().Register(context.Background(), repoPath, "Owner/Repo", "main")
	require.NoError(t, err)
	repos, err := srv.Repos().List(context.Background())
	require.NoError(t, err)
	require.NoError(t, srv.triggers.Rebuild(repos))

	rejected := []byte(`{"ref":"refs/heads/main","sender":{"id":7},"repository":{"full_name":"owner/repo"}}`)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, signedGitHubRequest("push", "delivery-1", rejected))
	require.Equal(t, http.StatusAccepted, rec.Code)
	jobs, err := srv.queue.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, jobs)

	authorized := []byte(`{"ref":"refs/heads/main","sender":{"id":42,"login":"trusted"},"repository":{"full_name":"owner/repo"}}`)
	rec = httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, signedGitHubRequest("push", "delivery-1", authorized))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	jobs, err = srv.queue.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.NotContains(t, jobs[0].Trigger, "trusted", "full payload and actor login must not be persisted")
	require.Contains(t, jobs[0].Trigger, `"body_sha"`)

	job, err := srv.queue.Claim(context.Background(), "worker")
	require.NoError(t, err)
	require.NoError(t, srv.queue.MarkRunning(context.Background(), job.ID))
	require.NoError(t, srv.queue.Complete(context.Background(), job.ID))
	require.NoError(t, srv.Stop(context.Background()))

	srv = newWebhookIntegrationServer(t, dbPath)
	repos, err = srv.Repos().List(context.Background())
	require.NoError(t, err)
	require.NoError(t, srv.triggers.Rebuild(repos))
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	for i, delivery := range []string{"delivery-1", "delivery-2"} {
		rec = httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, signedGitHubRequest("push", delivery, authorized))
		require.Equal(t, http.StatusOK, rec.Code, fmt.Sprintf("redelivery %d: %s", i, rec.Body.String()))
	}
	jobs, err = srv.queue.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "completion, restart, and changed delivery ID must not recreate webhook work")
}

func TestServerDirectListenerDefaultsAndRemoteRejection(t *testing.T) {
	t.Parallel()
	srv, err := New(Config{DBPath: testDBPath(t), ModelEndpoint: "http://127.0.0.1:9999/v1"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	require.Equal(t, "127.0.0.1:9091", srv.cfg.WebhookAddr)
	require.Equal(t, "127.0.0.1:9090", srv.cfg.DashboardAddr)
	require.Equal(t, 64<<10, srv.http.MaxHeaderBytes)
	require.Equal(t, 15*time.Second, srv.http.ReadTimeout)
	require.Equal(t, 60*time.Second, srv.http.IdleTimeout)
	require.Equal(t, 64<<10, srv.dashHTTP.MaxHeaderBytes)
	for _, cfg := range []Config{
		{WebhookAddr: "0.0.0.0:9091", DashboardAddr: "127.0.0.1:9090"},
		{WebhookAddr: "127.0.0.1:9091", DashboardAddr: "192.168.1.2:9090"},
	} {
		cfg.DBPath = testDBPath(t)
		cfg.ModelEndpoint = "http://127.0.0.1:9999/v1"
		_, err := New(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "loopback")
	}
}

func TestServerRejectsInvalidDirectActorPoliciesAndNormalizesDuplicates(t *testing.T) {
	t.Parallel()
	overLimit := make([]int64, gh.MaxWebhookActorIDs+1)
	for i := range overLimit {
		overLimit[i] = int64(i + 1)
	}
	for _, actors := range [][]int64{{42, 0}, {-1}, overLimit} {
		_, err := New(Config{DBPath: testDBPath(t), ModelEndpoint: "http://127.0.0.1:9999/v1", WebhookAllowedActorIDs: actors})
		require.Error(t, err)
		require.Contains(t, err.Error(), "actor")
	}
	srv, err := New(Config{DBPath: testDBPath(t), ModelEndpoint: "http://127.0.0.1:9999/v1", WebhookAllowedActorIDs: []int64{42, 42}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })
	require.Equal(t, []int64{42}, srv.cfg.WebhookAllowedActorIDs)
}

func TestServerHealthRejectsMutationMethods(t *testing.T) {
	t.Parallel()
	srv := &Server{}
	rec := httptest.NewRecorder()
	srv.HealthHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestLiveServerNoWebhookPolicyKeepsHealthAndQueueEmpty(t *testing.T) {
	webhookAddr := freeTCPAddr(t)
	dashboardAddr := freeTCPAddr(t)
	srv, err := New(Config{
		WebhookAddr: webhookAddr, DashboardAddr: dashboardAddr, DBPath: testDBPath(t),
		ModelEndpoint: "http://127.0.0.1:9999/v1",
	})
	require.NoError(t, err)
	repoPath := webhookIntegrationRepo(t)
	_, err = srv.Repos().Register(context.Background(), repoPath, "owner/repo", "main")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start(ctx) }()
	waitForHTTPStatus(t, "http://"+webhookAddr+"/healthz", http.StatusOK)

	body := []byte(`{"ref":"refs/heads/main","sender":{"id":42},"repository":{"full_name":"owner/repo"}}`)
	req, err := http.NewRequest(http.MethodPost, "http://"+webhookAddr+"/webhook", bytes.NewReader(body))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	jobs, err := srv.queue.RecentJobs(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, jobs)

	cancel()
	require.NoError(t, <-errCh)
}
