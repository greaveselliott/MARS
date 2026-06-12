# Validation Report: demo-12 Package-Managed Frontend Baseline (balanced model)

**Date:** 2026-06-12
**Author:** foundation-maintainer
**Purpose:** First recorded baseline replay for the package-managed-frontend
archetype (T-029 archetype-gap closure; AD-284 validation matrix row).
Recorded per the AD-285 evidence contract including model identity.

## Run 1: Vite/React habit tracker on v0.50.2 — 2026-06-12

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-12-balanced-frontend-start.log`
- **Target:** `/path/to/local-redacted` (fresh git
  repo seeded with a Vite/React habit-tracker brief; local bare origin
  `demo-12-origin.git`; per-repo DB cleared before the run)
- **Source ref / binary:** `mars-harness 0.50.2` built from `c0ebceb` on
  `codex/main-lifecycle-stabilization-rebased`
- **Model identity (AD-285):** reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072 reasoning :18081,
  ctx 32768 coding :18080); fast = `google_gemma-4-E4B-it-Q5_K_M.gguf`;
  resolved from `performance_profile: balanced`
- **Database / logs:** `~/.mars-harness/db/demo-12/mars.db`;
  `~/.mars-harness/traces/logs/demo-12-balanced-frontend-start.log`
- **Job sequence (10 jobs, ~62 min):** ceo 65s → coo 144s → cto-weekly 344s
  → orchestrator 96s → cto-weekly 166s → engineer failed (max_turns, 308s)
  → engineer failed (ticket gate, 256s) → engineer failed
  (**context_overflow**, 285s) → [operator retry] → engineer failed (ticket
  gate, 303s) → engineer failed (**context_overflow**, 190s); orchestrator
  survey then paused for the dirty target workspace
- **Target commits / tickets / docs produced:** CEO vision, COO plan +
  F-001 feature contract, CTO scenario tickets (T-001..S004 coverage),
  Engineer scaffolded a root-level Vite + React project (`package.json`,
  `vite.config.js`, `src/`, `index.html`) with `npm install` and a `dist/`
  build, claimed T-001, and committed the initial structure. T-001 remained
  in progress at stop; final wedge state checkpointed as
  `9e0f122 chore(evidence): checkpoint engineer context-overflow wedge
  state + scores export` and pushed to the local origin.
- **Telemetry highlights (`scores export`):** ceo 36.0 turns/64.5s; coo
  42.0/144.4s; cto-weekly 66.0 avg/255.1s avg over 2 jobs; engineer 80.6
  avg turns, 267.2s avg wall, **3 limit stops over 5 jobs** (1 max_turns,
  2 other limit stops); convergence section flags engineer as
  convergence-failure evidence; overall grade F.
- **Product progress reached:** building Vite/React scaffold with
  dependencies installed and `dist/` produced; no habit-tracker feature
  logic landed before the wedge.
- **Target intervention-debt count:** 0 open in target backlog
  (context_overflow signals were routed to foundation telemetry).
- **Runtime artifacts:** traces for all 10 jobs; per-repo DB retained.
- **Stop reason:** deterministic engineer wedge — coding-tier context
  overflow (32,883 and 33,281 prompt tokens vs 32,768 ctx) reproduced
  across an operator retry; run stopped by operator after the second
  overflow.

## Operator interventions

1. One `POST /api/run-role {"role":"engineer"}` retry after the first
   context_overflow runtime failure; the retry reproduced the overflow
   (ticket gate fail, then 32,883 tokens > 32,768), confirming the wedge is
   deterministic for this archetype state.

## Independent observer (read-only replay monitor)

A separate read-only replay-monitor agent watched the demo-12 session and
recorded one pre-run failure that the build-side record above does not
capture:

- **Session-opening CEO failure (~2s, `model_unavailable`):** before the
  recorded Run 1 sequence, the first CEO dispatch failed in ~2 seconds with
  `model_unavailable` because the operator had swapped to
  `performance_profile: balanced` — which resolves the reasoning/coding
  tiers to `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` — while only the
  quality-profile `Q8_0` weights were installed. The session proceeded only
  after the operator downloaded the Q4_K_M artifact. Classification:
  **foundation-owned** (setup/profile mismatch; a profile change after
  setup leaves the AD-032 download-marker verification stale). Ticket:
  T-033.
- **Post-failure orchestration gap:** after the `model_unavailable` failure
  the monitor observed the same "not dispatching runtime failure through
  Orchestrator; foundation telemetry or operator retry must resolve it
  first" halt seen after max_turns and circle_detected failures on demo-11
  — the gap is failure-class-independent (tracked under T-031).
- **Guardrail churn and post-max_turns cascades (second shift):** 88
  guardrail_block events on this run (highest of the three archetypes;
  demo-11: 65, demo-13: 12), with engineer `a397ebde` alone hitting 88
  blocks against browser-framework closure rules before max_turns. Two
  `ticket_gate` cascade failures followed max_turns stops (`28bd2736`,
  `04dc813d`) — post-failure handoff incompleteness: the gate repair job
  inherits a wedged state the failed job never dispositioned.
- **Monitor verdict: amber/degraded.** Real product output (working Vite
  scaffold) but systemic convergence failure on this archetype: the
  coding-tier context ceiling (T-032) compounded by guardrail-churn turn
  burn.

## Findings (failure ownership classification)

### F1 (foundation-owned, context assembly / model routing): engineer context assembly exceeds the balanced coding-tier 32k window on package-managed frontend targets

Once the Vite/React scaffold existed (root `package.json`,
`package-lock.json`, multi-file `src/`), engineer prompts crossed the
coding tier's 32,768-token window (33,281 then 32,883 tokens) and llama.cpp
rejected the request as non-retryable `exceed_context_size_error`. The
orchestrator correctly refused to redispatch, so the archetype cannot
progress past initial scaffolding. Ownership: foundation (context assembly
must fit the configured tier window — Tenet 9 — or the router must route
oversized engineer turns to the 131k reasoning endpoint, or the balanced
profile must raise the coding ctx). Compare: the heavy-model probe of this
archetype (2026-06-12 00:00–00:10, Q8_0) also wedged engineer but via
`dependency_sync` ENOENT against a nested project directory — both probes
wedge in engineer on this archetype, for different foundation-owned
reasons. Ticket: T-032.

### F2 (foundation-owned, observation): engineer ticket-gate failures consume retries without disposition

Two of the five engineer jobs ended with "ticket gate: engineer ended
without completing the claimed ticket" after burning their turn budget
mid-wedge. These are honest gate stops, but they alternate with the
overflow failures and add ~5 minutes per cycle without converging.
Covered by the same Phase 3 work as F1 (the gate is doing its job; the
context wedge is the root cause).

## Pass/fail against AD-284/AD-285

This is the first recorded baseline for the archetype, so there is no prior
lifecycle-reach bar; the run records the bar for future comparisons:
**lifecycle reach = engineer initial scaffold (T-001 in progress)**, wedged
on coding-tier context overflow. Matrix gating: source changes touching
context assembly, model routing, or dependency tooling must replay this
archetype and demonstrate reach beyond T-001 before claiming improvement.

**2026-06-12 superseded:** the Run 2 replay below (v0.50.11, AD-288 context
fix) cleared the wedge and exceeded this bar. Run 2 is the archetype pace
baseline from now on; Run 1 remains evidence of the wedge only.

## Run 2: T-032 context-fix replay on v0.50.11 — 2026-06-12 {#run-2-v05011-context-fix}

Validates T-032 / AD-288 (calibrated token estimation, served-window
budgeting, server-reported overflow clamp) against the archetype that
recorded the wedge. Source-change class: tool-policy-adjacent runtime
(agent loop, LLM client, inference routing) — package-managed frontend is
one of the two archetypes the matrix gate requires for this fix.

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-12-balanced-frontend-replay-v0.50.11.log`
- **Target:** `/path/to/local-redacted` reset to the
  seed commit `b6aa7a3` (`.git` + `spec.md` only, `git clean -fdx`), local
  bare origin `demo-12-origin.git` force-pushed back to the seed; per-repo
  DB `~/.mars-harness/db/demo-12` removed before the run
