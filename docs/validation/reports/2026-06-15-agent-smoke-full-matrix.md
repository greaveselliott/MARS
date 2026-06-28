# Validation Report: Agent Smoke Full Matrix

## Status

**Status:** validated all-pass full matrix.

The full `agent-smoke` matrix was run against the real local MARS
model path with no fake, stub, mock, canned, or scripted endpoint. The runner
generated ephemeral targets, executed every deployable target role through the
live server job path, ran two cases concurrently through one shared local
inference server, wrote a Markdown report, and discarded all successful run
directories. The source-only `foundation-maintainer` role remained explicitly
source-only in the report rather than being falsely mirrored into generated
target manifests.

Final result: `74 passed`, `0 failed`, `74 selected`: `70` live role jobs and
`4` source-only foundation-maintainer checks.

This report supersedes the earlier failed June 14 matrix report for completion
claims. The earlier report remains useful historical evidence for the failure
analysis and fixes that led to this all-pass run.

## Command

```bash
mars validation agent-smoke --suite full --parallel 2 --timeout 45m --report <validation-root>
```

- Date: 2026-06-15
- Installed command: `/path/to/local-redacted`
- Evidence source: `local-model`
- Model source: local MARS inference router; single local server tier `coding`
- Inference topology: one local llama-server tier with server parallel `2`
- Execution modes: `70` live server-job runs, `4` source-only foundation-maintainer checks
- Server port: `18080`
- Cleanup: all `74` successful run directories reported as `discarded`; no failed run directory remained from this final run
- Raw report path: `<validation-root>`

## Focused Rerun Evidence Before Final Matrix

Before the final all-pass matrix, the last retained failures were fixed and
validated with focused live runs:

```bash
mars validation agent-smoke --role janitor --case stale-game-rework --suite full --parallel 1 --timeout 20m --report <validation-root>
mars validation agent-smoke --role pipeline-fixer --case dependency-ci-heldout --suite full --parallel 1 --timeout 20m --report <validation-root>
mars validation agent-smoke --role dogfood --case dogfood-go-api-ready --suite full --parallel 1 --timeout 20m --report <validation-root>
```

All three focused reruns passed live before the full matrix was restarted.

## Validated Fixes

- Janitor stale-ticket fixtures now seed existing evidence before Janitor moves
  stale tickets to `docs/tickets/done/`, preserving ticket policy truth.
- Pipeline Fixer React cases now use React/Vite validation guidance and a
  targeted policy correction for `go test ./...` in non-Go agent-smoke targets.
- Dogfood Go ready cases now name `go test ./...` as the bounded user smoke and
  require the dogfood report/commit/disposition sequence, preventing no-op
  shell loops.
- The no-fake-endpoint design rule in AD-296 was upheld: this validation claim
  is based on the local model router, not a scripted endpoint.

## Per-Role Summary

| Role | Passed | Failed |
| --- | ---: | ---: |
| `ceo` | 5 | 0 |
| `coo` | 6 | 0 |
| `cto-weekly` | 6 | 0 |
| `dependency-manager` | 5 | 0 |
| `dogfood` | 6 | 0 |
| `engineer` | 7 | 0 |
| `foundation-maintainer` | 4 | 0 |
| `head-of-strategy` | 4 | 0 |
| `janitor` | 5 | 0 |
| `orchestrator` | 5 | 0 |
| `pipeline-fixer` | 5 | 0 |
| `qa` | 6 | 0 |
| `release-manager` | 5 | 0 |
| `security` | 5 | 0 |

## Per-Project-Type Summary

| Project Type | Passed | Failed |
| --- | ---: | ---: |
| `static-web` | 6 | 0 |
| `react-web` | 14 | 0 |
| `browser-game-phaser` | 13 | 0 |
| `canvas-game-vanilla` | 3 | 0 |
| `go-api` | 13 | 0 |
| `go-cli` | 12 | 0 |
| `go-library` | 2 | 0 |
| `docs-site` | 7 | 0 |
| `existing-maintenance` | 4 | 0 |

## Raw Runner Table

agent-smoke full: 74 passed, 0 failed, 74 selected

- Evidence source: `local-model`
- Model source: local MARS inference router; single local server tier coding
- Inference topology: single local server tier `coding` with server parallel `2`

