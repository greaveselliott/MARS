# Current Operating Plan

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** T-044, T-013, MH-051, TD-009, TD-010
**Goals:** G-001, G-003, G-004
**BDD Feature:** F-001, F-012, F-010
**Hypothesis:** Foundation improvement closure is not earned until both strict
AD-284 closure archetypes pass on fresh ephemeral targets with mechanical
closure-report gating.
**Success Evidence:** Closure report verdict is confirmed only after fresh
static-browser and `depot-supplies-api` rows pass, `mars-harness validation
check-closure` passes, and `make check` passes.
**Falsification Evidence:** COO max_turns repeats on fresh ephemeral API targets;
dogfood turn waste unchanged on static-browser replays; dashboard work starts
without AD-279 start condition documentation.
**Scenario Schedule:** F-012-S010, F-010-S010 (if dashboard promoted)
**Current Failing Scenario:** COO `max_turns` on `depot-supplies-api` ephemeral
profile (2026-06-13 closure Run 2, TD-009).
**Walking Skeleton Slice:** Earn closure assurity: correct overclaimed status,
add a mechanical closure gate, investigate TD-009, apply the smallest
evidence-backed fix, then rerun both closure archetypes.
**Learning Or MVP Outcome:** Foundation closure claims become mechanically honest;
dashboard epic can start only after strict closure passes or an operator
explicitly reprioritizes around the open validation blocker.
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
  AD-279 but promotion is blocked by closure assurity until strict AD-284
  closure passes or the operator explicitly reprioritizes
- Ephemeral validation: `scripts/validation-target.mjs` + `docs/validation/profiles/`
- Open factory debt: **T-044 / TD-009** (COO max_turns API profile), **T-013** (dogfood
  turn waste), **TD-010** (main/serve/scanner god-file splits, low priority)

## Next slice

1. [ ] **Closure assurity / TD-009:** Correct the closure report verdict, add
   the Go-native closure gate, investigate COO `max_turns` on
   `depot-supplies-api`, implement the smallest evidence-backed fix, then rerun
   both fresh closure archetypes.
2. [ ] **T-013:** Reduce dogfood turn waste for static-app validation (backlog
   ticket); confirmatory static-browser ephemeral replay.
3. [ ] **Optional — dashboard:** Promote `backlog/tanstack-dashboard-control-plane.md`
   beginning with **MH-051** when TD-009/T-013 are complete or operator
   reprioritizes.

## Current Priority Order

1. T-044 / TD-009 closure assurity and COO planning convergence on API ephemeral targets
2. T-013 static-app dogfood turn waste
3. TD-010 god-file splits (main.go, server.go, scanner init) as capacity allows
4. Dashboard epic MH-051..061 (backlog; AD-279 start condition satisfied 2026-06-13)
