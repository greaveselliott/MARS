---
id: T-031
title: Harden qa and dogfood terminal convergence so circle_detected runtime failures do not cap lifecycle reach
priority: P1
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-06-12-demo-14-convergence-routing-replay.md", "docs/design-docs/delivery-operating-model.md", "docs/design-docs/convergence-state-machine.md", "docs/validation/reports/2026-06-12-demo-11-pace-baseline.md", "docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md", "docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md"]
verified_by: "go test ./internal/serve -run 'TestHandleJobFailed_qaMaxTurnsEnqueuesConvergenceRetry|TestHandleJobFailed_dogfoodCircleDetectedEnqueuesConvergenceRetry|TestHandleJobFailed_engineerMaxTurnsWithoutTicketEnqueuesConvergenceRetry|TestHandleJobFailed_convergenceRetryFailureEscalatesWithDisposition|TestHandleJobFailed_productContinuationFailureEscalatesWithDisposition|TestHandleJobFailed_failedRetryFingerprintEscalatesNextFailure|TestHandleJobFailed_environmentFailureStillHaltsWithoutRetry'"
owner: "foundation-maintainer"
last_attempt: "2026-06-12"
blocker: "none"
blocked_by: []
trace_id: "747e5189 (cto-weekly max_turns) -> b63d6b31 (auto convergence_retry, completed)"
next_action: "Done. AD-284 replay passed (demo-14, zero operator run-role calls, all 10 product tickets closed); see docs/validation/reports/2026-06-12-demo-14-convergence-routing-replay.md."
kind: intervention-debt
source: weekly-priorities.md
created: 2026-06-12
depends_on: []
---

# T-031: Harden qa and dogfood terminal convergence so circle_detected runtime failures do not cap lifecycle reach

During the 2026-06-12 balanced-model demo-11 pace baseline (docs/validation/reports/2026-06-12-demo-11-pace-baseline.md, findings F1/F2), a qa review with clean evidence circled at turn 18/20 without recording a job_disposition_record, and the single dogfood job circled in 48s. Runtime failures are not redispatched by design, so the qa circle stalled the lifecycle until an operator POST /api/run-role retry and the dogfood circle ended the run before the release stage. Phase 3 should harden terminal-role disposition discipline (record the verdict before re-reading evidence) and/or give convergence runtime failures a bounded retry path.

2026-06-12 implementation (AD-289): dispatch-mode failure handling now
implements the AD-286 `operator-retry-routing` transition. Root cause of the
halt: the AD-135 runtime-failure stop in `internal/serve/server.go`
(`handleJobFailed` → `dispatchRuntimeFailureStops`) predates the
AD-227/AD-239 bounded continuation edges, which only cover Engineer
max_turns/circle_detected with an ordinary in-progress product ticket; qa,
dogfood, security, and engineer-without-a-continuable-ticket convergence
failures all fell through to the halt. The fix: convergence failures
(`max_turns`, `circle_detected` only) get one automatic same-role
`convergence_retry` dispatch per failure fingerprint (`repo:role:category`,
24h window); failed recovery jobs and repeat fingerprints escalate with a
recorded `blocked/operator_retry` disposition naming the exact retry
command. Environment failures (`model_unavailable`, `context_overflow`,
etc.) keep the fail-fast halt (T-033/T-032 own preflight). Unit evidence in
`verified_by`; F-006-S047 documents the business logic. Mirrored-doctrine
check (AGENTS.md rule 13): generated target role guidance describes runtime
failures only generically ("leave foundation/runtime failures as telemetry
or blocked dispositions"), which remains accurate under AD-289; no scanner
default or generated-guidance change is required. Ownership: foundation-owned
(orchestration runtime); the demo targets carry no part of this fix.

2026-06-12 CLOSED — AD-284 replay evidence (demo-14 Inventory/API, fresh
brief, v0.50.14, zero operator run-role calls): every AD-289 path fired
live. cto-weekly max_turns 747e5189 → automatic retry b63d6b31 completed
(the v0.50.2 stall state, recovered without operator); dogfood
56540d01/73dcbd2a → retries d8524b51/c94a8cad → chain-guard escalations
with recorded blocked/operator_retry dispositions naming the exact retry
command; dogfood 1950aa3b/58d48239 → immediate fingerprint-window
escalations (no extra job burn — the demo-13 repeated-failure shape
contained); engineer max_turns 3f7baafe took the richer AD-227
product_continuation (edge layering preserved). Lifecycle drained the full
backlog: T-001–T-010 done, 4 dogfood passes, 46 target commits, 126.5 min,
queue empty at stop. Full report:
docs/validation/reports/2026-06-12-demo-14-convergence-routing-replay.md.
Scope confirmation: T-035 (graceful-stop draining) and the post-max_turns
ticket_gate cascades do NOT share this code path — the routing fix lives in
the dispatch failure branch; the stop path and the gate-repair handoff are
separate edges with their own tickets.

2026-06-12 T-032 rerun cross-check (independent monitor) — budget
calibration evidence: with the AD-288 overflow fix landed, max_turns became
the dominant terminal failure — demo-12 rerun: 4 max_turns (engineer
c3a6da4a, e81444cc, cefd6681; qa 9bfcfb6e); demo-13 rerun: 5/5 engineer
jobs failed max_turns (4b2a331c, 09eab37b, 6b4b882c, 1f67b7ca, b7cbd006),
zero completed, longest 1,482s. A naive multi-retry would have burned five
more jobs on demo-13; the one-retry-per-fingerprint budget plus escalation
is calibrated against exactly this shape. Related monitor finding recorded
as AD-286 evidence (not this slice's scope): demo-13 guardrail churn rose
12→70 blocks after the overflow fix (demo-12 improved 88→65) — turn burn
shifted from context overflow to guardrail-fighting; the block-message
contract and T-029 slices own that frontier.

2026-06-12 evidence update (independent replay monitor): the post-failure
orchestration gap is failure-class-independent, not specific to qa/dogfood
circles. The monitor captured the log line "not dispatching runtime failure
through Orchestrator; foundation telemetry or operator retry must resolve it
first" after the 2026-06-11 cto-weekly max_turns failure
(docs/validation/reports/2026-06-11-demo-11-pace-baseline.md, Independent
observer section) and after the demo-12 session-opening model_unavailable
failure (docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md);
in every case the pipeline halts pending operator. The bounded-retry design
should distinguish retryable convergence failures (circle_detected,
max_turns — bounded automatic retry) from environment failures
(model_unavailable, context_overflow — fail fast to an actionable preflight
or telemetry finding, owned by T-033/T-032 respectively). The convergence
state-machine design doc maps this gap as a missing automatic transition.
Related but distinct: T-035 covers the graceful-stop/preemption gap (pending
jobs orphaned with no disposition when the orchestrator is stopped), and the
demo-12 second-shift evidence adds post-max_turns ticket_gate cascade
failures (28bd2736, 04dc813d) — handoff incompleteness after a runtime
failure, same missing-transition family.
