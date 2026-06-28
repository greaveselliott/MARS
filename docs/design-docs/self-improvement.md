# Self-Improvement Loop

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** MARS contributors

Closed-loop evolution: detect when humans or the platform had to compensate, classify root cause, propose or commit safe bounded changes, and measure before/after impact.

## Context

Self-improvement is a core tenet but must not destabilize repos or the harness itself. Signals include interventions, failed jobs, and score regressions. The **Reviewer** meta-role consumes traces; evolution must respect **safety rails** and acknowledge limits of models reviewing models.

Evolution outputs are **suggestions** until an autonomous trust policy allows direct trunk evolution. Human-triggered contributor runs remain the default for high-blast-radius repos.

## Key Design Decisions

_(Most AD IDs are assigned as concrete slices land.)_

### AD-071: Active Plan Drift Is Intervention Debt

Plans are part of the harness control surface. If an active exec plan presents
stale status, contradicts completed tickets, or points agents at superseded
workflow, that is an intervention-debt signal, not mere documentation tidying.

The first response is repo-visible correction: mark superseded plans clearly,
create a current operating plan, and create follow-up tickets for mechanical
plan hygiene checks. The long-term response is deterministic checking through
docs consistency, doctor, and CI so stale "active" plans cannot silently become
the next agent's source of truth.

As of 2026-05-03, the deterministic hygiene check validates the single active
plan lifecycle, backlog-plan priority, superseded-plan current pointers,
active-plan ticket-state claims, unresolved placeholders, relative status
language, and stale verification notes. `doctor --repo` surfaces the same
report as an operator warning with the first remediation action.

### AD-073: One Active Exec Plan At A Time

Exec plans now follow a ticket-like lifecycle: backlog, active, completed, or
superseded. `docs/exec-plans/active/` must contain exactly one active plan, and
that plan is the sole top-level control surface for execution. Other strategic
or tactical plans belong in `docs/exec-plans/backlog/` with explicit
`**Priority:**`, `**Depends On:**`, `**Blocks:**`, and `**Related Tickets:**`
metadata until the current active plan promotes a slice of their work.

This avoids plan pile-ups that behave like abandoned in-progress tickets. If an
agent wants to execute work from a backlog plan, it first checks dependencies
and blockers, updates the active plan's priority order, then performs the work
through normal tickets and commits.

### AD-276: Retire The Pipeline-Learnings Standing Tracker

`docs/exec-plans/pipeline-learnings.md` was created from the Mars meta-harness
relevance audit as a standing tracker for recurring failure patterns and fix
recipes. It stayed empty through the entire 2026-05 live validation campaign
because the learnings loop landed in stronger, already-checked artifacts
instead:

- Recurring failure patterns and their fixes are recorded as architecture
  decisions in `delivery-operating-model.md` (AD-164 through AD-218 and later),
  each tied to the replay run that exposed the failure.
- Run-level evidence lives in `docs/validation/` reports and the active plan's
  scenario schedule.
- Deterministic remediation recipes live in `internal/remediation` with
  trace-linked score evidence (MH-048).

The tracker is retired as of 2026-06-11 rather than backfilled: copying 65
replay runs into a parallel ledger would duplicate existing evidence and
create a second source of truth for the same learnings. If a future failure
class needs a durable recipe, it goes through the AD or remediation-recipe
path, not a freestanding tracker file.

### AD-277: Retire Completed Snapshot Docs Outside docs/exec-plans

`docs/prompt-port-status.md` tracked the MH-025 Mars prompt port. Every
checklist row completed in 2026-04, the role inventory moved to
`examples/roles/` plus the checked role registry in `docs/roles/ROLES.md`, and
the tier assignments were superseded by manifest trust metadata. Keeping a
fully-checked snapshot at the docs root presents stale role doctrine as
current.

As of 2026-06-11 the snapshot is retired. The durable rule: when a tracking
snapshot's content is wholly superseded by checked, living artifacts (role
registry, manifests, done tickets), the snapshot is deleted and the done
ticket remains the historical record. `docs/quickstart.md` was reconciled
against the 2026-06-11 CLI command surface in the same change.

### Design anchors

- **Intervention detection:** classify events as **clear interventions** (unambiguous human override), **ambiguous** (could be normal workflow), or **non-interventions** to reduce false-positive evolution; store classification rationale in the trace.
- **Reviewer meta-role:** dedicated analysis pass over traces and diffs; separate prompts/policies from worker roles where practical; same inference stack as workers—see circular trust below.
- **Root cause classification:** buckets aligned with tenets (prompt, skill, guardrail, trigger, policy, context, model limitation)—each maps to a concrete evolution target file or setting.
- **Telemetry triage:** recurring failure patterns and low scores become typed improvement proposals before any prompt, guardrail, context, tool, manifest, process, or inference change is attempted.
- **Plan discipline:** only one active exec plan can drive work at once; backlog plans are prioritized like tickets and carry dependencies, blockers, and related tickets.
- **Evolution commit creation:** concrete file edits (e.g. `.harness/roles/`, `.harness/skills/`, guardrails, manifest) with trace-linked diffs; include commit text linking to originating job and score snapshot.
- **Before/after tracking:** link evolution commits to subsequent score distributions and intervention rates; automatic rollback proposal if metrics violate guard thresholds.
- **Safety rails (non-exhaustive):** cannot modify **own meta-prompts** arbitrarily; **rate limits** (e.g. max one evolution commit per role and scope per day); **auto-disable** evolution if scores worsen beyond a threshold after a change lands.
- **Circular trust problem:** Reviewer uses the **same model stack** as workers—mitigate with deterministic checks, hard guardrails, human-triggered contributor runs for high-risk paths, and logging of reviewer conclusions for audit.

### Non-goals (v1)

Unbounded prompt rewriting without trace visibility, and cross-repo learning without explicit consent and scoping.

### Telemetry

Every evolution candidate should record **inputs hash** (manifest + role versions) so duplicate proposals can be deduplicated and A/B comparisons stay reproducible.

Store the **parent job id** in each evolution commit message and trace record for traceability across follow-up commits and reverts.

## Discoveries

- **2026-05-02 — Self-reflective telemetry triage:** Recurring telemetry patterns and low score snapshots now map to explicit improvement targets in `internal/telemetry`. The serve loop records bounded evolution reviews from those proposals instead of generic "investigate the prompt" notes.
- **2026-05-02 — Skill evolution target:** Repeated workflow confusion, max-turn loops, and human recovery procedures should create or update compact scoped skills instead of bloating role prompts.
- **2026-05-02 — Active plan drift:** The master execution plan and delivery schedule were kept as active even after the ticket tree moved far ahead of them. Stale active plans now count as intervention debt and should be corrected through a current operating plan plus mechanical hygiene checks.
- **2026-05-02 — Single active plan:** The active exec-plan directory was reduced to one current operating plan. Mars parity and model evaluation plans moved to the plan backlog with priorities, while historical baseline plans moved to superseded lineage.
- **2026-05-02 — Plan dependency metadata:** Active and backlog exec plans now require dependency, blocker, and related-ticket metadata so future orchestration can choose the next plan by priority and sequencing, not priority alone.
- **2026-05-03 — Active-plan hygiene checker:** `internal/planhygiene` now powers docs-consistency and `doctor --repo` warnings for lifecycle drift, stale active-plan ticket claims, unresolved placeholders, relative status language, old verification notes, and superseded plans without current-plan pointers.
