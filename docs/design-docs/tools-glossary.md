# Tools Glossary

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** Mars Harness maintainers
**Mirrors:** Generated target `docs/design-docs/tools-glossary.md`

## Purpose

This glossary is first-class mirrored tool context. It tells LLM chats which
tools exist, when to use them, and which guardrails shape their use in both the
foundation harness and deployed harnesses.

Read this file whenever a task involves tool availability, tool selection,
tool allowlists, tool policy, or CLI operation. Keep it current when built-in
tools are added, removed, renamed, or materially change behavior.

## Availability Rules

- Tools are available only when registered in the built-in registry and included
  in the current role allowlist.
- Universal mirrored tools are part of the same built-in registry in the
  foundation harness and every deployed harness initialized or upgraded by
  `mars-harness`.
- Universal tools must be readily discoverable through `mars-harness tools list`
  and executable through `mars-harness tools run <name>` when an operator or
  external LLM shell is outside an active agent run.
- Universal tools must also be exposed through the standard MCP stdio surface:
  `mars-harness mcp serve --repo <path>`. This is the preferred integration
  point for MCP-compatible clients and local harness agents because it makes
  Mars Harness tools native tools instead of shell conventions.
- The universal tool surface is model-provider agnostic. Deployed harnesses use
  local models by default, and MCP/tool transport must not assume frontier cloud
  model access.
- During active agent runs, universal tools are still trust-gated and must appear
  in the role's allowlist; outside an agent run, `mars-harness tools run` uses
  the same executor, repo-root resolution, trust policy, and JSON argument path.
- Mirrored tools are valid in both the foundation harness and deployed harnesses.
- Mutating tools are blocked at observer trust.
- Prefer purpose-built tools over `shell_exec` when a deterministic tool exists.
- Prefer structured argv over shell strings unless shell features are required.

## Mirrored Built-In Tools

