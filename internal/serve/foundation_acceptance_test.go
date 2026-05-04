/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/stretchr/testify/require"
)

func TestFoundationAcceptanceFreshBootstrapHappyPath(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, false)
	fake := newFakeChatServer(t,
		fakeToolResponse("write-probe", "file_write", `{"path":"docs/probe.txt","content":"foundation gate passed\n"}`),
		fakeToolResponse("record-disposition", "job_disposition_record", `{"status":"completed","reason":"foundation acceptance probe completed"}`),
		fakeTextResponse("Done."),
	)

	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	job := &queue.Job{ID: "job-happy", RepoID: repoID, Role: "coo", Trigger: `{"type":"acceptance"}`}

	require.NoError(t, exec.Execute(ctx, job))
	srv.handleJobComplete(ctx, job)

	require.Equal(t, 3, fake.RequestCount())
	data, err := os.ReadFile(filepath.Join(repo, "docs", "probe.txt"))
	require.NoError(t, err)
	require.Equal(t, "foundation gate passed\n", string(data))
	require.Equal(t, 0, countInterventionDebtTickets(t, repo))
	require.Equal(t, 1, countOutcomes(t, srv, "coo", "passed"))
}

func TestFoundationAcceptancePolicyBlockSuppressesTicketGateFallout(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, true)
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "src", "keep.txt"), []byte("keep\n"), 0o644))
	commitFoundationTarget(t, repo, "add source fixture")

	fake := newFakeChatServer(t,
		fakeToolResponse("delete-src", "shell_exec", `{"shell_command":"rm -rf src"}`),
		fakeTextResponse("Blocked by policy."),
	)
	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	job := &queue.Job{ID: "job-policy", RepoID: repoID, Role: "engineer", Trigger: `{"type":"acceptance"}`}

	err := exec.Execute(ctx, job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ticket gate")
	srv.handleJobFailed(ctx, job, err)

	require.FileExists(t, filepath.Join(repo, "src", "keep.txt"))
	require.Equal(t, 2, fake.RequestCount())
	require.Equal(t, 0, countInterventionDebtTickets(t, repo), "foundation-owned policy and ticket-gate fallout should stay out of the target backlog")
	require.Equal(t, 1, countTelemetryByCategory(t, srv, "guardrail_block"))
	require.Equal(t, 1, countTelemetryByCategory(t, srv, "ticket_gate"))
}

func TestFoundationAcceptanceDirtyWorktreeContainmentSkipsLLMAndRecovery(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, true)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "dirty-large.txt"), []byte(strings.Repeat("dirty\n", 600)), 0o644))
	fake := newFakeChatServer(t, fakeTextResponse("should not be called"))
	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	job := &queue.Job{ID: "job-dirty", RepoID: repoID, Role: "engineer", Trigger: `{"type":"acceptance"}`}

	err := exec.Execute(ctx, job)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dirty worktree containment")
	require.Contains(t, err.Error(), "blast radius exceeded")
	srv.handleJobFailed(ctx, job, err)
	srv.handleJobFailed(ctx, job, err)

	require.Equal(t, 0, fake.RequestCount(), "dirty preflight must happen before LLM invocation")
	require.Equal(t, 0, countInterventionDebtTickets(t, repo), "foundation-owned containment failures should stay out of the target backlog")
	require.Equal(t, 1, countJobsByStatus(t, srv, "pending"), "deterministic containment failure should enqueue a single dispatch review")
	require.Equal(t, 1, countJobsByStatusAndRole(t, srv, "pending", "orchestrator"), "dispatch review should return to Orchestrator")
	require.Equal(t, 0, countJobsByStatusAndRole(t, srv, "pending", "engineer"), "deterministic containment failure should not enqueue same-role recovery")
}

