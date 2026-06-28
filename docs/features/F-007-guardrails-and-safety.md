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
10. F-007-S010 - Untracked repo-local compiled binaries named after the repo or Go module may be removed as generated build artifacts, and blast-radius errors for those artifacts name the exact cleanup command without opening arbitrary deletion.
11. F-007-S011 - Validation build commands are blocked before execution when their explicit output path would create a compiled artifact inside the target repo.
12. F-007-S012 - New repo-root validation, smoke, and probe helpers are blocked so scratch validation does not become committed product noise.
13. F-007-S013 - Source, static asset, workflow, and test writes require valid top-of-file documentation metadata.
14. F-007-S014 - Dogfood finding tickets freeze further validation until the target-owned finding is committed and handed off.
15. F-007-S027 - Failing Engineer test/build evidence blocks runtime side paths, ticket evidence, ticket completion, and product commits until repaired.
16. F-007-S028 - CTO product-file writes are blocked so ticket shaping cannot become implementation.
17. F-007-S029 - Failing Engineer test/build repair allows same-lane validation after bounded repair writes and blocks helper-script workarounds.
18. F-007-S030 - Simple `cd <dir> && <test/build>` shell validation is classified without allowing arbitrary shell wrappers.
19. F-007-S031 - QA and Security clean review evidence forces terminal disposition instead of late tool drift.
20. F-007-S032 - QA and Security validation-procedure failures stay separate from target product failures.
21. F-007-S033 - Safe argv-shaped `cd <dir> && <test/build>` validation is normalized while arbitrary argv shell syntax stays blocked.
22. F-007-S034 - Clear CLI input-validation probes do not open the runtime repair guardrail.
23. F-007-S035 - Engineer command-procedure failures stay out of the product repair guardrail.
24. F-007-S036 - Test/build repair cleanup is limited to same-job test-like files.
25. F-007-S037 - Test/build repair source writes are limited to the failed package scope when known.
26. F-007-S038 - CLI secret scanning and optional hooks block credential leaks without printing secret values.

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

Given `.harness/.env.local` contains local cloud provider credentials
When scanning committed or staged files
Then the file is treated as ignored local state and the scan still blocks any secret-like value that appears in committed harness files, docs, logs, traces, or JSON reports

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

Given Engineer has an ordinary product ticket waiting in `docs/tickets/backlog/` and no in-progress ticket
When it calls `shell_exec` for anything except the backlog-to-in-progress claim move
Then the tool guardrail blocks the shell call and names the exact `git mv` claim path before further discovery, validation, or implementation shell commands can run

Given Dogfood is running observation-first validation
When it attempts to write product source, package manifests, lockfiles, config, or harness scaffold
Then the tool guardrail blocks the mutation while still allowing bounded evidence reports under `docs/reports/dogfood/`

Given Dogfood has created a target-owned finding ticket in the current run
When it attempts additional validation through `shell_exec` or tries to create another ticket
Then the tool guardrail blocks the call and tells Dogfood to commit the ticket, push if possible, and record disposition before continuing

### F-007-S010: Generated Build Artifact Cleanup

Given a validation command creates an untracked root-level compiled binary named after the repository or the root Go module
When an agent removes that exact binary with `shell_exec` `rm <artifact>` or `unlink <artifact>`
Then policy allows the cleanup only when the file is untracked, root-level, binary-looking, and named after the repo or module basename, while ordinary file removal and recursive deletion remain blocked

### F-007-S011: Validation Build Output Outside Repo

Given a role calls `shell_exec` for `go build` during validation
When the output path is omitted or an explicit `-o <path>` resolves inside the target repository
Then tool policy blocks the command before process execution, no compiled artifact is created, and the error instructs the role to use `go test ./...` for compile validation or write validation binaries to an external temp path so repository diffs stay source-only

### F-007-S038: CLI Secret Scan And Hooks

Given an operator runs `mars guardrails secret-scan --repo <path>`
When staged, tracked, or working-tree files contain common credential patterns
Then the command exits non-zero, names the file, line, and pattern, and redacts the matched value in text and JSON output

