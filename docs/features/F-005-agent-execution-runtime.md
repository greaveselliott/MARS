# F-005: Agent Execution Runtime

- Feature ID: F-005
- Goals: G-001, G-003
- Status: partially-passing
- Owner: CTO

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-005-S001 - Context assembly builds bounded, role-scoped prompts from repo-owned harness artifacts.
2. F-005-S002 - The agent loop handles multi-turn tool calling, completion, error outcomes, and retryable inference failures.
3. F-005-S003 - Tools are allowlisted, validated, and executed through repo-root containment.
4. F-005-S004 - Execution traces persist enough turn and tool detail to make outcomes auditable.
5. F-005-S005 - Budgets, max turns, max tool calls, context pruning, and wall time stop runaway jobs.
6. F-005-S006 - `mars-harness run <role> --repo <path>` executes one role with terminal-result truth.
7. F-005-S007 - Tool creation and CLI-operation tools are first-class mirrored harness capabilities.
8. F-005-S008 - Persona creation tools scaffold role manuals and prompt manuals with explicit ownership and feedback sections.
9. F-005-S009 - `ticket_create` prevents independent feature-ticket fan-out for the same BDD scenario.
10. F-005-S010 - Mutating tool policy preserves canonical feature-contract paths and clean terminal handoffs.
11. F-005-S011 - COO mutating tools are limited to planning artifacts before CTO ticketing.
12. F-005-S012 - Git push remains terminal-safe in local demo repositories without remotes.
13. F-005-S013 - Generated role guidance uses bounded, project-appropriate validation evidence instead of shell-heavy churn.
14. F-005-S014 - A stuck tool handler times out and returns control to the agent loop.
15. F-005-S015 - Long-running app validation uses managed background processes instead of shell-background process leaks.
16. F-005-S016 - Long-running tool histories prune old tool-call arguments as well as tool outputs before context overflow.
17. F-005-S017 - Dispatch jobs get one final terminal-tool opportunity at the turn-budget boundary.
18. F-005-S028 - Reviewer validation-procedure mistakes are tracked without becoming target validation failures.
19. F-005-S029 - Safe argv-shaped `cd <dir> && <test/build>` validation normalizes into shell-command execution.
20. F-005-S030 - Clear CLI input-validation probes count as expected negative-path runtime evidence.
21. F-005-S031 - Engineer validation-procedure mistakes are tracked without becoming product repair failures.
22. F-005-S032 - Test/build repair can remove same-job bad test-like files.
23. F-005-S033 - Test/build repair records failed package scope for later write policy.
24. F-005-S034 - Server jobs inject current run metadata so dated evidence uses the real execution date.
25. F-005-S035 - Review terminal convergence waits for docsync evidence before forcing `job_disposition_record`.
26. F-005-S036 - Review terminal convergence waits for tests when test files exist.
27. F-005-S037 - Review no-op recovery uses the same evidence gates as approval.
28. F-005-S038 - Direct `mars-harness` shell commands route to `mars_harness_cli`.

## Scenarios

### F-005-S001: Role-Scoped Context Assembly

Given a role, repo, ticket index, guardrail scope, knowledge routes, and context budget
When a job prompt is assembled
Then high-priority role and workflow context is included, irrelevant guardrails are filtered, and lower-priority context is truncated before exceeding budget

### F-005-S034: Current Run Metadata Grounds Dated Artifacts

Given a server job is assembled for any role
When the executor supplies the current time to the context assembler
Then the system prompt includes `## RUN METADATA` with `current_date`, `current_time`, and timezone
And budget trimming preserves the run metadata alongside the role prompt
And roles use `current_date` for evidence paths, report dates, release entries, and ticket timestamps instead of inferring dates from examples, model memory, or blocked shell commands

### F-005-S002: Multi-Turn Tool Loop

Given an LLM response includes tool calls and later a completion signal
When the agent loop runs
Then tool calls execute in order, tool results are returned to the model, and the job ends with a classified outcome

Given a local model emits a tool call as `<function=name>` with nested `<parameter=arg>` blocks instead of structured JSON
When the parser inspects assistant content
Then the runtime converts the tagged call into the same function-call shape and executes it rather than treating the reply as final prose

Given a local model emits a tool call as `<tool_call>name{key:<|"|>value<|"|>}</tool_call>`
When the parser inspects assistant content
Then the runtime converts the inline tagged call into JSON tool arguments and executes it rather than treating the reply as final prose

Given a server job successfully records a terminal disposition through `job_disposition_record`
When the tool result is traced
Then the agent loop stops immediately with a completed outcome instead of asking the model for more turns after it has already declared the job terminal

