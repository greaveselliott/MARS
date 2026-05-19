/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dogfood-matrix.md
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
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/stretchr/testify/require"
)

func TestFoundationAcceptanceFreshBootstrapHappyPath(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, false)
	fake := newFakeChatServer(t,
		fakeToolResponse("write-probe", "file_write", `{"path":"docs/exec-plans/backlog/foundation-acceptance-probe.md","content":"foundation gate passed\n"}`),
		fakeToolResponse("commit-probe", "git_commit", `{"message":"test: commit foundation acceptance probe"}`),
		fakeToolResponse("record-disposition", "job_disposition_record", `{"status":"completed","reason":"foundation acceptance probe completed"}`),
		fakeTextResponse("Done."),
	)

	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	job := &queue.Job{ID: "job-happy", RepoID: repoID, Role: "coo", Trigger: `{"type":"acceptance"}`}

	require.NoError(t, exec.Execute(ctx, job))
	srv.handleJobComplete(ctx, job)

	require.Equal(t, 3, fake.RequestCount(), "job_disposition_record is terminal in dispatch mode")
	data, err := os.ReadFile(filepath.Join(repo, "docs", "exec-plans", "backlog", "foundation-acceptance-probe.md"))
	require.NoError(t, err)
	require.Equal(t, "foundation gate passed\n", string(data))
	require.Equal(t, 0, countInterventionDebtTickets(t, repo))
	require.Equal(t, 1, countOutcomes(t, srv, "coo", "passed"))
}

func TestFoundationAcceptanceDispatchProseCompletionRepromptsForDisposition(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, false)
	fake := newFakeChatServer(t,
		fakeTextResponse("QA approves the change."),
		fakeToolResponse("record-disposition", "job_disposition_record", `{"status":"approved","next_need":"no_need","reason":"QA approved after reviewing the available evidence.","evidence_links":["git log --oneline -10"]}`),
		fakeTextResponse("Done."),
	)

	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	job := &queue.Job{ID: "job-qa-prose", RepoID: repoID, Role: "qa", Trigger: `{"type":"acceptance"}`}

	require.NoError(t, exec.Execute(ctx, job))

	require.Equal(t, 2, fake.RequestCount(), "prose-only QA completion should be reprompted until job_disposition_record is called")
	disposition, err := srv.orgStore.GetDisposition(ctx, job.ID)
	require.NoError(t, err)
	require.NotNil(t, disposition)
	require.Equal(t, "approved", disposition.Status)
	require.Equal(t, "no_need", disposition.NextNeed)
	require.Equal(t, []string{"git log --oneline -10"}, disposition.EvidenceLinks)
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
	require.Equal(t, 3, fake.RequestCount(), "dispatch protocol should give one terminal-disposition reminder, then fail through the existing ticket gate")
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
	require.Equal(t, 0, countJobsByStatus(t, srv, "pending"), "deterministic containment failure should not enqueue dispatch loops")
	require.Equal(t, 0, countJobsByStatusAndRole(t, srv, "pending", "orchestrator"), "deterministic containment should wait for operator cleanup")
	require.Equal(t, 0, countJobsByStatusAndRole(t, srv, "pending", "engineer"), "deterministic containment failure should not enqueue same-role recovery")
}

