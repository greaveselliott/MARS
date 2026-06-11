# Validation Report: demo-11 Inventory/API Pace Baseline

**Date:** 2026-06-11
**Author:** foundation-maintainer
**Purpose:** T-011 dated factory-pace baseline (measurement floor for the
foundation improvement plan) plus the AD-218 Inventory/API confirmation rerun
requested by T-011 `next_action`. Recorded per the AD-285 evidence contract.

> **Evidence reclassification (2026-06-12):** Run 1 below executed on the
> heavy quality-profile model (`Qwen3-Coder-30B-A3B-Instruct-Q8_0`, reasoning
> ctx 131072) whose weights were maxing unified memory and slowing inference.
> The operator swapped the harness to the balanced model for all future runs,
> so this run is **discarded as a Phase 3 pace baseline** (pace deltas against
> it would be confounded by the model change) and kept as **evidence-only**
> heavy-model performance data supporting the swap decision. The findings
> (F1–F3) remain valid: F1 is a deterministic policy wedge independent of
> model identity. The replacement baseline on the balanced model is
> `docs/validation/baselines/2026-06-12-factory-pace-baseline.md`.

## Heavy-model performance evidence (supports the model swap)

Model identity for Run 1: `Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf` (32.5 GB
weights) for both reasoning (ctx 131072) and coding (ctx 32768) tiers under
`performance_profile: quality` resolution on a 64 GiB unified-memory machine.
With both tier servers resident, weights alone exceeded 50 GB of unified
memory.

- Per-role wall times: CEO 30.0s (26 turns), COO 105.9s (42 turns),
  cto-weekly **724.5s (12.1 minutes, 105 turns)** before `max_turns`.
- The same heavy model also served a demo-12 frontend probe run
  (2026-06-12 00:00–00:10): two Engineer jobs ended `max_turns` at ~6 minutes
  each (51 LLM calls each), with multi-minute model load times whenever the
  coding-tier server started alongside the resident reasoning server.

## Run 1: Inventory/API Canary On v0.50.1 — 2026-06-11

### Setup

- **Exact command:** `mars-harness start --repo /path/to/local-redacted --debug --log-file ~/.mars-harness/traces/logs/demo-11-baseline-start.log`
- **Target:** `/path/to/local-redacted` — fresh git
  repo seeded with `spec.md` (small standard-library Go HTTP JSON API for
  inventory items with quantities and reorder thresholds, matching the
  run-65 Inventory/API archetype brief), local bare remote
  `/path/to/local-redacted`
- **Source ref / binary:** `mars-harness 0.50.1`, built with `make install`
  from `71cb744` on `codex/main-lifecycle-stabilization-rebased` (same tip as
  `origin/main`)
- **Model identity:** `Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf` for reasoning
  (ctx 131072) and coding (ctx 32768) tiers; quality-profile resolution
  (heavy model — see the reclassification banner above)
- **Database:** `~/.mars-harness/db/demo-11/mars.db`
- **Log:** `~/.mars-harness/traces/logs/demo-11-baseline-start.log`
- **Environment note:** before launch, a stale `mars-harness start` process
  from the earlier demo-10 session (all jobs terminal) was holding the
  orchestrator ports and two llama-server instances; it was stopped with
  SIGINT per the bounded environmental-fix exception.

### Job sequence

| # | Role | Outcome | Turns | Tool Invocations | Wall | Tokens |
| --- | --- | --- | ---: | ---: | ---: | ---: |
| 1 | ceo | completed | 26 | 12 | 30.0s | 136,429 |
| 2 | coo | completed | 42 | 20 | 105.9s | 405,609 |
| 3 | cto-weekly | **max_turns** | 105 | 51 | 724.5s | 2,336,357 |

