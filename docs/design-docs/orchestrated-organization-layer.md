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

The corrected foundation-agent ownership spine for product delivery is now:

`CEO -> COO -> CTO -> Engineer -> QA -> Security -> Dogfood -> Release Manager`

with Orchestrator reserved for ambiguous or governance-heavy handoffs rather
than inserted between every deterministic role. Dependency Manager remains a
scheduled or explicit maintenance owner rather than the default fresh-product
handoff.

## Decision

Add a manifest field:

```yaml
orchestration_mode: legacy | dispatch
```

`dispatch` is the generated default. Each role records a terminal disposition.
The server routes completed, approved, in-review, and no-work dispositions
directly when the next role is deterministic from `suggested_role`,
`handoff.target_role`, `feedback.for_role`, `next_need`, or the product
validation spine. Orchestrator remains the fallback for ambiguous, blocked,
failed, conflicting, or governance-heavy handoffs. `legacy` remains supported
for existing repos that deliberately want `then` and `idle_then` runtime
behavior.

A `next_need` that resolves to the same role that just completed is not a
same-role handoff loop. If the role recorded completed work and has a default
forward owner, direct dispatch moves to that owner; for example, a completed COO
planning job that still names `feature_contract` routes to CTO ticket
breakdown. If the role recorded no-work, direct dispatch stops with a same-role
reason unless the role provides an explicit structured target for another
owner. This keeps roles such as COO from immediately enqueueing another COO job
while still allowing completed planning work to continue toward implementation.
Review roles are the exception: if QA, Security, or Dogfood completes and names
its own review category, dispatch treats that as completed review evidence and
routes to the next configured review or release owner.

When a review role records `changes_requested` with
`next_need: implementation_rework` and a ticket ID, that ticket remains the
rework identity until Engineer resolves, blocks, or an explicit Orchestrator
override replaces it. If Orchestrator is inserted between the reviewer and
Engineer, the final Engineer trigger preserves the original review
`source_disposition` rather than replacing it with Orchestrator's summary, so
tool policy can enforce the dispatch-named ticket and fail closed when the
ticket is missing.

## Runtime Contract

In dispatch mode, successful roles must record a job disposition before they
complete. `job_disposition_record` is terminal: after the tool succeeds, the
agent loop stops instead of letting the model continue to spend turns after it
has already declared the job outcome. The executor rejects a dispatch-mode job
that finishes without a recorded disposition.

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
rather than implicit prose. The recorder accepts strict arrays plus simple
list-as-string shapes for evidence and handoff fields before validation, so
local-model formatting drift does not prevent durable terminal handoffs.

Dispatch handoff jobs outrank scheduled cron jobs when the queue claims work.
Cron schedules remain a safety net for unattended operation, but they must not
preempt an active bootstrap or product handoff chain.

Dispatch jobs carry a typed trigger payload. The payload includes the source
role, source job, orchestration decision, selected target role, and a
routing-safe `source_disposition` containing status, next need, ticket ID,
reason, evidence links, trace ID, handoff, and feedback. Direct deterministic
handoffs pass that packet to the selected target role. Ambiguous handoffs pass
it to Orchestrator first, where Orchestrator translates it into a cleaned target
handoff and records its own disposition before the chosen role runs.

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

Engineer dispatch is ticket-backed, but review rework reuses the ticket under
review. A fresh implementation handoff to Engineer requires an ordinary product
ticket in backlog or in progress. If QA or another reviewer records
`changes_requested` for an existing ordinary product ticket and the next need is
implementation rework, dispatch may route Engineer to the same ticket even when
the ticket currently lives in `done/` or `in-review/`. The runtime must not send
that case back to CTO for a duplicate ticket unless the review explicitly asks
for ticket breakdown instead of rework. When Engineer starts with ordinary
backlog product work and no in-progress ticket, `shell_exec` is claim-first:
the backlog-to-in-progress `git mv` is the only allowed shell command until
visible ticket ownership exists.

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

- Completed, approved, in-review, and no-work non-Orchestrator dispositions
  route directly when the target role is deterministic from `suggested_role`,
  `handoff.target_role`, `feedback.for_role`, `next_need`, or the default
  product validation spine. Ambiguous, blocked, failed, or conflicting
  dispositions route to the configured `orchestrator` role.
