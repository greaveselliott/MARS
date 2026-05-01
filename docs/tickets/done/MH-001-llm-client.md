---
id: MH-001
title: Implement OpenAI-compatible LLM client
priority: high
complexity: medium
source: delivery-schedule M1.1
created: 2026-04-11
---

# MH-001: Implement OpenAI-compatible LLM client

## Context

The LLM client is the interface between the agent runtime and any inference provider (local llama.cpp, external vLLM, cloud API). It must speak the OpenAI `/v1/chat/completions` protocol with full tool calling support.

Reference: [docs/design-docs/agent-runtime.md](../../design-docs/agent-runtime.md) (AD-004, AD-005, AD-006)

## Requirements

Implement `internal/llm/client.go`:

- HTTP client for OpenAI-compatible `/v1/chat/completions`
- Message types: system, user, assistant, tool
- Tool definitions via JSON Schema
- Streaming (SSE) and non-streaming modes
- Connection error handling with retry and backoff
- Timeout configuration
- Rate limit detection (429 status, `Retry-After` header)
- Malformed response handling (invalid JSON, unexpected structure)
- Token counting (tiktoken-compatible estimation for budget enforcement)

## Affected Files

- `internal/llm/client.go` (new)
- `internal/llm/types.go` (new — message, tool call, completion response types)
- `internal/llm/client_test.go` (new)

## Acceptance Criteria

### Functional (happy path)
- [x] Client sends a chat completion request and receives a response
- [x] Tool definitions are included in the request when provided
- [x] Tool call responses are correctly parsed from the completion
- [x] Streaming mode receives and assembles SSE chunks
- [x] Token count estimation returns a reasonable number for a given message list

### Edge cases and negative paths
- [x] Connection timeout returns a descriptive error (not a raw HTTP error)
- [x] 429 rate limit triggers retry with backoff
- [x] 500 server error triggers retry (max 3 attempts)
- [x] Malformed JSON response returns a descriptive error, does not panic
- [x] Empty response body is handled gracefully
- [x] Response with tool calls containing malformed arguments is surfaced as an error (not swallowed)

### Non-goals
- This ticket does NOT implement the conversation loop (that's MH-003)
- This ticket does NOT implement the router (that's part of MH-008 in M2)
- This ticket does NOT handle model-specific quirks (that's MH-004)

### Observability, docs, and regressions
- [x] Unit tests cover all happy path and error scenarios (mock HTTP server)
- [x] Types are documented with Go doc comments
- [x] agent-runtime.md updated if any design decisions change during implementation

## Notes

Test with a mock HTTP server returning canned responses. Real model integration tested in M2.