After the `cto-weekly` failure the orchestrator correctly refused to dispatch
the runtime failure ("foundation telemetry or operator retry must resolve it
first") and the survey paused on the dirty target workspace
(`docs/tickets/backlog/T-001-...` untracked) — an operator-visible blocker,
not a silent loop.

### Target commits, tickets, docs produced

Factory commits (in order): harness baseline init; CEO goal/spec commit
(`chore(learnings)` + plan updates); COO `plan: update active scenario
schedule and feature contract for inventory API demo` (F-001 rewritten in
place per canonical bootstrap guidance); CTO `docs: create implementation
ticket T-001 for inventory API health endpoint` (`cfb70d6` — committed only
`.harness/learnings.yaml`; the ticket file itself stayed untracked, see F2).

- Tickets: `T-001-implement-health-endpoint-for-inventory-api.md` (backlog,
  covers F-001-S001)
- Target intervention-debt tickets: **0** (all signals kept foundation-side)
- Operator cleanup commits after the run: ticket file commit + quality-score
  export commit, pushed to the local bare origin; target left clean.

### Telemetry highlights

- `guardrail_block` events: cto-weekly x16, ceo x1, coo x1 — all kept out of
  target backlog as foundation-owned intervention signals.
- `max_turns` event for cto-weekly routed to foundation telemetry.
- `scores export` (v0.50.1) rendered Factory Pace and the new AD-283
  Convergence And Guardrails sections from this live run. Factory Pace rows:
  ceo 26.0 turns / 12.0 tools / 30.0s; coo 42.0 / 20.0 / 105.9s; cto-weekly
  105.0 / 51.0 / 724.5s with 1 limit stop. Convergence rows: cto-weekly
  1 max_turns ("convergence-failure evidence"); telemetry triage rows:
  cto-weekly guardrail_block x16, ceo x1, coo x1.

### Product progress reached

Product-specific planning (Inventory API goals, exec plan, F-001 feature
contract rewritten for inventory endpoints), one ordinary product ticket
covering F-001-S001. No implementation: the run never reached Engineer.

### Stop reason

`cto-weekly` ended `max_turns` inside a deterministic policy wedge (F1
below); survey paused on the dirty workspace; operator stopped the run with
SIGINT after confirming the wedge, committed the dangling ticket file, and
captured the pace export.

## Findings And Ownership Classification

### F1 (foundation-owned, tool policy): `ticket_create` fuzzy title dedupe falsely rejects distinct endpoint tickets and wedges CTO against the scenario-coverage handoff gate

Sequence observed in the trace:

1. CTO created `T-001` "Implement health endpoint for inventory API"
   (F-001-S001).
2. The disposition gate blocked Engineer handoff: "cto cannot hand off
   implementation for F-001 after covering only 1/3 early product
   scenario(s). Create a small product backlog batch with ticket_create".
3. Every attempt to create the second ticket — "Implement list items
   endpoint for inventory API" — returned `DUPLICATE: ticket "Implement
   health endpoint for inventory API" already exists`. The titles are
   distinct, but `isSubsetMatch` in `internal/tools/ticket_create.go`
   matches at >=80% shared words for 5+ word titles, and the two titles share
   4/5 normalized words (implement, endpoint, inventory, api).
4. One grouped retry ("Implement health and list items endpoints...") was
   correctly rejected for re-covering F-001-S001, pushing the role back into
   the false-duplicate path.
5. CTO then looped `job_disposition_record` (~13 blocked retries, turns
   38–50) until `max_turns`.

The wedge is deterministic for archetypes whose natural ticket titles share a
common suffix ("... endpoint for inventory API"), which is exactly the
API/service archetype shape. The static-browser demo-10 run passed this gate
because its ticket titles diverge more. Classification: **foundation-owned**
(tool policy, `internal/tools`). Routed to Phase 3 as backlog ticket
`T-030-unwedge-cto-scenario-batch-handoff-from-ticket-create-false.md`; no
fix in this measurement phase.

### F2 (foundation-owned, evidence integrity): CTO commit claimed ticket creation but did not include the ticket file

`cfb70d6 docs: create implementation ticket T-001 ...` committed only
`.harness/learnings.yaml`; the ticket file remained untracked, which is what
paused the orchestrator survey. The commit-message/content mismatch made a
"create ticket" commit pass while the ticket stayed dangling. Recorded as
foundation evidence under the same Phase 3 ticket as F1 (`T-030`, same wedge
recovery path); no fix in this phase.

### F3 (evidence-only observation): outcome-derived guardrail Block Rate reads 0% while in-run telemetry recorded 16 blocks

The AD-283 Convergence And Guardrails table derives Guardrail Blocks/Block
Rate from terminal outcome counts, so a job with 16 in-run policy blocks that
ends `max_turns` contributes 0 to the block rate; the blocks surface only in
telemetry triage rows. Operators reading only the convergence table may
underestimate guardrail friction. Kept evidence-only for now; revisit when
Phase 3 convergence telemetry work touches this surface.

## AD-218 / T-027 validation status

- **AD-218 (post-runtime-validation convergence): not confirmed.** The run
  never reached Engineer, so the Inventory/API rerun requested by T-011
  `next_action` could not exercise AD-218. The F1 wedge must be fixed (Phase
  3) before that confirmation can run. Replay command for the rerun:
  the exact command in Setup above, against a fresh demo target with the same
  Inventory/API brief.
- **T-027 (convergence/guardrail telemetry): confirmed live.** This run is
  the live validation flagged in T-027's ticket: `scores export` against
  `~/.mars-harness/db/demo-11/mars.db` rendered the Convergence And
  Guardrails section, the per-role max-turns count, the convergence-failure
  signal, and the Evidence Signals roll-up from real run data, with missing
  evidence still rendered as missing elsewhere in the export.

## Pass/fail against AD-285

The prior recorded baseline for this archetype (`demo-inventory-api-run65`,
pre-0.50 patched binary) reached Engineer product rework. This run stops two
stages earlier, so the rerun does **not** meet the "reaches at least the
lifecycle stage of the prior baseline" bar: the lifecycle-reach regression is
owned by F1, and no improvement claim is made for this archetype on v0.50.1.
Per the 2026-06-12 reclassification banner, this run is heavy-model
evidence-only and is not the Phase 3 pace baseline; the balanced-model rerun
in `2026-06-12-demo-11-pace-baseline.md` owns that role.
