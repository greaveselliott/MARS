# Self-Reflective Telemetry

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [scoring-system.md](scoring-system.md), [self-improvement.md](self-improvement.md), [dogfood-and-decisions.md](dogfood-and-decisions.md), [skill-evolution.md](skill-evolution.md)

## Context

Mars proved that the harness must grade both itself and the feature work it is building. A score that only appears in a dashboard is passive telemetry. Mars Harness needs a self-reflective loop: consume traces, terminal outcomes, score trends, guardrail blocks, dogfood results, ticket state, and human follow-up, then proactively decide what part of the harness should improve.

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

### AD-065: Telemetry Triage Creates Intervention-Debt Tickets

Self-reflection must create durable work, not only dashboard signals or evolution rows. When recurring telemetry patterns or low score snapshots identify actionable improvement targets, Mars Harness creates or updates `kind: intervention-debt` tickets through the canonical ticket creation path.

The dedupe key is repo, role, target, category, and evidence window. This keeps repeated failures from creating ticket storms while still letting a new evidence window reopen durable work when the issue returns. Tickets carry role, repo, target, category, severity, confidence, source event or score snapshot, evidence, recommendation, candidate files, and acceptance criteria.

Direct evolution remains bounded. Process, product, unknown, or unsafe changes default to intervention-debt tickets; autonomous evolution can only happen after the ticketed evidence is constrained by trust, allowlists, and regression checks.

### AD-072: Quality Scores Are Generated Repo Artifacts

Mars Harness inherits Mars's A-F quality score pattern as a repo-owned artifact.
The source harness now carries `docs/QUALITY_SCORE.md`, and initialized target
repos receive the same seed file. This makes current quality, readiness, and
top improvement targets visible to any agent before it opens SQLite or the
dashboard.

`mars-harness scores export --repo <path>` is now the deterministic refresh
path. It reads role scores, terminal outcomes, ticket state, dogfood results,
guardrail blocks, check outcomes, no-op runs, human follow-up, and telemetry
triage from the repo-specific SQLite database and repo-visible ticket tree,
then rewrites `docs/QUALITY_SCORE.md`. Missing SQLite data is rendered as
`Insufficient evidence` instead of a fabricated grade.

Manual edits belong only inside the preserved manual-notes block. Low role
scores with at least five samples create or update deduped
`kind: intervention-debt` tickets, making quality regressions durable work.
Dashboard quality links point back to this generated artifact; the dashboard is
not the source of truth.

## Current Implementation

`internal/telemetry` now exposes triage functions that convert recurring failure patterns and low score snapshots into improvement proposals. `serve.checkEvolution` consumes those proposals, creates or updates intervention-debt tickets, emits dashboard events with ticket links, and records bounded evolution reviews with concrete suggestions and candidate files.

The first implementation is deliberately small:

- recurring context/budget failures point at glossary and role prompt scope
- inference failures point at model/server tuning and doctor checks
- tool timeouts point at tool policy and role command guidance
- max-turn and loop failures point at role prompt completion criteria or a missing reusable skill
- manifest failures point at `.harness/manifest.yaml`
- ticket-gate failures point at the role's ticket completion workflow, trust level, and target ticket state; self-chain auto-recovery does not retry them unchanged
- low scores point at process triage across prompt, skill, guardrail, tool, model, and intervention debt
- `mars-harness scores export --repo <path>` refreshes `docs/QUALITY_SCORE.md`
  from live evidence and preserves manual notes

## Required Next Steps

- Add Orchestrator survey support so triage runs even when no new job finishes.
- Add richer dashboard/API views for improvement proposals beyond the current event stream.
- Extend triage with dogfood-specific and ticket-state-specific signals.
- Add scanner-generated glossary and command-route updates when triage repeatedly identifies context gaps.
- Add skill creation/update proposals when triage repeatedly identifies missing reusable procedure.

## Non-Goals

- Unbounded self-modification.
- Treating scores as absolute truth without trace evidence.
- Cross-repo learning without explicit consent and scoping.
