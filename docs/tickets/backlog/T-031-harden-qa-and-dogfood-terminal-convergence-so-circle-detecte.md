---
id: T-031
title: Harden qa and dogfood terminal convergence so circle_detected runtime failures do not cap lifecycle reach
priority: P1
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
source: weekly-priorities.md
created: 2026-06-12
depends_on: []
---

# T-031: Harden qa and dogfood terminal convergence so circle_detected runtime failures do not cap lifecycle reach

During the 2026-06-12 balanced-model demo-11 pace baseline (docs/validation/reports/2026-06-12-demo-11-pace-baseline.md, findings F1/F2), a qa review with clean evidence circled at turn 18/20 without recording a job_disposition_record, and the single dogfood job circled in 48s. Runtime failures are not redispatched by design, so the qa circle stalled the lifecycle until an operator POST /api/run-role retry and the dogfood circle ended the run before the release stage. Phase 3 should harden terminal-role disposition discipline (record the verdict before re-reading evidence) and/or give convergence runtime failures a bounded retry path.
