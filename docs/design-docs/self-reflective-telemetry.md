# Self-Reflective Telemetry

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [scoring-system.md](scoring-system.md), [self-improvement.md](self-improvement.md), [dogfood-and-decisions.md](dogfood-and-decisions.md), [skill-evolution.md](skill-evolution.md)

## Context

Mars proved that the harness must grade both itself and the feature work it is building. A score that only appears in a dashboard is passive telemetry. MARS needs a self-reflective loop: consume traces, terminal outcomes, score trends, guardrail blocks, dogfood results, ticket state, and human follow-up, then proactively decide what part of the harness should improve.

The improvement target may be the process, role prompt, skill, guardrail, knowledge route, tool policy, manifest, model/inference configuration, scanner, ticket flow, or generated target harness. The loop must not jump straight from "bad outcome" to arbitrary self-editing. It should triage evidence first, create or update durable work, and only apply direct evolution inside strict trust and safety bounds.

## Decisions

### AD-037: Telemetry Must Be Triaged Into Improvement Targets

Raw telemetry is not enough. Recurring patterns and low scores are classified into explicit target surfaces:

- prompt
- skill
- process
- guardrail
- context
- inference
- manifest
- tool policy
- unknown

`workspace_hygiene` is a separate failure category for generated dependency or
build churn, missing ignore policy, tracked generated directories, and
pre-job/post-dependency hygiene blockers. Unlike context overflow, inference,
dispatch, or generic guardrail policy, workspace hygiene normally belongs to
the target repo because the remediation is usually `.gitignore`, dependency
tracking, or generated-output cleanup policy in that repository.

Each triage output includes role, repo, evidence, severity, confidence, suggestion, and candidate files when a bounded evolution can safely inspect or edit them.

### AD-038: Scores Are Control Signals, Not Vanity Metrics

Scores must drive behavior. A low rolling score with enough samples should produce a self-improvement proposal and eventually an intervention-debt ticket, quality-score update, planner item, or bounded evolution review.

The score does not decide the fix by itself. It says "this role's outcomes are unhealthy enough to inspect." Telemetry patterns and trace evidence decide whether the fix belongs in prompt scope, a reusable skill, process, guardrails, tool policy, model configuration, or generated guidance.

### AD-039: Self-Improvement Is Proactive But Bounded

The harness should proactively consume its own telemetry after jobs and on orchestrator surveys. Direct edits remain bounded:

- prompt and guardrail changes stay under `.harness/`
- skill changes stay under `.harness/skills/` and must stay compact, scoped, and trace-linked
- knowledge-route changes stay under `.harness/knowledge/`
- manifest changes are explicit and trace-linked
- process or product changes become tickets/plans unless an autonomous role has earned trust and the path is allowlisted
- repeated worsening disables further evolution for that role/scope

### AD-065: Telemetry Triage Quarantines Foundation Failures By Default

Self-reflection must create durable evidence, not only dashboard signals or evolution rows. When recurring telemetry patterns, non-success terminal agent results, guardrail or tool-policy blocks, repeated policy-block loops, repeated tool loops, manual stops, timeouts, low score snapshots, human follow-up, reverted agent commits, or stale in-progress tickets identify actionable improvement targets, MARS records them in telemetry, quality evidence, or bounded evolution review first.

The dedupe key is repo, role, target, category, and evidence window. This keeps repeated failures from creating ticket storms while still letting a new evidence window reopen durable work when the issue returns. Tickets carry role, repo, target, category, severity, confidence, source event, trace ID, score snapshot, commit, outcome, evidence, recommendation, candidate files, and acceptance criteria when those fields are available locally.

Target repo intervention-debt tickets are only the right durable work item when the remediation belongs to the target repository. Harness-owned failures such as dispatch protocol failures, loop/max-turn failures, guardrail/tool-policy workflow failures, context or inference failures, manifest/tool-policy gaps, and unknown terminal failures remain local telemetry first and are eligible for anonymous foundation telemetry reporting instead of being written into the target backlog.

Direct evolution remains bounded. Harness/runtime, unknown, or unsafe changes default to foundation telemetry rather than target backlog. High-signal foundation telemetry such as `guardrail_loop` can still record bounded evolution reviews against role guidance, manifest workflow, or loop-boundary surfaces without creating target backlog work. Target-owned human follow-up, reverted target commits, stale target tickets, and explicit operator requests may become intervention-debt tickets through the canonical ticket path.

### AD-072: Quality Scores Are Generated Repo Artifacts

MARS inherits Mars's A-F quality score pattern as a repo-owned artifact.
The source harness now carries `docs/QUALITY_SCORE.md`, and initialized target
repos receive the same seed file. This makes current quality, readiness, and
top improvement targets visible to any agent before it opens SQLite or the
dashboard.

