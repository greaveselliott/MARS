---
id: MH-046
title: Enforce in-progress ticket drain
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: TBD
dedupe_key: "public-example"
source: Mars parity workstream E
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
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

- [ ] Eligible in-progress tickets are selected before backlog tickets.
- [ ] Engineer context cannot claim new backlog work while eligible in-progress
      tickets remain.
- [ ] Unfinished in-progress tickets must record completion, blocker,
      dependency, or guardrail-blocked intervention debt.
- [ ] Stale in-progress tickets create a scan finding or Orchestrator trigger.

### Edge cases and negative paths

- [ ] Explicitly blocked tickets do not cause an infinite retry loop.
- [ ] Dependency tickets are deduped and linked back to the blocked ticket.
- [ ] Dogfood runs cannot create an unbounded number of tickets in one pass.

### Observability, docs, and regressions

- [ ] Tests cover queue ordering, blocked outcomes, stale detection, and dogfood
      ticket caps.
- [ ] Target ticket README generation mirrors the same drain rules.
- [ ] Doctor output gives concrete remediation for stale in-progress work.
