/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"

	"github.com/stretchr/testify/require"
)

type seqMock struct {
	replies []llm.ChatCompletionResponse
	errs    []error
	i       int
}

func (s *seqMock) ChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (llm.ChatCompletionResponse, error) {
	// Prefer scripted replies when present so tests can mix err-only and reply-only mocks.
	if len(s.replies) > 0 && s.i < len(s.replies) {
		r := s.replies[s.i]
		s.i++
		return r, nil
	}
	if s.i < len(s.errs) && s.errs[s.i] != nil {
		err := s.errs[s.i]
		s.i++
		return llm.ChatCompletionResponse{}, err
	}
	return llm.ChatCompletionResponse{}, fmt.Errorf("seqMock: exhausted replies")
}

func toolResp(name, id, args string) llm.ChatCompletionResponse {
	return llm.ChatCompletionResponse{
		Choices: []llm.Choice{{
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
		}},
	}
}

func textResp(content string) llm.ChatCompletionResponse {
	return llm.ChatCompletionResponse{
		Choices: []llm.Choice{{
			Message: llm.Message{Role: "assistant", Content: content},
		}},
	}
}

func TestRun_persistsTraceToSQLite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	dbPath := "file:" + filepath.Join(dir, "tr.sqlite") + "?mode=rwc"
	st, err := trace.OpenStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	rec := trace.NewRecorder(nil)
	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_write", "w1", `{"path":"t.txt","content":"x"}`),
		textResp("ok"),
	}}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_write"},
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "m", MaxTurns: 10},
		JobID:        "job-sql",
		Trace:        rec,
		TraceStore:   st,
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)

	got, err := st.GetLatestByJobID(context.Background(), "job-sql")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Contains(t, got.TurnsJSONL, `"type":"header"`)
	require.Contains(t, got.SummaryJSON, `"outcome":"completed"`)
}

func TestRun_multiToolThenComplete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_write", "file_read"}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_write", "t1", `{"path":"out.txt","content":"hello"}`),
		textResp("done"),
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: "You are a test harness.",
		UserMessage:  "Write a file then stop.",
		Config:       LoopConfig{Model: "test", MaxTurns: 10},
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.GreaterOrEqual(t, len(res.Messages), 4)
	require.Equal(t, 2, res.LLMCalls)
}

func TestRun_stopsAfterTerminalTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.StopAfterTool = func() bool { return true }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_write", "t1", `{"path":"out.txt","content":"hello"}`),
		textResp("this response should not be requested"),
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_write"},
		SystemPrompt: "You are a test harness.",
		UserMessage:  "Write a file then stop.",
		Config:       LoopConfig{Model: "test", MaxTurns: 10},
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 1, res.LLMCalls)
	require.Equal(t, 1, res.ToolInvocations)
	require.Equal(t, 1, mock.i)
}

