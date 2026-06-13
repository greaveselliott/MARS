---
id: T-035
title: Drain or disposition pending jobs when the orchestrator is stopped so preemption cannot orphan mid-lifecycle work
priority: medium
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-06-12-demo-11-pace-baseline.md
verified_by: "go test ./internal/queue -run PreemptPending; serve.Stop preemption log"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: queue.PreemptPending cancels pending jobs with explicit reason on graceful stop."
blocker: ""
blocked_by: []
trace_id: ""
next_action: ""
kind: intervention-debt
source: 2026-06-12 demo-11 operator preemption orphan
created: 2026-06-12
depends_on: []
---

# T-035: Preempt pending jobs on orchestrator stop

Closed 2026-06-12. `serve.Stop` calls `queue.PreemptPending` so pending jobs
receive an auditable cancelled disposition instead of silent orphaning.

## Acceptance criteria

- [x] Stopping with pending jobs marks them cancelled with explicit reason.
- [x] Regression test `TestQueue_PreemptPending`.
- [ ] Restart surfaces preempted jobs as resumable (deferred — next slice).
