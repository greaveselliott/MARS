/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/safety"
	"github.com/stretchr/testify/require"
)

func TestShellExec_argv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","echo hello"],"timeout_seconds":5}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Contains(t, strings.TrimSpace(res.Output), "hello")
}

func TestShellExecArgvAllowsLiteralNewlineArgument(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":["printf","%s","hello\nworld"],"timeout_seconds":5}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Equal(t, "hello\nworld", res.Output)
}

func TestShellExecArgvAllowsNodeEvalCodeArgument(t *testing.T) {
	t.Parallel()

	err := validateShellExecArgv([]string{
		"node",
		"-e",
		"import('./src/main.js'); console.log('browser smoke: Phaser canvas #game new Phaser.Game')",
	})
	require.NoError(t, err)
}

func TestShellExecPolicyBlocksMarsBinaryArgv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["mars","release","notes","--repo",".","--bump","auto","--dry-run"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "mars_cli")
	require.Contains(t, err.Error(), `["release","notes","--repo",".","--bump","auto","--dry-run"]`)
}

func TestShellExecPolicyBlocksLegacyMarsHarnessBinaryArgv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["mars-harness","release","notes","--repo",".","--bump","auto","--dry-run"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "mars_cli")
	require.Contains(t, err.Error(), `["release","notes","--repo",".","--bump","auto","--dry-run"]`)
}

func TestShellExecPolicyBlocksMarsBinaryShellCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"shell_command":"MARS_CLI_BIN=/tmp/current mars release backfill-notes --repo . --check"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "mars_cli")
	require.Contains(t, err.Error(), `["release","backfill-notes","--repo",".","--check"]`)
}

func TestPipelineFixerAgentSmokeReactRejectsGoValidation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "validation", "agent-smoke"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "validation", "agent-smoke", "current-case.md"), []byte("# Agent Smoke Case Contract\n\n- Project type: `react-web`\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`+"\n"), 0o644))

	ctx := WithSession(context.Background(), Session{Role: "pipeline-fixer", ToolCounts: map[string]int{}})
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","test","./..."]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "React agent-smoke target")
	require.Contains(t, err.Error(), "npm run build")
}

func TestRecordSessionToolOutcomeTracksValidationCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["go","test","./..."],"timeout_seconds":5}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[testCommandFailureKey])

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)
	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[testCommandSuccessKey])
}

func TestRecordSessionToolOutcomeReviewerGoBuildProcedureFailureDoesNotPoisonReview(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "temperature-json-cli"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "temperature-json-cli", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	badRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","cmd/temperature-json-cli"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", badRaw, ToolResult{
		ExitCode: 1,
		Stderr:   "package cmd/temperature-json-cli is not in std (/usr/local/go/src/cmd/temperature-json-cli)\n",
	}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 1, session.ToolCounts[validationProcedureFailureKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[buildCommandFailureKey])

	ctx := WithSession(context.Background(), *session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","./cmd/temperature-json-cli"]}`))
	require.NoError(t, err)
}

func TestRecordSessionToolOutcomeEngineerGoBuildProcedureFailureDoesNotPoisonRepairLane(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "temperature-json-cli"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "temperature-json-cli", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	badRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","cmd/temperature-json-cli"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", badRaw, ToolResult{
		ExitCode: 1,
		Stderr:   "package cmd/temperature-json-cli is not in std (/usr/local/go/src/cmd/temperature-json-cli)\n",
	}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 1, session.ToolCounts[validationProcedureFailureKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[testBuildValidationOutstandingKey])
	require.Equal(t, 0, session.ToolCounts[buildCommandFailureKey])

	ctx := WithSession(context.Background(), *session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","./cmd/temperature-json-cli"]}`))
	require.NoError(t, err)
}

func TestRecordSessionToolOutcomeEngineerPythonValidationHelperProcedureFailureDoesNotPoisonRepairLane(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["python3","-c","import html5lib; parser = html5lib.HTMLParser(); parser.parse('index.html')"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{
		ExitCode: 1,
		Stderr:   "Traceback (most recent call last):\n  File \"<string>\", line 1, in <module>\nModuleNotFoundError: No module named 'html5lib'\n",
	}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 1, session.ToolCounts[validationProcedureFailureKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.NotEmpty(t, session.ToolState[validationProcedureFailureCommandKey])

	ctx := WithSession(context.Background(), *session)
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["python3","-m","http.server","5173","--bind","127.0.0.1"],"background":true}`))
	require.NoError(t, err)
}

func TestRecordSessionToolOutcomeEngineerProjectPythonModuleMissingRemainsRuntimeFailure(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["python3","-c","import app; app.main()"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{
		ExitCode: 1,
		Stderr:   "Traceback (most recent call last):\n  File \"<string>\", line 1, in <module>\nModuleNotFoundError: No module named 'app'\n",
	}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 0, session.ToolCounts[validationProcedureFailureKey])
	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
}

func TestRecordSessionToolOutcomeReviewerRootBuildProcedureFailureDoesNotPoisonReview(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "temperature-json-cli"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "temperature-json-cli", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	badRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","."]}`)

	recordSessionToolOutcome(session, root, "shell_exec", badRaw, ToolResult{
		ExitCode: 1,
		Stderr:   fmt.Sprintf("no Go files in %s\n", dir),
	}, nil)

	require.Equal(t, 1, session.ToolCounts[validationProcedureFailureKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[buildCommandFailureKey])

	ctx := WithSession(context.Background(), *session)
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","./cmd/temperature-json-cli"]}`))
	require.NoError(t, err)
}

func TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["go","test","./cmd/note-stats/..."],"timeout_seconds":5}`)
	args := shellExecArgs{Argv: []string{"go", "test", "./cmd/note-stats/..."}, TimeoutSeconds: 5}
	focusedRaw := json.RawMessage(`{"shell_command":"cd cmd/note-stats && go test -v .","timeout_seconds":5}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{
		ExitCode: 1,
		Output:   `--- FAIL: TestCountWords (0.00s) main_test.go:62: countWords("Test@#$%Special Characters!") = 2, expected 3`,
	}, nil)
	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[testCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[testBuildValidationOutstandingKey])
	require.Equal(t, 1, session.ToolCounts[testBuildValidationFailureFingerprintKey(args)])
	require.Equal(t, 0, session.ToolCounts[testBuildValidationLastFailureEditKey])
	require.Contains(t, session.ToolState[testBuildValidationCommandKey], `"go","test","./cmd/note-stats/..."`)
	require.Contains(t, session.ToolState[testBuildValidationOutputKey], "countWords")
	require.Contains(t, testBuildValidationCorrectionGuidance(*session), "expected 3")
	require.Contains(t, testBuildValidationCorrectionGuidance(*session), "edit the implementation")

	recordSessionToolOutcome(session, root, "file_write", json.RawMessage(`{"path":"cmd/note-stats/main.go","content":"package main\n"}`), ToolResult{}, nil)
	require.Equal(t, 1, session.ToolCounts[testBuildValidationEditAfterFailureKey])

	recordSessionToolOutcome(session, root, "shell_exec", focusedRaw, ToolResult{ExitCode: 0}, nil)
	require.Equal(t, 0, session.ToolCounts[testBuildValidationOutstandingKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[testCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[testCommandSuccessKey])
	require.Empty(t, session.ToolState[testBuildValidationCommandKey])
}

func TestRecordSessionToolOutcomeDependencySyncCountsAsTestBuildRepair(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	buildRaw := json.RawMessage(`{"argv":["npm","run","build"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{
		ExitCode: 127,
		Stderr:   "sh: vite: command not found",
	}, nil)
	require.Equal(t, 1, session.ToolCounts[testBuildValidationOutstandingKey])
	require.Equal(t, 0, session.ToolCounts[testBuildValidationEditAfterFailureKey])

	ctx := WithSession(context.Background(), *session)
	err = preToolPolicy(ctx, root, "shell_exec", buildRaw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "latest repair edit")

	recordSessionToolOutcome(session, root, "dependency_sync", json.RawMessage(`{"action":"install","package_manager":"npm","reason":"Install build tool dependencies"}`), ToolResult{}, nil)
	require.Equal(t, 1, session.ToolCounts[testBuildValidationEditAfterFailureKey])

	ctx = WithSession(context.Background(), *session)
	err = preToolPolicy(ctx, root, "shell_exec", buildRaw)
	require.NoError(t, err)
}

func TestRecordSessionToolOutcomeTracksRuntimeValidationCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["go","run","cmd/note-stats/main.go","--text","hello world"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 0, session.ToolCounts[testCommandSuccessKey])
}

func TestRecordSessionToolOutcomeTracksBrowserProductSmokeNodeEval(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["node","-e","const fs=require('fs'); console.log('browser smoke: Phaser canvas #game new Phaser.Game');"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[browserProductSmokeSuccessKey])
}

func TestRecordSessionToolOutcomeTracksTicketDoneMove(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["git","mv","docs/tickets/in-progress/T-002-ship.md","docs/tickets/done/"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 1, session.ToolCounts[ticketDoneMoveSuccessKey])
	require.Equal(t, "T-002", session.ToolState[ticketDoneMoveLastIDKey])
}

func TestRecordSessionToolOutcomeTracksHTTPProbeValidation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["curl","-fsS","http://127.0.0.1:8765/"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 0, session.ToolCounts[testCommandSuccessKey])
	require.Equal(t, 0, session.ToolCounts[buildCommandSuccessKey])
}

func TestRecordSessionToolOutcomeTracksStaticProductSmoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["curl","-fsS","http://127.0.0.1:5173/"]}`)
	body := "<!DOCTYPE html><html><head><title>Static Focus Timer</title></head><body><h1>Focus Timer</h1></body></html>"

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0, Output: body}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[staticProductSmokeSuccessKey])
}