func TestRun_requiredTerminalToolRepromptsProseCompletion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "evidence.txt"), []byte("ok\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.StopAfterTool = func() bool { return true }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		textResp("The review is approved."),
		toolResp("file_read", "terminal", `{"path":"evidence.txt"}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read"},
		SystemPrompt:         "You are a test harness.",
		UserMessage:          "Finish with the terminal tool.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "file_read",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 2, res.LLMCalls)
	require.Equal(t, 1, res.ToolInvocations)
	require.Equal(t, 2, mock.i)

	var reminderFound bool
	for _, msg := range res.Messages {
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "cannot finish with prose only") &&
			strings.Contains(msg.Content, "`file_read`") &&
			strings.Contains(msg.Content, "If more inspection or verification is still needed") &&
			strings.Contains(msg.Content, "call an allowed non-terminal tool now") {
			reminderFound = true
		}
	}
	require.True(t, reminderFound, "loop should add a terminal-tool reminder after prose-only completion")
}

func TestRun_inlineToolCallTagExecutesThroughLoop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "evidence.txt"), []byte("ok\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.StopAfterTool = func() bool { return true }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		textResp(`<tool_call>file_read{path:<|"|>evidence.txt<|"|>}</tool_call>`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read"},
		SystemPrompt:         "You are a test harness.",
		UserMessage:          "Finish with the terminal tool.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "file_read",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 1, res.LLMCalls)
	require.Equal(t, 1, res.ToolInvocations)
	require.Equal(t, 1, mock.i)
}

func TestRun_requiredTerminalToolOnlyRepromptsOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		textResp("The review is approved."),
		textResp("Still approved."),
		toolResp("file_read", "unused", `{"path":"missing.txt"}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read"},
		SystemPrompt:         "You are a test harness.",
		UserMessage:          "Finish with the terminal tool.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "file_read",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 2, res.LLMCalls)
	require.Equal(t, 0, res.ToolInvocations)
	require.Equal(t, 2, mock.i)

	var reminders int
	for _, msg := range res.Messages {
		if msg.Role == "user" && strings.Contains(msg.Content, "cannot finish with prose only") {
			reminders++
		}
	}
	require.Equal(t, 1, reminders, "loop should avoid spending the job budget on repeated terminal-tool reminders")
}

func TestRun_requiredTerminalToolGetsOneBudgetGraceTurn(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "evidence.txt"), []byte("ok\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	successfulTools := 0
	ex.StopAfterTool = func() bool {
		successfulTools++
		return successfulTools == 2
	}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "inspect", `{"path":"evidence.txt"}`),
		toolResp("file_read", "terminal", `{"path":"evidence.txt"}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read"},
		SystemPrompt:         "You are a test harness.",
		UserMessage:          "Finish with the terminal tool.",
		Config:               LoopConfig{Model: "test", MaxTurns: 1},
		RequiredTerminalTool: "file_read",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 2, res.LLMCalls)
	require.Equal(t, 2, res.ToolInvocations)

	var budgetReminderFound bool
	for _, msg := range res.Messages {
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "reached its turn budget") &&
			strings.Contains(msg.Content, "`file_read`") &&
			strings.Contains(msg.Content, "Do not call any other tool") {
			budgetReminderFound = true
		}
	}
	require.True(t, budgetReminderFound, "loop should add one terminal-tool grace prompt at the max-turn boundary")
}

func TestRun_terminalToolGraceRejectsNonTerminalTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "evidence.txt"), []byte("ok\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "inspect", `{"path":"evidence.txt"}`),
		toolResp("shell_exec", "cleanup", `{"argv":["echo","late cleanup"]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec"},
		SystemPrompt:         "You are a test harness.",
		UserMessage:          "Finish with the terminal tool.",
		Config:               LoopConfig{Model: "test", MaxTurns: 1},
		RequiredTerminalTool: "file_read",
	})
	require.NoError(t, err)
	require.Equal(t, EndMaxTurns, res.EndReason)
	require.Equal(t, 2, res.LLMCalls)
	require.Equal(t, 1, res.ToolInvocations)
}

func TestRun_threeToolCallsHappyPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_write", "file_read", "shell_exec"}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_write", "w1", `{"path":"main.go","content":"package main\nfunc main() {}\n"}`),
		toolResp("file_read", "r1", `{"path":"main.go"}`),
		toolResp("shell_exec", "s1", `{"command":"echo build-ok"}`),
		textResp("All three steps completed successfully."),
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: "You are a coding agent.",
		UserMessage:  "Create main.go, read it back, then verify with echo.",
		Config:       LoopConfig{Model: "test", MaxTurns: 10},
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 4, res.LLMCalls)
	require.Equal(t, 3, res.ToolInvocations)

	hasAssistant := false
	for _, m := range res.Messages {
		if m.Role == "assistant" && strings.Contains(m.Content, "completed") {
			hasAssistant = true
		}
	}
	require.True(t, hasAssistant, "final assistant text response should be in messages")
}