| Tool | Use When | Notes |
| --- | --- | --- |
| `file_read` | Read a known file path from the repository. | Non-mutating. Use before editing or reviewing code. |
| `file_write` | Create or replace a file under the repository root. | Mutating. Guardrails and secret scanning apply. To reduce local-model tool-call waste, `file_write` repairs the common malformed payload where `<parameter=content>` is embedded in the path and the content field is empty. New ticket markdown is blocked; use `ticket_create`. Done feature-ticket writes are blocked when required BDD evidence is empty or the same ticket still exists in another lifecycle directory. Engineer cannot populate in-progress ticket `evidence_links` or `verified_by` before the same job records successful validation. New `docs/features/F-NNN*.md` writes are blocked when another contract with the same `F-NNN` ID already exists, feature files cannot contain duplicate scenario heading IDs, and scenario heading feature IDs must match the contract file path; duplicate and mismatch errors name heading line numbers and clarify that Scenario Schedule list references are allowed. New repo-root validation scripts such as `validate.sh` are blocked because scratch validation should use existing tests, direct build/run/curl evidence, or intentional durable tests. COO may only write planning artifacts. CTO may only write bounded technical planning artifacts under `docs/design-docs/`, `docs/reports/strategy/`, or `docs/goals/observations.md`; implementation, package/module, README usage, source, test, build, config, and root product-file edits belong to ticket-backed Engineer delivery. Engineer product writes require a claimed in-progress product ticket when ordinary backlog work exists, and review rework must reopen a done or in-review product ticket before product/source edits. |
| `file_search` | Find files by glob-style path patterns. | Non-mutating. Use for inventory before broad reads. |
| `grep` | Search file contents with a regex. | Non-mutating. Use to locate symbols, text, or repeated patterns. |
| `shell_exec` | Run a subprocess when no purpose-built tool fits. | Mutating. Prefer argv for a single executable and arguments; use `shell_command` when shell parsing, redirection, pipes, control operators, or shell builtins are required. Direct `mars-harness` binary invocations are blocked; use `mars_harness_cli` so agents resolve the active harness executable instead of stale PATH installs. To reduce local-model tool-call waste, `shell_exec` normalizes simple malformed argv shapes such as a JSON-encoded argv string, a Python-style quoted list string, a one-item simple command string, or the narrow validation-only argv shape `cd <dir> && <test-or-build command>` before both policy checks and execution, while still rejecting arbitrary shell syntax in argv mode; literal newlines inside an argv argument are allowed because argv does not invoke shell parsing. Engineer runs with an ordinary backlog product ticket and no in-progress ticket must use `shell_exec` to claim that ticket with `git mv ... docs/tickets/in-progress/` before any other shell command, including read-only discovery or no-op placeholders. Engineer review rework with only done or in-review product tickets must reopen the named ticket into `docs/tickets/in-progress/` before validation shell commands or product mutation. After Engineer has successful validation, a clean implementation commit, and a product ticket remains in progress, additional exploratory `shell_exec` calls are blocked; the next required tools are `file_read` and `file_write` on the in-progress ticket to populate evidence, and the allowed shell path is only the ticket lifecycle `git mv ... docs/tickets/done/` after evidence is updated, plus a fresh same-session `<validation-root>` binary execution that directly proves the review request. Direct runtime commands that execute ticket behavior, such as `go run`, `cargo run`, `dotnet run`, language interpreter entrypoints, package start scripts, or bounded smoke probes, count as validation evidence after successful exit. Use `expected_exit_code` only for intentional non-zero error-path validation probes; matching non-zero exits become expected negative-path evidence, while unexpected validation failures block review approval. Engineer cannot move, write, or commit product tickets to `docs/tickets/done/` while the same job has an unrepaired unexpected runtime validation failure; after Engineer observes an unexpected runtime failure, runtime probes are blocked until an implementation `file_write` occurs, and only a later successful run of the exact failed command repairs the blocker. Engineer may correct an obvious no-argument/missing-argument runtime probe by rerunning that exact command once with matching `expected_exit_code`, but cannot retroactively add `expected_exit_code` to clear a failed positive acceptance path. QA/Security retain the one-time exact-command `expected_exit_code` correction for review-procedure mistakes. Engineer, QA, and Security Go build/test procedure mistakes such as `go build cmd/<name>` without `./` or root `.` builds in `cmd/*` CLI repos are recorded separately as validation-procedure failures, allowing a corrected validation command without falsely routing implementation rework. After QA or Security observes any real failing build, failing test, or unexpected runtime validation command, further `shell_exec` validation is blocked and the next action is `job_disposition_record` with `status: changes_requested` and the exact failing command/output; the only runtime exception is one immediate rerun of the exact same command with matching `expected_exit_code` when the first command was an expected-negative probe. Empty argv and single `:` no-op calls do not execute a process; they fail the tool call with guidance to stop any tracked background PID, commit completed work, and record `job_disposition_record` instead of silently spending turns. Engineer no-op calls after successful validation and dirty ticket work immediately redirect to tracked PID cleanup, `git_status`, evidence update, `git_commit`, ticket lifecycle completion, push, and terminal disposition. `shell_command` rejects the shell background operator `&`; use the tool's `background:true` flag for long-running dev servers and run readiness probes as separate calls. Likely server/watch commands such as `go run` against a source file containing HTTP server markers, `npm start`, `npm run dev`, `python -m http.server`, `uvicorn`, `gunicorn`, `rails`, `vite`, and `next` are blocked in foreground mode and must use `background:true`. Bare port tokens such as `:8080` are rejected as malformed commands and redirected toward a real server command plus `curl http://localhost:8080/health` style probes. External `timeout`/`gtimeout` commands are rejected because the harness owns per-call `timeout_seconds` and managed background cleanup; those policy blocks classify as guardrail telemetry rather than retryable tool timeouts. Background startup exits are reported as tool errors with initial output instead of a misleading started-success result. Background startup snapshots are synchronized while the process keeps emitting logs, so long-running server capture is safe under race-detected CI. Background startup output reminds roles to stop the tracked PID after probes and not use empty argv or `:` as wait commands. Background cleanup terminates the tracked process, its process group, and known descendants so wrapper commands such as `go run` do not leave child servers occupying ports; `kill <tracked-background-pid>` is intercepted and applies the same process-tree cleanup during the job. `go build` without `-o`, `go build -o <path>` inside the target repo, and untracked external temp output paths without a `-validation` suffix are blocked before execution; use `go test ./...` for compile validation or `go build -o /tmp/<project>-validation <entrypoint>` for runnable validation evidence. Temp validation binaries such as `<validation-root>` are trusted only after the same role session successfully built that exact path; stale temp binaries are blocked before execution, and cleanup such as `rm <validation-root>` is allowed after ticket completion without forcing product rework. `file_write` rejects source/test files under source roots, including root-level `main.go` or `index.html`, unless the first write includes valid top-of-file `MarsDocSync` docs metadata that points at existing docs. `git remote add`, `git remote set-url`, and related remote mutation commands are blocked so agents cannot invent or rewrite repository remotes during release work. Ticket moves into `docs/tickets/done/` are blocked until required evidence fields are populated: feature tickets require BDD scenario evidence, while enabler/remediation tickets with `end_to_end_evidence: required` can close with evidence links and verifier metadata. Ticket lifecycle shell parsing preserves source path case while matching `git`, `mv`, and `cp` command names case-insensitively so uppercase ticket IDs are enforced on Linux and macOS. Copying tickets into `done/` is blocked so lifecycle completion uses `git mv`. Engineer product mutation through shell commands requires a claimed in-progress product ticket; `git mv` from backlog, in-review, or done to in-progress remains allowed for claim/rework reopening. |
| `workspace_hygiene` | Audit generated dependency/build churn, ignore policy, tracked generated paths, and deletion risk before agent work or dependency sync. | Non-mutating. Returns `status`, `blocking`, `auto_repairable`, `findings`, `recipe_id`, `message`, and `next_action`; `serve` can auto-commit safe `.gitignore`-only repairs before model loading. `paths` uses the same list-string normalization as other path-filtered tools. |
| `github_auth_check` | Check private Mars Harness GitHub Release auth readiness. | Non-mutating. Returns `status`, `auth_source`, `repo_access`, `release_access`, `message`, and `next_action` without revealing token values. |
| `dependency_sync` | Run package-manager install or fetch through deterministic workspace hygiene preflight and postflight. | Mutating. Performs the same safe `.gitignore`-only repair when needed. Use instead of raw `npm install`, `npm ci`, `pnpm install`, `yarn install`, `bun install`, `go get`, `go mod download`, `cargo fetch`, `pip install`, `bundle install`, or `composer install`. Engineer dependency sync requires a claimed in-progress product ticket when backlog product work exists. |
| `mars_harness_cli` | Read exhaustive CLI reference or run `mars-harness` commands with structured argv. | Mutating. Use for setup, init, upgrade, doctor, scan, run, start/serve, release, scores, trust, models, and update workflows. To reduce local-model tool-call waste, `args` accepts structured arrays plus JSON-encoded array strings, Python-style quoted list strings, and simple single command strings. The resolver prefers `MARS_HARNESS_CLI_BIN`, then the active harness executable, then `PATH`, and stale binaries produce actionable update guidance. When CLI commands or flags change, sync the reference, repo-shortcut map, skills, and generated doctrine per [cli-tool-skill-sync.md](cli-tool-skill-sync.md). |
| `record_decision` | Persist durable decisions, trade-offs, and reusable learnings. | Mutating. Use when the reasoning should survive the chat. |
| `ticket_create` | Create or update deduped markdown tickets. | Mutating. Use instead of hand-writing ticket files. Feature tickets require `bdd_scenarios` as a real JSON array such as `["F-001-S002"]`, not a quoted list string; parse errors for common list fields name the malformed field and show the correct shape. Failed `ticket_create` calls stay attached to the session until a later successful `ticket_create` clears them. |
| `job_disposition_record` | Record the terminal outcome of a dispatch-mode agent job. | Mutating. Required before dispatch-mode jobs complete. Non-Orchestrator roles must commit repo changes before terminal dispositions that approve, complete, request changes, block, fail, or otherwise hand off work; runtime-only `.harness/learnings.yaml` convention metadata does not block by itself. The recorder accepts strict arrays and simple list-as-string shapes for fields such as `evidence_links`, `work_product_ids`, `blocked_by`, handoff constraints, and feedback evidence so local-model formatting drift does not consume extra turns. Dispatch jobs that reach the model turn budget get one final terminal-tool reminder, allowing a reviewer that just found a failing validation command to record structured `changes_requested` instead of ending as raw `max_turns`; after that budget-edge reminder, only the configured terminal tool may execute. QA and Security jobs also get a terminal-only reminder after successful `file_read` inspection plus clean validation evidence, and the next response must call `job_disposition_record` instead of more tools or prose. Successful dispositions are blocked while the current session has an unresolved failed `ticket_create` or failed non-Engineer direct write under `docs/tickets/`; retry `ticket_create` with valid JSON or record `blocked`, `failed`, or `changes_requested` with the exact blocker. Engineer ticket evidence write failures do not count as ticket-creation debt because their own guardrail routes the role back to validation or evidence repair. Engineer successful dispositions with `ticket_id` require that ticket to live in `docs/tickets/done/` and require no outstanding unexpected runtime validation failure or failing test/build repair lane in the same job; `no_work` can finish without a ticket. QA and Security successful dispositions for a named ticket require successful in-job validation evidence, require a successful test command when test files exist, and are blocked after any failing build, test, or uncorrected unexpected runtime validation command in the same job; intentional non-zero runtime probes must use `shell_exec expected_exit_code` and be paired with positive validation or required tests. Successful Engineer, Pipeline Fixer, QA, Security, Dogfood, Release Manager, and Dependency Manager dispositions run DocSync mechanically and are blocked while `docsync_audit` has findings. |
| `tool_create` | Scaffold a new built-in Go tool and starter test. | Mutating. Follow with implementation, registration, trust policy, tests, and allowlist updates. |
| `persona_create` | Scaffold a repo-local persona manual, role prompt, registry row, and optional manifest role. | Mutating. Use for universal, foundation, or deployed persona proposals; foundation defaults still require adding the canonical Go entry in `internal/personas`. |

