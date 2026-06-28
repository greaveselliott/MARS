# Agent Runtime

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** MARS contributors

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
- **2026-05-04 — Local function tags:** Some local reasoning models emit tool calls as `<function=name>` blocks with nested `<parameter=arg>` tags instead of JSON. The runtime normalizes those tags into ordered function calls before falling back to JSON parsing, so bootstrap agents do not falsely complete with zero tool invocations.
- **2026-05-19 — Required terminal tools:** Server jobs can name a required terminal tool such as `job_disposition_record`. When the model tries to finish with prose only, the loop appends one corrective user turn requiring that tool call; if the next completion still lacks the tool, normal executor completion validation fails the job.
- **2026-05-19 — Inline tool-call tags:** A `demo-123` QA replay showed the fast local model emitting `<tool_call>name{key:<|"|>value<|"|>}</tool_call>` text. The parser now normalizes that inline tag format, including string arrays and nested feedback objects, before falling back to JSON parsing so attempted tool calls execute instead of being misclassified as prose.
- **2026-05-21 — Required terminal tool circle grace:** A live QA replay showed a reviewer repeatedly calling the same blocked no-op shell command after successful validation instead of recording `job_disposition_record`. When a server job has a required terminal tool and the next assistant message would otherwise trip circle detection, the loop now adds one corrective user turn requiring only that terminal tool. If the next response still calls a non-terminal tool, the job ends with `circle_detected`.
- **2026-05-21 — Review evidence convergence:** A live Security replay showed clean read plus validation evidence followed by a long model turn instead of terminal disposition. QA and Security jobs now get a terminal-only reminder as soon as clean review evidence is sufficient, and the next response is capped by a short grace timeout. A later clean QA replay showed one missed terminal-tool instruction after a corrected validation command; the loop now rejects that first non-terminal response without executing it and gives one stronger terminal-only correction before repeated misses end with `circle_detected`. A `demo-temp-run56` replay then showed the terminal boundary firing before QA could satisfy its required `docsync_audit` check; review terminal convergence now waits for docsync evidence before forcing `job_disposition_record`. A `demo-temp-run57` replay confirmed docsync ordering but showed build-only evidence forcing QA terminal before tests; the convergence heuristic now also waits for a successful test command when test files exist.
- **2026-05-21 — Review no-op recovery:** A `demo-temp-run58` replay showed that blocked QA `shell_exec` no-op recovery still forced terminal disposition after build evidence even though tests had not run. The executor no longer marks terminal disposition as required for every post-validation no-op failure; the agent loop performs the evidence-aware terminal decision, and the no-op policy points reviewers to missing tests or docsync before approval guidance appears.
- **2026-05-23 — Model-wait progress visibility:** A live `demo-6`
  dispatch replay showed a `cto-weekly` job appearing idle while the
  non-streaming local model was still generating its first response. The loop's
  existing turn callback now lets the terminal dashboard mark that phase as
  `waiting for model response` with elapsed phase age, without changing
  sequential execution or trace semantics.
- **2026-05-21 — Structured CLI recovery:** A `demo-temp-run59` release replay showed Release Manager reading the `mars_cli` reference but repeating a stale `shell_exec mars release notes` command until loop containment fired. Shell policy now blocks direct `mars` binary invocations and gives the equivalent structured `mars_cli` args before a stale PATH binary can create a liveness loop.
- **2026-06-26 - OpenAI-compatible tool-call adjacency:** A fresh OpenAI-backed lifecycle replay showed that a model-provided multi-tool assistant message must be followed by tool responses for every original `tool_call_id` before the runtime appends synthetic code-index refreshes or terminal-evidence prompts. The loop now finishes the model tool-call batch first, records skipped tool responses when a terminal tool stops the job mid-batch, and only then appends runtime-injected follow-up messages.
