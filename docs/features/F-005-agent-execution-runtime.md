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
6. F-005-S006 - `mars run <role> --repo <path>` executes one role with terminal-result truth.
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
28. F-005-S038 - Direct `mars` shell commands route to `mars_cli`.
29. F-005-S039 - Background server startup is validation setup, not review proof.
30. F-005-S040 - Test/build repair guidance includes failing output.
31. F-005-S042 - Browser-framework completion requires real build and product-state evidence.
32. F-005-S043 - Product brief capabilities must be represented in the scenario schedule and scenario headings before ticketing.
33. F-005-S044 - Browser smoke guidance remains runnable when copied into ticket evidence, and QA routes helper failures to foundation evidence instead of target rework.
34. F-005-S045 - Feature tickets cannot re-cover scenarios already covered by earlier ordinary tickets.
35. F-005-S046 - Browser validation helper files are not treated as product framework source.
36. F-005-S047 - CTO ticket creation can recover the pending product-scenario batch after a handoff gate.
37. F-005-S048 - Planner roles cannot mutate the target through shell execution.
38. F-005-S049 - Product capability extraction ignores active-goal non-goals and operational validation constraints.
39. F-005-S050 - Product capability matching accepts natural scenario titles.
40. F-005-S051 - Advanced score persistence does not descope basic scoring.
41. F-005-S052 - Review HTTP probes before server startup are validation-procedure failures.
42. F-005-S053 - Engineer review rework reopens the dispatch-named ticket.
43. F-005-S054 - Browser-framework modules loaded through plain Node eval are validation-procedure failures.
44. F-005-S055 - Dependency sync counts as a test/build repair action before same-lane validation reruns.
45. F-005-S056 - Browser-framework planning and review share the same product-smoke evidence path.
46. F-005-S057 - Generic gameplay summary labels do not become standalone capability requirements.
47. F-005-S058 - Alternate input exclusions do not descope basic keyboard movement.
48. F-005-S064 - Repeated policy blocks return actionable repair guidance to the active role.
49. F-005-S065 - OpenAI-compatible tool-call batches receive all model-requested tool results before synthetic runtime messages.

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

Given an OpenAI-compatible model emits multiple tool calls in one assistant message
And one of those tool calls triggers a runtime-injected follow-up such as `code_index` refresh or review terminal-evidence guidance
When the agent loop processes the batch
Then every model-provided `tool_call_id` receives a tool response immediately after the assistant message before any synthetic assistant/tool pair or user reminder is appended
And if a terminal tool completes the job before the whole batch executes, the remaining model-provided tool calls receive skipped tool responses so the transcript remains provider-valid

### F-005-S003: Tool Execution Containment

Given a role has an allowlist and trust level
When it asks to call a tool
Then unknown tools, disallowed tools, invalid JSON, observer mutations, path escapes, secret writes, destructive shell commands, and mutating shell commands in already-over-budget repos fail closed, while known read-only shell inspection remains available for diagnosis

Given a target repository contains a symlink as the parent or leaf of a requested `file_read`, `file_write`, `file_search`, or `grep` path
When the universal file tool resolves or opens that repository entry
Then the operation is bound through the standard-library repository descriptor, rejects the observed symlink, and neither reads nor mutates the symlink target outside the selected repository

Given a model- or agent-controlled operation reads or mutates repository-owned prompts, skills, policy, tickets, learnings, model overrides, local credential routing, release state, Jira mirrors, scanner inputs, or DocSync inputs
When that operation admits a repository
Then its related repository reads and writes retain the same descriptor-relative no-follow boundary for the operation, preserve documented optional-input fallbacks, and fail closed on an in-scope symlink or incomplete inventory without consuming outside content

Given `file_write` is admitted for a repository-relative path
When it creates or replaces the file
Then it creates missing real directories inside the repository and installs the new bytes through an exclusive same-directory temporary file, file sync, atomic rename, and directory sync with the requested `0644` mode

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

Given the COO role is updating the current failing scenario
When it attempts to create a second active exec-plan file such as `docs/exec-plans/active/current-failing-scenario.md`
Then tool policy blocks the write with guidance to update `docs/exec-plans/active/current-operating-plan.md` as the only active plan

Given the CEO role is shaping fresh product direction
When it attempts to write COO-owned planning artifacts such as `docs/exec-plans/*` or `docs/features/*`
Then tool policy blocks the mutation and requires CEO to update strategy artifacts only, then hand off with `next_need: exec_plan` or another explicit downstream need

Given a planner role such as CEO, Head of Strategy, COO, CTO, or CTO-weekly has shell access in an existing target manifest
When it attempts a mutating `shell_exec`
Then tool policy blocks the command so shell access cannot bypass the strategy, planning, ticketing, or implementation ownership boundary
And read-only inspection such as `git status --short` remains available when the command is otherwise policy-safe

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

Given a local model emits malformed `mars_cli` args
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
When the startup capture window ends while the process is still running and emitting logs
Then the tool returns a bounded initial output snapshot without racing the background capture goroutines

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

Given a user runs `mars run <role> --repo <path>`
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