Given an operator runs `mars guardrails install-hooks --repo <path>`
When the repository has no managed MARS pre-commit block
Then the hook is installed idempotently and invokes `mars guardrails secret-scan --repo <path> --staged`
And existing non-MARS hook content is preserved

Given a role wraps validation in shell syntax such as `mkdir -p bin && go build -o bin/app ...`
When any command segment would write a Go build output inside the target repository
Then tool policy blocks the shell command before process execution so the build artifact is not created before blast-radius validation

Given a role executes a temp validation binary such as `<validation-root>`
When that path was not produced by a successful `go build -o <validation-root> ...` command earlier in the same role session
Then tool policy blocks the execution as stale evidence and instructs the role to rebuild the external validation binary before running it

Given Engineer has an unresolved unexpected runtime validation failure in the current role session
When it attempts to move, write, commit, or successfully dispose a product ticket as complete
Then tool policy blocks completion before the ticket state can claim done evidence that contradicts the current validation transcript

Given Engineer tries to repair that blocker by rerunning the failed command with `expected_exit_code`
When the failed command was already observed as an unexpected runtime failure
Then the guardrail keeps the blocker open; Engineer must make the command pass instead of relabeling acceptance failure as expected behavior

Given Engineer forgets `expected_exit_code` on an obvious missing-argument runtime probe
When Engineer reruns that exact no-argument command with matching `expected_exit_code`
Then the guardrail permits the correction because the product behavior is an expected negative path, not a failed positive acceptance path

Given an unresolved runtime blocker came from an intentional missing-required-input probe
When guardrails block other runtime probes or completion attempts
Then the message names the exact-command `expected_exit_code` correction and keeps positive acceptance failures on the implementation-repair path

Given a direct runtime validation command exits 0 but prints error-shaped stderr
When Engineer tries to continue completion based on that probe
Then the guardrail treats the probe as failed evidence and requires a clean exact rerun before completion can proceed

Given Engineer reruns runtime probes after an unexpected runtime validation failure
When no post-failure implementation edit has happened
Then the guardrail blocks the runtime probe and routes the role back to file inspection and file editing before the exact failed command can be retried

Given Engineer has claimed a product ticket and repeats no-op `shell_exec` placeholders before implementation
When no successful validation evidence exists in the current job
Then the guardrail blocks the repeated no-op and routes the role to read the ticket/feature contract and write product code or record a blocked disposition

Given a planning role attempts to create a new ticket by writing markdown directly under `docs/tickets/`
When `file_write` is blocked because new tickets must use `ticket_create`
Then the failed bypass is retained as unresolved ticket-creation state until a real `ticket_create` succeeds or the role records a blocked disposition

### F-007-S016: Exact Missing-Argument Correction Guardrail

Given Engineer has an unresolved missing-required-input runtime probe with a stored exact `expected_exit_code` correction
When Engineer attempts unrelated edits, commits, pushes, decisions, dependency sync, CLI mutation, tool creation, or persona creation
Then the guardrail blocks those actions and tells Engineer to run the exact correction next or record a blocked disposition with the blocker

### F-007-S017: Stale External Validation Artifact Guardrail

Given an external `<validation-root>` binary was built before Engineer edited source after a runtime failure
When Engineer tries to execute that binary again
Then the guardrail blocks the stale artifact and requires a fresh `go build -o <validation-root> ...` before rerun validation

### F-007-S018: In-Progress Ticket Evidence Guardrail

Given Engineer has not produced successful validation evidence in the current job
When it attempts to populate `evidence_links` or `verified_by` in a ticket under `docs/tickets/in-progress/`
Then the guardrail blocks the write before ticket metadata can claim unproven product evidence

Given Engineer has already produced successful validation evidence in the current job
When it writes concrete ticket evidence and verifier metadata
Then the guardrail allows the update so the ticket can move toward done with proof attached

### F-007-S019: Exact External Artifact Rebuild Guardrail

Given a reviewer attempts to execute `<validation-root>` without a same-session build
When the stale artifact guardrail blocks the command
Then the error names the exact `shell_exec argv` rebuild command instead of only describing the policy

