---
id: MH-054
title: Build nonblocking dashboard read APIs and async command status
priority: high
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S014"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-051", "MH-053"]
trace_id: none
next_action: "Define dashboard read snapshots, typed unavailable states, event stream payloads, and async command-status storage."
dedupe_key: dashboard-control-plane:concurrent-read-model
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-051", "MH-053"]
---

# MH-054: Build nonblocking dashboard read APIs and async command status

## Context

The current dashboard shares runtime paths with the orchestrator. The next
dashboard must never make a UI request wait behind long-running model calls,
repo scans, release checks, queue worker critical sections, or orchestrator
processing locks.

## BDD Scenario IDs

- F-010-S014

## Affected Docs/Code Areas

- `internal/dashboard/`
- `internal/serve/`
- `internal/queue/`
- `internal/telemetry/`
- `internal/trace/`
- `internal/scoring/`
- `internal/trust/`
- `internal/models/`
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/design-docs/dashboard.md`

## Acceptance Criteria

- [ ] Dashboard read endpoints use cancellable request contexts and short timeouts.
- [ ] Read endpoints use snapshots, independent database reads, or event-derived state rather than orchestrator processing locks.
- [ ] Overview, Active Work, Preview, Roster, Models, DORA, telemetry, token usage, and command-status endpoints return typed unavailable states for partial data.
- [ ] Control actions return an asynchronous command id instead of blocking until completion.
- [ ] Command status is queryable and streamed to authenticated clients.
- [ ] Tests simulate active or blocked orchestrator work while dashboard reads continue to respond.

## Non-Goals

- Replacing SQLite.
- Rewriting queue ownership.
- Running multiple agents only for dashboard convenience.
- Hiding stale data as fresh success.

## Evidence Requirements

- Concurrency tests proving dashboard reads complete while a simulated long-running runtime path is active.
- Async command-status tests for accepted, running, succeeded, failed, cancelled, and unavailable states.
- Event stream tests for command updates and state changes.
- Timeout tests for slow data sources.

## Next Action

Inventory current dashboard data sources and choose the smallest read snapshot
that can power an authenticated Overview without touching worker execution.