Given an active backlog, in-progress, or in-review feature ticket already covers a BDD scenario
When Dogfood or another role calls `ticket_create` for a new ordinary feature ticket whose scenario list overlaps that active ticket
Then the tool returns the active ticket as a duplicate and creates no new backlog file
And the role can reference the existing active ticket in disposition evidence instead of multiplying findings

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

Given QA or Security runs an HTTP probe before any dev or static server is listening
When the probe fails with connection-refused or could-not-connect output
Then the tool session records the procedure failure separately, permits the reviewer to start the appropriate server with `background:true`, and still requires later successful validation evidence before approval

Given Engineer makes an obvious validation-procedure mistake in a Go build or test command during product delivery or review rework
When the command failure indicates a package-targeting mistake such as missing `./` before a repo-relative package path
Then the tool session records the procedure failure separately, permits a corrected validation command in the same job, and does not require a meaningless source edit before validation can continue

Given Orchestrator dispatches Engineer for review rework with a source disposition naming `ticket_id`
When no ordinary product ticket is currently in progress and the named ticket is in `docs/tickets/done/` or `docs/tickets/in-review/`
Then Engineer preflight requires reopening that dispatch-named ticket instead of selecting an older completed product ticket

Given Engineer observes a real compile error, failing test, or unexpected runtime validation failure
When Engineer attempts unrelated validation, ticket completion, commit, or successful disposition
Then the existing product repair guardrails still block progress until the target-owned failure is repaired and validated

Given Engineer has an unresolved test/build failure and writes a bad test-like repair file in the same job
When Engineer removes that same path with non-recursive `rm` or `unlink`
Then the runtime permits the cleanup so the role can continue same-lane repair and rerun the failing test/build command

Given Engineer tries to remove an unmarked test file, product source file, or recursive path during unresolved test/build repair
When the shell command is evaluated
Then the runtime blocks the cleanup and keeps the role in source/test repair plus same-lane validation

Given Engineer has an unresolved test/build failure and writes a focused test file under `test/` or `tests/`, with a conventional `*.test.*`, `*.spec.*`, or `_test.go` name
When the file write is evaluated
Then the runtime treats the edit as same-lane repair work rather than unrelated helper-script drift

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

Given a role writes or runs `make build` and the Makefile build target writes a Go binary into the repository such as `bin/<name>`
When file-write or shell policy inspects the target recipe
Then the Makefile write or command is blocked with guidance to run `go test ./...` or `go build -o <validation-root> <entrypoint>` so validation does not leave dirty build artifacts

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

### F-005-S038: MARS CLI Invocations Use The Structured Tool

Given any role attempts to run `mars` through `shell_exec`
When the command is provided as argv or as the first executable in a shell command
Then the runtime blocks the shell command before execution
And the error names the equivalent `mars_cli` args so the active harness executable is used instead of a stale installed binary

### F-005-S039: Background Server Startup Is Not Review Proof

Given QA or Security has inspected the completed ticket and implementation files
And the same job has run `docsync_audit`
When the reviewer starts a long-running app or static server with `background:true`
Then the server startup is treated as validation setup only
And review terminal convergence waits for a separate successful validation probe such as `curl -fsS http://127.0.0.1:<port>/`

Given that separate HTTP probe succeeds after the background server starts
When the same job has clean read and docsync evidence with no failing validation outstanding
Then the runtime may force the normal terminal `job_disposition_record` boundary

### F-005-S040: Background Server Output Stays Drained After Startup

Given a role starts a long-running dev server or static server with `shell_exec`
And the command uses `background:true`
When the tool returns the tracked PID after the startup capture window
Then the runtime continues draining stdout and stderr until the process exits
And later server request logs do not break the server pipe or cause empty HTTP replies

### F-005-S042: Browser Framework Completion Requires Build Evidence

Given an Engineer completes a browser-framework product ticket in a repository whose `package.json` declares a framework such as Phaser, Vite, React, Vue, Svelte, Three, Pixi, or Babylon
When the package has no deterministic build script, the build script is a no-op such as `echo` or `true`, the build script is copy-only such as `mkdir dist && cp ...`, the build script only runs syntax checks such as `node --check`, or the same job has not run a recognized build command successfully
Then the runtime blocks ticket evidence writes, done-ticket moves, and successful Engineer dispositions
And the error tells Engineer to add or fix the package build/browser validation surface, rerun validation, update evidence, and then close the ticket

Given an Engineer writes browser-framework package or entrypoint files for a Phaser brief
When `package.json` omits the local `phaser` dependency, omits a deterministic build script, uses only a no-op/copy-only/syntax-only build script, or `index.html` loads Phaser from a CDN-only script tag
Then `file_write` blocks the edit before the invalid target shape becomes committed product state
And the error directs Engineer to use a local npm dependency plus a real build/runtime validation path

Given an Engineer writes `package.json` for a Phaser brief
When a runtime script uses a MARS reserved port such as `18081` or starts a static source server such as `python3 -m http.server`
Then `file_write` blocks the edit before the target app collides with local harness runtime ports or bypasses Vite module bundling
And the error directs Engineer to use Vite dev or preview on an application port such as `5173`

Given an Engineer writes Phaser HTML or JavaScript source under a nested source path such as `src/index.html` or `src/main.js`
When the HTML loads Phaser from a CDN script, the JavaScript references `Phaser.*` without importing Phaser, or `new Phaser.Game` is constructed inside a scene callback
Then `file_write` blocks the edit before those lifecycle defects can grow into a long validation loop
And the error names the invalid source shape to repair

