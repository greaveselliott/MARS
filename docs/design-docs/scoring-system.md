# Scoring System

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

How the harness measures **accuracy** and **value** of autonomous work so thresholds, alerts, and self-improvement have a consistent, auditable signal.

## Context

Scores must correlate with outcomes users care about (merged PRs, fixed CI, reduced toil) without becoming gameable noise. Signals will often arrive asynchronously via **GitHub** (checks, merges, closes). Low-frequency roles complicate rolling comparisons; the system still needs stable defaults for progressive autonomy (see [tenets.md](tenets.md)).

Scores should be **explainable**: an operator can answer why a job raised or lowered a role’s rolling metric without reading raw model tokens.

## Key Design Decisions

_(No AD IDs assigned in the baseline plan; capture ADs here as they are minted.)_

### Directional commitments (to formalize)

- **Outcome tracking via GitHub events:** webhooks or polling normalized into an internal event stream; link events to `job_id` and `repo_id` for attribution lag (merge minutes later than push).
- **Scoring formula:** combine outcome classes (success, partial, failure, human superseded) with weights; cap influence of single giant PRs if needed; version the formula when coefficients change.
- **Rolling window:** aggregate recent performance per role/repo; configurable window length with sane defaults for noisy vs quiet repos.
- **Noop detection:** down-weight or flag jobs that produced no substantive diff, only comment churn, or repeated hollow actions; avoid punishing intentional “already clean” outcomes—needs explicit rules.
- **Time horizon for low-frequency roles:** wider windows, minimum sample counts, or damped estimates so rare roles are not whipsawed; document chosen approach when implemented.
- **Score-driven alerts:** thresholds for operator notification and for triggering Reviewer / evolution proposals; hysteresis or cooldown to reduce spam.

### Relationships

- [pipeline-engine.md](pipeline-engine.md) persists scores and raw inputs.  
- [self-improvement.md](self-improvement.md) consumes score trends and intervention overlap.

### Open questions

Whether to expose per-job **score breakdown** in the dashboard only, or also in GitHub check summaries, affects payload size and PII—decide before M1 ships public scoring.

Version any published score schema so external dashboards can migrate without silent semantic shifts.

## Discoveries

_(None yet.)_
