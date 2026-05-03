---
id: T-001
title: Intervention debt: Calibrate guardrail workflow for engineer
priority: medium
complexity: medium
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
next_action: "TBD"
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "guardrail_block"
  confidence: "0.80"
  evidence_window: "30d"
  origin_kind: "quality_score_outcome"
  outcome_count: "1"
  outcome_type: "guardrail_blocked"
  repo_id: "mars-harness"
  role: "engineer"
  severity: "medium"
  target: "guardrail"
source: quality-score-outcome:mars-harness:engineer:guardrail_blocked:30d
created: 2026-05-03
depends_on: []
---

# T-001: Intervention debt: Calibrate guardrail workflow for engineer

## Context

`mars-harness scores export` detected an intervention-debt outcome signal while refreshing `docs/QUALITY_SCORE.md`.

## Evidence

- Role: engineer
- Repo ID: mars-harness
- Outcome: guardrail_blocked
- Count: 1
- Evidence window: 30d

## Recommendation

Role "engineer" hit guardrail or tool-policy blocks; inspect the blocked operation, the relevant guardrail, trust level, and role guidance before loosening enforcement.

## Acceptance Criteria

### Functional (happy path)

- [ ] The originating outcome signal is linked to trace, commit, score, or ticket evidence where available.
- [ ] The harness workflow change prevents the same signal from recurring in the evidence window.

### Edge cases and negative paths

- [ ] Missing optional GitHub or commit metadata does not block local ticket creation.
- [ ] Existing matching intervention-debt tickets are updated instead of duplicated.

### Observability, docs, and regressions

- [ ] `docs/QUALITY_SCORE.md` and completion notes link the relevant verification evidence.