func TestFoundationAcceptanceBroaderDogfoodLoopCoversTicketCommitPushAndQualityExport(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, false)
	canonicalRepo, err := filepath.EvalSymlinks(repo)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Foundation acceptance target\n"), 0o644))
	commitFoundationTarget(t, repo, "add target readme")
	runFoundationGit(t, repo, "branch", "-M", "main")
	bin, cliLog := writeFoundationFakeMarsHarnessBinary(t)
	t.Setenv("MARS_HARNESS_CLI_BIN", bin)

	reportPath := "docs/reports/dogfood/foundation-broader-loop.md"
	fake := newFakeChatServer(t,
		fakeToolResponseArgs(t, "write-report", "file_write", map[string]any{
			"path":    reportPath,
			"content": "# Foundation Broader Dogfood Loop\n\n- test: `test -f README.md`\n- push: attempted with `git_push`\n- quality: `mars-harness scores export --repo .`\n",
		}),
		fakeToolResponseArgs(t, "run-test", "shell_exec", map[string]any{
			"argv":            []string{"test", "-f", "README.md"},
			"timeout_seconds": 5,
		}),
		fakeToolResponseArgs(t, "create-finding", "ticket_create", map[string]any{
			"title":               "Broader dogfood loop finding",
			"priority":            "medium",
			"complexity":          "small",
			"kind":                "intervention-debt",
			"work_type":           "intervention-debt",
			"end_to_end_evidence": "not_applicable",
			"evidence_links":      []string{reportPath},
			"verified_by":         "fake-LLM dogfood loop",
			"dedupe_key":          "dogfood:foundation:broader-loop:quality-export",
			"metadata": map[string]string{
				"category": "dogfood_broader_loop",
				"severity": "medium",
			},
			"source": "MH-049 broader fake-LLM dogfood loop",
			"body": strings.Join([]string{
				"## Context",
				"",
				"The broader fake-LLM dogfood loop found a target-owned validation finding.",
				"",
				"## Requirements",
				"",
				"- Keep the finding deduped through ticket_create.",
				"- Preserve test, commit, push, scoring, and quality-export evidence.",
				"",
				"## Acceptance Criteria",
				"",
				"- [ ] Evidence links identify the dogfood report.",
				"- [ ] Follow-up work decides whether the target or foundation owns the fix.",
			}, "\n"),
		}),
		fakeToolResponseArgs(t, "commit-evidence", "git_commit", map[string]any{
			"message": "test(dogfood): record broader fake loop",
		}),
		fakeToolResponseArgs(t, "push-evidence", "git_push", map[string]any{}),
		fakeToolResponseArgs(t, "export-quality", "mars_harness_cli", map[string]any{
			"mode":            "run",
			"args":            []string{"scores", "export"},
			"repo":            ".",
			"timeout_seconds": 5,
		}),
		fakeToolResponseArgs(t, "record-disposition", "job_disposition_record", map[string]any{
			"status":         "completed",
			"next_need":      "no_need",
			"reason":         "Broader fake dogfood loop covered ticket creation, test execution, commit, push attempt, scoring outcome, and quality export.",
			"evidence_links": []string{reportPath, "mars-harness scores export --repo ."},
		}),
	)

	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	job := &queue.Job{ID: "job-dogfood-broader", RepoID: repoID, Role: "dogfood", Trigger: `{"type":"dogfood_matrix","profile":"broader-fake-llm"}`}

	require.NoError(t, exec.Execute(ctx, job))
	srv.handleJobComplete(ctx, job)

	require.Equal(t, 7, fake.RequestCount(), "broader fake loop should stop after the terminal disposition")
	require.FileExists(t, filepath.Join(repo, reportPath))
	matches, err := filepath.Glob(filepath.Join(repo, "docs", "tickets", "backlog", "T-*-broader-dogfood-loop-finding.md"))
	require.NoError(t, err)
	require.Len(t, matches, 1, "dogfood finding should be materialized through ticket_create")
	require.Equal(t, 1, countInterventionDebtTickets(t, repo), "broader dogfood finding should create exactly one deduped intervention-debt ticket")
	require.Equal(t, 1, countOutcomes(t, srv, "dogfood", "passed"))
	require.Contains(t, readFile(t, cliLog), "args: scores export --repo "+canonicalRepo)
	require.Contains(t, gitOutput(t, repo, "log", "-1", "--pretty=%s"), "test(dogfood): record broader fake loop")
	require.Empty(t, strings.TrimSpace(gitOutput(t, repo, "status", "--porcelain")))
}

func TestFoundationAcceptanceMarsObserverProfileBlocksMutatingTools(t *testing.T) {
	ctx := context.Background()
	repo := setupFoundationTarget(t, false)
	fake := newFakeChatServer(t,
		fakeToolResponseArgs(t, "attempt-write", "file_write", map[string]any{
			"path":    "docs/reports/dogfood/mars-observer.md",
			"content": "observer mode should block this write\n",
		}),
		fakeToolResponseArgs(t, "record-blocked", "job_disposition_record", map[string]any{
			"status":         "blocked",
			"next_need":      "operator_review",
			"reason":         "Observer trust blocked a mutating write before Mars contributor-mode graduation.",
			"evidence_links": []string{"docs/validation/profiles/mars-observer.md"},
		}),
	)
	srv, repoID, exec := setupFoundationServer(t, repo, fake.URL())
	require.NoError(t, srv.trustStore.Set(ctx, "dogfood", repoID, trust.LevelObserver))
	job := &queue.Job{ID: "job-mars-observer", RepoID: repoID, Role: "dogfood", Trigger: `{"type":"dogfood_matrix","profile":"mars-observer"}`}

	require.NoError(t, exec.Execute(ctx, job))

	require.Equal(t, 2, fake.RequestCount(), "observer trust should return the policy block to the model and allow a blocked disposition")
	require.NoFileExists(t, filepath.Join(repo, "docs", "reports", "dogfood", "mars-observer.md"))
	require.Equal(t, 0, countInterventionDebtTickets(t, repo), "observer-mode foundation policy blocks should not write target intervention debt")
	disposition, err := srv.orgStore.GetDisposition(ctx, job.ID)
	require.NoError(t, err)
	require.NotNil(t, disposition)
	require.Equal(t, "blocked", disposition.Status)
	require.Equal(t, "operator_review", disposition.NextNeed)
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

func fakeToolResponseArgs(t *testing.T, id, name string, args any) llm.ChatCompletionResponse {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	return fakeToolResponse(id, name, string(raw))
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

func writeFoundationFakeMarsHarnessBinary(t *testing.T) (string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake mars-harness shell script is POSIX-only")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "mars-harness")
	logPath := filepath.Join(dir, "cli.log")
	t.Setenv("MARS_HARNESS_FAKE_CLI_LOG", logPath)
	script := `#!/bin/sh
printf 'args:' >> "$MARS_HARNESS_FAKE_CLI_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$MARS_HARNESS_FAKE_CLI_LOG"
done
printf '\n' >> "$MARS_HARNESS_FAKE_CLI_LOG"
if [ "$1" = "scores" ] && [ "$2" = "export" ]; then
  echo "quality score exported"
  exit 0
fi
echo "fake mars-harness"
`
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	return bin, logPath
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func gitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}
