# Factory Pace Baseline — 2026-06-12 (balanced model)

**Status:** Active baseline for Phase 3 pace comparisons
**Author:** foundation-maintainer
**Source:** Live demo-11 Inventory/API canary replay on `mars-harness 0.50.2`
**Run report:** [../reports/2026-06-12-demo-11-pace-baseline.md](../reports/2026-06-12-demo-11-pace-baseline.md)
**Ticket:** T-011 (acceptance: dated baseline before optimization work starts)

## Model identity (measurement contract — AD-285)

Pace deltas against this baseline are valid only on this model set.

| Tier | Model | Quant | Context | Server |
| --- | --- | --- | ---: | --- |
| reasoning | Qwen3-Coder-30B-A3B-Instruct | Q4_K_M (18.6 GB) | 131072 | llama-server :18081 |
| coding | Qwen3-Coder-30B-A3B-Instruct | Q4_K_M (18.6 GB) | 32768 | llama-server :18080 |
| fast | google_gemma-4-E4B-it | Q5_K_M (5.8 GB) | registry default | on demand |

Resolution: `performance_profile: balanced` in `~/.mars-harness/config.yaml`.
The prior 2026-06-11 attempt on the quality-profile Q8_0 weights (32.5 GB,
unified-memory saturation) is evidence-only; see
[../reports/2026-06-11-demo-11-pace-baseline.md](../reports/2026-06-11-demo-11-pace-baseline.md).

## Factory Pace (`scores export`, 22 traced jobs)

| Role | Jobs | Avg Turns | Avg Tool Invocations | Avg LLM Calls | Avg Wall | Limit Stops | Pace Signal |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| ceo | 1 | 40.0 | 19.0 | 19.0 | 71.8s | 0 | high-turn baseline |
| coo | 1 | 32.0 | 15.0 | 15.0 | 78.3s | 0 | high-turn baseline |
| cto-weekly | 1 | 42.0 | 20.0 | 20.0 | 204.1s | 0 | high-turn baseline |
| dogfood | 1 | 24.0 | 10.0 | 12.0 | 47.9s | 1 | limit-stop evidence |
| engineer | 6 | 75.2 | 36.3 | 36.7 | 244.7s | 2 | limit-stop evidence |
| orchestrator | 5 | 32.4 | 15.2 | 15.2 | 65.5s | 0 | high-turn baseline |
| qa | 5 | 39.0 | 17.8 | 18.4 | 63.3s | 1 | limit-stop evidence |
| security | 2 | 22.5 | 10.0 | 10.0 | 34.8s | 0 | trace baseline |

End-to-end wall clock: 51.6 minutes (2026-06-11 23:21:44 UTC seed →
2026-06-12 00:13:21 UTC final orchestrator loop-guard stop) for a full
lifecycle: CEO → COO → CTO → Engineer (T-001 plus three QA rework loops) →
QA → Security ×2 → Dogfood attempt → final QA, ending with the queue drained.

## Convergence And Guardrails (AD-283 / T-027)

| Role | Traced Jobs | Circle Detected | Max Turns | No-Op Outcomes | Guardrail Blocks | Signal |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| ceo | 1 | 0 | 0 | 0 | 0 | clean |
| coo | 1 | 0 | 0 | 0 | 0 | clean |
| cto-weekly | 1 | 0 | 0 | 0 | 0 | clean |
| dogfood | 1 | 1 | 0 | 0 | 0 | convergence-failure evidence |
| engineer | 6 | 1 | 1 | 0 | 0 | convergence-failure evidence |
| orchestrator | 5 | 0 | 0 | 0 | 0 | clean |
| qa | 5 | 1 | 0 | 0 | 0 | convergence-failure evidence |
| security | 2 | 0 | 0 | 0 | 0 | clean |

Telemetry triage (guardrail_block events routed to foundation telemetry, not
target backlog): engineer ×51, qa ×7, cto-weekly ×3 (plus ceo/coo/security
singles); 133 guardrail_block log lines total. Terminal outcomes: 18
positive, 4 negative. Convergence failures: 3 circle_detected, 1 max_turns.

## Baseline interpretation

- **Slowest role:** engineer (75.2 avg turns, 244.7s avg wall, 2 limit
  stops) — the primary Phase 3 pace-optimization target.
- **Interventions:** one operator retry (`POST /api/run-role` for qa after a
  circle_detected runtime failure that the orchestrator correctly refused to
  redispatch). All other failures self-recovered through queue retries.
- **Known stop:** dogfood circle_detected was never retried; the lifecycle
  ended at the orchestrator loop guard without reaching the release stage.
  Recorded as a foundation-owned finding in the run report.
- Heavy-model comparison (evidence-only, not a pace delta): cto-weekly went
  from a 12.1-minute max_turns wedge on Q8_0 to a 3.4-minute clean
  convergence on Q4_K_M.