func TestRecordSessionToolOutcomeDoesNotTreatDirectoryListingAsStaticProductSmoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["curl","-fsS","http://127.0.0.1:5173/"]}`)
	body := `<!DOCTYPE HTML>
<html lang="en">
<head><meta charset="utf-8"><title>Directory listing for /</title></head>
<body><h1>Directory listing for /</h1><ul><li><a href=".git/">.git/</a></li></ul></body>
</html>`

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0, Output: body}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 0, session.ToolCounts[staticProductSmokeSuccessKey])
}

func TestRecordSessionToolOutcomeDoesNotTreatReservedPortHTTPAsStaticProductSmoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["curl","-fsS","http://127.0.0.1:18080/"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 0, session.ToolCounts[staticProductSmokeSuccessKey])
}

func TestRecordSessionToolOutcomeDoesNotCountCannedNodeEvalValidation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["node","-e","console.log('Timer displays initial state correctly as 25:00'); console.log('Start button is visible')"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 0, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandSuccessKey])
}

func TestRecordSessionToolOutcomeTracksRuntimeToolErrorAsOutstandingFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["npm","run","start"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{}, errors.New("shell_exec: command timed out"))

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
}

func TestRecordSessionToolOutcomeDoesNotCountBackgroundServerAsValidation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["python3","-m","http.server","8080","--bind","127.0.0.1"],"background":true}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 0, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandSuccessKey])
}

func TestRecordSessionToolOutcomeTracksExpectedRuntimeFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	raw := json.RawMessage(`{"argv":["go","run",".","--missing"],"expected_exit_code":1}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 1}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeFailureSuccessKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandSuccessKey])
}

func TestRecordSessionToolOutcomeCorrectsUnexpectedRuntimeFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	missingTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation"]}`)
	expectedMissingTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", missingTextRaw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationFailureKey(shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}, 1)])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])

	recordSessionToolOutcome(session, root, "shell_exec", expectedMissingTextRaw, ToolResult{ExitCode: 1}, nil)

	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeFailureSuccessKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeValidationCorrectionKey(shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}, 1)])
}

func TestRecordSessionToolOutcomeTreatsMissingArgumentCLIProbeAsExpectedFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","."]}`)
	missingInputRaw := json.RawMessage(`{"argv":["/tmp/temperature-json-cli-validation"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", missingInputRaw, ToolResult{ExitCode: 1, Stderr: "Error: --celsius flag is required"}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandAttemptKey]-session.ToolCounts[buildCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeFailureSuccessKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Empty(t, session.ToolState[unexpectedRuntimeValidationMissingArgKey])
	require.Empty(t, session.ToolState[unexpectedRuntimeValidationCorrectionKey])
}

func TestRecordSessionToolOutcomeTreatsInvalidInputCLIProbeAsExpectedFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	validInputRaw := json.RawMessage(`{"argv":["go","run","cmd/temperature-json-cli/main.go","25"]}`)
	invalidInputRaw := json.RawMessage(`{"argv":["go","run","cmd/temperature-json-cli/main.go","invalid"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", validInputRaw, ToolResult{ExitCode: 0, Output: `{"celsius":25,"fahrenheit":77}`}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", invalidInputRaw, ToolResult{ExitCode: 1, Stderr: "Error: Invalid temperature value 'invalid'. Must be a number."}, nil)

	require.Equal(t, 2, session.ToolCounts[validationCommandAttemptKey])
	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeFailureSuccessKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
}

func TestRecordSessionToolOutcomeTreatsSurplusArgumentCLIProbeAsExpectedFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/temperature-json-cli-validation","."]}`)
	validInputRaw := json.RawMessage(`{"argv":["/tmp/temperature-json-cli-validation","25"]}`)
	surplusInputRaw := json.RawMessage(`{"argv":["/tmp/temperature-json-cli-validation","25","30"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", validInputRaw, ToolResult{ExitCode: 0, Output: `{"celsius":25,"fahrenheit":77}`}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", surplusInputRaw, ToolResult{ExitCode: 1, Stderr: "error: too many arguments provided"}, nil)

	require.Equal(t, 2, session.ToolCounts[validationCommandAttemptKey]-session.ToolCounts[buildCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey]-session.ToolCounts[buildCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeFailureSuccessKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
}

func TestRecordSessionToolOutcomeStillTreatsPositiveInputFailureAsUnexpected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	validInputRaw := json.RawMessage(`{"argv":["go","run","cmd/temperature-json-cli/main.go","25"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", validInputRaw, ToolResult{ExitCode: 1, Stderr: "Error: Invalid temperature value '25'. Must be a number."}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, 0, session.ToolCounts[expectedRuntimeFailureSuccessKey])
}

func TestRecordSessionToolOutcomeRepairsUnexpectedRuntimeFailureWithExactSuccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	emptyTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text",""]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])

	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, 2, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[runtimeValidationRepairKey(shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}})])
}

