/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/context-efficiency.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/llm"
	"github.com/greaveselliott/mars/internal/tools"
	"github.com/greaveselliott/mars/internal/trace"
)

const (
	prunedPlaceholder        = "[pruned — context limit]"
	prunedToolArgumentsJSON  = `{"pruned":"context limit"}`
	terminalToolGraceTimeout = 90 * time.Second
	maxOverflowClampRetries  = 3
)

// pruneContext replaces old tool-result message content with a short placeholder
// when the estimated token count exceeds the context window. It never touches
// the system prompt (index 0), the initial user message (index 1), or the most
// recent 4 messages (the active working set).
func pruneContext(messages []llm.Message, defs []llm.ToolDefinition, contextSize int) {
	est := llm.EstimateTokens(messages, defs)
	if est <= contextSize {
		return
	}
	pruneContextToTarget(messages, defs, contextSize, contextPruneTarget(contextSize))
}

// pruneContextToTarget prunes oldest-first down to an explicit estimator-token
// target. Split out from pruneContext so the server-reported overflow clamp
// (AD-288) can drive pruning to a calibrated target instead of the default
// margin.
func pruneContextToTarget(messages []llm.Message, defs []llm.ToolDefinition, contextSize, target int) {
	est := llm.EstimateTokens(messages, defs)

	protectedTail := 4
	if len(messages) < protectedTail+2 {
		return
	}
	pruneEnd := len(messages) - protectedTail

	prunedTools := 0
	for i := 2; i < pruneEnd && llm.EstimateTokens(messages, defs) > target; i++ {
		if messages[i].Role != "tool" {
			continue
		}
		if messages[i].Content == prunedPlaceholder {
			continue
		}
		messages[i].Content = prunedPlaceholder
		prunedTools++
	}

	prunedAssistantCalls := 0
	for i := 2; i < pruneEnd && llm.EstimateTokens(messages, defs) > target; i++ {
		if messages[i].Role != "assistant" || len(messages[i].ToolCalls) == 0 {
			continue
		}
		changed := false
		for j := range messages[i].ToolCalls {
			if messages[i].ToolCalls[j].Function.Arguments == prunedToolArgumentsJSON {
				continue
			}
			messages[i].ToolCalls[j].Function.Arguments = prunedToolArgumentsJSON
			changed = true
		}
		if messages[i].Content != "" && messages[i].Content != prunedPlaceholder {
			messages[i].Content = prunedPlaceholder
			changed = true
		}
		if changed {
			prunedAssistantCalls++
		}
	}

	prunedProse := 0
	for i := 2; i < pruneEnd && llm.EstimateTokens(messages, defs) > target; i++ {
		if messages[i].Role != "assistant" && messages[i].Role != "user" {
			continue
		}
		if messages[i].Content == "" || messages[i].Content == prunedPlaceholder {
			continue
		}
		messages[i].Content = prunedPlaceholder
		prunedProse++
	}

	if prunedTools > 0 || prunedAssistantCalls > 0 || prunedProse > 0 {
		slog.Info("agent: pruned context to fit window",
			"pruned_tool_messages", prunedTools,
			"pruned_assistant_tool_calls", prunedAssistantCalls,
			"pruned_prose_messages", prunedProse,
			"before_tokens", est,
			"after_tokens", llm.EstimateTokens(messages, defs),
			"context_size", contextSize,
			"target_tokens", target,
		)
	}
}

