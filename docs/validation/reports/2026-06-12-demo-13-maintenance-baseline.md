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

**2026-06-12 superseded:** the Run 2 replay below (v0.50.11, AD-288 context
fix) cleared the wedge and exceeded this bar. Run 2 is the archetype pace
baseline from now on; Run 1 remains evidence of the wedge only.

## Run 2: T-032 context-fix replay on v0.50.11 — 2026-06-12 {#run-2-v05011-context-fix}

Validates T-032 / AD-288 on the existing-repo-maintenance archetype — the
second of the two archetype replays the matrix gate requires for this fix
(see the demo-12 report Run 2 for the first).

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-13-balanced-maintenance-replay-v0.50.11.log`
- **Target:** `/path/to/local-redacted` restored to
  its pre-run committed state `fb08fdf` (maintenance brief; prior
  Phaser/Tetris product history and harness scaffold retained per the
  archetype definition), `git clean -fdx`, local bare origin force-pushed
  back to `fb08fdf`; per-repo DB `~/.mars-harness/db/demo-13` removed
  before the run
- **Source ref / binary:** `mars-harness 0.50.11` built from `12af153` on
  `codex/main-lifecycle-stabilization-rebased` (fix commit `bee4f5b`,
  tag `v0.50.11`)
- **Model identity (AD-285):** unchanged from Run 1 — reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072 reasoning :18081,
  ctx 32768 coding :18080); fast = `google_gemma-4-E4B-it-Q5_K_M.gguf`;
  resolved from `performance_profile: balanced` — pace comparison with
  Run 1 is valid
- **Database / logs:** `~/.mars-harness/db/demo-13/mars.db`;
  `~/.mars-harness/traces/logs/demo-13-balanced-maintenance-replay-v0.50.11.log`
- **Job sequence (8 jobs, 02:57:08–03:36 UTC; ~42 min, queue drained):**
  ceo `9ec8f113` 99s → engineer `4b2a331c` failed (max_turns, 616s) →
  coo `554404ff` 99s → cto-weekly `d1be85e4` 163s → engineer `09eab37b`
  failed (max_turns, 495s) → engineer `6b4b882c` failed (max_turns,
  495s) → engineer `1f67b7ca` failed (max_turns, 268s) → engineer
  `b7cbd006` failed (max_turns, 375s)
- **Context-fix evidence (the claim under test):** **zero
  `context_overflow` telemetry events** (Run 1: 2 of 2 engineer jobs
  overflowed at 32,923 tokens) and zero `exceed_context_size` rejections
  across five engineer jobs against the existing multi-file project. The
  budget pruner engaged **12 times** (04:07–04:40 BST), every time from
  over-window estimates back inside the margin, e.g. 33,090→27,795,
  34,601→27,192, 34,779→27,709, 32,818→26,103 (window 32,768, target
  27,852) — each of these prompts would have wedged the Run 1 binary.
- **Target commits / tickets / docs produced:** planning roles re-surveyed
  the repo and refreshed plan + feature contract; engineer claimed T-001,
  landed multiple real product commits — core Tetris mechanics
  (`a6a3f68`, `de8e829`), project foundation (`ea1c228`), Phaser
  scene-binding fix (`86a0627`) — and three evidence/verification ticket
  updates ending at `27132e2 chore(tickets): update T-001 evidence and
  ready for move to done`. Final state checkpointed as `a63f804
  chore(evidence): checkpoint v0.50.11 context-fix replay state + scores
  export` and pushed to the local origin.
- **Telemetry highlights:** ceo 22 turns/99s; coo 16/99s; cto-weekly
  20/163s; engineer 51 turns per job (turn-capped); failure categories:
  max_turns 5 (all engineer), guardrail_block 70, **context_overflow 0**
- **Product progress reached:** repeated engineer iterations on the
  claimed maintenance ticket — mechanics implementation, bug-fix commit,
  and validation/evidence cycles to "ready for move to done" — versus
  Run 1's single product commit before the wedge
- **Target intervention-debt count:** 0 open in target backlog
  (max_turns/guardrail signals routed to foundation telemetry)
- **Runtime artifacts:** traces for all 8 jobs in the per-repo DB; full
  debug log; scores export committed to the target
- **Stop reason:** operator graceful stop (`POST /api/stop`) after the
  queue drained into the known post-max_turns dispatch halt (T-031 scope)

### Run 2 independent observer cross-check (read-only replay monitor)

The independent monitor finished cross-checking this rerun on 2026-06-12
and **confirms the T-032 gate: PASS, zero context overflows** — with one
honesty correction against the builder-side record:

- **All five engineer jobs failed max_turns, zero completed:** `4b2a331c`,
  `09eab37b`, `6b4b882c`, `1f67b7ca`, `b7cbd006`; the longest ran 1,482s
  without converging. max_turns is the dominant terminal failure on this
  archetype post-fix.
- **Lifecycle-reach correction:** the builder record's "ticket evidence
  cycles toward done" was rosier than the monitor data — **T-001 never
  moved out of in-progress** despite 6+ real product commits. The reach
  delta vs Run 1 still holds (multiple product commits vs one commit then
  wedge), but the run produced no closed ticket.
- **Guardrail churn regressed 483% on this archetype:** 12 → 70
  guardrail_block events — turn burn shifted from context overflow to
  guardrail-fighting once engineer iterations ran long enough to accumulate
  blocks. Recorded as AD-286 evidence (agents unable to find the permitted
  transition); the block-message contract and T-029 slices own this next
  frontier, not the T-031 routing fix.
- **T-035 negative evidence:** zero orphaned pending jobs at orchestrator
  exit — the graceful-stop draining gap did not reproduce under natural
  termination.

### Run 2 pass/fail against AD-284/AD-285

**Pass** (independently confirmed by the monitor cross-check above, with
the lifecycle-reach wording corrected per that section). The rerun exceeds
Run 1's lifecycle reach (multiple product commits and repeated evidence
iterations vs one commit then wedge — though T-001 itself never closed);
the failure signature this change claimed to fix did not reappear despite
12 over-window prompt states that deterministically wedged Run 1; no new
foundation-owned failure class appeared; target intervention debt did not
increase. Residual failures are pre-existing classes out of this slice's
scope, recorded as evidence: engineer `max_turns` convergence churn with
post-failure dispatch halt (T-031, AD-286 state-machine scope) and
guardrail churn (70 blocks vs Run 1's 12 — engineer iteration on this
archetype now runs long enough to accumulate blocks that the wedge
previously cut short). **This run replaces Run 1 as the
existing-repo-maintenance archetype pace baseline** (Run 1 recorded only
the wedge).
