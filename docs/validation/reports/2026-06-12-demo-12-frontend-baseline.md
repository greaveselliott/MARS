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

### Run 2 independent observer cross-check (read-only replay monitor)

The independent monitor finished cross-checking this rerun on 2026-06-12
and **confirms the T-032 gate: PASS, zero context overflows**. Cross-check
data recorded for the evidence trail:

- **max_turns is the dominant terminal failure post-fix:** 4 max_turns
  failures — engineer `c3a6da4a`, `e81444cc`, `cefd6681` and qa `9bfcfb6e`.
  One operator `run-role` retry was needed mid-run (the post-failure
  dispatch halt, T-031/AD-289 scope).
- **Guardrail churn improved on this archetype:** 88 → 65 guardrail_block
  events vs Run 1.
- **Lifecycle-reach delta confirmed:** T-001 closed and T-002 in progress
  with a real feature commit, vs Run 1's T-001 stuck in progress at the
  wedge.
- **T-035 negative evidence:** zero orphaned pending jobs at orchestrator
  exit — the graceful-stop draining gap did not reproduce under natural
  termination (it occurred under operator preemption on demo-11).

### Run 2 pass/fail against AD-284/AD-285

**Pass** (independently confirmed by the monitor cross-check above). The
rerun exceeds Run 1's lifecycle reach (T-001 closed + T-002
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

## Run 3: AD-287 slice-1 extraction checkpoint replay on v0.50.16 — 2026-06-12 {#run-3-v05016-ad287-slice1-checkpoint}

AD-287 per-slice gate: confirmatory checkpoint replay after extraction
slice 1 (T-036, `policy_browser.go` — pure same-package code motion of the
browser-framework static-analysis domain). Source-change class per AD-284:
tool policy. This is the frontend-archetype canary of the two-archetype
minimum; the change is behavior-preserving by construction, so the claim
under test is **zero rule-level guardrail drift** vs the Run 2 baseline.
The binary also carries AD-288 (v0.50.11) and AD-289/T-031 convergence
retry routing (v0.50.12–15), which the Run 2 baseline binary predates —
failure-routing deltas are attributable to those landed changes, not to
the extraction.

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-12-balanced-frontend-replay-v0.50.16.log`
- **Target:** `/path/to/local-redacted` reset to
  seed commit `b6aa7a3` (`.git` + `spec.md` only, `git clean -fdx`), local
  bare origin force-pushed back to the seed; per-repo DB
  `~/.mars-harness/db/demo-12` removed before the run
- **Source ref / binary:** `mars-harness 0.50.16` built from `7df6520`
  (extraction commit `f5a1d6a`, tag `v0.50.16`), installed via
  `make install`
- **Model identity (AD-285):** unchanged from Run 2 — reasoning + coding =
  `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072 reasoning :18081,
  ctx 32768 coding :18080); fast = `google_gemma-4-E4B-it-Q5_K_M.gguf`;
  resolved from `performance_profile: balanced` — pace comparison with
  Run 2 is valid
- **Database / logs:** `~/.mars-harness/db/demo-12/mars.db`;
  `~/.mars-harness/traces/logs/demo-12-balanced-frontend-replay-v0.50.16.log`
