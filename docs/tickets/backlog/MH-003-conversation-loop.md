---
id: MH-003
title: Implement agent conversation loop
priority: high
complexity: large
source: delivery-schedule M1.3
created: 2026-04-11
---

# MH-003: Implement agent conversation loop

## Context

The conversation loop is the core of the product. It takes a system prompt and tools, calls the LLM, executes tool calls, feeds results back, and repeats until done or budget exhausted. This is where the "agent" behavior lives.

Reference: [docs/design-docs/agent-runtime.md](../../design-docs/agent-runtime.md) (AD-004, AD-005)

## Requirements

Implement `internal/agent/loop.go`:

### Loop logic
1. Assemble system prompt (role prompt + guardrails + context) via context assembler
2. Send to LLM with tool definitions
3. If response contains tool calls: execute each sequentially, append tool results, send updated conversation to LLM, repeat
4. If response is text only (no tool calls): loop complete
5. Budget check after each turn (tokens used, wall time, tool call count)
6. Max turns limit (default 50, configurable)

### Error handling
- Tool call execution failure: append error as tool result, let model retry
- LLM returns malformed tool call: log, skip, append error message, let model retry
- LLM goes in circles (same tool call with same arguments 3 times): force-end with diagnostic
- Budget exceeded (tokens, time, or tool calls): force-end, record reason
- LLM refuses to proceed (empty response): force-end, record
- Connection error to LLM: retry with backoff (max 3), then force-end

### Robust tool call parser (scrutiny fix M1.3.6)
- Handle partial JSON from streaming responses
- Handle model-specific format quirks (boolean corruption, string truncation)
- Fallback: if streaming tool calls fail to parse, retry the same turn with streaming disabled
- Test against recordings of real llama.cpp output (capture real responses, replay in tests)

## Affected Files

- `internal/agent/loop.go`, `loop_test.go`
- `internal/agent/parser.go`, `parser_test.go` (robust tool call parser)
- `internal/agent/types.go` (job config, loop result, budget)
- `internal/agent/testdata/` (recorded real model outputs for parser tests)

## Acceptance Criteria

### Functional (happy path)
- [ ] Loop completes a multi-turn conversation (3+ tool calls) with mock LLM
- [ ] Each tool call is executed and result appended to conversation
- [ ] Loop terminates when LLM responds with text only (no more tool calls)
- [ ] Full conversation history is available after loop completes

### Edge cases and negative paths
- [ ] Malformed tool call JSON is handled without panic; error appended to conversation
- [ ] Circle detection: same tool+args 3 times → loop terminates with diagnostic
- [ ] Token budget exceeded → loop terminates cleanly with "budget_exceeded" reason
- [ ] Wall time budget exceeded → loop terminates with "timeout" reason
- [ ] Max turns exceeded → loop terminates with "max_turns" reason
- [ ] Empty LLM response → loop terminates with "empty_response" reason
- [ ] LLM connection failure after retries → loop terminates with "llm_unreachable" reason
- [ ] Tool call parser handles boolean corruption (true → "true" string)
- [ ] Tool call parser handles truncated strings with internal quotes

### Non-goals
- [ ] Context assembly is a separate ticket (MH-004)
- [ ] Trace recording is a separate ticket (MH-005)
- [ ] Real model integration is M2 (this uses mock LLM)

### Observability, docs, and regressions
- [ ] Integration test: mock LLM → loop → real file tools → file created in temp dir
- [ ] Parser tests use recorded real model output (not just synthetic mocks)
- [ ] All termination reasons are tested
- [ ] agent-runtime.md updated with any design changes discovered during implementation

## Notes

The parser is the highest-risk component. Real local models produce messier output than mocks. Task 1.3.6 (robust parser) should be implemented before the happy-path loop, not after, so integration tests catch format issues early.