func TestRecordSessionToolOutcomeExactSuccessClearsRepeatedRuntimeFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	emptyTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text",""]}`)
	args := shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 1}, nil)
	recordSessionToolOutcome(session, root, "file_write", json.RawMessage(`{"path":"main.go","content":"package main\n"}`), ToolResult{}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, 2, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])

	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, 2, session.ToolCounts[runtimeValidationRepairKey(args)])
}

func TestRecordSessionToolOutcomeTreatsRuntimeErrorStderrAsFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	emptyTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text",""]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 0, Stderr: "error: --text flag is required\nUsage of /tmp/note-stats-validation:"}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[validationCommandSuccessKey]-session.ToolCounts[buildCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])

	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 0, Output: `{"word_count":0,"character_count":0,"line_count":0}`}, nil)

	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
}

func TestRecordSessionToolOutcomeEngineerExpectedExitDoesNotRepairUnexpectedRuntimeFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	emptyTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text",""]}`)
	expectedEmptyTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text",""],"expected_exit_code":1}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 1}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", expectedEmptyTextRaw, ToolResult{ExitCode: 1}, nil)

	require.Equal(t, 1, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 1, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeFailureSuccessKey])
	require.Equal(t, 0, session.ToolCounts[expectedRuntimeValidationCorrectionKey(shellExecArgs{Argv: []string{"/tmp/note-stats-validation", "--text", ""}}, 1)])
}

func TestRecordSessionToolOutcomeEngineerCorrectsMissingArgumentRuntimeFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	missingTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation"]}`)
	expectedMissingTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", missingTextRaw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, "true", session.ToolState[unexpectedRuntimeValidationMissingArgKey])
	require.Contains(t, session.ToolState[unexpectedRuntimeValidationCorrectionKey], `"expected_exit_code":1`)
	require.Contains(t, session.ToolState[unexpectedRuntimeValidationCorrectionKey], `"/tmp/note-stats-validation"`)

	recordSessionToolOutcome(session, root, "shell_exec", expectedMissingTextRaw, ToolResult{ExitCode: 1}, nil)

	require.Equal(t, 0, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 0, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, 1, session.ToolCounts[expectedRuntimeValidationCorrectionKey(shellExecArgs{Argv: []string{"/tmp/note-stats-validation"}}, 1)])
	require.Empty(t, session.ToolState[unexpectedRuntimeValidationMissingArgKey])
	require.Empty(t, session.ToolState[unexpectedRuntimeValidationCorrectionKey])
}

func TestRecordSessionToolOutcomeTracksFailedMissingArgumentCorrectionAttempt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	missingTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation"]}`)
	expectedMissingTextRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation"],"expected_exit_code":1}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", missingTextRaw, ToolResult{ExitCode: 2, Stderr: "panic: runtime error: index out of range"}, nil)
	require.Equal(t, "true", session.ToolState[unexpectedRuntimeValidationMissingArgKey])
	require.Contains(t, session.ToolState[unexpectedRuntimeValidationCorrectionKey], `"expected_exit_code":1`)

	recordSessionToolOutcome(session, root, "shell_exec", expectedMissingTextRaw, ToolResult{ExitCode: 2, Stderr: "panic: runtime error: index out of range"}, nil)

	require.Equal(t, 2, session.ToolCounts[validationCommandFailureKey])
	require.Equal(t, 2, session.ToolCounts[unexpectedRuntimeValidationOutstandingKey])
	require.Equal(t, session.ToolState[unexpectedRuntimeValidationCorrectionKey], session.ToolState[unexpectedRuntimeValidationAttemptedKey])
	require.True(t, missingArgumentCorrectionAttempted(*session))
}

func TestRecordSessionToolOutcomeTracksEditWatermarkAfterUnexpectedRuntimeFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	emptyTextRaw := json.RawMessage(`{"argv":["go","run","./cmd/note-stats","--text",""]}`)
	args := shellExecArgs{Argv: []string{"go", "run", "./cmd/note-stats", "--text", ""}}

	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, 0, session.ToolCounts[runtimeValidationFailureEditWatermarkKey(args)])

	recordSessionToolOutcome(session, root, "file_write", json.RawMessage(`{"path":"cmd/note-stats/main.go","content":"package main\n"}`), ToolResult{}, nil)
	require.Equal(t, 1, session.ToolCounts[runtimeValidationEditAfterFailureKey])

	recordSessionToolOutcome(session, root, "shell_exec", emptyTextRaw, ToolResult{ExitCode: 1}, nil)
	require.Equal(t, 1, session.ToolCounts[runtimeValidationFailureEditWatermarkKey(args)])
}

func TestRecordSessionToolOutcomeTracksNoopFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}

	recordSessionToolOutcome(session, root, "shell_exec", json.RawMessage(`{"argv":[]}`), ToolResult{}, errors.New("noop"))

	require.Equal(t, 1, session.ToolCounts[shellNoopFailureKey])
}

func TestRecordSessionToolOutcomeTracksCodeIntelEfficiencyMetrics(t *testing.T) {
	t.Parallel()
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}

	recordSessionToolOutcome(session, Root{}, "code_search", json.RawMessage(`{"query":"BuildContext"}`), ToolResult{Output: `{"results":[]}`}, nil)
	recordSessionToolOutcome(session, Root{}, "grep", json.RawMessage(`{"pattern":"BuildContext"}`), ToolResult{Output: "internal/codeintel/context.go:1:BuildContext"}, nil)
	recordSessionToolOutcome(session, Root{}, "file_search", json.RawMessage(`{"pattern":"**/*.go"}`), ToolResult{Output: "main.go\ninternal/app/app.go"}, nil)
	recordSessionToolOutcome(session, Root{}, "file_read", json.RawMessage(`{"path":"internal/app/app.go"}`), ToolResult{Output: strings.Repeat("x", 80)}, nil)
	recordSessionToolOutcome(session, Root{}, "file_read", json.RawMessage(`{"path":"internal/app/app.go","start_line":1,"end_line":3}`), ToolResult{Output: "package app"}, nil)
	recordSessionToolOutcome(session, Root{}, "shell_exec", json.RawMessage(`{"argv":["rg","BuildContext"]}`), ToolResult{Output: "hit"}, nil)

	require.Equal(t, 1, session.ToolCounts[codeIntelToolCallsKey])
	require.Equal(t, 1, session.ToolCounts[codeIntelToolSuccessKey])
	require.Equal(t, len(`{"results":[]}`), session.ToolCounts[codeIntelOutputBytesKey])
	require.Equal(t, 4, session.ToolCounts[broadRepoToolCallsKey])
	require.Equal(t, 1, session.ToolCounts[bulkFileReadCallsKey])
	require.Equal(t, 1, session.ToolCounts[broadShellSearchCallsKey])
	require.Equal(t, 80, session.ToolCounts[bulkFileReadOutputBytesKey])
}

func TestRecordSessionToolPolicyFailureTracksNoopFailuresWithoutForcingTerminal(t *testing.T) {
	t.Parallel()
	session := &Session{Role: "qa", ToolCounts: map[string]int{validationCommandSuccessKey: 1}}

	recordSessionToolPolicyFailure(session, "shell_exec", json.RawMessage(`{"argv":[]}`), errors.New("policy"))

	require.Equal(t, 1, session.ToolCounts[shellNoopFailureKey])
	require.Equal(t, 0, session.ToolCounts[reviewTerminalDispositionRequiredKey])
}

