/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
)

const prunedPlaceholder = "[pruned — context limit]"

// pruneContext replaces old tool-result message content with a short placeholder
// when the estimated token count exceeds the context window. It never touches
// the system prompt (index 0), the initial user message (index 1), or the most
// recent 4 messages (the active working set).
func pruneContext(messages []llm.Message, defs []llm.ToolDefinition, contextSize int) {
	est := llm.EstimateTokens(messages, defs)
	if est <= contextSize {
		return
	}

	protectedTail := 4
	if len(messages) < protectedTail+2 {
		return
	}
	pruneEnd := len(messages) - protectedTail

	pruned := 0
	for i := 2; i < pruneEnd && llm.EstimateTokens(messages, defs) > contextSize; i++ {
		if messages[i].Role != "tool" {
			continue
		}
		if messages[i].Content == prunedPlaceholder {
			continue
		}
		messages[i].Content = prunedPlaceholder
		pruned++
	}

	if pruned > 0 {
		slog.Info("agent: pruned context to fit window",
			"pruned_messages", pruned,
			"before_tokens", est,
			"after_tokens", llm.EstimateTokens(messages, defs),
			"context_size", contextSize,
		)
	}
}

// Completer is satisfied by *llm.Client for production use.
type Completer interface {
	ChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (llm.ChatCompletionResponse, error)
}

// LoopUI receives progress callbacks during the agent loop.
// All methods are optional (nil-safe).
type LoopUI interface {
	WriteToolCall(name, args string)
	WriteToolResult(name, output string)
	WriteAssistant(content string)
	WriteTurn(turn, maxTurns int)
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
	UI           LoopUI

	// RequiredTerminalTool keeps server jobs from ending with prose when a
	// durable terminal tool call, such as job_disposition_record, is required.
	RequiredTerminalTool string

	// Optional execution trace (MH-005). When Trace is set, each message is logged as JSONL.
	JobID      string
	Trace      *trace.Recorder
	TraceStore *trace.Store
}

func (p Params) modelName() string {
	if strings.TrimSpace(p.Config.Model) != "" {
		return strings.TrimSpace(p.Config.Model)
	}
	return "mars-harness"
}

func (p Params) jobID() string {
	if strings.TrimSpace(p.JobID) != "" {
		return strings.TrimSpace(p.JobID)
	}
	return "job"
}

func traceAppend(p Params, messages *[]llm.Message, defs []llm.ToolDefinition, msg llm.Message) error {
	*messages = append(*messages, msg)
	if p.Trace == nil {
		return nil
	}
	est := llm.EstimateTokens(*messages, defs)
	return p.Trace.WriteTurn(msg, est)
}