- **Job sequence (15 jobs, 08:05–08:47 BST; ~42 min):** ceo 55s → coo 142s
  → cto-weekly 143s → engineer `16989255` failed (max_turns, 431s) →
  engineer `c96e6337` failed (circle_detected, 116s; AD-289 bounded
  recovery job — escalated to operator after retry budget) → [operator
  retry via `POST /api/run-role`, mirroring Run 2's single retry] →
  engineer `c23b9bd8` failed (circle_detected, 58s) → engineer `73f10d29`
  failed (ticket gate, 302s) → engineer `0e6a9891` completed (159s;
  **T-001 moved to done**) → qa `c35067aa` completed (81s) → orchestrator
  completed (78s) → engineer `cf39e95e` failed (ticket gate, 190s) →
  engineer `167a35ef` completed (184s; **T-002 claimed and moved to
  done**) → qa `cd4d3b64` completed (52s) → orchestrator completed (48s)
  → engineer `0add1625` failed (llm_unreachable: context canceled —
  artifact of the operator graceful stop, not a runtime failure)
- **Rule-level guardrail verdict (the claim under test): PASS.** 83
  `guardrail_block` events, every one a pre-existing rule class: browser
  framework closure rules (post-build smoke-only, completion blockers,
  QA browser approval, browser ticket done-move, browser evidence
  population — all functions moved in slice 1, firing with identical
  message text), validation-lane blocks, claim-before-mutation gates,
  MarsDocSync file-write gates, no-op repetition, argv shell-syntax, and
  find-flood limits. Zero new or missing rule patterns, zero policy
  errors/panics. The moved browser domain fired correctly across
  engineer and QA jobs on both ticket cycles.
- **Pace and failure-mix deltas (attributed, not extraction drift):**
  failure mix shifted from Run 2's 4× max_turns to 1× max_turns + 2×
  circle_detected + 2× ticket_gate; one AD-289 bounded recovery job ran
  and one operator-escalation was recorded (`convergence failure
  escalated to operator after automatic retry budget`) — routing behavior
  landed in v0.50.12–15 (T-031/AD-289), absent from the Run 2 binary.
  Engineer guardrail churn per job: 73 blocks over 8 engineer jobs
  (~9.1/job) vs Run 2's 65 over 5 engineer jobs (13/job) — lower per-job
  churn, total higher only because the run progressed further.
- **Target commits / tickets / docs produced:** CEO vision, COO plan +
  F-001 contract, CTO scenario tickets; engineer landed
  `dfe8877 feat: implement first runnable product behavior for habit
  tracker`, **closed T-001 and T-002 through full engineer → QA →
  orchestrator cycles** (Run 2 reached T-001 closed + T-002 claimed).
  Final state checkpointed as `22f92f7 chore(evidence): checkpoint
  v0.50.16 AD-287 slice-1 replay state + scores export`, pushed to the
  local origin.
- **Telemetry highlights (`scores export`):** ceo 28 turns/42.7s; coo
  52/141.2s; cto-weekly 42/143.3s; engineer 66.7 avg turns/205.2s over 7
  traced jobs (3 limit stops); qa 36.5 avg/66.5s over 2 jobs (Run 2: qa
  failed max_turns); orchestrator 33 avg/62.8s over 2 jobs; failure
  categories: max_turns 1, circle_detected 2, ticket_gate 2,
  guardrail_block 83, **context_overflow 0** (AD-288 holds)
- **Product progress reached:** working Vite/React habit tracker with
  T-001 and T-002 closed and QA-approved — beyond Run 2's bar
- **Target intervention-debt count:** 0 open in target backlog (signals
  routed to foundation telemetry, unchanged)
- **Runtime artifacts:** traces for all 15 jobs in the per-repo DB; full
  debug log; scores export committed to the target
- **Stop reason:** operator graceful stop (`POST /api/stop`) after the
  success criteria were exceeded; the in-flight third-ticket engineer job
  was cancelled by the stop (recorded as llm_unreachable)

### Run 3 comparison table vs Run 2 baseline

| Metric | Run 2 (v0.50.11 baseline) | Run 3 (v0.50.16) | Verdict / attribution |
| --- | --- | --- | --- |
| Rule-level guardrail behavior | browser/validation/ticket/docsync rule classes | identical rule classes, identical message shapes, none new/missing | **PASS — no extraction drift** |
| guardrail_block total | 65 (5 engineer jobs) | 83 (8 engineer jobs; ~9.1/engineer-job vs 13) | per-job churn improved; total reflects longer run |
| Failure types | 4 max_turns | 1 max_turns, 2 circle_detected, 2 ticket_gate, 1 stop artifact | shift attributable to AD-289/T-031 routing (postdates baseline) |
| context_overflow | 0 | 0 | AD-288 holds |
| Lifecycle reach | T-001 closed, T-002 claimed + feature commit | **T-001 and T-002 closed**, 2 full QA cycles | exceeds baseline; attributable to AD-289 retry routing |
| Operator interventions | 1 run-role retry + stop | 1 run-role retry (after recorded AD-289 escalation) + stop | unchanged |
| Intervention debt (target) | 0 | 0 | unchanged |
| Model identity | balanced profile, Qwen3-Coder Q4_K_M + gemma-4-E4B Q5_K_M | identical | comparison valid |

### Run 3 pass/fail against AD-284/AD-285

**Pass.** The rerun reaches beyond the baseline lifecycle stage; the
extraction introduced zero new guardrail patterns or policy failures
(the specific drift signature this checkpoint exists to catch); no new
foundation-owned failure class appeared (circle_detected/ticket_gate are
pre-existing classes, newly *routed* by AD-289); target intervention debt
did not increase; the stop is operator-visible and recorded. AD-287's
slice-1 checkpoint is satisfied; per the AD's checkpoint policy,
subsequent pure-motion slices ride on the test suite until the final
slice of the extraction sequence. Run 3 does **not** replace Run 2 as the
pace baseline (Run 3 carries AD-289 routing changes that make pace
deltas vs Run 2 multi-causal; Run 2 remains the archetype baseline until
a deliberate re-baseline).

## Run 4: AD-287 final-sequence extraction checkpoint replay on v0.50.24 — 2026-06-12 {#run-4-v05024-ad287-final-checkpoint}

AD-287 final-sequence gate: confirmatory checkpoint replay after the last
extraction slice (T-043, `policy_validation.go` — pure same-package code
motion of the validation-lane/repair domain, the densest-firing rule family
in the system). The binary carries slices 2–8 (all pure motion) on top of
the Run 3 binary; no behavior-change commits landed between v0.50.16 and
v0.50.24, so the claim under test is **zero rule-level guardrail drift**
vs Runs 2/3. Source-change class per AD-284: tool policy (frontend canary
of the two-archetype minimum; the API archetype runs as demo-15 in
`2026-06-12-demo-15-api-ad287-final-checkpoint.md`).

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-12-balanced-frontend-replay-v0.50.24.log`
- **Target:** `/path/to/local-redacted` reset to
  seed commit `b6aa7a3` (`.git` + `spec.md` only, `git clean -fdx`), local
  bare origin force-pushed back to the seed; per-repo DB
  `~/.mars-harness/db/demo-12` removed before the run
- **Source ref / binary:** `mars-harness 0.50.24` built from `ffa2629`
  (extraction commit `5cd4eb3`, tag `v0.50.24` = `origin/main`), installed
  via `make install`
- **Model identity (AD-285):** unchanged from Runs 2/3 — reasoning +
  coding = `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` (ctx 131072
  reasoning :18081, ctx 32768 coding :18080); fast =
  `google_gemma-4-E4B-it-Q5_K_M.gguf`; resolved from
  `performance_profile: balanced` — comparison with Runs 2/3 is valid
- **Database / logs:** `~/.mars-harness/db/demo-12/mars.db`;
  `~/.mars-harness/traces/logs/demo-12-balanced-frontend-replay-v0.50.24.log`
- **Job sequence (24 jobs, 09:21:04–09:50:51 UTC; ~30 min):**
  ceo `58af4c99` 58s → coo `0420502f` 75s → cto-weekly `ad3ad112` 108s →
  engineer `c22a0910` completed (129s; **T-001 claimed, implemented,
  done**) → qa `424a2895` 40s → security `9d84ee50` 23s → dogfood
  `a6d0b126` 126s → engineer `4d2cc1ee` completed (96s; **T-002 done**) →
  qa `d81fed89` failed (circle_detected, 26s) → qa `cee153dd` completed
  (AD-289 automatic retry, 32s) → security `41b10fcd` 28s → dogfood
  `5f1ed3b0` 109s → cto-weekly `89544d0f` 99s → engineer `c10ea328`
  completed (94s; **T-003 done**) → qa `31a7851b` 31s → security
  `83ff653f` 24s → dogfood `7ef93e0b` failed (circle_detected, 55s) →
  dogfood `19b9cc18` completed (AD-289 automatic retry, 121s; created
  finding ticket T-004) → orchestrator `d061bd6b` 50s → engineer
  `b129deee` failed (ticket_gate, 101s) → engineer `af555cde` completed
  (existing `ticket_gate_repair` edge, 179s; **T-004 done**) → qa
  `7f411a12` 73s → orchestrator `62a95923` 68s → engineer `dadeaccb`
  failed (llm_unreachable: context canceled — artifact of the operator
  graceful stop cancelling the in-flight T-001 rework job)
- **Rule-level guardrail verdict (the claim under test): PASS.** 44
  `guardrail_block` events (engineer 18, dogfood 14, security 6,
  cto-weekly 3, qa 3), every one a pre-existing rule class with identical
  message shape: foreground long-running shell (8), engineer
  claim-before-mutation / reopen-before-rework gates (12 across
  T-001..T-004), disposition clean-tree gates (5), security
  review-shell limits (5), dogfood finding-disposition limits (5),
  **validation-lane rules moved in this slice firing identically** —
  `engineer already has successful validation and a clean implementation
  commit …` (post-validation completion gate, 2) and `qa already observed
  a failing build, test, or unexpected runtime validation …`
  (review-validation-failure shell gate consuming the moved classifiers,
  1) — CTO handoff coverage gate (1), argv shell-syntax (2), find-flood
  (1), shell arg-parse rejection (1). Zero new or missing rule classes
  among fired rules, zero policy errors/panics, **zero context_overflow**
  (AD-288 holds). The failing-test/build repair-lane and artifact
  freshness families did not fire because no validation failure lane was
  entered — consistent with the cleaner run path, not a rule change.
- **Failure mix:** 2 circle_detected (qa, dogfood — both recovered by
  AD-289 automatic retries with zero operator `run-role` calls), 1
  ticket_gate (repaired by the existing edge), 1 stop artifact. Run 3:
  1 max_turns + 2 circle_detected + 2 ticket_gate + 1 operator retry.
  Strictly cleaner; deltas attributable to run-path variance, no new
  classes.
- **Target commits / tickets / docs produced:** 24 commits pushed to the
  local origin; CEO vision, COO plan + F-001 contract, CTO scenario
  tickets; **T-001, T-002, T-003 closed through full
  engineer → qa → security → dogfood cycles; dogfood finding ticket
  T-004 created, implemented, and closed; T-001 rework loop opened** (the
  in-flight rework engineer job was cancelled by the stop). Final state
  checkpointed as `734991c chore(evidence): checkpoint v0.50.24 AD-287
  final-sequence replay state + scores export`, pushed to the local
  origin.
- **Telemetry highlights (per-role avg wall):** ceo 58s; coo 75s;
  cto-weekly 103.5s ×2; engineer 106.8s avg ×6 (2 failed); qa 40.4s ×5
  (1 failed); security 25s ×3; dogfood 102.8s ×4 (1 failed);
  orchestrator 59s ×2; failure categories: circle_detected 2,
  ticket_gate 1, llm_unreachable 1 (stop artifact), guardrail_block 44,
  **context_overflow 0**
- **Product progress reached:** working Vite/React habit tracker with
  **four tickets closed (T-001..T-004) including a dogfood-finding
  remediation cycle** — beyond Run 3's bar (T-001 + T-002)
- **Target intervention-debt count:** 0 open in target backlog at stop
  (T-004 was created and resolved in-run; the reopened T-001 rework was
  in flight at the operator stop, not debt)
- **Operator interventions: 0 retries** (Run 3: 1) — both
  circle_detected failures auto-recovered via AD-289; the only operator
  action was the graceful stop (`POST /api/stop`, `{"ok":true}`) after
  the success criteria were exceeded
- **Runtime artifacts:** traces for all 24 jobs in the per-repo DB; full
  debug log; scores export committed to the target
- **Stop reason:** operator graceful stop after criteria exceeded; queue
  drained to 0 pending/running within seconds of the stop

### Run 4 pass/fail against AD-284/AD-285

**Pass — no extraction drift.** The rerun exceeds the Run 2 baseline and
the Run 3 checkpoint lifecycle reach (4 tickets closed vs 2); every fired
guardrail is a pre-existing rule class with identical message shape,
including the validation-lane rules moved in this final slice; zero
context_overflow; zero policy errors; no new foundation-owned failure
class; target intervention debt did not increase; the stop is
operator-visible and recorded. Run 4 does **not** replace Run 2 as the
pace baseline (same rationale as Run 3).