func TestRecordSessionToolPolicyFailureSeparatesRepeatedPolicyKeys(t *testing.T) {
	t.Parallel()
	session := &Session{Role: "coo", ToolCounts: map[string]int{}, ToolState: map[string]string{}}

	require.Equal(t, 1, recordSessionToolPolicyFailure(session, "job_disposition_record", json.RawMessage(`{}`), errors.New("policy: missing capability")))
	require.Equal(t, 2, recordSessionToolPolicyFailure(session, "job_disposition_record", json.RawMessage(`{}`), errors.New("policy: missing capability")))
	require.Equal(t, 1, recordSessionToolPolicyFailure(session, "job_disposition_record", json.RawMessage(`{}`), errors.New("policy: different capability")))
	require.Equal(t, 1, recordSessionToolPolicyFailure(session, "file_write", json.RawMessage(`{}`), errors.New("policy: missing capability")))
}

func TestPolicyFailureRepairFeedbackGuidesUnresolvedShellValidationLane(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer"}
	policyErr := errors.New(`policy: engineer cannot run that shell_exec while a failing test or build command is unresolved in this job. Do not run runtime probes, shell wrappers, placeholders, discovery, ticket moves, or unrelated shell commands yet. Use file_read/file_write to repair source, tests, fixtures, or package/build config, or remove duplicate/generated test files created or rewritten earlier in this job, then rerun a test/build command successfully. The exact unresolved command was: shell_exec {"argv":["go","test","./pkg/input"]}. Latest failing output (compact): pkg/input/input_test.go:13:2: no required module provides package github.com/stretchr/testify/assert`)

	err = withPolicyFailureRepairFeedback(root, session, "pre", "shell_exec", policyErr, 2)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Guardrail repair required")
	require.Contains(t, err.Error(), "Stop trying alternate shell_exec or dependency commands")
	require.Contains(t, err.Error(), "repair the source, test, fixture, or package/build config")
	require.Contains(t, err.Error(), "rewrite it with standard-library testing assertions")
}

func TestRecordSessionToolOutcomeTracksTicketCreationFailures(t *testing.T) {
	t.Parallel()
	session := &Session{Role: "cto-weekly", ToolCounts: map[string]int{}}

	recordSessionToolPolicyFailure(session, "ticket_create", json.RawMessage(`{"bdd_scenarios":"[\"F-001-S002\"]"}`), errors.New("parse"))
	require.Equal(t, 1, session.ToolCounts[ticketCreationOutstandingFailureKey])

	recordSessionToolPolicyFailure(session, "file_write", json.RawMessage(`{"path":"docs/tickets/backlog/T-001-demo.md","content":"# demo"}`), errors.New("policy"))
	require.Equal(t, 2, session.ToolCounts[ticketCreationOutstandingFailureKey])

	recordSessionToolOutcome(session, Root{}, "ticket_create", json.RawMessage(`{"bdd_scenarios":["F-001-S002"]}`), ToolResult{}, nil)
	require.Equal(t, 0, session.ToolCounts[ticketCreationOutstandingFailureKey])
}

func TestRecordSessionToolOutcomeIgnoresEngineerTicketEvidencePolicyFailure(t *testing.T) {
	t.Parallel()
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}

	recordSessionToolPolicyFailure(session, "file_write", json.RawMessage(`{"path":"docs/tickets/in-progress/T-001-demo.md","content":"# demo"}`), errors.New("policy"))

	require.Equal(t, 0, session.ToolCounts[ticketCreationOutstandingFailureKey])
}

func TestRecordSessionToolOutcomeTracksValidationArtifactBuildAndRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "qa", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","./cmd/note-stats"]}`)
	runRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text","hello world"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	require.Equal(t, 1, session.ToolCounts[validationCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[buildCommandSuccessKey])
	require.Equal(t, 1, session.ToolCounts[validationArtifactSessionKey("/tmp/note-stats-validation")])
	require.Equal(t, 0, session.ToolCounts[validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation")])

	recordSessionToolOutcome(session, root, "shell_exec", runRaw, ToolResult{ExitCode: 0}, nil)
	require.Equal(t, 2, session.ToolCounts[validationCommandSuccessKey])
}

func TestRecordSessionToolOutcomeRefreshesValidationArtifactAfterRuntimeEdit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}}
	buildRaw := json.RawMessage(`{"argv":["go","build","-o","/tmp/note-stats-validation","."]}`)
	runRaw := json.RawMessage(`{"argv":["/tmp/note-stats-validation","--text",""]}`)

	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", runRaw, ToolResult{ExitCode: 1}, nil)
	recordSessionToolOutcome(session, root, "file_write", json.RawMessage(`{"path":"main.go","content":"package main\n"}`), ToolResult{}, nil)
	recordSessionToolOutcome(session, root, "shell_exec", buildRaw, ToolResult{ExitCode: 0}, nil)

	require.Equal(t, 1, session.ToolCounts[runtimeValidationEditAfterFailureKey])
	require.Equal(t, 1, session.ToolCounts[validationArtifactBuildEditWatermarkKey("/tmp/note-stats-validation")])
}

func TestShellExec_normalizesModelMalformedArgv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "json encoded argv string",
			raw:  `{"argv":"[\"echo\",\"hello\"]","timeout_seconds":5}`,
		},
		{
			name: "python style argv string",
			raw:  `{"argv":"['echo','hello']","timeout_seconds":5}`,
		},
		{
			name: "single simple command string in argv",
			raw:  `{"argv":["echo hello"],"timeout_seconds":5}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			res, err := handleShellExec(context.Background(), root, []byte(tt.raw))
			require.NoError(t, err)
			require.Equal(t, 0, res.ExitCode)
			require.Equal(t, "hello", strings.TrimSpace(res.Output))
		})
	}
}

func TestShellExecNormalizesSimpleCdValidationArgv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	appDir := filepath.Join(dir, "cmd", "temperature-json-cli")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "go.mod"), []byte("module temperature-json-cli\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "main.go"), []byte("package main\nfunc celsiusToFahrenheit(c int) int { return c*9/5 + 32 }\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "main_test.go"), []byte("package main\nimport \"testing\"\nfunc TestConvert(t *testing.T) { if got := celsiusToFahrenheit(100); got != 212 { t.Fatalf(\"got %d\", got) } }\n"), 0o644))
	root, err := NewRoot(dir)
	require.NoError(t, err)

	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":["cd","cmd/temperature-json-cli","&&","go","test","./..."],"timeout_seconds":30}`))

	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Contains(t, res.Output, "ok")
}

func TestShellExec_shellCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"shell_command":"echo ok","timeout_seconds":5}`))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
}

func TestShellExecRejectsShellCommandBackgroundOperator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "standalone background",
			raw:  `{"shell_command":"go run src/main.go & PID=$!"}`,
		},
		{
			name: "compact background",
			raw:  `{"shell_command":"go run src/main.go& PID=$!"}`,
		},
		{
			name: "unquoted URL ampersand",
			raw:  `{"shell_command":"curl http://localhost:8080/health?ready=1&verbose=1"}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "background:true")
			require.Contains(t, err.Error(), "leak child processes")
		})
	}
}

