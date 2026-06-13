# T-011 close-out — max-turn calibration + WS-D slice 3 + T-035 resume

**Date:** 2026-06-12  
**Harness version:** 0.51.1+ (foundation-restart branch)  
**Baseline:** [../baselines/2026-06-12-factory-pace-baseline.md](../baselines/2026-06-12-factory-pace-baseline.md)

## Changes

| Slice | Change | Evidence |
| --- | --- | --- |
| T-011 | Role `max_turns` calibrated from 2026-06-12 balanced-model baseline | engineer 50→100 (avg 75.2, 2 limit stops); qa 20→40 (avg 39, circle at 18/20); security 20→30 |
| WS-D slice 3 | Validation-failed rework guard clusters consult `EngineerDeliveryState()` | `go test ./internal/tools` PASS |
| T-035 resume | Cancelled idempotency-key jobs reactivate on re-enqueue | `go test ./internal/queue -run EnqueueReactivatesCancelledSeed` PASS; `TestStartCommandInitializesRegistersSeedsAndStops` PASS |
| Coverage | Restore `docsconsistency nostmt` floor; MarsDocSync on new files | `make check` PASS |

## T-011 acceptance

- Evidence-backed pace improvement stack: T-030 dedupe (demo-11 canary), T-031 convergence retry (demo-14), AD-288 overflow fix, plus this max-turn calibration slice.
- Max-turn limits now role-specific in generated manifest defaults instead of silent 50-turn agent default for Engineer.

## Residual bottlenecks

- Engineer guardrail-fighting turn burn (demo-12/13 post-T-032 shape) — WS-D further clusters + block-message contract (T-029 class).
- Dashboard epic MH-051..061 remains deferred per AD-279 until operator reprioritizes.