Runtime validation stderr note: `shell_exec` counts direct runtime probes as
validation evidence only when they exit successfully without error-shaped
stderr. Probes that exit 0 while emitting conservative failure markers such as
`error:`, `Usage of`, `panic:`, `Traceback`, or `exception` are failed evidence
and require a clean exact rerun.

No-op loop note: Engineer no-op `shell_exec` calls are phase-aware.
Before validation, a claimed product ticket routes back to reading the ticket
and feature contract plus product `file_write` implementation or a blocked
disposition. After validation and dirty ticket or product work, even the first
no-op routes to stopping tracked background validation, evidence update,
commit, lifecycle completion, push, and QA handoff.
For QA and Security, no-op placeholders are never review evidence. After
successful validation, a no-op block routes directly to
`job_disposition_record` with the quality decision; after failed validation, it
routes to structured `changes_requested`. Policy-blocked no-op calls are counted
as no-op failures so loop telemetry can distinguish review dithering from
product behavior failures.

Input-validation correction note: when an intentional no-argument,
missing-required-input, deliberately invalid-input, or surplus-argument runtime
probe exits non-zero with clear required-input, usage, invalid-input, or
surplus-argument output and no crash marker, the session treats it as expected
negative-path validation even when `expected_exit_code` was not supplied. When
the output does not clearly prove a normal input-validation path, the session
stores the exact failed `shell_exec` command and the exact correction with
`expected_exit_code`, usually `1`.
Engineer must run that correction before unrelated edits, commits, pushes,
decisions, dependency sync, CLI mutation, or other work. If that exact
expected-exit correction still fails, the attempted correction unlocks
implementation `file_write` so Engineer can repair the product, while commits,
ticket done-moves, completion dispositions, and unrelated runtime probes stay
blocked until the exact runtime failure is repaired. Positive acceptance
failures still require a clean exact rerun without `expected_exit_code`. When
the same exact runtime command failed multiple times in a job, one later
successful exact rerun clears every unmatched same-command failure count so
stale counters do not keep blocking after the command is repaired. While
Engineer has an outstanding positive runtime acceptance failure, `shell_exec`
stays inside the repair lane: it may rebuild the same stale `<validation-root>`
artifact or rerun the exact failed command after source edits, but unrelated
probes, shell wrappers, tests, ticket moves, and product commits are blocked
until the failure is repaired.
Engineer failing test/build commands have a parallel repair lane: once a test
or build command fails in the same job, runtime probes, unrelated shell
commands, ticket evidence updates, ticket completion, successful disposition,
and product commits are blocked until bounded repair writes occur and the same
validation lane passes. Test failures may be repaired by source, test,
fixture, or package/build config edits followed by any recognized test command;
build failures use the same bounded repair writes followed by a recognized
build command. A narrow shell form, `cd <dir> && <recognized test/build
command>`, is classified by the right-hand validation command for this repair
lane so focused package validation can run without becoming a shell-wrapper
bypass. Fresh Go module bootstrap is a narrow package-config repair: when the
latest failing output proves Go cannot find a main module and `go.mod` is
absent, Engineer may run `go mod init <module>` before rerunning same-lane
validation. Helper scripts, arbitrary chained shell control syntax, runtime
probes, cleanup, ticket moves, and root scratch validation files stay blocked
while the lane is unresolved.
The only cleanup exception is non-recursive `rm` or `unlink` of test-like
files written by the same Engineer job when the latest failure output looks
like duplicate/generated-test conflict such as redeclared symbols, duplicate
definitions, mixed packages, or parse/declaration errors. Assertion failures
and contract mismatches must preserve the test evidence; unmarked tests,
product source files, recursive removal, and ordinary cleanup remain blocked.

