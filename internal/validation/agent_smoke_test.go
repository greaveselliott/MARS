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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
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
	deterministic := newAgentSmokeDeterministicChatServer(t, deterministicAgentSmokeDisposition("completed", "exec_plan", "coo"))
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
	if result.JobID == "" || result.TerminalDisposition != "completed" || result.TerminalSuggested != "coo" {
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
	deterministic := newAgentSmokeDeterministicChatServer(t, deterministicAgentSmokeDisposition("completed", "exec_plan", "coo"))
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

func TestNormalizeAgentSmokeOptionsDefaultsSingleServerTier(t *testing.T) {
	opts := normalizeAgentSmokeOptions(AgentSmokeOptions{
		SingleServer: true,
		SingleTier:   "",
	})
	if opts.SingleTier != "coding" {
		t.Fatalf("expected coding single-server tier, got %q", opts.SingleTier)
	}
}

func TestParseAgentSmokeSingleTier(t *testing.T) {
	tier, err := parseAgentSmokeSingleTier("reasoning")
	if err != nil {
		t.Fatalf("parse tier: %v", err)
	}
	if string(tier) != "reasoning" {
		t.Fatalf("unexpected tier %q", tier)
	}
	if _, err := parseAgentSmokeSingleTier("not-a-tier"); err == nil {
		t.Fatal("expected invalid tier error")
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

func TestAssertAgentSmokeDispositionChecksRouting(t *testing.T) {
	c := AgentSmokeCase{
		ID:                  "cto-to-engineer-api",
		Role:                "orchestrator",
		ExpectedDisposition: "completed",
		WouldDispatch:       "engineer",
		Trigger:             map[string]string{"next_need": "implementation", "suggested_role": "engineer"},
	}
	if err := assertAgentSmokeDisposition(c, &orgstate.Disposition{Status: "completed", NextNeed: "implementation", SuggestedRole: "engineer"}); err != nil {
		t.Fatalf("expected matching routing to pass: %v", err)
	}
	err := assertAgentSmokeDisposition(c, &orgstate.Disposition{Status: "completed", NextNeed: "qa_review", SuggestedRole: "engineer"})
	if err == nil || !strings.Contains(err.Error(), "expected next_need") {
		t.Fatalf("expected next_need mismatch, got %v", err)
	}
	err = assertAgentSmokeDisposition(c, &orgstate.Disposition{Status: "completed", NextNeed: "implementation", SuggestedRole: "qa"})
	if err == nil || !strings.Contains(err.Error(), "expected suggested role") {
		t.Fatalf("expected suggested role mismatch, got %v", err)
	}
	stop := AgentSmokeCase{
		ID:                  "release-blocked-stop-heldout",
		Role:                "orchestrator",
		ExpectedDisposition: "blocked",
		WouldDispatch:       "",
		Trigger:             map[string]string{"next_need": "no_need"},
	}
	err = assertAgentSmokeDisposition(stop, &orgstate.Disposition{Status: "blocked", NextNeed: "no_need", SuggestedRole: "engineer"})
	if err == nil || !strings.Contains(err.Error(), "expected no suggested role") {
		t.Fatalf("expected stop case route rejection, got %v", err)
	}
}

func TestAgentSmokeReportSummaryAndMarkdown(t *testing.T) {
	cleanup := AgentSmokeReport{Root: "/tmp/smoke", CleanupOnly: true, Cleaned: 2}
	if got := cleanup.Summary(); !strings.Contains(got, "removed 2 retained") {
		t.Fatalf("unexpected cleanup summary %q", got)
	}
	report := AgentSmokeReport{
		Root:           "/tmp/smoke",
		Suite:          AgentSmokeSuiteFast,
		Evidence:       "local-model",
		ModelSource:    "local Mars Harness inference router; single local server tier coding",
		SingleServer:   true,
		SingleTier:     "coding",
		ServerParallel: 2,
		Selected:       1,
		Passed:         1,
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
	if !strings.Contains(string(data), "`live`") || !strings.Contains(string(data), "`completed`") || !strings.Contains(string(data), "single local server tier `coding`") {
		t.Fatalf("expected mode and disposition in report:\n%s", string(data))
	}

	endpointReport := report
	endpointReport.Evidence = "endpoint-override"
	endpointReport.ModelSource = "operator-supplied OpenAI-compatible endpoint; AD-296 requires this to be a real model endpoint for validation claims"
	endpointReport.SingleServer = false
	endpointReport.SingleTier = ""
	endpointReport.ServerParallel = 0
	out = filepath.Join(t.TempDir(), "endpoint-report.md")
	if err := writeAgentSmokeMarkdownReport(out, endpointReport); err != nil {
		t.Fatalf("write endpoint markdown report: %v", err)
	}
	data, err = os.ReadFile(out)
	if err != nil {
		t.Fatalf("read endpoint markdown report: %v", err)
	}
	if !strings.Contains(string(data), "AD-296") {
		t.Fatalf("expected endpoint warning in report:\n%s", string(data))
	}
}

func TestClassifyAgentSmokeError(t *testing.T) {
	cases := map[string]string{
		"policy: blocked":                                FailureToolPolicy,
		"fixture missing":                                FailureFixtureInvalid,
		"model unavailable":                              FailureEnvironmentModel,
		"executor: agent loop failed":                    FailureRoleBehavior,
		"executor: agent ended with max_turns":           FailureRoleBehavior,
		"executor: agent ended with empty_response":      FailureRoleBehavior,
		"executor: ticket gate blocked completion":       FailureRoleBehavior,
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
		if projectType == "browser-game-phaser" {
			if !strings.Contains(writes["index.html"], `id="game"`) || !strings.Contains(writes["index.html"], `/src/main.js`) {
				t.Fatalf("expected browser-game-phaser to seed Vite index.html entry, got %q", writes["index.html"])
			}
			if !strings.Contains(writes["src/main.js"], "new Phaser.Game") || !strings.Contains(writes["src/main.js"], "parent: 'game'") {
				t.Fatalf("expected browser-game-phaser to seed product smokeable Phaser entrypoint, got %q", writes["src/main.js"])
			}
		}
		if projectType == "react-web" {
			if !strings.Contains(writes["index.html"], `id="root"`) || !strings.Contains(writes["index.html"], `/src/main.jsx`) {
				t.Fatalf("expected react-web to seed Vite index.html entry, got %q", writes["index.html"])
			}
			if !strings.Contains(writes["src/main.jsx"], "createRoot") || !strings.Contains(writes["src/App.jsx"], `id="game"`) {
				t.Fatalf("expected react-web to seed product smokeable React entrypoint, got main=%q app=%q", writes["src/main.jsx"], writes["src/App.jsx"])
			}
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

	defectWrites := map[string]string{}
	defectCall := func(name string, args any) error {
		raw, _ := json.Marshal(args)
		var file struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatalf("decode defect file args: %v", err)
		}
		defectWrites[file.Path] = file.Content
		return nil
	}
	defect := AgentSmokeCase{ID: "dogfood-go-cli-defect-heldout", Role: "dogfood", Stage: "defect", ExpectedDisposition: "changes_requested"}
	if err := seedReports(defectCall, defect); err != nil {
		t.Fatalf("seed defect reports: %v", err)
	}
	if err := seedSpecialState(defectCall, defect); err != nil {
		t.Fatalf("seed defect state: %v", err)
	}
	if !strings.Contains(defectWrites["docs/reports/dogfood/seeded-defect.md"], "must request implementation rework") {
		t.Fatalf("expected seeded dogfood defect marker, got %#v", defectWrites)
	}
	if !strings.Contains(defectWrites[".mars/checks/latest.json"], `"status":"failed"`) {
		t.Fatalf("expected failed check marker for defect case, got %#v", defectWrites)
	}

	qaWrites := map[string]string{}
	qaCall := func(name string, args any) error {
		raw, _ := json.Marshal(args)
		var file struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatalf("decode QA file args: %v", err)
		}
		qaWrites[file.Path] = file.Content
		return nil
	}
	negativeQA := AgentSmokeCase{ID: "go-cli-missing-evidence-heldout", Role: "qa", Stage: "after-engineer", ExpectedDisposition: "changes_requested"}
	if err := seedReports(qaCall, negativeQA); err != nil {
		t.Fatalf("seed negative QA reports: %v", err)
	}
	if _, ok := qaWrites["docs/reports/qa/go-cli-missing-evidence-heldout.md"]; ok {
		t.Fatalf("QA role under test should not receive a prewritten case report: %#v", qaWrites)
	}
	if !strings.Contains(qaWrites["docs/reports/qa/seeded-gap-go-cli-missing-evidence-heldout.md"], "must request implementation rework") {
		t.Fatalf("expected seeded QA gap marker, got %#v", qaWrites)
	}

	dogfoodReadyWrites := map[string]string{}
	dogfoodReadyCall := func(name string, args any) error {
		raw, _ := json.Marshal(args)
		var file struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatalf("decode dogfood ready args: %v", err)
		}
		dogfoodReadyWrites[file.Path] = "written"
		return nil
	}
	dogfoodReady := AgentSmokeCase{ID: "dogfood-react-web-ready", Role: "dogfood", Stage: "ready", ExpectedDisposition: "approved"}
	if err := seedReports(dogfoodReadyCall, dogfoodReady); err != nil {
		t.Fatalf("seed dogfood ready reports: %v", err)
	}
	if _, ok := dogfoodReadyWrites["docs/reports/dogfood/dogfood-react-web-ready.md"]; ok {
		t.Fatalf("dogfood role under test should not receive a prewritten dogfood report: %#v", dogfoodReadyWrites)
	}
}

func TestMarkTicketStaleSeedsExistingEvidenceForJanitorLifecycle(t *testing.T) {
	dir := t.TempDir()
	ticketPath := "docs/tickets/in-progress/T-001-exercise-go-api-smoke-case-stale-api-ticket.md"
	abs := filepath.Join(dir, filepath.FromSlash(ticketPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("---\nevidence_links: []\nverified_by: \"TBD\"\n---\n# Ticket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var wrote string
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
			t.Fatalf("decode file_write: %v", err)
		}
		if file.Path != ticketPath {
			t.Fatalf("unexpected path %q", file.Path)
		}
		wrote = file.Content
		return nil
	}
	if err := markTicketStale(call, dir, ticketPath, AgentSmokeCase{ID: "stale-api-ticket"}); err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	for _, want := range []string{
		`evidence_links: [".mars/checks/latest.json"]`,
		`verified_by: "foundation-validation-seeder"`,
		"with existing evidence",
	} {
		if !strings.Contains(wrote, want) {
			t.Fatalf("expected stale ticket to contain %q:\n%s", want, wrote)
		}
	}
}

func TestAgentSmokeCaseContractAndTriggerAreTargetLocal(t *testing.T) {
	c := AgentSmokeCase{
		ID:                  "static-web-ticket",
		Role:                "engineer",
		ProjectType:         "static-web",
		Stage:               "ticket",
		ExpectedDisposition: "completed",
		WouldDispatch:       "qa",
		RequiredArtifacts:   []string{"index.html"},
		ForbiddenMutations:  []string{"docs/tickets/done/T-999-forbidden.md"},
		Trigger:             map[string]string{"next_need": "qa_review"},
	}
	trigger := agentSmokeTrigger(c)
	if trigger["case_contract_path"] != agentSmokeCaseContractPath {
		t.Fatalf("expected trigger to route target-local contract path, got %#v", trigger["case_contract_path"])
	}
	if got := fmt.Sprint(trigger["case_contract_instruction"]); !strings.Contains(got, "Do not read the foundation matrix") {
		t.Fatalf("expected trigger to forbid target-side matrix reads, got %q", got)
	}
	if got := fmt.Sprint(trigger["case_contract_summary"]); !strings.Contains(got, `shell_exec argv ["git","mv"`) {
		t.Fatalf("expected trigger summary to include role-local completion contract, got %q", got)
	}
	if got := fmt.Sprint(trigger["terminal_disposition_instruction"]); !strings.Contains(got, `suggested_role "qa"`) {
		t.Fatalf("expected terminal disposition instruction to require suggested role, got %q", got)
	}
	dispositionContract, ok := trigger["terminal_disposition_contract"].(map[string]string)
	if !ok {
		t.Fatalf("expected terminal disposition contract map, got %#v", trigger["terminal_disposition_contract"])
	}
	if dispositionContract["status"] != "completed" || dispositionContract["next_need"] != "qa_review" || dispositionContract["suggested_role"] != "qa" {
		t.Fatalf("unexpected terminal disposition contract: %#v", dispositionContract)
	}
	contract := caseContractForCase(c)
	for _, want := range []string{
		"target-local smoke contract",
		"docs/validation/agent-smoke/matrix.yaml",
		"docs/tickets/done/T-001-exercise-static-web-smoke-case-static-web-ticket.md",
		`next_need "qa_review"`,
		"`index.html`",
		"Do not create a parallel `src/` tree",
		`["python3","-m","http.server"`,
		"Do not use `python`",
	} {
		if !strings.Contains(contract, want) {
			t.Fatalf("expected contract to contain %q:\n%s", want, contract)
		}
	}
}

func TestAssertAgentSmokeCaseRequiresTargetLocalContract(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := AgentSmokeCase{ID: "static-web-empty", RequiredArtifacts: []string{"index.html"}}
	err := assertAgentSmokeCase(dir, c)
	if err == nil || !strings.Contains(err.Error(), "target-local smoke contract") {
		t.Fatalf("expected missing target-local contract error, got %v", err)
	}
	contractPath := filepath.Join(dir, filepath.FromSlash(agentSmokeCaseContractPath))
	if err := os.MkdirAll(filepath.Dir(contractPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contractPath, []byte("contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := assertAgentSmokeCase(dir, c); err != nil {
		t.Fatalf("expected contract-backed case assertion to pass: %v", err)
	}
}

func TestAssertAgentSmokeCaseBeforeSkipsRoleProducedArtifacts(t *testing.T) {
	dir := t.TempDir()
	contractPath := filepath.Join(dir, filepath.FromSlash(agentSmokeCaseContractPath))
	if err := os.MkdirAll(filepath.Dir(contractPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contractPath, []byte("contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := AgentSmokeCase{
		ID:                "static-web-after-engineer",
		Role:              "qa",
		RequiredArtifacts: []string{"docs/reports/qa/static-web-after-engineer.md"},
	}
	if err := assertAgentSmokeCaseBefore(dir, c); err != nil {
		t.Fatalf("expected pre-run assertion to skip QA-produced report: %v", err)
	}
	err := assertAgentSmokeCase(dir, c)
	if err == nil || !strings.Contains(err.Error(), "required artifact missing") {
		t.Fatalf("expected post-run assertion to require QA report, got %v", err)
	}
}

func TestAssertAgentSmokeCaseAfterRequiresLifecycleAndReviewerEvidence(t *testing.T) {
	write := func(t *testing.T, root, rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	engineerDir := t.TempDir()
	write(t, engineerDir, agentSmokeCaseContractPath, "contract\n")
	write(t, engineerDir, "docs/tickets/done/T-001.md", "# Done\n")
	if err := os.MkdirAll(filepath.Join(engineerDir, "docs", "tickets", "in-progress"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := assertAgentSmokeCaseAfter(engineerDir, AgentSmokeCase{ID: "go-api-ticket", Role: "engineer"}); err != nil {
		t.Fatalf("expected completed engineer ticket lifecycle to pass: %v", err)
	}
	write(t, engineerDir, "docs/tickets/in-progress/T-002.md", "# Still active\n")
	err := assertAgentSmokeCaseAfter(engineerDir, AgentSmokeCase{ID: "go-api-ticket", Role: "engineer"})
	if err == nil || !strings.Contains(err.Error(), "in-progress tickets remain") {
		t.Fatalf("expected in-progress ticket rejection, got %v", err)
	}

	qaDir := t.TempDir()
	write(t, qaDir, agentSmokeCaseContractPath, "contract\n")
	write(t, qaDir, "docs/reports/qa/go-cli-missing-evidence-heldout.md", "# QA\n\nLooks bad.\n")
	err = assertAgentSmokeCaseAfter(qaDir, AgentSmokeCase{ID: "go-cli-missing-evidence-heldout", Role: "qa", ExpectedDisposition: "changes_requested"})
	if err == nil || !strings.Contains(err.Error(), "seeded gap") {
		t.Fatalf("expected QA seeded gap reference rejection, got %v", err)
	}
	write(t, qaDir, "docs/reports/qa/go-cli-missing-evidence-heldout.md", "# QA\n\nReferences seeded-gap-go-cli-missing-evidence-heldout.md.\n")
	if err := assertAgentSmokeCaseAfter(qaDir, AgentSmokeCase{ID: "go-cli-missing-evidence-heldout", Role: "qa", ExpectedDisposition: "changes_requested"}); err != nil {
		t.Fatalf("expected QA seeded gap reference to pass: %v", err)
	}

	dogfoodDir := t.TempDir()
	write(t, dogfoodDir, agentSmokeCaseContractPath, "contract\n")
	write(t, dogfoodDir, "docs/reports/dogfood/go-cli-defect-heldout.md", "# Dogfood\n\nReferences seeded-defect.md.\n")
	for _, status := range []string{"backlog", "in-progress", "in-review"} {
		if err := os.MkdirAll(filepath.Join(dogfoodDir, "docs", "tickets", status), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	err = assertAgentSmokeCaseAfter(dogfoodDir, AgentSmokeCase{ID: "go-cli-defect-heldout", Role: "dogfood", ExpectedDisposition: "changes_requested"})
	if err == nil || !strings.Contains(err.Error(), "finding ticket missing") {
		t.Fatalf("expected missing dogfood finding ticket rejection, got %v", err)
	}
	write(t, dogfoodDir, "docs/tickets/backlog/T-002.md", "# Finding\n\nReferences seeded-defect.md.\n")
	if err := assertAgentSmokeCaseAfter(dogfoodDir, AgentSmokeCase{ID: "go-cli-defect-heldout", Role: "dogfood", ExpectedDisposition: "changes_requested"}); err != nil {
		t.Fatalf("expected dogfood finding ticket evidence to pass: %v", err)
	}
}

func TestAssertAgentSmokeCaseAfterRequiresRoleEvidence(t *testing.T) {
	dir := t.TempDir()
	contractPath := filepath.Join(dir, filepath.FromSlash(agentSmokeCaseContractPath))
	if err := os.MkdirAll(filepath.Dir(contractPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contractPath, []byte("contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checkPath := filepath.Join(dir, ".mars", "checks", "latest.json")
	if err := os.MkdirAll(filepath.Dir(checkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	pipeline := AgentSmokeCase{ID: "go-api-test-failure", Role: "pipeline-fixer", Stage: "failure"}
	if err := os.WriteFile(checkPath, []byte(`{"status":"failed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := assertAgentSmokeCaseAfter(dir, pipeline)
	if err == nil || !strings.Contains(err.Error(), "want passed") {
		t.Fatalf("expected failed check evidence to be rejected, got %v", err)
	}
	if err := os.WriteFile(checkPath, []byte(`{"status":"passed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := assertAgentSmokeCaseAfter(dir, pipeline); err != nil {
		t.Fatalf("expected passed check evidence to be accepted: %v", err)
	}
	blockedPipeline := AgentSmokeCase{ID: "foundation-runtime-heldout", Role: "pipeline-fixer", Stage: "failure", ExpectedDisposition: "blocked"}
	if err := os.WriteFile(checkPath, []byte(`{"status":"failed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := assertAgentSmokeCaseAfter(dir, blockedPipeline); err != nil {
		t.Fatalf("expected blocked pipeline evidence to retain failed state: %v", err)
	}
	if err := os.WriteFile(checkPath, []byte(`{"status":"passed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err = assertAgentSmokeCaseAfter(dir, blockedPipeline)
	if err == nil || !strings.Contains(err.Error(), "rewrote check evidence to passed") {
		t.Fatalf("expected blocked pipeline pass rewrite to be rejected, got %v", err)
	}

	releaseDir := t.TempDir()
	releaseContract := filepath.Join(releaseDir, filepath.FromSlash(agentSmokeCaseContractPath))
	if err := os.MkdirAll(filepath.Dir(releaseContract), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(releaseContract, []byte("contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "VERSION"), []byte("0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "CHANGELOG.md"), []byte("# Changelog\n\n## 0.1.0\n\n- Release smoke evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	releaseReport := filepath.Join(releaseDir, "docs", "reports", "release", "release-go-api-ready.md")
	if err := os.MkdirAll(filepath.Dir(releaseReport), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(releaseReport, []byte("# Release Evidence\n\nLocal tag and notes ready.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	_ = runCommand(context.Background(), releaseDir, "git", "config", "user.email", "test@example.invalid")
	_ = runCommand(context.Background(), releaseDir, "git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(releaseDir, "README.md"), []byte("# release\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "commit", "-m", "seed"); err != nil {
		t.Fatal(err)
	}
	release := AgentSmokeCase{ID: "release-go-api-ready", Role: "release-manager", Stage: "ready"}
	err = assertAgentSmokeCaseAfter(releaseDir, release)
	if err == nil || !strings.Contains(err.Error(), "local tag missing") {
		t.Fatalf("expected missing release tag to be rejected, got %v", err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "tag", "v0.1.0"); err != nil {
		t.Fatal(err)
	}
	if err := assertAgentSmokeCaseAfter(releaseDir, release); err != nil {
		t.Fatalf("expected release tag evidence to be accepted: %v", err)
	}
	learningsPath := filepath.Join(releaseDir, ".harness", "learnings.yaml")
	if err := os.MkdirAll(filepath.Dir(learningsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(learningsPath, []byte("entries:\n- role: release-manager\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "add", ".harness/learnings.yaml"); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "commit", "-m", "chore(learnings): update runtime learnings for release-manager"); err != nil {
		t.Fatal(err)
	}
	if err := assertAgentSmokeCaseAfter(releaseDir, release); err != nil {
		t.Fatalf("expected release tag followed by runtime learnings to be accepted: %v", err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "README.md"), []byte("# release\n\npost-tag product drift\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(context.Background(), releaseDir, "git", "commit", "-m", "docs: post tag drift"); err != nil {
		t.Fatal(err)
	}
	err = assertAgentSmokeCaseAfter(releaseDir, release)
	if err == nil || !strings.Contains(err.Error(), "runtime-learnings-only tail") {
		t.Fatalf("expected post-tag product drift to be rejected, got %v", err)
	}
}

func TestRoleSmokeCaseInstructionsMatchRoleContracts(t *testing.T) {
	cto := roleSmokeCaseInstructions(AgentSmokeCase{Role: "cto-weekly", ExpectedDisposition: "completed"})
	for _, want := range []string{`"bdd_scenarios":["F-001-S001"]`, "not a quoted string", "backlog is empty", "do not loop on `git_diff`", "immediately call `git_commit`"} {
		if !strings.Contains(cto, want) {
			t.Fatalf("expected CTO contract to contain %q:\n%s", want, cto)
		}
	}
	dogfood := roleSmokeCaseInstructions(AgentSmokeCase{Role: "dogfood", ProjectType: "static-web", ID: "dogfood-static-web-ready", ExpectedDisposition: "approved"})
	for _, want := range []string{"python3 -m http.server", "HTTP 200", "do not call empty/no-op shell commands", "`approved`"} {
		if !strings.Contains(dogfood, want) {
			t.Fatalf("expected dogfood contract to contain %q:\n%s", want, dogfood)
		}
	}
	docsReleaseContract := roleSmokeCaseInstructions(AgentSmokeCase{Role: "release-manager", ProjectType: "docs-site", ID: "docs-site-notes-only-heldout", Stage: "ready", ExpectedDisposition: "completed"})
	for _, want := range []string{"Write `docs/reports/release/docs-site-notes-only-heldout.md` before committing release notes", "commit VERSION, CHANGELOG, the release report", "`release: notes <VERSION>`", "Create the local tag `v<VERSION>` at that release-note HEAD"} {
		if !strings.Contains(docsReleaseContract, want) {
			t.Fatalf("expected release-manager contract to contain %q:\n%s", want, docsReleaseContract)
		}
	}
	phaserDogfood := caseContractForCase(AgentSmokeCase{Role: "dogfood", ProjectType: "browser-game-phaser", ID: "dogfood-browser-game-ready", ExpectedDisposition: "approved"})
	for _, want := range []string{"npm run build", "browser smoke: Phaser canvas #game new Phaser.Game", "index.html", "src/main.js", "commit the generated `package-lock.json` as validation provenance"} {
		if !strings.Contains(phaserDogfood, want) {
			t.Fatalf("expected Phaser dogfood contract to contain %q:\n%s", want, phaserDogfood)
		}
	}
	reactDogfood := caseContractForCase(AgentSmokeCase{Role: "dogfood", ProjectType: "react-web", ID: "dogfood-react-web-ready", ExpectedDisposition: "approved"})
	for _, want := range []string{"dependency_sync", `"frozen":false`, "npm run build", "browser smoke: React document.querySelector #game score UI state", "commit the generated `package-lock.json` as validation provenance"} {
		if !strings.Contains(reactDogfood, want) {
			t.Fatalf("expected React dogfood contract to contain %q:\n%s", want, reactDogfood)
		}
	}
	defectDogfood := roleSmokeCaseInstructions(AgentSmokeCase{Role: "dogfood", ProjectType: "go-cli", ID: "go-cli-defect-heldout", ExpectedDisposition: "changes_requested"})
	for _, want := range []string{"intentionally contains", "seeded-defect.md", "Passing unit tests do not clear", "Do not approve", "ticket_create", "`changes_requested`"} {
		if !strings.Contains(defectDogfood, want) {
			t.Fatalf("expected defect dogfood contract to contain %q:\n%s", want, defectDogfood)
		}
	}
	dogfoodGoAPI := caseContractForCase(AgentSmokeCase{Role: "dogfood", ProjectType: "go-api", ID: "dogfood-go-api-ready", ExpectedDisposition: "approved", Trigger: map[string]string{"next_need": "release_review", "suggested_role": "release-manager"}})
	for _, want := range []string{"`go test ./...` as the bounded user smoke", "Do not run `go build ./...`", "do not issue empty argv/no-op shell commands", "docs/reports/dogfood/dogfood-go-api-ready.md", "`approved`"} {
		if !strings.Contains(dogfoodGoAPI, want) {
			t.Fatalf("expected dogfood Go API contract to contain %q:\n%s", want, dogfoodGoAPI)
		}
	}
	qa := roleSmokeCaseInstructions(AgentSmokeCase{Role: "qa", ProjectType: "static-web", ID: "static-web-after-engineer", ExpectedDisposition: "approved"})
	for _, want := range []string{"node --check app.js", "python3 -m http.server", "Do not use `python`", "`git_status`", "`git_commit`", "policy-blocked while the report is uncommitted", "`approved`"} {
		if !strings.Contains(qa, want) {
			t.Fatalf("expected QA contract to contain %q:\n%s", want, qa)
		}
	}
	negativeQA := roleSmokeCaseInstructions(AgentSmokeCase{Role: "qa", ProjectType: "go-cli", ID: "go-cli-missing-evidence-heldout", ExpectedDisposition: "changes_requested"})
	for _, want := range []string{"intentionally lacks acceptable QA evidence", "`git_status`", "`git_commit`", "`changes_requested`", "Do not approve"} {
		if !strings.Contains(negativeQA, want) {
			t.Fatalf("expected negative QA contract to contain %q:\n%s", want, negativeQA)
		}
	}
	security := roleSmokeCaseInstructions(AgentSmokeCase{Role: "security", ProjectType: "go-api", ID: "go-api-after-qa", ExpectedDisposition: "approved"})
	for _, want := range []string{"docs/reports/security/go-api-after-qa.md", "generic dated `docs/reports/security/security-audit-[date].md`", "`git_status`", "`git_commit`", "`approved`"} {
		if !strings.Contains(security, want) {
			t.Fatalf("expected security contract to contain %q:\n%s", want, security)
		}
	}
	engineerCanvas := caseContractForCase(AgentSmokeCase{Role: "engineer", ProjectType: "canvas-game-vanilla", ID: "canvas-game-smoke-gap-heldout", Stage: "gap", ExpectedDisposition: "completed", WouldDispatch: "qa", Trigger: map[string]string{"next_need": "qa_review", "suggested_role": "qa"}})
	for _, want := range []string{
		"`docs/tickets/in-progress/T-001-exercise-canvas-game-vanilla-smoke-case-canvas-game-smoke-ga.md`",
		"`node tests/game-state.test.js`",
		"`evidence_links`",
		`["git","mv","docs/tickets/in-progress/T-001-exercise-canvas-game-vanilla-smoke-case-canvas-game-smoke-ga.md","docs/tickets/done/T-001-exercise-canvas-game-vanilla-smoke-case-canvas-game-smoke-ga.md"]`,
		`Final job_disposition_record MUST include status "completed", next_need "qa_review", and suggested_role "qa"`,
		"Never call `shell_exec` with empty argv",
	} {
		if !strings.Contains(engineerCanvas, want) {
			t.Fatalf("expected engineer canvas contract to contain %q:\n%s", want, engineerCanvas)
		}
	}
	engineerGoAPI := caseContractForCase(AgentSmokeCase{Role: "engineer", ProjectType: "go-api", ID: "go-api-ticket", Stage: "ticket", ExpectedDisposition: "completed", WouldDispatch: "qa", Trigger: map[string]string{"next_need": "qa_review", "suggested_role": "qa"}})
	for _, want := range []string{"Focused validation is `go test ./...`", "do not run `go build ./...`", "starting and ending with `go test ./...` when it passes", `suggested_role "qa"`} {
		if !strings.Contains(engineerGoAPI, want) {
			t.Fatalf("expected engineer go-api contract to contain %q:\n%s", want, engineerGoAPI)
		}
	}
	engineerPhaser := caseContractForCase(AgentSmokeCase{Role: "engineer", ProjectType: "browser-game-phaser", ID: "browser-game-ticket", Stage: "ticket", ExpectedDisposition: "completed", WouldDispatch: "qa", Trigger: map[string]string{"next_need": "qa_review", "suggested_role": "qa"}})
	for _, want := range []string{"`frozen:false`", "`npm run build`", "browser smoke: Phaser canvas #game new Phaser.Game", "without reading `dist/`", "do not read `dist/`"} {
		if !strings.Contains(engineerPhaser, want) {
			t.Fatalf("expected engineer Phaser contract to contain %q:\n%s", want, engineerPhaser)
		}
	}
	engineerReact := caseContractForCase(AgentSmokeCase{Role: "engineer", ProjectType: "react-web", ID: "react-web-ticket", Stage: "ticket", ExpectedDisposition: "completed", WouldDispatch: "qa", Trigger: map[string]string{"next_need": "qa_review", "suggested_role": "qa"}})
	for _, want := range []string{"`frozen:false`", "`npm run build`", "browser smoke: React document.querySelector #game score UI state", "do not read `dist/`"} {
		if !strings.Contains(engineerReact, want) {
			t.Fatalf("expected engineer React contract to contain %q:\n%s", want, engineerReact)
		}
	}
	janitor := caseContractForCase(AgentSmokeCase{Role: "janitor", ProjectType: "go-api", ID: "stale-api-ticket", Stage: "stale", ExpectedDisposition: "completed", WouldDispatch: "engineer", Trigger: map[string]string{"next_need": "implementation"}})
	for _, want := range []string{"Do not call `file_write` under `docs/tickets/`", `["mv","docs/tickets/in-progress/T-001-exercise-go-api-smoke-case-stale-api-ticket.md","docs/tickets/done/T-001-exercise-go-api-smoke-case-stale-api-ticket.md"]`, "suggested role `engineer`"} {
		if !strings.Contains(janitor, want) {
			t.Fatalf("expected janitor stale contract to contain %q:\n%s", want, janitor)
		}
	}
	release := roleSmokeCaseInstructions(AgentSmokeCase{Role: "release-manager", ExpectedDisposition: "completed"})
	for _, want := range []string{"missing GitHub remote", "not a blocker", "`v<VERSION>`", "Do not call `git_push`", "`completed`"} {
		if !strings.Contains(release, want) {
			t.Fatalf("expected release contract to contain %q:\n%s", want, release)
		}
	}
	blockedRelease := roleSmokeCaseInstructions(AgentSmokeCase{Role: "release-manager", ExpectedDisposition: "blocked", Trigger: map[string]string{"next_need": "no_need"}})
	for _, want := range []string{"blocked by design", "Do not run release notes", "`blocked`"} {
		if !strings.Contains(blockedRelease, want) {
			t.Fatalf("expected blocked release contract to contain %q:\n%s", want, blockedRelease)
		}
	}
	docsRelease := roleSmokeCaseInstructions(AgentSmokeCase{Role: "release-manager", ProjectType: "docs-site", ExpectedDisposition: "completed"})
	for _, want := range []string{"notes-only", "no build", "`v<VERSION>`"} {
		if !strings.Contains(docsRelease, want) {
			t.Fatalf("expected docs release contract to contain %q:\n%s", want, docsRelease)
		}
	}
	pipeline := roleSmokeCaseInstructions(AgentSmokeCase{Role: "pipeline-fixer", ExpectedDisposition: "completed"})
	for _, want := range []string{".mars/checks/latest.json", "file_write", `"status":"passed"`, "Do not keep searching for CI workflow files"} {
		if !strings.Contains(pipeline, want) {
			t.Fatalf("expected pipeline contract to contain %q:\n%s", want, pipeline)
		}
	}
	pipelineReact := caseContractForCase(AgentSmokeCase{Role: "pipeline-fixer", ProjectType: "react-web", ID: "dependency-ci-heldout", Stage: "failure", ExpectedDisposition: "completed", Trigger: map[string]string{"next_need": "dependency_maintenance"}})
	for _, want := range []string{
		"`dependency_sync`",
		"`npm run build`",
		"browser smoke: React document.querySelector #game score UI state",
		"do not run `go test ./...` in this non-Go target",
		`"npm run build","browser smoke: React document.querySelector #game score UI state"`,
	} {
		if !strings.Contains(pipelineReact, want) {
			t.Fatalf("expected pipeline React contract to contain %q:\n%s", want, pipelineReact)
		}
	}
	blockedPipelineContract := roleSmokeCaseInstructions(AgentSmokeCase{Role: "pipeline-fixer", ExpectedDisposition: "blocked", Trigger: map[string]string{"next_need": "operator_review"}})
	for _, want := range []string{"blocked by design", "Do not rewrite `.mars/checks/latest.json`", "`blocked`"} {
		if !strings.Contains(blockedPipelineContract, want) {
			t.Fatalf("expected blocked pipeline contract to contain %q:\n%s", want, blockedPipelineContract)
		}
	}
	blockedOrchestrator := roleSmokeCaseInstructions(AgentSmokeCase{Role: "orchestrator", ExpectedDisposition: "blocked", Trigger: map[string]string{"next_need": "operator_review"}})
	for _, want := range []string{"expects a stop", "Do not choose a next role", "`blocked`"} {
		if !strings.Contains(blockedOrchestrator, want) {
			t.Fatalf("expected blocked orchestrator contract to contain %q:\n%s", want, blockedOrchestrator)
		}
	}
}

func TestResolveAndValidateAgentSmokeErrors(t *testing.T) {
	if got := normalizeAgentSmokeOptions(AgentSmokeOptions{}).MaxTurns; got != agentSmokeDefaultMaxTurns {
		t.Fatalf("expected default max turns %d, got %d", agentSmokeDefaultMaxTurns, got)
	}
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