Given the target is a root Go CLI
When the rebuild correction is generated
Then it uses `go build -o <validation-root> .` so the reviewer can recover without guessing a package path

### F-007-S020: Go Build Output Guardrail Gives Exact Recovery

Given a role runs `go build ./cmd/<name>` without `-o`
When the command would create a repo-local binary
Then the guardrail blocks it before artifact creation and names `shell_exec argv ["go","build","-o","/tmp/<name>-validation","./cmd/<name>"]`

Given a role runs `go build -o <unsafe-output> <target>`
When the output path is inside the repo or outside `/tmp` without the `-validation` suffix
Then the guardrail preserves `<target>` in the corrected external validation build command

### F-007-S021: Missing-Input Guardrail Allows Repair After Repro

Given Engineer triggers the missing-input runtime guardrail
When the exact `expected_exit_code` correction still fails
Then the guardrail permits implementation `file_write` so the defect can be repaired

Given the missing-input runtime failure is still outstanding
When Engineer attempts unrelated probes or lifecycle completion
Then the guardrail blocks those actions until the exact runtime command is repaired

### F-007-S022: Runtime Repair Clears Same-Command Guardrail Debt

Given a runtime guardrail has recorded repeated failures for the same command
When an exact rerun of that command succeeds
Then the guardrail clears all same-command outstanding counts for the current job

Given a different runtime command has failed
When only the first command succeeds
Then the different command remains blocked until its exact repair evidence passes

### F-007-S012: Repo-Root Validation Script Prevention

Given a role attempts to create a new root-level validation shell script such as `validate.sh`
When the write is not updating an existing project-owned script
Then tool policy blocks the file before creation and instructs the role to use existing tests, direct build/run/curl evidence, or intentional durable validation code under a tests directory

Given a role calls an external `timeout` or `gtimeout` validation command through `shell_exec`
When tool policy rejects the non-portable wrapper before process execution
Then telemetry classifies the event as a guardrail block, not a retryable tool timeout, so the runtime records foundation evidence without enqueueing duplicate retry work

Given a role validates a web service or watcher through `shell_exec`
When it runs a likely long-running command in the foreground, including an HTTP `go run`, `npm start`, `npm run dev`, `python -m http.server`, `uvicorn`, `vite`, or `next`
Then tool policy blocks the command before process execution and requires managed background mode, a separate readiness probe, and tracked PID cleanup so validation does not spend turns on foreground timeouts

### F-007-S013: Source Writes Carry Documentation Metadata

Given a role writes source, static asset, workflow, or test code through `file_write`
When the content lacks top-of-file `MarsDocSync` docs metadata or references a missing feature-contract path
Then tool policy blocks the write before the source file is created or replaced and tells the role to reference the existing canonical documentation path instead of a scenario ID path

### F-007-S023: Ticket Done Moves Are Lifecycle-Only

Given Engineer has uncommitted non-ticket product changes
When it tries to move an in-progress ticket to `docs/tickets/done/`
Then tool policy blocks the move and instructs Engineer to commit implementation, tests, docs, package files, and config before closing the ticket lifecycle

Given only ticket evidence or lifecycle files are dirty
When the ticket has required BDD and evidence metadata
Then the `git mv` to `docs/tickets/done/` can proceed as a lifecycle-only commit

### F-007-S024: Reviewer No-Op Guardrails Preserve Review Progress

Given a reviewer has already produced passing validation evidence
When it tries to use a no-op shell placeholder as a wait or separator
Then the guardrail blocks the placeholder and names the terminal disposition path instead of asking for more validation

Given the no-op block came from policy before shell execution
When session accounting records the policy failure
Then the job still tracks it as a no-op failure so loop and scoring evidence can distinguish review dithering from product failure

### F-007-S025: Runtime Failure Guardrail Blocks Commit Bypass

Given Engineer has an unresolved positive runtime acceptance failure
When it attempts to commit product work before the exact failed command passes
Then the guardrail blocks the commit and instructs Engineer to keep the failed implementation uncommitted while repairing source and rerunning the exact failure

