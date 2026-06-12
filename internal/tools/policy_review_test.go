/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewApprovalRequiresPassingValidationWhenTestsExist(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	testPath := filepath.Join(dir, "cmd", "note-stats", "main_test.go")
	if err := os.MkdirAll(filepath.Dir(testPath), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(testPath, []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	raw := []byte(`{"status":"approved","ticket_id":"T-001","next_need":"security_review"}`)

	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA approval without tests to be blocked")
	}
	if !strings.Contains(err.Error(), "must run the repository's authoritative test command") {
		t.Fatalf("expected test command guidance, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "security", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandFailureKey:       1,
	}})
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected Security approval after failing tests to be blocked")
	}
	if !strings.Contains(err.Error(), "after a failing build or test command") {
		t.Fatalf("expected failing validation guidance, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:      1,
		expectedRuntimeFailureSuccessKey: 1,
		testCommandSuccessKey:            1,
	}})
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected QA approval after passing tests and an expected runtime error probe to pass, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		validationCommandFailureKey: 1,
		testCommandSuccessKey:       1,
	}})
	err = preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA approval after an unexpected runtime validation failure to be blocked")
	}
	if !strings.Contains(err.Error(), "unexpected failing validation command") {
		t.Fatalf("expected unexpected runtime validation guidance, got %v", err)
	}

	ctx = WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandSuccessKey:       1,
	}})
	if err := preToolPolicy(ctx, root, "job_disposition_record", raw); err != nil {
		t.Fatalf("expected QA approval after passing tests to pass, got %v", err)
	}
}

func TestReviewValidationFailureBlocksFurtherShellBeforeDisposition(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		testCommandFailureKey:       1,
		validationCommandFailureKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected further QA shell validation to be blocked after validation failure")
	}
	if !strings.Contains(err.Error(), "already observed a failing build, test, or unexpected runtime validation command") ||
		!strings.Contains(err.Error(), "job_disposition_record") ||
		!strings.Contains(err.Error(), "changes_requested") {
		t.Fatalf("expected terminal changes_requested guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"changes_requested","ticket_id":"T-001","next_need":"implementation_rework"}`)); err != nil {
		t.Fatalf("expected changes_requested disposition to remain available after validation failure, got %v", err)
	}
}

func TestReviewValidationFailureAllowsExactExpectedExitCorrection(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	session := Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey:                                1,
		testCommandSuccessKey:                                      1,
		validationCommandSuccessKey:                                1,
		validationArtifactSessionKey("/tmp/note-stats-validation"): 1,
		unexpectedRuntimeValidationFailureKey(failedArgs, 1):       1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)); err != nil {
		t.Fatalf("expected exact expected-exit correction to be allowed, got %v", err)
	}

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected unrelated validation to remain blocked")
	}
	if !strings.Contains(err.Error(), "rerun that exact command once with shell_exec expected_exit_code") {
		t.Fatalf("expected exact-rerun guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyBlocksMutatingSetupCommands(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","note-stats-cli"]}`))
	if err == nil {
		t.Fatal("expected QA mutating setup command to be blocked")
	}
	if !strings.Contains(err.Error(), "qa shell_exec is validation-only") ||
		!strings.Contains(err.Error(), "go mod init note-stats-cli") {
		t.Fatalf("expected validation-only shell guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyAllowsTrackedBackgroundKill(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	t.Cleanup(KillBackgroundProcs)

	started, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","sleep 5"],"background":true}`))
	require.NoError(t, err)
	pid := backgroundPIDFromOutput(t, started.Output)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})

	raw := []byte(`{"argv":["kill","-TERM","` + strconv.Itoa(pid) + `"]}`)
	if err := preToolPolicy(ctx, root, "shell_exec", raw); err != nil {
		t.Fatalf("expected QA to stop tracked background PID, got %v", err)
	}
}

func TestReviewShellExecPolicyAllowsValidationCommands(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := Session{Role: "qa", ToolCounts: map[string]int{
		validationArtifactSessionKey("/tmp/note-stats-validation"): 1,
	}}
	ctx := WithSession(context.Background(), session)

	for _, raw := range [][]byte{
		[]byte(`{"argv":["go","test","./..."]}`),
		[]byte(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`),
		[]byte(`{"argv":["/tmp/note-stats-validation","--text","hello"]}`),
		[]byte(`{"argv":["curl","-fsS","http://127.0.0.1:8080/health"]}`),
	} {
		if err := preToolPolicy(ctx, root, "shell_exec", raw); err != nil {
			t.Fatalf("expected review validation command to pass for %s, got %v", string(raw), err)
		}
	}
}

func TestReviewShellExecPolicyBlocksNoopPlaceholders(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "cannot use shell_exec no-op placeholders") {
		t.Fatalf("expected no-op review guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesPostValidationNoopToDisposition(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		"tool:docsync_audit:success": 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "after successful validation") ||
		!strings.Contains(err.Error(), "job_disposition_record") ||
		!strings.Contains(err.Error(), "status approved") {
		t.Fatalf("expected successful-validation disposition guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesPostBuildNoopToTestsWhenTestsExist(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		buildCommandSuccessKey:      1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "authoritative test command") ||
		strings.Contains(err.Error(), "status approved") {
		t.Fatalf("expected missing-test guidance instead of approval guidance, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesNoTestGoRepoToChangesRequested(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	src := filepath.Join(dir, "cmd", "temperature-json-cli", "main.go")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "changes_requested") ||
		!strings.Contains(err.Error(), "no _test.go files") {
		t.Fatalf("expected missing-test changes_requested guidance, got %v", err)
	}
}

func TestReviewTerminalEvidenceWaitsForDocSyncAudit(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:file_read:success":    1,
	}}

	if ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence to wait for docsync_audit")
	}

	session.ToolCounts["tool:docsync_audit:success"] = 1
	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence after validation, read evidence, and docsync_audit")
	}
}

func TestReviewTerminalEvidenceIgnoresBackgroundServerStart(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}}
	raw := json.RawMessage(`{"argv":["python3","-m","http.server","8080","--bind","127.0.0.1"],"background":true}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	if ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence to wait for a concrete probe, not just background server startup")
	}

	probeRaw := json.RawMessage(`{"argv":["curl","-fsS","http://127.0.0.1:8080/"]}`)
	recordSessionToolOutcome(session, root, "shell_exec", probeRaw, ToolResult{ExitCode: 0}, nil)
	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence after static HTTP probe, read evidence, and docsync_audit")
	}
}