| Role | Case | Project | Mode | Disposition | Status | Failure | Run |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ceo` | `browser-game-empty` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `docs-site-ambiguous-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `go-api-empty` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `react-web-conflicting-goals-heldout` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `ceo` | `static-web-empty` | `static-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `browser-game-after-ceo` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `docs-site-content-workflow-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `go-api-after-ceo` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `go-cli-scope-change-heldout` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `react-web-after-ceo` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `coo` | `static-web-after-ceo` | `static-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `cto-weekly` | `browser-game-after-coo` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `cto-weekly` | `go-api-after-coo` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `cto-weekly` | `go-cli-after-coo` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `cto-weekly` | `go-library-ticket-gap-heldout` | `go-library` | `live` | `completed` | `passed` | `-` | `discarded` |
| `cto-weekly` | `react-web-after-coo` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `cto-weekly` | `static-web-duplicate-scenario-risk-heldout` | `static-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `dependency-manager` | `breaking-frontend-update-heldout` | `react-web` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `dependency-manager` | `browser-game-outdated` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `dependency-manager` | `go-api-stdlib-noop` | `go-api` | `live` | `no_work` | `passed` | `-` | `discarded` |
| `dependency-manager` | `go-cli-module-drift-heldout` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `dependency-manager` | `react-web-outdated` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `dogfood` | `canvas-game-defect-heldout` | `canvas-game-vanilla` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `dogfood` | `dogfood-browser-game-ready` | `browser-game-phaser` | `live` | `approved` | `passed` | `-` | `discarded` |
| `dogfood` | `dogfood-go-api-ready` | `go-api` | `live` | `approved` | `passed` | `-` | `discarded` |
| `dogfood` | `dogfood-react-web-ready` | `react-web` | `live` | `approved` | `passed` | `-` | `discarded` |
| `dogfood` | `dogfood-static-web-ready` | `static-web` | `live` | `approved` | `passed` | `-` | `discarded` |
| `dogfood` | `go-cli-defect-heldout` | `go-cli` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `engineer` | `browser-game-ticket` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `engineer` | `canvas-game-smoke-gap-heldout` | `canvas-game-vanilla` | `live` | `completed` | `passed` | `-` | `discarded` |
| `engineer` | `go-api-ticket` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `engineer` | `go-cli-rework-heldout` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `engineer` | `go-library-test-gap-heldout` | `go-library` | `live` | `completed` | `passed` | `-` | `discarded` |
| `engineer` | `react-web-ticket` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `engineer` | `static-web-ticket` | `static-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `generated-doctrine-drift-heldout` | `docs-site` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `release-blocker-heldout` | `go-cli` | `source-only` | `blocked` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `source-dry-run` | `existing-maintenance` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `foundation-maintainer` | `validation-classification` | `existing-maintenance` | `source-only` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `browser-game-roadmap-tradeoff` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `docs-site-executive-narrative-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `go-api-conflicting-bets` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `head-of-strategy` | `go-cli-market-fit-heldout` | `go-cli` | `live` | `completed` | `passed` | `-` | `discarded` |
| `janitor` | `blocked-cli-ticket-heldout` | `go-cli` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `janitor` | `clean-tree-heldout` | `docs-site` | `live` | `no_work` | `passed` | `-` | `discarded` |
| `janitor` | `stale-api-ticket` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `janitor` | `stale-game-rework` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `janitor` | `stale-web-ticket` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `cto-to-engineer-api` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `engineer-to-qa-web` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `qa-to-security-game` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `orchestrator` | `release-blocked-stop-heldout` | `go-cli` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `orchestrator` | `runtime-failure-stop-heldout` | `existing-maintenance` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `browser-game-smoke-failure` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `dependency-ci-heldout` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `foundation-runtime-heldout` | `existing-maintenance` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `go-api-test-failure` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `pipeline-fixer` | `react-web-build-failure` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `qa` | `browser-game-after-engineer` | `browser-game-phaser` | `live` | `approved` | `passed` | `-` | `discarded` |
| `qa` | `canvas-game-bad-smoke-heldout` | `canvas-game-vanilla` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `qa` | `go-api-after-engineer` | `go-api` | `live` | `approved` | `passed` | `-` | `discarded` |
| `qa` | `go-cli-missing-evidence-heldout` | `go-cli` | `live` | `changes_requested` | `passed` | `-` | `discarded` |
| `qa` | `react-web-after-engineer` | `react-web` | `live` | `approved` | `passed` | `-` | `discarded` |
| `qa` | `static-web-after-engineer` | `static-web` | `live` | `approved` | `passed` | `-` | `discarded` |
| `release-manager` | `docs-site-notes-only-heldout` | `docs-site` | `live` | `completed` | `passed` | `-` | `discarded` |
| `release-manager` | `go-cli-blocked-heldout` | `go-cli` | `live` | `blocked` | `passed` | `-` | `discarded` |
| `release-manager` | `release-browser-game-ready` | `browser-game-phaser` | `live` | `completed` | `passed` | `-` | `discarded` |
| `release-manager` | `release-go-api-ready` | `go-api` | `live` | `completed` | `passed` | `-` | `discarded` |
| `release-manager` | `release-react-web-ready` | `react-web` | `live` | `completed` | `passed` | `-` | `discarded` |
| `security` | `browser-game-after-qa` | `browser-game-phaser` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `docs-site-no-runtime-risk-heldout` | `docs-site` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `go-api-after-qa` | `go-api` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `go-cli-input-risk-heldout` | `go-cli` | `live` | `approved` | `passed` | `-` | `discarded` |
| `security` | `react-web-after-qa` | `react-web` | `live` | `approved` | `passed` | `-` | `discarded` |
