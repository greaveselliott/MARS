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
