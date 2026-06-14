# Validation Report: Agent Smoke Full Matrix

## Status

**Status:** validated setup and live execution path; matrix failing.

The full `agent-smoke` matrix was run against real local Mars Harness model
routing, with no `--model-endpoint` override and no fake/stub endpoint. The
lane generated ephemeral targets, executed selected roles through the server
job path, wrote the report, discarded successful runs, and retained failed run
directories for diagnosis.

The matrix did **not** pass: `21 passed`, `53 failed`, `74 selected`.

## Command

```bash
MARS_HARNESS_PERFORMANCE_PROFILE=quality /path/to/local-redacted validation agent-smoke --suite full --parallel 1 --max-turns 2 --timeout 10m --root <validation-root> --report docs/validation/reports/2026-06-14-agent-smoke-full-matrix.md --json
```

- Date: 2026-06-14
- Started: 2026-06-14T13:20:33Z
- Finished: 2026-06-14T13:55:21Z
- Installed binary: `/path/to/local-redacted`
- Binary version: `0.60.1`
- Source ref after release: `558b9ed`
- Evidence source: `local-model`
- Model source: local Mars Harness inference router
- Hardware profile outside sandbox: `high`
- Cleanup: 21 successful runs discarded; 53 failed runs retained under
  `<validation-root>`

## Model Identity

| Tier | Model | Quantization | Context | Port |
| --- | --- | --- | --- | --- |
| reasoning | `Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf` | Q8_0 | 131072 | 18081 |
| coding | `Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf` | Q8_0 | 32768 | 18080 |
| fast | `google_gemma-4-E4B-it-Q8_0.gguf` | Q8_0 | 16384 | 18082 |

## Result Summary

| Role | Passed | Failed |
| --- | ---: | ---: |
| `ceo` | 4 | 1 |
| `head-of-strategy` | 2 | 2 |
| `coo` | 0 | 6 |
| `cto-weekly` | 0 | 6 |
| `engineer` | 0 | 7 |
| `qa` | 2 | 4 |
| `security` | 2 | 3 |
| `dependency-manager` | 0 | 5 |
| `release-manager` | 2 | 3 |
| `dogfood` | 0 | 6 |
| `pipeline-fixer` | 0 | 5 |
| `orchestrator` | 4 | 1 |
| `janitor` | 1 | 4 |
| `foundation-maintainer` | 4 | 0 |

| Project Type | Passed | Failed |
| --- | ---: | ---: |
| `static-web` | 1 | 5 |
| `react-web` | 2 | 12 |
| `browser-game-phaser` | 1 | 12 |
| `canvas-game-vanilla` | 1 | 2 |
| `go-api` | 3 | 10 |
| `go-cli` | 4 | 8 |
| `go-library` | 0 | 2 |
| `docs-site` | 6 | 1 |
| `existing-maintenance` | 3 | 1 |

## Failure Summary

The runner emitted `foundation-tool-generation` for many live execution stops.
That was an evidence-classification bug discovered by this validation run:
`max_turns`, `ticket gate`, and `empty_response` happened after target
generation during live role execution. The source classifier has been corrected
after this run so future reports classify those signals as `role-behavior`.

Normalized failure signatures from the retained `result.json` files:

| Signature | Count |
| --- | ---: |
| wrong terminal disposition | 25 |
| max turns before terminal contract | 21 |
| Engineer ticket gate not satisfied | 5 |
| missing required disposition before completion | 1 |
| empty model response | 1 |

Raw failure classes emitted by this run:

| Emitted Failure Class | Count |
| --- | ---: |
| `foundation-tool-generation` | 27 |
| `role-behavior` | 25 |
| `dispatch-context` | 1 |

## Passing Cases

- `ceo/docs-site-ambiguous-heldout`
- `ceo/go-api-empty`
- `ceo/react-web-conflicting-goals-heldout`
- `ceo/static-web-empty`
- `foundation-maintainer/generated-doctrine-drift-heldout`
- `foundation-maintainer/release-blocker-heldout`
- `foundation-maintainer/source-dry-run`
- `foundation-maintainer/validation-classification`
- `head-of-strategy/docs-site-executive-narrative-heldout`
- `head-of-strategy/go-api-conflicting-bets`
- `janitor/clean-tree-heldout`
- `orchestrator/cto-to-engineer-api`
- `orchestrator/engineer-to-qa-web`
- `orchestrator/qa-to-security-game`
- `orchestrator/runtime-failure-stop-heldout`
- `qa/canvas-game-bad-smoke-heldout`
- `qa/go-cli-missing-evidence-heldout`
- `release-manager/docs-site-notes-only-heldout`
- `release-manager/go-cli-blocked-heldout`
- `security/docs-site-no-runtime-risk-heldout`
- `security/go-cli-input-risk-heldout`

## Raw Runner Table

The table below is the runner-emitted per-case report from the real local-model
run. It is intentionally retained for traceability, including the emitted
failure classes from before the classifier correction.

