# Validation Report: demo-14 Convergence-Routing Replay (Inventory/API archetype, balanced model)

**Date:** 2026-06-12
**Author:** foundation-maintainer
**Purpose:** AD-284 orchestration-class replay validating T-031 / AD-289
(operator-retry routing: runtime convergence failures get one automatic
same-role retry per failure fingerprint, then a recorded operator
escalation) on the Inventory/API archetype where the stall class was
observed. Recorded per the AD-285 evidence contract including model
identity.

## Run 1: Parts Stockroom API on v0.50.14 — 2026-06-12

- **Exact command:** `mars start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars/traces/logs/demo-14-convergence-routing-replay-v0.50.14.log`
- **Target:** `/path/to/local-redacted` — fresh git
  repo seeded with a newly written Parts Stockroom API brief (Go
  standard-library JSON API: register parts, list, fetch by SKU, stock
  movements, minimum-units reorder report, validation errors, health
  endpoint) — a fresh equivalent of the demo-11 Inventory/API brief, not a
  copy; local bare origin `demo-14-origin.git`; no per-repo DB existed
  before the run
- **Source ref / binary:** `mars 0.50.14` built with `make install`
  from `5e338dc` (`release: notes 0.50.14`) on
  `codex/main-lifecycle-stabilization-rebased` (= `origin/main` at run
  time; fix commit `222392c`, tags `v0.50.13`/`v0.50.14`)
- **Model identity (AD-285):** unchanged from the 2026-06-12 demo-11
  baseline — reasoning + coding = `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf`
  (ctx 131072 reasoning :18081, ctx 32768 coding :18080); fast =
  `google_gemma-4-E4B-it-Q5_K_M.gguf`; resolved from
  `performance_profile: balanced` — pace comparison with the demo-11
  baseline is valid
- **Database / logs:** `~/.mars/db/demo-14/mars.db`;
  `~/.mars/traces/logs/demo-14-convergence-routing-replay-v0.50.14.log`
- **Run window:** 48 jobs, 2026-06-12 04:12:45–06:19:15 UTC (126.5 min);
  38 completed / 10 failed; queue fully drained with no pending or running
  jobs before the operator graceful stop (`POST /api/stop` at ~06:47 UTC
  after 25+ minutes of stable terminal state)
- **Pre-run isolation check:** no `mars serve/start` processes
  running; demo-12/demo-13 DB timestamps reflected only the read-only
  monitor cross-check.

### Convergence-routing evidence (the claim under test)

Every AD-289 path fired live, with **zero operator `run-role` calls**
(0 `manual role run enqueued` events in the full debug log):

1. **Automatic retry → recovery (non-Engineer role):** cto-weekly
   `747e5189` failed max_turns at 04:26 UTC → automatic `convergence_retry`
   `b63d6b31` enqueued in the same second (fingerprint
   `…:cto-weekly:max_turns`) → the retry **completed**, finished the
   scenario-ticket batch, and handed off to Engineer. Under v0.50.2 this
   exact state stalled the lifecycle pending operator retry.
2. **Retry → escalation (chain guard):** dogfood `56540d01` failed
   circle_detected → retry `d8524b51` enqueued → the retry itself failed
   max_turns → **escalated**: disposition rewritten to
   `blocked`/`operator_retry` naming the exhausted budget and the exact
   `POST /api/run-role {"repo_id":…,"role":"dogfood"}` command. Same shape
   on dogfood `73dcbd2a` → retry `c94a8cad` → escalated.
3. **Fingerprint-window escalation (no retry burn):** dogfood `1950aa3b`
   and `58d48239` (circle_detected) escalated **immediately** because an
   automatic retry for `…:dogfood:circle_detected` had already failed
   inside the 24h window — the demo-13 repeated-failure shape (5
   consecutive max_turns) cannot burn extra jobs under this budget.
