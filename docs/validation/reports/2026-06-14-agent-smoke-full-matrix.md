# Validation Report: Agent Smoke Full Matrix

## Status

**Status:** validated single-server parallel setup and live execution path; matrix failing.

The full `agent-smoke` matrix was run against a real local Mars Harness model
with no `--model-endpoint` override and no fake/stub endpoint. The lane
generated ephemeral targets, executed selected roles through the server job
path, wrote a Markdown report, discarded successful run directories, and
retained failed run directories for diagnosis.

The setup works in practice: all `74` assigned role/project cases were selected,
the runner used one shared local server, two cases executed concurrently, and
the final shutdown reported `inference router: stopped managed servers count=1`.

The matrix did **not** pass: `22 passed`, `52 failed`, `74 selected`.

## Command

```bash
/path/to/local-redacted validation agent-smoke --suite full --parallel 2 --single-server --single-server-tier coding --max-turns 2 --timeout 10m --root <validation-root> --report <validation-root>
```

- Date: 2026-06-14
- Started: 2026-06-14T14:18:12Z
- Finished: 2026-06-14T14:44:02Z
- Installed binary: `/path/to/local-redacted`
- Binary version: `0.60.2`
- Source ref at start: `7e79d8d` plus the working-tree single-server/context changes captured by this report update
- Evidence source: `local-model`
- Model source: local Mars Harness inference router; single local server tier `coding`
- Inference topology: one local server on port `18080`, `--parallel 2`, server `--ctx-size 65536`, preserving a 32768-token per-slot coding-tier window
- Hardware profile outside sandbox: `high`
- Cleanup: 22 successful runs discarded; 52 failed runs retained under `<validation-root>`

## Model Identity

| Tier Used | Model | Quantization | Server Context | Per-Slot Context | Port |
| --- | --- | --- | ---: | ---: | ---: |
| coding | `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` | Q4_K_M | 65536 | 32768 | 18080 |

The earlier unvalidated serial run used tiered routing across three model
servers. This report supersedes it for the Compartmentalised Agent Smoke
testing claim because the primary lane requires one local server with parallel
case execution.

## Result Summary

| Role | Passed | Failed |
| --- | ---: | ---: |
| `ceo` | 4 | 1 |
| `head-of-strategy` | 3 | 1 |
| `coo` | 0 | 6 |
| `cto-weekly` | 0 | 6 |
| `engineer` | 0 | 7 |
| `qa` | 2 | 4 |
| `security` | 2 | 3 |
| `dependency-manager` | 2 | 3 |
| `release-manager` | 0 | 5 |
| `dogfood` | 1 | 5 |
| `pipeline-fixer` | 0 | 5 |
| `orchestrator` | 2 | 3 |
| `janitor` | 2 | 3 |
| `foundation-maintainer` | 4 | 0 |

| Project Type | Passed | Failed |
| --- | ---: | ---: |
| `static-web` | 1 | 5 |
| `react-web` | 2 | 12 |
| `browser-game-phaser` | 1 | 12 |
| `canvas-game-vanilla` | 1 | 2 |
| `go-api` | 3 | 10 |
| `go-cli` | 6 | 6 |
| `go-library` | 0 | 2 |
| `docs-site` | 5 | 2 |
| `existing-maintenance` | 3 | 1 |

## Failure Summary

All emitted failures from the validated full run were classified as
`role-behavior`. The earlier classifier bug that mislabeled max-turns and
ticket-gate failures as `foundation-tool-generation` did not recur.

| Signature | Count |
| --- | ---: |
| wrong terminal disposition | 35 |
| max turns before terminal contract | 10 |
| Engineer ticket gate not satisfied | 7 |

## Passing Cases

- `ceo/docs-site-ambiguous-heldout`
- `ceo/go-api-empty`
- `ceo/react-web-conflicting-goals-heldout`
- `ceo/static-web-empty`
- `dependency-manager/go-api-stdlib-noop`
- `dependency-manager/go-cli-module-drift-heldout`
- `dogfood/go-cli-defect-heldout`
- `foundation-maintainer/generated-doctrine-drift-heldout`
- `foundation-maintainer/release-blocker-heldout`
- `foundation-maintainer/source-dry-run`
- `foundation-maintainer/validation-classification`
- `head-of-strategy/docs-site-executive-narrative-heldout`
- `head-of-strategy/go-api-conflicting-bets`
- `head-of-strategy/go-cli-market-fit-heldout`
- `janitor/blocked-cli-ticket-heldout`
- `janitor/clean-tree-heldout`
- `orchestrator/qa-to-security-game`
- `orchestrator/runtime-failure-stop-heldout`
- `qa/canvas-game-bad-smoke-heldout`
- `qa/go-cli-missing-evidence-heldout`
- `security/docs-site-no-runtime-risk-heldout`
- `security/react-web-after-qa`

## Validation Notes

- A focused fast-suite proof first ran with one server and `--parallel 2`; it
  exposed that llama-server divided the configured context across slots.
- The router was corrected before the full run to scale total server context by
  slot count, preserving the tier window for each concurrent request.
- The corrected fast-suite rerun completed `14 selected`, `5 passed`,
  `9 failed`, with `single_server=true`, `single_server_tier=coding`,
  `server_parallel=2`, and no context-overflow setup failure.
- The full run then validated the complete 74-case matrix under the same
  topology.

