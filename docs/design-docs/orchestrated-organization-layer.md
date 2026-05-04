# Orchestrated Organization Layer

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** Mars Harness maintainers
**Related:** [delivery operating model](delivery-operating-model.md), [tools glossary](tools-glossary.md), [dashboard](dashboard.md), [conversation-as-system-record](conversation-as-system-record.md)

## Purpose

Mars Harness originally executed the delivery model as a mostly linear chain:

`CEO -> CTO -> COO -> Engineer -> QA -> Security -> Dependency Manager`

with scheduled or parallel roles such as Release Manager, Dogfood, Pipeline
Fixer, and Janitor. That chain was simple and inspectable, but it assumed the
previous role's output was good enough to hand forward. Real delivery needs
backward motion: QA can ask Engineer for changes, Engineer can ask CTO for an
architecture decision, Security can block release, and Janitor can wake a stale
ticket without pretending the next linear role is always right.

The organization layer adds a dispatch coordinator around the existing queue,
roles, tools, tickets, traces, scores, and dashboard. It is not a second
scheduler and it does not replace repo-owned tickets, goals, exec plans, or BDD
evidence.

The corrected foundation-agent ownership spine is now:

`CEO -> COO -> CTO -> Engineer -> QA -> Security -> Dependency Manager -> Release Manager`

with Orchestrator between every active role.

## Decision

Add a manifest field:

```yaml
orchestration_mode: legacy | dispatch
```

`dispatch` is the generated default. Each role records a terminal disposition,
returns that disposition to the configured `orchestrator` role, and the
Orchestrator chooses the next best manifest role or stops. `legacy` remains
supported for existing repos that deliberately want `then` and `idle_then`
runtime behavior.

## Runtime Contract

In dispatch mode, successful roles must record a job disposition before they
complete. The executor rejects a dispatch-mode job that finishes without a
recorded disposition.

Supported dispositions:

- `completed`: work is done and normal review or next-need routing may proceed.
- `blocked`: the job found a blocker and normal downstream review must not run.
- `in_review`: route to the named reviewer role, defaulting to QA.
- `changes_requested`: route back to Engineer with review context.
- `no_work`: preserve the existing idle bridge semantics for Engineer; other
  roles stop unless they name a next need.
- `approved`: route Engineer to finish or release the work when appropriate.
- `ambiguous`: route to Orchestrator when configured, then Janitor as the safe
  fallback.
- `failed`: synthesized by the server on failed jobs so dispatch mode can
  recover through the same routing path.

Roles record dispositions through `job_disposition_record`, a built-in tool
that writes SQLite state scoped to the current job, repo, and role. The tool is
registered like other built-in tools and is trust-gated by the role allowlist.
Dispositions may also carry structured `handoff` and `feedback` objects so the
next role gets an explicit ask and the prior role gets explicit correction
rather than implicit prose.

## Ticket State

Ticket docs remain the inspectable source of delivery truth:

- `docs/tickets/backlog/`
- `docs/tickets/in-progress/`
- `docs/tickets/in-review/`
- `docs/tickets/done/`

`blocked/` is deliberately not a directory in V1. Blocked work stays in
`in-progress/` with `blocker`, `blocked_by`, and `next_action` metadata so the
current ticket remains visible to Engineer, Janitor, Doctor, and the active
plan. This de-risks migration by extending the existing drain gate rather than
forking the ticket lifecycle into a second blocked queue.

Feature tickets still require BDD evidence before `done/`. Work products,
approvals, and dispositions are supporting liveness state, not substitutes for
BDD completion evidence.

## Data Model

`internal/orgstate` owns operational liveness tables in the existing per-repo or
serve database:

- `job_dispositions`: terminal status, next need, ticket, evidence links,
  approval/work-product references, trace, reason, structured handoff, and
  structured feedback for a job.
- `orchestration_decisions`: recorded dispatch choice, policy, input hash, stop
  reason, and loop-break evidence.
- `approvals`: lifecycle state for review requests and approvals.
- `work_products`: recorded deliverables, reports, release artifacts, dogfood
  outputs, and other evidence.
- `organization_repos`: repo membership metadata for future portfolio views.

Only dispositions and decisions are active in the first cut. The remaining
tables are intentionally present so approvals, work products, and portfolio
views can land without another database migration shape change.

## Routing Rules

The orchestration engine uses these rules:

- Non-Orchestrator dispositions route to the configured `orchestrator` role.
- Orchestrator dispositions honor `suggested_role` after validating that the
  role exists in the manifest.
- Strategy advisory needs (`strategy_advice`, `executive_narrative`,
  `tradeoff_analysis`, and `goal_conflict`) route to the optional
  `head-of-strategy` role when the manifest defines it, and otherwise fall back
  to CEO. The advisor can shape options and narrative, but CEO records the
  actual goal or vision decision.
- Goal, vision, and scope decisions route to CEO.
- Exec plans, planning, BDD feature contracts, scenario schedules, and current
  failing scenarios route to COO.
