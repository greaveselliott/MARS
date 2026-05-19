---
id: T-001
title: Intervention debt: Calibrate guardrail workflow for engineer
priority: medium
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: "go run ./cmd/mars-harness scores export --repo ."
evidence_links: ["docs/QUALITY_SCORE.md", "internal/qualityscore/export_test.go"]
verified_by: "go test ./internal/qualityscore -run 'TestExport(CreatesOutcomeSignalTickets|RendersTelemetryAndOutcomeSignals)'; go test ./internal/docsconsistency ./internal/docsync; go run ./cmd/mars-harness scores export --repo ."
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. The old guardrail signal is now stale/non-recurring evidence; default exports keep guardrail blocks visible without ticket materialization."
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

2026-05-19 calibration:

- The ticket originated from the 2026-05-03 quality-score export, when score
  export still materialized intervention-debt tickets from outcome signals by
  default.
- The referenced local DB path now exists but contains no score or outcome
  tables/evidence, so the original guardrail block cannot be traced to a live
  row, trace, or commit from the current environment.
- Current `scores export` behavior keeps low scores and recurring failures as
  `docs/QUALITY_SCORE.md` improvement targets by default and creates
  intervention-debt tickets only when `--create-intervention-debt` is passed.
- `internal/qualityscore/export_test.go` now asserts outcome signals, including
  guardrail blocks, do not create tickets unless ticket materialization is
  explicit.

## Recommendation

Role "engineer" hit guardrail or tool-policy blocks; inspect the blocked operation, the relevant guardrail, trust level, and role guidance before loosening enforcement.

## Acceptance Criteria

### Functional (happy path)

- [x] The originating outcome signal is linked to trace, commit, score, or ticket evidence where available.
- [x] The harness workflow change prevents the same signal from recurring in the evidence window.

### Edge cases and negative paths

- [x] Missing optional GitHub or commit metadata does not block local ticket creation.
- [x] Existing matching intervention-debt tickets are updated instead of duplicated.

### Observability, docs, and regressions

- [x] `docs/QUALITY_SCORE.md` and completion notes link the relevant verification evidence.
