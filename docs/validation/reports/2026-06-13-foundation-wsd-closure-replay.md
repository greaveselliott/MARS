# Validation Report: foundation WS-D closure + AD-290/291 batched replay

**Date:** 2026-06-13
**Author:** foundation-maintainer
**Purpose:** AD-284 batched validation for WS-D slices 6–8, CTO ticket-gate loop
fix (AD-290), and foundation operating model (AD-291/AD-293) on **fresh ephemeral
targets** per [foundation-operating-model.md](../../design-docs/foundation-operating-model.md).

**Source:** `v0.55.0` (`5ae8a92` release notes commit on `main`)

## Scope

| Change class | Minimum replays (AD-284) |
| --- | --- |
| Tool policy (WS-D 6–8) | Static browser + API/service |
| Orchestration (AD-290 ticket gate) | Static browser + API/service |

Runs use **new folders** from `validation-target.mjs` (AD-293): `spec.md` brief +
`mars-harness init` + fresh per-repo DB — not checkpoint repos or reused demo
worktrees (see invalid
[demo-14 canary](2026-06-13-demo-14-wsd-slice4-canary-invalid.md)).

## Superseded: demo-11-closure-replay worktree — STOPPED

The first closure attempt reused `demo-11-closure-replay` (worktree at harness-init
commit). That predates AD-293. The run was **stopped** when ephemeral-run doctrine
landed; do not use its partial evidence.

- **Was:** `mars-harness start --repo .../demo-11-closure-replay`
- **Reason stopped:** Superseded by AD-293 ephemeral targets

## Run 1: ephemeral static-browser — PENDING

- **Create:**
  `node scripts/validation-target.mjs create --profile static-browser-todo --label wsd-closure`
- **Start:** `mars-harness start --repo <run-path> --debug --log-file ~/.mars-harness/traces/logs/<run-id>.log`
- **Binary:** installed `mars-harness` from current `main` after `make install`
- **Pass criteria:** Lifecycle reaches Engineer + QA without CTO-only loop; AD-290
  does not wedge; product progress beyond planning-only churn
- **Stop rule (AD-292):** Stop immediately if >3 serial `cto-weekly` with zero
  engineer/qa — record wedge, do not drain-monitor
- **Discard when done:** `node scripts/validation-target.mjs discard <run-id>`

### Run 1 evidence

_(filled when run completes or stops)_

## Run 2: ephemeral API/service — PENDING

- **Create:**
  `node scripts/validation-target.mjs create --profile depot-supplies-api --label wsd-closure`
- **Depends on:** Run 1 pass or documented non-regression of shared tool-policy paths

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 1 | static-browser (`static-browser-todo`) | PENDING |
| 2 | api-service (`depot-supplies-api`) | PENDING |
