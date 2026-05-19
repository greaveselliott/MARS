# F-007: Guardrails And Safety

- Feature ID: F-007
- Goals: G-001, G-003
- Status: partially-passing
- Owner: QA

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-007-S001 - Hard guardrails block violating tool actions before mutation.
2. F-007-S002 - Advisory guardrails enter context without blocking execution.
3. F-007-S003 - Guardrail rules are scoped, deduplicated, stale-checked, and regex-validated.
4. F-007-S004 - File tools and shell tools stay inside the repo root.
5. F-007-S005 - Secret scanning blocks writes that would introduce credential material.
6. F-007-S006 - Sandbox execution enforces working directory, timeouts, and process isolation.
7. F-007-S007 - Emergency stop is available from runtime and dashboard controls.
8. F-007-S008 - Blast-radius limits are checked before commit and push workflows.
9. F-007-S009 - Workspace hygiene blocks generated dependency/build churn before model work and package-manager mutation.

## Scenarios

### F-007-S001: Hard Guardrail Block

Given a role attempts a mutating action that violates a hard guardrail
When the tool executor evaluates the action
Then the action is blocked before mutation and the result explains the violated rule

### F-007-S002: Advisory Prompting

Given advisory guardrails apply to a role or path
When context is assembled
Then advisory guidance is included for the role without blocking all execution

### F-007-S003: Rule Hygiene

Given guardrail definitions include scopes, regexes, dates, and advisory text
When the engine loads them
Then invalid regexes are rejected, stale rules are reported, and duplicate advisory guidance is collapsed

### F-007-S004: Repo-Root Containment

Given a tool call references a file path or shell working directory
When path resolution occurs
Then nested repo paths are allowed and absolute paths or parent escapes are rejected unless explicitly supported by the command contract

### F-007-S005: Secret Write Block

Given a tool would write AWS keys, GitHub tokens, private keys, password URLs, or generic API keys
When the write is evaluated
Then secret scanning blocks the mutation and reports the finding

### F-007-S006: Sandbox Limits

Given a shell command runs under the sandbox
When working directory, timeout, process group, or resource limits apply
Then the sandbox enforces them and returns actionable failure state

### F-007-S007: Emergency Stop

Given active workers or dashboard controls are available
When emergency stop is invoked
Then all registered stop callbacks execute and failures are collected rather than hidden

### F-007-S008: Blast Radius Gates

Given a role is about to commit or push changes
When the safety layer evaluates the diff and configured limits
Then excessive file count, line count, or forbidden path changes are blocked or require explicit escalation, while root dependency lock/checksum files remain git-visible and secret-scanned without counting generated line churn as source-file blast radius

Given a role moves a ticket between `docs/tickets/backlog/`, `docs/tickets/in-progress/`, `docs/tickets/in-review/`, and `docs/tickets/done/`
When the same ticket ID appears as a new, staged, or already-present ticket markdown file in another lifecycle directory
Then the deletion side of that ticket lifecycle move does not count as a forbidden file deletion, while unpaired ticket deletions and arbitrary file deletions remain blocked

Given a role moves or writes a feature ticket under `docs/tickets/done/`
When required BDD evidence fields such as `evidence_links` or `verified_by` are empty
Then tool policy blocks the mutation before the post-run ticket gate and reports the missing evidence fields

Given a role copies a ticket into `docs/tickets/done/` or writes a done-ticket copy while the same ticket ID remains in another lifecycle directory
When tool policy evaluates the mutation
Then the mutation is blocked and the role is told to use one `git mv` lifecycle transition instead of copy-and-delete cleanup

Given a role writes a feature contract
When the file contains duplicate scenario heading IDs such as two `F-001-S001` headings
Then tool policy blocks the write, names the duplicate heading line numbers, tells the role to revise the existing scenario instead of appending another duplicate ID, and clarifies that Scenario Schedule list references do not count as duplicate headings

Given a successful dispatch disposition is being recorded
When the target worktree contains only runtime-managed `.harness/learnings.yaml` convention metadata
Then the clean-handoff guardrail allows the disposition and the server may commit that runtime-only metadata after the job, but still blocks if any product, ticket, documentation, or source path is dirty alongside that runtime metadata

Given a planner role has a narrow ownership boundary
When COO tries to create product implementation files or run a mutating shell command before CTO ticketing
Then the tool guardrail blocks the action and leaves implementation to ticket-backed Engineer work

Given Dogfood is running observation-first validation
When it attempts to write product source, package manifests, lockfiles, config, or harness scaffold
Then the tool guardrail blocks the mutation while still allowing bounded evidence reports under `docs/reports/dogfood/`

### F-007-S009: Workspace Hygiene Gates

Given a repository has missing generated-directory ignore policy, tracked generated dependency output, dirty generated build output, large generated diffs, or deletion state
When an agent job starts or a dependency install/fetch is requested
Then generated paths are classified separately from implementation files, safely inferable missing ignore entries are committed as a `.gitignore`-only repair before model loading, `workspace_hygiene` reports a deterministic recipe with `recipe_id` and `next_action` for non-repairable cases, raw package-manager mutation through `shell_exec` is blocked, and `dependency_sync` performs package-manager work only after preflight passes

## Out of Scope

- AST-level semantic policy enforcement in v1.
- Silent override of hard guardrails.
- Guaranteeing safety for external commands outside the configured sandbox.

## Descoped Scenarios

None.

## Evidence

- F-007-S001: `go test ./internal/guardrails -run TestEngine_hardRuleBlocksViolation`
- F-007-S002: `go test ./internal/guardrails -run TestEngine_advisoryInPrompt`
- F-007-S003: `go test ./internal/guardrails`
- F-007-S004: `go test ./internal/tools -run TestRoot`
- F-007-S005: `go test ./internal/safety -run TestScanForSecrets` and `go test ./internal/tools -run TestExecutor_secretScannerBlocksFileWrite`
- F-007-S006: `go test ./internal/sandbox`
- F-007-S007: `go test ./internal/safety -run TestEmergencyStop` and `go test ./internal/dashboard -run TestDashboard_emergencyStop`
- F-007-S008: `go test ./internal/safety -run TestCheck`, `go test ./internal/tools -run TestValidateRepoDiffIgnoresGeneratedDependencyMetadataLineChurn`, `go test ./internal/tools -run TestJobDispositionPolicyIgnoresRuntimeLearningsOnlyDirtyState`, `go test ./internal/tools -run 'TestCOO(FileWrite|ShellExec)Policy|TestDogfoodFileWritePolicyBlocksProductMutation'`, and `go test ./internal/tools -run 'TestShellExecPolicy.*FeatureTicketDone(Move|Copy)|TestFileWritePolicyBlocks(DoneFeatureTicket|DuplicateFeatureScenario)'`
- F-007-S009: `go test ./internal/tools -run 'TestWorkspaceHygiene|TestDependencySync|TestShellPolicyBlocksRawDependencyMutationCommands'` and `go test ./internal/serve -run TestHandleJobFailedDoesNotRecoverDeterministicFailures`
