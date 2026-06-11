# Execution Plans

This directory holds execution plans with a ticket-like lifecycle:

- `backlog/` for prioritized plans that are not current
- `active/` for exactly one active plan
- `completed/` for finished plans
- `superseded/` for historical plans that must not drive current work

There must be only one active exec plan at a time. Promote work by updating
`active/current-operating-plan.md`, not by adding another active plan. Backlog
plans must carry `**Priority:**`, `**Depends On:**`, `**Blocks:**`, and
`**Related Tickets:**` metadata. Active and backlog plans also carry
`**Goals:**`, `**BDD Feature:**`, `**Hypothesis:**`, `**Success Evidence:**`,
`**Falsification Evidence:**`, `**Scenario Schedule:**`,
`**Current Failing Scenario:**`, `**Walking Skeleton Slice:**`, and
`**Learning Or MVP Outcome:**`. Plans should be ordered like tickets: the
active plan chooses the next work, backlog plans wait their turn, and superseded
plans are lineage only.

Plan priority never overrides dependencies. A P1 plan with an unresolved
dependency waits behind a lower-risk unblocked slice or creates the dependency
ticket first.

## Planning Order

Bootstrap and delivery order is strict:

1. Update `docs/exec-plans/active/current-operating-plan.md`.
2. Create or update the `docs/features/F-NNN-*.md` contract named by the plan.
3. Create tickets from the current failing scenario or scenario group.
4. Deliver one ticket with evidence.

In shorthand: exec plan, feature contract, ticket, delivery.

Feature contracts, tickets, and implementation work without an active plan
pointer are drift. Repair the exec plan first.

## Plan Metadata

Active and backlog plans require:

- `**Status:**` — `Active` or `Backlog`
- `**Priority:**` — `P0`, `P1`, `P2`, `P3`, or `P4`
- `**Depends On:**` — tickets, plan paths, checks, or `None`
- `**Blocks:**` — tickets, claims, releases, plan paths, or `Nothing`
- `**Related Tickets:**` — ticket IDs or `None yet`
- `**Goals:**` — active goal IDs this plan advances
- `**BDD Feature:**` — feature contract IDs this plan schedules
- `**Hypothesis:**` — why this work advances active goals
- `**Success Evidence:**` — what validates or closes the plan
- `**Falsification Evidence:**` — what proves the plan wrong or low-value
- `**Scenario Schedule:**` — ordered failing BDD scenarios or scenario groups
- `**Current Failing Scenario:**` — the next scenario or blocked reason
- `**Walking Skeleton Slice:**` — the thinnest real E2E path for the current scenario
- `**Learning Or MVP Outcome:**` — value or learning the slice should produce

## BDD-Led Planning Rules

- BDD defines the full feature. Walking skeleton is the implementation strategy.
- All business logic must be documented step by step in `docs/features/`,
  including rules, branches, state transitions, validations, permissions,
  scoring/trust behavior, routing behavior, and user-visible outcomes.
- No stale documentation: implementation slices identify associated docs with
  top-of-file `MarsDocSync` metadata, and plans or tickets record whether those
  docs were updated or explicitly checked as current.
- The active plan schedule is the ordered list of failing BDD scenarios.
- Feature tickets are created only from the current failing scenario or scenario group.
- A feature is not shipped until in-scope BDD scenarios pass or are explicitly descoped by the CEO.

## Plan Hygiene

Run `go test ./internal/docsconsistency/...` or `mars-harness doctor --repo .`
after changing plan state. The active-plan hygiene check reports actionable
warnings when plan lifecycle state drifts from the ticket tree.

- Supersede a plan by moving it to `superseded/`, setting
  `**Status:** Superseded`, and adding a visible pointer to
  `docs/exec-plans/active/current-operating-plan.md`.
- Complete a plan by moving it to `completed/` with `**Status:** Completed`;
  completed historical plans are not active-plan failures.
- Reconcile a stale active plan by updating ticket-state claims after moving
  tickets between `backlog/`, `in-progress/`, and `done/`.
- Replace `TBD`, relative status language such as `latest` or `currently`, and
  stale verification notes with absolute dates, concrete blockers, or durable
  source-of-truth pointers.

## Standing trackers

| File | Purpose |
|------|---------|
| [tech-debt.md](tech-debt.md) | Known debt and acceptable gaps. |

The former `pipeline-learnings.md` tracker was retired on 2026-06-11 (AD-276):
recurring failure learnings live as `delivery-operating-model.md` architecture
decisions, `docs/validation/` reports, and deterministic remediation recipes.

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