`mars scores export --repo <path>` is now the deterministic refresh
path. It reads role scores, terminal outcomes, execution trace summaries,
ticket state, dogfood results, guardrail blocks, check outcomes, no-op runs,
human follow-up, and telemetry triage from the repo-specific SQLite database
and repo-visible ticket tree, then rewrites `docs/QUALITY_SCORE.md`. Missing
SQLite data or missing trace pace evidence is rendered as `Insufficient
evidence` instead of a fabricated grade.

Manual edits belong only inside the preserved manual-notes block. Low role
scores with at least five samples become improvement targets by default.
Operators pass `--create-intervention-debt` only when they deliberately want
score/outcome signals materialized as target tickets. Dashboard quality links
point back to this generated artifact; the dashboard is not the source of
truth.

### AD-278: Quality Score Regeneration Cadence And Regression Gating

`docs/QUALITY_SCORE.md` regeneration has a defined cadence instead of being
refreshed only when an agent remembers. The triggers, all served by the
existing `mars scores export --repo <path>` surface (AD-072, F-008):

1. **After every live validation run batch.** When a validation report lands
   under `docs/validation/reports/`, the same change set regenerates the
   quality score so Factory Pace rows and outcome signals reflect the run.
2. **Before any quality or readiness claim.** Tickets, plans, or release
   evidence that cite a grade must cite a same-day export, not a stale file.
3. **At least once per release-note batch** when harness jobs ran since the
   last export for the repo.

Gating on grade regressions: a drop in any Overall Roll-Up area grade between
two exports is an improvement-target signal first (AD-038). It blocks
completion claims for work that touched that area until the regression is
explained in the ticket or validation report; ticket materialization still
requires the explicit `--create-intervention-debt` flag (AD-072).

Automation status as of 2026-06-11: the cadence is operator/agent doctrine.
Wiring the export into a post-run hook or scheduled survey is recorded as a
follow-up in `docs/exec-plans/tech-debt.md` and is expected to land with the
WS-C pace/convergence telemetry slice, which extends the same export surface.

### AD-283: Convergence Failures And Guardrail Block Rates Are First-Class Export Evidence

Factory Pace (T-011, F-008-S008) collapsed every unproductive stop into a
single limit-stop count, and guardrail blocks appeared only as a raw total.
Operators triaging convergence problems need the failure kinds separated:
circle detection means the loop guard fired, max-turn/max-tool stops mean the
budget ran out, no-op outcomes mean a job ended without useful work, and
guardrail block rate shows how often policy had to intervene relative to all
terminal outcomes.

`scores export` therefore renders a Convergence And Guardrails section in
`docs/QUALITY_SCORE.md` with per repo/role counts of circle-detected stops,
max-turn/max-tool stops, other limit stops (timeout, budget, empty response),
no-op terminal outcomes, guardrail blocks, and the guardrail block rate over
terminal outcomes, plus a Convergence failures roll-up row in Evidence
Signals. All data comes from existing trace summaries and scoring outcome
tables — no runtime recording changed, so no canary replay is required for
this slice (T-027). A clean window reports that no failures were recorded;
missing evidence still renders as missing rather than healthy (F-008-S007).

### AD-104: Foundation Telemetry Uses Opt-In Anonymous Reports Through a Pluggable Collector

Raw deployed-harness telemetry remains local in the repo-specific SQLite
database. A deployed harness may derive sanitized aggregate reports into a local
outbox, but it never uploads raw traces, prompts, repo paths, remotes, ticket
text, command output, file paths, commit SHAs, usernames, source content, or raw
error messages.

Anonymous foundation telemetry is opt-in. A deployed harness sends only
allowlisted aggregate envelopes to a configured collector endpoint. The collector
owns the foundation telemetry database. For local dogfood, the collector stores
reports in SQLite. For broader public operation, the same collector API can use
a hosted Postgres-compatible backend such as Neon without changing the
deployed-harness protocol.

The write path is:

1. raw local telemetry: `~/.mars/db/{repo-name}/mars.db`
2. local anonymous outbox: `telemetry_report_outbox` in the same repo DB
3. collector intake: local SQLite for dogfood, hosted Postgres-compatible storage later
4. foundation triage: repeated anonymous patterns across distinct anonymous report keys or harness versions become MARS source work, not target repo intervention debt

Remote reporting defaults to off, disabled reporting is healthy, and send
failures never block local harness operation.

## Current Implementation

`internal/telemetry` now exposes triage functions that convert recurring failure patterns, terminal failure signals, guardrail blocks, human follow-up, reverted commits, stale ticket state, manual stops, and low score snapshots into improvement proposals. `serve.checkEvolution`, failed-job handling, tool-policy reporting, and quality-score export consume those proposals, create or update intervention-debt tickets, emit dashboard events with ticket links, and record bounded evolution reviews with concrete suggestions and candidate files when confidence and allowlisted paths make direct review safe.

The first implementation is deliberately small:

