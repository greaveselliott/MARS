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

**Historical API archetype reference:** the 2026-06-12 **demo-14** Inventory/API
replay (AD-289) drained T-001–T-010 with zero operator `run-role` calls. It
remains useful background evidence, but it is not a strict AD-284 closure
substitute for this fresh ephemeral Run 2.

**Retry command recorded:** `POST /api/run-role {"repo_id":"1c0aa858-bcb4-42f0-bdf8-b34879b9f18c","role":"coo"}` or
`mars-harness run coo --repo <run-path>` — tracked as **TD-009** (COO turn budget for
depot-supplies-api profile).

### Run 2 forensics — recorded 2026-06-13

- **Trace:** `~/.mars-harness/traces/logs/run-20260613-020934-depot-supplies-api-wsd-closure.log`
  exists and contains 481 lines.
- **Discard manifest:**
  `/path/to/local-redacted`
  records profile `depot-supplies-api`, archetype `api-service`, and status
  `discarded`.
- **Role sequence:** CEO completed once; COO failed three times with
  `max_turns`; CTO, Engineer, and QA were not reached.
- **COO attempts:** `d9f3506a...` (51 LLM calls / 51 tool calls / 45,881
  tokens), `1afa96db...` (51 / 51 / 40,260), and `3416dfd3...` (51 / 51 /
  41,126).
- **Guardrail signal:** COO emitted 47 `guardrail_block` telemetry events
  across the three attempts (12, 19, 16). CEO emitted 3. The trace keeps these
  as foundation-owned signals, not target backlog.
- **Writes and commits:** the trace shows 23 `file_write` executions and 17
  `git_commit` attempts. It does not log file path arguments, so it cannot name
  the exact feature-contract or exec-plan files rewritten during the failed run.
- **Seed state:** current validation tooling writes profile content to `spec.md`
  before `mars-harness init`, and `init` derives the starter product brief only
  from README variants. The preserved trace does not include literal
  `README.md`, `spec.md`, `docs/features/F-001*`, or `docs/exec-plans/*` path
  arguments, so README/spec seed mismatch is a code-path hypothesis rather than
  directly proven by the discarded trace.
- **Stop reason:** after two automatic COO convergence failures and one
  operator/manual COO retry all hit `max_turns`, the server received a dashboard
  stop at `2026-06-13T03:50:09+01:00` and shut down gracefully.

### Seed smoke after source fix — 2026-06-13

Command:

```bash
node scripts/validation-target.mjs create --profile depot-supplies-api --label closure-seed-smoke --root <validation-root> --skip-init
```

Result: created
`<validation-root>`
with both `README.md` and `spec.md` committed. `README.md` begins with
`# Depot Supplies API Demo` and the API brief paragraph, so the existing
`mars-harness init` README brief path can see the validation profile context.
The smoke run was discarded with:

```bash
node scripts/validation-target.mjs discard run-20260613-121728-depot-supplies-api-closure-seed-smoke --root <validation-root>
```

This confirms seed behavior only; strict closure remains unconfirmed until the
fresh factory replays below pass.

### Installed init smoke after source fix — 2026-06-13

Command:

```bash
make install
node scripts/validation-target.mjs create --profile depot-supplies-api --label closure-assurity-init --root <validation-root>
mars-harness doctor --repo <validation-root> --skip-remote
```

Result: installed `mars-harness init` generated target planning docs from the
README seed. The target active plan includes:

```text
Project Brief: Depot Supplies API Demo: Build a small standard-library Go HTTP JSON API for tracking consumable supplies in a maintenance depot.
```

`docs/features/F-001-product-walking-skeleton.md` likewise records the same
Depot Supplies API product brief. `mars-harness doctor --skip-remote` reported
operating-model, role-registry, active-plan hygiene, ticket-drain, and workspace
hygiene `ok`; it warned only that physical RAM was unknown and the per-repo DB
directory was not present before `start`. The init smoke run was discarded:

```bash
node scripts/validation-target.mjs discard run-20260613-122343-depot-supplies-api-closure-assurity-init --root <validation-root>
```

This confirms the bootstrap behavior with the installed candidate binary; strict
closure remains unconfirmed until fresh factory replays pass.

## Run 3: ephemeral static-browser closure-assurity — PASS (threshold)

- **Run ID:** `run-20260613-122453-static-browser-todo-closure-assurity`
- **Path:**
  `<validation-root>`
- **Create:**
  `node scripts/validation-target.mjs create --profile static-browser-todo --label closure-assurity --root <validation-root>`
- **Start:**
  `mars-harness start --repo <validation-root> --debug --log-file <validation-root>`
- **Monitor:**
  `node scripts/replay-progress.mjs --repo run-20260613-122453-static-browser-todo-closure-assurity`
- **Binary:** installed `mars-harness` after `make install`
- **Stopped:** operator stop after QA completed and the closure threshold was
  reached. The downstream security job was in flight and was cancelled by the
  stop, so this run does not claim natural queue drain.

### Run 3 evidence

| Phase | Result | Notes |
| --- | --- | --- |
| CEO | completed | Handed off to COO. |
| COO | completed | Updated active plan and feature contract; recovered from one implementation-write guardrail. |
| CTO-weekly | completed | Created T-001..T-004 and handed off to Engineer after resolving ticket coverage guardrails. |
| Engineer | completed | Claimed T-001, wrote `index.html`, ran `docsync_audit`, served the static page, committed product work, and moved T-001 to done. |
| QA | completed | Reviewed T-001 and approved. |

`git log --oneline -12` in the run repo included:

