# Validation Report: demo-11 Inventory/API Pace Baseline (balanced model)

**Date:** 2026-06-12
**Author:** foundation-maintainer
**Purpose:** Re-capture the T-011 dated factory-pace baseline on the balanced
model after the 2026-06-11 heavy-model run was reclassified evidence-only
(see `2026-06-11-demo-11-pace-baseline.md`). Recorded per the AD-285 evidence
contract, including the model-identity requirement added in v0.50.3.

> **Evidence correction (2026-06-12, second monitor shift):** the original
> "natural end / queue drained" stop-reason claim below was wrong, and the
> final lifecycle state was internally inconsistent. See the Independent
> observer section for the corrected record. The per-role pace measurements
> (turns, tool calls, wall times) remain valid baseline data; the
> lifecycle-health claim carries the caveats below.

## Run 1: Inventory/API canary on v0.50.2 — 2026-06-12

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-11-balanced-baseline-start.log`
- **Target:** `/path/to/local-redacted` (fresh git
  repo re-created from the same Inventory API brief; local bare origin
  `demo-11-origin.git`; per-repo DB cleared before the run)
- **Source ref / binary:** `mars-harness 0.50.2`, built with `make install`
  from `c0ebceb` (`release: notes 0.50.2`) on
  `codex/main-lifecycle-stabilization-rebased` (= `origin/main` at run time)
- **Model identity (AD-285):** reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (18.6 GB; ctx 131072 reasoning
  on :18081, ctx 32768 coding on :18080); fast =
  `google_gemma-4-E4B-it-Q5_K_M.gguf` (5.8 GB); resolved from
  `performance_profile: balanced`
- **Database / logs:** `~/.mars-harness/db/demo-11/mars.db`;
  `~/.mars-harness/traces/logs/demo-11-balanced-baseline-start.log`
- **Job sequence (22 jobs, 2026-06-11 23:21:44 UTC → 2026-06-12 00:13:21 UTC,
  51.6 min):** ceo 78s → coo 78s → cto-weekly 205s → engineer failed
  (max_turns, 276s) → engineer 150s (T-001 implemented) → qa 56s (rework
  requested) → orchestrator 80s → engineer 421s (rework) → **qa failed
  (circle_detected, 68s)** → [operator retry] → qa 46s → security 45s →
  orchestrator 47s → engineer failed (circle_detected, 251s) → engineer 198s
  → orchestrator 72s → qa 57s → security 25s → **dogfood failed
  (circle_detected, 48s)** → engineer 176s → orchestrator 70s → qa 91s →
  orchestrator 60s (loop guard: "repeated route without ticket-state
  change"; queue drained)
- **Target commits / tickets / docs produced:** 21 commits on demo-11
  `main` (pushed to local origin): CEO vision + goal, COO plan/feature
  contract for F-001, CTO scenario tickets, Engineer inventory-api Go
  implementation (models/storage/handlers/main) with unit + integration
  tests, a `security: audit 2026-06-12` commit, three QA-driven
  reopen-for-rework → done cycles on T-001, and the exported
  `docs/QUALITY_SCORE.md`. Ticket state at end: T-001 in `done/`, no
  backlog/in-progress tickets remaining.
- **Telemetry highlights:** Factory Pace and Convergence And Guardrails
  tables captured in
  [../baselines/2026-06-12-factory-pace-baseline.md](../baselines/2026-06-12-factory-pace-baseline.md)
  (T-027 sections rendered live from this run). 18 positive / 4 negative
  terminal outcomes; 3 circle_detected + 1 max_turns convergence failures;
  guardrail_block triage concentrated on engineer (×51).
- **Product progress reached:** working Go inventory API with CRUD handlers,
  in-memory storage, health endpoint, and passing `go test` suites; QA,
  security, and three rework loops all executed against real code.
- **Target intervention-debt count:** 0 open / 0 total (foundation-owned
  signals were kept out of the target backlog by the serve-side filter).
- **Runtime artifacts:** traces under `~/.mars-harness/traces/` for all 22
  jobs; per-repo DB retained.
- **Stop reason (corrected):** originally recorded as "natural end — queue
  drained and the final orchestrator survey was stopped by the loop guard."
  The second monitor shift corrected this: the orchestrator was stopped by
  the operator at 01:21:45 BST to start the demo-12 replay, leaving engineer
  job `4b659db8` (dogfood-failure rework, enqueued 01:21:44 BST) orphaned in
  `pending` — it never ran. The stop was operator preemption with one
  undrained job, not a drained queue.

## Operator interventions

1. One `POST /api/run-role {"role":"qa"}` retry (dashboard control surface)
   after the qa circle_detected runtime failure, which the orchestrator
   correctly refused to redispatch ("foundation telemetry or operator retry
   must resolve it first"). The retried qa job completed in 46s and the
   lifecycle continued through security, dogfood, and further rework.

## Independent observer (read-only replay monitor)

A separate read-only replay-monitor agent watched this run across two
shifts (the second 00:41–02:42 BST) without write access to the target or
this report. First-shift verification, second-shift corrections, and the
monitor's overall verdict:

- **Product output independently verified:** the monitor confirmed real
  product delivery via three independent surfaces — the per-repo DB job
  outcomes (18 positive / 4 negative terminals), the demo-11 git history
  (21 commits on `main` pushed to the local origin, including the three
  QA-driven rework cycles and the security audit commit), and the on-disk
  tree (working Go inventory API with test suites, T-001 in `done/`). The
  baseline's product-progress claim does not rest solely on the build
  agent's self-report.
- **Correction — the run did not terminate naturally:** the orchestrator was
  stopped at 01:21:45 BST (preempted to start demo-12), orphaning engineer
  job `4b659db8` (dogfood-failure rework, enqueued 01:21:44 BST) in
  `pending`. Foundation-owned finding: sequential preemption leaves queues
  undrained with no disposition — distinct from T-031's operator-retry
  routing gap. Ticket: T-035.
- **Correction — internally inconsistent final state:** T-001 sits in
  `done/`, but the last QA job (`9c049078`, 01:10–01:12 BST) returned
  `changes_requested` citing build/runtime validation failures, and dogfood
  never passed (`ff7b701e` circle_detected at 01:04 BST). The per-role pace
  data remains valid measurement; the lifecycle-health claim carries this
  caveat. The final build/runtime QA failure is classified mixed/unclear
  (target build state vs reviewer procedure) pending triage.
- **Convergence events (full second-shift inventory):** qa `497d29c6`
  circle_detected (00:44 BST), engineer `2be4ad82` circle_detected during
  security rework (00:54 BST), dogfood `ff7b701e` circle_detected
  (01:04 BST); 65 guardrail_block events across the run (compare demo-12:
  88, demo-13: 12).
- **Post-failure orchestration gap:** the monitor observed the same
  "not dispatching runtime failure through Orchestrator; foundation
  telemetry or operator retry must resolve it first" halt after the qa
  circle_detected failure that the Operator interventions section records.
  This is one instance of the failure-class-independent gap tracked under
  T-031.
- **Deployed-owned signals (target product, not foundation):** missing
  automated tests flagged by QA `54e57be2`, and a hardcoded API key with
  insecure defaults flagged by security `1929fab4`. These belong in the
  demo-11 target backlog, not foundation doctrine.
- **Monitor verdict: amber/degraded.** The factory delivers real product
  output, but convergence failures are systemic on the balanced profile:
  the coding-tier context ceiling (T-032), guardrail-churn turn burn, and
  the preemption/draining gap (T-035).

## Findings (failure ownership classification)

### F1 (foundation-owned, convergence): qa circle_detected mid-review with no terminal disposition

The second qa review (job `497d29c6`) performed a productive review (file
reads, `docsync_audit` clean, focused `go test` pass) and then repeated
itself into the circle detector at turn 18/20 without recording a
`job_disposition_record`. The orchestrator's runtime-failure policy then
stalled the lifecycle until an operator retry. Ownership: foundation
(reviewer convergence / disposition discipline under the balanced model).
Evidence kept for Phase 3; same family as the dogfood finding below.

### F2 (foundation-owned, convergence + lifecycle reach): dogfood circle_detected is never retried, ending the run before release

The single dogfood job failed with circle_detected in 48s. Runtime failures
are not redispatched by design, and no operator retry was performed (the run
was allowed its natural end for baseline purposes), so the lifecycle ended
at the loop guard without a dogfood pass or release-manager stage. Ownership:
foundation. Phase 3 should either harden terminal-role convergence (shared
root cause with F1) or give runtime convergence failures a bounded retry
path so a single circle does not silently cap lifecycle reach.

### F3 (foundation-owned, observation): engineer limit stops self-recover but dominate pace

Engineer accounted for both remaining convergence failures (1 max_turns on
the first scaffold attempt, 1 circle_detected) and the slowest pace row
(75.2 avg turns, 244.7s avg wall). Queue retries recovered both failures
without intervention. No ticket yet — this is the headline optimization
target T-011 Phase 3 already owns.

## Pass/fail against AD-285

Pass on lifecycle reach, with corrected caveats. The prior recorded baseline
for this archetype (`demo-inventory-api-run65`) reached Engineer product
rework; this run goes materially further (three QA rework loops, two
security audits, a dogfood attempt, T-001 done), so the lifecycle-reach bar
is met. The heavy-model F1 wedge from 2026-06-11 (ticket T-030) did not
recur: cto-weekly created scenario tickets and converged cleanly in 205s.
Caveats from the second monitor shift (Independent observer section): the
stop was operator preemption with one undrained rework job, not a drained
queue, and the final state is internally inconsistent (T-001 in `done/`
against a final QA `changes_requested` and no dogfood pass). Future
comparisons against this baseline must use the per-role pace rows, not the
end-state as a converged-lifecycle exemplar.

## Validation confirmations

- **AD-218 (post-validation engineer no-ops):** confirmed live on this
  archetype — six engineer jobs including three QA-driven rework cycles
  produced 0 no-op outcomes in the Convergence And Guardrails export and "No-op
  runs: None recorded" in Evidence Signals.
- **T-027 / AD-283 (convergence + guardrail telemetry):** confirmed live a
  second time — both export sections rendered from this run's traces with
  non-zero convergence counters (3 circle_detected, 1 max_turns).
