package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/tools"

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
