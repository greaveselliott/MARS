# Validation Report: demo-15 AD-287 Final-Sequence Checkpoint (Inventory/API archetype, balanced model)

**Date:** 2026-06-12
**Author:** foundation-maintainer
**Purpose:** AD-287 final-sequence extraction checkpoint, API/service leg of
the two-archetype tool-policy minimum (AD-284), validating the T-043
validation-lane extraction (v0.50.24) on a fresh Inventory/API target.
demo-14's backlog was already consumed by its own run, so this uses a fresh
target with a fresh equivalent brief per the demo-14 convention (newly
written, not a copy). Recorded per the AD-285 evidence contract. The
frontend leg of this checkpoint is demo-12 Run 4 in
`2026-06-12-demo-12-frontend-baseline.md`.

## Target seed

- **Target:** `/path/to/local-redacted` — fresh git
  repo seeded with a newly written Depot Supplies API brief (Go
  standard-library JSON API: register supply items, list, fetch by item
  code, stock receipts/withdrawals, reorder-threshold restock report,
  validation errors, health endpoint); seed commit `62861dc`; local bare
  origin `demo-15-origin.git`.

## Run 1: operator-preempted (PAUSED — evidence-only, not checkpoint evidence) — 2026-06-12

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-15-api-ad287-final-checkpoint-v0.50.24.log`
- **Source ref / binary:** `mars-harness 0.50.24` built from `ffa2629`
  (extraction commit `5cd4eb3`, tag `v0.50.24`), installed via `make install`
- **Model identity (AD-285):** reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072 reasoning :18081,
  ctx 32768 coding :18080); fast = `google_gemma-4-E4B-it-Q5_K_M.gguf`;
  resolved from `performance_profile: balanced`
- **Database / logs:** `~/.mars-harness/db/demo-15/mars.db`;
  `~/.mars-harness/traces/logs/demo-15-api-ad287-final-checkpoint-v0.50.24.log`
- **Job sequence (4 jobs, 09:52:15–~10:04 UTC; preempted at ~12 min):**
  ceo `ab2b16fc` completed → coo `1983b1ee` failed (max_turns; churned
  five feature-contract rewrite commits) → coo `ed5a5b7b` (AD-289
  automatic retry) preempted by the operator stop (recorded
  llm_unreachable: context canceled — stop artifact, not a runtime
  failure) → dogfood `e8ee85a9` **left pending at exit (orphaned by
  preemption — the known T-035 graceful-stop draining gap reproduced
  under operator preemption, matching its original demo-11 shape)**
- **Target output at stop:** CEO vision/goals commit plus five COO
  planning commits (F-001 superseded by F-002 supply-item-management
  contract churn); worktree left dirty by the preempted COO retry
  (`.harness/learnings.yaml`, `docs/features/F-002-…md` modified,
  uncommitted)
- **Stop reason:** operator pause request mid-session; graceful stop
  (`POST /api/stop`, `{"ok":true}`); harness exited 0 and shut down its
  llama-server subprocesses
- **Verdict:** evidence-only. This run is **not** usable as the API-leg
  checkpoint: the lifecycle was preempted before Engineer ever ran, so no
  validation-lane rules fired. The COO max_turns + contract-churn shape is
  a pre-existing convergence class (AD-286 evidence), not extraction
  drift. On resume the run must be discarded and replayed from a clean
  reset: reset demo-15 to `62861dc` (`git reset --hard 62861dc && git
  clean -fdx`), force-push `demo-15-origin.git` back to the seed, delete
  `~/.mars-harness/db/demo-15`, and rerun the exact command above to
  natural queue drain (demo-14 took ~126 min).

## Checkpoint status

**PASS — no extraction drift (Run 2).** The AD-287 two-archetype final-sequence
checkpoint is satisfied: demo-12 Run 4 (frontend leg) passed with no extraction
drift; demo-15 Run 2 (API leg) drained naturally with convergence failures only
(max_turns, circle_detected, ticket_gate) — no validation-lane rule regression,
zero `context_overflow`, zero policy panics. T-043 closed 2026-06-12.

## Run 2: clean-seed replay to natural drain — 2026-06-12

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-15-api-ad287-final-checkpoint-v0.50.24.log`
- **Pre-run reset:** `git reset --hard 62861dc && git clean -fdx`; force-push
  `demo-15-origin.git`; deleted `~/.mars-harness/db/demo-15`
- **Source ref / binary:** `mars-harness 0.50.24` (extraction commit `5cd4eb3`,
  tag `v0.50.24`)
- **Model identity (AD-285):** balanced profile — reasoning/coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf`; fast =
  `google_gemma-4-E4B-it-Q5_K_M.gguf`
- **Database / logs:** `~/.mars-harness/db/demo-15/mars.db`;
  `~/.mars-harness/traces/logs/demo-15-api-ad287-final-checkpoint-v0.50.24.log`
- **Wall clock:** 2026-06-12 10:55 UTC → 18:39 UTC (~7h 44m job span; orchestrator
  uptime ~8h 58m including idle tail)
- **Job totals:** 80 jobs — 67 completed, 12 failed, 1 cancelled; **0
  pending/running at drain**
- **Role summary:** ceo 1✓; coo 1✓; cto-weekly 7✓; engineer 13✓/9✗; qa 14✓;
  security 8✓; dogfood 5✓/3✗; orchestrator 18✓
- **Failure mix:** 9× engineer `max_turns`, 1× engineer `circle_detected`, 2×
  dogfood convergence (circle + max_turns), ticket_gate messages on later cycles
  — all pre-existing AD-286 convergence classes; **0× `context_overflow`**
- **Target output:** 56 commits since seed `62861dc`; walking-skeleton tickets
  closed through multiple engineer → qa → security → dogfood cycles (T-001,
  T-002, T-004..T-006); Go Depot Supplies API with README and handler fixes
- **Guardrail / validation-lane assessment:** guardrail_block telemetry present
  (dirty workspace, ticket lifecycle) with no policy errors; validation-lane rules
  moved in T-043 slice did not introduce new failure classes — failures match
  demo-14 Inventory/API convergence shape, not extraction drift
- **Operator interventions:** deployed-owned compile/router wedge on first T-001
  cycle required external repair before QA dispatch; subsequent cycles completed
  within harness dispatch (AD-289 automatic retries on several roles)
- **Telemetry:** `mars-harness scores export --repo demo-15` →
  `docs/QUALITY_SCORE.md` (overall grade D — convergence-heavy, expected for
  first API archetype run)
- **Stop reason:** natural queue drain (orchestrator idle, `active_jobs: 0`)

### Run 2 pass/fail against AD-284/AD-285

**Pass — no extraction drift.** Lifecycle reached product delivery, review, and
dogfood repeatedly; queue drained; zero context_overflow; failure categories are
convergence/ticket-gate only (same family as pre-extraction baselines). Operator
deployed wedge on first engineer cycle is recorded but does not indicate
validation-lane policy regression from `policy_validation.go` extraction.
