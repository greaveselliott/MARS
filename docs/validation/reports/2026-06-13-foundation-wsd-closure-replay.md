# Validation Report: foundation WS-D closure + AD-290/291 batched replay

**Date:** 2026-06-13
**Author:** foundation-maintainer
**Purpose:** AD-284 batched validation for WS-D slices 6–8, CTO ticket-gate loop
fix (AD-290), and foundation operating model (AD-291/AD-293) on **fresh ephemeral
targets** per [foundation-operating-model.md](../../design-docs/foundation-operating-model.md).

**Source:** `v0.56.0` (`e07494c` release notes commit on `foundation-restart`)

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

## Run 1: ephemeral static-browser — PASS (partial lifecycle)

- **Run ID:** `run-20260613-015932-static-browser-todo-wsd-closure`
- **Path:** `/path/to/local-redacted`
- **Create:**
  `node scripts/validation-target.mjs create --profile static-browser-todo --label wsd-closure`
- **Start:** `mars-harness start --repo <run-path> --debug --log-file ~/.mars-harness/traces/logs/run-20260613-015932-static-browser-todo-wsd-closure.log`
- **Binary:** `mars-harness` v0.56.0 after `make install`
- **Model profile:** balanced (`Qwen3-Coder-30B-A3B Q4_K_M` per factory-pace baseline)
- **Stopped:** operator stop after pass criteria met (~10m elapsed); queue not fully drained

### Run 1 evidence

| Phase | Result | Duration / notes |
| --- | --- | --- |
| CEO | completed | ~51s |
| COO | completed | ~1m 30s |
| CTO-weekly | completed | ~2m 45s; created T-001..T-003 in backlog (no wedge) |
| Engineer | 1 success | T-001 implementation; no eng failures |
| QA | started | first QA job dispatched at ~9m 50s elapsed |

**AD-290 wedge check:** PASS — exactly one `cto-weekly` job before Engineer; no
CTO-only loop (>3 serial CTO with zero engineer/qa did not occur).

**Product progress:** tickets created and first implementation slice claimed; beyond
planning-only churn.

**Independent observer:** replay-progress snapshots at 03:09:24 confirmed
`Running: qa` with `1 eng ✓ · 0 eng ✗`.

## Run 2: ephemeral API/service — BLOCKED (COO max_turns)

- **Run ID:** `run-20260613-020934-depot-supplies-api-wsd-closure`
- **Path:** `/path/to/local-redacted` (discarded)
- **Create:**
  `node scripts/validation-target.mjs create --profile depot-supplies-api --label wsd-closure`
- **Start:** `mars-harness start --repo <run-path> --debug`
- **Depends on:** Run 1 pass (satisfied)

### Run 2 evidence

| Phase | Result | Notes |
| --- | --- | --- |
| CEO | completed | ~2m 11s |
| COO | **blocked** | 3× `max_turns` (2 automatic + 1 operator retry at 03:40 UTC); paused ~03:31 |
| CTO / Engineer / QA | not reached | stopped per AD-292 wedge rule |

**AD-290 wedge check:** N/A — did not reach CTO (COO planning convergence failure, not
CTO-only loop).

**Supplementary API archetype evidence:** the 2026-06-12 **demo-14** Inventory/API
replay (AD-289) drained T-001–T-010 with zero operator `run-role` calls and is the
prior lifecycle-reach reference for api-service archetype tool-policy changes.

**Retry command recorded:** `POST /api/run-role {"repo_id":"1c0aa858-bcb4-42f0-bdf8-b34879b9f18c","role":"coo"}` or
`mars-harness run coo --repo <run-path>` — tracked as **TD-009** (COO turn budget for
depot-supplies-api profile).

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 1 | static-browser (`static-browser-todo`) | **PASS** (Engineer + QA reached; AD-290 wedge clear) |
| 2 | api-service (`depot-supplies-api`) | **BLOCKED** (COO max_turns; supplementary demo-14 PASS) |

## Closure verdict

**Foundation improvement plan WS-D closure: confirmed for static-browser + prior API
evidence.** Run 2 ephemeral replay blocked on COO planning turns; does not invalidate
WS-D slices 6–8 or AD-290 (CTO gate) — those were validated on Run 1 and on demo-14.
Follow-up: TD-009.
