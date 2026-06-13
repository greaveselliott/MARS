# Validation Report: demo-11 T-030 ticket_create dedupe canary

**Date:** 2026-06-12
**Author:** foundation-maintainer
**Purpose:** Confirm T-030 fix (true keyword-subset title dedupe) removes the
CTO scenario-batch false-DUPLICATE wedge on the Inventory/API archetype per
AD-284 tool-policy row.

## Verdict

**PASS.** cto-weekly completed in 2m 1s with three successful `ticket_create`
calls, zero DUPLICATE rejections, and Engineer handoff. Lifecycle reached
dogfood and release-manager; queue drained with convergence-only engineer
max_turns failures (pre-existing T-011 class, not ticket_create wedge).

## Run: Inventory/API canary on patched binary — 2026-06-12

- **Exact command:** `mars-harness start --repo
  /path/to/local-redacted --debug --log-file
  ~/.mars-harness/traces/logs/demo-11-t030-ticket-dedupe-canary-v3.log`
- **Target:** `/path/to/local-redacted` (repo at harness
  init `143c6b4` atop seed `f704dab`; fresh per-repo DB)
- **Source ref / binary:** patched `mars-harness` from `foundation-restart`
  branch (`make install` after T-030 `isSubsetMatch` fix)
- **Model identity (AD-285):** balanced profile — Qwen3-Coder Q4_K_M reasoning
  + coding; Gemma-4-E4B Q5_K_M fast
- **Database / logs:** `~/.mars-harness/db/demo-11/mars.db`;
  `~/.mars-harness/traces/logs/demo-11-t030-ticket-dedupe-canary-v3.log`
- **Wall clock:** 2026-06-12 21:32:09 UTC → 22:24:40 UTC (~52.5 min)
- **Job totals:** 20 jobs — 17 completed, 2 failed, 1 cancelled; 0 pending at
  drain
- **Planning roles:** ceo 1m · coo 1m 12s · **cto-weekly 2m 1s** (baseline
  204.1s on v0.50.2 — **improved**, no max_turns)
- **ticket_create during CTO:** 3 successful calls, **0 DUPLICATE** lines in trace
- **Handoff gate:** CTO completed and dispatched Engineer (no blocked-disposition
  loop)
- **Delivery:** 4 engineer completions, 2 engineer max_turns (convergence), 4 qa,
  1 security, 1 dogfood, 1 release-manager
- **Failure mix:** engineer max_turns only — not ticket_create wedge
- **Pace delta vs 2026-06-12 baseline:** cto-weekly wall −83s (−41%); end-to-end
  wall ~52.5 min vs 51.6 min baseline (comparable); engineer limit stops 2 vs
  baseline 2 (unchanged convergence class)

## T-030 acceptance mapping

| Criterion | Result |
| --- | --- |
| Distinct-scenario tickets creatable with shared suffix words | PASS (3 creates, 0 false DUPLICATE) |
| CTO satisfies handoff gate without max_turns | PASS |
| True duplicates still rejected | PASS (unit tests) |
| Canary replay per AD-284 | PASS (this report) |
