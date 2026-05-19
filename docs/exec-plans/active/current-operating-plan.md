# Current Operating Plan

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** MH-034, MH-035, MH-037, MH-031, MH-030, MH-038, MH-039, MH-040, MH-041, MH-042, MH-043, MH-044, MH-045, MH-046, MH-047, MH-048, MH-049, T-002, T-003, T-004, T-005
**Goals:** G-001, G-002, G-003, G-004
**BDD Feature:** F-001, F-004, F-009, F-012
**Hypothesis:** Naming the foundation/deployed harness architecture will reduce doctrine drift by making clear which feedback belongs to the foundation harness, which belongs to a deployed harness, which rules mirror, and which source-only release or runtime mechanics must not leak into target guidance.
**Success Evidence:** A glossary-grounded design doc explains why the foundation/deployed split exists, how feedback is collected and routed, how doctrine is maintained as the system evolves, and how generated target guidance mirrors only the reusable core; docsconsistency, docsync, and scanner checks pass for the mirrored surfaces that change.
**Falsification Evidence:** Agents still have to infer whether a rule is foundation-only or deployed, feedback still becomes target backlog churn when it is foundation-owned, generated target guidance imports source-only release asset mechanics, or doctrine changes can land without updating the owning docs/routes/tests.
**Scenario Schedule:** F-001-S015, F-004-S007, F-012-S006, F-012-S007, F-009-S013
**Current Failing Scenario:** F-001-S015 and F-004-S007 need a recorded doctrine drift review after the source architecture doc and generated target route landed.
**Walking Skeleton Slice:** Run the doctrine drift review, record any remaining mismatch as follow-up work, then decide whether the recursive improvement loop should become a skill.
**Learning Or MVP Outcome:** Future foundation and deployed agents can tell where doctrine lives, how feedback becomes durable work, when skills/tools/guardrails are the right evolution surface, and how release/publication rules stay visible without confusing source-only binary asset mechanics with target project release discipline.
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
  - `docs/tickets/in-progress/` contains `MH-048`.
  - `docs/tickets/backlog/` contains `MH-049`, `MH-050`, `T-001`, and
    `T-004` through `T-005`.
  - `docs/tickets/done/` contains `MH-001` through `MH-047`, `T-002`, and
    `T-003`.
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
  `v0.41.10`, `v0.41.11`, and `v0.41.12` release notes and tags were pushed on
  2026-05-19, but CI and Release workflow jobs were not started because GitHub
  reported recent account payment failure or a spending-limit increase
  requirement.
  Notes-only GitHub Releases for `v0.36.4` through `v0.41.12` were created from
  the generated changelog entries on 2026-05-19 so the Releases page is no
  longer stale at `v0.36.3`. `mars-harness release verify-assets --version
  v0.41.12` is still blocked because the `v0.41.12` release is missing
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt` until the workflow can publish assets or a release backfill is
  run after the billing blocker is cleared.
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

1. **Foundation/deployed doctrine architecture**: Complete the ticket-backed
   mirroring and drift-review slice that follows the accepted source
   architecture doc, then return to the Mars parity queue.
   The operator explicitly selected this architecture work on 2026-05-19
   because the recursive improvement loop is changing how the foundation and
   deployed operating models should be understood.
2. **Mars parity execution**: Continue deterministic remediation (`MH-048`,
   claimed under `docs/tickets/in-progress/` on 2026-05-19), then the dogfood
   matrix (`MH-049`). Use [OpenHarness comparator](../../references/openharness-comparator.md)
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
| F-001-S015 | Passing | [foundation-deployed-harness-architecture.md](../../design-docs/foundation-deployed-harness-architecture.md) records the foundation/deployed doctrine boundary, feedback routing, tool/skill/runtime split, generated-target implications, and doctrine-maintenance duties. |
| F-004-S007 | Passing | `go test ./internal/scanner -run TestInit_success` verifies generated targets receive the foundation/deployed route and AD-139 core doctrine without source binary asset names. |
| F-012-S006 | Planned | Skill/tool evaluation ticket will decide whether the recursive improvement loop should become a universal skill, foundation skill, deployed skill pattern, or remain design doctrine. |
| F-012-S007 | Passing | Generated target knowledge routes and mirrored harness docs carry the reusable feedback and improvement-loop doctrine after the AD-139 source doc. |
| F-009-S013 | Passing | `go test ./internal/docsconsistency ./internal/docsync` and `gh release view v0.41.12 --repo greaveselliott/mars-harness` cover the release-object gate and notes-only fallback. `mars-harness release verify-assets --version v0.41.12` records the separate missing-asset blocker. |

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

- `MH-048`: deterministic remediation recipes are in progress. The first
  completed slices add the remediation registry, applicability planner, and
  trace-linked score evidence in `serve`; as of 2026-05-19, the generated-docs
  auto-safe recipe executes through `scanner.Upgrade`, and `doctor --repo`
  reports manifest/generated-docs recipe IDs before runtime. The next slice
  should decide whether MH-048 is complete or needs another permanent check.
- `T-002`: foundation/deployed architecture source doc is done and should be
  used as the input for mirroring and drift review.
- `T-003`: generated target mirroring is done and should be used as input for
  the drift review.
- `T-004` through `T-005`: foundation/deployed architecture follow-up ticket
  set. Complete in dependency order, then refresh this plan back to Mars parity
  work.