Given an Engineer writes `vite.config.js` or `vite.config.ts` for a Phaser brief
When the config imports Phaser, browser runtime code, or local `src/*` game modules
Then `file_write` blocks the edit before `vite build` can fail from Node evaluating browser-only code
And the error directs Engineer to keep Vite config limited to Vite/plugin configuration and import game code from browser entrypoints

Given an Engineer writes `vite.config.js` or `vite.config.ts` for a Phaser brief
When the config externalizes `phaser` from the production browser bundle
Then `file_write` blocks the edit before `vite build` can pass with an unresolved browser runtime import
And source inspection blocks ticket evidence, done moves, and approval if the externalization already exists

Given that browser-framework product ticket has build evidence but no browser-product smoke or equivalent source/runtime assertion
When Engineer tries to populate ticket evidence, move the ticket to done, or record a successful disposition
Then the runtime blocks completion before QA receives the ticket
And the error states that `node --check` or grep-only evidence is insufficient because the module graph and mounted UI state still need proof
And it names a literal `shell_exec argv ["node","-e", ...]` source/runtime assertion as an acceptable fallback when Playwright or Puppeteer is unavailable

Given Engineer runs a browser-product source/runtime assertion through `shell_exec`
When the command uses `argv: ["node", "-e", "..."]` and the eval code contains JavaScript semicolons or import expressions
Then argv validation treats the eval body as a language-code argument rather than shell syntax
And a successful command that proves Phaser canvas or `#game` mounting records browser-product smoke evidence

Given Engineer writes a durable validation helper such as `scripts/validate-phaser.js` or `tests/*.spec.js`
When that helper inspects source strings such as `new Phaser.Game` to prove browser-framework lifecycle shape
Then Phaser product-source lifecycle checks do not classify the helper as app source
And browser-framework source inspection ignores the helper while continuing to inspect `src/`, entrypoints, package scripts, and Vite config

Given Engineer has successful validation and a clean implementation commit for a browser-framework product ticket
And the same job is still missing required package build evidence or browser-product smoke evidence
When Engineer runs the missing build command or the bounded product smoke assertion
Then the post-validation shell convergence gate allows that command to execute
And other exploratory post-validation shell commands remain blocked until the ticket evidence is updated and the lifecycle move is ready

Given any role validates browser source
When it tries to run `node --check` against an `.html` or `.htm` file
Then the command is blocked as a validation-procedure mistake, not tracked as a product runtime failure
And the role is told to validate HTML entrypoints through package build and browser/static smoke instead

Given Engineer validates a browser-framework target
When a plain Node eval command imports or requires project browser modules and fails with missing browser globals such as `window is not defined` from framework package startup
Then the failure is recorded as a validation-procedure mistake, not an unresolved product runtime failure
And Engineer may continue with package build, managed server, browser-product smoke, or source/runtime assertion evidence without first making unrelated product edits

Given the README, vision, feature contract, or HTML source references Phaser before a package manifest exists
When CTO creates a feature ticket for the current scenario
Then the ticket cannot prescribe Go CLI paths, `go.mod`, `cmd/*`, or Go module setup unless the README explicitly names a Go backend
And the ticket must instead point Engineer toward a browser JavaScript package shape with local Phaser dependency and build evidence

Given a Phaser implementation uses ES module imports between local source files
When a source file imports a named symbol that the local module does not export, uses a symbol exported by another local module without importing it, references `Phaser.*` or `extends Phaser.Scene` without importing Phaser in that module, or an HTML file loads module syntax through a classic script tag
Then browser-framework completion is blocked with the exact source and target file named
And the role must repair the module graph before evidence, lifecycle closure, or approval can proceed

Given QA, Security, or Dogfood reviews that browser-framework ticket
When the package manifest is missing, Phaser is only loaded through a CDN/script tag, the package has no real build script, a framework build has not passed, no browser-product smoke or equivalent source/runtime assertion checked mounted UI state, or source inspection finds an obvious Phaser lifecycle defect such as config callbacks that are not defined or imported
Then approval is blocked
And terminal review guidance routes the role to `changes_requested` instead of approving based on static HTTP `curl` evidence alone

Given QA reviews a browser-framework ticket whose build passed and source inspection shows no framework lifecycle defects
When QA's smoke setup fails because the dev server was not running, the localhost probe was aimed at a stopped server, or the helper assertion was malformed
Then `changes_requested` with `next_need: implementation_rework` is blocked
And QA must rerun a managed server/product-smoke setup, approve with corrected evidence, or record a foundation/dogfood validation finding instead of sending target Engineer rework

### F-005-S043: Product Brief Capabilities Must Become Scenarios Before Ticketing

Given README, active goals, or the product brief says the target product should include or support explicit capabilities
And the generated `docs/features/F-001-product-walking-skeleton.md` still has generic starter scenario headings
When COO attempts to hand off planning as complete
Then `job_disposition_record` is blocked until the Scenario Schedule and scenario headings represent those explicit capabilities or list them under Descoped Scenarios with reasons

