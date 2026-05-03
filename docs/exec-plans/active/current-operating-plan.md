# Current Operating Plan

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** MH-034, MH-035, MH-037, MH-031, MH-030, MH-038, MH-039, MH-040, MH-041, MH-042, MH-043, MH-044, MH-045, MH-046, MH-047, MH-048, MH-049
**Goals:** G-001, G-002, G-003, G-004
**BDD Feature:** F-001, F-002
**Hypothesis:** Zero-config CLI availability removes the first-run `Unknown command: mars-harness` failure mode while the BDD-led operating loop keeps completion evidence-based.
**Success Evidence:** `make install`, `mars-harness setup`, and `mars-harness update tool` configure supported user shell profiles idempotently, and F-001 remains passing.
**Falsification Evidence:** Fish, Zsh, Bash, or POSIX shell users still need manual PATH edits after install/update/setup.
**Scenario Schedule:** F-002-S001, F-002-S002, F-002-S003, F-002-S004, F-002-S005
**Current Failing Scenario:** None for F-002; the next active-plan refresh should promote the highest-value backlog scenario group.
**Walking Skeleton Slice:** Use one shellpath package across source install, first-run setup, update-tool reinstall, and an explicit `mars-harness path setup` command.
**Learning Or MVP Outcome:** Users can install or update once and run `mars-harness` from new terminals across common shells without hand-written shell setup.
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
- BDD feature contracts live in `docs/features/`; the current operating-model feature is `F-001` and the current install/setup feature is `F-002`.
- Ticket state:
  - `docs/tickets/in-progress/` is empty.
  - `docs/tickets/backlog/` contains `MH-044` through `MH-049`.
  - `docs/tickets/done/` contains `MH-001` through `MH-043`.
- Exec-plan state:
  - `docs/exec-plans/active/` contains exactly one active plan: this file.
  - `docs/exec-plans/backlog/` contains prioritized waiting plans with dependencies and blockers.
  - `docs/exec-plans/superseded/` contains historical plans that must not drive current work.
- GitHub release notes are published for semantic versions generated from `VERSION`.
- Release binary assets are published for `v0.14.5`; `MH-031` is done.
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

1. **Mars parity execution**: work through the tickets materialized on 2026-05-03 for
   operating model (`MH-042`), role registry (`MH-043`),
   conversation-as-system-record (`MH-044`), intervention debt (`MH-045`),
   active-ticket drain (`MH-046`), orchestrator recovery (`MH-047`),
   deterministic remediation (`MH-048`), and dogfood matrix (`MH-049`).

## Scenario Schedule

| Scenario | Status | Evidence |
| --- | --- | --- |
| F-001-S001 | Passing | `docs/goals/active.md`, `docs/features/F-001-delivery-operating-model.md`, and this active plan are linked. |
| F-001-S002 | Passing | Engineer ticket gate validates feature ticket evidence before done. |
| F-001-S003 | Passing | `mars-harness init` emits goals, feature contracts, AD-074, role guidance, ticket metadata, exec-plan metadata, quality guidance, and knowledge routes. |
| F-001-S004 | Passing | `update check` and `doctor --repo` report operating-model drift without overwriting user-owned docs. |
| F-001-S005 | Passing | Ticket docs, quality score, and release-note generation distinguish shipped feature scenarios from enabler work when commits reference tickets. |
| F-001-S006 | Passing | Telemetry proposals can create or update active goals/observations with dedupe evidence. |
| F-002-S001 | Passing | `make install` invokes `mars-harness path setup --install-dir <dir>` after `go install`. |
| F-002-S002 | Passing | `mars-harness setup` includes a `configure-shell-path` step. |
| F-002-S003 | Passing | `mars-harness update tool` runs shell PATH setup after reinstalling the binary. |
| F-002-S004 | Passing | Shell profile updates are idempotent and covered by tests. |
| F-002-S005 | Passing | Unsupported shells return explicit manual remediation without writing profile files. |

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

- `MH-044`: conversation-as-system-record guidance is the next backlog ticket.
- Next active-plan refresh: promote the highest-value Mars parity scenario group.