func TestShellExecAllowsShellCommandNonBackgroundAmpersands(t *testing.T) {
	t.Parallel()
	tests := []string{
		`{"shell_command":"printf ok && printf done","timeout_seconds":5}`,
		`{"shell_command":"printf ok 2>&1","timeout_seconds":5}`,
		`{"shell_command":"printf 'a&b'","timeout_seconds":5}`,
		`{"shell_command":"printf \"a&b\"","timeout_seconds":5}`,
	}
	for _, raw := range tests {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(raw))
			require.NoError(t, err)
		})
	}
}

func TestShellExecRejectsBarePortCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "argv",
			raw:  `{"argv":[":8080"]}`,
		},
		{
			name: "shell command",
			raw:  `{"shell_command":":8080"}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)

			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "is a port, not an executable command")
			require.Contains(t, err.Error(), "background:true")
			require.Contains(t, err.Error(), "curl http://localhost:8080/health")
		})
	}
}

func TestShellExecRejectsExternalTimeoutCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "argv timeout",
			raw:  `{"argv":["timeout","5","go","run","main.go"]}`,
		},
		{
			name: "shell command timeout",
			raw:  `{"shell_command":"timeout 5 go run main.go"}`,
		},
		{
			name: "gnu timeout alias",
			raw:  `{"argv":["gtimeout","5","go","test","./..."]}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)

			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "external timeout command")
			require.Contains(t, err.Error(), "timeout_seconds")
			require.Contains(t, err.Error(), "background:true")
		})
	}
}

func TestShellExecPolicyBlocksForegroundServerCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import "net/http"

func main() {
	http.ListenAndServe(":8080", nil)
}
`), 0o644))

	err = checkForegroundLongRunningShellPolicy(root, shellExecArgs{Argv: []string{"go", "run", "main.go"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "likely a long-running server or watcher")
	require.Contains(t, err.Error(), "background:true")
	require.Contains(t, err.Error(), "stop the tracked PID")

	err = checkForegroundLongRunningShellPolicy(root, shellExecArgs{Argv: []string{"go", "run", "main.go"}, Background: true})
	require.NoError(t, err)
}

func TestShellExecPolicyAllowsForegroundGoRunForNonServerCLI(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import "fmt"

func main() {
	fmt.Println("ok")
}
`), 0o644))

	err = checkForegroundLongRunningShellPolicy(root, shellExecArgs{Argv: []string{"go", "run", "main.go"}})
	require.NoError(t, err)
}

func TestShellExec_timeout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = handleShellExec(ctx, root, []byte(`{"shell_command":"sleep 5","timeout_seconds":1}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
}

func TestShellExecBackgroundReportsEarlyExit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(`{"shell_command":"echo boom >&2; exit 7","background":true}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "background process exited during startup")
	require.Equal(t, 7, res.ExitCode)
	require.Contains(t, res.Stderr, "boom")
}

func TestShellExecBackgroundReturnsPIDForLongRunningProcess(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	defer KillBackgroundProcs()
	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","sleep 5"],"background":true}`))
	require.NoError(t, err)
	require.Contains(t, res.Output, "Started in background (PID")
	require.Contains(t, res.Output, "After probes, stop this tracked PID")
	require.Contains(t, res.Output, "Do not call shell_exec with empty argv or :")
}

func TestShellExecBackgroundKeepsOutputDrainedAfterStartup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell loop is unix-specific")
	}
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	defer KillBackgroundProcs()

	res, err := handleShellExec(context.Background(), root, []byte(`{"shell_command":"while true; do echo tick >&2; sleep 1; done","background":true}`))
	require.NoError(t, err)
	pid := backgroundPIDFromOutput(t, res.Output)

	time.Sleep(1500 * time.Millisecond)
	bgMu.Lock()
	_, tracked := bgProcs[pid]
	bgMu.Unlock()
	require.True(t, tracked, "background process should still be tracked after writing post-startup stderr")
}

func TestShellExecNoopReturnsCompletionGuidance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty argv", raw: `{"argv":[]}`},
		{name: "blank argv", raw: `{"argv":["  "]}`},
		{name: "colon argv", raw: `{"argv":[":"]}`},
		{name: "colon shell", raw: `{"shell_command":":"}`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			res, err := handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "no-op command cannot advance work")
			require.Contains(t, res.Output, "No command was run")
			require.Contains(t, res.Output, "job_disposition_record")
		})
	}
}

func TestShellExecNoopAfterBackgroundListsTrackedPID(t *testing.T) {
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	defer KillBackgroundProcs()

	started, err := handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","sleep 5"],"background":true}`))
	require.NoError(t, err)
	pid := backgroundPIDFromOutput(t, started.Output)

	res, err := handleShellExec(context.Background(), root, []byte(`{"argv":[":"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no-op command cannot advance work")
	require.Contains(t, res.Output, fmt.Sprintf("Active background PID(s): %d", pid))
	require.Contains(t, res.Output, `["kill","<pid>"]`)
}

func TestKillBackgroundProcsKillsEscapedChildProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group cleanup test is unix-specific")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "leaker.go")
	bin := filepath.Join(dir, "leaker")
	pidFile := filepath.Join(dir, "child.pid")
	require.NoError(t, os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	cmd := exec.Command("/bin/sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}
`), 0o644))
	build := exec.Command("go", "build", "-o", bin, src)
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleShellExec(context.Background(), root, []byte(fmt.Sprintf(`{"argv":[%q,%q],"background":true}`, bin, pidFile)))
	require.NoError(t, err)
	t.Cleanup(KillBackgroundProcs)

	var childPID int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return false
		}
		childPID = pid
		return syscall.Kill(childPID, 0) == nil
	}, 3*time.Second, 50*time.Millisecond)

	KillBackgroundProcs()
	require.Eventually(t, func() bool {
		return syscall.Kill(childPID, 0) != nil
	}, 3*time.Second, 50*time.Millisecond)
}

func TestShellExecKillTrackedBackgroundPIDKillsDescendant(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group cleanup test is unix-specific")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "leaker.go")
	bin := filepath.Join(dir, "leaker")
	pidFile := filepath.Join(dir, "child.pid")
	require.NoError(t, os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	cmd := exec.Command("/bin/sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}
`), 0o644))
	build := exec.Command("go", "build", "-o", bin, src)
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	root, err := NewRoot(dir)
	require.NoError(t, err)
	res, err := handleShellExec(context.Background(), root, []byte(fmt.Sprintf(`{"argv":[%q,%q],"background":true}`, bin, pidFile)))
	require.NoError(t, err)
	t.Cleanup(KillBackgroundProcs)
	parentPID := backgroundPIDFromOutput(t, res.Output)

	var childPID int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return false
		}
		childPID = pid
		return syscall.Kill(childPID, 0) == nil
	}, 3*time.Second, 50*time.Millisecond)

	res, err = handleShellExec(context.Background(), root, []byte(fmt.Sprintf(`{"argv":["kill","-TERM","%d"]}`, parentPID)))
	require.NoError(t, err)
	require.Contains(t, res.Output, "Killed background process tree")
	require.Eventually(t, func() bool {
		return syscall.Kill(childPID, 0) != nil
	}, 3*time.Second, 50*time.Millisecond)
}

func backgroundPIDFromOutput(t *testing.T, output string) int {
	t.Helper()
	start := strings.Index(output, "PID ")
	require.NotEqual(t, -1, start, output)
	start += len("PID ")
	end := strings.Index(output[start:], ")")
	require.NotEqual(t, -1, end, output)
	pid, err := strconv.Atoi(output[start : start+end])
	require.NoError(t, err)
	return pid
}

func TestShellExec_mutexArgs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := NewRoot(dir)
	require.NoError(t, err)
	_, err = handleShellExec(context.Background(), root, []byte(`{"argv":["sh","-c","echo x"],"shell_command":"echo y"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one")
}

func TestShellExecArgvRejectsShellSyntax(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{name: "shell builtin truncate", raw: `{"argv":[":",">","src/test.html"]}`},
		{name: "redirection as executable", raw: `{"argv":[">","/dev/null"]}`},
		{name: "redirection argument", raw: `{"argv":["echo","ok",">","out.txt"]}`},
		{name: "file descriptor redirection", raw: `{"argv":["echo","ok","2>/dev/null"]}`},
		{name: "control syntax in single argv string", raw: `{"argv":["echo ok && rm nope"]}`},
		{name: "pipeline argument", raw: `{"argv":["cat","README.md","|","wc","-l"]}`},
		{name: "command substitution", raw: `{"argv":["echo","$(pwd)"]}`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "argv mode cannot run shell syntax")
			require.Contains(t, err.Error(), "Use shell_command")
		})
	}
}

func TestShellExecBlocksGitRemoteMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{name: "argv remote add", raw: `{"argv":["git","remote","add","origin","https://example.invalid/repo.git"]}`},
		{name: "shell command remote set url", raw: `{"shell_command":"git remote set-url origin https://example.invalid/repo.git"}`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, err := NewRoot(dir)
			require.NoError(t, err)
			_, err = handleShellExec(context.Background(), root, []byte(tt.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), "git remote")
			require.Contains(t, err.Error(), "blocked")
		})
	}
}

func TestShellExecPolicyBlocksReleaseTagBeforeReleaseNotesCommit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644))
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "feat: seed product"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.2.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.2.0]\n"), 0o644))

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["git","tag","-a","v0.2.0","-m","Release v0.2.0","HEAD"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "release tag v0.2.0 must be created after VERSION and CHANGELOG.md are committed")
	require.Contains(t, err.Error(), "release: notes 0.2.0")
}

func TestShellExecPolicyAllowsReleaseTagWithOnlyRuntimeLearningsDirty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.2.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.2.0]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harness", "learnings.yaml"), []byte("schema_version: 1\n"), 0o644))
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md", ".harness/learnings.yaml"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "release: notes 0.2.0"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harness", "learnings.yaml"), []byte("schema_version: 1\nconventions:\n  release: local\n"), 0o644))

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["git","tag","v0.2.0","HEAD"]}`))
	require.NoError(t, err)
}

func TestShellExecPolicyBlocksReleaseTagTargetThatIsNotReleaseNotesHead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644))
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "feat: seed product"))
	previous := strings.TrimSpace(gitOutput(context.Background(), root, "rev-parse", "HEAD"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.2.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.2.0]\n"), 0o644))
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "release: notes 0.2.0"))

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(fmt.Sprintf(`{"argv":["git","tag","-a","v0.2.0","-m","Release v0.2.0","%s"]}`, previous)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not current release-note HEAD")
}

func TestShellExecPolicyAllowsReleaseTagAtReleaseNotesHead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initGitRepo(t, dir)
	root, err := NewRoot(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.2.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.2.0]\n"), 0o644))
	require.NoError(t, runGitExit0(context.Background(), root, "add", "VERSION", "CHANGELOG.md"))
	require.NoError(t, runGitExit0(context.Background(), root, "commit", "-m", "release: notes 0.2.0"))

	err = preToolPolicy(context.Background(), root, "shell_exec", []byte(`{"argv":["git","tag","-a","v0.2.0","-m","Release v0.2.0"]}`))
	require.NoError(t, err)
}

func TestShellExecReleaseTagMutationPreservesOperandCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       shellExecArgs
		wantTag    string
		wantTarget string
	}{
		{
			name:       "argv HEAD target",
			args:       shellExecArgs{Argv: []string{"git", "tag", "v0.2.0", "HEAD"}},
			wantTag:    "v0.2.0",
			wantTarget: "HEAD",
		},
		{
			name:       "shell HEAD target",
			args:       shellExecArgs{ShellCommand: "git tag v0.2.0 HEAD"},
			wantTag:    "v0.2.0",
			wantTarget: "HEAD",
		},
		{
			name:       "case-sensitive file flag",
			args:       shellExecArgs{Argv: []string{"git", "tag", "-F", "RELEASE_NOTES.md", "v0.2.0", "HEAD"}},
			wantTag:    "v0.2.0",
			wantTarget: "HEAD",
		},
		{
			name:       "case-insensitive executable basename",
			args:       shellExecArgs{Argv: []string{"GIT", "tag", "v0.2.0", "HEAD"}},
			wantTag:    "v0.2.0",
			wantTarget: "HEAD",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tag, target, ok := shellExecReleaseTagMutation(tt.args)
			require.True(t, ok)
			require.Equal(t, tt.wantTag, tag)
			require.Equal(t, tt.wantTarget, target)
		})
	}
}

func TestShellExecReleaseTagMutationKeepsExactGrammar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args shellExecArgs
	}{
		{name: "uppercase subcommand", args: shellExecArgs{Argv: []string{"git", "TAG", "v0.2.0", "HEAD"}}},
		{name: "delete", args: shellExecArgs{Argv: []string{"git", "tag", "--delete", "v0.2.0"}}},
		{name: "list", args: shellExecArgs{Argv: []string{"git", "tag", "--list", "v0.2.0"}}},
		{name: "shell control syntax", args: shellExecArgs{ShellCommand: "git tag v0.2.0 HEAD && git push origin v0.2.0"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, ok := shellExecReleaseTagMutation(tt.args)
			require.False(t, ok)
		})
	}
}

func TestShellPolicyBlocksDestructiveVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "force push reordered shell command",
			raw:  `{"shell_command":"git push origin main --force"}`,
			want: "git push --force",
		},
		{
			name: "force with lease shell command",
			raw:  `{"shell_command":"git push origin main --force-with-lease"}`,
			want: "git push --force",
		},
		{
			name: "short force push argv",
			raw:  `{"argv":["git","push","origin","main","-f"]}`,
			want: "git push --force",
		},
		{
			name: "reset hard",
			raw:  `{"shell_command":"git reset --hard HEAD~1"}`,
			want: "git reset --hard",
		},
		{
			name: "clean combined flags",
			raw:  `{"shell_command":"git clean -dfx"}`,
			want: "git clean -fd",
		},
		{
			name: "branch delete uppercase",
			raw:  `{"shell_command":"git branch -D topic"}`,
			want: "git branch -d",
		},
		{
			name: "root delete reordered flags",
			raw:  `{"argv":["rm","-fr","--","/"]}`,
			want: "rm -rf /",
		},
		{
			name: "repo delete shell command",
			raw:  `{"shell_command":"rm -rf src docs/tickets"}`,
			want: "rm",
		},
		{
			name: "git rm shell command",
			raw:  `{"shell_command":"git rm -r src"}`,
			want: "git rm",
		},
		{
			name: "find delete shell command",
			raw:  `{"shell_command":"find . -name '*.tmp' -delete"}`,
			want: "find -delete",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkShellPolicy(json.RawMessage(tc.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestShellExecReadOnlyClassification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "ls argv", raw: `{"argv":["ls","docs/tickets"]}`, want: true},
		{name: "find without actions", raw: `{"argv":["find","docs/tickets","-name","*.md","-type","f"]}`, want: true},
		{name: "safe git status", raw: `{"argv":["git","status","--short"]}`, want: true},
		{name: "safe branch current", raw: `{"argv":["git","branch","--show-current"]}`, want: true},
		{name: "sed no print", raw: `{"shell_command":"sed -n '1,20p' docs/tickets/README.md"}`, want: true},
		{name: "sed in place", raw: `{"argv":["sed","-i","s/a/b/","file.txt"]}`, want: false},
		{name: "find exec", raw: `{"argv":["find",".","-exec","rm","{}",";"]}`, want: false},
		{name: "shell control", raw: `{"shell_command":"ls docs | wc -l"}`, want: false},
		{name: "touch", raw: `{"argv":["touch","x.txt"]}`, want: false},
		{name: "background", raw: `{"argv":["ls"],"background":true}`, want: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, shellExecReadOnly(json.RawMessage(tt.raw)))
		})
	}
}

func TestShellPolicyBlocksRawDependencyMutationCommands(t *testing.T) {
	t.Parallel()
	cases := []string{
		`{"argv":["npm","install"]}`,
		`{"argv":["npm","ci"]}`,
		`{"argv":["pnpm","install"]}`,
		`{"argv":["yarn","install"]}`,
		`{"argv":["bun","install"]}`,
		`{"argv":["go","get","github.com/stretchr/testify"]}`,
		`{"argv":["go","mod","download"]}`,
		`{"argv":["cargo","fetch"]}`,
		`{"argv":["pip","install","-r","requirements.txt"]}`,
		`{"argv":["python","-m","pip","install","-r","requirements.txt"]}`,
		`{"argv":["bundle","install"]}`,
		`{"argv":["composer","install"]}`,
	}
	for _, raw := range cases {
		err := checkShellPolicy(json.RawMessage(raw))
		require.Error(t, err)
		require.Contains(t, err.Error(), "dependency_sync")
	}
}

func TestShellPolicyBlocksBroadFindWithoutGeneratedExcludes(t *testing.T) {
	t.Parallel()
	err := checkShellPolicy(json.RawMessage(`{"argv":["find",".","-name","*.js"]}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated-directory excludes")

	err = checkShellPolicy(json.RawMessage(`{"argv":["find",".","-path","./node_modules","-prune","-o","-name","*.js"]}`))
	require.NoError(t, err)
}

