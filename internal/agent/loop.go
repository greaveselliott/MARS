package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/tools"
)

// Completer is satisfied by *llm.Client for production use.
type Completer interface {
	ChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (llm.ChatCompletionResponse, error)
}

// Params drives one agent loop run (MH-003).
type Params struct {
	Completer    Completer
	Registry     *tools.Registry
	Executor     *tools.Executor
	Root         tools.Root
	Allowlist    []string
	SystemPrompt string
	UserMessage  string
	Config       LoopConfig
}

func (p Params) modelName() string {
	if strings.TrimSpace(p.Config.Model) != "" {
		return strings.TrimSpace(p.Config.Model)
	}
	return "mars-harness"
}

// Run executes the synchronous tool loop until completion or a terminal condition.
func Run(ctx context.Context, p Params) (LoopResult, error) {
	if p.Completer == nil {
		return LoopResult{}, fmt.Errorf("agent: Completer is nil")
	}
	if p.Registry == nil {
		return LoopResult{}, fmt.Errorf("agent: Registry is nil")
	}
	if p.Executor == nil {
		return LoopResult{}, fmt.Errorf("agent: Executor is nil")
	}
	defs, err := p.Registry.Definitions(p.Allowlist)
	if err != nil {
		return LoopResult{}, err
	}

	messages := []llm.Message{
		{Role: "system", Content: p.SystemPrompt},
		{Role: "user", Content: p.UserMessage},
	}

	start := time.Now()
	maxTurns := p.Config.effectiveMaxTurns()
	retries := p.Config.effectiveLLMRetries()

	llmCalls := 0
	toolInvocations := 0
	identicalStreak := 0
	var lastFingerprint string

	for {
		if p.Config.WallTime > 0 && time.Since(start) >= p.Config.WallTime {
			return finish(messages, defs, EndTimeout, llmCalls, toolInvocations, start, ""), nil
		}
		if llmCalls >= maxTurns {
			return finish(messages, defs, EndMaxTurns, llmCalls, toolInvocations, start, ""), nil
		}
		if p.Config.TokenBudget > 0 {
			if llm.EstimateTokens(messages, defs) >= p.Config.TokenBudget {
				return finish(messages, defs, EndBudgetExceeded, llmCalls, toolInvocations, start, ""), nil
			}
		}
		if p.Config.MaxToolCalls > 0 && toolInvocations >= p.Config.MaxToolCalls {
			return finish(messages, defs, EndMaxToolCalls, llmCalls, toolInvocations, start, ""), nil
		}

		req := llm.ChatCompletionRequest{
			Model:    p.modelName(),
			Messages: messages,
			Tools:    defs,
		}
		resp, err := chatWithRetries(ctx, p.Completer, req, retries)
		if err != nil {
			return LoopResult{
				Messages:        messages,
				EndReason:       EndLLMUnreachable,
				LLMCalls:        llmCalls,
				ToolInvocations: toolInvocations,
				WallTime:        time.Since(start),
				Err:             err,
			}, nil
		}
		llmCalls++
		if p.Config.WallTime > 0 && time.Since(start) >= p.Config.WallTime {
			return finish(messages, defs, EndTimeout, llmCalls, toolInvocations, start, ""), nil
		}

		if len(resp.Choices) == 0 {
			return finish(messages, defs, EndEmptyResponse, llmCalls, toolInvocations, start, ""), nil
		}

		am := resp.Choices[0].Message
		calls, parseErr := ToolCallsFromAssistantMessage(am)
		if parseErr != nil {
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: fmt.Sprintf("(tool parse error: %v)", parseErr),
			})
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: "Your previous reply did not contain valid tool calls. Reply with valid tool JSON in a markdown code block, or answer without tools.",
			})
			continue
		}

		content := strings.TrimSpace(am.Content)
		if len(calls) == 0 {
			if content == "" {
				return finish(messages, defs, EndEmptyResponse, llmCalls, toolInvocations, start, ""), nil
			}
			messages = append(messages, llm.Message{Role: "assistant", Content: am.Content})
			return finish(messages, defs, EndCompleted, llmCalls, toolInvocations, start, ""), nil
		}

		fp := fingerprintToolCalls(calls)
		if fp == lastFingerprint {
			identicalStreak++
		} else {
			identicalStreak = 1
			lastFingerprint = fp
		}

		assistantMsg := llm.Message{
			Role:      "assistant",
			Content:   am.Content,
			ToolCalls: calls,
		}
		messages = append(messages, assistantMsg)

		if identicalStreak >= 3 {
			return LoopResult{
				Messages:           messages,
				EndReason:          EndCircleDetected,
				LLMCalls:           llmCalls,
				ToolInvocations:    toolInvocations,
				TokenEstimate:      llm.EstimateTokens(messages, defs),
				WallTime:           time.Since(start),
				CircleDiagnostic:   fp,
			}, nil
		}

		for _, tc := range calls {
			tres, execErr := p.Executor.Execute(ctx, p.Root, p.Allowlist, tc.Function.Name, tc.Function.Arguments)
			toolInvocations++
			body := tres.FormatForModel()
			if execErr != nil {
				body = fmt.Sprintf("error: %v\n%s", execErr, body)
			}
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    body,
			})
		}
	}
}

func finish(msgs []llm.Message, defs []llm.ToolDefinition, reason EndReason, llmCalls, tools int, start time.Time, circle string) LoopResult {
	return LoopResult{
		Messages:           msgs,
		EndReason:          reason,
		LLMCalls:           llmCalls,
		ToolInvocations:    tools,
		TokenEstimate:      llm.EstimateTokens(msgs, defs),
		WallTime:           time.Since(start),
		CircleDiagnostic:   circle,
	}
}

func chatWithRetries(ctx context.Context, c Completer, req llm.ChatCompletionRequest, maxRetries int) (llm.ChatCompletionResponse, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.ChatCompletion(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt == maxRetries-1 {
			break
		}
		ms := 100
		for i := 0; i < attempt && ms < 2000; i++ {
			ms *= 2
		}
		d := time.Duration(ms) * time.Millisecond
		if d > 2*time.Second {
			d = 2 * time.Second
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			_ = t.Stop()
			return llm.ChatCompletionResponse{}, ctx.Err()
		case <-t.C:
		}
		_ = t.Stop()
	}
	return llm.ChatCompletionResponse{}, lastErr
}
