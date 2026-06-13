# Validation Report: foundation WS-D closure + AD-290/291 batched replay

**Date:** 2026-06-13
**Author:** foundation-maintainer
**Purpose:** AD-284 batched validation for WS-D slices 6–8, CTO ticket-gate loop
fix (AD-290), and foundation operating model (AD-291) on **clean seeds** per
[foundation-operating-model.md](../../design-docs/foundation-operating-model.md).

**Source:** `v0.55.0` (`5ae8a92` release notes commit on `main`)

## Scope

| Change class | Minimum replays (AD-284) |
| --- | --- |
| Tool policy (WS-D 6–8) | Static browser + API/service |
| Orchestration (AD-290 ticket gate) | Static browser + API/service |

Runs use clean git at harness-init commit + fresh per-repo DB — not checkpoint
repos with all tickets in `done/` (see invalid
[demo-14 canary](2026-06-13-demo-14-wsd-slice4-canary-invalid.md)).

## Run 1: demo-11-closure-replay (static browser / Inventory API) — IN PROGRESS

- **Exact command:** `mars-harness start --repo /path/to/local-redacted --debug --log-file ~/.mars-harness/traces/logs/demo-11-wsd-closure-20260613.log`
- **Target git:** worktree at `143c6b4` (`chore(harness): initialize mars harness`) — empty product ticket tree, matches fresh bootstrap
- **Binary:** `mars-harness 0.55.0` (`make install` from `main` @ `5ae8a92`)
- **Database:** `~/.mars-harness/db/demo-11-closure-replay/mars.db` (fresh)
- **Model identity:** TBD from trace (balanced profile expected)
- **Pass criteria:** Lifecycle reaches Engineer + QA without CTO-only loop; AD-290
  does not wedge; product progress beyond planning-only churn
- **Stop rule (AD-292):** Stop immediately if >3 serial `cto-weekly` with zero
  engineer/qa — record wedge, do not drain-monitor

### Run 1 evidence

_(filled when run completes or stops)_

## Run 2: demo-15-closure-replay (API/service) — PENDING

- **Target:** TBD — worktree at harness-init commit + fresh DB (same seed discipline)
- **Depends on:** Run 1 pass or documented non-regression of shared tool-policy paths

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 1 | Static browser / Inventory API | IN PROGRESS |
| 2 | API/service | PENDING |
