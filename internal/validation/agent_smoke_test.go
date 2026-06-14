/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-smoke-validation.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/validation-matrix-gating.md
- docs/validation/README.md
- docs/validation/agent-smoke/README.md
- docs/product-specs/product-surface.md
- docs/features/F-012-self-improvement-loop.md
*/
package validation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestLoadAgentSmokeMatrixCoversDefaultRoles(t *testing.T) {
	matrix, err := LoadAgentSmokeMatrix(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load matrix: %v", err)
	}
	roles := map[string]bool{}
	for _, c := range matrix.Cases {
		roles[c.Role] = true
	}
	for _, role := range []string{
		"ceo",
		"head-of-strategy",
		"coo",
		"cto-weekly",
		"engineer",
		"qa",
		"security",
		"dependency-manager",
		"release-manager",
		"dogfood",
		"pipeline-fixer",
		"orchestrator",
		"janitor",
		"foundation-maintainer",
	} {
		if !roles[role] {
			t.Fatalf("matrix missing role %s", role)
		}
	}
}

func TestSelectAgentSmokeFastRotatesOneCasePerRole(t *testing.T) {
	matrix, err := LoadAgentSmokeMatrix(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load matrix: %v", err)
	}
	selected, err := SelectAgentSmokeCases(matrix, AgentSmokeOptions{Suite: AgentSmokeSuiteFast, Cycle: "0"})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	seen := map[string]bool{}
	for _, c := range selected {
		if seen[c.Role] {
			t.Fatalf("fast suite selected role %s more than once", c.Role)
		}
		seen[c.Role] = true
	}
	if len(seen) < 10 {
		t.Fatalf("expected broad role coverage, got %d roles", len(seen))
	}
}

func TestRunAgentSmokeGeneratesAndDiscardsEphemeralTarget(t *testing.T) {
	root := t.TempDir()
	report, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot: repoRootForTest(t),
		Root:        root,
		Suite:       AgentSmokeSuiteFast,
		Role:        "ceo",
		ProjectType: "static-web",
		FixtureOnly: true,
	})
	if err != nil {
		t.Fatalf("run smoke: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected report ok: %+v", report)
	}
	if report.Selected != 1 || report.Passed != 1 {
		t.Fatalf("expected one passing case, got selected=%d passed=%d failed=%d", report.Selected, report.Passed, report.Failed)
	}
	if !report.Results[0].Discarded {
		t.Fatalf("successful run should be discarded by default: %+v", report.Results[0])
	}
	if _, err := os.Stat(report.Results[0].RunPath); !os.IsNotExist(err) {
		t.Fatalf("expected run path to be removed, stat err=%v", err)
	}
}

func TestRunAgentSmokeKeepRunsAndCleanupOnly(t *testing.T) {
	root := t.TempDir()
	report, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot: repoRootForTest(t),
		Root:        root,
		Suite:       AgentSmokeSuiteDefault,
		Role:        "engineer",
		ProjectType: "go-api",
		KeepRuns:    true,
		FixtureOnly: true,
	})
	if err != nil {
		t.Fatalf("run smoke: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected report ok: %+v", report)
	}
	runPath := report.Results[0].RunPath
	if _, err := os.Stat(filepath.Join(runPath, "target", ".harness", "manifest.yaml")); err != nil {
		t.Fatalf("expected retained harness manifest: %v", err)
	}
	cleanup, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot: repoRootForTest(t),
		Root:        root,
		CleanupOnly: true,
	})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleanup.Cleaned != 1 {
		t.Fatalf("expected one cleaned run, got %d", cleanup.Cleaned)
	}
	if _, err := os.Stat(runPath); !os.IsNotExist(err) {
		t.Fatalf("expected cleanup to remove run path, stat err=%v", err)
	}
}