func setupFoundationTarget(t *testing.T, withTicket bool) string {
	t.Helper()
	repo := t.TempDir()
	runFoundationGit(t, repo, "init")
	runFoundationGit(t, repo, "config", "user.email", "test@example.com")
	runFoundationGit(t, repo, "config", "user.name", "Test User")
	require.NoError(t, scanner.Init(repo, false))
	if withTicket {
		mustWriteFile(t, filepath.Join(repo, "docs", "tickets", "backlog", "T-001-acceptance-ticket.md"), []byte(`---
id: T-001
title: Acceptance ticket
priority: high
work_type: enabler
blocker: none
blocked_by: []
---

# T-001: Acceptance ticket
`))
	}
	commitFoundationTarget(t, repo, "initial harness baseline")
	return repo
}

func setupFoundationServer(t *testing.T, repo, fallbackURL string) (*Server, string, *Executor) {
	t.Helper()
	srv, err := New(Config{
		WebhookAddr:   "127.0.0.1:0",
		DashboardAddr: "127.0.0.1:0",
		DBPath:        testDBPath(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })

	repoID, err := srv.Repos().Register(context.Background(), repo, "", "main")
	require.NoError(t, err)

	router := inference.NewRouter(inference.RouterConfig{FallbackURL: fallbackURL})
	lookup := func(ctx context.Context, id string) (string, error) {
		rec, err := srv.repos.FindByID(ctx, id)
		if err != nil {
			return "", err
		}
		if rec == nil {
			return "", fmt.Errorf("repo %s not found", id)
		}
		return rec.Path, nil
	}
	exec := NewExecutor(lookup, router, srv.traceStore, srv.trustStore)
	exec.SetInterventionSignalHandler(srv.recordInterventionDebtSignal)
	exec.SetOrgState(srv.orgStore)
	return srv, repoID, exec
}

type fakeChatServer struct {
	server    *httptest.Server
	mu        sync.Mutex
	responses []llm.ChatCompletionResponse
	requests  int
}

func newFakeChatServer(t *testing.T, responses ...llm.ChatCompletionResponse) *fakeChatServer {
	t.Helper()
	f := &fakeChatServer{responses: responses}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests++
		idx := f.requests - 1
		if idx >= len(f.responses) {
			idx = len(f.responses) - 1
		}
		require.NoError(t, json.NewEncoder(w).Encode(f.responses[idx]))
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeChatServer) URL() string {
	return f.server.URL
}

func (f *fakeChatServer) RequestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

func fakeToolResponse(id, name, args string) llm.ChatCompletionResponse {
	return llm.ChatCompletionResponse{Choices: []llm.Choice{{
		Message: llm.Message{
			Role: "assistant",
			ToolCalls: []llm.ToolCall{{
				ID:   id,
				Type: "function",
				Function: llm.FunctionCall{
					Name:      name,
					Arguments: args,
				},
			}},
		},
	}}}
}

func fakeTextResponse(content string) llm.ChatCompletionResponse {
	return llm.ChatCompletionResponse{Choices: []llm.Choice{{
		Message: llm.Message{Role: "assistant", Content: content},
	}}}
}

func commitFoundationTarget(t *testing.T, repo, message string) {
	t.Helper()
	runFoundationGit(t, repo, "add", ".")
	runFoundationGit(t, repo, "commit", "-m", message)
}

func runFoundationGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func countInterventionDebtTickets(t *testing.T, repo string) int {
	t.Helper()
	var count int
	for _, status := range []string{"backlog", "in-progress", "done"} {
		entries, err := os.ReadDir(filepath.Join(repo, "docs", "tickets", status))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(repo, "docs", "tickets", status, entry.Name()))
			require.NoError(t, err)
			if strings.Contains(string(data), "kind: intervention-debt") || strings.Contains(string(data), "work_type: intervention-debt") {
				count++
			}
		}
	}
	return count
}

func countTelemetryByCategory(t *testing.T, srv *Server, category string) int {
	t.Helper()
	var count int
	require.NoError(t, srv.db.QueryRow(`SELECT COUNT(*) FROM telemetry_events WHERE category = ?`, category).Scan(&count))
	return count
}

func countOutcomes(t *testing.T, srv *Server, role, outcome string) int {
	t.Helper()
	var count int
	require.NoError(t, srv.db.QueryRow(`SELECT COUNT(*) FROM outcomes WHERE role = ? AND type = ?`, role, outcome).Scan(&count))
	return count
}