Given a server job requires a terminal tool such as `job_disposition_record`
When the model attempts to finish with prose and no tool call
Then the loop appends one corrective user turn requiring the terminal tool call, and if the model still finishes without the tool the executor's normal completion validation fails the job

### F-005-S003: Tool Execution Containment

Given a role has an allowlist and trust level
When it asks to call a tool
Then unknown tools, disallowed tools, invalid JSON, observer mutations, path escapes, secret writes, destructive shell commands, and mutating shell commands in already-over-budget repos fail closed, while known read-only shell inspection remains available for diagnosis

Given `docs/features/F-001-product-walking-skeleton.md` already exists
When a role attempts to create any second `docs/features/F-001*.md` contract with `file_write`
Then the tool policy blocks the write and instructs the role to update the canonical feature contract

Given a role writes `docs/features/F-001-product-walking-skeleton.md`
When the file contains a scenario heading with a different feature ID such as `### F-002-S001`
Then the tool policy blocks the write, names the mismatched heading line, and instructs the role to rename the scenario to `F-001-SNNN` or create the matching `F-002` contract first

Given a non-Orchestrator role has uncommitted repo changes
When it attempts to record a terminal `job_disposition_record` that approves, completes, requests changes, blocks, fails, or hands off work
Then the tool policy blocks the disposition, names the dirty paths, and instructs the role to run `git_status`, commit with `git_commit`, and record the disposition after the tree is clean

Given the only uncommitted path is the runtime-managed `.harness/learnings.yaml`
When a non-Orchestrator role records a terminal `job_disposition_record`
Then the disposition is allowed because runtime convention-learning metadata is not product work, the server may commit that runtime-only metadata after the job, and any additional product or documentation dirty path still blocks the handoff

Given the COO role is shaping the first product slice
When it attempts to write implementation files such as root HTML, source, package, test, or build artifacts
Then tool policy blocks the mutation and requires the work to pass through CTO ticketing and Engineer delivery, while allowing planning writes under `docs/exec-plans`, `docs/features`, and `docs/goals/observations.md`

Given the CEO role is shaping fresh product direction
When it attempts to write COO-owned planning artifacts such as `docs/exec-plans/*` or `docs/features/*`
Then tool policy blocks the mutation and requires CEO to update strategy artifacts only, then hand off with `next_need: exec_plan` or another explicit downstream need

Given the COO role has shell access in an existing target manifest
When it attempts a mutating `shell_exec`
Then tool policy blocks the command so shell access cannot bypass the planning-only ownership boundary

Given a role calls `shell_exec` with `argv`
When the argv tokens contain shell-only syntax such as redirection, pipes, control operators, command substitution, or shell builtins
Then the tool fails before process execution with an actionable instruction to use `shell_command` for shell syntax or file tools for content changes

Given a local model emits a simple malformed `shell_exec` argv shape
When `argv` is a JSON-encoded array string, a Python-style quoted list string, or a one-item simple command string with no shell syntax
Then tool policy and execution normalize it into the same argv tokens so ownership checks, guardrails, and execution do not spend extra turns recovering from harmless formatting drift

Given a local model emits simple list fields for `job_disposition_record` as strings
When `evidence_links`, `work_product_ids`, `blocked_by`, handoff list fields, or feedback evidence are JSON-encoded list strings, Python-style quoted list strings, or a single path string
Then disposition recording normalizes those fields before validation so terminal handoff evidence is durable without repeated formatting retries

Given a local model emits a no-op `shell_exec` call
When `argv` is empty, `argv` contains only blank text, or the command is a single `:`
Then the tool does not execute a process, returns a tool error that cannot be mistaken for progress, and includes phase-aware guidance for either implementation work before validation or ticket evidence, commit, push, and `job_disposition_record` after validation

Given a role runs a direct runtime command through `shell_exec`
When a command such as `go run`, `cargo run`, `dotnet run`, a language interpreter entrypoint, a package start script, or a bounded smoke probe exits successfully after exercising ticket behavior
Then the role session records that command as validation evidence so later completion policy can distinguish proven product behavior from unresolved work

Given Engineer has already seen a no-op `shell_exec` failure after successful validation and dirty ticket work
When it repeats another no-op placeholder call instead of committing or closing the ticket
Then policy treats the repeated no-op as a loop boundary and redirects Engineer to `git_status`, evidence update, `git_commit`, lifecycle completion, and terminal `job_disposition_record`

Given Engineer has claimed an in-progress product ticket and has already seen a no-op `shell_exec` failure before validation
When it repeats another no-op placeholder call before product implementation
Then policy treats the repeated no-op as a loop boundary and redirects Engineer to read the ticket and feature contract, then use `file_write` for implementation or record a blocked disposition