func TestRun_circleDetected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_read"}
	same := toolResp("file_read", "x", `{"path":"a.txt"}`)
	mock := &seqMock{replies: []llm.ChatCompletionResponse{same, same, same}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: "sys",
		UserMessage:  "user",
		Config:       LoopConfig{Model: "test", MaxTurns: 20},
	})
	require.NoError(t, err)
	require.Equal(t, EndCircleDetected, res.EndReason)
	require.NotEmpty(t, res.CircleDiagnostic)
}

func TestRun_requiredTerminalToolGetsOneCircleGraceTurn(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "evidence.txt"), []byte("ok\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.StopAfterTool = func() bool { return true }

	noOp := toolResp("shell_exec", "noop", `{"argv":[]}`)
	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		noOp,
		noOp,
		noOp,
		toolResp("file_read", "terminal", `{"path":"evidence.txt"}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"shell_exec", "file_read"},
		SystemPrompt:         "You are a test harness.",
		UserMessage:          "Validate, then finish with the terminal tool.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "file_read",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 4, res.LLMCalls)
	require.Equal(t, 3, res.ToolInvocations, "the third repeated no-op should be replaced by a terminal-tool reminder")
	require.Equal(t, 4, mock.i)

	var reminderFound bool
	for _, msg := range res.Messages {
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "repeated the same tool call shape") &&
			strings.Contains(msg.Content, "`file_read`") &&
			strings.Contains(msg.Content, "Do not call any other tool") {
			reminderFound = true
		}
	}
	require.True(t, reminderFound, "loop should add a terminal-tool reminder before circle detection stops the job")
}

func TestRun_reviewEvidenceReminderAllowsOneTerminalCorrection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeReviewEvidenceFixture(t, dir)
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	dispositions := 0
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{
		Role:       "qa",
		ToolCounts: map[string]int{},
		DispositionRecorder: func(context.Context, json.RawMessage) error {
			dispositions++
			return nil
		},
	}
	ex.StopAfterTool = func() bool { return dispositions > 0 }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "read", `{"path":"README.md"}`),
		toolResp("shell_exec", "test", `{"argv":["./go","test","./..."]}`),
		toolResp("docsync_audit", "docsync", `{}`),
		toolResp("shell_exec", "late", `{"argv":["./go","test","./..."]}`),
		toolResp("job_disposition_record", "terminal", `{"status":"approved","ticket_id":"T-001","next_need":"security_review","evidence_links":["./go test ./..."]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec", "docsync_audit", "job_disposition_record"},
		SystemPrompt:         "You are a QA reviewer.",
		UserMessage:          "Review the completed ticket.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "job_disposition_record",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 5, res.LLMCalls)
	require.Equal(t, 4, res.ToolInvocations)
	require.Equal(t, 1, dispositions)

	var reminderFound, correctionFound bool
	for _, msg := range res.Messages {
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "Review evidence is sufficient") &&
			strings.Contains(msg.Content, "`job_disposition_record`") &&
			strings.Contains(msg.Content, "Do not call any other tool") {
			reminderFound = true
		}
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "that tool was not executed") &&
			strings.Contains(msg.Content, "`job_disposition_record`") &&
			strings.Contains(msg.Content, "Do not call any other tool") {
			correctionFound = true
		}
	}
	require.True(t, reminderFound, "loop should add a terminal-only reminder after clean review evidence")
	require.True(t, correctionFound, "loop should give one bounded correction after a non-terminal response")
}

