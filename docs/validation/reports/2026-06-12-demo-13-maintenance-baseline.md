# Validation Report: demo-13 Existing-Repo Maintenance Baseline (balanced model)

**Date:** 2026-06-12
**Author:** foundation-maintainer
**Purpose:** First recorded baseline replay for the existing-repo-maintenance
archetype (T-029 archetype-gap closure; AD-284 validation matrix row).
Recorded per the AD-285 evidence contract including model identity.

## Run 1: Phaser/Tetris maintenance target on v0.50.2 — 2026-06-12

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-13-balanced-maintenance-start.log`
- **Target:** `/path/to/local-redacted` — an
  existing repo with prior history (Phaser/Tetris game copied from the
  demo-10 session: prior harness scaffold, claimed ticket, in-progress
  product code) plus a maintenance brief recording a known start-screen
  bug; local bare origin `demo-13-origin.git`; no per-repo DB existed
  before the run
- **Source ref / binary:** `mars-harness 0.50.2` built from `c0ebceb` on
  `codex/main-lifecycle-stabilization-rebased`
- **Model identity (AD-285):** reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072 reasoning :18081,
  ctx 32768 coding :18080); fast = `google_gemma-4-E4B-it-Q5_K_M.gguf`;
  resolved from `performance_profile: balanced`
- **Database / logs:** `~/.mars-harness/db/demo-13/mars.db`;
  `~/.mars-harness/traces/logs/demo-13-balanced-maintenance-start.log`
- **Job sequence (5 jobs, ~45 min):** ceo 70s → engineer failed
  (**context_overflow**, 1079s) → coo 126s → cto-weekly 122s → engineer
  failed (**context_overflow**, 448s; 32,923 prompt tokens vs 32,768 ctx);
  orchestrator then withheld redispatch per runtime-failure policy
- **Target commits / tickets / docs produced:** CEO/COO/CTO surveyed the
  existing repo and produced plan, feature-contract, and ticket updates;
  engineer claimed T-004 and landed a real product commit
  (`feat(tetris): implement core game mechanics and UI elements for
  playable Tetris game (T-004 step 1)`) plus a learnings update before
  wedging. Final wedge state (modified `index.html`, `package.json`,
  `vite.config.js`) checkpointed as `c4fd580 chore(evidence): checkpoint
  engineer context-overflow wedge state + scores export` and pushed to the
  local origin.
- **Telemetry highlights (`scores export`):** ceo 34.0 turns/63.8s; coo
  40.0/126.0s; cto-weekly 42.0/121.4s; engineer 74.0 avg turns, 378.5s avg
  wall, **2 limit stops over 2 jobs** (both context_overflow); engineer
  flagged convergence-failure evidence; overall grade D.
- **Product progress reached:** real maintenance progress — core Tetris
  mechanics and UI commit on the claimed ticket before the wedge.
- **Target intervention-debt count:** 0 open in target backlog
  (context_overflow routed to foundation telemetry).
- **Runtime artifacts:** traces for all 5 jobs; per-repo DB retained.
- **Stop reason:** deterministic engineer wedge — coding-tier context
  overflow reproduced on both engineer jobs; run stopped by operator
  without a retry because demo-12 had already demonstrated the wedge is
  retry-stable on package-managed targets.

## Independent observer (read-only replay monitor)

The second monitoring shift (00:41–02:42 BST) cross-checked this run:

- **Guardrail churn was low on this archetype:** 12 guardrail_block events
  (vs demo-11: 65, demo-12: 88) — the existing-repo planning survey ran
  cleanly; turn burn concentrated entirely in the engineer context-overflow
  wedge (T-032).
- **Monitor verdict: amber/degraded** for the balanced-profile portfolio
  overall: real product output on every archetype, but systemic convergence
  failures (context ceiling, guardrail-churn turn burn, and the
  preemption/draining gap recorded on demo-11 as T-035).

## Findings (failure ownership classification)

### F1 (foundation-owned, context assembly / model routing): the T-032 coding-tier context overflow generalizes to existing package-managed repos

Both engineer jobs exceeded the balanced coding tier's 32,768-token window
once the existing multi-file Phaser/Tetris project plus harness context was
assembled (32,923 tokens observed). This is the same root cause as the
demo-12 frontend finding (T-032) on a different archetype: any
package-managed JS target with a non-trivial file tree wedges engineer at
the 32k coding window. Ownership: foundation. T-032 now carries both
archetype observations; whichever fix lands (context trimming, oversized
turns routed to the 131k reasoning endpoint, or larger coding ctx) must
replay **both** archetypes per AD-284.

## Pass/fail against AD-284/AD-285

First recorded baseline for the archetype; the bar for future comparisons:
**lifecycle reach = engineer real product commit on a claimed maintenance
ticket (T-004 step 1)**, wedged on coding-tier context overflow. Planning
roles (ceo/coo/cto-weekly) handled the existing-repo survey cleanly with no
convergence failures — the archetype gap is isolated to engineer context
assembly.