agent-smoke full: 21 passed, 53 failed, 74 selected

- Evidence source: `local-model`
- Model source: local Mars Harness inference router

| Role | Case | Project | Mode | Disposition | Status | Failure | Run |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ceo` | `browser-game-empty` | `browser-game-phaser` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `ceo` | `docs-site-ambiguous-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `go-api-empty` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `react-web-conflicting-goals-heldout` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `static-web-empty` | `static-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `browser-game-after-ceo` | `browser-game-phaser` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `coo` | `docs-site-content-workflow-heldout` | `docs-site` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `coo` | `go-api-after-ceo` | `go-api` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `go-cli-scope-change-heldout` | `go-cli` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `coo` | `react-web-after-ceo` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `static-web-after-ceo` | `static-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `browser-game-after-coo` | `browser-game-phaser` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `cto-weekly` | `go-api-after-coo` | `go-api` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `cto-weekly` | `go-cli-after-coo` | `go-cli` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `go-library-ticket-gap-heldout` | `go-library` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `react-web-after-coo` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `static-web-duplicate-scenario-risk-heldout` | `static-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `breaking-frontend-update-heldout` | `react-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `browser-game-outdated` | `browser-game-phaser` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `go-api-stdlib-noop` | `go-api` | `live` | `in_review` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `go-cli-module-drift-heldout` | `go-cli` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `react-web-outdated` | `react-web` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `dogfood` | `canvas-game-defect-heldout` | `canvas-game-vanilla` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `dogfood` | `dogfood-browser-game-ready` | `browser-game-phaser` | `live` | `blocked` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `dogfood-go-api-ready` | `go-api` | `live` | `blocked` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `dogfood-react-web-ready` | `react-web` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `dogfood` | `dogfood-static-web-ready` | `static-web` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `dogfood` | `go-cli-defect-heldout` | `go-cli` | `live` | `blocked` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `browser-game-ticket` | `browser-game-phaser` | `live` | `no_work` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `engineer` | `canvas-game-smoke-gap-heldout` | `canvas-game-vanilla` | `live` | `no_work` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `engineer` | `go-api-ticket` | `go-api` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `engineer` | `go-cli-rework-heldout` | `go-cli` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `engineer` | `go-library-test-gap-heldout` | `go-library` | `live` | `no_work` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `engineer` | `react-web-ticket` | `react-web` | `live` | `no_work` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `engineer` | `static-web-ticket` | `static-web` | `live` | `no_work` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `foundation-maintainer` | `generated-doctrine-drift-heldout` | `docs-site` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `release-blocker-heldout` | `go-cli` | `source-only` | `blocked` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `source-dry-run` | `existing-maintenance` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `validation-classification` | `existing-maintenance` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `browser-game-roadmap-tradeoff` | `browser-game-phaser` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `head-of-strategy` | `docs-site-executive-narrative-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `go-api-conflicting-bets` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `go-cli-market-fit-heldout` | `go-cli` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `janitor` | `blocked-cli-ticket-heldout` | `go-cli` | `live` | `-` | `failed` | `dispatch-context` | `<validation-root>` |
| `janitor` | `clean-tree-heldout` | `docs-site` | `live` | `no_work` | `passed` | `-` | `discarded` |
| `janitor` | `stale-api-ticket` | `go-api` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `janitor` | `stale-game-rework` | `browser-game-phaser` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `janitor` | `stale-web-ticket` | `react-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `orchestrator` | `cto-to-engineer-api` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `engineer-to-qa-web` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `qa-to-security-game` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `release-blocked-stop-heldout` | `go-cli` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `orchestrator` | `runtime-failure-stop-heldout` | `existing-maintenance` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `browser-game-smoke-failure` | `browser-game-phaser` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `dependency-ci-heldout` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `foundation-runtime-heldout` | `existing-maintenance` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `go-api-test-failure` | `go-api` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `react-web-build-failure` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `qa` | `browser-game-after-engineer` | `browser-game-phaser` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `qa` | `canvas-game-bad-smoke-heldout` | `canvas-game-vanilla` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `qa` | `go-api-after-engineer` | `go-api` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `qa` | `go-cli-missing-evidence-heldout` | `go-cli` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `qa` | `react-web-after-engineer` | `react-web` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `qa` | `static-web-after-engineer` | `static-web` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `release-manager` | `docs-site-notes-only-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `release-manager` | `go-cli-blocked-heldout` | `go-cli` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `release-manager` | `release-browser-game-ready` | `browser-game-phaser` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `release-manager` | `release-go-api-ready` | `go-api` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
| `release-manager` | `release-react-web-ready` | `react-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `browser-game-after-qa` | `browser-game-phaser` | `live` | `completed` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `docs-site-no-runtime-risk-heldout` | `docs-site` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `go-api-after-qa` | `go-api` | `live` | `completed` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `go-cli-input-risk-heldout` | `go-cli` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `react-web-after-qa` | `react-web` | `live` | `-` | `failed` | `foundation-tool-generation` | `<validation-root>` |