func TestRun_reviewEvidenceReminderRejectsRepeatedNonTerminalTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeReviewEvidenceFixture(t, dir)
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{
		Role:       "qa",
		ToolCounts: map[string]int{},
		DispositionRecorder: func(context.Context, json.RawMessage) error {
			t.Fatal("non-terminal responses should be rejected before disposition")
			return nil
		},
	}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "read", `{"path":"README.md"}`),
		toolResp("shell_exec", "test", `{"argv":["./go","test","./..."]}`),
		toolResp("docsync_audit", "docsync", `{}`),
		toolResp("shell_exec", "late", `{"argv":["./go","test","./..."]}`),
		toolResp("shell_exec", "late-again", `{"argv":["./go","test","./..."]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec", "docsync_audit", "job_disposition_record"},
		SystemPrompt:         "You are a QA reviewer.",
		UserMessage:          "Review the completed ticket.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "job_disposition_record",
	})
	require.NoError(t, err)
	require.Equal(t, EndCircleDetected, res.EndReason)
	require.Equal(t, 5, res.LLMCalls)
	require.Equal(t, 3, res.ToolInvocations)

	var reminderFound, correctionFound bool
	for _, msg := range res.Messages {
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "Review evidence is sufficient") &&
			strings.Contains(msg.Content, "`job_disposition_record`") &&
			strings.Contains(msg.Content, "Do not call any other tool") {
			reminderFound = true
		}
		if msg.Role == "user" &&
			strings.Contains(msg.Content, "that tool was not executed") &&
			strings.Contains(msg.Content, "`job_disposition_record`") {
			correctionFound = true
		}
	}
	require.True(t, reminderFound, "loop should add a terminal-only reminder after clean review evidence")
	require.True(t, correctionFound, "loop should add one correction before failing repeated misses")
}

func TestRun_reviewEvidenceReminderAcceptsTerminalTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeReviewEvidenceFixture(t, dir)
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	dispositions := 0
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{
		Role:       "qa",
		ToolCounts: map[string]int{},
		DispositionRecorder: func(context.Context, json.RawMessage) error {
			dispositions++
			return nil
		},
	}
	ex.StopAfterTool = func() bool { return dispositions > 0 }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "read", `{"path":"README.md"}`),
		toolResp("shell_exec", "test", `{"argv":["./go","test","./..."]}`),
		toolResp("docsync_audit", "docsync", `{}`),
		toolResp("job_disposition_record", "terminal", `{"status":"approved","ticket_id":"T-001","next_need":"security_review","evidence_links":["./go test ./..."]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec", "docsync_audit", "job_disposition_record"},
		SystemPrompt:         "You are a QA reviewer.",
		UserMessage:          "Review the completed ticket.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "job_disposition_record",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 4, res.LLMCalls)
	require.Equal(t, 4, res.ToolInvocations)
	require.Equal(t, 1, dispositions)
}

func TestRun_reviewEvidenceAllowsDocSyncBeforeTerminalBoundary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeReviewEvidenceFixture(t, dir)
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	dispositions := 0
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{
		Role:       "qa",
		ToolCounts: map[string]int{},
		DispositionRecorder: func(context.Context, json.RawMessage) error {
			dispositions++
			return nil
		},
	}
	ex.StopAfterTool = func() bool { return dispositions > 0 }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "read", `{"path":"README.md"}`),
		toolResp("shell_exec", "test", `{"argv":["./go","test","./..."]}`),
		toolResp("docsync_audit", "docsync", `{}`),
		toolResp("job_disposition_record", "terminal", `{"status":"approved","ticket_id":"T-001","next_need":"security_review","evidence_links":["./go test ./...","docsync_audit"]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec", "docsync_audit", "job_disposition_record"},
		SystemPrompt:         "You are a QA reviewer.",
		UserMessage:          "Review the completed ticket.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "job_disposition_record",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 4, res.LLMCalls)
	require.Equal(t, 4, res.ToolInvocations)
	require.Equal(t, 1, dispositions)

	for _, msg := range res.Messages {
		require.False(t,
			msg.Role == "user" && strings.Contains(msg.Content, "Review evidence is sufficient") && strings.Contains(msg.Content, "that tool was not executed"),
			"docsync_audit should be allowed before the terminal-only correction path",
		)
	}
}

func TestRun_reviewEvidenceDoesNotForceTerminalBeforeTestCommandWhenTestsExist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeReviewEvidenceFixture(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "features"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"), []byte("# F-001\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("/* MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] */\npackage main\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{
		Role:       "qa",
		ToolCounts: map[string]int{},
	}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "read", `{"path":"README.md"}`),
		toolResp("docsync_audit", "docsync", `{}`),
		toolResp("shell_exec", "build", `{"argv":["./go","build","./..."]}`),
		toolResp("shell_exec", "test", `{"argv":["./go","test","./..."]}`),
		toolResp("shell_exec", "late", `{"argv":["./go","test","./..."]}`),
		toolResp("shell_exec", "late-again", `{"argv":["./go","test","./..."]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec", "docsync_audit", "job_disposition_record"},
		SystemPrompt:         "You are a QA reviewer.",
		UserMessage:          "Review the completed ticket.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "job_disposition_record",
	})
	require.NoError(t, err)
	require.Equal(t, EndCircleDetected, res.EndReason)
	require.Equal(t, 6, res.LLMCalls)
	require.Equal(t, 4, res.ToolInvocations)
}