func TestReviewTerminalEvidenceWaitsForTestsWhenTestFilesExist(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}}

	if ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence to wait for test command success when test files exist")
	}

	session.ToolCounts[testCommandSuccessKey] = 1
	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected review terminal evidence after tests, validation, read evidence, and docsync_audit")
	}
}

func TestReviewTerminalDispositionRequiredBlocksFurtherShellExec(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:          1,
		reviewTerminalDispositionRequiredKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","./..."]}`))
	if err == nil {
		t.Fatal("expected further QA shell_exec to be blocked")
	}
	if !strings.Contains(err.Error(), "Do not call more tools except job_disposition_record") {
		t.Fatalf("expected terminal-only guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"approved","ticket_id":"T-001","next_need":"no_need"}`)); err != nil {
		t.Fatalf("expected terminal disposition to remain available, got %v", err)
	}
}

func TestReviewShellExecPolicyRoutesPostFailureNoopToChangesRequested(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey: 1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected QA no-op shell placeholder to be blocked")
	}
	if !strings.Contains(err.Error(), "already observed a failing build, test, or unexpected runtime validation command") ||
		!strings.Contains(err.Error(), "job_disposition_record") ||
		!strings.Contains(err.Error(), "changes_requested") {
		t.Fatalf("expected failing-validation disposition guidance, got %v", err)
	}
}

func TestQAChangesRequestedBlocksFoundationValidationIssueRoutedToEngineer(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	writeValidPhaserSource(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey:   1,
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
		"tool:file_read:success":      1,
		"tool:docsync_audit:success":  1,
	}})

	raw := json.RawMessage(`{
  "status":"changes_requested",
  "ticket_id":"T-001",
  "next_need":"implementation_rework",
  "reason":"Browser smoke test validation error; implementation is correct but the test should look for Phaser.Game differently.",
  "feedback":{"for_role":"engineer","summary":"Browser smoke test validation error","requested_change":"The implementation is correct; the test should be corrected."}
}`)
	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA rework routing to be blocked for foundation validation issue")
	}
	if !strings.Contains(err.Error(), "foundation validation/test wording issue") ||
		!strings.Contains(err.Error(), "Do not send implementation_rework") {
		t.Fatalf("expected foundation-validation routing guidance, got %v", err)
	}
}

func TestQAChangesRequestedBlocksDevServerSetupFailureRoutedToEngineer(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	writeValidPhaserSource(t, dir)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandFailureKey:  1,
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}})

	raw := json.RawMessage(`{
  "status":"changes_requested",
  "ticket_id":"T-001",
  "next_need":"implementation_rework",
  "reason":"Build succeeded but browser smoke test failed due to server not running. The curl test to localhost:5173 failed.",
  "feedback":{"for_role":"engineer","summary":"Build succeeded but smoke tests failed","requested_change":"Ensure the development server is running during testing and verify the browser smoke test works."}
}`)
	err := preToolPolicy(ctx, root, "job_disposition_record", raw)
	if err == nil {
		t.Fatal("expected QA rework routing to be blocked for QA-owned dev-server setup failure")
	}
	if !strings.Contains(err.Error(), "foundation validation/test wording issue") ||
		!strings.Contains(err.Error(), "Do not send implementation_rework") {
		t.Fatalf("expected foundation-validation routing guidance, got %v", err)
	}
}

func TestQAApprovalRequiresGoTestsForGoSource(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	sourcePath := filepath.Join(dir, "cmd", "note-stats", "main.go")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main

func main() {}
`), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		testCommandSuccessKey:       1,
	}})
	err := preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"approved","ticket_id":"T-001","next_need":"security_review"}`))
	if err == nil {
		t.Fatal("expected QA approval without Go tests to be blocked")
	}
	if !strings.Contains(err.Error(), "Go source files exist but no _test.go files are present") {
		t.Fatalf("expected Go test coverage guidance, got %v", err)
	}
}