Given the feature contract mentions every explicit product capability only inside one broad runnable or inspectable scenario body
When COO attempts to hand off planning as complete
Then `job_disposition_record` is blocked until the Scenario Schedule entries or scenario headings break those capabilities into visible product slices for CTO ticketing

Given a product brief wraps one capability list across single line breaks
When COO or CTO capability checks extract explicit requirements from the brief
Then single newlines are treated as wrapped prose rather than sentence boundaries
And required capabilities later in the wrapped list must still become scenarios or explicit descoped entries

Given product strategy text introduces a capability list with a category prefix such as "all core product capabilities:"
When COO or CTO capability checks extract explicit requirements from the brief, vision, or active goals
Then the category label is removed before item matching
And individual capabilities such as a visible board, workflow controls, scoring or reporting, state transitions, and reset behavior match the scenario schedule directly

Given a product brief contains validation guidance such as build evidence, smoke evidence, or proof that the app mounts
When COO or CTO capability checks extract explicit requirements from the brief
Then those validation-evidence fragments are not treated as product capabilities
And short proof tails such as "mounts" or "plays" do not force standalone product scenarios

Given a product brief asks for validation instructions, manual reviewer validation, or enough instructions for a reviewer to confirm that the product works
When COO or CTO capability checks extract explicit requirements from the brief
Then that reviewer-validation wording is treated as validation procedure rather than product capability scope
And it does not require a standalone Scenario Schedule entry when the actual product behaviors are already represented

Given the brief names keyboard movement
When the feature scenario schedule covers keyboard controls, left/right/down input, and rotation
Then the capability guard treats the movement requirement as covered

Given the feature contract names explicit brief capabilities under `Out of Scope`
And those capabilities are not represented under `Descoped Scenarios` with rationale
When COO attempts to hand off planning as complete or CTO attempts to create tickets
Then the runtime blocks the handoff because required product behavior cannot disappear into generic out-of-scope text

Given the feature contract includes a required capability in the scenario schedule
And an Out of Scope line excludes only an advanced extension beyond that basic capability
When COO attempts to hand off planning as complete
Then the runtime does not treat the basic capability as descoped
And direct Out of Scope lines for required behavior still require Scenario Schedule coverage or explicit descoping rationale

Given the same incomplete scenario schedule reaches CTO
When CTO attempts to create a feature ticket for the current scenario
Then `ticket_create` is blocked with feedback to rewrite the scenario schedule before implementation tickets are created

Given the feature contract has an ordered scenario schedule
And no done ticket has covered the earliest scenario in that schedule
When CTO attempts to create a feature ticket only for a later scenario
Then `ticket_create` is blocked until the ticket includes the earliest uncovered scenario
And later scenarios may be batched only when that earliest scenario is also part of the same ticket

Given ordinary feature tickets already cover earlier scenarios in a feature contract
When CTO calls `ticket_create` for a new feature ticket that includes those already-covered scenarios again
Then the tool policy blocks the ticket and names the already-covered scenario IDs
And the feedback directs CTO to create the next uncovered scenario ticket only, or group it with later uncovered adjacent scenarios

Given CTO has already created the first early product ticket
And the implementation handoff gate names the next required product scenarios
When CTO retries `ticket_create` without a usable `bdd_scenarios` array but the pending gate state names those scenarios
Then the tool fills the missing BDD scenario list from the pending handoff state before creating the next ticket
And the created ticket records those BDD scenario IDs in frontmatter and evidence sections

Given COO rewrites the generated feature contract with product-specific scenarios
And the contract still uses durable BDD vocabulary such as product rules, workflow branches, state transitions, and user-visible outcomes
When COO records a completed planning disposition
Then the runtime accepts that vocabulary as valid product-specific documentation
And starter-placeholder blocking is limited to actual scaffold phrases such as starter contract text or placeholder-noun instructions

Given active goals include Markdown Scope bullets, Non-Goals, and implementation constraints such as npm install/build scripts
When COO records a completed planning disposition after covering the actual product behaviors in the feature scenario schedule
Then capability extraction treats each Markdown bullet as a separate statement
And non-goals such as optional future workflows, generic access wording, operational script constraints, and validation/build wording do not become required product scenarios
And natural product qualifiers do not block when the scenario schedule clearly covers the requested behavior
And documentation paths or Markdown references such as `docs/features/F-001-score-summary.md` are treated as citations rather than product capability tokens

### F-005-S064: Repeated Policy Blocks Return Role Repair Guidance

Given a role repeats the same guardrail or tool-policy blocked tool call in one job
When the tool executor returns the repeated policy error to the model
Then the tool result includes a compact `Guardrail repair required` section with the next allowed repair action instead of only replaying the original policy text

Given COO repeats a blocked planning disposition because product capability coverage is missing
When the repeated policy guidance is returned
Then the message tells COO not to call `job_disposition_record` again for the same payload, names the missing capabilities, names the feature contract path to repair, and directs COO to use `file_read`, `file_write`, then retry the disposition only after scenario coverage is updated

Given Engineer repeats a blocked `shell_exec` while an earlier test or build command is still unresolved
When the repeated policy guidance is returned
Then the message tells Engineer to stop trying alternate shell or dependency commands, repair the exact validation lane with `file_read` or `file_write`, and rerun the focused test/build before ticket moves, evidence, commits, runtime probes, or further shell exploration
And missing Go assertion dependencies introduced by a new test are routed toward standard-library test rewrites or dependency removal before rerunning `go test`

