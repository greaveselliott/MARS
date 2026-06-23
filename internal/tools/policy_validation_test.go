/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
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
	"strings"
	"testing"
)

func TestEngineerPostValidationCommitBlocksExploratoryShellUntilTicketDone(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module note-stats\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement ticket"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:git_commit:success":   1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["ls","-la","/tmp/"]}`))
	if err == nil {
		t.Fatal("expected post-validation exploratory shell to be blocked")
	}
	if !strings.Contains(err.Error(), "successful validation and a clean implementation commit") ||
		!strings.Contains(err.Error(), "file_read") ||
		!strings.Contains(err.Error(), "file_write") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected ticket completion guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/in-progress/T-001-ship.md","docs/tickets/done/"]}`))
	if err != nil {
		t.Fatalf("expected ticket lifecycle move to remain allowed, got %v", err)
	}
}

func TestEngineerPostValidationAllowsFreshExternalValidationArtifact(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go build -o /tmp/note-stats-validation ./cmd/note-stats
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): reopen T-001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:                                1,
		"tool:git_commit:success":                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"): 1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text","hello world"]}`))
	if err != nil {
		t.Fatalf("expected fresh validation artifact execution to pass, got %v", err)
	}
}

func TestEngineerCannotCompleteTicketWithUnresolvedRuntimeValidationFailure(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- /tmp/note-stats-validation --text "hello world"
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): claim T-001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:               1,
		unexpectedRuntimeValidationOutstandingKey: 1,
		"tool:git_commit:success":                 1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["git","mv","docs/tickets/in-progress/T-001-ship.md","docs/tickets/done/"]}`))
	if err == nil {
		t.Fatal("expected unresolved runtime validation failure to block done move")
	}
	if !strings.Contains(err.Error(), "unexpected runtime validation failure is unresolved") ||
		!strings.Contains(err.Error(), "rerun the exact failing command successfully") ||
		!strings.Contains(err.Error(), "missing-required-input") ||
		!strings.Contains(err.Error(), "expected_exit_code") {
		t.Fatalf("expected runtime validation repair guidance, got %v", err)
	}

	if err := runGitExit0(context.Background(), root, "mv", "docs/tickets/in-progress/T-001-ship.md", "docs/tickets/done/"); err != nil {
		t.Fatalf("stage done move: %v", err)
	}
	err = preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"chore(tickets): move T-001 to done"}`))
	if err == nil {
		t.Fatal("expected unresolved runtime validation failure to block done commit")
	}

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"in_review","ticket_id":"T-001","next_need":"qa_review"}`))
	if err == nil {
		t.Fatal("expected unresolved runtime validation failure to block successful disposition")
	}
}

func TestExternalValidationArtifactMustBeBuiltInSameSession(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- /tmp/note-stats-validation --text hello
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): claim T-001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text","hello world"]}`))
	if err == nil {
		t.Fatal("expected stale external validation artifact to be blocked")
	}
	if !strings.Contains(err.Error(), "must be built in this role session") ||
		!strings.Contains(err.Error(), "shell_exec argv") {
		t.Fatalf("expected freshness guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), `shell_exec argv ["go","build","-o","/tmp/note-stats-validation","."]`) {
		t.Fatalf("expected exact build correction, got %v", err)
	}
}

func TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go run cmd/note-stats/main.go --text "hello world"
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement note stats"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	recordSessionToolOutcome(session, root, "shell_exec", json.RawMessage(`{"argv":["go","run","cmd/note-stats/main.go","--text","hello world"]}`), ToolResult{ExitCode: 0}, nil)
	session.ToolCounts["tool:git_commit:success"] = 1
	ctx := WithSession(context.Background(), *session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[]}`))
	if err == nil {
		t.Fatal("expected post-runtime-validation no-op to be redirected")
	}
	if !strings.Contains(err.Error(), "successful validation and a clean implementation commit") ||
		!strings.Contains(err.Error(), "file_read") ||
		!strings.Contains(err.Error(), "file_write") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected ticket completion guidance after runtime validation, got %v", err)
	}
}

func TestEngineerPostValidationDirtyNoopBlocksBeforeGenericNoop(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-002-repair-route.md", `---
id: T-002
title: Repair route registration
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- curl http://localhost:8080/health
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Repair route registration
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): claim T-002"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write dirty implementation: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:git_commit:success":   1,
	}})
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":[":"]}`))
	if err == nil {
		t.Fatal("expected post-validation no-op with dirty work to be blocked")
	}
	if !strings.Contains(err.Error(), "successful validation and dirty implementation or ticket work") ||
		!strings.Contains(err.Error(), "T-002") ||
		!strings.Contains(err.Error(), "main.go") ||
		!strings.Contains(err.Error(), "git_commit") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected dirty-work convergence guidance, got %v", err)
	}
}

