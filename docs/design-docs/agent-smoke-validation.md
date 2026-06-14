# Agent Smoke Validation

**Status:** Accepted
**Date:** 2026-06-14
**Author:** foundation-maintainer
**Related:** AD-084, AD-090, AD-138, AD-284, AD-285, AD-291, AD-293, AD-294

## Context

Full clean-project `mars-harness start` sweeps are still the broadest evidence
for lifecycle claims, but they are slow and sequential. They also naturally
optimize early roles because CEO, strategy, COO, and CTO behavior is observed
more often than later roles. Foundation maintainers need a faster lane that
can exercise every role at the lifecycle checkpoint where that role normally
acts, across multiple project shapes, without maintaining static target repos.

Compartmentalised Agent Smoke testing fills that gap. The lane must validate
live per-agent execution, not merely target generation. Fixture generation is a
precondition; the validation claim comes from executing the selected role
through the same server job path used by autonomous jobs.

## Key Design Decisions

### AD-295: Agent Smoke Executes Selected Roles Live And In Parallel

`mars-harness validation agent-smoke` is a source-only role-local validation
lane. Each selected matrix case creates a fresh ephemeral target repo, isolated
SQLite database, logs, trace directory, result file, and manifest under
`../demo/validation-runs/agent-smoke/` by default. Successful run directories
are discarded after report generation unless `--keep-runs` is set; failed run
directories are retained unless `--discard-failed` is set.

The primary behavior is **live role execution**:

1. Select cases from `docs/validation/agent-smoke/matrix.yaml`.
2. Initialize a one-use git repo under `target/`.
3. Seed the target through foundation surfaces only: target harness scaffold,
   `file_write`, `ticket_create`, `record_decision`, `git_status`,
   `git_commit`, `workspace_hygiene`, and related built-in tools.
4. Patch the generated manifest role's `max_turns` through `file_write` so
   smoke execution is bounded.
5. Run fixture assertions before execution.
6. Execute the selected role through `serve.Executor.Execute` with the
   generated repo as `RepoID`, the per-case DB as the executor DB, isolated
   trust/org-state/trace stores, the role manifest allowlist, structured
   trigger context, and a per-role log file.
7. Read the terminal `job_disposition_record` from org-state.
8. Run fixture assertions again after execution so agent-created forbidden
   mutations fail the case.
9. Write `result.json`, `manifest.json`, and any requested Markdown report.

The runner constructs one shared inference router for a batch. Local-model
agent-smoke runs default to `--single-server --single-server-tier coding`,
which forces every role and manifest model hint onto one local llama-server
process. `--parallel N` then runs up to N selected cases concurrently, each
with its own target repo, DB, logs, and trace state, while the single server's
`--parallel` slot count is raised to at least N. The router scales the
server's total context by that slot count so each concurrent request keeps the
configured tier window instead of receiving a divided slice of context.
Operators may pass `--single-server=false` to diagnose tier-routing behavior,
but the primary Compartmentalised Agent Smoke claim is one shared local server
with parallel case execution.

Follow-on dispatch is intentionally suppressed. The runner directly invokes
the executor and does not call the server job-completion callback that would
enqueue the Orchestrator or chained roles. Each case records `would_dispatch`
and terminal disposition fields so routing expectations remain visible without
turning a compartmentalized smoke case into a full lifecycle sweep.

`--fixture-only` is a diagnostic mode for generator and lint failures. It is
not valid evidence that any agent role executed.

### Source-Only Foundation Maintainer Exception

`foundation-maintainer` is a source-only role for this `mars-harness` repo and
must not be mirrored into deployed target manifests. Agent-smoke therefore
reports foundation-maintainer cases with `execution_mode: source-only` rather
than mutating generated target manifests to include the role. This preserves
AD-274's source/deployed role boundary.

### Project Shape Coverage

The checked-in matrix cycles the same role behavior across multiple generated
project shapes so the factory does not overfit one project style:

- static web
- React/package-managed web
- Phaser browser game
- vanilla canvas game
- Go API
- Go CLI
- Go library
- docs site
- existing-maintenance state with stale tickets, failed checks, or doc drift

Each matrix case defines required artifacts, forbidden mutations, expected
terminal disposition, and the would-be next role. The fast suite rotates one
case per role via `--cycle`; default and held-out suites broaden project shape
coverage; full runs every checked-in case.

