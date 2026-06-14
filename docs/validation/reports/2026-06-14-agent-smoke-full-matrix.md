# Validation Report: Agent Smoke Full Matrix

## Summary

**Status:** blocked, not validated.

The full all-agent/all-project agent-smoke matrix has not yet completed with a
real model endpoint. A fake OpenAI-compatible endpoint was used during one
plumbing attempt; that run is explicitly excluded from validation claims under
AD-296 because it creates false positives for role behavior.

One real local-model probe passed:

- `ceo/static-web-empty`
- suite: `fast`
- evidence source: local Mars Harness inference router
- model profile: `quality`
- terminal disposition: `completed`
- LLM calls: 3
- tool invocations: 3
- tokens: 6971
- wall time: 20761 ms
- trace id: `tr-1781441849460392000`

This proves the command can generate an ephemeral target, execute one selected
role through the server job path with a real local model, write results, and
clean up the successful run. It does not prove the full 74-case matrix.

## Selected Matrix

- Command family: `mars-harness validation agent-smoke`
- Intended suite: `full`
- Selected cases: 74
- Role coverage: `ceo`, `head-of-strategy`, `coo`, `cto-weekly`, `engineer`,
  `qa`, `security`, `dependency-manager`, `release-manager`, `dogfood`,
  `pipeline-fixer`, `orchestrator`, `janitor`, `foundation-maintainer`
- Project shapes: static web, React web, Phaser browser game, vanilla canvas
  game, Go API, Go CLI, Go library, docs site, existing maintenance

## Attempts

| Attempt | Command | Result | Failure class | Evidence status |
| --- | --- | --- | --- | --- |
| Default-root full matrix | `/path/to/local-redacted validation agent-smoke --suite full --parallel 2 --max-turns 2 --timeout 5m --report docs/validation/reports/2026-06-14-agent-smoke-full-matrix.md --json` | All selected cases failed while creating run directories under `../demo/validation-runs/agent-smoke/` | `foundation-tool-generation` via workspace sandbox write restriction | Blocked setup attempt; no role validation |
| Writable-root full matrix with real local model | `/path/to/local-redacted validation agent-smoke --suite full --parallel 2 --max-turns 2 --timeout 5m --root <validation-root> --report docs/validation/reports/2026-06-14-agent-smoke-full-matrix.md --json` | Stopped after repeated local model health-check timeouts for `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` on port `18081` | `environment/model` | Blocked real-model attempt; no matrix pass |
| Full matrix with fake endpoint | `/path/to/local-redacted validation agent-smoke --suite full --parallel 4 --max-turns 2 --timeout 5m --root <validation-root> --model-endpoint <scripted OpenAI-compatible endpoint> --report docs/validation/reports/2026-06-14-agent-smoke-full-matrix.md --json` | Reported `60 passed, 14 failed, 74 selected` | Mixed fixture failures plus fake-backed terminal dispositions | Invalid for validation; evidence-only plumbing and excluded by AD-296 |
| Real local-model single-case probe | `MARS_HARNESS_PERFORMANCE_PROFILE=quality /path/to/local-redacted validation agent-smoke --suite fast --role ceo --case static-web-empty --parallel 1 --max-turns 2 --timeout 5m --root <validation-root> --report <validation-root> --json` | `1 passed, 0 failed, 1 selected`; successful run cleaned | none | Valid evidence for this one role/case only |
| Speed-profile single-case probe | `MARS_HARNESS_PERFORMANCE_PROFILE=speed /path/to/local-redacted validation agent-smoke --suite fast --role ceo --case static-web-empty --parallel 1 --max-turns 2 --timeout 5m --root <validation-root> --json` | Failed before role execution because the configured speed-profile model artifact was missing | `environment/model` | Blocked setup attempt; no role validation |

## Cleanup

Retained run directories were cleaned through the harness cleanup path after
the invalid and blocked attempts:

| Root | Cleanup result |
| --- | --- |
| `<validation-root>` | removed 14 retained runs |
| `<validation-root>` | removed 8 retained runs |
| `<validation-root>` | removed 4 retained runs |
| `<validation-root>` | removed 1 retained run |
| `<validation-root>` | removed 0 retained runs; successful case had already been discarded |

## Pass Criteria

The full matrix remains unconfirmed until every selected case runs against a
real model endpoint and the report records:

- terminal disposition matching each case contract
- required artifacts present
- forbidden mutations absent
- generation provenance through foundation tools
- isolated target repos, DBs, logs, and traces
- cleanup status for successful and failed runs
- failure classes for any non-passing case
- model identity or endpoint provenance that is not fake/stub/scripted

Exit code alone does not count as a pass.

## Blocker And Rerun

The next valid full-matrix attempt should use a writable run root and a real
local model profile that can pass health checks:

```bash
MARS_HARNESS_PERFORMANCE_PROFILE=quality /path/to/local-redacted validation agent-smoke --suite full --parallel 1 --max-turns 2 --timeout 10m --root <validation-root> --report docs/validation/reports/2026-06-14-agent-smoke-full-matrix.md --json
```

Use `--parallel 2` only after the local inference server is stable for this
matrix. Any endpoint override must be a real model endpoint; fake, stub, mock,
canned, or scripted endpoints are not validation evidence.