Given Engineer tries to wrap a different runtime probe in shell syntax while that failure is outstanding
When the command is not the same stale validation artifact rebuild or exact repaired rerun
Then the guardrail blocks the shell call as a side path rather than recording it as new progress

### F-007-S026: Reviewer No-Op Guardrail Becomes Terminal-Only

Given a reviewer has successful validation evidence and policy blocks a no-op shell placeholder
When the same job attempts another shell command or non-terminal tool instead of disposition
Then the guardrail blocks that tool call and instructs the role to use `job_disposition_record`

Given QA reviews a Go target with source files but no `_test.go` files
When a no-op guardrail asks for terminal disposition
Then the guardrail guidance names `changes_requested` and missing tests instead of approval

### F-007-S027: Failed Test/Build Evidence Blocks False Progress

Given Engineer has an unresolved failing test or build command in the current job
When it attempts runtime probes, unrelated shell commands, ticket evidence writes, ticket done moves, successful disposition, or a product commit
Then the guardrail blocks the action and names the failing validation lane that must pass after bounded repair

Given Engineer has edited source or tests after the failed test/build command
When it reruns a same-lane test/build command successfully
Then the guardrail clears the outstanding repair blocker and normal product lifecycle actions may continue

### F-007-S028: CTO Product Writes Are Blocked

Given CTO is running technical planning or ticket shaping
When it attempts to write product implementation, package/module, README usage, source, test, build, config, or root product files
Then the guardrail blocks the write before mutation and instructs CTO to use `ticket_create` plus Engineer delivery

Given CTO writes bounded technical planning artifacts
When the path is under `docs/design-docs/`, `docs/reports/strategy/`, or `docs/goals/observations.md`
Then the guardrail allows the write because it remains technical planning rather than product implementation

### F-007-S029: Test/Build Repair Blocks Workarounds

Given Engineer has an unresolved failing test command
When it edits product source, tests, fixtures, or package/build config
Then the guardrail allows a later recognized test command in the same validation lane even when the focused package path differs from the original command

Given Engineer has an unresolved failing build command
When it edits product source, tests, fixtures, or package/build config
Then the guardrail allows a later recognized build command in the same validation lane even when the build output or package path differs from the original command

Given that repair lane is unresolved
When Engineer attempts to write a helper verification script or root scratch probe
Then the guardrail blocks the write and keeps repair inside product source, tests, fixtures, or package/build config

Given Engineer is working on a browser-framework product ticket
When it tries to create a new repo-root helper such as `validate-game.js`, `browser-smoke.js`, or `probe-ui.js`
Then the guardrail blocks the file as scratch validation noise
And the role must use direct `shell_exec` evidence or create intentional durable validation under `scripts/` or `tests/`

### F-007-S030: Simple CD Validation Shell Is Recognized

Given Engineer has an unresolved failing test/build repair lane
When it calls `shell_exec` with a shell command shaped exactly like `cd <dir> && <recognized test/build command>`
Then the guardrail classifies the right-hand test/build command as same-lane validation and allows it after bounded repair edits

Given Engineer has an unresolved failing test/build repair lane
When it calls `shell_exec` with arbitrary shell control syntax, multiple chained operations, pipes, redirection, substitutions, cleanup, runtime probes, or ticket moves
Then the guardrail blocks the command as a side path

### F-007-S031: Review Terminal Boundary Blocks Late Tools

Given QA or Security has successful read and validation evidence with no outstanding validation failure
When the runtime marks terminal disposition as required
Then subsequent non-terminal tools are rejected and the reviewer is instructed to call `job_disposition_record`

Given the reviewer calls `job_disposition_record`
When the disposition satisfies review validation and DocSync policy
Then the terminal tool remains available

### F-007-S032: Reviewer Procedure Failures Do Not Poison Product Review

Given QA or Security runs a Go validation build or test command with an obvious procedure mistake such as `go build cmd/<name>` without `./` or `go build .` in a CLI repo whose entrypoint lives under `cmd/*`
When the command fails with the corresponding Go package-target error
Then the session records a validation-procedure failure instead of a target validation failure, leaves corrected validation commands available, and still requires successful validation before approval

