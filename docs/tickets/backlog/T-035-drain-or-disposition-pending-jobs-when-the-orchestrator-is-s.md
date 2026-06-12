---
id: T-035
title: Drain or disposition pending jobs when the orchestrator is stopped so preemption cannot orphan mid-lifecycle work
priority: medium
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-06-12-demo-11-pace-baseline.md"]
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Decide the graceful-stop contract (drain pending jobs with a bounded deadline, or mark them with an explicit preempted disposition plus a resume path) and implement it in the serve/queue stop path with tests."
kind: intervention-debt
source: 2026-06-12 independent replay-monitor second shift (demo-11 run 2)
created: 2026-06-12
depends_on: []
---

# T-035: Drain or disposition pending jobs when the orchestrator is stopped so preemption cannot orphan mid-lifecycle work

## Context

During the 2026-06-12 demo-11 balanced-model baseline, the operator stopped the orchestrator at 01:21:45 BST to start the demo-12 replay. Engineer job 4b659db8 (dogfood-failure rework) had been enqueued at 01:21:44 BST and was left orphaned in pending — it never ran and received no disposition. The baseline report originally claimed the queue drained naturally; the second monitor shift corrected this (docs/validation/reports/2026-06-12-demo-11-pace-baseline.md, Independent observer section).

Classification: foundation-owned (queue/serve stop semantics). Distinct from T-031: T-031 covers runtime-failure routing (circle_detected/max_turns halting pending operator retry); this ticket covers graceful stop abandoning pending work with no record. Execution truth (tenet 7) requires that stopping the factory either finishes claimed work, or leaves an explicit auditable record of what was abandoned and how to resume it.

2026-06-12 negative evidence (independent monitor, T-032 rerun cross-check): zero orphaned pending jobs at orchestrator exit in both T-032 reruns (demo-12 and demo-13, natural graceful stops after the queue drained into the post-failure halt). The gap did not reproduce under natural termination — it occurred under operator preemption mid-queue (demo-11, job 4b659db8 enqueued one second before stop). The eventual fix should target the preemption path specifically (stop while pending jobs exist), not the drained-queue stop path. Also confirmed separate from T-031's AD-289 routing fix: the code paths share nothing (stop path vs failure-handling path).

## Requirements

- Graceful stop (q key, SIGINT, POST /api/stop) either drains pending jobs within a bounded deadline or marks each undrained job with an explicit preempted state/disposition that telemetry and `doctor`/status surfaces report.
- A subsequent `start`/`serve` against the same per-repo DB surfaces preempted jobs as resumable work instead of silently ignoring them.
- Stop output names the undrained jobs so validation reports cannot honestly claim a drained queue when work was abandoned.

## Affected Files

- internal/serve (stop path)
- internal/queue (pending-job preemption state)
- internal/dashboard / internal/ui (status surfaces)
- docs/design-docs/orchestrated-organization-layer.md or pipeline-engine.md (AD)

## Acceptance Criteria

- Stopping with pending jobs produces either a drained queue or explicitly preempted jobs with an auditable record; tests cover both paths.
- Restart surfaces preempted jobs.
- Orchestration change class per AD-284: two archetype replays (static browser app plus API/service or CLI/tooling) before claiming the fix.