- `internal/remediation` owns the first deterministic remediation registry.
  Recipes are selected from normalized failure signals before LLM repair work
  and include stable IDs, target area, safety classification, candidate
  commands, candidate files, skipped reasons, and next actions. The first
  catalog covers dirty worktrees before run, stale in-progress tickets, missing
  or invalid manifests, missing generated docs, known doctor remediations,
  repeated scanner duplicate tickets, missing dependency setup, missing
  optional tools, and model artifact checksum mismatches.
- `internal/serve` records applicable remediation attempts in failed outcome
  details with trace IDs before generic telemetry remediation runs. Auto-safe
  ready recipes defer generic retry jobs so deterministic repair can run first;
  operator-required and approval-required recipes remain visible evidence while
  preserving the existing retry behavior.
- The generated-docs auto-safe recipe executes through the existing
  `scanner.Upgrade` API, not a shell runner. Outcome details record whether the
  update applied files, no-oped, or failed before the generic retry path is
  considered.
- Auto-safe planning alone is not enough to suppress generic telemetry retry.
  `serve` suppresses retry only when the ready recipe has a registered
  deterministic executor, preventing future catalog additions from creating a
  no-progress stall.
- `doctor --repo` includes a deterministic-remediation check that surfaces
  known recipe IDs and fixes for missing target harness scaffolds, manifests,
  and generated metadata before a run has to fail and rediscover the same issue.
- `mars scores export` reads remediation attempt and execution evidence
  from scoring outcome details and renders deterministic-remediation summaries
  in `docs/QUALITY_SCORE.md`, including skipped-without-executor and failed
  execution improvement targets.
- `mars scores export` also joins terminal outcomes to trace summaries
  by `job_id` and renders Factory Pace rows by repo and role. The rows show
  average turns, tool invocations, LLM calls, wall time, and limit stops so pace
  optimization starts from durable evidence.
- recurring context/budget failures point at glossary and role prompt scope
- inference failures point at model/server tuning and doctor checks
- tool timeouts point at tool policy and role command guidance
- max-turn and loop failures point at role prompt completion criteria or a missing reusable skill
- manifest failures point at `.harness/manifest.yaml`
- ticket-gate failures point at the role's ticket completion workflow, trust level, and target ticket state; self-chain auto-recovery does not retry them unchanged
- guardrail and tool-policy blocks point at guardrail calibration, trust level, and role guidance without weakening enforcement first
- `guardrail_loop` signals point at the repeated policy message, terminal disposition guidance, role workflow, and loop-boundary behavior before retrying unchanged
- workspace hygiene failures point at `.gitignore`, package manager manifests,
  tracked generated paths, and the `workspace_hygiene`/`dependency_sync`
  recipe output before any retry
- missing optional tools point at doctor/install/skip guidance instead of a
  successful repair claim, so hosted-model or reduced-tooling runs can continue
  honestly while local-only workflows still get actionable setup instructions
- human follow-up and reverted commits point at the role workflow, reusable skills, and guardrails that should have prevented the correction
- stale in-progress tickets point at Engineer and Janitor ticket-drain behavior
- manual stops point at role stop conditions, timeout policy, recovery, or escalation behavior
- low scores point at process triage across prompt, skill, guardrail, tool, model, and intervention debt
- harness-owned telemetry patterns stay out of target backlogs and can be
  exported as opt-in anonymous aggregate reports through `mars telemetry`
- `mars scores export --repo <path>` refreshes `docs/QUALITY_SCORE.md`
  from live evidence, reports low-score/outcome/ticket-state improvement
  targets, reports pace bottlenecks from traces, and preserves manual notes.
  Ticket materialization requires `--create-intervention-debt`.
- `serve` runs a native Orchestrator survey loop on startup and a watchdog
  interval. The survey consumes ticket state, recent scored outcomes, telemetry
  patterns, low score snapshots, active recovery jobs, and stuck running jobs
  even when no new agent job finishes. It routes stale and blocked tickets to
  Janitor, eligible product in-progress work to Engineer, failed checks to
  Pipeline Fixer, dogfood failures to Engineer, and no-op outcomes to Janitor
  with payload-mode, concurrency-group, and daily-cap queue metadata. Recurring
  runtime telemetry and low scores stay quarantined as foundation telemetry by
  default instead of creating target backlog churn.

## Required Next Steps

- Add similarly narrow internal executors before any additional recipe becomes
  auto-safe; recipes without an executor must stay planned or skipped.
- Promote repeated successful recipes into doctor or setup checks by naming the
  stable recipe ID and a concrete remediation command before a target run fails.
- Add richer dashboard/API views for improvement proposals beyond the current event stream.
- Extend triage with dogfood-specific signal detail and richer commit metadata when optional GitHub evidence is configured.
- Add scanner-generated glossary and command-route updates when triage repeatedly identifies context gaps.
- Add skill creation/update proposals when triage repeatedly identifies missing reusable procedure.

## Non-Goals

- Unbounded self-modification.
- Treating scores as absolute truth without trace evidence.
- Cross-repo learning without explicit consent and scoping.