Given a long-running job accumulates large historical `file_write`, `shell_exec`, or review tool calls
When the next model request would exceed the configured context window
Then the agent loop prunes old tool results, old assistant tool-call arguments, and older prose while preserving the system prompt, initial user request, and recent working tail with enough margin for provider tokenization differences

Given a dispatch-mode job requires a terminal tool such as `job_disposition_record`
When the role reaches its configured model turn budget before recording that terminal tool
Then the agent loop adds exactly one final prompt requiring the terminal tool, forbids further inspection or validation, and allows that one extra response to record a structured disposition before ending as `max_turns`

Given a local model emits malformed `mars_harness_cli` args
When `args` is a JSON-encoded array string, a Python-style quoted list string, or a simple single command string
Then the tool normalizes the command before binary resolution so release, score, trust, and update flows do not fall back to stale PATH binaries because of harmless formatting drift

Given a local model emits malformed path-list fields for built-in tools
When tools such as `workspace_hygiene`, `git_diff`, or `git_commit` receive `paths` as a JSON-encoded array string or Python-style quoted list string
Then the tool normalizes the path list before policy checks so generic recovery works across project archetypes instead of only one demo path

Given a role writes a source or test file through `file_write`
When the path is a source root such as `src/`, `cmd/`, `app/`, `pages/`, `web/`, `static/`, `.github/workflows/`, or a root-level source file
Then the write is blocked unless the content already contains valid top-of-file `MarsDocSync` docs metadata that points at existing documentation

Given a local model emits a malformed `file_write` payload with `<parameter=content>` embedded in `path`
When the separate `content` field is empty
Then the tool splits the marker into the intended path and content before policy checks and file creation

Given a role attempts to configure repository remotes through `shell_exec`
When the command is `git remote add`, `git remote set-url`, `git remote remove`, `git remote rm`, or `git remote rename`
Then the tool blocks the command and instructs the role to record a release blocker instead of inventing or rewriting remotes

Given a role calls `shell_exec` with `shell_command`
When the command uses the shell background operator `&` to emulate a long-running server
Then the tool rejects the call before process execution and instructs the role to use `background:true`, run readiness probes separately, and avoid process leaks

Given a role starts a long-running command with `background:true`
When the process exits during the startup capture window
Then the tool returns an error with the initial output and exit code so the role treats the result as a boot failure instead of continuing as though the server is running

Given a role starts a long-running command with `background:true`
When the agent job ends or the harness cleans background processes
Then the tool terminates the tracked process, its process group, and any known descendant processes so wrapper commands such as `go run` do not leave child servers occupying ports for later runs

Given a role starts a long-running command with `background:true`
When the role later calls `shell_exec` `kill <tracked-pid>` for that managed background PID
Then the tool intercepts the kill request and terminates the tracked process tree, including known descendants, so same-job cleanup does not leave wrapper child servers occupying ports

Given a role calls `shell_exec` with a bare port token such as `:8080`
When the tool validates the command
Then the call fails before process execution and tells the role that ports are not executable commands, to start the app with its real server command using `background:true`, and to probe separately with `curl http://localhost:8080/health` or the target route

Given a role calls `shell_exec` for a likely server or watcher command
When the command is run in the foreground and matches a server entrypoint such as an HTTP `go run`, `npm start`, `npm run dev`, `python -m http.server`, `uvicorn`, `vite`, or `next`
Then the call fails before process execution and tells the role to rerun it with `background:true`, probe readiness with a separate request, and stop the tracked PID after validation

Given a role calls `shell_exec` with an external `timeout` or `gtimeout` executable
When the tool validates the command
Then the call fails before process execution and tells the role to use `timeout_seconds` for bounded foreground commands or `background:true` with separate probes for long-running servers

### F-005-S004: Auditable Trace

Given an agent job runs
When turns and tool calls occur
Then trace storage records enough context to diagnose what happened after the run

### F-005-S005: Runtime Limits

Given a model loops, exceeds budget, calls too many tools, or exceeds wall time
When the agent loop reaches the configured limit
Then the run stops with a non-success outcome that telemetry can classify

Given a tool handler does not return before the executor TTL
When the timeout expires
Then the executor stops waiting, returns an actionable timeout error to the model, records duration evidence, and lets the agent record a blocker or failure instead of stranding the job indefinitely

### F-005-S006: Manual Role Run

Given a user runs `mars-harness run <role> --repo <path>`
When manifest, context, tools, and inference endpoint are available
Then exactly one role run executes against the target repo and reports the terminal result