func TestRunAgentSmokeExecutesLiveRoleThroughServerPath(t *testing.T) {
	root := t.TempDir()
	deterministic := newAgentSmokeDeterministicChatServer(t, deterministicAgentSmokeDisposition("completed", "ticket_breakdown", "cto-weekly"))
	report, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot:   repoRootForTest(t),
		Root:          root,
		Suite:         AgentSmokeSuiteFast,
		Role:          "ceo",
		ProjectType:   "static-web",
		ModelEndpoint: deterministic.URL(),
		MaxTurns:      2,
		Timeout:       30 * time.Second,
		KeepRuns:      true,
	})
	if err != nil {
		t.Fatalf("run live smoke: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected report ok: %+v", report)
	}
	result := report.Results[0]
	if result.ExecutionMode != "live" {
		t.Fatalf("expected live execution mode, got %+v", result)
	}
	if result.JobID == "" || result.TerminalDisposition != "completed" || result.TerminalSuggested != "cto-weekly" {
		t.Fatalf("expected terminal disposition fields, got %+v", result)
	}
	if deterministic.RequestCount() == 0 {
		t.Fatal("expected deterministic test endpoint to be called")
	}
	if _, err := os.Stat(filepath.Join(result.RunPath, "result.json")); err != nil {
		t.Fatalf("expected retained result.json: %v", err)
	}
}

func TestRunAgentSmokeExecutesCasesInParallel(t *testing.T) {
	root := t.TempDir()
	deterministic := newAgentSmokeDeterministicChatServer(t, deterministicAgentSmokeDisposition("completed", "ticket_breakdown", "cto-weekly"))
	report, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot:   repoRootForTest(t),
		Root:          root,
		Suite:         AgentSmokeSuiteDefault,
		Role:          "ceo",
		ModelEndpoint: deterministic.URL(),
		MaxTurns:      2,
		Parallel:      2,
		Timeout:       30 * time.Second,
	})
	if err != nil {
		t.Fatalf("run parallel smoke: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected report ok: %+v", report)
	}
	if report.Selected < 2 {
		t.Fatalf("expected multiple ceo cases, got %d", report.Selected)
	}
	if deterministic.RequestCount() != report.Selected {
		t.Fatalf("expected one model call per case, got calls=%d selected=%d", deterministic.RequestCount(), report.Selected)
	}
	for _, result := range report.Results {
		if result.ExecutionMode != "live" || result.TerminalDisposition != "completed" {
			t.Fatalf("expected live completed result, got %+v", result)
		}
		if !result.Discarded {
			t.Fatalf("successful parallel run should be discarded by default: %+v", result)
		}
	}
}

func TestSetManifestRoleMaxTurns(t *testing.T) {
	in := "roles:\n  ceo:\n    prompt: roles/ceo.md\n    model: reasoning\n    tools: [file_read]\n  engineer:\n    prompt: roles/engineer.md\n    model: coding\n    max_turns: 100\n    tools: [file_read]\n"
	out, changed, err := setManifestRoleMaxTurns(in, "ceo", 3)
	if err != nil {
		t.Fatalf("set ceo max turns: %v", err)
	}
	if !changed || !strings.Contains(out, "  ceo:\n    prompt: roles/ceo.md\n    model: reasoning\n    max_turns: 3\n") {
		t.Fatalf("expected ceo max_turns inserted, got:\n%s", out)
	}
	out, changed, err = setManifestRoleMaxTurns(out, "engineer", 4)
	if err != nil {
		t.Fatalf("set engineer max turns: %v", err)
	}
	if !changed || !strings.Contains(out, "    max_turns: 4\n") {
		t.Fatalf("expected engineer max_turns updated, got:\n%s", out)
	}
}

