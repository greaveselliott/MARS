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
4. Write a target-local case contract at
   `docs/validation/agent-smoke/current-case.md` from the matrix recipe. The
   trigger context names this path so live agents can inspect the bounded smoke
   contract without trying to read the foundation matrix from inside the
   generated target repo.
5. Seed `.harness/learnings.yaml` before the role starts so runtime hygiene
   does not create a new untracked learning file as the first live-agent
   action.
6. Patch the generated manifest role's `max_turns` through `file_write` so
   smoke execution is bounded. Live runs default to 32 turns; operators may
   lower this for diagnostic fixture pacing, but two-turn live role sweeps are
   not valid evidence for ticket-bearing roles.
7. Run fixture assertions before execution.
8. Execute the selected role through `serve.Executor.Execute` with the
   generated repo as `RepoID`, the per-case DB as the executor DB, isolated
   trust/org-state/trace stores, the role manifest allowlist, structured
   trigger context, and a per-role log file.
9. Read the terminal `job_disposition_record` from org-state.
10. Run fixture assertions again after execution so agent-created forbidden
   mutations fail the case.
11. Write `result.json`, `manifest.json`, and any requested Markdown report.

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
forbidden mutations, target-local case contract, generation provenance, cleanup
status, and failure class to match the case contract. Exit code alone never
counts as a pass.

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
| Live agents waste turns looking for the foundation matrix inside generated targets | Each generated target now contains `docs/validation/agent-smoke/current-case.md`, and the trigger routes agents to that target-local contract. |

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
- **2026-06-14 — Live roles need a target-local case contract:** Retained
  local-model failures showed agents correctly operating inside the generated
  target, then trying to inspect `docs/validation/agent-smoke/matrix.yaml`,
  which exists only in the foundation repo. The generator now writes
  `docs/validation/agent-smoke/current-case.md` into every ephemeral target and
  makes that file a fixture assertion so live role evidence does not depend on
  unreachable source-repo context.
- **2026-06-14 — Fast-suite role contracts must encode the bounded stop:**
  The first patched fast-suite run passed 11 of 14 roles but exposed three
  role-local contract gaps: CTO created the right planning ticket but hit
  terminal disposition with uncommitted ticket state, Dogfood validated static
  HTTP delivery and then looped on no-op waits, and Release Manager treated the
  intentionally remote-less ephemeral target as blocked after local release
  artifacts were ready. The target-local contract now gives those roles bounded
  stop instructions for ticket commit, static HTTP shutdown/evidence, and
  local-release readiness without a GitHub remote.
- **2026-06-14 — Trigger summaries and post-run assertions prevent hidden
  false positives:** A later parallel fast-suite rerun showed that some roles
  can act from trigger/system context without opening the target-local contract.
  The runner now includes a compact role-specific contract summary directly in
  the structured trigger and marks validation jobs with payload mode
  `agent_smoke`, which gives the executor a smoke-specific user message that
  tells the role to follow the trigger contract before broad discovery. It also
  enforces role-owned post-run evidence for sensitive cases: Pipeline-Fixer
  failure cases must update `.mars/checks/latest.json` to `passed`, and Release
  Manager ready cases must leave a local `v<VERSION>` tag.
- **2026-06-14 — Tool-boundary normalization must preserve truth:** A live CTO
  fast-suite case reached the correct bounded action but passed
  `bdd_scenarios` as a quoted JSON-array string to `ticket_create`, causing a
  parser failure and a false behavioral failure. `ticket_create` now accepts
  strict arrays and quoted JSON-array strings for list fields, then writes
  canonical ticket frontmatter. This is not fake validation evidence: non-array
  strings still fail, and the generated ticket remains the durable artifact
  that the post-run case assertions inspect.
- **2026-06-14 — Static-web review smoke needs deterministic command shape:**
  Parallel live fast-suite evidence showed Dogfood and QA were both derailed by
  avoidable static-server ambiguity: Dogfood proved HTTP 200, then looped after
  cleanup reported an already-gone server PID, while QA tried `python` instead
  of `python3` and then treated a localhost curl failure as product rework. The
  target-local contract now gives static-web roles a deterministic per-case
  port, `python3 -m http.server`, a single HTTP 200 probe, and bounded cleanup
  guidance so review roles validate the product instead of the operator's
  default Python alias or an implicit port.