### F-005-S033: Test-Build Repair Scope Is Recorded

Given Engineer runs a recognized Go test or build command with a narrow package target
And that command fails as product validation rather than a command-procedure error
When the runtime records the unresolved test/build failure
Then it stores the failed package repair scope until a same-lane test/build command repairs the failure

### F-005-S041: Test-Build Repair Guidance Includes Failing Output

Given Engineer runs a recognized test or build command and it fails
When the runtime records the unresolved test/build repair state
Then it stores the exact unresolved command and a compact copy of the latest failing output

Given Engineer later attempts unrelated shell validation, completion, commit, or disposition while the test/build failure is unresolved
When guardrail guidance is returned
Then the message includes the exact command, a bounded compact failing-output excerpt, and an instruction to edit implementation rather than weakening a test when the failing assertion matches the ticket, README, or BDD contract
And repeated guardrail blocks do not replay long build output into the role context

Given Engineer reruns a same-lane test/build command successfully
When the runtime clears the unresolved repair state
Then the stored failing output is cleared with the command and scope state

### F-005-S055: Dependency Sync Repairs Missing Build Dependencies

Given Engineer has an unresolved test/build failure caused by missing package-manager dependencies or missing local build tools
When Engineer successfully runs `dependency_sync`
Then the runtime treats that dependency sync as a repair action for the unresolved test/build lane
And Engineer may immediately rerun the same-lane test/build command
And ticket completion, commits, runtime probes, and successful disposition remain blocked until that same-lane validation passes

### F-005-S056: Browser Framework Planning And Review Share Product-Smoke Guidance

Given CTO creates a feature ticket for a Phaser or browser-framework target
When the ticket prescribes CDN-only runtime loading or CDN acceptance criteria
Then ticket creation is blocked with guidance to require local package dependencies, deterministic build evidence, and browser-product smoke evidence

Given QA or Security has build and HTTP evidence for a browser-framework ticket but no product-smoke evidence
When approval is attempted
Then the approval blocker includes the canonical browser-smoke command or equivalent source/runtime assertion guidance

Given QA or Security starts a managed validation server through `shell_exec background:true`
When the reviewer stops the tracked PID with `kill`
Then the validation-only shell policy allows that tracked cleanup while continuing to block arbitrary cleanup or untracked process kills

### F-005-S057: Generic Gameplay Summary Labels Are Not Standalone Capabilities

Given an active goal heading says to implement core gameplay mechanics
And the README or goal body names concrete product behaviors such as movement, rotation, line clearing, score, game over, and restart
When COO or CTO capability checks extract product requirements
Then generic words such as core, gameplay, mechanic, and mechanics do not create a separate standalone capability
And the concrete behavior words remain required before planning can hand off to ticket breakdown

### F-005-S058: Alternate Input Exclusions Do Not Descope Basic Movement

Given a feature contract covers movement through keyboard controls or directional movement scenarios
And the Out of Scope section excludes mobile touch controls or another alternate input mode
When COO or CTO capability checks compare required movement behavior against out-of-scope text
Then generic controls wording alone does not count as movement coverage or movement descoping
And keyboard controls still count as movement coverage when the required behavior is keyboard movement

### F-005-S050: Product Capability Matching Accepts Natural Scenario Titles

Given a product brief asks for a named product workflow and state-transition behavior
When COO writes a feature contract whose scenario schedule names the concrete workflow and state transition without repeating the target's product label
Then the capability guard accepts the scenario outline as coverage for the concrete behavior
And product labels from the target brief are not required outline keywords when the behavior words are present
And the guard still requires distinct requested product capabilities to remain visible in the scenario schedule or scenario headings before CTO ticketing

### F-005-S051: Advanced Score Persistence Does Not Descope Basic Scoring

Given a product brief requires score tracking
And the feature scenario schedule includes line clearing with score points
When COO lists high-score tracking or persistence under Out of Scope
Then the capability guard treats that line as an advanced scoring extension rather than a descoping of basic score tracking
And `beyond ...` qualifier lines such as animations beyond basic movement or UI beyond the score display do not descope the basic behavior named after the qualifier
And enhancement-only exclusions such as animation polish or animations for piece movement or line clearing do not descope the covered basic movement or line-clearing behavior
And directly listing basic scoring under Out of Scope still requires a Descoped Scenarios rationale

### F-005-S059: Browser Evidence Completion Stops Shell Drift

Given an Engineer job is delivering a browser-framework ticket
And the same job has passed the required deterministic package build
And the same job has passed the required browser-product smoke
And implementation or ticket files remain dirty
When Engineer attempts another shell exploration command instead of closing the ticket lifecycle
Then the runtime blocks the shell command
And the policy guidance directs Engineer to commit dirty work, update ticket evidence, move the ticket to `done`, commit the lifecycle move, push when configured, and record disposition
And tracked background PID cleanup remains allowed so validation servers are not left running

### F-005-S060: Post-Build Browser Smoke Gate Blocks Substitute Probes