Given QA or Security observes a real compile error, failing test, or unexpected runtime validation failure
When it attempts more shell validation
Then the existing review failure guard still blocks additional shell work and routes to `job_disposition_record` with `changes_requested`

### F-007-S033: Simple CD Validation Argv Is The Only Shell-Syntax Exception

Given a role sends `shell_exec` argv tokens shaped exactly like `cd <dir> && <recognized test-or-build command>`
When the command has no redirection, pipes, substitutions, background operators, cleanup, or non-validation action
Then the tool normalizes it into the shell-command validation path so focused package validation can run

Given a role sends any other shell syntax through argv mode
When the tool validates the command
Then the command is rejected before execution and the role is told to use `shell_command` or purpose-built tools instead

### F-007-S034: CLI Input-Validation Probes Stay Out Of Repair Lane

Given Engineer runs a direct CLI validation binary or runtime entrypoint with no required product input, with a deliberately invalid input such as `invalid`, or with surplus positional arguments
When the command exits non-zero with clear required-input, usage, invalid-input, or surplus-argument output and no panic, traceback, exception, runtime-error, or segmentation-fault marker
Then the guardrail records expected negative-path evidence instead of opening an unresolved runtime repair blocker

Given the same probe exits with crash-like output or a positive acceptance command with valid input fails
When Engineer tries to commit, move a ticket to done, or record successful disposition
Then the existing runtime repair guardrail still blocks completion until the behavior is repaired and the exact command passes

### F-007-S035: Engineer Procedure Failures Stay Out Of Repair Lane

Given Engineer runs a Go validation build or test command with an obvious procedure mistake such as `go build cmd/<name>` without `./`
When the command fails with the corresponding Go package-target error
Then the session records a validation-procedure failure instead of opening the product repair guardrail, and a corrected validation command remains available in the same job

Given Engineer observes a real compile error, failing test, or unexpected runtime validation failure
When Engineer attempts unrelated validation, ticket completion, commit, or successful disposition
Then the existing product repair guardrail still blocks progress until the target-owned failure is repaired and validated

### F-007-S036: Same-Job Test Repair Cleanup

Given Engineer has an unresolved test/build failure and writes a bad test-like repair file in the same job
When Engineer removes that same path with non-recursive `rm` or `unlink`
Then the guardrail permits the cleanup so the role can continue same-lane repair and rerun the failing test/build command

Given Engineer tries to remove an unmarked test file, product source file, or recursive path while a test/build failure is unresolved
When the shell command is evaluated
Then the guardrail blocks the cleanup and keeps the role in source/test repair plus same-lane validation

### F-007-S009: Workspace Hygiene Gates

Given a repository has missing generated-directory ignore policy, tracked generated dependency output, dirty generated build output, large generated diffs, host OS metadata noise, or deletion state
When an agent job starts or a dependency install/fetch is requested
Then generated paths and host OS metadata are classified separately from implementation files, safely inferable missing ignore entries are committed as a `.gitignore`-only repair before model loading, `workspace_hygiene` reports a deterministic recipe with `recipe_id` and `next_action` for non-repairable cases, raw package-manager mutation through `shell_exec` is blocked, `dependency_sync` performs package-manager work only after preflight passes, and `.DS_Store`-style noise does not force a completed product ticket to reopen before terminal disposition

### F-007-S037: Test-Build Repair Writes Stay In Failed Scope

Given Engineer has an unresolved test/build failure from a narrow package target
When it writes a source or test file outside that package while the failure is still outstanding
Then policy blocks the write and tells Engineer to repair the failed test/build scope instead of creating alternate entrypoints

Given the write targets the failed package scope or package/build configuration
When Engineer reruns same-lane validation successfully
Then the guardrail allows the repair lane to clear

### F-007-S038: Unresolved Test-Build Guidance Repeats Failure Output

Given Engineer has an unresolved test/build failure
When Engineer attempts unrelated shell validation, ticket evidence, ticket completion, commit, or successful disposition
Then guardrails block the action and include the latest failing stdout/stderr or exit code in the message

Given the failing assertion is aligned with the ticket, README, or BDD contract
When Engineer receives the guardrail guidance
Then the message directs Engineer to repair implementation behavior instead of deleting or weakening the test