func TestRun_reviewNoopAfterBuildAllowsMissingTestCorrection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeReviewEvidenceFixture(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "features"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "features", "F-001-product-walking-skeleton.md"), []byte("# F-001\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("/* MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] */\npackage main\n"), 0o644))
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	dispositions := 0
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{
		Role:       "qa",
		ToolCounts: map[string]int{},
		DispositionRecorder: func(context.Context, json.RawMessage) error {
			dispositions++
			return nil
		},
	}
	ex.StopAfterTool = func() bool { return dispositions > 0 }

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "read", `{"path":"README.md"}`),
		toolResp("docsync_audit", "docsync", `{}`),
		toolResp("shell_exec", "build", `{"argv":["./go","build","-o","/tmp/demo-validation","./..."]}`),
		toolResp("shell_exec", "noop", `{"argv":[]}`),
		toolResp("shell_exec", "test", `{"argv":["./go","test","./..."]}`),
		toolResp("job_disposition_record", "terminal", `{"status":"approved","ticket_id":"T-001","next_need":"security_review","evidence_links":["./go test ./..."]}`),
	}}

	res, err := Run(context.Background(), Params{
		Completer:            mock,
		Registry:             reg,
		Executor:             ex,
		Root:                 root,
		Allowlist:            []string{"file_read", "shell_exec", "docsync_audit", "job_disposition_record"},
		SystemPrompt:         "You are a QA reviewer.",
		UserMessage:          "Review the completed ticket.",
		Config:               LoopConfig{Model: "test", MaxTurns: 10},
		RequiredTerminalTool: "job_disposition_record",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 1, dispositions)

	var missingTestGuidance bool
	for _, msg := range res.Messages {
		if msg.Role == "tool" &&
			strings.Contains(msg.Content, "authoritative test command") &&
			!strings.Contains(msg.Content, "status approved") {
			missingTestGuidance = true
		}
	}
	require.True(t, missingTestGuidance, "no-op after build should guide QA to tests instead of approval")
}

func writeReviewEvidenceFixture(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Demo\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
}

func TestRun_maxTurns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_read"}
	var reps []llm.ChatCompletionResponse
	for i := 0; i < 20; i++ {
		reps = append(reps, toolResp("file_read", fmt.Sprintf("id%d", i), fmt.Sprintf(`{"path":"m%d.txt"}`, i)))
	}
	mock := &seqMock{replies: reps}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 2},
	})
	require.NoError(t, err)
	require.Equal(t, EndMaxTurns, res.EndReason)
}

func TestRun_maxToolCalls(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_read"}
	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "a", `{"path":"x.txt"}`),
		toolResp("file_read", "b", `{"path":"y.txt"}`),
	}}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 10, MaxToolCalls: 1},
	})
	require.NoError(t, err)
	require.Equal(t, EndMaxToolCalls, res.EndReason)
	require.Equal(t, 1, res.ToolInvocations)
	require.Equal(t, 1, res.LLMCalls)
}