// clampAndPruneForOverflow reacts to a server-side context rejection (AD-288).
// It clamps the loop's working window to the server-reported serving window
// when that is smaller than the configured one, then prunes to a calibrated
// target: the server's measured prompt size is used to rescale the estimator
// target so the retried request lands inside the served window even when the
// local estimate and the server tokenizer disagree. Returns the clamped
// window and whether pruning reduced the estimate (retry is pointless when
// nothing could be pruned).
func clampAndPruneForOverflow(messages []llm.Message, defs []llm.ToolDefinition, ctxErr *llm.ContextSizeError, window int) (int, bool) {
	if ctxErr.ContextWindow > 0 && ctxErr.ContextWindow < window {
		window = ctxErr.ContextWindow
	}
	target := contextPruneTarget(window)
	before := llm.EstimateTokens(messages, defs)
	if ctxErr.PromptTokens > 0 && before > 0 {
		// Measured calibration: estimator target × (estimate / served count).
		// When the estimator under-counts relative to the server tokenizer,
		// this shrinks the target proportionally.
		scaled := int(int64(target) * int64(before) / int64(ctxErr.PromptTokens))
		if scaled > 0 && scaled < target {
			target = scaled
		}
	}
	pruneContextToTarget(messages, defs, window, target)
	after := llm.EstimateTokens(messages, defs)
	return window, after < before
}

func contextPruneTarget(contextSize int) int {
	if contextSize <= 0 {
		return 0
	}
	target := contextSize * 85 / 100
	if target < 1 {
		return contextSize
	}
	return target
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
	Completer           Completer
	Registry            *tools.Registry
	Executor            *tools.Executor
	Root                tools.Root
	Allowlist           []string
	SystemPrompt        string
	UserMessage         string
	Preflight           []PreflightToolCall
	CodeIntelMode       string
	CodeIntelModeSource string
	// MaintainCodeGraph refreshes code-intel after successful repository
	// mutations so later turns see current graph state.
	MaintainCodeGraph bool
	Config            LoopConfig
	UI                LoopUI

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
	return "mars"
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

func executeLoopToolCall(ctx context.Context, p Params, messages *[]llm.Message, defs []llm.ToolDefinition, call llm.ToolCall, toolInvocations *int) (error, error) {
	if p.UI != nil {
		p.UI.WriteToolCall(call.Function.Name, call.Function.Arguments)
	}
	tres, execErr := p.Executor.Execute(ctx, p.Root, p.Allowlist, call.Function.Name, call.Function.Arguments)
	*toolInvocations++
	body := tres.FormatForModel()
	if execErr != nil {
		body = fmt.Sprintf("error: %v\n%s", execErr, body)
	}
	if p.UI != nil {
		p.UI.WriteToolResult(call.Function.Name, body)
	}
	if err := traceAppend(p, messages, defs, llm.Message{
		Role:       "tool",
		ToolCallID: call.ID,
		Content:    body,
	}); err != nil {
		return execErr, err
	}
	return execErr, nil
}

func shouldRefreshCodeGraphAfterMutation(p Params, toolName, argsJSON string) bool {
	return p.MaintainCodeGraph && tools.Allowlisted("code_index", p.Allowlist) && codeGraphMutationTool(toolName, argsJSON)
}

func refreshCodeGraph(ctx context.Context, p Params, messages *[]llm.Message, defs []llm.ToolDefinition, toolInvocations *int) error {
	call := llm.ToolCall{
		ID:   fmt.Sprintf("codegraph_refresh_%d", *toolInvocations+1),
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "code_index",
			Arguments: `{}`,
		},
	}
	if err := traceAppend(p, messages, defs, llm.Message{
		Role:      "assistant",
		Content:   "Mars code graph maintenance: refresh changed files after repository mutation.",
		ToolCalls: []llm.ToolCall{call},
	}); err != nil {
		return err
	}
	_, traceErr := executeLoopToolCall(ctx, p, messages, defs, call, toolInvocations)
	return traceErr
}

func maybeRefreshCodeGraphAfterMutation(ctx context.Context, p Params, messages *[]llm.Message, defs []llm.ToolDefinition, toolName, argsJSON string, toolInvocations *int) error {
	if !shouldRefreshCodeGraphAfterMutation(p, toolName, argsJSON) {
		return nil
	}
	return refreshCodeGraph(ctx, p, messages, defs, toolInvocations)
}