func TestEngineerPostValidationGateAllowsValidationWhileImplementationDirty(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-ship.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios:
- F-001-S002
end_to_end_evidence: required
evidence_links:
- go test ./...
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship note stats
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore(tickets): claim T-001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`), 0o644); err != nil {
		t.Fatalf("write dirty implementation: %v", err)
	}

	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:git_commit:success":   1,
	}})
	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`)); err != nil {
		t.Fatalf("expected validation shell to pass while implementation is dirty, got %v", err)
	}
}

func TestReviewHTTPProbeBeforeServerStartIsProcedureFailure(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["curl","-f","http://localhost:5173/"]}`)
	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{
		ExitCode: 7,
		Stderr:   "curl: (7) Failed to connect to localhost port 5173 after 0 ms: Couldn't connect to server",
	}, nil)
	if session.ToolCounts[validationProcedureFailureKey] != 1 {
		t.Fatalf("expected HTTP probe connection failure to be a validation-procedure failure, got counts %#v", session.ToolCounts)
	}
	if session.ToolCounts[validationCommandFailureKey] != 0 {
		t.Fatalf("expected no product validation failure for pre-server curl, got counts %#v", session.ToolCounts)
	}

	ctx := WithSession(context.Background(), *session)
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["npm","run","dev"],"background":true}`))
	if err != nil {
		t.Fatalf("expected reviewer to recover by starting the dev server after pre-server curl, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksRepeatedRuntimeProbeUntilEdit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text",""]}`))
	if err == nil {
		t.Fatal("expected repeated runtime probe to be blocked until Engineer edits")
	}
	if !strings.Contains(err.Error(), "already failed unexpectedly") || !strings.Contains(err.Error(), "file_read/file_write") {
		t.Fatalf("expected edit-before-rerun guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureAllowsExactRerunAfterEdit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text",""]}`)); err != nil {
		t.Fatalf("expected exact runtime rerun after edit to be allowed, got %v", err)
	}
}

func TestExternalValidationArtifactMustBeRebuiltAfterRuntimeFailureEdit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                             1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs):          1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):                  0,
		runtimeValidationEditAfterFailureKey:                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):            1,
		validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation"): 0,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text",""]}`))
	if err == nil {
		t.Fatal("expected stale external validation artifact to be blocked")
	}
	if !strings.Contains(err.Error(), "built before a post-failure implementation edit") ||
		!strings.Contains(err.Error(), `shell_exec argv ["go","build","-o","/tmp/note-stats-validation","."]`) {
		t.Fatalf("expected rebuild guidance, got %v", err)
	}
}

func TestExternalValidationArtifactAllowsRerunAfterRuntimeFailureEditAndRebuild(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                             1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs):          1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):                  0,
		runtimeValidationEditAfterFailureKey:                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):            2,
		validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation"): 1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation","--text",""]}`)); err != nil {
		t.Fatalf("expected rebuilt external validation artifact rerun to pass, got %v", err)
	}
}