func TestRun_tokenBudget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_read"}
	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_read", "a", `{"path":"x.txt"}`),
		textResp("more"),
	}}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: strings.Repeat("big ", 2000),
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 10, TokenBudget: 50},
	})
	require.NoError(t, err)
	require.Equal(t, EndBudgetExceeded, res.EndReason)
	require.Zero(t, res.LLMCalls, "budget should trip before the first LLM call with an oversized system prompt")
}

type slowMock struct {
	delay time.Duration
}

func (s *slowMock) ChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (llm.ChatCompletionResponse, error) {
	select {
	case <-time.After(s.delay):
		return textResp("late"), nil
	case <-ctx.Done():
		return llm.ChatCompletionResponse{}, ctx.Err()
	}
}

func TestRun_wallTimeout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	mock := &slowMock{delay: 150 * time.Millisecond}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_read"},
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 10, WallTime: 50 * time.Millisecond},
	})
	require.NoError(t, err)
	require.Equal(t, EndTimeout, res.EndReason)
	require.Equal(t, 1, res.LLMCalls)
}

func TestRun_emptyResponse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		{Choices: []llm.Choice{{Message: llm.Message{Role: "assistant"}}}},
	}}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_read"},
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 5},
	})
	require.NoError(t, err)
	require.Equal(t, EndEmptyResponse, res.EndReason)
}

func TestRun_llmUnreachable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	mock := &seqMock{errs: []error{fmt.Errorf("down"), fmt.Errorf("down"), fmt.Errorf("down")}}
	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_read"},
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", LLMMaxRetries: 3},
	})
	require.NoError(t, err)
	require.Equal(t, EndLLMUnreachable, res.EndReason)
	require.Error(t, res.Err)
}

func TestPruneContext_replacesOldToolMessages(t *testing.T) {
	t.Parallel()
	bigContent := strings.Repeat("x", 4000) // ~1000 tokens each
	msgs := []llm.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "go"},
		{Role: "assistant", Content: "calling tool", ToolCalls: []llm.ToolCall{{ID: "1", Function: llm.FunctionCall{Name: "shell_exec"}}}},
		{Role: "tool", ToolCallID: "1", Content: bigContent},
		{Role: "assistant", Content: "calling tool", ToolCalls: []llm.ToolCall{{ID: "2", Function: llm.FunctionCall{Name: "shell_exec"}}}},
		{Role: "tool", ToolCallID: "2", Content: bigContent},
		{Role: "assistant", Content: "calling tool", ToolCalls: []llm.ToolCall{{ID: "3", Function: llm.FunctionCall{Name: "file_read"}}}},
		{Role: "tool", ToolCallID: "3", Content: bigContent},
		{Role: "assistant", Content: "calling tool", ToolCalls: []llm.ToolCall{{ID: "4", Function: llm.FunctionCall{Name: "file_read"}}}},
		{Role: "tool", ToolCallID: "4", Content: bigContent},
	}

	// Context size that forces pruning of at least the oldest tool results.
	pruneContext(msgs, nil, 2500)

	// Oldest tool messages (indices 3, 5) should be pruned.
	require.Equal(t, prunedPlaceholder, msgs[3].Content)
	require.Equal(t, prunedPlaceholder, msgs[5].Content)
	// System and user are never touched.
	require.Equal(t, "system prompt", msgs[0].Content)
	require.Equal(t, "go", msgs[1].Content)
	// Recent tail (last 4: indices 6-9) protected — tool results at 7 and 9 kept.
	require.Equal(t, bigContent, msgs[7].Content)
	require.Equal(t, bigContent, msgs[9].Content)
}