func TestShellExecReadOnlyAllowedInDirtyRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 12)
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	var policyEvents []PolicyEvent
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
		PolicyRecorder: func(evt PolicyEvent) {
			policyEvents = append(policyEvents, evt)
		},
	}

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["ls","."]}`)
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Empty(t, policyEvents, "read-only inspection should not emit blast-radius policy noise")
	require.NoFileExists(t, filepath.Join(dir, "should-not-exist"))
}

func TestShellExecUnknownCommandBlockedBeforeExecutionInDirtyRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty-00.txt"), []byte(strings.Repeat("dirty\n", 600)), 0o644))
	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	var policyEvents []PolicyEvent
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
		PolicyRecorder: func(evt PolicyEvent) {
			policyEvents = append(policyEvents, evt)
		},
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["touch","should-not-exist"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "may mutate")
	require.Contains(t, err.Error(), "blast radius exceeded")
	require.NoFileExists(t, filepath.Join(dir, "should-not-exist"))
	require.Len(t, policyEvents, 1)
	require.Equal(t, "pre", policyEvents[0].Stage)
}

func TestShellExecAllowsUntrackedRootBuildArtifactCleanup(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	artifact := filepath.Base(dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, artifact), append([]byte{0}, bytes.Repeat([]byte("binary\n"), 600)...), 0o755))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), artifact)
	require.Contains(t, err.Error(), "rm "+artifact)

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["rm","%s"]}`, artifact))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.NoFileExists(t, filepath.Join(dir, artifact))
}

func TestShellExecAllowsUntrackedGoModuleBuildArtifactCleanup(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	artifact := "task-notes-api"
	require.NoError(t, os.WriteFile(filepath.Join(dir, artifact), append([]byte{0}, bytes.Repeat([]byte("binary\n"), 600)...), 0o755))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), artifact)
	require.Contains(t, err.Error(), "rm "+artifact)

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["rm","%s"]}`, artifact))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.NoFileExists(t, filepath.Join(dir, artifact))
	require.FileExists(t, filepath.Join(dir, "go.mod"))
}

func TestShellExecBlocksGoBuildOutputInsideRepoBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["go","build","-o","task-notes-api","main.go"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build output")
	require.Contains(t, err.Error(), "inside the target repo")
	require.Contains(t, err.Error(), "/tmp/task-notes-api-validation")
	require.Contains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/task-notes-api-validation","main.go"]`)
	require.NoFileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestShellExecBlocksDefaultGoBuildInsideRepoBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "dogfood",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["go","build","./..."]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build without -o")
	require.Contains(t, err.Error(), `shell_exec argv ["go","test","./..."]`)
	require.NotContains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/task-notes-api-validation","./..."]`)
	require.NoFileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestShellExecBlocksDefaultGoBuildInShellCommandBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "dogfood",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"shell_command":"go build ./... && go test ./..."}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build without -o")
	require.Contains(t, err.Error(), `shell_exec argv ["go","test","./..."]`)
	require.NotContains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/task-notes-api-validation","./..."]`)
	require.NoFileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestShellExecBlocksDefaultGoBuildForCmdPackageWithExactCorrection(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module note-stats\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "qa",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["go","build","./cmd/note-stats"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build without -o")
	require.Contains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/note-stats-validation","./cmd/note-stats"]`)
	require.NoFileExists(t, filepath.Join(dir, "note-stats"))
}

func TestShellExecBlocksGoBuildOutputInShellCommandSegmentBeforeArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module note-stats\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "note-stats"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "note-stats", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"shell_command":"mkdir -p bin && go build -o bin/note-stats cmd/note-stats/main.go"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go build output")
	require.Contains(t, err.Error(), "inside the target repo")
	require.Contains(t, err.Error(), "/tmp/note-stats-validation")
	require.Contains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/note-stats-validation","cmd/note-stats/main.go"]`)
	require.NoFileExists(t, filepath.Join(dir, "bin", "note-stats"))
}

func TestShellExecBlocksMakeBuildTargetWritingGoBinaryInsideRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module health-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "health-api"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "health-api", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(".PHONY: build test\n\nbuild:\n\tgo build -o bin/health-api cmd/health-api/main.go\n\ntest:\n\tgo test ./...\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["make","build"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "make build target writes Go build output")
	require.Contains(t, err.Error(), "bin/health-api")
	require.Contains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/health-api-validation","cmd/health-api/main.go"]`)
	require.Contains(t, err.Error(), `shell_exec argv ["go","test","./..."]`)
	require.NoFileExists(t, filepath.Join(dir, "bin", "health-api"))
}