func TestRecordSessionToolOutcomeTreatsNodeCheckMissingFileAsProcedureFailure(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["node","--check","main.js"]}`)
	result := ToolResult{
		ExitCode: 1,
		Stderr:   "Error: Cannot find module '/tmp/demo/main.js'\n  code: 'MODULE_NOT_FOUND'",
	}

	recordSessionToolOutcome(session, root, "shell_exec", raw, result, nil)

	if session.ToolCounts[validationProcedureFailureKey] != 1 {
		t.Fatalf("expected validation procedure failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] != 0 {
		t.Fatalf("expected no unresolved runtime validation failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolState[validationProcedureFailureCommandKey] == "" {
		t.Fatalf("expected procedure failure command to be recorded, got state %+v", session.ToolState)
	}
}

func TestEngineerRuntimeFailureBlocksDifferentRuntimeProbe(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats"]}`))
	if err == nil {
		t.Fatal("expected different runtime probe to be blocked while exact repair is outstanding")
	}
	if !strings.Contains(err.Error(), "unresolved unexpected runtime validation failure from an earlier command") {
		t.Fatalf("expected exact-repair guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing-required-input") || !strings.Contains(err.Error(), "expected_exit_code") {
		t.Fatalf("expected missing-argument correction guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksShellWrapperBypass(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):   1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"cd /tmp && /tmp/note-stats-validation --text \"hello world\""}`))
	if err == nil {
		t.Fatal("expected shell wrapper runtime probe to be blocked while exact repair is outstanding")
	}
	if !strings.Contains(err.Error(), "shell wrappers") ||
		!strings.Contains(err.Error(), "rerun the exact failing command successfully") {
		t.Fatalf("expected shell-wrapper bypass guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksValidationUnrelatedShell(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	if err == nil {
		t.Fatal("expected unrelated validation shell command to be blocked while exact repair is outstanding")
	}
	if !strings.Contains(err.Error(), "Do not run shell_exec for other probes") ||
		!strings.Contains(err.Error(), "rerun the exact failing command successfully") {
		t.Fatalf("expected exact-repair guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureBlocksExpectedExitRerun(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text",""],"expected_exit_code":1}`))
	if err == nil {
		t.Fatal("expected Engineer expected-exit rerun to be blocked after unexpected runtime failure")
	}
	if !strings.Contains(err.Error(), "cannot use expected_exit_code") ||
		!strings.Contains(err.Error(), "missing-required-input") {
		t.Fatalf("expected expected-exit block guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureAllowsMissingArgumentExpectedExitCorrection(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureKey(failedArgs, 1):         1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):   1,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)); err != nil {
		t.Fatalf("expected Engineer missing-argument expected-exit correction to be allowed, got %v", err)
	}
}

func TestEngineerMissingArgumentRuntimeFailureBlocksUnrelatedMutation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			unexpectedRuntimeValidationOutstandingKey:                    1,
			unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		},
		ToolState: map[string]string{
			unexpectedRuntimeValidationMissingArgKey: "true",
			unexpectedRuntimeValidationCorrectionKey: `shell_exec {"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`,
		},
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"cmd/note-stats/main.go","content":"package main\n"}`))
	if err == nil {
		t.Fatal("expected unrelated mutation to be blocked until missing-argument correction")
	}
	if !strings.Contains(err.Error(), "Run the exact correction next") ||
		!strings.Contains(err.Error(), `"expected_exit_code":1`) ||
		!strings.Contains(err.Error(), "/tmp/note-stats-validation") {
		t.Fatalf("expected exact correction guidance, got %v", err)
	}
}

func TestEngineerMissingArgumentRuntimeFailureAllowsImplementationEditAfterCorrectionAttempt(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}
	correction := `shell_exec {"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`
	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			unexpectedRuntimeValidationOutstandingKey:                    1,
			unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		},
		ToolState: map[string]string{
			unexpectedRuntimeValidationMissingArgKey: "true",
			unexpectedRuntimeValidationCorrectionKey: correction,
			unexpectedRuntimeValidationAttemptedKey:  correction,
		},
	}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "file_write", []byte(`{"path":"cmd/note-stats/main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)); err != nil {
		t.Fatalf("expected implementation edit after failed missing-argument correction attempt, got %v", err)
	}

	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"fix cli missing input handling"}`))
	if err == nil {
		t.Fatal("expected commit to remain blocked until validation repairs the outstanding failure")
	}
	if !strings.Contains(err.Error(), "Run the exact correction next") ||
		!strings.Contains(err.Error(), `"expected_exit_code":1`) {
		t.Fatalf("expected commit to keep exact validation guidance, got %v", err)
	}
}

