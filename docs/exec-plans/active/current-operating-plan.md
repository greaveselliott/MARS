# Current Operating Plan

**Status:** Active
**Created:** 2026-05-02
**Owner:** Mars Harness maintainers
**Source:** Exec-plan review and repository state audit on 2026-05-02

## Purpose

This is the current execution map for Mars Harness. It exists because the
original master plan and delivery schedule are now historical baseline
documents: useful for lineage, but stale as a status source.

Future agents should use this file, the ticket tree, and the Mars parity plan
to decide what to do next.

## Current Truth

- Current source version is recorded in `VERSION`.
- Current branch: `main`
- Ticket state:
  - `docs/tickets/in-progress/` has no active tickets.
  - `docs/tickets/backlog/` contains `MH-030`, `MH-031`, `MH-034`, `MH-035`, and `MH-037`.
  - `docs/tickets/done/` contains `MH-001` through `MH-029` plus `MH-032`, `MH-033`, and `MH-036`.
- GitHub release notes are published for semantic versions generated from `VERSION`.
- Release binary assets are still missing and tracked by `MH-031`.
- Model evaluation, Ollama catalog support, and model swaps are still tracked by
  `MH-030`.

## Plan State

| Plan | State | How to use it |
| --- | --- | --- |
| `current-operating-plan.md` | Active | Use this file as the top-level execution map. |
| `mars-parity-supersession-plan.md` | Active | Strategic parity/supersession roadmap. Workstreams remain mostly open. |
| `model-evaluation-refresh-plan.md` | Active | Tactical plan for `MH-030`. |
| `master-execution-plan.md` | Superseded pending reconciliation | Historical baseline. Do not use its checkbox status as truth. |
| `delivery-schedule.md` | Superseded pending reconciliation | Historical milestone schedule. Do not use its checkbox status as truth. |

## Current Priority Order

1. **Plan hygiene and stale-plan prevention**: reconcile superseded plans and add
   mechanical active-plan checks so future agents cannot mistake stale plans for
   current truth.
2. **Mars parity ticket materialization**: turn the first ten work items in the
   Mars parity supersession plan into normal backlog tickets.
3. **Quality score export (`MH-037`)**: replace the seeded scorecard with a
   deterministic export from live scores, telemetry, tickets, checks, and
   dogfood evidence.
4. **Release asset contract (`MH-031`)**: publish checksum-verified binaries so
   install and `update tool` no longer require Go or a source checkout.
5. **Model evaluation and swap workflow (`MH-030`)**: complete Ollama catalog
   support, explicit role/tier swaps, persistent reports, and promotion checks.
6. **Mars parity execution**: work through operating model, role registry,
   conversation-as-system-record, intervention debt, active-ticket drain,
   quality exports, orchestrator recovery, deterministic remediation, and
   dogfood matrix workstreams.

## Quality State

Latest checks observed during the review:

- `go test ./...` passes.
- `go test -cover ./...` passes after making update-check fixtures
  release-agnostic.
- Coverage remains below the stated 70 percent target in several packages,
  including `internal/inference`, `internal/setup`, `internal/ui`,
  `internal/serve`, `internal/hardware`, and command entrypoint code.
- `golangci-lint` was not installed in the local environment during the latest
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

- `MH-033`: this plan hygiene reconciliation.
- `MH-034`: mechanical active-plan hygiene checks in doctor and CI.
- `MH-035`: materialize Mars parity workstreams as backlog tickets.