External validation artifact freshness note: `<validation-root>` binaries are
source snapshots. Before QA or Security trust an existing temp binary, the
binary must be rebuilt in the same role session. Tool errors name the exact
`shell_exec argv ["go","build","-o","<validation-root>",...]` correction,
preserving the original Go package target such as `.` or `./cmd/<name>`. After
a positive runtime acceptance failure and source edit, the old artifact must be
rebuilt before Engineer reruns it. Direct source execution such as `go run`
validates current files without this rebuild step.

Ticket evidence validation note: Engineer in-progress ticket evidence follows
validation. `file_write` blocks non-empty `evidence_links` or `verified_by`
until the same job has successful validation evidence from a test, build, or
runtime command. Empty placeholders remain writable before validation; concrete
proof belongs after the behavior has been exercised.
| `release_orchestrate` | Plan and preflight the full semantic commit, release notes, push, tag, workflow, and asset verification ritual. | Mutating workflow. Use before driving release state with `mars_harness_cli` and git tools. |
| `github_release_status` | Inspect the release-status workflow and decide whether to wait, rerun, verify, or record a blocker. | Non-mutating. Pairs local tag state with GitHub inspection commands. |
| `architecture_audit` | Check architecture docs against current CLI, generated harness layout, tool registry, and runtime boundaries. | Non-mutating. Use after architecture-affecting changes and before doc reviews. |
| `harness_doctrine_sync` | Check mirrored foundation and deployed harness doctrine for glossary, tools, operating-model, and generated-target consistency. | Non-mutating. Use when changing operating doctrine or mirrored definitions. |
| `docsync_audit` | Audit source files for `MarsDocSync` metadata and associated documentation pointers. | Non-mutating. Use before commits that touch code or when validating the no-stale-docs operating model in [documentation-sync-architecture.md](documentation-sync-architecture.md). Foundation source checkouts enforce expected-doc prefix mappings; deployed target repos require valid metadata and existing docs without forcing foundation-only source-doc references. |
| `git_release_guard` | Check git, tag, version, and release-note invariants around the release flow. | Non-mutating. Use before and after release-note generation. Fails when a version tag exists but does not point at the current release-note commit. |
| `tool_inventory_audit` | Compare registered tools, mutating policy, tools glossary, generated target guidance, and role exposure. | Non-mutating. Use whenever tools are added, removed, renamed, or reclassified. |
| `tool_creation_guard` | Audit whether built-in tool creation followed the governed `tool_create` and `record_decision` path. | Non-mutating. Use when reviewing new tool work or exception handling. |
| `task_trace_summarize` | Summarize a recent work trace and identify repeated manual processes that should become formal tools. | Non-mutating. Use after multi-step work or recurring manual recovery. |
| `git_status` | Inspect repository state. | Non-mutating. Use before commits or risky operations. |
| `git_diff` | Inspect unstaged or staged changes. | Non-mutating. Use before review, commit, and release notes. `paths` uses the same list-string normalization as other path-filtered tools. |
| `git_commit` | Stage files and create a semantic commit. | Mutating. Requires meaningful diff and strict-trunk discipline. `paths` uses the same list-string normalization as other path-filtered tools. Engineer commits that include product files require a visible in-progress product ticket; the final implementation commit may include product files plus an active move from `docs/tickets/in-progress/` to `docs/tickets/done/`, while product mutation against tickets already in `done/` or `in-review/` still requires reopening first. |
| `git_branch` | Create or switch a local branch. | Mutating. Use only for explicit branch workflows; trunk-based delivery normally stays on `main`. |
| `git_push` | Push committed changes. | Mutating. Strict trunk allows pushing `main`. |

