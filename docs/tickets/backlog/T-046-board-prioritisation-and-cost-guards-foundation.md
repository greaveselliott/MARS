---
id: T-046
title: Implement board prioritisation and cost guards foundation
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-013-S003"]
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "cto-weekly"
last_attempt: "TBD"
blocker: "none"
blocked_by: [T-045]
trace_id: "TBD"
next_action: "Scope and implement Plan 3 only: ready-ticket selection by active sprint, configured priority order, LexoRank, and age; dispatch cto-weekly; add DailyCap, interval floors, cost/turn telemetry, circuit breaker, and dashboard/log signals. Do not add Figma, PR delivery, or frontier model routing."
dedupe_key: "public-example"
source: docs/exec-plans/active/current-operating-plan.md
created: 2026-06-23
depends_on: [T-045]
---

# T-046: Implement Board Prioritisation And Cost Guards Foundation

## Context

Plan 2 mirrors scoped JIRA issues into local tickets without creating LLM work.
Plan 3 is the first slice that may turn mirrored backlog state into a harness
dispatch decision. Because board dispatch can spend paid gateway tokens in later
flows, the prioritisation selector and cost guards must ship together.

## Requirements

- Extend ticket parsing/selection to consume mirrored JIRA fields: `jira_key`, `jira_updated`, `jira_created`, `sprint`, `sprint_active`, `rank`, `jira_status`, and `epic`; reuse existing `priority` and `blocked_by`.
- Select only ready backlog tickets from the active sprint when no product ticket is already in progress.
- Order candidates by configured priority order, LexoRank, and age.
- Treat unknown priority as last.
- Skip unresolved blockers, non-ready statuses, and closed-sprint tickets.
- Dispatch exactly one `cto-weekly` scoping job for the winning ticket.
- Add DailyCap, interval floors, cost/turn telemetry, and a circuit breaker before any board-driven dispatch can loop.
- Surface paused or blocked board dispatch through logs, status/dashboard APIs, and validation evidence.

## Acceptance Criteria

- Selector tests cover active sprint, P1/P2/P3 ordering, unknown priority last, LexoRank ordering, age fallback, blockers, ready statuses, and closed-sprint exclusion.
- Survey/orchestrator tests prove only one `cto-weekly` job is enqueued for the selected ready ticket.
- No-config and `ceo-led` repos keep existing survey, scheduler, and startup behavior.
- DailyCap and interval-floor tests prove board-driven dispatch pauses without silently looping.
- Cost/turn telemetry and circuit-breaker state are visible in logs or status/dashboard output.
- Installed-binary validation records a clean board-driven target with mirrored tickets and one bounded dispatch decision.

## Non-Goals

- JIRA write-back.
- Frontier model endpoint or API-key routing.
- Figma context tools.
- Pull-request delivery.