Given an Engineer job is delivering a browser-framework ticket
And the same job has passed the required deterministic package build
And the same job has not yet passed browser-product smoke
And implementation or ticket files remain dirty
When Engineer attempts generated-bundle inspection, plain Node `require('phaser')`, requiring browser bundles from Node, `node --check` on HTML, or trivial environment probes
Then the runtime blocks the shell command
And the policy guidance sends Engineer to the canonical browser-product smoke or equivalent source/runtime assertion
And package build reruns plus tracked background PID cleanup remain allowed

### F-005-S061: Advanced Out-Of-Scope Exclusions Do Not Descope Covered Basics

Given a feature contract covers basic product capabilities in the Scenario Schedule and scenario headings
And the Out of Scope section includes explanatory prose such as clear reasons or explicit rationale
And the Out of Scope section lists advanced-only extensions such as high-score persistence, combos, next-piece previews, multiplayer, mobile touch controls, sounds, or animation polish
When COO hands off to CTO for ticket breakdown
Then the runtime does not treat the explanatory prose or advanced-only exclusions as descoping covered basic capabilities
And required behavior still fails if it is directly listed under Out of Scope without Descoped Scenarios rationale

### F-005-S062: Capability Matching Ignores Glue Words Around Concrete Behaviors

Given README, goals, or feature prose uses phrases such as core gameplay including visible grid, show game over, or game over detection
And the Scenario Schedule or scenario headings cover the concrete behavior words around those phrases
When COO or CTO capability checks compare required product behavior against the feature contract
Then include/includes/including, show/shows/showing, display/displays/displayed, and detect/detected/detection do not become standalone missing capabilities
And concrete capabilities such as visible grid, line clearing, score tracking, game over, and restart remain required

### F-005-S063: Source-Only Foundation Maintainer Role

Given an operator runs `mars run foundation-maintainer --repo . --dry-run --no-init` from the MARS source repository
When the source repo has no `.harness/manifest.yaml`
Then the runtime assembles the source-only foundation role context from canonical repo docs
And it does not scaffold a source `.harness/manifest.yaml`
And running `foundation-maintainer` against a non-source repository fails with an actionable source-only role error
And generated target manifests do not include `foundation-maintainer`

### F-005-S065: Static Browser Smoke Requires Served Product Evidence

Given Engineer implements an intentionally static browser ticket
And `node --check` or another syntax command has passed
When Engineer tries to populate ticket evidence or move toward done without a same-job served-page smoke
Then the runtime blocks completion until the job starts the static server on an application port such as `5173` or `5174`, runs a separate HTTP probe such as `curl -fsS http://127.0.0.1:<port>/`, and records those exact commands
And a successful HTTP probe only counts as static product smoke when the response body is the served product page rather than a directory listing or generic host page
And package scripts using reserved MARS ports `18080`-`18089` or canned smoke scripts such as `node -e "console.log(...)"`, `echo`, or `true` are blocked before they become product evidence

Given QA or Security reviews a static browser ticket
When the only successful validation is a syntax check or a canned console-output command
Then approval is blocked and terminal guidance routes to `changes_requested` unless a served-page static smoke on an application port has passed in the same job
And `node -e` remains valid only when it performs a real source/runtime assertion that reads product files, checks product state, and fails on mismatch

Given Engineer has already committed static browser source after syntax validation
And the product ticket remains in progress without served-page smoke evidence
When Engineer starts a static server on a non-reserved application port or runs the matching HTTP probe
Then post-validation convergence allows those missing smoke commands while continuing to block unrelated shell exploration

Given Engineer attempts an ad hoc parser validation such as a Python `html5lib` probe
When the helper dependency is missing from the local environment
Then the runtime records a validation-procedure failure rather than an unrepaired product runtime failure
And Engineer can recover by running the canonical static served-page smoke instead of looping on the missing helper

## Out of Scope

- Parallel tool execution inside a single agent turn.
- Unbounded prompt stuffing.
- Treating chat transcript memory as the system of record.

## Descoped Scenarios

None.

## Evidence