func TestEngineerPositiveRuntimeFailureBlocksImplementationCommit(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                    1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs): 1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):         0,
		runtimeValidationEditAfterFailureKey:                         1,
	}}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"feat: implement note stats"}`))
	if err == nil {
		t.Fatal("expected implementation commit to be blocked while runtime acceptance failure is unresolved")
	}
	if !strings.Contains(err.Error(), "cannot commit product work") ||
		!strings.Contains(err.Error(), "Keep the failed implementation uncommitted") {
		t.Fatalf("expected commit block guidance, got %v", err)
	}
}

func TestEngineerRuntimeFailureAllowsStaleValidationArtifactRebuild(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		unexpectedRuntimeValidationOutstandingKey:                             1,
		unexpectedRuntimeValidationFailureFingerprintKey(failedArgs):          1,
		runtimeValidationFailureEditWatermarkKey(failedArgs):                  0,
		runtimeValidationEditAfterFailureKey:                                  1,
		validationArtifactSessionKey("/tmp/note-stats-validation"):            1,
		validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation"): 0,
	}}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)); err != nil {
		t.Fatalf("expected stale validation artifact rebuild to remain available, got %v", err)
	}
}

func TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats/..."}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./cmd/note-stats/..."]}`,
		testBuildValidationScopeKey:   "cmd/note-stats/",
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","run","./cmd/note-stats","--text","hello"]}`))
	if err == nil {
		t.Fatal("expected runtime probe to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "failing test or build command") ||
		!strings.Contains(err.Error(), "test/build command") {
		t.Fatalf("expected test/build repair guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./cmd/note-stats"]}`))
	if err == nil {
		t.Fatal("expected unchanged same-lane test rerun to be blocked")
	}
	if !strings.Contains(err.Error(), "latest repair edit") ||
		!strings.Contains(err.Error(), "file_read/file_write") {
		t.Fatalf("expected edit-before-rerun guidance, got %v", err)
	}

	session.ToolCounts[testBuildValidationEditAfterFailureKey] = 1
	ctx = WithSession(context.Background(), session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","./cmd/note-stats"]}`))
	if err == nil {
		t.Fatal("expected build command to be blocked while test failure is unresolved")
	}
	if !strings.Contains(err.Error(), "unresolved test failure") {
		t.Fatalf("expected same-lane guidance, got %v", err)
	}

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./cmd/note-stats"]}`)); err != nil {
		t.Fatalf("expected focused same-lane test rerun after source edit to be allowed, got %v", err)
	}
	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"cd cmd/note-stats && go test -v ."}`)); err != nil {
		t.Fatalf("expected simple cd plus same-lane test shell command to be allowed, got %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", []byte(`{"path":"verify_functionality.sh","content":"#!/bin/sh\necho ok\n"}`))
	if err == nil {
		t.Fatal("expected helper script write to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "helper scripts") {
		t.Fatalf("expected helper script guidance, got %v", err)
	}

	sourceRaw := []byte(`{"path":"cmd/note-stats/main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	if err := preToolPolicy(ctx, root, "file_write", sourceRaw); err != nil {
		t.Fatalf("expected source repair write to be allowed, got %v", err)
	}

	rootSourceRaw := []byte(`{"path":"main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	err = preToolPolicy(ctx, root, "file_write", rootSourceRaw)
	if err == nil {
		t.Fatal("expected alternate root source write to be blocked while package test is unresolved")
	}
	if !strings.Contains(err.Error(), "failed test/build scope") && !strings.Contains(err.Error(), "alternate entrypoints") {
		t.Fatalf("expected failed-scope guidance, got %v", err)
	}
}

func TestEngineerFailingTestAllowsSameJobRepairTestFileRemoval(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main_test.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "old_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write old_test.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats"}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey:                              `shell_exec {"argv":["go","test","./cmd/note-stats"]}`,
		testBuildValidationOutputKey:                               "main_test.go: helper redeclared in this block\nold_test.go: other declaration of helper",
		testBuildRepairWritePathKey("cmd/note-stats/main_test.go"): "true",
		testBuildRepairWritePathKey("cmd/note-stats/main.go"):      "true",
	}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main_test.go"]}`)); err != nil {
		t.Fatalf("expected same-job repair test file removal to be allowed, got %v", err)
	}

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/old_test.go"]}`))
	if err == nil {
		t.Fatal("expected unmarked test file removal to remain blocked")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected failing-test repair-lane guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main.go"]}`))
	if err == nil {
		t.Fatal("expected source removal to remain blocked even when the source was rewritten during repair")
	}
}

