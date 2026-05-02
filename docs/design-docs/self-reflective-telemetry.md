# Self-Reflective Telemetry

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [scoring-system.md](scoring-system.md), [self-improvement.md](self-improvement.md), [dogfood-and-decisions.md](dogfood-and-decisions.md)

## Context

Mars proved that the harness must grade both itself and the feature work it is building. A score that only appears in a dashboard is passive telemetry. Mars Harness needs a self-reflective loop: consume traces, terminal outcomes, score trends, guardrail blocks, dogfood results, ticket state, and human follow-up, then proactively decide what part of the harness should improve.

The improvement target may be the process, role prompt, guardrail, knowledge route, tool policy, manifest, model/inference configuration, scanner, ticket flow, or generated target harness. The loop must not jump straight from "bad outcome" to arbitrary self-editing. It should triage evidence first, create or update durable work, and only apply direct evolution inside strict trust and safety bounds.

## Decisions

### AD-037: Telemetry Must Be Triaged Into Improvement Targets

Raw telemetry is not enough. Recurring patterns and low scores are classified into explicit target surfaces:

- prompt
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

The score does not decide the fix by itself. It says "this role's outcomes are unhealthy enough to inspect." Telemetry patterns and trace evidence decide whether the fix belongs in prompt scope, process, guardrails, tool policy, model configuration, or generated guidance.

### AD-039: Self-Improvement Is Proactive But Bounded

The harness should proactively consume its own telemetry after jobs and on orchestrator surveys. Direct edits remain bounded:

- prompt and guardrail changes stay under `.harness/`
- knowledge-route changes stay under `.harness/knowledge/`
- manifest changes are explicit and trace-linked
- process or product changes become tickets/plans unless an autonomous role has earned trust and the path is allowlisted
- repeated worsening disables further evolution for that role/scope

## Current Implementation

`internal/telemetry` now exposes triage functions that convert recurring failure patterns and low score snapshots into improvement proposals. `serve.checkEvolution` consumes those proposals and records bounded evolution reviews with concrete suggestions and candidate files.

The first implementation is deliberately small:

- recurring context/budget failures point at glossary and role prompt scope
- inference failures point at model/server tuning and doctor checks
- tool timeouts point at tool policy and role command guidance
- max-turn and loop failures point at role prompt completion criteria
- manifest failures point at `.harness/manifest.yaml`
- low scores point at process triage across prompt, guardrail, tool, model, and intervention debt

## Required Next Steps

- Create or update intervention-debt tickets from triage proposals.
- Add quality-score exports that include top improvement targets.
- Add Orchestrator survey support so triage runs even when no new job finishes.
- Add dashboard/API views for improvement proposals.
- Extend triage with dogfood-specific and ticket-state-specific signals.
- Add scanner-generated glossary and command-route updates when triage repeatedly identifies context gaps.

## Non-Goals

- Unbounded self-modification.
- Treating scores as absolute truth without trace evidence.
- Cross-repo learning without explicit consent and scoping.
