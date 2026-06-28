---
id: MH-033
title: Reconcile active exec plan state
priority: high
complexity: small
kind: intervention-debt
dedupe_key: "public-example"
source: exec-plan review 2026-05-02
created: 2026-05-02
completed: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars"
  target: "exec-plans"
  category: "stale_plan_state"
  severity: "high"
---

# MH-033: Reconcile active exec plan state

## Context

The ticket tree shows the original MH-001 through MH-029 baseline is complete,
but `master-execution-plan.md` and `delivery-schedule.md` still present many of
those same milestones as partial or not started. That makes the repo-owned plan
surface misleading for future agents.

## Requirements

- Add a current operating plan that reflects actual ticket and release state.
- Mark stale baseline plans as superseded for status purposes.
- Update the exec-plan README so agents know which plan to read first.
- Create follow-up backlog work for mechanical plan hygiene checks.

## Affected Files

- `docs/exec-plans/README.md`
- `docs/exec-plans/active/current-operating-plan.md`
- `docs/exec-plans/superseded/master-execution-plan.md`
- `docs/exec-plans/superseded/delivery-schedule.md`
- `docs/tickets/backlog/`
- `docs/design-docs/self-improvement.md`

## Acceptance Criteria

### Functional

- [x] Current operating plan exists and names the current priority order.
- [x] Superseded baseline plans point to the current operating plan.
- [x] Exec-plan README names the current operating plan as the first read.
- [x] Follow-up tickets exist for mechanical active-plan hygiene and Mars parity ticket materialization.

### Edge cases and negative paths

- [x] Historical baseline plans remain available for lineage.
- [x] The plan update does not mark unfinished parity/model/release work as complete.

### Observability, docs, and regressions

- [x] Self-improvement docs record active-plan drift as a durable improvement signal.
- [x] No unchecked acceptance criteria remain in this completed ticket.