- Tickets, ticket shaping, ticket breakdown, technical tickets, implementation
  tickets, and architecture review route to CTO.
- Implementation routes to Engineer; QA and evidence review route to QA.
- Manifests without an `orchestrator` keep deterministic fallback routing for
  compatibility.
- Repeated identical role/ticket/need decisions on the same ticket-state hash
  route back to Orchestrator or stop with a loop-guard reason. If the repeated
  route already originated from Orchestrator, dispatch stops instead of
  enqueueing Orchestrator again.
- Generated role guidance resolves BDD feature IDs by `docs/features/F-NNN*.md`
  so slugged feature contracts count as present. Missing exact paths such as
  `docs/features/F-001.md` must not override an existing
  `docs/features/F-001-delivery-operating-model.md` contract.

The manifest remains the executable role registry. The engine never invents a
role that is absent from the repo manifest.

## Dashboard

The dashboard gains an Organization/Orchestration view backed by repo docs and
SQLite liveness state:

- registered repos and their orchestration mode
- hub-and-spoke role topology
- recent job dispositions
- recent dispatch decisions
- live SSE events for dispositions, dispatch decisions, and enqueued dispatch
  jobs

The Pipeline view renders dispatch repos as a hub-and-spoke Orchestrator view
and keeps the legacy chain rendering only for legacy repos.

## Tool Creation Rationale

`job_disposition_record` was added manually instead of first using
`tool_create`. This is an explicit exception because the handler depends on
executor-injected session state and raw disposition JSON, which would have
required immediately replacing almost all generated scaffold code. The exception
is documented here, the tool is registered in the normal built-in registry, and
the tools glossary and generated target guidance are updated in the same change.

Future org-layer tools should use `tool_create` unless they require comparable
executor or session integration that the scaffold cannot express.

## Assumptions And Mitigations

| Assumption | Risk | Mitigation |
| --- | --- | --- |
| Existing repos still need the linear chain. | Breaking deployed harnesses would violate plug-and-play. | `legacy` remains supported for existing manifests, while new generated harnesses default to dispatch. |
| Orchestrator on every handoff adds another job. | Extra latency compared with deterministic direct chaining. | The routing truth is clearer: roles return dispositions, Orchestrator owns next-role selection, and loop guards still prevent repeated churn. |
| Ticket docs and SQLite can diverge. | Dashboard or agents could trust stale liveness state. | Ticket docs remain completion truth; SQLite records operational liveness. Janitor and dashboard surface stale/missing records instead of treating SQLite as a release gate. |
| Backward routing can create loops. | Engineer/QA or CTO/COO loops can burn queue capacity. | Decisions record ticket-state hashes; repeated identical decisions route to Orchestrator/Janitor or stop. |
| Blocked work needs a truthful home. | A new `blocked/` directory would split existing drain behavior. | V1 keeps blocked tickets in `in-progress/` with explicit blocker metadata. |
| Review and approval state can become another source of truth. | QA approvals could replace BDD evidence. | Approvals/work products support routing only; `done/` still requires ticket evidence fields. |
| Dispatch mode can be enabled before roles are updated. | Jobs could complete without dispositions and silently chain wrong. | Executor fails dispatch-mode successes that did not call `job_disposition_record`. |
| Role names differ by target repo. | Dispatch could enqueue nonexistent roles. | The engine validates every target role against the manifest and falls back only to configured roles. |
| Dashboard could become the system of record. | Operators may edit from the UI instead of repo docs. | V1 dashboard is read-only for org state and points at repo/SQLite state. |

## Migration Strategy

1. Keep existing target manifests on their declared mode during upgrade.
2. Generate new target manifests with `orchestration_mode: dispatch`.
3. Add `docs/tickets/in-review/` during init/upgrade and ticket listing.
4. Update role allowlists and prompts for dispatch-mode targets so terminal
   roles can call `job_disposition_record`.
5. Enable `orchestration_mode: dispatch` only after the target manifest has an
   Orchestrator fallback role and updated role prompts.
6. Use Janitor to reconcile stale checkouts, missing dispositions, unresolved
   approvals, blocked tickets without `next_action`, and work products without
   ticket links.

## Test Expectations

The dispatch layer must be covered by:

- org store CRUD and disposition validation
- manifest parsing for `orchestration_mode`
- Orchestrator-return routing, Orchestrator suggested-role routing, deterministic
  fallback routing, and loop-guard cases
- ticket gate behavior for `in-review`
- executor validation that dispatch-mode successes record dispositions
- chaining tests proving legacy mode remains unchanged and dispatch mode bypasses
  normal `then` chaining
- dashboard/API tests for the orchestration page and JSON endpoints
- scanner/init tests proving generated target guidance includes `in-review` and
  org-layer tool context

## Non-Goals

- Cross-repo ticket execution in V1.
- Replacing goals, exec plans, tickets, BDD evidence, queue, or role manifests.
- Making the dashboard a mutating org-management system.
- Treating Paperclip's company/employee/issue vocabulary as Mars vocabulary.