- Orchestrator dispositions honor `suggested_role`, then
  `handoff.target_role`, then `feedback.for_role`, after validating that the
  selected role exists in the manifest and that structured target fields do not
  conflict.
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
- During fresh bootstrap or an empty ordinary product backlog, CTO routing is a
  bounded ticket-shaping stop: create or confirm one current-scenario
  implementation ticket, record implementation as the next need, and return to
  Orchestrator before any broad governance or audit work.
- Implementation routes to Engineer; QA and evidence review route to QA.
- Engineer implementation dispatch requires an ordinary product ticket in
  `docs/tickets/backlog/` or `docs/tickets/in-progress/`. If Orchestrator
  selects Engineer while no open product ticket exists, the runtime rewrites
  the dispatch to `cto-weekly` for ticket shaping instead of allowing
  free-floating implementation work.
- When no open product ticket remains after a completed Engineer source
  disposition, the next dispatch is QA review before any further CTO planning
  or implementation handoff, even if Orchestrator selects ticket shaping.
- Engineer ticket-gate failures are repaired by one bounded Engineer
  `ticket_gate_repair` job that carries the gate error in its trigger. A repair
  job that fails the gate again stops instead of routing through Orchestrator or
  enqueueing another repair.
- Dispatch protocol completion is backed by the agent loop: when a dispatch job
  tries to finish with prose only, the runtime gives one corrective prompt for
  the required `job_disposition_record` tool call instead of ending the job
  immediately. If the role still finishes without a disposition, the
  deterministic protocol failure is recorded as telemetry and stops instead of
  routing through Orchestrator.
- QA liveness blocks caused by missing trigger-provided source context do not
  route backward into CTO, COO, CEO, or Janitor. The repo is the review context:
  dispatch retries QA with a repository-inspection handoff so it reads the
  ticket, recent commits, and named implementation files before blocking.
- Pending Engineer survey jobs for in-progress tickets are cancelled when a
  successful Engineer completion leaves none of their referenced tickets
  eligible in `docs/tickets/in-progress/`.
- Ticket-owner survey routing pauses after a recent same-role runtime failure,
  including `max_turns`, so the survey watchdog does not immediately retry the
  same eligible in-progress ticket that failure handling deliberately left as
  foundation telemetry.
- Approved or completed QA, Security, and Dogfood handoffs move forward through
  the product validation and versioning chain. QA routes to Security, Security
  routes to Dogfood when that role exists or Release Manager when it does not,
  and Dogfood routes to Release Manager when configured. Dependency Manager
  remains routable when a role explicitly asks for dependency work, but it is
  not the default fresh-product handoff.
- The completed-Engineer/no-open-ticket guard is pre-review only. It can rewrite
  Orchestrator fallback decisions to QA before review, but after QA approves
  with a forward validation need the runtime honors QA's current disposition and
  does not loop the same ticket back into QA because of stale Engineer trigger
  context.
- Manifests without an `orchestrator` keep deterministic fallback routing for
  compatibility.
- Repeated identical role/ticket/need decisions on the same ticket-state hash
  route back to Orchestrator or stop with a loop-guard reason. If the repeated
  route already originated from Orchestrator, dispatch stops instead of
  enqueueing Orchestrator again.
- If an Orchestrator job itself fails before recording a disposition, the
  runtime never routes that failed Orchestrator disposition back into
  Orchestrator. When the dispatch trigger still carries a non-Orchestrator
  source disposition with a deterministic routing signal, the runtime falls
  forward to that target role using the original source handoff. When the
  source handoff is missing, ambiguous, or would select Orchestrator again,
  dispatch records a stopped decision and leaves one operator-visible blocker.
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
| Orchestrator on every handoff adds another job. | Extra latency compared with deterministic direct chaining. | Deterministic product handoffs route directly; Orchestrator remains the fallback for ambiguous, blocked, failed, conflicting, or governance-heavy handoffs. |
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
- direct deterministic role routing, Orchestrator-return routing, Orchestrator
  suggested-role routing, deterministic fallback routing, and loop-guard cases
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
