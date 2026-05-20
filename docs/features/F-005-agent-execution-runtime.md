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

## Scenarios

### F-005-S001: Role-Scoped Context Assembly

Given a role, repo, ticket index, guardrail scope, knowledge routes, and context budget
When a job prompt is assembled
Then high-priority role and workflow context is included, irrelevant guardrails are filtered, and lower-priority context is truncated before exceeding budget

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
Then the tool normalizes it into executable argv tokens so the run does not spend extra turns recovering from harmless formatting drift

Given a local model emits malformed `mars_harness_cli` args
When `args` is a JSON-encoded array string, a Python-style quoted list string, or a simple single command string
Then the tool normalizes the command before binary resolution so release, score, trust, and update flows do not fall back to stale PATH binaries because of harmless formatting drift

Given a local model emits malformed path-list fields for built-in tools
When tools such as `workspace_hygiene`, `git_diff`, or `git_commit` receive `paths` as a JSON-encoded array string or Python-style quoted list string
Then the tool normalizes the path list before policy checks so generic recovery works across project archetypes instead of only one demo path

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

### F-005-S011: Role Ownership Tool Boundaries

Given the Dogfood role is validating a target
When it attempts to write product source, package manifests, lockfiles, config, or harness scaffold with `file_write`
Then tool policy blocks the write and directs Dogfood to create target-owned tickets or write bounded evidence under `docs/reports/dogfood/`

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

## Out of Scope

- Parallel tool execution inside a single agent turn.
- Unbounded prompt stuffing.
- Treating chat transcript memory as the system of record.

## Descoped Scenarios

None.

## Evidence

- F-005-S001: `go test ./internal/context`
- F-005-S002: `go test ./internal/agent -run TestRun`
- F-005-S003: `go test ./internal/tools -run 'TestExecutor|TestShellExec'` and `go test ./internal/serve -run TestFoundationAcceptance`
- F-005-S004: `go test ./internal/agent -run TestRun_persistsTraceToSQLite`
- F-005-S005: `go test ./internal/agent -run 'TestRun_(max|token|wall|circle|empty)'`
- F-005-S006: `go test ./cmd/mars-harness -run 'TestRunStartServeExposeDebugAndLogFileFlags'` and planned broader E2E dogfood evidence
- F-005-S007: `go test ./internal/tools -run 'TestToolCreate|TestMarsHarnessCLI'`
- F-005-S008: `go test ./internal/tools -run TestPersonaCreate` and `go test ./internal/personas`
- F-005-S009: `go test ./internal/tools -run TestTicketCreate_dedupesIndependentFeatureTicketsForSameBDDScenario`
- F-005-S010: `go test ./internal/tools -run 'TestJobDispositionPolicy|TestEngineerDispositionPolicyRequiresTicketDoneBeforeSuccess|TestEngineerClaimPolicyRequiresInProgressBeforeProductMutation'`
- F-005-S011: `go test ./internal/tools -run 'TestCOO(FileWrite|ShellExec)Policy|TestDogfoodFileWritePolicyBlocksProductMutation'`
- F-005-S012: `go test ./internal/tools -run TestGitPush_noRemote`
- F-005-S013: `go test ./internal/scanner -run TestInit_success`
- F-005-S014: `go test ./internal/tools -run TestExecutor_toolHandlerHardTimeout`
- F-005-S015: `go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'` and `demo-api-run7` live evidence
