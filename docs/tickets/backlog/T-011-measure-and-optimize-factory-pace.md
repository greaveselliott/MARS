---
id: T-011
title: Measure and optimize factory pace
priority: high
complexity: large
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Establish the current pace baseline from traces and job outcomes, define target thresholds, then implement and test the smallest changes that materially reduce turns-to-solid-outcome without hiding runaway work."
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "factory_pace"
  confidence: "high"
  evidence: "User reported on 2026-05-19 that agents take many turns before solid outcomes, the upper turn limit is too low, and factory pace is not measured."
  origin: "user request 2026-05-19"
  repo: "mars-harness"
  role: "all factory agents"
  severity: "high"
  target: "foundation and deployed harnesses"
source: user request 2026-05-19
created: 2026-05-19
depends_on: []
---

# T-011: Measure and optimize factory pace

## Context

Factory pace is currently an implicit concern rather than a measured performance signal. The user reported that different agents take many turns before reaching a solid outcome while the upper turn limit is also too low for hard work. That combination creates two separate risks: wasted turns when agents drift, and premature failure when a valid task needs a longer but still productive run.

The factory needs a durable pace metric that shows how quickly roles reach useful outcomes, how many turns and tool calls are spent on avoidable loops, and whether runtime limits are calibrated by evidence instead of guesswork.

## Requirements

- Build a clear baseline view of current factory pace from available traces, queue/job outcomes, scoring evidence, and dogfood reports.
- Define pace metrics that operators and future automation can reason about, including turns-to-solid-outcome, tool calls per successful outcome, wall-clock duration, retry count, max-turn failure rate, and avoidable-loop indicators.
- Decide where pace belongs in the operating model: scoring, quality score export, dashboard/CLI visibility, trace summaries, telemetry triage, or some combination.
- Plan and implement targeted speed improvements only after the baseline identifies the largest turn sinks. Candidate fixes may include role prompt tightening, terminal-handoff enforcement, context routing changes, deterministic preflight/remediation shortcuts, model/profile routing, or max-turn calibration by role.
- Calibrate upper limits so productive work can continue while unproductive loops still stop quickly and produce actionable intervention debt.
- Add tests and repeatable evidence that pace improved or that the new metric exposes the bottleneck honestly when improvement is not yet proven.

## Affected Files

- internal/agent/
- internal/serve/
- internal/scoring/
- internal/qualityscore/
- internal/trace/
- internal/ui/
- internal/dashboard/
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-008-scoring-trust-quality.md
- docs/design-docs/scoring-system.md
- docs/design-docs/dogfood-and-decisions.md
- docs/QUALITY_SCORE.md

## BDD Evidence

- Scenario IDs: to be added to F-005 and/or F-008 before implementation changes behavior.
- Evidence links: expected trace fixture tests, scoring/quality export tests, and a before/after dogfood or replay report that shows turns, tools, duration, retry count, and terminal outcome.
- Verified by: TBD

## Acceptance Criteria

### Functional (happy path)
- [ ] Current factory pace is measured from durable evidence, with a dated baseline documented before optimization work starts.
- [ ] Pace is represented as a first-class metric with role/repo/job attribution rather than as ad hoc notes in chat.
- [ ] The implementation surfaces pace where operators already inspect factory health, such as scores export, trace summaries, dashboard, or CLI output.
- [ ] At least one evidence-backed resolution reduces avoidable turns or improves successful completion before max-turn failure.
- [ ] Max-turn and budget behavior is calibrated by role or work type so higher limits are available for productive work without allowing silent loops.

### Edge cases and negative paths
- [ ] Sparse or missing trace data reports insufficient pace evidence instead of a healthy default.
- [ ] Long but productive runs are distinguished from circular runs using terminal outcomes, tool diversity, diff/evidence progress, or other auditable signals.
- [ ] Pace scoring does not reward rushed incomplete work or penalize necessary validation on complex tickets.
- [ ] Max-turn failures create actionable evidence or intervention debt with the observed turn sink.

### Non-goals
- Blindly raising every role max-turn limit without pace evidence.
- Treating wall-clock speed as more important than correct, tested, documented outcomes.
- Replacing accuracy/value scores; pace should complement them.

### Observability, docs, and regressions
- [ ] Feature contracts document any new pace metric, score effect, runtime limit rule, or user-visible output.
- [ ] Design docs explain the metric formula, thresholds, and how it avoids gaming.
- [ ] Tests cover metric computation, empty evidence, max-turn classification, and any CLI/dashboard/export output changes.
- [ ] A final evidence note records baseline pace, changes made, post-change pace, residual bottlenecks, and follow-up tickets if optimum is not reached.

## Notes

Treat optimum as an evidence-backed operating target, not a fixed magic number. The first implementation slice should make pace visible and identify the biggest bottleneck before changing runtime limits.
