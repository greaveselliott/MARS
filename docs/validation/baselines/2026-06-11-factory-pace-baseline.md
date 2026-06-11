# Factory Pace Baseline — 2026-06-11

**Status:** Recorded
**Date:** 2026-06-11
**Owner:** foundation-maintainer (T-011)
**Binary:** `mars-harness 0.50.1` (source ref `71cb744`, branch `codex/main-lifecycle-stabilization-rebased`)
**Purpose:** Dated pace baseline required by T-011 and the foundation improvement
plan measurement floor. Every Phase 3 (WS-D convergence, WS-E decomposition)
slice records its pace delta against this baseline per AD-138 and AD-284/AD-285.

This baseline is measurement-only evidence. It records what the factory does on
v0.50.1 before any convergence or policy changes land. No source fixes were
made in response to the findings below; they are classified and routed to
Phase 3.

## Archetype: API or service (Inventory/API canary, demo-11)

- Run report: [reports/2026-06-11-demo-11-pace-baseline.md](reports/2026-06-11-demo-11-pace-baseline.md)
- Target: `/path/to/local-redacted` (fresh git repo,
  `spec.md` brief: standard-library Go HTTP JSON inventory API, local bare
  `origin`)
- Database: `~/.mars-harness/db/demo-11/mars.db`
- Lifecycle reach: CEO → COO → CTO (`cto-weekly` ended `max_turns`; run never
  reached Engineer)

### Factory Pace (from `scores export`, 2026-06-11)

| Role | Jobs | Avg Turns | Avg Tool Invocations | Avg LLM Calls | Avg Wall | Limit Stops | Pace Signal |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| ceo | 1 | 26.0 | 12.0 | 12.0 | 30.0s | 0 | trace baseline |
| coo | 1 | 42.0 | 20.0 | 20.0 | 105.9s | 0 | high-turn baseline |
| cto-weekly | 1 | 105.0 | 51.0 | 51.0 | 724.5s | 1 | limit-stop evidence |

### Convergence And Guardrails (from `scores export`, 2026-06-11)

| Role | Traced Jobs | Circle Detected | Max Turns | Other Limit Stops | No-Op Outcomes | Guardrail Blocks | Block Rate | Signal |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| ceo | 1 | 0 | 0 | 0 | 0 | 0 | 0% | clean |
| coo | 1 | 0 | 0 | 0 | 0 | 0 | 0% | clean |
| cto-weekly | 1 | 0 | 1 | 0 | 0 | 0 | 0% | convergence-failure evidence |

Telemetry triage recorded in-run guardrail/tool-policy blocks separately:
`cto-weekly` guardrail_block x16, `ceo` x1, `coo` x1. The outcome-derived
Block Rate column counts terminal guardrail outcomes only, so it reads 0%
while 16 in-run blocks occurred; see the observation log in the run report.

### Token cost (trace summaries)

| Role | Total Tokens | Wall |
| --- | ---: | ---: |
| ceo | 136,429 | 30.0s |
| coo | 405,609 | 105.9s |
| cto-weekly | 2,336,357 | 724.5s |

## Archetype coverage state at baseline date

| Archetype | Most recent baseline | State |
| --- | --- | --- |
| Static browser app | demo-10 Tetris run (2026-06-11, v0.50.x) reached Engineer implementation with playable product output; 2 engineer jobs failed late-lifecycle | informal — predates this baseline contract |
| API or service | demo-11 (this document) | recorded |
| CLI/tooling | `demo-slug-run62` full lifecycle (2026-05-2x, pre-0.50 patched binary) | stale — predates v0.50.x |
| Package-managed frontend | none | gap (owned by T-029) |
| Existing-repo maintenance | none | gap (owned by T-029) |

## Baseline reading

- Planning pace (CEO 26 turns, COO 42 turns) is the healthy reference band for
  bootstrap roles on this archetype.
- The dominant turn sink is `cto-weekly`: 105 turns, 51 tool calls, 12 minutes,
  2.3M tokens, ending in `max_turns` with ~13 consecutive blocked
  `job_disposition_record` retries. Root cause is a deterministic policy wedge
  (false-duplicate `ticket_create` vs the 3-scenario handoff gate) recorded as
  finding F1 in the run report and routed to Phase 3 as ticket `T-030`.
- AD-218 (Engineer post-runtime-validation convergence) remains unvalidated on
  this archetype: the run never reached Engineer. The Phase 3 fix for F1 must
  land before the Inventory/API canary can confirm AD-218.
- Max-turn calibration (T-011 close-out) stays open: per T-011's falsification
  clause, limits are derived from accumulated post-fix turn distributions, and
  this single wedged run shows a policy defect, not a limit mis-calibration.