Given the same command is attached to an interactive TTY
When the role runs without `--debug`
Then the console uses a full-screen terminal dashboard, writes verbose logs to a command log file, and does not stream raw tool results into scrollback

Given the user passes `--debug` or legacy `--trace`
When the role runs
Then verbose trace and log output streams inline while the same command log file is still written

Given the target has no `.harness/manifest.yaml`
When the user passes `--dry-run --no-init`
Then the command reports the missing harness boundary, writes no target harness files, and exits without calling the LLM

### F-005-S007: Mirrored Built-In Tools

Given a recurring deterministic harness operation exists
When it is implemented as a built-in tool
Then the tool has schema, tests, registry exposure, trust-policy review, glossary documentation, and target mirroring when applicable

### F-005-S008: Persona Creation Tool

Given a role persona needs to be created or revised
When `persona_create` runs with ownership, feedback, stop-condition, and handoff sections
Then it scaffolds a repo-local persona manual, prompt manual, role registry row, and optional manifest role, and foundation defaults still require adding the canonical entry to `internal/personas`

### F-005-S009: BDD Scenario Ticket Dedupe

Given a feature ticket already exists for a BDD scenario
When a role calls `ticket_create` for another ordinary feature ticket with the same exact BDD scenario set and no `depends_on`
Then the tool returns the existing ticket as a duplicate and creates no new backlog file

Given the second ticket is an explicitly dependent decomposition
When it carries `depends_on` metadata naming the earlier ticket
Then `ticket_create` may create the dependent ticket with the same BDD scenario because the relationship is no longer independent fan-out

### F-005-S010: Clean Terminal Handoff Policy

Given an Engineer records a successful disposition for a named product ticket
When that ticket does not live in `docs/tickets/done/`
Then `job_disposition_record` blocks the disposition and instructs the Engineer to claim or update evidence, move the ticket to done, commit, and then request QA review

Given an Engineer has an ordinary product ticket in `docs/tickets/backlog/`
When it tries to mutate product, package, config, feature, dependency, build, or commit state before moving a ticket to `docs/tickets/in-progress/`
Then tool policy blocks the mutation and instructs the Engineer to claim the product ticket with `git mv`, commit the claim, and then continue

Given an Engineer has an ordinary product ticket in `docs/tickets/backlog/` and no in-progress product ticket
When it tries any non-claim `shell_exec`, including read-only discovery or a no-op placeholder
Then tool policy blocks the shell call and instructs the Engineer to run the backlog-to-in-progress `git mv` claim first

Given QA or Security reviews a named ticket
When it attempts to record `approved`, `completed`, or `in_review`
Then tool policy requires successful validation evidence from the current job, requires a passing test command when test files exist, blocks approval after failing build, test, or unexpected runtime validation commands, and allows non-zero runtime probes to serve as expected negative-path evidence only when `shell_exec expected_exit_code` is set and matched

Given QA reviews a named ticket in the generated default manifest
When it needs to prove the ticket's BDD scenario with command evidence
Then QA may use `shell_exec` for bounded validation commands such as tests, builds, or direct runtime probes, including `expected_exit_code` for intentional error-path probes, while repo writes and product mutation remain unavailable by default

Given QA or Security makes an obvious validation-procedure mistake in a Go build or test command
When the command failure indicates a package-targeting mistake rather than a compile, test, or product runtime failure
Then the tool session records the procedure failure separately, permits a corrected validation command in the same review job, and still blocks approval until successful validation evidence exists

Given Engineer makes an obvious validation-procedure mistake in a Go build or test command during product delivery or review rework
When the command failure indicates a package-targeting mistake such as missing `./` before a repo-relative package path
Then the tool session records the procedure failure separately, permits a corrected validation command in the same job, and does not require a meaningless source edit before validation can continue

Given Engineer observes a real compile error, failing test, or unexpected runtime validation failure
When Engineer attempts unrelated validation, ticket completion, commit, or successful disposition
Then the existing product repair guardrails still block progress until the target-owned failure is repaired and validated

Given Engineer has an unresolved test/build failure and writes a bad test-like repair file in the same job
When Engineer removes that same path with non-recursive `rm` or `unlink`
Then the runtime permits the cleanup so the role can continue same-lane repair and rerun the failing test/build command

Given Engineer tries to remove an unmarked test file, product source file, or recursive path during unresolved test/build repair
When the shell command is evaluated
Then the runtime blocks the cleanup and keeps the role in source/test repair plus same-lane validation

Given a role calls `shell_exec` with argv tokens shaped exactly like `cd <dir> && <recognized test-or-build command>`
When every token is simple and the right-hand side is validation-only
Then the executor normalizes the call into the existing `shell_command` path before policy and execution

