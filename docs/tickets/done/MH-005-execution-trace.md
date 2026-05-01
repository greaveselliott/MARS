---
id: MH-005
title: Implement execution trace recorder
priority: medium
complexity: medium
source: delivery-schedule M1.5
created: 2026-04-11
---

# MH-005: Implement execution trace recorder

## Context

Every job's full LLM conversation must be recorded for auditability (tenet 7), debugging, and self-improvement (tenet 2). The trace is the raw material the Reviewer meta-role analyses when a job fails.

Reference: [docs/design-docs/agent-runtime.md](../../design-docs/agent-runtime.md), tenet 7 (Execution Truth)

## Requirements

Implement `internal/trace/recorder.go`:

- Record each turn of the conversation: role, content, tool calls, tool results, timestamp, token count
- JSON Lines format (one JSON object per turn) for streaming and incremental writes
- Summary generation: extract outcome, total duration, total tokens, tools called, number of turns
- SQLite storage: write trace to a `traces` table (trace_id, job_id, turns JSONL, summary JSON, created_at)
- File export: optionally write trace to a file for debugging (`mars-harness run --trace-file`)

## Affected Files

- `internal/trace/recorder.go`, `recorder_test.go`
- `internal/trace/types.go` (turn, summary types)
- `internal/trace/store.go`, `store_test.go` (SQLite persistence)

## Acceptance Criteria

### Functional (happy path)
- [x] Recorder captures all turns of a multi-turn conversation
- [x] Each turn includes: role, content, tool calls (if any), tool results (if any), timestamp, token estimate
- [x] Summary correctly computes: total duration, total tokens, tool list, turn count, outcome
- [x] Trace persists to SQLite and can be retrieved by job_id
- [x] File export writes valid JSON Lines

### Edge cases and negative paths
- [x] Very long tool result (>100KB) is truncated in the trace with a note
- [x] SQLite write failure returns error, does not crash the job
- [x] Retrieving a non-existent trace returns nil, not error

### Non-goals
- Dashboard rendering of traces (M9)
- Terminal trace output (MH-008 in M3)
- Trace retention/cleanup policy (M9 or later)

### Observability, docs, and regressions
- [x] Test records a 5-turn conversation and verifies summary accuracy
- [x] Test verifies SQLite round-trip (write and read back)

## Notes

The SQLite schema defined here will be extended in later milestones (M5 for jobs, M6 for scores). Design the schema to be additive — migrations via versioned SQL files or Go code.
