# Agent Runtime

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

Execution model for harness roles: how a job runs the model, assembles context, parses tool calls, and advances the conversation until completion or failure.

## Context

The harness runs autonomous work in discrete **jobs**. Each job must be deterministic to trace, safe under partial failures, and compatible with local models that may emit imperfect structured output. This document scopes the **M1** agent loop; concurrency, scheduling, and persistence live in [pipeline-engine.md](pipeline-engine.md).

Traces produced here feed the **scoring** and **self-improvement** pipelines. A linear, turn-by-turn record simplifies debugging and Reviewer analysis, which motivates the synchronous, sequential choices below.

## Key Design Decisions

### AD-004: Synchronous single-threaded agent loop per job

Concurrency is expressed at the **job** level (multiple jobs may run in parallel across workers), not within a single job. One turn completes before the next begins; no intra-job goroutine fan-out for model steps.

Rationale: avoids races on shared job state (working tree, pending tool results, partial messages) and keeps cancellation and timeout semantics obvious.

### AD-005: Sequential tool execution

Tools run **one at a time** in declaration order after the model proposes them. Parallel tool calls are out of scope for v1: sequential execution is simpler, safer for filesystem and Git state, and produces linear traces for scoring and Reviewer analysis.

If the model emits multiple tool calls in one assistant message, the runtime defines a **canonical order** (documented in code) and executes strictly in that order.

### AD-006: Additive context assembly

Context is built by **concatenation** of named sections with explicit headers (system, role, retrieved snippets, tool results, etc.). No template engines or macro languages in the hot path—reduces injection surface and keeps assembly auditable in logs.

Section order is stable and versioned so replay and tests can diff context reliably.

### M1 topics (to refine during implementation)

- **Multi-turn conversation structure:** message roles, where tool results attach, and how history is trimmed under budget; preserve tool-call/result pairing for models that require it.
- **Tool call parsing:** robust parser for local-model quirks (partial JSON, markdown fences, extra prose); explicit **parse error** path that feeds back to the model once with a structured hint.
- **Error handling:** malformed calls, repeated no-op turns (**circles** detection), tool failures, and **budget exceeded** (tokens, wall time, max turns)—each with a terminal job state and user-visible reason code.
- **Max turns limit:** hard cap per job; when hit, persist partial trace and surface “max turns exceeded” for operators and scoring.

## Discoveries

- **2026-04-11 — MH-003 loop:** The synchronous loop enforces `MaxToolCalls` **before** each new LLM round so the transcript never contains a fresh `assistant` message with `tool_calls` that would lack paired `tool` results. Wall-clock budget is checked at the start of each iteration and immediately after each LLM completion returns (so slow providers cannot exceed wall time silently). Circle detection compares a fingerprint of the ordered `(tool_name, compacted_arguments_json)` list and aborts on the third consecutive identical proposal (tools from that turn are not executed).
- **2026-04-11 — Tool text vs JSON:** When the API returns assistant `content` without structured `tool_calls`, the parser only surfaces a **hard parse error** if the text looks like attempted tool JSON (starts with `{`/`[` or contains a markdown fence). Plain prose final answers (for example `done.`) are treated as **no tool calls** so the loop can terminate without an extra LLM round.
- **2026-04-11 — MH-005 trace:** `agent.Params` accepts an optional `*trace.Recorder` plus `*trace.Store`. The loop writes a JSONL header, mirrors every appended message (truncating large tool text to 100KiB), and on exit builds a `trace.Summary` persisted to SQLite (`traces` table) when a store is configured.
