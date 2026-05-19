# Current Operating Plan

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** MH-034, MH-035, MH-037, MH-031, MH-030, MH-038, MH-039, MH-040, MH-041, MH-042, MH-043, MH-044, MH-045, MH-046, MH-047, MH-048, MH-049
**Goals:** G-001, G-002, G-003, G-004
**BDD Feature:** F-001, F-002, F-006, F-009
**Hypothesis:** Zero-config CLI availability removes the first-run `Unknown command: mars-harness` failure mode while the BDD-led operating loop keeps completion evidence-based, and native Orchestrator surveys prevent unattended failure states from silently stalling work.
**Success Evidence:** `make install`, `mars-harness setup`, and `mars-harness update tool` configure supported user shell profiles idempotently, F-001 remains passing including first-class BDD and no-stale-documentation checks, and `F-006-S009` has survey-to-queue tests for stale tickets, failed checks, no-op outcomes, telemetry patterns, low scores, queue ownership, payload modes, concurrency groups, daily caps, and stuck-job watchdog behavior.
**Falsification Evidence:** Fish, Zsh, Bash, or POSIX shell users still need manual PATH edits after install/update/setup, or stale ticket/check/telemetry/score/no-op signals remain unrouted until a human or GitHub event intervenes.
**Scenario Schedule:** F-001-S007, F-001-S008, F-002-S001, F-002-S002, F-002-S003, F-002-S004, F-002-S005, F-006-S009, F-009-S008
**Current Failing Scenario:** None for F-002 or F-006-S009; continue the Mars parity backlog with deterministic remediation.
**Walking Skeleton Slice:** Use one shellpath package across source install, first-run setup, update-tool reinstall, and an explicit `mars-harness path setup` command; use one native Orchestrator survey path across ticket ownership, telemetry triage, failed checks, dogfood failures, no-op outcomes, queue caps, and stuck-job watchdogs.
**Learning Or MVP Outcome:** Users can install or update once and run `mars-harness` from new terminals across common shells without hand-written shell setup; as of 2026-05-03, unattended harness failure states become bounded queue work or deduped intervention-debt tickets.
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
- BDD feature contracts live in `docs/features/`; the current operating-model feature is `F-001`, the current install/setup feature is `F-002`, the current queue/orchestration survey scenario is `F-006-S009`, and detailed release-note narrative is `F-009-S008`.
- Ticket state:
  - `docs/tickets/in-progress/` contains `MH-048`.
  - `docs/tickets/backlog/` contains `MH-049`.
  - `docs/tickets/done/` contains `MH-001` through `MH-047`.
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
  `v0.40.0`, `v0.40.1`, `v0.41.0`, and `v0.41.1` release notes and tags were
  pushed on 2026-05-19, but CI and Release workflow jobs were not started
  because GitHub reported recent account payment failure or a spending-limit
  increase requirement. `mars-harness release verify-assets --version v0.41.1` is
  blocked because the tag release returns `404 Not Found` until the workflow can
  publish assets or a release backfill is run after the billing blocker is
  cleared.
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

1. **Mars parity execution**: Continue deterministic remediation (`MH-048`,
   claimed under `docs/tickets/in-progress/` on 2026-05-19), then the dogfood
   matrix (`MH-049`).
   Use [OpenHarness comparator](../../references/openharness-comparator.md)
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
| F-002-S001 | Passing | `make install` invokes `mars-harness path setup --install-dir <dir>` after `go install`. |
| F-002-S002 | Passing | `mars-harness setup` includes a `configure-shell-path` step. |
| F-002-S003 | Passing | `mars-harness update tool` runs shell PATH setup after reinstalling the binary. |
| F-002-S004 | Passing | Shell profile updates are idempotent and covered by tests. |
| F-002-S005 | Passing | Unsupported shells return explicit manual remediation without writing profile files. |
| F-006-S009 | Passing | `go test ./internal/serve -run TestOrchestratorSurvey` and `go test ./internal/queue -run 'TestQueue_concurrencyGroupSerialization|TestQueue_dailyCapConstrainsRepeatedScheduling|TestQueue_claimDoesNotResetHealthyRunningJob|TestQueue_failStuckRunningJobs'` cover native survey routing, ownership metadata, daily caps, telemetry/score triage, no-op detection, and stuck-job safety. |
| F-009-S008 | Passing | `go test ./internal/release -run 'TestRenderReleaseNarrative(UsesImpactWhyAndWhat|ProfilesStructuredDispatch)'` covers generated Impact, Why, What Changed, and topic-aware structured-dispatch fallback narrative. |

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
- Every non-release semantic commit still requires generated release notes and
  a matching release commit before pushing `main`.

## Next Ticket Work

- `MH-048`: deterministic remediation recipes are in progress. The first
  completed slices add the remediation registry, applicability planner, and
  trace-linked score evidence in `serve`; as of 2026-05-19, the generated-docs
  auto-safe recipe executes through `scanner.Upgrade`, and `doctor --repo`
  reports manifest/generated-docs recipe IDs before runtime. The next slice
  should decide whether MH-048 is complete or needs another permanent check.
- Next active-plan refresh: promote the deterministic-first repair scenario group.