## Selection Guide

- Need Mars Harness behavior, versioning, setup, release, score, trust, or target
  harness lifecycle operations: use `mars_harness_cli`.
- Need to verify private Mars Harness release access before update, release
  verification, install repair, or version-drift remediation: use
  `github_auth_check` or `mars-harness auth github check`.
- Need to add, remove, rename, or change a `mars-harness` CLI command or flag:
  update `mars_harness_cli`, generated skills, generated doctrine, and product
  docs using [cli-tool-skill-sync.md](cli-tool-skill-sync.md).
- Need to discover or invoke the universal tool surface from an operator shell
  or external LLM context: use `mars-harness tools list` and
  `mars-harness tools run <name> --args-json '{...}'`. Add
  `--trust contributor` only for deliberate mutating tool calls.
- Need an MCP-compatible client or local harness agent to see Mars Harness tools
  as native tools: configure it to launch
  `mars-harness mcp serve --repo <path> --trust observer|contributor`.
- Need to run or prepare the whole release ritual: use `release_orchestrate`,
  `git_release_guard`, and `github_release_status` before mutating state; use
  `mars_harness_cli` with `release backfill-notes --check` when auditing
  historical changelog narrative compliance.
- Need a durable repo-owned note: use `record_decision`.
- Need backlog, dogfood, dependency, or intervention-debt work item creation:
  use `ticket_create`. Do not hand-write new ticket markdown with
  `file_write`.