### AD-296: Fake Endpoints Are Not Validation Evidence

Agent-smoke validation reports must never count fake, stub, mock, canned, or
scripted LLM endpoints as evidence that an agent role works. They can only
prove that runner plumbing is wired: target generation, executor invocation,
parallel isolation, result writing, and cleanup. They do not prove reasoning,
tool choice, role judgement, project-shape behavior, or live model reliability.

`--model-endpoint` remains available for real OpenAI-compatible providers such
as a local llama.cpp server, a local proxy to an installed model, or an
operator-approved real model endpoint. Using that flag shifts provenance to an
operator-supplied endpoint; the report must name that as an endpoint override.
If the endpoint is fake or scripted, the run is evidence-only plumbing and must
be excluded from validation pass claims.

Automated tests may continue to use fake LLM servers to keep CI deterministic,
but test names, assertions, and documentation must describe that evidence as
deterministic executor-path coverage. They must not be cited as live
agent-smoke validation.

### Validation Evidence Contract

Agent-smoke evidence has two levels:

- **Automated deterministic evidence:** fake-LLM integration tests prove only
  that selected cases execute through the server job path, write terminal
  dispositions, run in parallel against isolated repos/DBs, and clean up
  successful ephemeral runs.
- **Live model evidence:** operator-triggered runs without `--fixture-only`
  and backed by a real model endpoint prove model behavior for the selected
  role/case matrix. Primary agent-smoke validation reports must record the
  inference topology, including `single_server`, `single_server_tier`, and
  server parallel slots, so reviewers can distinguish the intended
  single-server parallel lane from tiered-router diagnostics. These runs must
  be recorded under `docs/validation/reports/` when used for a source-change
  claim.

A passing case requires the expected terminal disposition, required artifacts,
forbidden mutations, generation provenance, cleanup status, and failure class
to match the case contract. Exit code alone never counts as a pass.

## Consequences

- Agents and maintainers can exercise all manifest roles in parallel at their
  natural lifecycle checkpoints instead of waiting for a full sequential
  project sweep.
- Full clean-project sweeps remain mandatory for broad lifecycle,
  orchestration, release, and cross-role handoff claims.
- Validation reports must not blur fake-LLM executor-path proof with real model
  role-quality proof.
- Fake, stub, mock, canned, or scripted LLM endpoints are excluded from live
  validation claims even when the runner reports `execution_mode: live`.
- Retained failed run directories are debugging artifacts, not reusable
  fixtures. Checked-in artifacts remain recipes, templates, matrix definitions,
  docs, and reports only.

## Failure Modes And Mitigations

| Failure mode | Mitigation |
| --- | --- |
| Smoke lane silently becomes fixture generation only | Live execution is default; `result.json` and Markdown reports record `execution_mode`; `--fixture-only` is explicitly diagnostic only. |
| Agent smoke overfits one project type | Matrix spans API, web, game, CLI, library, docs-site, and maintenance cases, with `--cycle` and held-out suites. |
| Parallel cases contaminate each other | Every case owns a target repo, DB, log file, and trace path; only the inference router is shared. |
| Parallel validation silently starts one model server per tier | Local-model agent-smoke defaults to `--single-server`; reports record single-server topology and server parallel slots. |
| Follow-on dispatch turns smoke into an unbounded lifecycle run | Runner calls `serve.Executor.Execute` directly and skips job-completion routing. |
| Source-only roles leak into deployed manifests | `foundation-maintainer` is reported as source-only and is never added to target manifests. |
| Fake-LLM tests are mistaken for real model validation | AD-296 bans fake endpoints as validation evidence; reports name endpoint overrides and classify fake-backed runs as evidence-only plumbing. |

## Discoveries

- **2026-06-14 — Initial implementation under-documented the primary
  purpose:** The first implementation created the matrix and ephemeral
  generation lane, but the design and validation evidence did not clearly state
  that live per-agent execution in parallel is the main purpose. AD-295 fixes
  that by naming the execution path, evidence levels, and source-only role
  exception directly.
- **2026-06-14 — Fake endpoint evidence created false positives:** A full
  matrix plumbing attempt was run through a scripted OpenAI-compatible endpoint
  and produced report rows that looked like live role success. That evidence is
  invalid for role-behavior claims. AD-296 makes fake/model-stub endpoints
  test-only and requires validation reports to distinguish endpoint overrides
  from real local model evidence.
