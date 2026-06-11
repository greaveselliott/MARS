# Scoring System

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

How the harness measures **accuracy** and **value** of autonomous work so thresholds, alerts, and self-improvement have a consistent, auditable signal.

## Context

Scores must correlate with outcomes users care about (commits to `main`, fixed CI, reduced toil) without becoming gameable noise. Signals can arrive from local git state, CI checks, guardrail blocks, optional GitHub status/check telemetry, and human follow-up commits. Low-frequency roles complicate rolling comparisons; the system still needs stable defaults for progressive autonomy (see [tenets.md](tenets.md)).

Deterministic remediation evidence is part of outcome quality. When failed
outcome details contain remediation attempts or executions, `scores export`
summarizes those recipe IDs and statuses in `docs/QUALITY_SCORE.md` so operators
can see whether the factory is repairing known failures, skipping recipes that
lack executors, or producing failed deterministic repairs that need follow-up.

Scores should be **explainable**: an operator can answer why a job raised or lowered a role’s rolling metric without reading raw model tokens.

Factory pace is a quality signal, not a replacement for accuracy. `scores
export` joins terminal scoring outcomes to trace summaries by `job_id` and
renders repo/role pace rows in `docs/QUALITY_SCORE.md`: job count, average
turns, average tool invocations, average LLM calls, average wall time, and
limit-stop count. The signal is intentionally descriptive first. It identifies
roles that spend many turns, hit max-turn/tool/wall limits, or lack trace
evidence before maintainers change budgets or prompts.

## Key Design Decisions

_(No AD IDs assigned in the baseline plan; capture ADs here as they are minted.)_

### Directional commitments (to formalize)

- **Outcome tracking via trunk events:** commits, pushes to `main`, check results, guardrail blocks, reverts, human follow-up commits touching the same files, and terminal failures normalized into an internal event stream; link events to `job_id` and `repo_id` for attribution lag.
- **Scoring formula:** combine outcome classes (success, partial, failure, human superseded) with weights; cap influence of single giant changeset if needed; version the formula when coefficients change.
- **Rolling window:** aggregate recent performance per role/repo; configurable window length with sane defaults for noisy vs quiet repos.
- **Noop detection:** down-weight or flag jobs that produced no substantive diff, only repeated hollow actions, or failed to act when actionable work existed; avoid punishing intentional “already clean” outcomes with explicit rules.
- **Time horizon for low-frequency roles:** wider windows, minimum sample counts, or damped estimates so rare roles are not whipsawed; document chosen approach when implemented.
- **Score-driven alerts:** thresholds for operator notification and for triggering Reviewer / evolution proposals; hysteresis or cooldown to reduce spam.
- **Score-driven triage:** low rolling scores with enough samples feed self-reflective telemetry triage. Scores identify unhealthy roles; telemetry, traces, and ticket state decide whether the fix belongs in prompt, skill, process, guardrail, context, inference, manifest, or tool policy.
- **Factory pace baseline:** trace pace rows are grouped by repo and role. High
  average turns/tool invocations or limit-stop outcomes become improvement
  targets in the quality artifact, but they do not automatically penalize
  accuracy scores until a later formula revision defines calibrated weights.

### Relationships

- [pipeline-engine.md](pipeline-engine.md) persists scores and raw inputs.  
- [self-improvement.md](self-improvement.md) consumes score trends and intervention overlap.

### Open questions

Whether to expose per-job **score breakdown** in the dashboard only, or also in optional GitHub check summaries, affects payload size and PII—decide before M1 ships public scoring.

Version any published score schema so external dashboards can migrate without silent semantic shifts.

## Discoveries

- **2026-05-02 — Scores become control signals:** Low score snapshots now produce typed self-improvement proposals through `internal/telemetry.TriageScore`, which the serve loop records as bounded evolution reviews when rate limits allow.
- **2026-05-20 — Factory pace becomes visible quality evidence:** Quality score
  export now renders a Factory Pace section from trace summaries and terminal
  outcomes so optimization work starts from a dated baseline instead of chat
  impressions.
- **2026-06-11 — Score windows accept a reference time:** `ComputeScore`
  evaluated its window cutoff against the wall clock even when the caller
  (quality-score export) pinned evidence to an explicit `Now`, which made the
  export grade time-dependent and time-bombed the qualityscore tests once the
  pinned fixture date aged past the window. `ComputeScoreAt` now threads the
  caller's reference time through the cutoff and `ComputedAt`; `ComputeScore`
  delegates with the wall clock.