```text
52cb4db chore(learnings): update runtime learnings for qa
f5eab21 chore(tickets): move T-001 to done
c66ea7b feat: implement basic todo app functionality to add todos from text input (T-001)
bd4e68a chore(tickets): claim T-001
1dfb426 tickets: create implementation tickets for current scenario F-001-S001 and next scenarios
41beb49 plan: update active scenario schedule and feature contract for Simple Todo App
```

`replay-progress` reported Jobs 5/6 done, CEO/COO/CTO phases complete, one
Engineer success, one QA job, and the expected post-threshold security
interruption after the operator stop. `git status --short` was clean. Log grep
for `run-role`, `/api/run-role`, and `mars-harness run` returned no matches.

## Run 4: ephemeral API/service closure-assurity — BLOCKED (CTO handoff coverage)

- **Run ID:** `run-20260613-124152-depot-supplies-api-closure-assurity`
- **Path:**
  `<validation-root>`
- **Create:**
  `node scripts/validation-target.mjs create --profile depot-supplies-api --label closure-assurity --root <validation-root>`
- **Start:**
  `mars-harness start --repo <validation-root> --debug --log-file <validation-root>`
- **Monitor:**
  `node scripts/replay-progress.mjs --repo run-20260613-124152-depot-supplies-api-closure-assurity`
- **Binary:** installed `mars-harness` after `make install`
- **Stopped:** operator stop after a diagnosed CTO handoff wedge. This was a
  diagnostic stop, not a closure pass.

### Run 4 evidence

| Phase | Result | Notes |
| --- | --- | --- |
| CEO | completed | Read `README.md` seeded with `# Depot Supplies API Demo`; recorded decision "Define first visible product slice for Depot Supplies API Demo". |
| COO | completed | Updated API-specific active plan and feature contract after three product-capability guardrail refinements; handed off to CTO. |
| CTO-weekly | completed once, then blocked on second pass | Created T-001..T-003 with `bdd_scenarios` `F-001-S001`..`F-001-S003`, then handed back to COO instead of Engineer after the handoff gate reported `0/1` early product coverage. |
| COO repair | completed | Read T-001..T-003, confirmed scenario metadata, updated active plan/current failing scenario, and returned to CTO. |
| CTO-weekly retry | **blocked** | Read T-002 with `bdd_scenarios: ["F-001-S002"]`, but `job_disposition_record` still reported `0/1` early product coverage. It then created T-004/T-005 and repeatedly hit already-covered scenario guardrails instead of reaching Engineer. |
| Engineer / QA | not reached | Product delivery phase remained at 0 Engineer successes and 0 QA jobs. |

`replay-progress` reported Jobs 4/5 done, CEO/COO/CTO phases complete, and
Product delivery at `0 eng ✓ · 0 eng ✗ · 0 qa · 0 dogfood`. `git log --oneline`
showed API-specific plan and ticket commits, including:

```text
982606e chore(learnings): update runtime learnings for coo
01ee753 plan: update active scenario schedule and feature contract for Depot Supplies API Demo
c32b27e docs: remove incomplete ticket files for F-001-S001 and F-001-S002 before creating proper implementation ticket
d95a537 plan: finalize feature contract with distinct product capabilities for Depot Supplies API Demo
a4f8df2 plan: update active scenario schedule and feature contract for Depot Supplies API Demo
```

The committed T-001/T-002 tickets include machine-readable frontmatter:

```yaml
bdd_scenarios: ["F-001-S001"]
bdd_scenarios: ["F-001-S002"]
```

The latest target worktree had uncommitted T-004/T-005 backlog files at the
diagnostic stop, which are evidence of the wedge and not closure-ready target
work. Log grep for `run-role`, `/api/run-role`, and `mars-harness run` returned
no matches.

**Failure ownership:** foundation-owned planning/ticket handoff convergence.
The README/profile seed fix worked: CEO and COO consumed the Depot Supplies API
brief and the API run advanced past the original COO max-turn blocker. Strict
AD-284 closure remains open because the CTO handoff gate could not reconcile
existing API scenario tickets with the `0/1` early product coverage rule and
never dispatched Engineer.

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 3 (`run-20260613-122453-static-browser-todo-closure-assurity`) | static-browser (`static-browser-todo`) | **PASS** (Engineer + QA completed; stopped after closure threshold, not natural queue drain) |
| 4 (`run-20260613-124152-depot-supplies-api-closure-assurity`) | api-service (`depot-supplies-api`) | **BLOCKED** (README seed fixed and COO completed; CTO handoff coverage gate blocked Engineer dispatch) |

## Closure verdict

**Foundation improvement plan WS-D closure: UNCONFIRMED.** The original Run 2
ephemeral replay blocked on COO planning turns before CTO, Engineer, or QA. The
post-fix Run 4 API replay confirms the README seed repair and advances through
COO and CTO ticket creation, but it still blocks before Engineer because the CTO
handoff gate reports `0/1` early product scenario coverage despite committed API
scenario tickets. Demo-14 remains historical API evidence only; it does not
close the strict fresh-ephemeral AD-284 api-service row. Follow-up: TD-009 /
T-044.

### Closure pass gate

Closure can move back to confirmed only when all of the following are true:

| Gate | Mechanism |
| --- | --- |
| Static-browser replay PASS | Fresh ephemeral report row with monitor/log evidence |
| API/service replay PASS | Fresh ephemeral report row with monitor/log evidence |
| Mechanical closure gate PASS | `mars-harness validation check-closure --report docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md` |
| CI/local gate PASS | `make check` |
