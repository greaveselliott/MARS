# Current Operating Plan

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** MH-034, MH-035, MH-037, MH-031, MH-030, MH-038, MH-039, MH-040, MH-041, MH-042, MH-043, MH-044, MH-045, MH-046, MH-047, MH-048, MH-049, MH-050, T-002, T-003, T-004, T-005, T-006, T-009
**Goals:** G-001, G-002, G-003, G-004
**BDD Feature:** F-001, F-004, F-009, F-012
**Hypothesis:** Returning to the Mars parity remediation lane after the observer-safe dry-run slice will keep the factory improving the runtime paths that most reduce wasted model turns and recovery loops.
**Success Evidence:** `T-009` closed with focused CLI tests, docs/tool-reference sync, and a live observer-safe dry-run against `/path/to/local-redacted` that left the real target clean.
**Falsification Evidence:** The next parity slice fails to broaden dogfood evidence, clean target replays create intervention-debt churn before product work, persistent store upgrades remain fixture-light, or completed remediation work remains in the active plan instead of returning to the parity queue.
**Scenario Schedule:** F-012-S010, F-001-S015, F-004-S007, F-012-S006, F-012-S007, F-009-S013
**Current Failing Scenario:** Persistent SQLite stores still lack broad legacy-schema upgrade fixtures beyond the queue regression that originally exposed `concurrency_group` migration ordering.
**Walking Skeleton Slice:** Claim `MH-050` next unless the operator explicitly prioritizes a live dogfood replay first.
**Learning Or MVP Outcome:** Future agents inherit the foundation/deployed architecture decision, generated target mirror, drift review, skill/tool decision, and a refreshed path back to runtime remediation work.
**Created:** 2026-05-02
**Owner:** Mars Harness maintainers
**Source:** Exec-plan review and repository state audit on 2026-05-02

## Purpose

This is the current execution map for Mars Harness. It exists because the
original master plan and delivery schedule are historical baseline
documents: useful for lineage, but stale as a status source.

Future agents should use this file, the ticket tree, and prioritized backlog
plans to decide what to do next.

## Current Truth

- Current source version is recorded in `VERSION`.
- Current branch: `main`
- Active goals live in `docs/goals/active.md`; the current plan references `G-001`, `G-002`, `G-003`, and `G-004`.
- BDD feature contracts live in `docs/features/`; the current operating-model feature is `F-001`, target-harness mirroring is `F-004`, release publication discipline is `F-009`, and feedback/self-improvement routing is `F-012`.
- Ticket state:
  - `docs/tickets/in-progress/` contains no tickets.
  - `docs/tickets/backlog/` contains `MH-050` and `T-001`.
  - `docs/tickets/done/` contains `MH-001` through `MH-049`, `T-002`, and
    `T-003` through `T-009`.
- Exec-plan state:
  - `docs/exec-plans/active/` contains exactly one active plan: this file.
  - `docs/exec-plans/backlog/` contains prioritized waiting plans with dependencies and blockers.
  - `docs/exec-plans/superseded/` contains historical plans that must not drive current work.
- GitHub release notes are published for semantic versions generated from `VERSION`.
- Release binary assets are published for `v0.14.5`; `MH-031` is done.
- `v0.21.0` release notes and tag were pushed on 2026-05-03, but
  `mars-harness release verify-assets --version v0.21.0` is blocked because
  GitHub returned `404 Not Found` for the tag release immediately after the
  tag push.
- `v0.23.0` release notes and tag were pushed on 2026-05-03 for `MH-047`, but
  `go run ./cmd/mars-harness release verify-assets --version v0.23.0` is
  blocked because GitHub returned `404 Not Found` for the tag release
  immediately after the tag push.