Given argv tokens include arbitrary shell syntax such as pipes, redirects, substitutions, background operators, cleanup commands, or non-validation work
When `shell_exec` validates the request
Then argv mode still rejects the call before process execution

Given a role builds a runnable Go artifact outside the target repo for validation
When it uses a temp output path that does not include the `-validation` suffix
Then tool policy blocks the build before execution and redirects the role to `/tmp/<project>-validation` so same-session freshness tracking protects the later runtime probe

Given Engineer has an outstanding unexpected runtime validation failure for an exact command in the current job
When the same command later exits successfully
Then the session clears that runtime-failure blocker so completion can proceed on repaired evidence rather than stale ticket metadata

Given Engineer has already observed an unexpected runtime validation failure for an exact command in the current job
When Engineer reruns that command with `expected_exit_code`
Then the expected-exit result does not clear the outstanding failure; only a successful exit for the exact command repairs the Engineer completion blocker

Given Engineer has accidentally run an obvious missing-argument runtime probe without `expected_exit_code`
When Engineer immediately reruns that exact no-argument command with matching `expected_exit_code`
Then the result counts as expected negative-path evidence and clears that procedural blocker

Given Engineer has an unresolved runtime failure from an earlier missing-required-input probe
When policy blocks a different runtime probe, ticket completion, or successful disposition
Then the blocker text tells Engineer to rerun the exact missing-argument command with `expected_exit_code`, while positive acceptance failures still require a clean exact rerun

Given a direct runtime validation command exits 0 but prints error-shaped stderr such as `error:` or usage text
When the role session records the tool result
Then the runtime probe is tracked as failed evidence until the exact command later exits 0 without error-shaped stderr

Given Engineer has already observed an unexpected runtime validation failure for an exact command in the current job
When Engineer attempts another runtime probe before editing the implementation
Then tool policy blocks the probe and tells Engineer to inspect and edit the implementation before rerunning the exact failed command

Given an enabler or remediation ticket explicitly requires end-to-end evidence
When evidence links and verifier metadata are present
Then the ticket can move to `docs/tickets/done/` without being converted into a feature ticket or inventing BDD scenarios

### F-005-S011: Role Ownership Tool Boundaries

Given the Dogfood role is validating a target
When it attempts to write product source, package manifests, lockfiles, config, or harness scaffold with `file_write`
Then tool policy blocks the write and directs Dogfood to create target-owned tickets or write bounded evidence under `docs/reports/dogfood/`

Given Dogfood has created a target-owned finding ticket in the current run
When it attempts additional shell validation, another ticket, or other non-handoff work
Then tool policy blocks the action and directs Dogfood to commit, push if possible, and record disposition before continuing

Given the COO role has shell access in an existing target manifest
When it attempts a mutating `shell_exec`
Then tool policy blocks the command so shell access cannot bypass the planning-only ownership boundary

### F-005-S012: Local Demo Push Skip

Given a local throwaway repository has commits on `main` but no configured `origin` remote
When an agent calls `git_push`
Then the tool returns a successful skipped-push result explaining that the commit remains local instead of causing push retry loops

### F-005-S013: Bounded Static Demo Validation

Given a generated target is intentionally static HTML/CSS/JS with no package manifest
When Engineer closes a product ticket or Dogfood validates the product path
Then generated role guidance accepts a bounded static HTTP smoke test as build/run evidence, instructs Engineer to update ticket evidence with one full-file replacement instead of repeated shell substitutions, instructs Dogfood to skip package-manager and container-build expectations that do not apply, and keeps Dogfood observation-first without product/package mutation

### F-005-S016: Structured Argv Validation Evidence

Given a command receives literal multiline text as a single argument
When a role runs it through `shell_exec` argv mode
Then the command receives the newline-containing argument without shell parsing, while shell control syntax remains blocked in argv mode

### F-005-S017: Tool Failures Stay Attached To Session Progress

Given a planning role calls `ticket_create` with list fields encoded as quoted strings
When JSON unmarshalling rejects the payload before a ticket is created
Then the tool error names the malformed field, shows the required JSON array shape, and records unresolved ticket-creation failure state in the role session

Given a later valid `ticket_create` succeeds in the same role session
When the role records a terminal disposition
Then the unresolved ticket-creation failure state is cleared and normal dispatch can continue

### F-005-S018: Exact Runtime Correction State

Given Engineer runs an obvious no-argument or missing-required-input runtime probe without `expected_exit_code`
When the command exits non-zero in the current role session
Then the session records the exact failed `shell_exec` command and the exact correction with matching `expected_exit_code`

Given that exact correction later runs successfully
When the expected negative-path result matches
Then the session clears the outstanding missing-argument runtime blocker and allows normal validation to continue

