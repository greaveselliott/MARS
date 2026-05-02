# Goals

Goals define outcomes and competing priorities. They are the strategy layer for
Mars Harness, but they do not directly create work. The CEO aligns the single
active exec plan to active goals, BDD feature contracts, and evidence.

## Lifecycle

- `observation`: weak/noisy evidence not ready to drive work.
- `active`: an outcome currently allowed to influence the active exec plan.
- `paused`: valid, but deliberately not active.
- `validated`: success evidence closed the goal.
- `superseded`: replaced by a newer goal.
- `invalidated`: falsification evidence closed the goal.
- `merged`: absorbed into another goal.
- `split`: divided into narrower goals.

## Goal Schema

- `ID`
- `Status`: observation, active, paused, validated, superseded, invalidated, merged, split
- `Category`: product, operational, quality, safety, learning, distribution
- `Priority`: P0-P4
- `Confidence`: high, medium, low
- `Source`: user_chat, product_requirement, telemetry, quality_score, dogfood, github_issue, feedback_form, manual_doc
- `Dedupe Key`
- `Hypothesis`
- `Success Evidence`
- `Falsification Evidence`
- `Competes With`
- `Supports`
- `Last Reviewed`
- `Review Trigger`
- `Owner`

## Autonomous Goal Rule

Structured actionable evidence may create or update active goals directly.
Weak/noisy evidence goes to `docs/goals/observations.md`. Duplicate evidence
updates an existing goal or observation. Active goals do not directly create
work; the CEO must align the active exec plan, and work must flow through
tickets.

## Review Rules

- Review P0/P1 active goals whenever a feature scenario passes, a dogfood run
  fails, a quality score changes, or telemetry triage finds a repeated pattern.
- Move closed goals to `docs/goals/superseded.md` with the evidence and date.
- Keep competing goals explicit. If two goals pull in different directions,
  name the tradeoff in the active exec plan.