### F-007-S039: Missing Go Module Bootstrap Is Repair Work

Given Engineer has an unresolved failing Go test/build command whose latest output says Go cannot find a main module
And the target repo has no `go.mod`
When Engineer runs `go mod init <module>`
Then the guardrail allows the command as package/build configuration repair

Given the repo already has `go.mod` or the latest failure output does not prove a missing Go module
When Engineer runs `go mod init <module>` while the repair lane is unresolved
Then the guardrail blocks the command as unrelated shell work

### F-007-S040: Dependency Mutation And Test Cleanup Preserve Evidence

Given an agent calls `shell_exec` with raw `go get <module>`
When shell policy evaluates the command
Then the guardrail blocks it as dependency mutation and points to `dependency_sync`

Given Engineer has an unresolved test/build assertion failure
And a test file was written by the same job
When Engineer tries to remove that test file
Then the guardrail blocks the cleanup and keeps assertion evidence intact

Given Engineer has an unresolved duplicate/generated-test failure
And a test-like file was written by the same job
When Engineer removes that test-like file non-recursively
Then the guardrail allows the cleanup before same-lane validation reruns

### F-007-S041: Engineer Runtime Validation Converges Without No-Op Waits

Given Engineer has successful validation evidence and dirty implementation or ticket work
When it tries to use an empty `shell_exec` argv or single `:` placeholder as a wait
Then the guardrail blocks the no-op before process execution and directs Engineer to stop tracked background validation, commit the dirty files, update ticket evidence, move the ticket to done, push, and record `job_disposition_record`

### F-007-S042: Capability Guards Ignore Readable Outcome Glue

Given a product brief says the first product should let a user see, play, or use a useful outcome
And the feature contract breaks out the concrete requested behaviors into scenario schedule entries or scenario headings
When COO records a successful planning disposition or CTO creates a ticket
Then the guardrail ignores glue words such as `see`, `useful`, `usable`, and `playable` as standalone capabilities
And still requires concrete behavior words such as workflow, validation, calculation, state transition, or product controls to be covered or deliberately descoped

### F-007-S043: Capability Guards Strip Product Labels Dynamically