func TestFileWriteBlocksMakefileBuildTargetWritingGoBinaryInsideRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module health-api\n\ngo 1.24\n"), 0o644))
	content := ".PHONY: build test\n\nbuild:\n\tgo build -o go-health-api ./cmd/go-health-api\n\ntest:\n\tgo test ./...\n"
	raw, err := json.Marshal(fileWriteArgs{Path: "Makefile", Content: content})
	require.NoError(t, err)

	err = preToolPolicy(context.Background(), root, "file_write", raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Makefile build target writes Go build output")
	require.Contains(t, err.Error(), "go-health-api")
	require.Contains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/go-health-api-validation","./cmd/go-health-api"]`)
}

func TestFileWriteAllowsMakefileBuildTargetWritingValidationBinaryOutsideRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module health-api\n\ngo 1.24\n"), 0o644))
	content := ".PHONY: build test\n\nbuild:\n\tgo build -o /tmp/go-health-api-validation ./cmd/go-health-api\n\ntest:\n\tgo test ./...\n"
	raw, err := json.Marshal(fileWriteArgs{Path: "Makefile", Content: content})
	require.NoError(t, err)

	if err := preToolPolicy(context.Background(), root, "file_write", raw); err != nil {
		t.Fatalf("expected external validation build output to be allowed in Makefile, got %v", err)
	}
}

func TestShellExecAllowsGoBuildOutputOutsideRepo(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	output := filepath.Join(t.TempDir(), "task-notes-api-validation")

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["go","build","-o",%q,"main.go"]}`, output))
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.FileExists(t, output)
}

func TestShellExecBlocksGoBuildOutputOutsideRepoWithoutValidationSuffix(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	output := filepath.Join(t.TempDir(), "task-notes-api")

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", fmt.Sprintf(`{"argv":["go","build","-o",%q,"main.go"]}`, output))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a tracked validation artifact")
	require.Contains(t, err.Error(), "task-notes-api-validation")
	require.Contains(t, err.Error(), `shell_exec argv ["go","build","-o","/tmp/task-notes-api-validation","main.go"]`)
	require.NoFileExists(t, output)
}

func TestShellExecNoopArgsNotMaskedByDirtyArtifact(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	artifact := "task-notes-api"
	require.NoError(t, os.WriteFile(filepath.Join(dir, artifact), append([]byte{0}, bytes.Repeat([]byte("binary\n"), 600)...), 0o755))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	res, err := ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":[]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no-op command cannot advance work")
	require.Contains(t, res.Output, "No command was run")
	require.FileExists(t, filepath.Join(dir, artifact))
}

func TestShellExecStillBlocksRemovalOfOrdinaryFiles(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep me\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["rm","notes.txt"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "forbidden operation")
	require.FileExists(t, filepath.Join(dir, "notes.txt"))
}

func TestShellExecStillBlocksGoModuleNamedTextFileRemoval(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/task-notes-api\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task-notes-api"), []byte("keep me\n"), 0o644))

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"shell_exec"}, "shell_exec", `{"argv":["rm","task-notes-api"]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "forbidden operation")
	require.FileExists(t, filepath.Join(dir, "task-notes-api"))
}

func TestFileWriteBlocksNewRootValidationScript(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	_, err = ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"validate.sh","content":"#!/bin/sh\ngo test ./...\n"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "repo-root scratch validation file")
	require.Contains(t, err.Error(), "direct shell_exec build/run/curl evidence")
	require.NoFileExists(t, filepath.Join(dir, "validate.sh"))
}

func TestFileWriteBlocksNewRootScratchProbe(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	raw := `{"path":"debug.go","content":"/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\npackage main\nfunc main() {}\n"}`
	_, err = ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "repo-root scratch validation file")
	require.Contains(t, err.Error(), "scratch probes become committed product noise")
	require.NoFileExists(t, filepath.Join(dir, "debug.go"))
}

func TestFileWriteAllowsExistingRootValidationScriptUpdate(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "validate.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	runTestGit(t, dir, "add", "validate.sh")
	runTestGit(t, dir, "commit", "-m", "add validation script")

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "engineer",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	res, err := ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"validate.sh","content":"#!/bin/sh\ngo test ./...\n"}`)
	require.NoError(t, err)
	require.Contains(t, res.Output, "wrote")
	content, err := os.ReadFile(filepath.Join(dir, "validate.sh"))
	require.NoError(t, err)
	require.Contains(t, string(content), "go test")
}

func TestSecurityFileWriteBlocksProductRemediation(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "security",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	raw := `{"path":"cmd/app/main.go","content":"package main\nfunc main() {}\n"}`
	_, err = ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "security review cannot write product or ticket files")
	require.Contains(t, err.Error(), "record changes_requested for Engineer")
	require.NoFileExists(t, filepath.Join(dir, "cmd", "app", "main.go"))
}

func TestSecurityFileWriteAllowsSecurityReport(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)

	reg, err := DefaultRegistry()
	require.NoError(t, err)
	ex := NewExecutor(reg)
	ex.Session = &Session{
		Role:         "security",
		RepoID:       "repo-1",
		TrustLevel:   "contributor",
		SafetyLimits: safety.DefaultLimits(),
	}

	res, err := ex.Execute(context.Background(), root, []string{"file_write"}, "file_write", `{"path":"docs/reports/security/security-audit-2026-05-20.md","content":"# Security Audit\n\nPASS\n"}`)
	require.NoError(t, err)
	require.Contains(t, res.Output, "wrote")
	require.FileExists(t, filepath.Join(dir, "docs", "reports", "security", "security-audit-2026-05-20.md"))
}

func TestValidateRepoDiffIgnoresGeneratedUntrackedFiles(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "huge"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "huge", "index.js"), []byte(strings.Repeat("generated\n", 1200)), 0o644))

	err := ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "implementation.js"), []byte(strings.Repeat("source\n", 1200)), 0o644))
	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "implementation.js")
}

func TestValidateRepoDiffIgnoresGeneratedDependencyMetadataLineChurn(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", "package-lock.json")
	runTestGit(t, dir, "commit", "-m", "add lockfile")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(strings.Repeat("lock\n", 1200)), 0o644))
	err := ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "implementation.js"), []byte(strings.Repeat("source\n", 1200)), 0o644))
	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "implementation.js")
	require.NotContains(t, err.Error(), "package-lock.json")
}

func TestValidateRepoDiffIgnoresUntrackedGeneratedDependencyMetadataLineChurn(t *testing.T) {
	dir, root := setupDirtyGitRepo(t, 0)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(strings.Repeat("lock\n", 1200)), 0o644))

	err := ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "implementation.js"), []byte(strings.Repeat("source\n", 1200)), 0o644))
	err = ValidateRepoDiff(context.Background(), root, Session{SafetyLimits: safety.DefaultLimits()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "implementation.js")
	require.NotContains(t, err.Error(), "package-lock.json")
}

func setupDirtyGitRepo(t *testing.T, changedFiles int) (string, Root) {
	t.Helper()
	dir := t.TempDir()
	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "test@example.com")
	runTestGit(t, dir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("baseline\n"), 0o644))
	runTestGit(t, dir, "add", ".")
	runTestGit(t, dir, "commit", "-m", "initial")
	for i := 0; i < changedFiles; i++ {
		name := fmt.Sprintf("dirty-%02d.txt", i)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("dirty\n"), 0o644))
	}
	root, err := NewRoot(dir)
	require.NoError(t, err)
	return dir, root
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
