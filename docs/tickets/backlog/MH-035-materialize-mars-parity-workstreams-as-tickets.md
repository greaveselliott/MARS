---
id: MH-035
title: Materialize Mars parity workstreams as tickets
priority: high
complexity: medium
kind: intervention-debt
dedupe_key: "public-example"
source: mars-parity-supersession-plan.md
created: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "mars-parity"
  category: "plan_to_ticket_gap"
  severity: "high"
---

# MH-035: Materialize Mars parity workstreams as tickets

## Context

The Mars parity supersession plan names the right strategic work, including the
first ten implementation items, but most of that work does not yet exist as
normal backlog tickets. Agents should not have to execute a large strategic plan
directly.

## Requirements

- Convert the first ten implementation tickets listed in the Mars parity plan into backlog ticket files.
- Use durable dedupe keys so the work is not recreated repeatedly.
- Preserve dependencies between role model, role registry, active-plan hygiene, intervention debt, in-progress drain, quality score, orchestrator, remediation, and dogfood matrix work.
- Update the Mars parity plan to link to the new tickets.
- Keep ticket scope small enough for one coherent implementation slice.

## Affected Files

- `docs/exec-plans/backlog/mars-parity-supersession-plan.md`
- `docs/tickets/backlog/`
- `docs/tickets/README.md`

## Acceptance Criteria

### Functional

- [ ] Each first-ten Mars parity item has a corresponding backlog ticket.
- [ ] The Mars parity plan links to the ticket IDs.
- [ ] Tickets have acceptance criteria and affected files.
- [ ] Ticket priorities reflect the current operating plan.

### Edge cases and negative paths

- [ ] Existing tickets are reused or linked instead of duplicated.
- [ ] Broad workstreams are split into coherent implementation slices.

### Observability, docs, and regressions

- [ ] Ticket creation follows canonical backlog path and naming.
- [ ] No generated ticket contains unchecked assumptions that conflict with strict trunk.