Given Engineer runs an obvious CLI input-validation runtime probe without `expected_exit_code`
When the command exits non-zero with required-input, usage, invalid-input, or surplus-argument output and no crash marker
Then the result counts as expected negative-path runtime evidence without opening an unresolved runtime blocker

Given Engineer runs a positive runtime acceptance command with valid input
When the command rejects that input as invalid
Then the result remains an unexpected runtime validation failure that blocks completion until repaired

### F-005-S019: External Validation Artifact Freshness After Edits

Given a role builds an external validation artifact such as `<validation-root>`
When the build succeeds
Then the session records both same-session artifact freshness and the current runtime-edit watermark for that artifact

Given Engineer observes a positive acceptance runtime failure and then edits source
When Engineer tries to rerun the old external validation artifact
Then tool policy requires rebuilding that artifact before trusting rerun evidence

### F-005-S020: Ticket Evidence Requires Current Validation

Given Engineer is delivering an in-progress product ticket
When the role writes non-empty `evidence_links` or `verified_by` into that ticket
Then the active session must already contain successful validation evidence from a test, build, or runtime command

Given no successful validation command has run in the current Engineer job
When Engineer tries to populate those evidence fields
Then `file_write` blocks the ticket update and tells Engineer to run validation before writing exact ticket evidence

### F-005-S021: External Validation Artifact Correction Is Exact

Given a role tries to run a `<validation-root>` binary that was not built in the current role session
When `shell_exec` policy blocks the stale artifact
Then the tool error includes the exact `shell_exec argv` `go build -o <validation-root> ...` correction to run next

Given the target is a root Go CLI with `go.mod` and `main.go`
When that correction is generated
Then the build target is `.` so QA or Security can rebuild the binary without guessing the entrypoint

### F-005-S022: Go Build Guard Corrections Preserve Targets

Given a role runs `go build` without `-o` or with an unsafe output path
When shell policy blocks the build before creating an artifact
Then the error includes an exact `shell_exec argv` correction that writes to `<validation-root>`

Given the original build targeted `./...`, `./cmd/<name>`, or a concrete Go file
When the correction is generated
Then the target argument is preserved exactly so the next role does not guess `.` for a cmd-layout project

### F-005-S023: Failed Missing-Input Corrections Unlock Repair

Given Engineer has an unexpected no-argument or missing-required-input runtime failure
When the exact `expected_exit_code` correction is attempted and still fails
Then the session records that correction attempt as repair evidence rather than asking for the same repro forever

Given that failed correction attempt is recorded
When Engineer writes implementation code to fix the product behavior
Then `file_write` is allowed while completion, commits, ticket done-moves, and unrelated runtime probes remain blocked until the exact runtime failure is repaired

### F-005-S024: Exact Runtime Repair Clears Repeated Failure Counts

Given the same exact runtime validation command fails multiple times in one job
When that exact command later exits successfully
Then all unmatched outstanding failures for that command fingerprint are repaired in the session

Given other runtime validation commands remain unrepaired
When the exact successful rerun repairs only its own fingerprint
Then unrelated outstanding runtime blockers still remain until their exact commands pass

### F-005-S025: Reviewer Shell Is Validation-Only

Given QA or Security has shell access for review
When it runs tests, builds, direct runtime probes, fresh `<validation-root>` binaries, or HTTP probes
Then those commands can count as bounded validation evidence

Given QA or Security tries to run setup mutation such as `go mod init`, product edits, broad discovery, cleanup, or a no-op placeholder
When the command does not directly validate the completed ticket
Then tool policy blocks the command before it mutates review state

### F-005-S026: Required Terminal Tools Get Circle Grace

Given a server job requires a terminal tool such as `job_disposition_record`
When the model repeats the same non-terminal tool call shape enough to trigger circle detection
Then the loop adds one corrective user turn requiring only the terminal tool instead of stopping immediately

Given that circle-grace reminder has been sent
When the next model response calls any non-terminal tool
Then the job ends with `circle_detected`

### F-005-S027: Review Evidence Converges To Terminal Tool

Given QA or Security has inspected a source or ticket file with `file_read`
And the same job has successful validation evidence with no failing test, build, or unexpected runtime validation outstanding
And the same job has run `docsync_audit`
When the next loop turn is prepared
Then the loop adds a terminal-only reminder requiring `job_disposition_record`

Given that clean-review-evidence reminder has been sent
When the next model response calls a non-terminal tool
Then the tool is not executed and the loop adds one stronger terminal-only correction

Given that clean-review-evidence correction has been sent
When the next model response still calls any non-terminal tool or replies in prose only
Then the job ends as a loop boundary before more tools execute

