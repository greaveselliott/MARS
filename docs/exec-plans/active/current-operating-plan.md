# Current Operating Plan

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** T-013, MH-051, TD-009, TD-010
**Goals:** G-001, G-003, G-004
**BDD Feature:** F-001, F-012, F-010
**Hypothesis:** After foundation improvement closure (2026-06-13), the highest leverage work is
remaining factory-pace debt (COO planning turns, dogfood validation waste) before
promoting the dashboard epic after `T-011` closed 2026-06-12.
**Success Evidence:** TD-009 replay reaches CTO+Engineer on `depot-supplies-api`;
T-013 reduces static-app dogfood turns with replay proof; or MH-051 docs
registration lands with F-010 scenario schedule if dashboard is promoted.
**Falsification Evidence:** COO max_turns repeats on fresh ephemeral API targets;
dogfood turn waste unchanged on static-browser replays; dashboard work starts
without AD-279 start condition documentation.
**Scenario Schedule:** F-012-S010, F-010-S010 (if dashboard promoted)
**Current Failing Scenario:** COO `max_turns` on `depot-supplies-api` ephemeral
profile (2026-06-13 closure Run 2, TD-009).
**Walking Skeleton Slice:** Calibrate COO turn budget or tighten planning guidance,
then rerun one ephemeral API validation target and record pass/fail.
**Learning Or MVP Outcome:** Factory pace improvements stay measured on ephemeral
targets per AD-293; dashboard epic can start when factory debt is bounded or
explicitly reprioritized.
**Created:** 2026-06-13
**Owner:** Mars Harness maintainers
**Source:** Completion of
[foundation-improvement-and-operating-model.md](../completed/foundation-improvement-and-operating-model.md)

## Purpose

Post-foundation successor plan. The foundation improvement workstreams
(WS-A–WS-F, WS-D, WS-E) completed 2026-06-13; see the completed plan and
[2026-06-13-foundation-wsd-closure-replay.md](../../validation/reports/2026-06-13-foundation-wsd-closure-replay.md).

## Current Truth

- Source version: `VERSION`
- Completed foundation plan:
  `docs/exec-plans/completed/foundation-improvement-and-operating-model.md`
- `T-011` done (max-turn calibration); dashboard epic start condition met per
  AD-279
- Ephemeral validation: `scripts/validation-target.mjs` + `docs/validation/profiles/`
- Open factory debt: **TD-009** (COO max_turns API profile), **T-013** (dogfood
  turn waste), **TD-010** (main/serve/scanner god-file splits, low priority)

## Next slice

1. [ ] **TD-009:** Investigate COO `max_turns` on `depot-supplies-api` profile;
   smallest evidence-backed fix (turn budget or planning guidance); rerun
   `validation-target.mjs create --profile depot-supplies-api --label td-009`.
2. [ ] **T-013:** Reduce dogfood turn waste for static-app validation (backlog
   ticket); confirmatory static-browser ephemeral replay.
3. [ ] **Optional — dashboard:** Promote `backlog/tanstack-dashboard-control-plane.md`
   beginning with **MH-051** when TD-009/T-013 are complete or operator
   reprioritizes.

## Current Priority Order

1. TD-009 COO planning convergence on API ephemeral targets
2. T-013 static-app dogfood turn waste
3. TD-010 god-file splits (main.go, server.go, scanner init) as capacity allows
4. Dashboard epic MH-051..061 (backlog; AD-279 start condition satisfied 2026-06-13)