- Need ticket lifecycle movement after a ticket already exists: use `git mv`
  through `shell_exec` and commit it. Blast-radius policy permits the deletion
  side only when the same ticket ID is present in another lifecycle directory as
  the staged, untracked, or already-existing counterpart. For feature tickets,
  fill `evidence_links` and `verified_by` before moving to `done/`; tool policy
  blocks missing evidence before the move. Do not copy a ticket into `done/`
  and then delete the source; tool policy requires one lifecycle move.
- Need COO planning updates: use `file_write` only for `docs/exec-plans`,
  `docs/features`, or `docs/goals/observations.md`; product implementation
  files must wait for CTO tickets and Engineer delivery.
- Need a dispatch-mode handoff, blocker, review request, no-work outcome, or
  completed-work signal: use `job_disposition_record` after `git_status` is
  clean or after committing the produced work with `git_commit`. Runtime-only
  `.harness/learnings.yaml` metadata is ignored by the disposition gate and may
  be auto-committed by the server when it is the only dirty path, but any
  product, ticket, documentation, or source change must be committed first.
  Review roles that receive no-op shell placeholder guidance after successful
  validation must stop tool use and call `job_disposition_record`; if Go source
  exists without `_test.go` files, QA records `changes_requested` for tests.
- Need a new deterministic capability: use `tool_create`, then finish the code
  and tests manually.