### F-005-S035: Review Terminal Boundary Waits For DocSync

Given QA or Security has inspected the completed ticket and implementation files
And the same job has successful validation evidence
But `docsync_audit` has not run in that review job
When the model calls `docsync_audit`
Then the runtime executes the audit instead of rejecting it as post-validation churn

Given `docsync_audit` has run after clean review validation
When the next model response calls a non-terminal tool instead of `job_disposition_record`
Then the runtime rejects that tool call and issues one terminal-only correction

### F-005-S036: Review Terminal Boundary Waits For Tests

Given QA or Security has inspected the completed ticket and implementation files
And the same job has successful docsync and build evidence
And the target repository contains test files
When the model calls a test command next
Then the runtime executes the test command instead of forcing terminal disposition from build-only evidence

Given the same review job has a successful test command after docsync and clean read evidence
When the next model response calls a non-terminal tool instead of `job_disposition_record`
Then the runtime rejects that tool call and issues one terminal-only correction

### F-005-S037: Review No-Op Recovery Does Not Bypass Missing Tests

Given QA or Security has inspected the completed ticket and implementation files
And the same job has successful build or runtime validation evidence
And the target repository contains test files
But the review job has not recorded a successful authoritative test command
When the model calls `shell_exec` with an empty argv or no-op placeholder
Then the runtime blocks the no-op with test-command guidance
And it does not force terminal-only `job_disposition_record` approval guidance

Given the same review job then runs the authoritative test command successfully
When the required docsync and read evidence are also present
Then the runtime may force the normal terminal `job_disposition_record` boundary

### F-005-S038: Mars Harness CLI Invocations Use The Structured Tool

Given any role attempts to run `mars-harness` through `shell_exec`
When the command is provided as argv or as the first executable in a shell command
Then the runtime blocks the shell command before execution
And the error names the equivalent `mars_harness_cli` args so the active harness executable is used instead of a stale installed binary

### F-005-S033: Test-Build Repair Scope Is Recorded

Given Engineer runs a recognized Go test or build command with a narrow package target
And that command fails as product validation rather than a command-procedure error
When the runtime records the unresolved test/build failure
Then it stores the failed package repair scope until a same-lane test/build command repairs the failure

### F-005-S039: Test-Build Repair Guidance Includes Failing Output

Given Engineer runs a recognized test or build command and it fails
When the runtime records the unresolved test/build repair state
Then it stores the exact unresolved command and a compact copy of the latest failing output

Given Engineer later attempts unrelated shell validation, completion, commit, or disposition while the test/build failure is unresolved
When guardrail guidance is returned
Then the message includes the exact command, latest failing output, and an instruction to edit implementation rather than weakening a test when the failing assertion matches the ticket, README, or BDD contract

Given Engineer reruns a same-lane test/build command successfully
When the runtime clears the unresolved repair state
Then the stored failing output is cleared with the command and scope state

## Out of Scope

- Parallel tool execution inside a single agent turn.
- Unbounded prompt stuffing.
- Treating chat transcript memory as the system of record.

## Descoped Scenarios

None.

## Evidence