func TestEngineerFailingTestAllowsMissingGoModuleBootstrap(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-notes-api.md", "# T-001\n")

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./internal/note"}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./internal/note"]}`,
		testBuildValidationOutputKey:  "go: cannot find main module, but found .git/config in /tmp/demo-notes-api\n\tto create a module there, run:\n\tgo mod init",
		testBuildValidationScopeKey:   "internal/note/",
	}
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","demo-notes-api"]}`)); err != nil {
		t.Fatalf("expected missing Go module bootstrap to be allowed, got %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo-notes-api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","demo-notes-api"]}`))
	if err == nil {
		t.Fatal("expected go mod init to be blocked once go.mod exists")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected unresolved failure guidance, got %v", err)
	}

	if err := os.Remove(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("remove go.mod: %v", err)
	}
	session.ToolState[testBuildValidationOutputKey] = "FAIL: TestCreateNote expected title"
	ctx = WithSession(context.Background(), session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","mod","init","demo-notes-api"]}`))
	if err == nil {
		t.Fatal("expected go mod init to stay blocked when failure output is not missing-module evidence")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected unresolved failure guidance, got %v", err)
	}
}

func TestEngineerFailingTestAllowsRemovalOfTestFileWrittenBeforeFailure(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "old_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write old_test.go: %v", err)
	}

	session := Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	writtenTestRaw := []byte(`{"path":"cmd/note-stats/main_test.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	recordSessionToolOutcome(&session, root, "file_write", writtenTestRaw, ToolResult{}, nil)
	writtenSourceRaw := []byte(`{"path":"cmd/note-stats/main.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	recordSessionToolOutcome(&session, root, "file_write", writtenSourceRaw, ToolResult{}, nil)

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats"}}
	session.ToolCounts[testBuildValidationOutstandingKey] = 1
	session.ToolCounts[testCommandFailureKey] = 1
	session.ToolCounts[testBuildValidationFailureFingerprintKey(failedArgs)] = 1
	session.ToolCounts[testBuildValidationFailureEditWatermarkKey(failedArgs)] = 0
	session.ToolState[testBuildValidationCommandKey] = `shell_exec {"argv":["go","test","./cmd/note-stats"]}`
	session.ToolState[testBuildValidationOutputKey] = "main_test.go: helper redeclared in this block\nold_test.go: other declaration of helper"
	ctx := WithSession(context.Background(), session)

	if err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"shell_command":"rm -f cmd/note-stats/main_test.go"}`)); err != nil {
		t.Fatalf("expected same-job pre-failure test file removal to be allowed, got %v", err)
	}

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/old_test.go"]}`))
	if err == nil {
		t.Fatal("expected pre-existing test file removal to remain blocked")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected failing-test repair-lane guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main.go"]}`))
	if err == nil {
		t.Fatal("expected source removal to remain blocked even when the source was written earlier in the job")
	}
}

func TestEngineerFailingTestBlocksSameJobTestRemovalForAssertionFailure(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", "# T-001\n")

	session := Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	writtenTestRaw := []byte(`{"path":"cmd/note-stats/main_test.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\n"}`)
	recordSessionToolOutcome(&session, root, "file_write", writtenTestRaw, ToolResult{}, nil)

	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats"}}
	session.ToolCounts[testBuildValidationOutstandingKey] = 1
	session.ToolCounts[testCommandFailureKey] = 1
	session.ToolCounts[testBuildValidationFailureFingerprintKey(failedArgs)] = 1
	session.ToolCounts[testBuildValidationFailureEditWatermarkKey(failedArgs)] = 0
	session.ToolState[testBuildValidationCommandKey] = `shell_exec {"argv":["go","test","./cmd/note-stats"]}`
	session.ToolState[testBuildValidationOutputKey] = "main_test.go:42: expected 3 items, got 2"
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["rm","cmd/note-stats/main_test.go"]}`))
	if err == nil {
		t.Fatal("expected assertion-failure test removal to remain blocked")
	}
	if !strings.Contains(err.Error(), "failing test or build command") {
		t.Fatalf("expected failing-test repair-lane guidance, got %v", err)
	}
}

func TestEngineerFailingBuildGuidanceCompactsRepeatedOutput(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	longOutput := strings.Repeat("vite build failed because Phaser browser runtime was imported from vite.config.js and window is not defined ", 20)
	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			testBuildValidationOutstandingKey: 1,
			buildCommandFailureKey:            1,
		},
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["npm","run","build"]}`,
			testBuildValidationOutputKey:  longOutput,
		},
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["node","scripts/probe.js"]}`))
	if err == nil {
		t.Fatal("expected unresolved build failure to block unrelated shell_exec")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Latest failing output (compact):") || !strings.Contains(msg, "npm") {
		t.Fatalf("expected compact unresolved guidance, got %v", err)
	}
	if strings.Count(msg, "window is not defined") > 1 || len(msg) > 950 {
		t.Fatalf("expected compact repeated-output guidance, len=%d msg=%v", len(msg), msg)
	}
}

func TestEngineerFailingTestAllowsIntegrationTestRepairWrite(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-playfield.md", "# T-001\n")

	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			testBuildValidationOutstandingKey: 1,
			testCommandFailureKey:             1,
		},
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["npm","run","test:integration"]}`,
			testBuildValidationOutputKey:  "jest-environment-jsdom cannot be found",
		},
	}
	ctx := WithSession(context.Background(), session)
	raw := []byte(`{"path":"tests/integration/playfield.test.js","content":"import { describe, expect, test } from '@jest/globals';\n\ndescribe('playfield', () => {\n  test('renders', () => {\n    expect(true).toBe(true);\n  });\n});\n"}`)

	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected same-lane integration test repair write to be allowed, got %v", err)
	}
}