// Run executes the synchronous tool loop until completion or a terminal condition.
func Run(ctx context.Context, p Params) (res LoopResult, err error) {
	slog.Info("agent loop starting",
		"job_id", p.JobID,
		"model", p.modelName(),
		"max_turns", p.Config.effectiveMaxTurns(),
	)
	defer func() {
		slog.Info("agent loop finished",
			"job_id", p.JobID,
			"end_reason", string(res.EndReason),
			"llm_calls", res.LLMCalls,
			"tool_invocations", res.ToolInvocations,
			"tokens", res.TokenEstimate,
			"wall_ms", res.WallTime.Milliseconds(),
		)
	}()

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

	traceStarted := false
	if p.Trace != nil {
		tid := trace.NewID()
		if err := p.Trace.WriteHeader(p.jobID(), tid, p.modelName()); err != nil {
			return LoopResult{}, fmt.Errorf("agent: %w", err)
		}
		traceStarted = true
		for i := range messages {
			est := llm.EstimateTokens(messages[:i+1], defs)
			if err := p.Trace.WriteTurn(messages[i], est); err != nil {
				return LoopResult{}, fmt.Errorf("agent: %w", err)
			}
		}
	}

	defer func() {
		if p.Trace == nil || !traceStarted {
			return
		}
		summ := p.Trace.Finalize(p.jobID(), string(res.EndReason), res.WallTime, res.LLMCalls, res.ToolInvocations, res.Err)
		sj, jErr := json.Marshal(summ)
		if jErr != nil || p.TraceStore == nil {
			return
		}
		if saveErr := p.TraceStore.Save(context.Background(), summ.JobID, p.Trace.TraceID(), p.Trace.JSONL(), string(sj)); saveErr != nil {
			slog.Error("failed to persist trace", "job_id", p.JobID, "error", saveErr)
			if err == nil {
				err = fmt.Errorf("agent: trace store: %w", saveErr)
			}
		}
	}()

	start := time.Now()
	maxTurns := p.Config.effectiveMaxTurns()
	retries := p.Config.effectiveLLMRetries()

	llmCalls := 0
	toolInvocations := 0
	identicalStreak := 0
	terminalToolReminderSent := false
	var lastFingerprint string

	for {
		if p.Config.WallTime > 0 && time.Since(start) >= p.Config.WallTime {
			res = finish(messages, defs, EndTimeout, llmCalls, toolInvocations, start, "")
			return res, nil
		}
		if llmCalls >= maxTurns {
			res = finish(messages, defs, EndMaxTurns, llmCalls, toolInvocations, start, "")
			return res, nil
		}
		if p.Config.TokenBudget > 0 {
			if llm.EstimateTokens(messages, defs) >= p.Config.TokenBudget {
				res = finish(messages, defs, EndBudgetExceeded, llmCalls, toolInvocations, start, "")
				return res, nil
			}
		}
		if p.Config.MaxToolCalls > 0 && toolInvocations >= p.Config.MaxToolCalls {
			res = finish(messages, defs, EndMaxToolCalls, llmCalls, toolInvocations, start, "")
			return res, nil
		}

		if p.UI != nil {
			p.UI.WriteTurn(llmCalls+1, maxTurns)
		}

		pruneContext(messages, defs, p.Config.effectiveContextSize())

		req := llm.ChatCompletionRequest{
			Model:    p.modelName(),
			Messages: messages,
			Tools:    defs,
		}
		resp, err := chatWithRetries(ctx, p.Completer, req, retries)
		if err != nil {
			res = LoopResult{
				Messages:        messages,
				EndReason:       EndLLMUnreachable,
				LLMCalls:        llmCalls,
				ToolInvocations: toolInvocations,
				WallTime:        time.Since(start),
				Err:             err,
			}
			return res, nil
		}
		llmCalls++
		if p.Config.WallTime > 0 && time.Since(start) >= p.Config.WallTime {
			res = finish(messages, defs, EndTimeout, llmCalls, toolInvocations, start, "")
			return res, nil
		}

		if len(resp.Choices) == 0 {
			res = finish(messages, defs, EndEmptyResponse, llmCalls, toolInvocations, start, "")
			return res, nil
		}

		am := resp.Choices[0].Message
		calls, parseErr := ToolCallsFromAssistantMessage(am)
		if parseErr != nil {
			if err := traceAppend(p, &messages, defs, llm.Message{
				Role:    "assistant",
				Content: fmt.Sprintf("(tool parse error: %v)", parseErr),
			}); err != nil {
				return LoopResult{}, err
			}
			if err := traceAppend(p, &messages, defs, llm.Message{
				Role:    "user",
				Content: "Your previous reply did not contain valid tool calls. Reply with valid tool JSON in a markdown code block, or answer without tools.",
			}); err != nil {
				return LoopResult{}, err
			}
			continue
		}

		content := strings.TrimSpace(am.Content)
		if len(calls) == 0 {
			if content == "" {
				res = finish(messages, defs, EndEmptyResponse, llmCalls, toolInvocations, start, "")
				return res, nil
			}
			if p.UI != nil {
				p.UI.WriteAssistant(content)
			}
			if err := traceAppend(p, &messages, defs, llm.Message{Role: "assistant", Content: am.Content}); err != nil {
				return LoopResult{}, err
			}
			if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && !terminalToolReminderSent {
				if err := traceAppend(p, &messages, defs, llm.Message{
					Role: "user",
					Content: fmt.Sprintf(
						"This server job cannot finish with prose only. If your work is complete, call `%s` now with the terminal status, reason, evidence_links, and handoff or feedback fields required by the dispatch protocol. If more inspection or verification is still needed, call an allowed non-terminal tool now, then call `%s` when the work is complete. Do not narrate next steps without a tool call.",
						required,
						required,
					),
				}); err != nil {
					return LoopResult{}, err
				}
				terminalToolReminderSent = true
				continue
			}
			res = finish(messages, defs, EndCompleted, llmCalls, toolInvocations, start, "")
			return res, nil
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
		if err := traceAppend(p, &messages, defs, assistantMsg); err != nil {
			return LoopResult{}, err
		}

		if identicalStreak >= 3 {
			res = LoopResult{
				Messages:         messages,
				EndReason:        EndCircleDetected,
				LLMCalls:         llmCalls,
				ToolInvocations:  toolInvocations,
				TokenEstimate:    llm.EstimateTokens(messages, defs),
				WallTime:         time.Since(start),
				CircleDiagnostic: fp,
			}
			return res, nil
		}

		for _, tc := range calls {
			if p.UI != nil {
				p.UI.WriteToolCall(tc.Function.Name, tc.Function.Arguments)
			}
			tres, execErr := p.Executor.Execute(ctx, p.Root, p.Allowlist, tc.Function.Name, tc.Function.Arguments)
			toolInvocations++
			body := tres.FormatForModel()
			if execErr != nil {
				body = fmt.Sprintf("error: %v\n%s", execErr, body)
			}
			if p.UI != nil {
				p.UI.WriteToolResult(tc.Function.Name, body)
			}
			if err := traceAppend(p, &messages, defs, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    body,
			}); err != nil {
				return LoopResult{}, err
			}
			if execErr == nil && p.Executor.StopAfterTool != nil && p.Executor.StopAfterTool() {
				res = finish(messages, defs, EndCompleted, llmCalls, toolInvocations, start, "")
				return res, nil
			}
		}
	}
}

func finish(msgs []llm.Message, defs []llm.ToolDefinition, reason EndReason, llmCalls, tools int, start time.Time, circle string) LoopResult {
	return LoopResult{
		Messages:         msgs,
		EndReason:        reason,
		LLMCalls:         llmCalls,
		ToolInvocations:  tools,
		TokenEstimate:    llm.EstimateTokens(msgs, defs),
		WallTime:         time.Since(start),
		CircleDiagnostic: circle,
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
		slog.Warn("agent: LLM call failed, will retry",
			"attempt", attempt+1, "max", maxRetries, "err", err)
		if attempt == maxRetries-1 {
			break
		}
		ms := 2000
		for i := 0; i < attempt && ms < 15000; i++ {
			ms *= 2
		}
		d := time.Duration(ms) * time.Millisecond
		if d > 15*time.Second {
			d = 15 * time.Second
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