- F-005-S001: `go test ./internal/context`
- F-005-S002: `go test ./internal/agent -run TestRun`
- F-005-S003: `go test ./internal/tools -run 'TestExecutor|TestShellExec|TestFileWritePolicyBlocksScenarioIDsThatDoNotMatchFeatureContract'` and `go test ./internal/serve -run TestFoundationAcceptance`
- F-005-S004: `go test ./internal/agent -run TestRun_persistsTraceToSQLite`
- F-005-S005: `go test ./internal/agent -run 'TestRun_(max|token|wall|circle|empty)'`
- F-005-S006: `go test ./cmd/mars-harness -run 'TestRunStartServeExposeDebugAndLogFileFlags'` and planned broader E2E dogfood evidence
- F-005-S007: `go test ./internal/tools -run 'TestToolCreate|TestMarsHarnessCLI'`
- F-005-S008: `go test ./internal/tools -run TestPersonaCreate` and `go test ./internal/personas`
- F-005-S009: `go test ./internal/tools -run TestTicketCreate_dedupesIndependentFeatureTicketsForSameBDDScenario`
- F-005-S010: `go test ./internal/tools -run 'TestJobDispositionPolicy|TestEngineerDispositionPolicyRequiresTicketDoneBeforeSuccess|TestEngineerClaimPolicyRequiresInProgressBeforeProductMutation|TestReviewApprovalRequiresPassingValidationWhenTestsExist|TestShellExecPolicyAllowsEvidencedEnablerTicketDoneMove|TestShellExecPolicyBlocksEnablerTicketDoneMoveWithoutEvidence|TestRecordSessionToolOutcomeRepairsUnexpectedRuntimeFailureWithExactSuccess|TestRecordSessionToolOutcomeCorrectsUnexpectedRuntimeFailure|TestRecordSessionToolOutcomeEngineerExpectedExitDoesNotRepairUnexpectedRuntimeFailure'` and `go test ./internal/scanner -run TestInit_success`
- F-005-S011: `go test ./internal/tools -run 'TestCOO(FileWrite|ShellExec)Policy|TestDogfoodFileWritePolicyBlocksProductMutation|TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation'`
- F-005-S012: `go test ./internal/tools -run TestGitPush_noRemote`
- F-005-S013: `go test ./internal/scanner -run TestInit_success`
- F-005-S014: `go test ./internal/tools -run TestExecutor_toolHandlerHardTimeout`
- F-005-S015: `go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|RejectsBarePortCommands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'`, `demo-api-run7`, and `demo-api-run8` live evidence
- F-005-S016: `go test ./internal/tools -run 'TestShellExecArgvAllowsLiteralNewlineArgument|TestRecordSessionToolOutcomeTracksValidationCommands|TestReviewApprovalRequiresPassingValidationWhenTestsExist'`
- F-005-S017: `go test ./internal/tools -run 'TestTicketCreate_parseHintForQuotedBDDScenarios|TestRecordSessionToolOutcomeTracksTicketCreationFailures'`
- F-005-S018: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerCorrectsMissingArgumentRuntimeFailure`
- F-005-S019: `go test ./internal/tools -run 'TestRecordSessionToolOutcome(TracksValidationArtifactBuildAndRun|RefreshesValidationArtifactAfterRuntimeEdit)'`
- F-005-S020: `go test ./internal/tools -run 'TestEngineerTicketEvidenceWrite(RequiresValidation|AllowedAfterValidation)'`
- F-005-S021: `go test ./internal/tools -run TestExternalValidationArtifactMustBeBuiltInSameSession`
- F-005-S022: `go test ./internal/tools -run 'TestShellExecBlocks(DefaultGoBuildInsideRepoBeforeArtifact|DefaultGoBuildInShellCommandBeforeArtifact|DefaultGoBuildForCmdPackageWithExactCorrection|GoBuildOutputInsideRepoBeforeArtifact|GoBuildOutputInShellCommandSegmentBeforeArtifact|GoBuildOutputOutsideRepoWithoutValidationSuffix)'`
- F-005-S023: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksFailedMissingArgumentCorrectionAttempt|TestEngineerMissingArgumentRuntimeFailureAllowsImplementationEditAfterCorrectionAttempt'`
- F-005-S024: `go test ./internal/tools -run TestRecordSessionToolOutcomeExactSuccessClearsRepeatedRuntimeFailures`
- F-005-S025: `go test ./internal/tools -run TestReviewShellExecPolicy`
- F-005-S026: `go test ./internal/agent -run 'TestRun_requiredTerminalToolGetsOneCircleGraceTurn|TestRun_requiredTerminalTool'`
- F-005-S027: `go test ./internal/agent -run TestRun_reviewEvidenceReminder`
- F-005-S036: `go test ./internal/agent -run TestRun_reviewEvidenceDoesNotForceTerminalBeforeTestCommandWhenTestsExist` and `go test ./internal/tools -run TestReviewTerminalEvidenceWaitsForTestsWhenTestFilesExist`
- F-005-S037: `go test ./internal/agent -run TestRun_reviewNoopAfterBuildAllowsMissingTestCorrection` and `go test ./internal/tools -run TestReviewShellExecPolicyRoutesPostBuildNoopToTestsWhenTestsExist`
- F-005-S038: `go test ./internal/tools -run TestShellExecPolicyBlocksMarsHarnessBinary`
- F-005-S039: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane`
- F-005-S028: `go test ./internal/tools -run TestRecordSessionToolOutcomeReviewer`
- F-005-S029: `go test ./internal/tools -run 'TestShellExecNormalizesSimpleCdValidationArgv|TestShellExecArgvRejectsShellSyntax'`
- F-005-S030: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTreatsMissingArgumentCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeTreatsInvalidInputCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeTreatsSurplusArgumentCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeStillTreatsPositiveInputFailureAsUnexpected'`
- F-005-S031: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerGoBuildProcedureFailureDoesNotPoisonRepairLane`
- F-005-S032: `go test ./internal/tools -run TestEngineerFailingTestAllowsSameJobRepairTestFileRemoval`
- F-005-S033: `go test ./internal/tools -run TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation`
