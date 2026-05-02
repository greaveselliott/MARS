# Execution Plans

This directory holds execution plans with a ticket-like lifecycle:

- `backlog/` for prioritized plans that are not current
- `active/` for exactly one active plan
- `completed/` for finished plans
- `superseded/` for historical plans that must not drive current work

There must be only one active exec plan at a time. Promote work by updating
`active/current-operating-plan.md`, not by adding another active plan. Backlog
plans must carry `**Priority:**`, `**Depends On:**`, `**Blocks:**`, and
`**Related Tickets:**` metadata and should be ordered like tickets: the active
plan chooses the next work, backlog plans wait their turn, and superseded plans
are lineage only.

Plan priority never overrides dependencies. A P1 plan with an unresolved
dependency waits behind a lower-risk unblocked slice or creates the dependency
ticket first.

## Plan Metadata

Active and backlog plans require:

- `**Status:**` — `Active` or `Backlog`
- `**Priority:**` — `P0`, `P1`, `P2`, `P3`, or `P4`
- `**Depends On:**` — tickets, plan paths, checks, or `None`
- `**Blocks:**` — tickets, claims, releases, plan paths, or `Nothing`
- `**Related Tickets:**` — ticket IDs or `None yet`

## Standing trackers

| File | Purpose |
|------|---------|
| [tech-debt.md](tech-debt.md) | Known debt and acceptable gaps. |
| [pipeline-learnings.md](pipeline-learnings.md) | Recurring failure patterns and fix recipes (populated during operation). |

## Where to look next

- **Current operating plan:** [active/current-operating-plan.md](active/current-operating-plan.md) — first read for current execution state, priority order, and plan hygiene
- **Mars parity supersession plan:** [backlog/mars-parity-supersession-plan.md](backlog/mars-parity-supersession-plan.md) — P1 backlog plan for making Mars Harness supersede the Mars meta-harness
- **Model evaluation refresh plan:** [backlog/model-evaluation-refresh-plan.md](backlog/model-evaluation-refresh-plan.md) — P4 backlog plan for keeping local/remote model defaults current
- **Master execution plan:** [superseded/master-execution-plan.md](superseded/master-execution-plan.md) — superseded baseline checklist covering M0–M10 + MH-001–MH-028; do not use checkbox status as current truth
- **Delivery schedule:** [superseded/delivery-schedule.md](superseded/delivery-schedule.md) — superseded milestone schedule; kept for lineage and reconciliation only
- **Design decisions:** [docs/design-docs/index.md](../design-docs/index.md)
- **Active work:** [docs/exec-plans/active/](active/)
- **Plan backlog:** [docs/exec-plans/backlog/](backlog/)
- **Shipped plans:** [docs/exec-plans/completed/](completed/)
- **Superseded plans:** [docs/exec-plans/superseded/](superseded/)
- **Product specs:** [docs/product-specs/index.md](../product-specs/index.md)