func TestPruneContext_replacesOldAssistantToolArguments(t *testing.T) {
	t.Parallel()
	bigArgs := `{"path":"cmd/app/main.go","content":"` + strings.Repeat("x", 12000) + `"}`
	msgs := []llm.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "go"},
		{Role: "assistant", Content: "write large file", ToolCalls: []llm.ToolCall{{ID: "1", Function: llm.FunctionCall{Name: "file_write", Arguments: bigArgs}}}},
		{Role: "tool", ToolCallID: "1", Content: prunedPlaceholder},
		{Role: "assistant", Content: "read it", ToolCalls: []llm.ToolCall{{ID: "2", Function: llm.FunctionCall{Name: "file_read", Arguments: `{"path":"cmd/app/main.go"}`}}}},
		{Role: "tool", ToolCallID: "2", Content: "ok"},
		{Role: "assistant", Content: "recent", ToolCalls: []llm.ToolCall{{ID: "3", Function: llm.FunctionCall{Name: "git_status", Arguments: `{}`}}}},
		{Role: "tool", ToolCallID: "3", Content: "clean"},
		{Role: "assistant", Content: "latest", ToolCalls: []llm.ToolCall{{ID: "4", Function: llm.FunctionCall{Name: "git_diff", Arguments: `{}`}}}},
		{Role: "tool", ToolCallID: "4", Content: ""},
	}

	pruneContext(msgs, nil, 1200)

	require.Equal(t, prunedToolArgumentsJSON, msgs[2].ToolCalls[0].Function.Arguments)
	require.Equal(t, prunedPlaceholder, msgs[2].Content)
	require.Equal(t, `{"path":"cmd/app/main.go"}`, msgs[4].ToolCalls[0].Function.Arguments)
	require.Equal(t, "system prompt", msgs[0].Content)
	require.Equal(t, "go", msgs[1].Content)
	require.Equal(t, `{}`, msgs[6].ToolCalls[0].Function.Arguments)
	require.Equal(t, `{}`, msgs[8].ToolCalls[0].Function.Arguments)
	require.LessOrEqual(t, llm.EstimateTokens(msgs, nil), contextPruneTarget(1200))
}

// stepMock scripts each ChatCompletion call individually so error and reply
// steps can interleave (seqMock prefers replies over errors).
type stepMock struct {
	steps []func() (llm.ChatCompletionResponse, error)
	i     int
}

func (s *stepMock) ChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (llm.ChatCompletionResponse, error) {
	if s.i >= len(s.steps) {
		return llm.ChatCompletionResponse{}, fmt.Errorf("stepMock: exhausted steps")
	}
	step := s.steps[s.i]
	s.i++
	return step()
}

func proseResp(content string) llm.ChatCompletionResponse {
	return llm.ChatCompletionResponse{Choices: []llm.Choice{{Message: llm.Message{Role: "assistant", Content: content}}}}
}

// TestRun_overflowClampPrunesAndRecovers reproduces the T-032 wedge shape:
// the server rejects a grown conversation as over its context window
// (llama.cpp exceed_context_size_error). The loop must prune and retry
// instead of failing the job (AD-288).
func TestRun_overflowClampPrunesAndRecovers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	big := strings.Repeat("x", 6000)
	for _, name := range []string{"big1.txt", "big2.txt", "big3.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(big), 0o644))
	}
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)

	overflow := &llm.ContextSizeError{
		PromptTokens:  33281,
		ContextWindow: 32768,
		Body:          `{"error":{"code":400,"message":"request (33281 tokens) exceeds the available context size (32768 tokens), try increasing it","type":"exceed_context_size_error","n_prompt_tokens":33281,"n_ctx":32768}}`,
	}
	mock := &stepMock{steps: []func() (llm.ChatCompletionResponse, error){
		func() (llm.ChatCompletionResponse, error) { return toolResp("file_read", "1", `{"path":"big1.txt"}`), nil },
		func() (llm.ChatCompletionResponse, error) { return toolResp("file_read", "2", `{"path":"big2.txt"}`), nil },
		func() (llm.ChatCompletionResponse, error) { return toolResp("file_read", "3", `{"path":"big3.txt"}`), nil },
		func() (llm.ChatCompletionResponse, error) { return llm.ChatCompletionResponse{}, overflow },
		func() (llm.ChatCompletionResponse, error) { return proseResp("done"), nil },
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_read"},
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 10},
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason, "overflow must be recovered by pruning, not end the job")
	require.Equal(t, 5, mock.i, "the overflowed turn must be retried after pruning")

	pruned := 0
	for _, m := range res.Messages {
		if m.Content == prunedPlaceholder {
			pruned++
		}
	}
	require.Greater(t, pruned, 0, "older tool results must be pruned before the retry")
}