## Raw Runner Table

The table below is the runner-emitted per-case report from the real local-model
single-server parallel run.

agent-smoke full: 22 passed, 52 failed, 74 selected

- Evidence source: `local-model`
- Model source: local Mars Harness inference router; single local server tier coding
- Inference topology: single local server tier `coding` with server parallel `2`

| Role | Case | Project | Mode | Disposition | Status | Failure | Run |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ceo` | `browser-game-empty` | `browser-game-phaser` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `ceo` | `docs-site-ambiguous-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `go-api-empty` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `react-web-conflicting-goals-heldout` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `static-web-empty` | `static-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `browser-game-after-ceo` | `browser-game-phaser` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `docs-site-content-workflow-heldout` | `docs-site` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `go-api-after-ceo` | `go-api` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `go-cli-scope-change-heldout` | `go-cli` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `react-web-after-ceo` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `coo` | `static-web-after-ceo` | `static-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `browser-game-after-coo` | `browser-game-phaser` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `go-api-after-coo` | `go-api` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `go-cli-after-coo` | `go-cli` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `go-library-ticket-gap-heldout` | `go-library` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `react-web-after-coo` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `cto-weekly` | `static-web-duplicate-scenario-risk-heldout` | `static-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `breaking-frontend-update-heldout` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `browser-game-outdated` | `browser-game-phaser` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `dependency-manager` | `go-api-stdlib-noop` | `go-api` | `live` | `no_work` | `passed` | `-` | `discarded` |
| `dependency-manager` | `go-cli-module-drift-heldout` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `dependency-manager` | `react-web-outdated` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `canvas-game-defect-heldout` | `canvas-game-vanilla` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `dogfood-browser-game-ready` | `browser-game-phaser` | `live` | `blocked` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `dogfood-go-api-ready` | `go-api` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `dogfood-react-web-ready` | `react-web` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `dogfood-static-web-ready` | `static-web` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `dogfood` | `go-cli-defect-heldout` | `go-cli` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `engineer` | `browser-game-ticket` | `browser-game-phaser` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `canvas-game-smoke-gap-heldout` | `canvas-game-vanilla` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `go-api-ticket` | `go-api` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `go-cli-rework-heldout` | `go-cli` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `go-library-test-gap-heldout` | `go-library` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `react-web-ticket` | `react-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `engineer` | `static-web-ticket` | `static-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `foundation-maintainer` | `generated-doctrine-drift-heldout` | `docs-site` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `release-blocker-heldout` | `go-cli` | `source-only` | `blocked` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `source-dry-run` | `existing-maintenance` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `validation-classification` | `existing-maintenance` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `browser-game-roadmap-tradeoff` | `browser-game-phaser` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `head-of-strategy` | `docs-site-executive-narrative-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `go-api-conflicting-bets` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `go-cli-market-fit-heldout` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `janitor` | `blocked-cli-ticket-heldout` | `go-cli` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `janitor` | `clean-tree-heldout` | `docs-site` | `live` | `no_work` | `passed` | `-` | `discarded` |
| `janitor` | `stale-api-ticket` | `go-api` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `janitor` | `stale-game-rework` | `browser-game-phaser` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `janitor` | `stale-web-ticket` | `react-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `orchestrator` | `cto-to-engineer-api` | `go-api` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `orchestrator` | `engineer-to-qa-web` | `react-web` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `orchestrator` | `qa-to-security-game` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `release-blocked-stop-heldout` | `go-cli` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `orchestrator` | `runtime-failure-stop-heldout` | `existing-maintenance` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `browser-game-smoke-failure` | `browser-game-phaser` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `dependency-ci-heldout` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `foundation-runtime-heldout` | `existing-maintenance` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `go-api-test-failure` | `go-api` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `pipeline-fixer` | `react-web-build-failure` | `react-web` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `qa` | `browser-game-after-engineer` | `browser-game-phaser` | `live` | `changes_requested` | `failed` | `role-behavior` | `<validation-root>` |
| `qa` | `canvas-game-bad-smoke-heldout` | `canvas-game-vanilla` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `qa` | `go-api-after-engineer` | `go-api` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `qa` | `go-cli-missing-evidence-heldout` | `go-cli` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `qa` | `react-web-after-engineer` | `react-web` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `qa` | `static-web-after-engineer` | `static-web` | `live` | `changes_requested` | `failed` | `role-behavior` | `<validation-root>` |
| `release-manager` | `docs-site-notes-only-heldout` | `docs-site` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `release-manager` | `go-cli-blocked-heldout` | `go-cli` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `release-manager` | `release-browser-game-ready` | `browser-game-phaser` | `live` | `failed` | `failed` | `role-behavior` | `<validation-root>` |
| `release-manager` | `release-go-api-ready` | `go-api` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `release-manager` | `release-react-web-ready` | `react-web` | `live` | `no_work` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `browser-game-after-qa` | `browser-game-phaser` | `live` | `-` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `docs-site-no-runtime-risk-heldout` | `docs-site` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `go-api-after-qa` | `go-api` | `live` | `completed` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `go-cli-input-risk-heldout` | `go-cli` | `live` | `completed` | `failed` | `role-behavior` | `<validation-root>` |
| `security` | `react-web-after-qa` | `react-web` | `live` | `approved` | `passed` | `-` | `discarded` |