- F-005-S001: `go test ./internal/context`
- F-005-S002: `go test ./internal/agent -run TestRun`
- F-005-S003: `go test ./internal/tools -run 'TestExecutor|TestShellExec|TestFileWritePolicyBlocksScenarioIDsThatDoNotMatchFeatureContract|TestCOOFileWritePolicyBlocksSecondActiveExecPlanWithSpecificGuidance'`, `go test ./internal/serve -run TestFoundationAcceptance`, the focused T-075 repository-boundary suites, and installed-candidate Dogfood at `c18030e` and `9ba8156`
- F-005-S004: `go test ./internal/agent -run TestRun_persistsTraceToSQLite`
- F-005-S005: `go test ./internal/agent -run 'TestRun_(max|token|wall|circle|empty)'`
- F-005-S006: `go test ./cmd/mars -run 'TestRunStartServeExposeDebugAndLogFileFlags'` and planned broader E2E dogfood evidence
- F-005-S007: `go test ./internal/tools -run 'TestToolCreate|TestMarsCLI'`
- F-005-S008: `go test ./internal/tools -run TestPersonaCreate` and `go test ./internal/personas`
- F-005-S009: `go test ./internal/tools -run 'TestTicketCreate_dedupes(IndependentFeatureTicketsForSameBDDScenario|ActiveFeatureTicketsForOverlappingBDDScenario)'`
- F-005-S010: `go test ./internal/tools -run 'TestJobDispositionPolicy|TestEngineerDispositionPolicyRequiresTicketDoneBeforeSuccess|TestEngineerClaimPolicyRequiresInProgressBeforeProductMutation|TestReviewApprovalRequiresPassingValidationWhenTestsExist|TestShellExecPolicyAllowsEvidencedEnablerTicketDoneMove|TestShellExecPolicyBlocksEnablerTicketDoneMoveWithoutEvidence|TestTicketDoneMoveSourcesPreserveShellCommandPathCase|TestRecordSessionToolOutcomeRepairsUnexpectedRuntimeFailureWithExactSuccess|TestRecordSessionToolOutcomeCorrectsUnexpectedRuntimeFailure|TestRecordSessionToolOutcomeEngineerExpectedExitDoesNotRepairUnexpectedRuntimeFailure'` and `go test ./internal/scanner -run TestInit_success`
- F-005-S011: `go test ./internal/tools -run 'TestCOO(FileWrite|ShellExec)Policy|TestDogfoodFileWritePolicyBlocksProductMutation|TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation'`
- F-005-S012: `go test ./internal/tools -run TestGitPush_noRemote`
- F-005-S013: `go test ./internal/scanner -run TestInit_success`
- F-005-S014: `go test ./internal/tools -run TestExecutor_toolHandlerHardTimeout`
- F-005-S015: `go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|RejectsBarePortCommands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'`, `go test ./internal/tools -race -count=1 -coverprofile=<validation-root> -covermode=atomic`, `demo-api-run7`, and `demo-api-run8` live evidence
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
- F-005-S038: `go test ./internal/tools -run TestShellExecPolicyBlocksMarsBinary`
- F-005-S039: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeDoesNotCountBackgroundServerAsValidation|TestReviewTerminalEvidenceIgnoresBackgroundServerStart'`
- F-005-S040: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane`
- F-005-S028: `go test ./internal/tools -run TestRecordSessionToolOutcomeReviewer`
- F-005-S029: `go test ./internal/tools -run 'TestShellExecNormalizesSimpleCdValidationArgv|TestShellExecArgvRejectsShellSyntax|TestShellExecArgvAllowsNodeEvalCodeArgument'`
- F-005-S030: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTreatsMissingArgumentCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeTreatsInvalidInputCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeTreatsSurplusArgumentCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeStillTreatsPositiveInputFailureAsUnexpected'`
- F-005-S031: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerGoBuildProcedureFailureDoesNotPoisonRepairLane`
- F-005-S032: `go test ./internal/tools -run TestEngineerFailingTestAllowsSameJobRepairTestFileRemoval`
- F-005-S033: `go test ./internal/tools -run TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation`
- F-005-S041: `go test ./internal/tools -run TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation`
- F-005-S042: `go test ./internal/tools -run 'TestCTOTicketCreateBlocksGoShapeForPhaserBrief|TestEngineerFileWriteBlocksGoScaffoldForPhaserBrief|TestEngineerFileWriteBlocksPhaserPackageWithoutBuildScript|TestEngineerFileWriteBlocksPhaserPackageCopyOnlyBuildScript|TestEngineerFileWriteBlocksPhaserCDNScriptTag|TestEngineerFileWriteBlocksNestedPhaserCDNScriptTag|TestEngineerFileWriteBlocksPhaserPackageReservedRuntimePort|TestEngineerFileWriteBlocksPhaserPackageStaticSourceServer|TestEngineerFileWriteBlocksPhaserSourceWithoutModuleImport|TestEngineerFileWriteBlocksRecursivePhaserGameConstruction|TestEngineerFileWriteBlocksPhaserRuntimeInViteConfig|TestEngineerFileWriteBlocksPhaserExternalInViteConfig|TestEngineerBrowserFrameworkEvidenceRequiresPackageForPhaserBrief|TestEngineerBrowserFrameworkTicketEvidenceRequiresBuildScript|TestEngineerBrowserFrameworkTicketEvidenceRequiresBuildSuccess|TestEngineerBrowserFrameworkTicketEvidenceRejectsNoopBuildScript|TestEngineerBrowserFrameworkTicketEvidenceRejectsSyntaxOnlyBuildScript|TestEngineerBrowserFrameworkTicketEvidenceRejectsCopyOnlyBuildScript|TestEngineerBrowserFrameworkTicketEvidenceRequiresProductSmoke|TestEngineerPostValidationAllowsMissingBrowserBuildAfterCommit|TestEngineerPostValidationAllowsMissingBrowserSmokeAfterBuild|TestEngineerBrowserFrameworkTicketEvidenceBlocksMissingNamedExports|TestEngineerBrowserFrameworkTicketEvidenceBlocksClassicScriptModuleEntry|TestEngineerBrowserFrameworkTicketEvidenceBlocksPhaserExternalInViteConfig|TestEngineerBrowserFrameworkTicketEvidenceBlocksMissingPhaserImport|TestEngineerBrowserFrameworkTicketEvidenceBlocksMissingLocalExportImport|TestShellExecPolicyBlocksNodeCheckHTML|TestRecordSessionToolOutcomeTreatsNodeCheckHTMLAsProcedureFailure|TestRecordSessionToolOutcomeTracksBrowserProductSmokeNodeEval|TestQABrowserFrameworkApprovalBlocksPhaserLifecycleDefect|TestQABrowserFrameworkApprovalRequiresProductSmoke|TestDogfoodBrowserFrameworkApprovalRequiresProductSmoke|TestReviewTerminalEvidenceForBrowserFrameworkWithoutBuildRequestsChanges'` and `go test ./internal/scanner -run 'TestInit_success|TestInitGeneratedRolePrompts'`
- F-005-S043: `go test ./internal/tools -run 'TestCOOCompletionRequiresProductSpecificFeatureContract|TestCOOCompletionAllowsProductSpecificContractWithBusinessLogicLanguage|TestCOOCompletionRequiresBriefCapabilitiesInScenarioSchedule|TestCOOCompletionRejectsCollapsedProductCapabilityScenario|TestCOOCompletionIgnoresValidationEvidenceAndAcceptsControlSynonym|TestCOOCompletionRejectsBriefCapabilitiesInOutOfScope|TestCOOCompletionAllowsAdvancedOutOfScopeQualifierForCoveredBasicCapabilities|TestCTOTicketCreateRequiresBriefCapabilitiesInScenarioSchedule|TestCTOTicketCreateRequiresEarliestUncoveredFeatureScenario|TestCTOTicketCreateAllowsScenarioGroupStartingWithEarliestUncoveredScenario'`
- F-005-S046: `go test ./internal/tools -run 'TestEngineerFileWriteAllowsPhaserValidationHelperProbeStrings|TestBrowserFrameworkSourceFindingsIgnorePhaserValidationHelper'`
- F-005-S047: `go test ./internal/tools -run TestCTOTicketCreateInfersPendingHandoffScenarios`
- F-005-S048: `go test ./internal/tools -run TestPlanningRoleShellExecPolicyBlocksMutatingCommands`
- F-005-S049: `go test ./internal/tools -run TestCOOCompletionIgnoresActiveGoalNonGoalsAndOperationalConstraints`
- F-005-S050: `go test ./internal/tools -run 'TestCOOCompletionIgnoresProjectNameTokensFromBriefHeadings|TestCapabilityMatchingIgnoresIncludingAndDetectionGlue'`
- F-005-S051: `go test ./internal/tools -run TestCOOCompletionAllowsHighScorePersistenceOutOfScope`
- F-005-S052: `go test ./internal/tools -run TestReviewHTTPProbeBeforeServerStartIsProcedureFailure`
- F-005-S053: `go test ./internal/tools -run TestEngineerReworkUsesDispatchTicketBeforeOlderDoneTicket`
- F-005-S056: `go test ./internal/tools -run 'TestCTOTicketCreateBlocksCDNShapeForPhaserBrief|TestQABrowserFrameworkApprovalRequiresProductSmoke|TestReviewShellExecPolicyAllowsTrackedBackgroundKill'`
- F-005-S057: `go test ./internal/tools -run TestCOOCompletionIgnoresGenericGameplayMechanicsGoalHeading`
- F-005-S058: `go test ./internal/tools -run TestCOOCompletionDoesNotTreatMobileTouchControlsAsMovementDescoped`
- F-005-S059: `go test ./internal/tools -run TestEngineerPostValidationBrowserEvidenceBlocksDirtyExploration`
- F-005-S060: `go test ./internal/tools -run TestEngineerPostBuildBrowserFrameworkBlocksSmokeSubstitutesWhileDirty`
- F-005-S061: `go test ./internal/tools -run TestCOOCompletionAllowsOutOfScopeIntroAndAdvancedScoringSystems`
- F-005-S062: `go test ./internal/tools -run TestCapabilityMatchingIgnoresIncludingAndDetectionGlue`
- F-005-S063: `go test ./cmd/mars -run TestRunCommandFoundationMaintainer` and `go test ./internal/scanner -run TestInit_success`
- F-005-S064: `go test ./internal/tools -run 'TestCOOCompletionTreatsFeatureDocReferenceAsCitation|TestCOOCompletionMissingFeatureDocCapabilityNamesCapabilityNotCitation|TestCOORepeatedFeatureSpecificityBlockReturnsRepairGuidance|TestRecordSessionToolPolicyFailureSeparatesRepeatedPolicyKeys|TestPolicyFailureRepairFeedbackGuidesUnresolvedShellValidationLane'`
- F-005-S065: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksStaticProductSmoke|TestRecordSessionToolOutcomeDoesNotTreatReservedPortHTTPAsStaticProductSmoke|TestRecordSessionToolOutcomeDoesNotCountCannedNodeEvalValidation|TestRecordSessionToolOutcomeTracksRuntimeToolErrorAsOutstandingFailure|TestPackageWriteBlocksReservedHarnessPortForStaticWeb|TestPackageWriteBlocksCannedSmokeScriptForStaticWeb|TestEngineerPostValidationAllowsMissingStaticSmokeAfterCommit|TestEngineerStaticBrowserTicketEvidenceRequiresStaticSmoke|TestEngineerStaticBrowserTicketEvidenceAllowedAfterStaticSmoke|TestReviewApprovalBlocksStaticBrowserWithoutStaticSmoke'`