func TestEngineerFailingGoIntegrationTestAllowsProductSourceRepairWrite(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-api.md", "# T-001\n")
	if err := os.MkdirAll(filepath.Join(dir, "internal", "api"), 0o755); err != nil {
		t.Fatalf("mkdir internal/api: %v", err)
	}

	session := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			testBuildValidationOutstandingKey: 1,
			testCommandFailureKey:             1,
		},
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["go","test","-v","./tests/integration"]}`,
			testBuildValidationScopeKey:   "tests/integration",
			testBuildValidationOutputKey:  "tests/integration/tasknotesapi_test.go:91:16: handler.AddTaskNote undefined",
		},
	}
	ctx := WithSession(context.Background(), session)
	raw := []byte(`{"path":"internal/api/handlers.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage api\n"}`)

	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected integration test source repair write to be allowed, got %v", err)
	}
}

func TestEngineerFailingTestBlocksCommitTicketEvidenceAndDisposition(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-note-stats.md", `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship note stats
`)
	failedArgs := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats/..."}}
	session := Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:                            1,
		testBuildValidationOutstandingKey:                      1,
		testCommandFailureKey:                                  1,
		testBuildValidationFailureFingerprintKey(failedArgs):   1,
		testBuildValidationFailureEditWatermarkKey(failedArgs): 0,
	}}
	session.ToolState = map[string]string{
		testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./cmd/note-stats/..."]}`,
	}
	ctx := WithSession(context.Background(), session)

	err := preToolPolicy(ctx, root, "git_commit", []byte(`{"message":"feat: implement note stats"}`))
	if err == nil {
		t.Fatal("expected product commit to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "cannot commit product work") ||
		!strings.Contains(err.Error(), "exact unresolved command") {
		t.Fatalf("expected commit block guidance, got %v", err)
	}

	ticketContent := `---
id: T-001
title: Ship note stats
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links:
- go test ./cmd/note-stats/...
verified_by:
- engineer
---

# Ship note stats
`
	raw, marshalErr := json.Marshal(fileWriteArgs{
		Path:    filepath.Join("docs", "tickets", "in-progress", "T-001-note-stats.md"),
		Content: ticketContent,
	})
	if marshalErr != nil {
		t.Fatalf("marshal file_write: %v", marshalErr)
	}
	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected ticket evidence write to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "update ticket evidence") ||
		!strings.Contains(err.Error(), "failing test or build") {
		t.Fatalf("expected ticket evidence block guidance, got %v", err)
	}

	sourceRaw, marshalErr := json.Marshal(fileWriteArgs{
		Path: "cmd/note-stats/main.go",
		Content: `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main
`,
	})
	if marshalErr != nil {
		t.Fatalf("marshal source file_write: %v", marshalErr)
	}
	if err := preToolPolicy(ctx, root, "file_write", sourceRaw); err != nil {
		t.Fatalf("expected source repair write to remain available, got %v", err)
	}

	err = preToolPolicy(ctx, root, "job_disposition_record", []byte(`{"status":"completed","ticket_id":"T-001","next_need":"qa_review"}`))
	if err == nil {
		t.Fatal("expected successful disposition to be blocked while failing tests are unresolved")
	}
	if !strings.Contains(err.Error(), "record a successful product disposition") ||
		!strings.Contains(err.Error(), "failing test or build") {
		t.Fatalf("expected disposition block guidance, got %v", err)
	}
}