- **2026-06-14 — Ready browser-game fixtures must be product-smokeable:**
  Full-matrix live evidence showed `dogfood-browser-game-ready` could not pass
  honestly because the generated Phaser/Vite target lacked `index.html`, so
  `npm run build` failed and Dogfood correctly could not approve without a
  browser-product smoke. Browser-game fixtures now include a Vite entry page
  and a `parent: 'game'` Phaser entrypoint, and the target-local contract names
  the source/runtime smoke phrase recognized by tool policy:
  `browser smoke: Phaser canvas #game new Phaser.Game`.
- **2026-06-14 — React ready fixtures need real Vite mounting and hydrated
  dependencies:** Full-matrix evidence then showed `dogfood-react-web-ready`
  had the same class of fixture problem: missing Vite `index.html`, a
  non-mounting `src/main.jsx`, and an empty placeholder lockfile that pushed
  the model toward failing `npm ci`. React fixtures now include the Vite entry
  page, `createRoot` mounting, and a `#game` UI marker for a source/runtime
  browser-product smoke. `dependency_sync` also accepts quoted boolean strings
  for `frozen` so local-model payload drift does not block the intended
  unfrozen dependency hydration path.
- **2026-06-14 — Held-out negative cases need negative contracts:** Several
  held-out cases intentionally expect `blocked`, `changes_requested`, or
  `no_work`, but early generic contracts pushed roles toward the ready-case
  success path. Role contracts now branch on expected disposition for blocked
  Pipeline Fixer, blocked Orchestrator, blocked Release Manager, negative QA,
  and defect Dogfood cases so the runner tests truthful stops instead of
  rewarding forced green evidence.
- **2026-06-14 — Defect Dogfood cases need hard seeded evidence:** A live
  full-matrix run reached `dogfood-go-cli-defect-heldout`, where unit tests
  passed and the role looped because the fixture only described a defect in
  prompt text. Defect held-outs now generate a target-local
  `docs/reports/dogfood/seeded-defect.md` marker and failed
  `.mars/checks/latest.json` state. Dogfood contracts and trigger summaries
  explicitly say that passing unit tests do not clear this held-out product
  evidence gap, preventing prompt-only defects from becoming false positives.
- **2026-06-14 — Green reports must verify routing and role-owned outputs:**
  A follow-up audit found several ways the runner could report green while only
  proving that fixture directories existed: dispositions were not checked
  against `next_need`/suggested-role routing, current-role QA/Security/Dogfood
  reports could be preseeded, Engineer and Janitor ticket lifecycle was only
  checked at the directory level, and Release Manager could tag the seeded
  `0.0.0` version. The runner now separates fixture-before checks from
  post-role checks, skips current-role output artifacts only before execution,
  and then requires the role-owned report/ticket/release/tag evidence after
  execution. Routing mismatches are classified as dispatch-context failures.
- **2026-06-14 — QA needs bounded report-write authority:** The stricter
  negative-QA live case showed QA could read the seeded evidence gap and record
  `changes_requested`, but could not create the required QA report because the
  generated QA manifest lacked `file_write` and git commit tools. Generated QA
  now has `file_write`, `git_commit`, and `git_push` for bounded review-report
  evidence, while shell policy remains validation-only and product mutation
  remains outside QA ownership.
- **2026-06-15 — Full parallel matrix passed after contract fixes:** The
  validated completion run used
  `mars-harness validation agent-smoke --suite full --parallel 2 --timeout 45m`
  with the local model router and no endpoint override. It selected all `74`
  cases and reported `74 passed`, `0 failed`, `74 selected`: `70` deployable
  role cases executed live through the server job path, while the `4`
  `foundation-maintainer` cases remained explicitly source-only to preserve the
  source/deployed role boundary. The final fixes were contract-level rather
  than fake-endpoint workarounds: stale Janitor tickets now seed existing
  evidence before lifecycle cleanup, React Pipeline Fixer cases route to
  dependency hydration/build/source smoke instead of Go validation, and Go
  Dogfood ready cases name `go test ./...`, report commit, and approval as the
  bounded user smoke. The durable evidence is recorded in
  `docs/validation/reports/2026-06-15-agent-smoke-full-matrix.md`.