func TestAgentSmokeReportSummaryAndMarkdown(t *testing.T) {
	cleanup := AgentSmokeReport{Root: "/tmp/smoke", CleanupOnly: true, Cleaned: 2}
	if got := cleanup.Summary(); !strings.Contains(got, "removed 2 retained") {
		t.Fatalf("unexpected cleanup summary %q", got)
	}
	report := AgentSmokeReport{
		Root:        "/tmp/smoke",
		Suite:       AgentSmokeSuiteFast,
		Evidence:    "endpoint-override",
		ModelSource: "operator-supplied OpenAI-compatible endpoint; AD-296 requires this to be a real model endpoint for validation claims",
		Selected:    1,
		Passed:      1,
		Results: []AgentSmokeResult{{
			Role:                "ceo",
			CaseID:              "static-web-empty",
			ProjectType:         "static-web",
			Status:              "passed",
			ExecutionMode:       "live",
			TerminalDisposition: "completed",
			Discarded:           true,
		}},
	}
	if got := report.Summary(); !strings.Contains(got, "1 passed, 0 failed") {
		t.Fatalf("unexpected report summary %q", got)
	}
	out := filepath.Join(t.TempDir(), "report.md")
	if err := writeAgentSmokeMarkdownReport(out, report); err != nil {
		t.Fatalf("write markdown report: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	if !strings.Contains(string(data), "`live`") || !strings.Contains(string(data), "`completed`") || !strings.Contains(string(data), "AD-296") {
		t.Fatalf("expected mode and disposition in report:\n%s", string(data))
	}
}

func TestClassifyAgentSmokeError(t *testing.T) {
	cases := map[string]string{
		"policy: blocked":                                FailureToolPolicy,
		"fixture missing":                                FailureFixtureInvalid,
		"model unavailable":                              FailureEnvironmentModel,
		"executor: agent loop failed":                    FailureRoleBehavior,
		"expected disposition completed got blocked":     FailureRoleBehavior,
		"dispatch mode requires job_disposition_record":  FailureDispatchContext,
		"executor: role \"ghost\" not found in manifest": FailureDispatchContext,
		"unknown project type \"x\"":                     FailureProjectTypeGap,
		"something else":                                 FailureFoundationGeneration,
	}
	for msg, want := range cases {
		if got := classifyAgentSmokeError(errors.New(msg)); got != want {
			t.Fatalf("classify %q got %q want %q", msg, got, want)
		}
	}
	if got := classifyAgentSmokeError(nil); got != "" {
		t.Fatalf("nil error classified as %q", got)
	}
}

func TestSeedProjectFilesCoversProjectTypes(t *testing.T) {
	projectTypes := []string{
		"static-web",
		"react-web",
		"browser-game-phaser",
		"canvas-game-vanilla",
		"go-api",
		"go-cli",
		"go-library",
		"docs-site",
		"existing-maintenance",
	}
	for _, projectType := range projectTypes {
		writes := map[string]string{}
		call := func(name string, args any) error {
			if name != "file_write" {
				t.Fatalf("unexpected tool %s", name)
			}
			raw, _ := json.Marshal(args)
			var file struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(raw, &file); err != nil {
				t.Fatalf("decode file args: %v", err)
			}
			writes[file.Path] = file.Content
			return nil
		}
		if err := seedProjectFiles(call, AgentSmokeCase{ProjectType: projectType}); err != nil {
			t.Fatalf("seed %s: %v", projectType, err)
		}
		if len(writes) == 0 {
			t.Fatalf("expected writes for %s", projectType)
		}
	}
	if err := seedProjectFiles(func(string, any) error { return nil }, AgentSmokeCase{ProjectType: "unknown"}); err == nil {
		t.Fatal("expected unknown project type error")
	}
}

func TestSeedReportsSpecialStateAndHelpers(t *testing.T) {
	writes := map[string]string{}
	call := func(name string, args any) error {
		raw, _ := json.Marshal(args)
		var file struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatalf("decode file args: %v", err)
		}
		writes[file.Path] = file.Content
		return nil
	}
	c := AgentSmokeCase{
		ID:                  "release-blocker-drift-failure",
		Role:                "orchestrator",
		ProjectType:         "go-api",
		Stage:               "ready blocked failure",
		ExpectedDisposition: "blocked",
		Trigger:             map[string]string{"source_role": "qa"},
	}
	if err := seedReports(call, c); err != nil {
		t.Fatalf("seed reports: %v", err)
	}
	if err := seedSpecialState(call, c); err != nil {
		t.Fatalf("seed special state: %v", err)
	}
	for _, path := range []string{
		"docs/reports/qa/" + c.ID + ".md",
		"docs/reports/security/" + c.ID + ".md",
		"docs/reports/dogfood/" + c.ID + ".md",
		".mars/checks/latest.json",
		".mars/orgstate/source-disposition.json",
		"docs/reports/dogfood/doctrine-drift.md",
		"docs/reports/release/release-blocked.md",
		"CHANGELOG.md",
		"VERSION",
	} {
		if _, ok := writes[path]; !ok {
			t.Fatalf("expected seeded path %s, got %#v", path, writes)
		}
	}
	if !strings.Contains(strategyForCase(c), c.ID) || !strings.Contains(planForCase(c), c.ProjectType) {
		t.Fatal("expected strategy and plan helpers to include case context")
	}
	args := ticketArgsForCase(c)
	if args["verified_by"] == "" || args["evidence_links"] == nil {
		t.Fatalf("expected completed-ticket evidence fields, got %#v", args)
	}
}

