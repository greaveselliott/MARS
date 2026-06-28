# Confidence-Gated Live Validation Report

**Date:** 2026-06-16  
**Source build:** local `build/mars` from the working tree  
**Scope:** forward-progress guard follow-up, parallel scoped `start`, local inference port isolation, and planner handoff observations

## Primary Outcome Contract

**Primary Outcome:** Greenfield `mars start` reaches Engineer and records
first successful build/smoke evidence for static web, Phaser/browser-game, and
Go API targets.

**Primary Pass Gate:** All selected targets reach Engineer, produce product
mutation, and record project-appropriate build/smoke evidence without planner or
CTO handoff blockage.

**Primary Status:** `primary_failed`

**Current Primary Blocker:** CTO scenario-ticket coverage and Engineer handoff
remain unproven for the static web and Phaser/browser-game paths.

**Next Primary Action:** Fix the CTO ticket-shaping/handoff blocker, then rerun
the clean greenfield lifecycle until first Engineer build/smoke evidence exists
for static web, Phaser/browser-game, and Go API.

**Supporting Evidence:** Parallel scoped startup, safe scoped cleanup, and
distinct local inference port allocation were proven as setup-layer support, not
as completion of the primary lifecycle outcome.

## Summary

Primary validation failed: this run did not prove greenfield lifecycle progress
through Engineer build/smoke for the selected static web, Phaser/browser-game,
and Go API targets. It records supporting live validation performed while
implementing the confidence-gated follow-up fixes. The run used real
`mars start` processes against ephemeral static-web, Phaser/browser-game,
and Go API targets under `/tmp`. No fake model endpoint or scripted
endpoint was used.

The validation did not prove end-to-end product build/smoke completion. It did prove the setup layer now supports same-machine parallel scoped starts without webhook/dashboard bind failure, and it proved the corrected inference lock path allocates distinct local llama-server ports under real multi-process contention. Full lifecycle build/smoke confidence remains below completion threshold until a long-running sweep reaches Engineer build/smoke evidence.

## Evidence And Classification

| Observation | Evidence | Classification | Result |
| --- | --- | --- | --- |
| Parallel `start` originally failed before agent work because multiple scoped runs bound `:9091`. | `<validation-root>`, `<validation-root>`, terminal output: `failed to bind :9091`. | foundation-owned | Fixed by scoped control-plane fallback. |
| Scoped `start` cleanup could kill shared control-plane or llama-server processes, which is incompatible with same-machine parallel runs. | `internal/serve/cleanup.go` before fix called port kill and global llama-server kill from `start`. | foundation-owned | Fixed by `CleanupScopedLifecycle` for `start`. |
| Rebuilt parallel `start` kept all three runs alive and assigned ephemeral control/dashboard ports to later runs. | `<validation-root>`, `<validation-root>`. | foundation-owned | Passed bounded live setup validation. |
| First rebuilt rerun still allowed duplicate llama-server starts on `18081` for Phaser and Go API. | `<validation-root>`, `<validation-root>`, restart limit at `go-api.start.log:170`. | foundation-owned | Fixed by treating fresh invalid lock metadata as active startup lock. |
| Second bounded rerun allocated distinct inference ports for the three concurrent starts. | `<validation-root>` (`18081`), `go-api.start.log:133` (`18091`), `static-web.start.log:133` (`18101`). | foundation-owned | Passed bounded live port validation. |
| Static web reached healthy inference on its distinct port. | `<validation-root>`. | evidence-only | Supports port allocation, not lifecycle completion. |
| Manual interruption during health checks produced `context canceled`/`inference_crash` rows for Phaser and Go API. | `<validation-root>`, `go-api.start.log:189`. | evidence-only | Probe stop noise; not a product or setup failure claim. |
| In the earlier observed Phaser run, CEO and COO completed and CTO created tickets, but CTO still failed a scenario-coverage handoff policy. | `<validation-root>`, terminal output recorded CTO policy block at handoff. | mixed/unclear | Needs a separate planner-ticket coverage follow-up before claiming full lifecycle pace. |

## Assumption Confidence Matrix

| Assumption | Evidence | Confidence | Validation Required |
| --- | --- | ---: | --- |
| Scoped `start` no longer fails when default webhook/dashboard ports are already occupied. | Unit test `TestServer_startUsesEphemeralHTTPFallbackWhenDefaultPortsBusy`; bounded live logs show static and Go API fallback to ephemeral control/dashboard ports. | 0.95 | Keep in matrix/live smoke for same-machine parallel starts. |
| Scoped `start` no longer kills peer lifecycle or llama-server processes during cleanup. | `start` now calls `CleanupScopedLifecycle`; docs and tests cover behavior indirectly. | 0.9 | Add a future process-safety integration test if cleanup behavior changes again. |
| Inference port reservation prevents duplicate llama-server launches under concurrent process startup. | Regression test `TestRouterResolveServerPortTreatsFreshInvalidLockAsActive`; bounded live rerun allocated `18081`, `18091`, and `18101`. | 0.92 | Run a longer live sweep to ensure no late supervisor churn under sustained generation. |
| `start --model-endpoint` is available for real endpoint validation without local model preflight. | CLI/unit tests and `serve.New` test. No live endpoint was used. | 0.85 | Validate with a real OpenAI-compatible local endpoint; fake endpoints remain excluded. |
| Guardrail loop remediation and blocked/operator_retry routing behave correctly. | Unit tests for telemetry remediation and unremediated loop escalation. No live loop replay in this report. | 0.85 | Add/live-run a targeted guardrail loop fixture. |
| The factory reaches Engineer/build/smoke faster on static, Phaser, and Go API greenfield targets. | Earlier Phaser observation reached CTO and ticket creation, but not Engineer; bounded reruns intentionally stopped during CEO inference setup. | 0.55 | Required: full lifecycle live sweep until first Engineer job and first build/smoke evidence on all three target types. |
| CTO ticket shaping is sufficient for scenario-covered Engineer handoff. | Observed CTO created tickets after feedback but failed handoff scenario coverage. | 0.45 | Required: source fix or prompt/policy refinement, then rerun through CTO handoff. |

## Validation Commands

Unit and docs checks:

```bash
go test ./internal/inference ./internal/serve ./internal/telemetry ./internal/tools ./cmd/mars
go test ./internal/docsconsistency/...
build/mars tools run docsync_audit --repo . --args-json '{}'
```

Live setup probes:

```bash
build/mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
build/mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
build/mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

## Supporting Claims

Primary outcome not claimed: `Primary Status` is `primary_failed`.

Supporting setup-layer claims:

- Same-machine scoped `start` setup no longer fails on default control/dashboard port conflicts.
- Scoped `start` avoids destructive peer-process cleanup.
- Local inference port startup now avoids duplicate same-port launches during fresh lock metadata races.
- CLI/tool/docs surfaces expose the real `--model-endpoint`, `--addr`, and `--dashboard-addr` behavior.

Not claimed:

- Full greenfield lifecycle completion.
- First successful build/smoke on static web, Phaser game, or Go API.
- CTO scenario-ticket coverage is fully solved.
- A real endpoint override live run has not yet been completed.