Given a target brief has a project title or branded product name
And brief-derived capability text repeats that label next to concrete behaviors
When COO records a successful planning disposition or CTO creates a ticket
Then the guardrail strips project-label tokens from required capability matching only when concrete behavior words remain
And the guardrail does not add validating demo names or product-object nouns as global capability stopwords or synonyms

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
- F-007-S010: `go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'`
- F-007-S011: `go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|BlocksGoBuildOutputInShellCommandSegmentBeforeArtifact|AllowsGoBuildOutputOutsideRepo|NoopArgsNotMaskedByDirtyArtifact)'` and `go test ./internal/tools -run 'TestExternalValidationArtifactMustBeBuiltInSameSession|TestRecordSessionToolOutcomeTracksValidationArtifactBuildAndRun|TestEngineerCannotCompleteTicketWithUnresolvedRuntimeValidationFailure|TestRecordSessionToolOutcomeEngineerExpectedExitDoesNotRepairUnexpectedRuntimeFailure'`
- F-007-S014: `go test ./internal/tools -run 'TestDogfoodUncommittedFindingBlocksFurtherValidationAndTickets|TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation'`
- F-007-S015: `go test ./internal/tools -run 'TestSuccessfulDispositionBlocksUnresolvedTicketCreationFailure|TestRecordSessionToolOutcomeTracksTicketCreationFailures'`
- F-007-S016: `go test ./internal/tools -run TestEngineerMissingArgumentRuntimeFailureBlocksUnrelatedMutation`
- F-007-S017: `go test ./internal/tools -run TestExternalValidationArtifactMustBeRebuiltAfterRuntimeFailureEdit`
- F-007-S018: `go test ./internal/tools -run 'TestEngineerTicketEvidenceWrite(RequiresValidation|AllowedAfterValidation)'`
- F-007-S019: `go test ./internal/tools -run TestExternalValidationArtifactMustBeBuiltInSameSession`
- F-007-S020: `go test ./internal/tools -run 'TestShellExecBlocks(DefaultGoBuildForCmdPackageWithExactCorrection|GoBuildOutputInsideRepoBeforeArtifact|GoBuildOutputOutsideRepoWithoutValidationSuffix)'`
- F-007-S021: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksFailedMissingArgumentCorrectionAttempt|TestEngineerMissingArgumentRuntimeFailureAllowsImplementationEditAfterCorrectionAttempt'`
- F-007-S022: `go test ./internal/tools -run TestRecordSessionToolOutcomeExactSuccessClearsRepeatedRuntimeFailures`
- F-007-S023: `go test ./internal/tools -run 'TestShellExecPolicy.*TicketDoneMove'`
- F-007-S024: `go test ./internal/tools -run 'TestReviewShellExecPolicyRoutesPostValidationNoopToDisposition|TestRecordSessionToolPolicyFailureTracksNoopFailures'`
- F-007-S025: `go test ./internal/tools -run 'TestEngineerRuntimeFailureBlocks(ShellWrapperBypass|ValidationUnrelatedShell)|TestEngineerPositiveRuntimeFailureBlocksImplementationCommit|TestEngineerRuntimeFailureAllowsStaleValidationArtifactRebuild'`
- F-007-S026: `go test ./internal/tools -run 'TestReviewShellExecPolicyRoutesNoTestGoRepoToChangesRequested|TestReviewTerminalDispositionRequiredBlocksFurtherShellExec|TestRecordSessionToolPolicyFailureTracksNoopFailures'`
- F-007-S027: `go test ./internal/tools -run 'TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation|TestEngineerFailingTestBlocksCommitTicketEvidenceAndDisposition|TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane'`
- F-007-S028: `go test ./internal/tools -run TestCTOFileWritePolicyAllowsTechnicalPlanningAndBlocksImplementation`
- F-007-S038: `go test ./internal/safety ./cmd/mars -run 'TestGuardrailsSecretScan|TestGuardrailsInstallHooks'`
- F-007-S029: `go test ./internal/tools -run 'TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation|TestFileWriteBlocksNewRootScratchProbe'`
- F-007-S030: `go test ./internal/tools -run 'TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation|TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane'`
- F-007-S031: `go test ./internal/agent -run TestRun_reviewEvidenceReminder`
- F-007-S032: `go test ./internal/tools -run TestRecordSessionToolOutcomeReviewer`
- F-007-S033: `go test ./internal/tools -run 'TestShellExecNormalizesSimpleCdValidationArgv|TestShellExecArgvRejectsShellSyntax'`
- F-007-S034: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTreatsMissingArgumentCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeTreatsInvalidInputCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeTreatsSurplusArgumentCLIProbeAsExpectedFailure|TestRecordSessionToolOutcomeStillTreatsPositiveInputFailureAsUnexpected'`
- F-007-S035: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerGoBuildProcedureFailureDoesNotPoisonRepairLane`
- F-007-S036: `go test ./internal/tools -run TestEngineerFailingTestAllowsSameJobRepairTestFileRemoval`
- F-007-S037: `go test ./internal/tools -run TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation`
- F-007-S038: `go test ./internal/tools -run TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane`
- F-007-S039: `go test ./internal/tools -run TestEngineerFailingTestAllowsMissingGoModuleBootstrap`
- F-007-S040: `go test ./internal/tools -run 'TestShellPolicyBlocksRawDependencyMutationCommands|TestEngineerFailingTestBlocksSameJobTestRemovalForAssertionFailure|TestEngineerFailingTestAllowsSameJobRepairTestFileRemoval'`
- F-007-S041: `go test ./internal/tools -run TestEngineerPostValidationDirtyNoopBlocksBeforeGenericNoop`
- F-007-S042: `go test ./internal/tools -run TestCOOCompletionAcceptsOutcomeGlueWhenCapabilitiesAreBrokenOut`
- F-007-S043: `go test ./internal/tools -run TestCOOCompletionIgnoresProjectNameTokensFromBriefHeadings`