- Need a new or revised agent persona: use `persona_create`, then add canonical
  foundation entries to `internal/personas` when the persona is a foundation
  default.
- Need to decide whether repeated work deserves a tool: use
  `task_trace_summarize`, then create or update a ticket or tool.
- Need to keep documentation, doctrine, and tools mirrored: use
  `docsync_audit`, `architecture_audit`, `harness_doctrine_sync`,
  `tool_creation_guard`, and `tool_inventory_audit`.
- Need to inspect generated dependency/build churn before a job, commit, or
  package-manager operation: use `workspace_hygiene`. Missing ignore policy may
  be auto-repaired by `serve` as a `.gitignore`-only commit when generated paths
  are untracked and `.gitignore` has no user changes.
- Need dependency setup or package fetch/install: use `dependency_sync`, not raw
  package-manager commands through `shell_exec`.
- Need to know which docs must be checked after touching a code file: read the
  file's `MarsDocSync` block and run `docsync_audit` or
  `mars-harness docsync audit --repo .`.
- Need ordinary repository inspection: use `file_search`, `grep`, `file_read`,
  `git_status`, or `git_diff`.
- Need ordinary repository mutation: use `file_write`, `git_commit`, and
  `git_push` with the repository's operating rules. In local throwaway demos
  with no configured remote, `git_push` is a clean skip that leaves the commit
  local instead of creating a retry loop.
- Need to repair a failing Engineer test/build command: edit source, tests,
  fixtures, or package/build config inside the failed package scope when the
  failed command named one. If the failure is caused by duplicate generated
  tests, Engineer may use non-recursive `rm` or `unlink` only for test-like
  files created or rewritten earlier in the same job. Do not create alternate
  root entrypoints while the repair lane is unresolved.
- Need a command outside the built-in tool surface: use `shell_exec`, keep the
  command narrow, and record any reusable gap as a tool improvement.
- Need Dogfood validation evidence: keep it observation-first. Dogfood may
  write `docs/reports/dogfood/*.md` and create target-owned tickets with
  `ticket_create`, but it must not edit product source, package manifests,
  lockfiles, config, or harness scaffold to make validation pass.

## Maintenance Rules

- New built-in tools must originate through `tool_create` before manual
  implementation. If an agent bypasses `tool_create`, it must first record a
  durable exception with `record_decision` and add design-doc rationale before
  the change is complete.
- Every newly created tool must extend this glossary in the same change that
  implements or exposes the tool.
- Update this glossary in the same change that removes, renames, or materially
  changes a built-in tool.
- Mirror changes into generated target defaults in `internal/scanner/init.go`.
- Update scanner tests so initialized harnesses keep this first-class tool
  context.
- Keep use cases short and action-oriented; deeper rationale belongs in design
  decisions.