- **Source ref / binary:** `mars-harness 0.50.11` built from `12af153` on
  `codex/main-lifecycle-stabilization-rebased` (fix commit `bee4f5b`,
  tag `v0.50.11`)
- **Model identity (AD-285):** unchanged from Run 1 — reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072 reasoning :18081,
  ctx 32768 coding :18080); fast = `google_gemma-4-E4B-it-Q5_K_M.gguf`;
  resolved from `performance_profile: balanced` — pace comparison with
  Run 1 is therefore valid
- **Database / logs:** `~/.mars-harness/db/demo-12/mars.db`;
  `~/.mars-harness/traces/logs/demo-12-balanced-frontend-replay-v0.50.11.log`
- **Job sequence (9 jobs, 02:18:17–02:53 UTC; ~47 min):**
  ceo `62defe5f` 34s → coo `8c64c5a2` 126s → cto-weekly `01aa5f83` 49s →
  cto-weekly `b8cafbfa` 104s → engineer `c3a6da4a` failed (max_turns,
  235s) → engineer `552bab1b` completed (151s; **T-001 claimed,
  implemented, moved to done**) → qa `9bfcfb6e` failed (max_turns, 85s) →
  [operator retry via `POST /api/run-role`, mirroring Run 1's retry] →
  engineer `e81444cc` failed (max_turns, 368s; **claimed T-002 and landed
  the feature commit**) → engineer `cefd6681` failed (max_turns, 327s)
- **Context-fix evidence (the claim under test):** **zero
  `context_overflow` telemetry events** (Run 1: 2 in 5 engineer jobs) and
  zero `exceed_context_size` rejections. The budget pruner engaged twice at
  exactly the former wedge state, both inside engineer jobs against the
  fully populated Vite/React tree:
  - 03:46:33 BST (job `e81444cc`): 33,901 est tokens → pruned 34 oldest
    tool messages → 27,690 (window 32,768, target 27,852)
  - 03:51:42 BST (job `cefd6681`): 32,991 est tokens → pruned 14 →
    27,665
  Run 1 wedged at 33,281/32,883 served tokens in this identical state.
- **Target commits / tickets / docs produced:** CEO vision, COO plan +
  feature contract, CTO scenario tickets; engineer scaffolded the Vite +
  React project, **closed T-001** (claim → implement → evidence → done),
  then **claimed T-002 and committed
  `02304b0 feat: implement basic habit tracking functionality (T-002)`**.
  Final state checkpointed as `772de25 chore(evidence): checkpoint
  v0.50.11 context-fix replay state + scores export` and pushed to the
  local origin.
- **Telemetry highlights:** ceo 13 turns/34s; coo 23/126s; cto-weekly
  9/49s and 18/104s; engineer 36–51 turns per job; failure categories:
  max_turns 4 (engineer ×3, qa ×1), guardrail_block 65,
  **context_overflow 0**
- **Product progress reached:** working Vite/React scaffold with T-001
  closed and T-002 habit-tracking feature logic committed — beyond Run 1's
  bar (T-001 stuck in progress at the wedge)
- **Target intervention-debt count:** 0 open in target backlog (max_turns
  and guardrail signals routed to foundation telemetry, as in Run 1)
- **Runtime artifacts:** traces for all 9 jobs in the per-repo DB; full
  debug log; scores export committed to the target
- **Stop reason:** operator graceful stop (`POST /api/stop`) after the
  success criteria were met and the orchestrator entered the known
  post-max_turns dispatch halt (T-031 scope)

### Run 2 pass/fail against AD-284/AD-285

**Pass.** The rerun exceeds Run 1's lifecycle reach (T-001 closed + T-002
feature commit vs T-001 in progress); the failure signature the change
claimed to fix (`context_overflow` / `exceed_context_size_error`) did not
reappear under the exact conditions that produced it deterministically in
Run 1 and an operator retry; no new foundation-owned failure class
appeared; target intervention debt did not increase. Residual failures are
pre-existing classes explicitly out of this slice's scope and recorded as
evidence: engineer/qa `max_turns` with post-failure dispatch halt (T-031,
AD-286 state-machine scope) and browser-framework guardrail churn (65
blocks, Run 1: 88). **This run replaces Run 1 as the package-managed
frontend archetype pace baseline** (Run 1 recorded only the wedge).
