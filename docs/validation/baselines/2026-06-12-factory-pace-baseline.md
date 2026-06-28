# Factory Pace Baseline — 2026-06-12 (balanced model)

**Status:** Active baseline for Phase 3 pace comparisons
**Archetype update (2026-06-12, v0.50.14):** for the Inventory/API
archetype, the demo-14 convergence-routing replay
([../reports/2026-06-12-demo-14-convergence-routing-replay.md](../reports/2026-06-12-demo-14-convergence-routing-replay.md))
supersedes the demo-11 run as the lifecycle-reach reference: 48 jobs /
126.5 min, all 10 product tickets closed, 4 dogfood passes, 0 operator
interventions (AD-289 automatic convergence routing). The demo-11 per-role
pace rows below remain the v0.50.2 measurement record; engineer/qa pace on
v0.50.14 was same-or-better (engineer 235.9s vs 244.7s avg wall, qa 49.0s
vs 63.3s).
**Author:** foundation-maintainer
**Source:** Live demo-11 Inventory/API canary replay on `mars 0.50.2`
**Run report:** [../reports/2026-06-12-demo-11-pace-baseline.md](../reports/2026-06-12-demo-11-pace-baseline.md)
**Ticket:** T-011 (acceptance: dated baseline before optimization work starts)

## Model identity (measurement contract — AD-285)

Pace deltas against this baseline are valid only on this model set.

| Tier | Model | Quant | Context | Server |
| --- | --- | --- | ---: | --- |
| reasoning | Qwen3-Coder-30B-A3B-Instruct | Q4_K_M (18.6 GB) | 131072 | llama-server :18081 |
| coding | Qwen3-Coder-30B-A3B-Instruct | Q4_K_M (18.6 GB) | 32768 | llama-server :18080 |
| fast | google_gemma-4-E4B-it | Q5_K_M (5.8 GB) | registry default | on demand |

Resolution: `performance_profile: balanced` in `~/.mars/config.yaml`.
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
QA → Security ×2 → Dogfood attempt → final QA.

> **Correction (2026-06-12, second monitor shift):** the run did **not** end
> with a drained queue. The orchestrator was stopped at 01:21:45 BST
> (preempted to start demo-12), orphaning engineer job `4b659db8`
> (dogfood-failure rework) in `pending` (T-035). The final state is also
> internally inconsistent: T-001 sits in `done/` against a final QA
> `changes_requested` (`9c049078`) and no dogfood pass (`ff7b701e`
> circle_detected). The per-role pace rows above remain valid measurement;
> do **not** cite this run as a converged-lifecycle exemplar. See the run
> report's Independent observer section.

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
  never reached the release stage, and the final QA job returned
  `changes_requested` while T-001 stayed in `done/`. The run ended by
  operator preemption with one undrained rework job (T-035), not by a
  drained queue. Recorded as foundation-owned findings in the run report.
  Lifecycle-health claims from this baseline are caveated accordingly; the
  pace rows are unaffected.
- Heavy-model comparison (evidence-only, not a pace delta): cto-weekly went
  from a 12.1-minute max_turns wedge on Q8_0 to a 3.4-minute clean
  convergence on Q4_K_M.

## Archetype coverage (T-029)

All rows captured on this model identity, binary 0.50.2.

| Archetype | Target | Lifecycle reach bar | Status | Report |
| --- | --- | --- | --- | --- |
| Inventory/API (Go service) | demo-11 | Full lifecycle: T-001 done, 3 QA rework loops, 2 security audits, dogfood attempt; stopped by operator preemption with one undrained rework job and a final QA `changes_requested` (see correction above) | Recorded | [2026-06-12-demo-11-pace-baseline.md](../reports/2026-06-12-demo-11-pace-baseline.md) |
| Package-managed frontend (Vite/React) | demo-12 | **v0.50.11 (T-032 fix):** T-001 closed + T-002 feature commit; zero context_overflow (2 pruner saves); residual max_turns ×4 (T-031 scope). Supersedes the v0.50.2 wedge row (engineer scaffold, T-001 in progress, context overflow). | Recorded | [2026-06-12-demo-12-frontend-baseline.md Run 2](../reports/2026-06-12-demo-12-frontend-baseline.md#run-2-v05011-context-fix) |
| Existing-repo maintenance (Phaser/Tetris) | demo-13 | **v0.50.11 (T-032 fix):** multiple product commits + T-001 evidence cycles toward done; zero context_overflow (12 pruner saves); residual max_turns ×5 (T-031 scope). Supersedes the v0.50.2 wedge row (one product commit, both engineer jobs overflowed). | Recorded | [2026-06-12-demo-13-maintenance-baseline.md Run 2](../reports/2026-06-12-demo-13-maintenance-baseline.md#run-2-v05011-context-fix) |

Pace snapshots (slowest role = engineer on all archetypes):

- **demo-12 v0.50.11 archetype baseline (current):** engineer 47.3 avg
  turns / 270.2s avg wall over 4 jobs, limit stops 3 (all max_turns);
  zero context overflow. The superseded v0.50.2 wedge snapshot (80.6 avg
  turns / 267.2s / 3 limit stops over 5 jobs incl. 2 context_overflow) is
  evidence-only.
- **demo-13 v0.50.11 archetype baseline (current):** engineer 51.0 avg
  turns / 449.8s avg wall over 5 jobs, limit stops 5 (all max_turns);
  zero context overflow. The superseded v0.50.2 wedge snapshot (74.0 avg
  turns / 378.5s / 2 context_overflow stops over 2 jobs) is evidence-only.

Note: demo-12/demo-13 archetype pace rows were re-captured on binary
0.50.11 (AD-288 context fix). The model identity is unchanged, so the
demo-11 rows above (binary 0.50.2) remain valid; engineer rows for the
two package-managed archetypes must be compared against the v0.50.11
numbers. Full role tables live in each target's committed
`docs/QUALITY_SCORE.md` and the run reports.