// TestRun_overflowWithNothingToPruneFailsCleanly: when even the protected
// working set exceeds the served window, the loop fails with the typed
// context error (classified context_overflow by telemetry) instead of
// retrying a request that can never succeed.
func TestRun_overflowWithNothingToPruneFailsCleanly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)

	overflow := &llm.ContextSizeError{PromptTokens: 40000, ContextWindow: 32768, Body: "exceed_context_size_error"}
	calls := 0
	mock := &stepMock{steps: []func() (llm.ChatCompletionResponse, error){
		func() (llm.ChatCompletionResponse, error) { calls++; return llm.ChatCompletionResponse{}, overflow },
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_read"},
		SystemPrompt: "s",
		UserMessage:  "u",
		Config:       LoopConfig{Model: "test", MaxTurns: 10, LLMMaxRetries: 3},
	})
	require.NoError(t, err)
	require.Equal(t, EndLLMUnreachable, res.EndReason)
	var ctxErr *llm.ContextSizeError
	require.ErrorAs(t, res.Err, &ctxErr)
	require.Equal(t, 1, calls, "context-size errors must not be retried verbatim")
}

func TestClampAndPruneForOverflow_clampsToServedWindowAndCalibrates(t *testing.T) {
	t.Parallel()
	big := strings.Repeat("x", 4000)
	msgs := []llm.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "go"},
		{Role: "tool", ToolCallID: "1", Content: big},
		{Role: "tool", ToolCallID: "2", Content: big},
		{Role: "tool", ToolCallID: "3", Content: big},
		{Role: "tool", ToolCallID: "4", Content: big},
		{Role: "tool", ToolCallID: "5", Content: big},
		{Role: "tool", ToolCallID: "6", Content: big},
		// Small recent working set — the protected tail must not block the
		// estimate from reaching the calibrated target.
		{Role: "tool", ToolCallID: "7", Content: "ok"},
		{Role: "tool", ToolCallID: "8", Content: "ok"},
		{Role: "tool", ToolCallID: "9", Content: "ok"},
		{Role: "tool", ToolCallID: "10", Content: "ok"},
	}
	before := llm.EstimateTokens(msgs, nil)
	// Configured window says 131072 but the server actually serves 4000 and
	// counted more tokens than the local estimate.
	ctxErr := &llm.ContextSizeError{PromptTokens: before * 5 / 4, ContextWindow: 4000}
	window, prunedAny := clampAndPruneForOverflow(msgs, nil, ctxErr, 131072)
	require.Equal(t, 4000, window, "working window must clamp to the served window")
	require.True(t, prunedAny)
	after := llm.EstimateTokens(msgs, nil)
	require.Less(t, after, before)
	// Calibrated target: 85% of served window scaled by estimate/served ratio.
	require.LessOrEqual(t, after-0, contextPruneTarget(4000), "estimate must land at or below the margin target")
}

func TestPruneContext_noOpWhenUnderLimit(t *testing.T) {
	t.Parallel()
	msgs := []llm.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "done"},
	}
	pruneContext(msgs, nil, 100000)
	require.Equal(t, "sys", msgs[0].Content)
	require.Equal(t, "hello", msgs[1].Content)
	require.Equal(t, "done", msgs[2].Content)
}