- `v0.36.4`, `v0.36.5`, `v0.36.6`, `v0.37.0`, `v0.38.0`, `v0.39.0`,
  `v0.40.0`, `v0.40.1`, `v0.41.0`, `v0.41.1`, `v0.41.2`, `v0.41.3`,
  `v0.41.4`, `v0.41.5`, `v0.41.6`, `v0.41.7`, `v0.41.8`, `v0.41.9`,
  `v0.41.10`, `v0.41.11`, `v0.41.12`, `v0.41.13`, `v0.41.14`,
  `v0.41.15`, `v0.41.16`, `v0.41.17`, `v0.41.18`, `v0.41.19`, `v0.41.20`,
  `v0.41.21`, `v0.41.22`, `v0.41.23`, and `v0.41.24` release notes and tags
  were pushed on 2026-05-19, but CI and Release workflow jobs were not started
  because GitHub reported recent account payment failure or a spending-limit
  increase requirement.
  Notes-only GitHub Releases for `v0.36.4` through `v0.41.24` were created from
  the generated changelog entries on 2026-05-19 so the Releases page is no
  longer stale at `v0.36.3`. `mars-harness release verify-assets --version
  v0.41.24` is still blocked because the `v0.41.24` release is missing
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`; GitHub Actions runs `26128695343` and `26128778584` failed
  with "recent account payments have failed or your spending limit needs to be
  increased" before assets could be built. The `v0.41.24` main-branch CI run
  `26128685246` hit the same runner-start billing blocker. `v0.41.16`,
  `v0.41.17`, `v0.41.18`, `v0.41.19`, `v0.41.20`, `v0.41.21`, `v0.41.22`,
  and `v0.41.23` have the same missing-asset blocker via runs
  `26126035892`, `26126035944`, `26126461151`, `26127153189`, `26127529878`,
  `26127808895`, `26128342280`, and `26128605770`.
- Model evaluation, Ollama catalog support, model overrides, persisted reports,
  repo-backed benchmark cases, and promotion blocking shipped under `MH-030`.

## Plan State

| Plan | State | Depends On | Blocks | How to use it |
| --- | --- | --- | --- | --- |
| `active/current-operating-plan.md` | Active, P0 | None | Plan promotions until this file names the next slice | Use this file as the only top-level execution map and scenario schedule. |
| `backlog/mars-parity-supersession-plan.md` | Backlog, P1, G-001/G-002/F-001 | Ticket materialization completed on 2026-05-03; no open blocker | Supersession readiness claims | Pull slices into tickets and this active plan before execution. |
| `backlog/model-evaluation-refresh-plan.md` | Completed initial model-refresh slice; future P4, G-003/F-001 | Higher-priority release/quality work for any future promotion ticket | Default model registry promotion until live benchmark evidence exists | Use only to seed future model-refresh tickets. |
| `superseded/master-execution-plan.md` | Superseded | None | Nothing | Historical baseline. Do not use its checkbox status as truth. |
| `superseded/delivery-schedule.md` | Superseded | None | Nothing | Historical milestone schedule; kept for lineage only. |

## Current Priority Order

1. **Persistent store upgrade fixtures**: Implement `MH-050` so every
   persistent SQLite store has legacy-schema fixture coverage and migration
   ordering checks.
2. **Mars parity continuation**: Use
   [OpenHarness comparator](../../references/openharness-comparator.md)
   as reference input for readiness, skill metadata, compaction, and
   remediation ergonomics without creating a parallel roadmap.

## Scenario Schedule

| Scenario | Status | Evidence |
| --- | --- | --- |
| F-001-S001 | Passing | `docs/goals/active.md`, `docs/features/F-001-delivery-operating-model.md`, and this active plan are linked. |
| F-001-S002 | Passing | Engineer ticket gate validates feature ticket evidence before done. |
| F-001-S003 | Passing | `mars-harness init` emits goals, feature contracts, AD-074, role guidance, ticket metadata, exec-plan metadata, quality guidance, and knowledge routes. |
| F-001-S004 | Passing | `update check` and `doctor --repo` report operating-model drift without overwriting user-owned docs. |
| F-001-S005 | Passing | Ticket docs, quality score, and release-note generation distinguish shipped feature scenarios from enabler work when commits reference tickets. |
| F-001-S006 | Passing | Telemetry proposals can create or update active goals/observations with dedupe evidence. |
| F-001-S007 | Passing | `go test ./internal/docsconsistency -run TestFeatureContractsDeclareRequiredFields` checks feature contracts include first-class business-logic sections. |
| F-001-S008 | Passing | `go test ./internal/docsconsistency -run TestOperatingModelCodeFilesDeclareDocSyncMetadata` checks operating-model code files carry associated documentation metadata. |
| F-012-S010 | Passing | `go test ./internal/remediation`, `go test ./internal/serve -run 'TestHandleJobFailed(RecordsDeterministicRemediation|ExecutesGeneratedDocs)|TestHandleRemediation(ExecutableReadyRecipe|AutoSafeWithoutExecutor|OperatorRecipe)'`, `go test ./internal/doctor -run TestCheckDeterministicRemediationHealth`, `go test ./internal/qualityscore -run TestExportRendersTelemetryAndOutcomeSignals`, `go test ./internal/docsconsistency ./internal/docsync`, `go test ./...`, and the 2026-05-19 clean `<validation-root>` replay cover the completed `MH-048` edge slice. |
| F-001-S015 | Passing | [foundation-deployed-harness-architecture.md](../../design-docs/foundation-deployed-harness-architecture.md) records the foundation/deployed doctrine boundary, feedback routing, tool/skill/runtime split, generated-target implications, doctrine-maintenance duties, and 2026-05-19 drift review. |
| F-004-S007 | Passing | `go test ./internal/scanner -run TestInit_success` verifies generated targets receive the foundation/deployed route and AD-139 core doctrine without source binary asset names. |
| F-012-S006 | Passing | [skill-evolution.md](../../design-docs/skill-evolution.md) AD-140 keeps the recursive improvement loop as operating doctrine and creates `T-006` for a foundation Release Manager skill. |
| F-012-S007 | Passing | Generated target knowledge routes and mirrored harness docs carry the reusable feedback and improvement-loop doctrine after the AD-139 source doc. |
| F-009-S013 | Passing | `go test ./internal/docsconsistency ./internal/docsync` and `gh release view v0.41.22 --repo greaveselliott/mars-harness` cover the release-object gate and notes-only fallback. `mars-harness release verify-assets --version v0.41.22` records the separate missing-asset blocker. |
| F-010-S003 | Passing | `go test ./internal/serve -run 'TestServer_(dashboardStopEndpointStopsStart|startStop)'`, `go test ./internal/dashboard -run 'TestDashboard_(stopEndpoint|controlEndpoints_methodNotAllowed|controlEndpoints_nilCallbacks)'`, and the 2026-05-19 clean `demo-123-stop-check2` replay verify dashboard stop returns success and exits `start` without manual kill. |
| F-005-S006 | Passing | `go test ./cmd/mars-harness -run 'TestRunCommand(NoInit|AutoInit|RejectsRepoLocalLogFile)|TestMarsHarnessCLI'` and `go run ./cmd/mars-harness run engineer --repo /path/to/local-redacted --dry-run --trace --no-init` verify observer-safe dry-run exits without scaffolding an uninitialized target. |

## Quality State

Checks recorded during the 2026-05-02 review:

- `go test ./...` passes.
- `go test -cover ./...` passes after making update-check fixtures
  release-agnostic.
- Coverage remains below the stated 70 percent target in several packages,
  including `internal/inference`, `internal/setup`, `internal/ui`,
  `internal/serve`, `internal/hardware`, and command entrypoint code.
- `golangci-lint` was not installed in the local environment during the 2026-05-02
  review, so local lint status is unknown.

## Operating Rules

- In-progress tickets are still highest priority. If any appear, drain them
  before taking backlog work.
- Intervention-debt tickets outrank ordinary feature work unless explicitly
  downgraded.
- Superseded plans should not remain silently active. Either reconcile them,
  move them, or mark them with a clear status and pointer to the current plan.
- Large strategic plans must be materialized into ticket files before agents
  are expected to execute them autonomously.
- Every non-release semantic commit still requires generated release notes, a
  matching release commit, a pushed `vX.Y.Z` tag, `gh release view vX.Y.Z`, a
  notes-only GitHub Release from `CHANGELOG.md` if the tag workflow did not
  create one, and `mars-harness release verify-assets --version vX.Y.Z` before
  release work can be claimed complete. Missing binary assets are recorded as a
  blocker instead of allowing the GitHub Releases page to stay stale.

## Next Ticket Work

- `MH-048`: deterministic remediation recipes are done. The completed slices
  add the remediation registry, applicability planner, trace-linked score
  evidence in `serve`, generated-docs execution through `scanner.Upgrade`,
  doctor recipe output, score-export summaries, destructive-git negative-path
  coverage, dirty-worktree blockers, and missing-optional-tool guidance.
- `MH-049`: dogfood matrix supersession benchmark is done as of 2026-05-19. The
  2026-05-19 slices add broader fake-LLM loop coverage, observer-trust mutation
  blocking, a live `demo-123` lifecycle report that reached product
  implementation, QA, Security, Dogfood, and Release Manager, and the first
  Mars observer validation report. This ticket is done.
- `T-007`: deployed `mars_harness_cli` binary resolution is done as of
  2026-05-19. The tool prefers the active harness executable before stale PATH
  binaries and adds actionable stale-binary guidance.
- `T-008`: dashboard stop is done as of 2026-05-19. Stop requests now route
  through the server loop, the regression test covers `/api/stop`, and the
  clean `demo-123-stop-check2` replay exited `start` without manual kill.
- `T-002`: foundation/deployed architecture source doc is done and should be
  used as the input for mirroring and drift review.
- `T-003`: generated target mirroring is done and should be used as input for
  the drift review.
- `T-004`: doctrine drift review is done and found no unowned mismatch across
  source and generated target surfaces.
- `T-005`: skill/tool/doctrine evaluation is done; the recursive improvement
  loop remains operating doctrine and `T-006` captures the foundation Release
  Manager skill implementation.
- `T-006`: foundation release publication skill is done as of 2026-05-19.
  `.harness/skills/release-publication/SKILL.md` covers release-note
  commit, push, tag, GitHub Release object, notes-only fallback, asset
  verification, token safety, and blocker recording. Generated target mirroring
  is deferred with rationale because target publication modes differ from the
  source binary-release workflow.
- `T-009`: observer-safe dry-run is done as of 2026-05-19. `run --dry-run
  --no-init` reports the missing-harness boundary without writing `.harness/`,
  the Mars observer profile uses it, and the live Mars checkout stayed clean.
- `MH-050`: next parity slice. Add persistent store upgrade fixtures so legacy
  SQLite schemas are exercised across queue, telemetry, trace, scoring, trust,
  evolution, orgstate, and any other store initialization path.