func codeGraphMutationTool(name, argsJSON string) bool {
	switch strings.TrimSpace(name) {
	case "file_write", "dependency_sync", "mars_cli", "mars_harness_cli", "record_decision", "ticket_create", "tool_create", "persona_create":
		return true
	case "shell_exec":
		return shellExecMayMutateRepo(argsJSON)
	default:
		return false
	}
}

func shellExecMayMutateRepo(argsJSON string) bool {
	var args struct {
		Argv         []string `json:"argv"`
		ShellCommand string   `json:"shell_command"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return false
	}
	if len(args.Argv) > 0 {
		return argvMayMutateRepo(args.Argv)
	}
	return shellCommandMayMutateRepo(args.ShellCommand)
}

func argvMayMutateRepo(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	program := commandBase(argv[0])
	if program == "" {
		return false
	}
	switch program {
	case "touch", "mkdir", "mv", "cp", "rm", "rmdir", "tee":
		return true
	case "sed", "perl":
		return argvContains(argv[1:], "-i") || argvContainsPrefix(argv[1:], "-i")
	case "git":
		if len(argv) < 2 {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(argv[1])) {
		case "mv", "rm", "apply", "am", "merge", "rebase", "cherry-pick", "checkout", "switch", "restore", "reset", "clean", "pull", "stash":
			return true
		default:
			return false
		}
	case "go":
		if len(argv) < 2 {
			return false
		}
		sub := strings.ToLower(strings.TrimSpace(argv[1]))
		return sub == "generate" || (sub == "mod" && len(argv) >= 3 && goModSubcommandMayMutate(argv[2])) || sub == "get"
	case "npm", "pnpm", "yarn", "bun":
		return packageManagerArgsMayMutate(argv[1:])
	case "cargo":
		if len(argv) < 2 {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(argv[1])) {
		case "add", "remove", "update", "generate-lockfile":
			return true
		default:
			return false
		}
	case "mars":
		return marsArgsMayMutate(argv[1:])
	default:
		return false
	}
}

func shellCommandMayMutateRepo(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	lower := strings.ToLower(command)
	for _, token := range []string{">", ">>", " tee ", "sed -i", "perl -i", "git mv", "git rm", "git apply", "git checkout", "git switch", "git restore", "git reset", "git clean", "go generate", "go get", "go mod tidy", "npm install", "npm i ", "pnpm install", "yarn install", "bun install", "cargo add", "cargo update", "touch ", "mkdir ", "mv ", "cp ", "rm "} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func commandBase(cmd string) string {
	cmd = strings.Trim(strings.TrimSpace(cmd), `"'`)
	if cmd == "" {
		return ""
	}
	if idx := strings.LastIndexAny(cmd, `/\`); idx >= 0 && idx < len(cmd)-1 {
		cmd = cmd[idx+1:]
	}
	return strings.ToLower(cmd)
}

func argvContains(args []string, needle string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == needle {
			return true
		}
	}
	return false
}

func argvContainsPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(strings.TrimSpace(arg), prefix) {
			return true
		}
	}
	return false
}

func goModSubcommandMayMutate(sub string) bool {
	switch strings.ToLower(strings.TrimSpace(sub)) {
	case "tidy", "edit", "init", "vendor":
		return true
	default:
		return false
	}
}

func packageManagerArgsMayMutate(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "install", "i", "add", "remove", "update", "upgrade", "ci":
		return true
	default:
		return false
	}
}

func marsArgsMayMutate(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "init", "update", "scan", "release":
		return true
	default:
		return false
	}
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
		if counts := copyToolCounts(p.Executor); len(counts) > 0 {
			summ.ToolCounts = counts
		}
		if mode := strings.TrimSpace(p.CodeIntelMode); mode != "" {
			summ.CodeIntel = &trace.CodeIntelSummary{
				Mode:   mode,
				Source: strings.TrimSpace(p.CodeIntelModeSource),
			}
		}
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
	overflowClamps := 0
	contextWindow := p.Config.effectiveContextSize()
	identicalStreak := 0
	terminalToolReminderSent := false
	terminalToolBudgetReminderSent := false
	terminalToolCircleReminderSent := false
	terminalToolEvidenceReminderSent := false
	terminalToolEvidenceRetrySent := false
	var lastFingerprint string

	for _, preflight := range p.Preflight {
		name := strings.TrimSpace(preflight.Name)
		if name == "" {
			continue
		}
		args := strings.TrimSpace(preflight.ArgsJSON)
		if args == "" {
			args = `{}`
		}
		callID := fmt.Sprintf("preflight_%d", toolInvocations+1)
		content := strings.TrimSpace(preflight.Rationale)
		if content == "" {
			content = fmt.Sprintf("Preflight %s", name)
		}
		assistantMsg := llm.Message{
			Role:    "assistant",
			Content: content,
			ToolCalls: []llm.ToolCall{{
				ID:   callID,
				Type: "function",
				Function: llm.FunctionCall{
					Name:      name,
					Arguments: args,
				},
			}},
		}
		if err := traceAppend(p, &messages, defs, assistantMsg); err != nil {
			return LoopResult{}, err
		}
		if _, err := executeLoopToolCall(ctx, p, &messages, defs, assistantMsg.ToolCalls[0], &toolInvocations); err != nil {
			return LoopResult{}, err
		}
	}

	for {
		if p.Config.WallTime > 0 && time.Since(start) >= p.Config.WallTime {
			res = finish(messages, defs, EndTimeout, llmCalls, toolInvocations, start, "")
			return res, nil
		}
		if llmCalls >= maxTurns {
			if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && !terminalToolBudgetReminderSent {
				if err := traceAppend(p, &messages, defs, llm.Message{
					Role: "user",
					Content: fmt.Sprintf(
						"The job has reached its turn budget. Stop inspection and validation now. Call `%s` in the next response with the terminal status, reason, evidence_links, and any handoff or feedback fields required by the dispatch protocol. If validation failed, record changes_requested with the exact failing command and output. Do not call any other tool.",
						required,
					),
				}); err != nil {
					return LoopResult{}, err
				}
				terminalToolBudgetReminderSent = true
				maxTurns++
				continue
			}
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

		pruneContext(messages, defs, contextWindow)

		req := llm.ChatCompletionRequest{
			Model:    p.modelName(),
			Messages: messages,
			Tools:    defs,
		}
		chatCtx := ctx
		cancelChat := func() {}
		if terminalToolBudgetReminderSent || terminalToolCircleReminderSent || terminalToolEvidenceReminderSent {
			chatCtx, cancelChat = context.WithTimeout(ctx, terminalToolGraceTimeout)
		}
		resp, err := chatWithRetries(chatCtx, p.Completer, req, retries)
		cancelChat()
		if err != nil {
			var ctxErr *llm.ContextSizeError
			if errors.As(err, &ctxErr) && overflowClamps < maxOverflowClampRetries {
				overflowClamps++
				if clamped, pruned := clampAndPruneForOverflow(messages, defs, ctxErr, contextWindow); pruned {
					contextWindow = clamped
					slog.Warn("agent: server rejected prompt as over context window; pruned and retrying",
						"job_id", p.JobID,
						"server_prompt_tokens", ctxErr.PromptTokens,
						"server_context_window", ctxErr.ContextWindow,
						"clamped_window", contextWindow,
						"overflow_clamp", overflowClamps,
					)
					continue
				}
			}
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
			if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && terminalToolEvidenceReminderSent {
				if !terminalToolEvidenceRetrySent {
					if err := traceAppend(p, &messages, defs, llm.Message{
						Role: "user",
						Content: fmt.Sprintf(
							"Your last response had no tool call after review evidence was already sufficient. %s",
							reviewTerminalEvidencePrompt(required, tools.ReviewTerminalDispositionGuidance(p.Root, p.Executor.Session)),
						),
					}); err != nil {
						return LoopResult{}, err
					}
					terminalToolEvidenceRetrySent = true
					if llmCalls >= maxTurns {
						maxTurns++
					}
					continue
				}
				res = finish(messages, defs, EndCircleDetected, llmCalls, toolInvocations, start, "terminal evidence reminder received prose-only response")
				return res, nil
			}
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
			if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && !terminalToolEvidenceReminderSent && tools.ReviewTerminalEvidenceSatisfied(p.Root, p.Executor.Session) {
				tools.MarkReviewTerminalDispositionRequired(p.Executor.Session)
				guidance := tools.ReviewTerminalDispositionGuidance(p.Root, p.Executor.Session)
				if err := traceAppend(p, &messages, defs, llm.Message{
					Role:    "user",
					Content: reviewTerminalEvidencePrompt(required, guidance),
				}); err != nil {
					return LoopResult{}, err
				}
				terminalToolEvidenceReminderSent = true
				continue
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

		if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && terminalToolBudgetReminderSent {
			if len(calls) != 1 || strings.TrimSpace(calls[0].Function.Name) != required {
				res = finish(messages, defs, EndMaxTurns, llmCalls, toolInvocations, start, "")
				return res, nil
			}
		}
		if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && terminalToolCircleReminderSent {
			if len(calls) != 1 || strings.TrimSpace(calls[0].Function.Name) != required {
				res = finish(messages, defs, EndCircleDetected, llmCalls, toolInvocations, start, "")
				return res, nil
			}
		}
		if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && terminalToolEvidenceReminderSent {
			guidance := tools.ReviewTerminalDispositionGuidance(p.Root, p.Executor.Session)
			if (len(calls) != 1 || strings.TrimSpace(calls[0].Function.Name) != required) && !reviewTerminalEvidenceAllowsReportTool(guidance, calls) {
				if terminalToolEvidenceRetrySent {
					res = finish(messages, defs, EndCircleDetected, llmCalls, toolInvocations, start, "terminal evidence reminder received repeated non-terminal tool")
					return res, nil
				}
				if err := traceAppend(p, &messages, defs, llm.Message{
					Role: "user",
					Content: fmt.Sprintf(
						"Your last response called %s after review evidence was already sufficient; that tool was not executed. %s",
						toolCallNamesForPrompt(calls),
						reviewTerminalEvidencePrompt(required, tools.ReviewTerminalDispositionGuidance(p.Root, p.Executor.Session)),
					),
				}); err != nil {
					return LoopResult{}, err
				}
				terminalToolEvidenceRetrySent = true
				if llmCalls >= maxTurns {
					maxTurns++
				}
				continue
			}
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
			if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && !terminalToolCircleReminderSent {
				if err := traceAppend(p, &messages, defs, llm.Message{
					Role: "user",
					Content: fmt.Sprintf(
						"You have repeated the same tool call shape and are about to be stopped as a loop. Stop inspection and validation now. Call `%s` in the next response with the terminal status, reason, evidence_links, ticket_id when applicable, and any handoff or feedback fields required by the dispatch protocol. If validation passed, record the successful disposition. If validation failed, record changes_requested with the exact failing command and output. Do not call any other tool.",
						required,
					),
				}); err != nil {
					return LoopResult{}, err
				}
				terminalToolCircleReminderSent = true
				identicalStreak = 0
				lastFingerprint = ""
				if llmCalls >= maxTurns {
					maxTurns++
				}
				continue
			}
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

		refreshCodeGraphAfterBatch := false
		reviewTerminalEvidencePromptAfterBatch := ""
		stopAfterToolBatch := false
		for _, tc := range calls {
			if stopAfterToolBatch {
				if err := traceAppend(p, &messages, defs, llm.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    "skipped: terminal tool already completed this job",
				}); err != nil {
					return LoopResult{}, err
				}
				continue
			}
			execErr, traceErr := executeLoopToolCall(ctx, p, &messages, defs, tc, &toolInvocations)
			if traceErr != nil {
				return LoopResult{}, traceErr
			}
			if required := strings.TrimSpace(p.RequiredTerminalTool); required != "" && strings.TrimSpace(tc.Function.Name) != required && !terminalToolEvidenceReminderSent && tools.ReviewTerminalEvidenceSatisfied(p.Root, p.Executor.Session) {
				tools.MarkReviewTerminalDispositionRequired(p.Executor.Session)
				reviewTerminalEvidencePromptAfterBatch = reviewTerminalEvidencePrompt(required, tools.ReviewTerminalDispositionGuidance(p.Root, p.Executor.Session))
				terminalToolEvidenceReminderSent = true
			}
			if execErr == nil {
				refreshCodeGraphAfterBatch = refreshCodeGraphAfterBatch || shouldRefreshCodeGraphAfterMutation(p, tc.Function.Name, tc.Function.Arguments)
			}
			if execErr == nil && p.Executor.StopAfterTool != nil && p.Executor.StopAfterTool() {
				stopAfterToolBatch = true
			}
		}
		if stopAfterToolBatch {
			res = finish(messages, defs, EndCompleted, llmCalls, toolInvocations, start, "")
			return res, nil
		}
		if refreshCodeGraphAfterBatch {
			if err := refreshCodeGraph(ctx, p, &messages, defs, &toolInvocations); err != nil {
				return LoopResult{}, err
			}
		}
		if reviewTerminalEvidencePromptAfterBatch != "" {
			if err := traceAppend(p, &messages, defs, llm.Message{
				Role:    "user",
				Content: reviewTerminalEvidencePromptAfterBatch,
			}); err != nil {
				return LoopResult{}, err
			}
		}
	}
}

func copyToolCounts(ex *tools.Executor) map[string]int {
	if ex == nil || ex.Session == nil || len(ex.Session.ToolCounts) == 0 {
		return nil
	}
	out := make(map[string]int, len(ex.Session.ToolCounts))
	for key, value := range ex.Session.ToolCounts {
		out[key] = value
	}
	return out
}

func toolCallNamesForPrompt(calls []llm.ToolCall) string {
	if len(calls) == 0 {
		return "no terminal tool"
	}
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		name := strings.TrimSpace(call.Function.Name)
		if name == "" {
			name = "an unnamed tool"
		}
		names = append(names, "`"+name+"`")
	}
	return strings.Join(names, ", ")
}

func reviewTerminalEvidenceAllowsReportTool(guidance string, calls []llm.ToolCall) bool {
	if len(calls) == 0 {
		return false
	}
	reportMissing := strings.Contains(guidance, "file_write path")
	for _, call := range calls {
		switch strings.TrimSpace(call.Function.Name) {
		case "file_write":
			if !reportMissing {
				return false
			}
		case "git_status", "git_commit":
		default:
			return false
		}
	}
	return true
}

func reviewTerminalEvidencePrompt(required, guidance string) string {
	guidance = strings.TrimSpace(guidance)
	if strings.Contains(guidance, "file_write path") {
		return "Review evidence is sufficient, but required report evidence is missing. " + guidance + " Do not call unrelated tools or finish in prose."
	}
	return fmt.Sprintf(
		"Review evidence is sufficient for a terminal decision. Stop inspection and validation now. Call `%s` in the next response only. %s Do not call any other tool.",
		required,
		guidance,
	)
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
		var ctxErr *llm.ContextSizeError
		if errors.As(err, &ctxErr) {
			// Retrying the identical over-window request can never succeed;
			// surface immediately so the loop can prune and rebuild (AD-288).
			return llm.ChatCompletionResponse{}, err
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
