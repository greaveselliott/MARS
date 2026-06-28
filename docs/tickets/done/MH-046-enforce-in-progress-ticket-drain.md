---
id: MH-046
title: Enforce in-progress ticket drain
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["go test ./internal/tickets ./internal/tools ./internal/serve ./internal/scanner ./internal/doctor ./internal/roleregistry", "go test ./internal/docsconsistency", "go test ./..."]
verified_by: "go test"
owner: "task-runner"
last_attempt: "2026-05-03"
blocker: "none"
blocked_by: []
trace_id: "task-runner-2026-05-03"
next_action: "Move MH-047 next after release notes."
dedupe_key: "public-example"
source: Mars parity workstream E
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars"
  target: "ticket-queue"
  category: "active_ticket_accumulation"
  severity: "high"
  confidence: "high"
---

# MH-046: Enforce in-progress ticket drain

## Context

Mars's operating model treats in-progress tickets as the top of the queue.
Harness now surfaces that rule in context, but it still needs mechanical
enforcement, blocker metadata, stale-ticket detection, and bounded dogfood
ticket creation so autonomous runs cannot accumulate unfinished work silently.

## Requirements

- Add ticket metadata for owner, last attempt, blocker, blocked-by, trace ID,
  and next action.
- Enforce queue policy so eligible in-progress tickets are considered before
  backlog tickets.
- Prevent Engineer runs from opening new backlog tickets while eligible
  in-progress tickets remain.
- Require unfinished in-progress tickets to end as completed, returned to
  backlog with a blocker note, explicitly blocked by a dependency ticket, or
  guardrail-blocked with intervention debt.
- Add stale in-progress scanning and an Orchestrator trigger.
- Cap dogfood ticket creation by severity, grouping, and dedupe key.

## Affected Files

- `internal/serve/executor.go`
- `internal/scheduler/`
- `internal/scanner/`
- `internal/tools/ticket_create.go`
- `internal/doctor/`
- `docs/tickets/README.md`
- `internal/scanner/init.go`

## Acceptance Criteria

### Functional

- [x] Eligible in-progress tickets are selected before backlog tickets.
- [x] Engineer context cannot claim new backlog work while eligible in-progress
      tickets remain.
- [x] Unfinished in-progress tickets must record completion, blocker,
      dependency, or guardrail-blocked intervention debt.
- [x] Stale in-progress tickets create a scan finding or Orchestrator trigger.

### Edge cases and negative paths

- [x] Explicitly blocked tickets do not cause an infinite retry loop.
- [x] Dependency tickets are deduped and linked back to the blocked ticket.
- [x] Dogfood runs cannot create an unbounded number of tickets in one pass.

### Observability, docs, and regressions

- [x] Tests cover queue ordering, blocked outcomes, stale detection, and dogfood
      ticket caps.
- [x] Target ticket README generation mirrors the same drain rules.
- [x] Doctor output gives concrete remediation for stale in-progress work.

## Completion Notes

- Added `internal/tickets` as the shared ticket-state parser for eligibility,
  blocker, dependency, and stale-ticket detection.
- Updated Engineer tool policy and post-run gates to block ordinary backlog
  creation during eligible in-progress drain while allowing linked dependency
  and intervention-debt outcomes.
- Scanner now emits stale in-progress findings, creates deduped intervention
  debt for those findings, and `serve` enqueues Janitor through
  `ticket.stale_in_progress` when configured.
- Doctor now reports stale eligible in-progress tickets with concrete
  remediation.