func TestResolveAndValidateAgentSmokeErrors(t *testing.T) {
	root, err := resolveAgentSmokeRoot(AgentSmokeOptions{HarnessRoot: "/tmp/harness"})
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(root), "/demo/validation-runs/agent-smoke") {
		t.Fatalf("unexpected root %s", root)
	}
	if cleaned, err := cleanupAgentSmokeRuns(filepath.Join(t.TempDir(), "missing")); err != nil || cleaned != 0 {
		t.Fatalf("cleanup missing got cleaned=%d err=%v", cleaned, err)
	}
	err = ValidateAgentSmokeMatrix(AgentSmokeMatrix{Cases: []AgentSmokeCase{{ID: "bad"}}}, repoRootForTest(t))
	if err == nil || !strings.Contains(err.Error(), "missing role") {
		t.Fatalf("expected missing role error, got %v", err)
	}
	_, err = SelectAgentSmokeCases(AgentSmokeMatrix{}, AgentSmokeOptions{Suite: "bogus"})
	if err == nil || !strings.Contains(err.Error(), "unsupported suite") {
		t.Fatalf("expected unsupported suite error, got %v", err)
	}
}

type agentSmokeDeterministicChatServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	response llm.ChatCompletionResponse
	requests int
}

func newAgentSmokeDeterministicChatServer(t *testing.T, response llm.ChatCompletionResponse) *agentSmokeDeterministicChatServer {
	t.Helper()
	f := &agentSmokeDeterministicChatServer{response: response}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		f.mu.Lock()
		f.requests++
		f.mu.Unlock()
		if err := json.NewEncoder(w).Encode(f.response); err != nil {
			t.Fatalf("encode deterministic test response: %v", err)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *agentSmokeDeterministicChatServer) URL() string {
	return f.server.URL
}

func (f *agentSmokeDeterministicChatServer) RequestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

func deterministicAgentSmokeDisposition(status, nextNeed, suggestedRole string) llm.ChatCompletionResponse {
	args, _ := json.Marshal(map[string]any{
		"status":         status,
		"next_need":      nextNeed,
		"suggested_role": suggestedRole,
		"reason":         "deterministic executor-path test terminal disposition",
		"evidence_links": []string{"agent-smoke deterministic executor-path test"},
	})
	return llm.ChatCompletionResponse{Choices: []llm.Choice{{
		Message: llm.Message{
			Role: "assistant",
			ToolCalls: []llm.ToolCall{{
				ID:   "terminal-disposition",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "job_disposition_record",
					Arguments: string(args),
				},
			}},
		},
		FinishReason: "tool_calls",
	}}}
}