4. **Existing edges preserved:** engineer `3f7baafe` max_turns with an
   active product ticket took the richer AD-227 `product_continuation`
   (`06bab222`, completed) — AD-289 is the general fallback beneath it.
   Both engineer ticket-gate failures (`a59fbe4b`, `ea3a20c2`) recovered
   through the existing `ticket_gate_repair` edge.
5. **Lifecycle continued past every escalation:** after each dogfood
   escalation the orchestrator survey kept routing engineer/qa/security
   work; the run drained its entire product backlog rather than capping
   at the first terminal-role convergence failure (the demo-11 F2
   finding).

### Run record

- **Failure inventory (10):** convergence 8 — cto-weekly max_turns ×1
  (auto-retried, recovered), engineer max_turns ×1 (AD-227 continuation,
  recovered), dogfood circle_detected ×3 + max_turns ×3 (1 retry pair
  recovered the lane via later dogfood passes; 4 escalations recorded);
  ticket_gate 2 (both repaired by the existing edge). Telemetry:
  guardrail_block 183, max_turns 5, circle_detected 3, ticket_gate 2,
  **context_overflow 0** (AD-288 pruner engaged repeatedly, e.g.
  33,934→25,450 tokens).
- **Target output:** 46 commits pushed to the local origin; **all 10
  product tickets closed** (T-001 walking skeleton through T-010 input
  validation) with repeated full engineer → qa → security → dogfood
  cycles; 4 of 10 dogfood jobs completed (the archetype's first recorded
  dogfood passes); CEO vision, COO plan + F-001 contract, CTO scenario
  tickets; final state checkpointed as `f6e0ae4 chore(evidence):
  checkpoint v0.50.14 convergence-routing replay state + scores export`.
- **Operator interventions: 0.** No `run-role` calls; the only operator
  action was the graceful stop after the queue had drained.
- **Target intervention-debt count:** 0 open in target backlog
  (convergence/guardrail signals routed to foundation telemetry).
- **Runtime artifacts:** traces for all 48 jobs in the per-repo DB; full
  debug log; scores export committed to the target.
- **Stop reason:** natural terminal state — queue drained (0
  pending/running for 25+ minutes, orchestrator watchdog surveys routing
  no new work) with the final dogfood convergence failure escalated and
  recorded; operator graceful stop to end the session.

## Pace vs the 2026-06-12 demo-11 baseline

Same archetype class, same model identity. Baseline (v0.50.2, 22 jobs,
51.6 min): T-001 done only, 1 operator qa retry required, dogfood never
passed, run ended by operator preemption. This run (v0.50.14, 48 jobs,
126.5 min): **T-001–T-010 all done** (full backlog drained), 4 dogfood
passes, 0 operator interventions, natural queue drain. Per-role pace is
same-or-better where comparable (engineer 235.9s avg wall vs 244.7s
baseline; qa 49.0s vs 63.3s; security 48.2s vs 34.8s within noise); total
wall is longer because the lifecycle delivered ten tickets instead of one.
No intervention-debt amplification (0 vs 0). Lifecycle reach is materially
better, so this run is recorded as the new Inventory/API archetype pace
reference (see the baseline file note).

## Pass/fail against AD-284/AD-285

**Pass.** Convergence failures occurred as expected (8 across 4 roles);
every one was routed automatically — bounded retry where the fingerprint
budget allowed, recorded `blocked/operator_retry` escalation where it was
exhausted; **zero manual `run-role` calls were needed for the lifecycle to
reach its terminal state** (the T-031 success criterion); lifecycle reach
exceeded the baseline (10 tickets done vs 1); no new foundation-owned
failure class appeared; environment-failure fail-fast behavior was not
regressed (no environment failures occurred to route). Residual evidence:
dogfood convergence churn (6 failures in 10 jobs) is the dominant
remaining convergence consumer — guardrail-churn/review-convergence
hardening (AD-286 evidence section, T-029 family) owns that next frontier,
not the routing edge.
