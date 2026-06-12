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

**OPEN.** The AD-287 final-sequence checkpoint is half-satisfied: demo-12
Run 4 (frontend leg) **passed with no extraction drift**; the API leg must
be replayed after the pause. Per AD-284, the T-043 lifecycle claim stays
unconfirmed until this replay completes; T-043 remains in progress with
the replay command recorded above.
