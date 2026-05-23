# F-006: Queue And Orchestration

- Feature ID: F-006
- Goals: G-001, G-002, G-003
- Status: partially-passing
- Owner: COO

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-006-S001 - Repositories are registered with per-repo database isolation.
2. F-006-S002 - Jobs enqueue, claim, lease, complete, cancel, and prune through SQLite.
3. F-006-S003 - Workers pause, resume, serialize per repo, and fire completion hooks.
4. F-006-S004 - Triggers match webhooks, schedules, and chains from manifests.
5. F-006-S005 - `start` initializes, registers, seeds, and runs a single-repo pipeline.
6. F-006-S006 - `serve` runs the multi-repo orchestrator, dashboard, webhook receiver, scheduler, workers, and control plane.
7. F-006-S007 - Recovery jobs are bounded, idempotent, self-healed when stale or duplicated, and suppressed for deterministic failures.
8. F-006-S008 - Product tickets stay ahead of intervention debt unless an active product ticket names the debt as a blocker.
9. F-006-S009 - Native Orchestrator surveys route unattended failure states into bounded jobs or intervention-debt tickets.
10. F-006-S010 - Dispatch-mode jobs record terminal dispositions, deterministic product handoffs route directly, and Orchestrator remains the ambiguous or governance fallback.
11. F-006-S011 - Review rework reuses the existing product ticket instead of creating duplicate planning work.
12. F-006-S012 - Feature tickets cannot move to done before required BDD evidence is populated.
13. F-006-S013 - Runtime failures stop as foundation telemetry instead of restarting product planning loops.
14. F-006-S014 - Product validation routes into release review instead of stopping before versioning.
15. F-006-S015 - Scheduled triggers skip same-repo same-role work that is already pending, claimed, or running.
16. F-006-S016 - Fresh bootstrap roles reuse canonical feature-contract paths and rewrite starter scenarios in place.
17. F-006-S017 - Successful implementation, review, validation, and release dispositions cannot approve source with failing DocSync evidence.
18. F-006-S018 - Dogfood-created target findings must be committed and handed off before further validation or duplicate ticket creation.
19. F-006-S019 - Review rework reopens done or in-review product tickets before Engineer product mutation or validation shell work.
20. F-006-S020 - Direct runtime validation commands count as completion evidence for lifecycle convergence.
21. F-006-S021 - Expected runtime error probes can be review evidence without blocking approval.
22. F-006-S022 - Post-validation completion directs Engineer away from shell retries and into ticket evidence updates.
23. F-006-S023 - Review validation failures become structured rework handoffs before more shell validation.
24. F-006-S036 - Engineer failing test/build evidence becomes a repair lane before runtime probes, evidence, ticket completion, or product commits can continue.
25. F-006-S037 - CTO ticket shaping cannot mutate product implementation files before Engineer claims the ticket.
26. F-006-S038 - Engineer test/build repair lanes accept focused same-lane validation after bounded repair edits.
27. F-006-S039 - Simple `cd <dir> && <test/build>` shell commands count as same-lane validation for repair classification.
28. F-006-S041 - Engineer test/build repair writes stay inside the failed package scope when one is known.
29. F-006-S042 - Engineer can remove duplicate test files it created earlier in the same job while repairing a failing test lane.
30. F-006-S045 - Release review waits for open product tickets and uncovered generated feature scenarios.
31. F-006-S046 - Fresh bootstrap CTO ticketing can seed a small ordered product backlog batch from early product scenarios so later product work is ready after the first slice.
32. F-006-S047 - Engineer closes one product ticket per job before claiming another product ticket.

## Scenarios

### F-006-S001: Repo Registration Isolation

Given a repo is registered without an explicit database path
When registration completes
Then queue, telemetry, schedule, and repo state use the repo-specific database path rather than contaminating another repo

### F-006-S002: SQLite Job Lifecycle

Given jobs are enqueued for a repo
When workers claim, complete, cancel, expire, or prune them
Then SQLite state changes are durable, idempotent where required, and queryable for recent job history

Given scheduled role work and dispatch handoff work are both pending for the same repo
When the worker claims the next job
Then seed/bootstrap jobs claim first, dispatch handoffs claim before scheduled work, and cron safety-net jobs cannot preempt the active product handoff chain

### F-006-S003: Worker Controls

Given the worker pool is running
When pause, resume, or completion hooks are used
Then new claims stop while paused, resume claims pending work, and success hooks enqueue downstream work

Given an operator stop or process shutdown cancels the worker context while a job is finishing
When the worker records the terminal job state
Then the queue uses a short finalization context to mark the running job completed or failed, so shutdown cannot strand a claimed job in `running`

### F-006-S004: Trigger Routing

Given manifests define webhook, schedule, or chain triggers
When events, cron ticks, or completed jobs occur
Then matching roles are enqueued and invalid manifests do not take down the router

Given a manifest schedule fires for a repo and role
When that same repo and role already has a pending, claimed, or running job
Then the scheduler records the skip and does not enqueue another same-role job for that repo, so periodic triggers cannot stack duplicate product workers behind an active lifecycle

### F-006-S005: Single-Repo Start

Given a user runs `mars-harness start --repo <path>`
When the target needs scaffolding or registration
Then the harness initializes if needed through the same committed generated-scaffold baseline used by `init`, `run`, `register`, and `scan`, registers the repo, seeds the CEO role, and runs with repo-scoped state

### F-006-S006: Serve Orchestrator

Given a user runs `mars-harness serve`
When the server starts
Then health, dashboard, webhook, scheduler, worker, recovery, and API control surfaces are available

### F-006-S007: Recovery Containment

Given a job fails in a recoverable or deterministic category
When recovery is enqueued or existing recovery jobs are stale
Then the system creates at most one active recovery job per repo/role key for transient failures, avoids recursive recovery, cancels duplicates, and records deterministic runtime failures as telemetry without same-role retry or dispatch loops

### F-006-S008: Ticket Claim Priority

Given tickets exist in in-progress, intervention-debt, and ordinary product backlog states
When engineer context is assembled
Then existing product in-progress work is prioritized first, ordinary product backlog stays ahead of intervention debt, and intervention debt remains visible without blocking product progress unless an active product ticket names it in `blocked_by`

### F-006-S009: Native Orchestrator Survey

Given queue, ticket, score, telemetry, and recent outcome signals exist
When the Orchestrator survey runs
Then stale tickets, blocked tickets, failed checks, dogfood failures, and no-op outcomes create bounded queue work, while runtime telemetry patterns and low scores stay quarantined as foundation telemetry unless deliberately materialized

Given an eligible in-progress ticket remains after a same-role runtime failure such as Engineer `max_turns`
When the Orchestrator survey runs within the runtime-failure cooldown window
Then ticket-owner routing pauses instead of immediately enqueueing another same-role ticket-delivery job, leaving the failure as foundation telemetry or operator retry evidence

Given a target repo has uncommitted non-runtime changes such as a Dogfood-created backlog ticket or evidence file
When the Orchestrator survey runs
Then autonomous survey routing pauses for that repo until the dirty target work is committed or cleaned, while runtime-only `.harness/learnings.yaml` remains non-blocking

### F-006-S010: Dispatch-Mode Orchestration

Given a generated target manifest uses `orchestration_mode: dispatch`
When a non-Orchestrator role completes successfully
Then it must record a terminal `job_disposition_record`, the agent loop stops after that terminal tool result, and the executor records a dispatch decision from manifest state, ticket state, loop guards, and the disposition rather than following a fixed role-to-role chain

Given a non-Orchestrator dispatch-mode role records a completed, approved, in-review, or no-work disposition with a deterministic route
When the route can be selected from `suggested_role`, `handoff.target_role`, `feedback.for_role`, `next_need`, or the default product validation spine
Then dispatch enqueues that target role directly without an Orchestrator LLM detour, while preserving role validation, ticket prerequisites, review-chain progression, and loop guards

Given a non-Orchestrator dispatch-mode role records a completed disposition whose `next_need` maps back to the same planning role
When no explicit structured target role points to a different owner
Then direct dispatch treats the role's completed status as evidence that local planning is done and routes to the default forward owner instead of enqueueing a same-role handoff loop

Given a non-Orchestrator dispatch-mode role records a no-work disposition whose `next_need` maps back to the same role
When no explicit structured target role points to a different owner
Then direct dispatch stops with an operator-visible same-role reason instead of pretending no-work is progress

Given a review role records approved or completed work with a `next_need` that names its own review category
When a later review or release owner is configured
Then dispatch treats that as completed review evidence and routes to the next review owner instead of applying the planning-role same-role stop

Given a non-Orchestrator dispatch-mode role records an ambiguous, blocked, failed, or conflicting disposition
When the route is not deterministic enough for the runtime to select safely
Then dispatch routes to the configured Orchestrator fallback for synthesis or stops with an operator-visible reason when no safe fallback exists

Given a dispatch-mode role finishes without a disposition
When the executor validates completion
Then the job fails closed with an actionable error so the pipeline cannot silently skip orchestration

Given a non-Orchestrator dispatch-mode role changed target files
When it records a terminal disposition before committing those changes
Then the tool policy rejects the disposition so the role must commit the produced work before Orchestrator can route the next handoff

Given Engineer, QA, Security, Dogfood, Release Manager, Dependency Manager, or Pipeline Fixer records a successful disposition such as `completed`, `approved`, or `in_review`
When `docsync_audit` finds missing `MarsDocSync` metadata, missing docs, invalid documentation references, or missing foundation expected-doc mappings
Then tool policy rejects the disposition with the failing files, so the role must fix DocSync or record `changes_requested`/`blocked` feedback instead of approving stale documentation

Given Orchestrator repeatedly chooses the same next role and next need without a ticket-state change
When the loop guard evaluates the new dispatch decision
Then dispatch stops with a loop-guard reason instead of enqueueing Orchestrator or the same target role again

Given QA has already requested implementation rework and the same ticket later reaches Security
When Security records fresh `changes_requested` feedback for the same product ticket without a ticket-state change
Then the loop guard treats the Security feedback as a later review-stage handoff and routes one bounded Engineer rework instead of stopping on the earlier QA route history

Given a dispatch-mode Orchestrator job fails before it records a disposition
When its trigger carries a non-Orchestrator `source_disposition` with a deterministic routing signal such as `next_need`, `suggested_role`, `handoff.target_role`, or `feedback.for_role`
Then the server records the failure as telemetry and falls forward to the deterministic target role using the original source handoff instead of enqueueing Orchestrator again

Given a dispatch-mode Orchestrator job fails before it records a disposition
When its trigger lacks a usable non-Orchestrator source disposition or the fallback would select Orchestrator again
Then dispatch records a stopped decision and creates no recursive Orchestrator job

Given Orchestrator selects Engineer for implementation
When the target has no ordinary product ticket in `docs/tickets/backlog/` or `docs/tickets/in-progress/`
Then dispatch routes to `cto-weekly` for ticket shaping instead of enqueueing Engineer with free-floating implementation work

Given Orchestrator selects Engineer for implementation
When an ordinary product ticket exists in backlog or in-progress
Then dispatch may enqueue Engineer and the ticket gate remains responsible for claim, completion, evidence, and handoff correctness

Given CTO is shaping a fresh product feature with a clear scenario schedule
When the earliest uncovered scenario is already covered by an ordinary ticket in backlog, in-progress, in-review, or done
Then CTO may create the next scenario ticket instead of being forced to wait for the earlier ticket to reach done, allowing a small ordered backlog batch for the project build

Given CTO records an implementation handoff for a feature with multiple early scheduled product scenarios
When fewer than the first two or three early product scenarios are covered by ordinary tickets
Then the disposition is blocked until CTO creates a small product backlog batch or deliberately groups adjacent early product scenarios in one bounded ticket

Given the active operating plan names a specific BDD feature and scenario schedule
When other active feature contracts still exist from starter planning or historical slices
Then the CTO implementation handoff batch is evaluated against the active-plan feature first, so stale unselected contracts do not block the current Engineer handoff

Given a later scheduled scenario is only about evidence ordering, governance, telemetry, or intervention-debt containment
When the early product scenarios are already covered by ordinary tickets
Then CTO may hand off to Engineer without creating a process-only implementation ticket

Given Engineer completed an ordinary product ticket and moved it to `docs/tickets/done/`
When Orchestrator or deterministic Orchestrator-failure fallback attempts another implementation, CTO planning, or other pre-review handoff but no open product ticket remains
Then dispatch routes QA review instead of routing back to Engineer, CTO ticket shaping, or another planning loop

Given QA approves a completed ordinary product ticket with `next_need: dogfood_validation`
When the QA dispatch trigger still carries the original completed Engineer source disposition for that ticket
Then dispatch honors QA's current approval and routes forward to Dogfood instead of reapplying the pre-review Engineer ticket guard and enqueueing QA again

Given Dogfood approves or completes validation after product work
When the manifest includes a `release-manager` role
Then dispatch routes directly to Release Manager so `VERSION`, `CHANGELOG.md`, tags, and release blockers are handled as part of the same lifecycle rather than relying on a later weekly schedule

Given Release Manager records a blocked disposition with `next_need: release_blocked`
When local release notes or tag evidence exists but publication is blocked by a missing remote, credentials, workflow failure, or asset publication failure
Then dispatch stops with an operator-visible release-publication blocker and does not enqueue Orchestrator, Dogfood, or earlier product-validation roles unless product validation evidence changed

Given Dogfood, Security, or another review role completes a validated product slice
When dispatch would route to Release Manager but ordinary product tickets remain open
Then dispatch routes Engineer before release so the backlog continues draining

Given no ordinary product tickets remain open
And the generated target feature contract still contains scenarios without a done ticket that references them in `bdd_scenarios`
When dispatch would route to Release Manager
Then dispatch routes CTO ticket shaping before release so generated feature scenarios are covered before version publication

Given Engineer fails a ticket gate such as missing BDD scenario evidence on a done ticket
When dispatch-mode failure handling records the failure
Then the server enqueues one bounded Engineer `ticket_gate_repair` job with the failure reason and a ticket-lifecycle/evidence-only repair scope instead of routing the failure through Orchestrator

Given Engineer tries to move a feature ticket to `docs/tickets/done/`
When `evidence_links` or `verified_by` are still empty
Then tool policy blocks the move before the post-run ticket gate and tells Engineer to populate the missing evidence fields in the same run

Given Engineer tries to copy a feature ticket into `docs/tickets/done/`
When the same ticket remains in backlog, in-progress, or in-review
Then tool policy blocks the copy and requires a single `git mv` lifecycle transition

Given an Engineer `ticket_gate_repair` job fails the ticket gate again
When dispatch-mode failure handling records the second failure
Then no recursive repair or Orchestrator job is enqueued automatically

Given an Engineer `ticket_gate_repair` job is triggered only because ticket evidence is missing
When the Engineer reads the trigger
Then generated role guidance tells it to update ticket evidence/lifecycle metadata and avoid broad product-code implementation unless the gate reason explicitly names invalid code

### F-006-S013: Runtime Failure Stops

Given a non-Orchestrator dispatch-mode job fails with a runtime-owned failure such as max turns, context overflow, inference/model availability, tool timeout, guardrail block, manifest error, manual stop, or unknown terminal failure
When failure handling records telemetry
Then the server keeps the signal as foundation telemetry and does not enqueue Orchestrator, CTO ticket shaping, Engineer retry, or target backlog intervention debt by default

Given the same ticket is still eligible in `docs/tickets/in-progress/`
When the native survey loop sees that ticket before the runtime-failure cooldown expires
Then the survey also refrains from same-role ticket-owner retry so runtime-failure containment cannot be bypassed by the watchdog

Given a runtime-owned failed role leaves uncommitted target-owned artifacts behind before hitting a terminal failure
When the native survey loop sees recent dogfood failures, failed checks, stale tickets, or other survey signals for that repo
Then the survey records the dirty workspace as an operator-visible pause and does not route Engineer, Orchestrator, Janitor, or other autonomous follow-up until those target artifacts are committed or deliberately cleaned

Given QA or another dispatch-mode role exits without recording `job_disposition_record`
When the model first tries to finish with prose only
Then the agent loop gives one corrective prompt for the required terminal `job_disposition_record` tool call instead of ending the job immediately

Given a dispatch-mode role has already received the turn-budget terminal-tool reminder
When the next model response calls any tool other than the configured terminal disposition tool
Then the loop ends as `max_turns` without executing that non-terminal tool, so the grace turn cannot mutate the repo and still fail disposition

Given generated QA is dispatched to review a completed implementation ticket
When the job starts in the default target harness
Then QA uses the `reasoning` model tier, starts with an allowed read-only inspection tool call, and has `git_status` plus `git_diff` available for repository evidence before recording `job_disposition_record`

Given QA or another dispatch-mode role still exits without recording `job_disposition_record` after the corrective terminal-tool prompt
When the executor marks the job failed with a dispatch protocol error
Then failure handling records telemetry and does not route that deterministic protocol miss through Orchestrator

Given QA blocks because implementation source or diffs were absent from the dispatch trigger
When the Orchestrator selects CTO, COO, CEO, or Janitor even though QA has repository read tools
Then dispatch retries QA with a repository-inspection handoff instead of turning missing trigger context into planning work

Given the orchestrator survey enqueued an Engineer job for an in-progress ticket
When a bounded repair or delivery job moves that ticket to `docs/tickets/done/` before the survey job runs
Then the stale pending survey Engineer job is cancelled instead of being claimed after the ticket is already complete

Given Security records an approved or completed dispatch disposition for a ticket
When Orchestrator suggests QA, Security, Dependency Manager, Release Manager, CTO, or another governance role before product dogfood has run
Then dispatch rewrites the decision to Dogfood when that role exists, or stops when no forward product validation owner remains

Given Security records an approved or completed dispatch disposition for a ticket
When Orchestrator explicitly requests Dependency Manager or Release Manager through `next_need` or `suggested_role`
Then dispatch may route to that governance role, but dependency and release work are no longer automatic review-chain defaults for a fresh product slice

Given Security reviews a feature ticket that Engineer completed and QA approved
When secrets scan, changed-code inspection, `docsync_audit`, tests, and at most one managed runtime smoke probe have passed
Then Security writes and commits a bounded audit report and records terminal `job_disposition_record` instead of repeating equivalent validation until `max_turns`

Given Security observes that an invalid-input command already exits non-zero safely
When the only remaining concern is speculative future hardening or a non-current risk
Then Security records the item as a non-blocking PASS note or low-severity observation instead of sending Engineer into changes-requested rework

### F-006-S019: Review Rework Reopens Tickets

Given Engineer receives a `changes_requested` handoff for a completed ticket
When shell validation or product/source edits are required to answer the review
Then Engineer must first reopen the ticket from `docs/tickets/done/` or `docs/tickets/in-review/` into `docs/tickets/in-progress/`, commit that rework claim, and only then mutate product files or run validation shell commands

Given Engineer receives a `changes_requested` handoff for a completed ticket
When the exact requested command or evidence passes after either no code change or one minimal patch
Then Engineer commits any changed work and records terminal `job_disposition_record` without adding unrelated exploratory probes that can consume the turn budget

Given Engineer implements an ordinary feature ticket with explicit acceptance criteria and BDD scenarios
When exploratory validation suggests an edge-case behavior that is not part of the selected ticket contract
Then Engineer keeps tests and code aligned to the ticket and feature contract, records the new edge case as follow-up evidence when useful, and closes the selected ticket once required evidence passes

Given Engineer has passed the selected ticket's acceptance evidence and committed the product implementation
When the ticket does not explicitly require a packaged binary, install artifact, or release distribution
Then Engineer moves the ticket to done and records disposition before creating repo-local build outputs or running packaging exploration

Given Engineer has successful validation evidence and a committed implementation while an ordinary product ticket remains in `docs/tickets/in-progress/`
When the worktree is clean and Engineer attempts additional `shell_exec` exploration instead of moving the ticket through the lifecycle
Then policy blocks the shell command and instructs Engineer to update ticket evidence, move the ticket to `docs/tickets/done/`, commit that lifecycle move, and record `job_disposition_record` with `next_need: qa_review`

Given Engineer has moved an ordinary product ticket from `docs/tickets/in-progress/` to `docs/tickets/done/`
When another ordinary product ticket remains in backlog
Then the lifecycle move commit is still allowed before the next claim gate runs
And any attempt to claim or mutate the next product ticket in the same job is blocked until Engineer commits the lifecycle move, pushes when a remote exists, and records `job_disposition_record` with `next_need: qa_review`

Given Engineer has successful validation evidence, dirty implementation work, and an in-progress product ticket
When it repeats empty-argv or single-colon `shell_exec` no-op calls
Then policy treats the repeated no-op as a loop boundary and instructs Engineer to stop shell placeholders, inspect status, commit the dirty work, complete the ticket lifecycle, and record `job_disposition_record`

Given Engineer reopens a completed ticket after QA requests validation rework
When Engineer builds an external validation binary with `go build -o /tmp/<project>-validation ...`
Then Engineer may execute that same fresh binary in the current role session as validation evidence instead of being blocked by the post-validation shell convergence gate

### F-006-S020: Runtime Validation Counts For Lifecycle Convergence

Given Engineer has an ordinary product ticket in `docs/tickets/in-progress/`
When a direct runtime command such as `go run`, `cargo run`, `dotnet run`, a language interpreter entrypoint, a package start script, or a bounded smoke probe successfully exercises the ticket behavior
Then the role session records that command as validation evidence, so after the implementation commit further placeholder or exploratory shell calls are redirected toward ticket evidence, `docs/tickets/done/`, and `job_disposition_record`

### F-006-S021: Expected Runtime Error Probes In Review

Given QA or Security reviews a named ticket and test files exist
When the role has a successful test command plus positive runtime evidence and a non-zero runtime probe for an expected invalid-input or error path that declares and matches `shell_exec expected_exit_code`
Then the review disposition may approve because the runtime error is negative-path evidence rather than a failed build or test

Given QA or Security accidentally runs an expected negative-path runtime probe without `expected_exit_code`
When the same command exited non-zero and no build or test command has failed in the current job
Then the role may rerun that exact command once with the matching non-zero `expected_exit_code`, and the corrected probe no longer blocks approval

Given QA or Security reviews a named ticket after a build command, test command, or unexpected runtime validation command failed in the same job
When the role attempts to approve, complete, or move the ticket to in-review
Then the disposition is blocked and the reviewer must record `changes_requested` or `blocked` with the failing command and requested Engineer action

### F-006-S023: Review Failure Handoff

Given QA or Security reviews a named ticket after a build command, test command, or unexpected runtime validation command failed in the same job
When the role attempts another `shell_exec` command before recording disposition
Then tool policy blocks the shell command and directs the reviewer to record `job_disposition_record` with `status: changes_requested`, `next_need: implementation_rework`, `feedback.for_role: engineer`, and the exact failing command/output

Given the only current-job validation failure is an expected negative-path runtime probe that was first run without `expected_exit_code`
When QA or Security attempts to rerun the exact same command with the matching non-zero `expected_exit_code`
Then tool policy allows that one corrective shell command, records expected negative-path evidence, and blocks any different shell command until disposition

### F-006-S022: Post-Validation Ticket Evidence Recovery

Given Engineer has successful validation evidence, a clean implementation commit, and an ordinary product ticket still in `docs/tickets/in-progress/`
When Engineer attempts any non-lifecycle `shell_exec` command instead of closing the ticket
Then policy blocks the shell command and says the next tool must be `file_read` on the in-progress ticket followed by `file_write` on the same ticket to populate `evidence_links` and `verified_by`

Given that in-progress ticket is for a browser-framework package
And required package build evidence or browser-product smoke evidence is still missing
When Engineer runs the missing build command or the bounded smoke/source-runtime assertion after the implementation commit
Then the post-validation convergence gate allows that evidence command
And blocks other exploratory shell commands until the required evidence exists

Given Engineer has claimed an ordinary product ticket but has not yet produced validation evidence
When Engineer repeats an empty-argv or single-colon `shell_exec` no-op
Then policy blocks the repeated no-op and routes the session back to ticket/feature reading plus product `file_write` implementation or a blocked disposition

Given the ticket evidence has been updated after successful validation
When Engineer runs the exact `git mv` lifecycle command to move the ticket into `docs/tickets/done/`
Then the shell command is allowed so the lifecycle move, commit, push, and QA handoff can finish

Given Engineer has observed an unexpected runtime validation failure in the current job
When the exact failing command has not subsequently passed and has not been corrected with matching `expected_exit_code`
Then tool policy blocks writing or moving a product ticket to `docs/tickets/done/`, blocks committing a staged ticket-done move, and blocks successful disposition until the runtime evidence is repaired

Given Engineer has observed an unexpected runtime validation failure in the current job
When Engineer reruns that same command with `expected_exit_code`
Then the outstanding completion blocker remains because retroactive expected-exit correction is reserved for QA/Security review-procedure mistakes

Given Engineer has observed an obvious missing-argument runtime validation failure in the current job
When Engineer reruns that exact no-argument command with matching `expected_exit_code`
Then the queue session treats it as expected negative-path evidence and clears the blocker so the role can continue positive acceptance validation

Given Engineer has an unresolved missing-argument runtime validation failure in the current job
When policy blocks a different runtime probe, ticket completion, or successful disposition
Then the queue-facing blocker text names the exact `expected_exit_code` correction instead of only saying to rerun the command successfully

Given a direct runtime validation command exits 0 while printing error-shaped stderr
When the queue session evaluates validation progress
Then the session treats the command as failed runtime evidence and keeps completion blocked until an exact clean rerun succeeds

Given Engineer has observed an unexpected runtime validation failure in the current job
When Engineer attempts to rerun runtime validation before any post-failure implementation edit
Then policy blocks the runtime probe and directs Engineer to inspect/edit the implementation, then rerun the exact failed command successfully

Given Engineer has implemented and validated an ordinary product ticket
When the worktree contains product changes plus a lifecycle move from `docs/tickets/in-progress/` to `docs/tickets/done/`
Then the final `git_commit` is allowed as ordinary completion rather than being treated as hidden review rework against an already-done ticket

Given Dogfood creates a target-owned finding ticket during validation
When that ticket exists in the current Dogfood run
Then Dogfood may inspect status, diff, read files, commit, push, or record disposition, but further shell validation and additional `ticket_create` calls are blocked until the finding is handed off through terminal disposition

Given Engineer records a successful dispatch disposition for a feature ticket
When the named ticket still exists in `docs/tickets/backlog/`, `docs/tickets/in-progress/`, or `docs/tickets/in-review/`
Then tool policy rejects the disposition before the server records it, forcing same-run ticket evidence and lifecycle completion instead of a later repair job

Given Engineer selects an ordinary product ticket from `docs/tickets/backlog/`
When it attempts product mutation before claiming that ticket into `docs/tickets/in-progress/`
Then tool policy blocks the mutation so backlog work cannot become source commits without a visible ticket claim

Given Engineer selects an ordinary product ticket from `docs/tickets/backlog/`
When it attempts read-only discovery, broad traversal, validation, or no-op placeholder work through `shell_exec` before claiming that ticket into `docs/tickets/in-progress/`
Then tool policy blocks the shell call with claim-first guidance so implementation handoff cannot spend turns before visible ticket ownership

Given an active plan names a BDD feature ID such as `F-001`
When generated planner, ticketing, or Orchestrator prompts check for the feature contract
Then they search `docs/features/F-001*.md`, treat slugged contracts as present, and do not block solely because `docs/features/F-001.md` is absent

Given a target already has a `docs/features/F-001*.md` contract
When a planner or ticketing role tries to create a second contract path with the same `F-001` feature ID
Then `file_write` rejects the duplicate so the target keeps one canonical feature contract per feature ID

### F-006-S016: Canonical Bootstrap Feature Contracts

Given generated CEO runs fresh bootstrap and a product brief implies feature-contract work
When the CEO prepares a planning handoff
Then the CEO does not write `docs/features/` and names the existing canonical feature-contract path for COO instead of inventing a new `F-001` slug

Given generated COO updates a starter `F-001` feature contract
When `docs/features/F-001-product-walking-skeleton.md` already exists
Then COO edits that existing path in place and replaces starter scenarios with one unique scenario set instead of appending duplicate scenario headings

Given generated COO updates a starter `F-001` feature contract
When the planner adds product-specific scenario headings
Then every heading uses the same `F-001-SNNN` feature ID prefix as the contract path, and work that needs `F-002-SNNN` first creates or updates the canonical `docs/features/F-002*.md` contract

Given Orchestrator receives a next need of `strategy_advice`, `executive_narrative`, `tradeoff_analysis`, or `goal_conflict`
When the target manifest includes the optional `head-of-strategy` role
Then Orchestrator routes to that advisor for memo, tradeoff, and narrative work while CEO remains owner of final goal or vision decisions

Given Orchestrator receives one of those strategy advisory next needs
When the target manifest does not include `head-of-strategy`
Then Orchestrator routes to CEO instead of inventing an absent role or blocking delivery

Given a role records `handoff` or `feedback` in `job_disposition_record`
When Orchestrator selects the next role
Then the disposition stores those objects durably, the dispatch trigger carries them in `source_disposition`, and the next ask names the target role, requested change or action, context, constraints, expected output, and evidence instead of relying on implicit handoff prose

Given Orchestrator records `suggested_role`, `handoff.target_role`, or `feedback.for_role`
When more than one structured target is present
Then the targets must agree or the disposition is rejected with an actionable error instead of guessing the next owner

Given Orchestrator receives `ticket`, `ticket_shaping`, or `ticket_breakdown`
When the target manifest contains `cto-weekly`
Then Orchestrator routes to CTO for technical decomposition and implementation ticket creation, while `exec_plan`, `feature_contract`, `scenario_schedule`, and `current_failing_scenario` route to COO

Given QA records `changes_requested` with `next_need: implementation_rework` for an existing ordinary product ticket
When that ticket is already in `docs/tickets/done/` or `docs/tickets/in-review/` and Orchestrator selects Engineer
Then dispatch routes Engineer to rework the same ticket instead of rewriting to CTO ticket shaping or creating a duplicate product ticket

Given Orchestrator suggests a canonical role alias such as `cto`, `release`, or `dependency`
When the target manifest contains the corresponding executable role key such as `cto-weekly`, `release-manager`, or `dependency-manager`
Then dispatch normalizes to the manifest role key and does not fall back into another Orchestrator loop

### F-006-S023: Ticket-Creation Failure Cannot Handoff As Product Progress

Given a non-Orchestrator planning role has an unresolved failed `ticket_create` or failed ticket-file write in the current session
When it attempts `job_disposition_record` with a successful status such as `completed`, `approved`, `in_review`, or `no_work`
Then policy blocks the disposition and directs the role to retry `ticket_create` with valid JSON or record an honest blocked disposition with the exact failure

Given the role records `status: blocked`, `status: failed`, or `status: changes_requested`
When ticket creation is still unresolved
Then the disposition remains available so orchestration can surface the blocker without pretending an implementation ticket exists

### F-006-S024: Missing-Argument Runtime Corrections Converge Before Mutation

Given Engineer has an unresolved no-argument or missing-required-input runtime probe in the current session
When the session stores an exact `shell_exec` correction with matching `expected_exit_code`
Then unrelated mutations, commits, decisions, pushes, dependency sync, CLI mutation, tool creation, and persona creation are blocked until that correction runs or Engineer records an honest blocked disposition

Given Engineer records `status: blocked` with the exact blocker
When the correction cannot be applied
Then the dispatcher receives explicit blocker evidence instead of another silent runtime-probe loop

### F-006-S025: External Validation Artifacts Rebuild After Runtime Failure Edits

Given Engineer has an unresolved positive acceptance failure from an external `<validation-root>` artifact
When Engineer edits implementation source after that failure
Then queue-session policy treats the previously built artifact as stale until a fresh `go build -o <validation-root> ...` succeeds

Given the artifact is rebuilt after the edit
When Engineer reruns the same runtime acceptance probe
Then the rerun can repair the outstanding runtime blocker based on current source behavior

### F-006-S026: Ticket Evidence Follows Validation

Given Engineer is working an in-progress ticket and has not produced successful validation in the current job
When it attempts to fill `evidence_links` or `verified_by`
Then orchestration-facing ticket state remains unchanged and the role is directed back to validation first

Given a validation command succeeds in that job
When Engineer updates the in-progress ticket with concrete evidence and verifier metadata
Then the ticket can continue through the normal done move, commit, and QA handoff path

### F-006-S027: Reviewer Artifact Rebuild Recovery

Given QA or Security tries to execute an external validation artifact built by an earlier role session
When same-session artifact freshness blocks the command
Then the next review action is the exact rebuild command from the tool error or a structured blocked/changes-requested disposition

Given the reviewer runs that exact rebuild command
When the binary build succeeds in the same role session
Then the reviewer can rerun the runtime probe as current evidence

### F-006-S028: Review Build Guard Recovery Is Deterministic

Given QA or Security attempts a Go build that would write a repo-local binary
When the build-output guardrail blocks the command
Then the reviewer receives an exact package-preserving `shell_exec argv` correction instead of prose-only recovery guidance

Given the reviewer follows that exact correction
When the build succeeds in the same role session
Then review can continue with current runtime evidence instead of routing changes-requested from a guessed build target

Given QA reviews a Go feature ticket with non-test `.go` source files
When no `_test.go` files exist in the repository
Then approval is blocked and QA must request Engineer tests before the ticket can continue to Security or release

### F-006-S029: Missing-Input Repro Loops Route To Repair

Given Engineer has a missing-required-input runtime failure in the current job
When the exact `expected_exit_code` correction is attempted but the command still panics or exits incorrectly
Then the job can proceed to implementation repair instead of repeatedly dispatching the same repro command

Given that runtime failure remains outstanding
When Engineer tries to commit, move the ticket to done, or record QA handoff before the exact runtime path is repaired
Then orchestration policy still blocks completion and keeps the ticket in the implementation loop

### F-006-S030: Repeated Exact Runtime Failures Converge

Given Engineer repeats the same runtime validation command during implementation repair
When later source changes make that exact command exit successfully
Then the queue session clears every outstanding same-command failure count so dispatch does not end in stale `circle_detected`

Given the job has no other outstanding runtime blockers
When the exact command succeeds
Then Engineer can continue toward tests, evidence, ticket lifecycle, or reviewer handoff

### F-006-S031: Review And Ticket Closure Preserve Live Product State

Given QA or Security reviews a completed ticket
When it needs shell validation
Then shell access is limited to read-only inspection, tests, builds, fresh external validation binaries, runtime probes, and HTTP probes

Given a reviewer attempts package/module initialization, product mutation, cleanup, broad discovery, or a placeholder no-op through `shell_exec`
When the command is not validation evidence
Then policy blocks it and routes the reviewer to read-only tools, validation commands, or a structured disposition

Given Engineer has uncommitted source, tests, docs, package manifests, lockfiles, config, or validation code
When it tries to move a product ticket into `docs/tickets/done/`
Then the lifecycle move is blocked until those non-ticket changes are committed separately

Given Orchestrator reads `docs/tickets/README.md`
When the README contains example ticket text
Then Orchestrator treats it as conventions only and routes from lifecycle directories plus `source_disposition` ticket and handoff state

### F-006-S032: Review No-Op Loops Converge To Disposition

Given QA or Security has already run successful validation evidence in the current job
When it calls `shell_exec` with empty `argv` or another no-op placeholder
Then policy tells the reviewer to stop shell validation and record `job_disposition_record`

Given the reviewer repeats the same no-op command shape
When the job has `job_disposition_record` as a required terminal tool
Then the agent loop gives one terminal-tool-only circle grace turn before failing the job

### F-006-S033: Runtime Repair Freezes False Progress

Given Engineer has an unresolved positive runtime acceptance failure
When it tries to run other shell probes, shell wrappers, tests, ticket moves, or product commits
Then policy blocks those side paths until source repair plus the exact failed runtime command prove the acceptance path now passes

Given the failing runtime command used a same-session `<validation-root>` artifact
When Engineer edits source after the failure
Then rebuilding that same stale validation artifact remains allowed before the exact failed command is rerun

### F-006-S034: Ticket Evidence Blocks Do Not Poison Implementation Handoff

Given Engineer attempts to populate an in-progress ticket's evidence before successful validation
When policy blocks that evidence write
Then the failure is treated as an evidence-ordering guardrail, not unresolved ticket-creation debt

Given Engineer later validates the behavior, updates ticket evidence, commits product work, moves the ticket to done, and records `job_disposition_record`
When no actual `ticket_create` failure remains unresolved in the same job
Then the disposition is allowed to hand off to QA review

### F-006-S035: Review No-Op Recovery Uses Terminal Disposition

Given QA or Security has already recorded successful validation evidence in the current job
When a no-op shell placeholder is blocked with terminal disposition guidance
Then any later non-terminal tool call is blocked and only `job_disposition_record` may complete the job

Given the reviewed target has Go source files but no `_test.go` files
When QA reaches terminal disposition after validation
Then the role records `changes_requested` for Engineer to add durable tests instead of approving from runtime smoke evidence alone

Given COO or another non-ticket-owning planner hits an attempted ticket-creation policy block
When it records `next_need: ticket_breakdown` for CTO
Then the handoff remains available and does not require a blocked disposition detour

### F-006-S036: Engineer Test/Build Failure Repair Lane

Given Engineer observes a failing test or build command in the current job
When the role attempts runtime probes, unrelated shell commands, ticket evidence updates, ticket done moves, successful disposition, or product commits before same-lane validation passes
Then the session blocks those actions and routes Engineer to bounded source, test, fixture, or package/build config repair followed by a same-lane test/build command

Given Engineer edits source or tests after the failing test/build command
When the role reruns a same-lane test/build command and it exits successfully
Then the outstanding repair lane clears and normal ticket evidence, lifecycle completion, and QA handoff can continue

### F-006-S037: CTO Ticket Shaping Does Not Implement

Given CTO is shaping a ticket from COO planning
When it attempts to write product implementation files, package/module files, README usage notes, tests, build config, or root product files
Then the session blocks the write and keeps implementation behind `ticket_create` plus Engineer delivery

Given CTO needs to record architecture rationale before ticketing
When it writes bounded technical planning artifacts under `docs/design-docs/`, `docs/reports/strategy/`, or `docs/goals/observations.md`
Then those writes remain available and can be committed before the CTO disposition

### F-006-S038: Test/Build Repair Allows Same-Lane Validation

Given Engineer observes a failing test command in the current job
When it edits product source, tests, fixtures, or package/build config
Then a later recognized test command in the same lane may repair the blocker even when its package scope differs from the original command
And tests include files under `test/` or `tests/` plus conventional `*.test.*`, `*.spec.*`, and Go `_test.go` files

Given Engineer observes a failing build command in the current job
When it edits product source, tests, fixtures, or package/build config
Then a later recognized build command in the same lane may repair the blocker even when its output target or package scope differs from the original command

Given a failing test/build repair lane is unresolved
When Engineer attempts runtime probes, helper scripts, unrelated shell commands, ticket evidence updates, ticket done moves, successful disposition, or product commits
Then those actions remain blocked until the same validation lane passes
And repeated guardrail messages summarize the failing output compactly so repair guidance does not overflow the role context

### F-006-S039: Simple CD Validation Commands Stay In Lane

Given Engineer has an unresolved failing test/build repair lane
When it runs a shell command shaped exactly like `cd <dir> && <recognized test/build command>`
Then the command is classified by the right-hand test/build command and may repair the same validation lane after bounded repair edits

Given Engineer has an unresolved failing test/build repair lane
When it uses arbitrary shell control syntax, multiple chained operations, pipes, redirection, substitutions, cleanup, runtime probes, or ticket moves
Then the command remains blocked as a side path rather than counting as validation repair

### F-006-S040: Clean Review Evidence Stops More Review Work

Given a dispatch-mode QA or Security job requires `job_disposition_record`
And the role has read relevant source or ticket context and has clean validation evidence
When the role tries to continue with another non-terminal action after the terminal reminder
Then orchestration ends the job as a contained loop boundary instead of executing more review work or routing target intervention debt

Given the role responds with `job_disposition_record`
When the disposition is valid
Then normal forward review dispatch can continue

### F-006-S041: Test-Build Repair Writes Stay Scoped

Given Engineer has an unresolved failing test/build repair lane from a narrow package command
When it edits source, tests, fixtures, or testdata outside that failed package scope
Then the write is blocked as an alternate implementation path

Given Engineer edits source, tests, fixtures, testdata, or package/build config inside the failed scope
When it reruns a same-lane test/build command successfully
Then the repair lane can clear and normal ticket lifecycle may resume
And focused test-file edits stay eligible repair work instead of being treated as unrelated helper scripts

### F-006-S042: Same-Job Test Cleanup Covers Pre-Failure Writes

Given Engineer created or rewrote a test-like file earlier in the same job
And a later test/build command fails in that package
When Engineer removes that same-job test file with non-recursive `rm` or `unlink`
Then the cleanup remains inside the failing test/build repair lane so duplicate generated tests can be pruned before rerunning validation

Given a test-like file was not created or rewritten by the same Engineer job
When Engineer tries to remove it while a test/build repair lane is unresolved
Then the deletion remains blocked so old project tests cannot be discarded to make validation pass

### F-006-S043: Startup Cleanup Preserves SQLite Recovery State

Given `start` or `serve` is retried after a failed bind, interrupted process, or stale worker cleanup
And the configured SQLite database has `-wal` or `-shm` sidecar files
When startup cleanup runs before opening the server
Then the harness leaves those sidecar files in place and asks SQLite to recover or checkpoint them instead of deleting queue or repo registry state

Given SQLite recovery fails during cleanup
When startup continues or reports the failure
Then the sidecar files remain operator-visible and are not silently removed by automatic cleanup

### F-006-S044: Ticket Lifecycle Blocks Recover After Max Turns

Given Engineer has implemented and committed product work for an in-progress ticket
And tool policy blocks ticket evidence, done-ticket moves, or successful disposition because lifecycle evidence is incomplete
When the Engineer then reaches `max_turns`
Then dispatch enqueues one bounded `ticket_gate_repair` Engineer job instead of stopping with only telemetry

Given the bounded repair job runs
When evidence is missing but the code state is not proven invalid
Then Engineer runs one validation command that exercises the named BDD scenario, updates ticket evidence, moves the ticket to `docs/tickets/done/`, commits the lifecycle correction, and records `job_disposition_record`

Given a static HTML/CSS/JS ticket has no package manifest
When Engineer validates the ticket by serving the HTML entry and probing it with HTTP
Then the successful HTTP probe counts as validation evidence for ticket evidence updates and lifecycle completion

### F-006-S046: Engineer Product Progress Continues After Max Turns

Given Engineer reaches `max_turns` or `circle_detected` while an ordinary product ticket remains in `docs/tickets/in-progress/`
And the failure is not already running a bounded ticket-gate repair or product-continuation job
When dispatch-mode failure handling records the runtime failure
Then the harness enqueues one bounded Engineer `product_continuation` job for the same active product ticket instead of stopping with only foundation telemetry

Given that product-continuation job runs
When it inspects the target state
Then it continues from the latest commits and dirty files, fixes only the remaining product/build/validation/lifecycle gaps, updates evidence, moves the ticket to done when acceptance is met, and records `job_disposition_record`

Given a product-continuation job also reaches `max_turns`
When failure handling runs again
Then the harness does not enqueue recursive product-continuation jobs

## Out of Scope

- External queue systems such as Redis.
- Multi-agent concurrent edits to the same repo as the default path.
- Infinite recovery recursion.

## Descoped Scenarios

None.

## Evidence

- F-006-S001: `go test ./internal/serve -run TestRepoRegistry`
- F-006-S002: `go test ./internal/queue -run TestQueue`
- F-006-S003: `go test ./internal/queue -run TestWorkerPool`
- F-006-S004: `go test ./internal/serve -run 'TestTriggerRouter|TestResolveSchedule|TestHandleJobComplete'`
- F-006-S005: `go test ./cmd/mars-harness -run 'Test(Start|Init|Run|Register|Scan).*GeneratedHarnessBaseline'`
- F-006-S006: `go test ./internal/serve -run TestServer`
- F-006-S007: `go test ./internal/serve -run 'TestHandleJobFailed|TestSelfHealRecoveryQueue'` and `go test ./internal/queue -run TestQueue_repairActiveRecoveryJobs`
- F-006-S008: `go test ./internal/serve -run 'TestValidateEngineerTicketGate|TestBuildTicketIndex|TestFirstBacklogInterventionDebt'`
- F-006-S009: `go test ./internal/serve -run TestOrchestratorSurvey` and `go test ./internal/queue -run 'TestQueue_concurrencyGroupSerialization|TestQueue_dailyCapConstrainsRepeatedScheduling|TestQueue_claimDoesNotResetHealthyRunningJob|TestQueue_failStuckRunningJobs'`
- F-006-S010: `go test ./internal/orchestration ./internal/orgstate` and `go test ./internal/serve -run TestFoundationAcceptance`
- F-006-S011: `go test ./internal/serve -run TestHandleJobComplete_reviewReworkReusesExistingDoneProductTicket`
- F-006-S012: `go test ./internal/tools -run 'TestShellExecPolicy.*FeatureTicketDone(Move|Copy)|TestFileWritePolicyBlocksDoneFeatureTicket'`
- F-006-S013: `go test ./internal/serve -run TestHandleJobFailed_maxTurnsDoesNotRouteOrchestrator`
- F-006-S015: `go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive` and `go test ./internal/queue -run TestQueue_activeJobForRepoRole`
- F-006-S018: `go test ./internal/tools -run 'TestDogfoodUncommittedFindingBlocksFurtherValidationAndTickets|TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation'`
- F-006-S019: `go test ./internal/tools -run 'TestEngineerPostValidationGateAllowsValidationWhileImplementationDirty|TestEngineerMustReopenDoneTicketBeforeProductMutation|TestEngineerPostValidationAllowsFreshExternalValidationArtifact'`
- F-006-S020: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksRuntimeValidationCommands|TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion'`
- F-006-S021: `go test ./internal/tools -run TestReviewApprovalRequiresPassingValidationWhenTestsExist`
- F-006-S022: `go test ./internal/tools -run 'TestEngineerPostValidationCommitBlocksExploratoryShellUntilTicketDone|TestEngineerPostValidationAllowsMissingBrowserBuildAfterCommit|TestEngineerPostValidationAllowsMissingBrowserSmokeAfterBuild|TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion|TestEngineerCannotCompleteTicketWithUnresolvedRuntimeValidationFailure'` and `go test ./internal/agent -run 'TestRun_requiredTerminalToolGetsOneBudgetGraceTurn|TestRun_terminalToolGraceRejectsNonTerminalTool'`
- F-006-S023: `go test ./internal/tools -run TestSuccessfulDispositionBlocksUnresolvedTicketCreationFailure`
- F-006-S024: `go test ./internal/tools -run TestEngineerMissingArgumentRuntimeFailureBlocksUnrelatedMutation`
- F-006-S025: `go test ./internal/tools -run 'TestExternalValidationArtifact(MustBeRebuiltAfterRuntimeFailureEdit|AllowsRerunAfterRuntimeFailureEditAndRebuild)'`
- F-006-S026: `go test ./internal/tools -run 'TestEngineerTicketEvidenceWrite(RequiresValidation|AllowedAfterValidation)'`
- F-006-S027: `go test ./internal/tools -run TestExternalValidationArtifactMustBeBuiltInSameSession`
- F-006-S028: `go test ./internal/tools -run 'TestShellExecBlocksDefaultGoBuildForCmdPackageWithExactCorrection|TestQAApprovalRequiresGoTestsForGoSource'`
- F-006-S029: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksFailedMissingArgumentCorrectionAttempt|TestEngineerMissingArgumentRuntimeFailureAllowsImplementationEditAfterCorrectionAttempt'`
- F-006-S030: `go test ./internal/tools -run TestRecordSessionToolOutcomeExactSuccessClearsRepeatedRuntimeFailures`
- F-006-S031: `go test ./internal/tools -run 'TestReviewShellExecPolicy|TestShellExecPolicy.*TicketDoneMove'` and `go test ./internal/scanner -run TestInit_success`
- F-006-S032: `go test ./internal/agent -run TestRun_requiredTerminalToolGetsOneCircleGraceTurn` and `go test ./internal/tools -run 'TestReviewShellExecPolicyRoutesPostValidationNoopToDisposition|TestRecordSessionToolPolicyFailureTracksNoopFailures'`
- F-006-S033: `go test ./internal/tools -run 'TestEngineerRuntimeFailureBlocks(ShellWrapperBypass|ValidationUnrelatedShell)|TestEngineerPositiveRuntimeFailureBlocksImplementationCommit|TestEngineerRuntimeFailureAllowsStaleValidationArtifactRebuild'`
- F-006-S034: `go test ./internal/tools -run 'TestRecordSessionToolOutcome(TracksTicketCreationFailures|IgnoresEngineerTicketEvidencePolicyFailure)|TestSuccessfulDispositionBlocksUnresolvedTicketCreationFailure'`
- F-006-S035: `go test ./internal/tools -run 'TestReviewShellExecPolicyRoutesNoTestGoRepoToChangesRequested|TestReviewTerminalDispositionRequiredBlocksFurtherShellExec|TestPlanningRoleCanHandOffUnownedTicketCreationFailure'`
- F-006-S036: `go test ./internal/tools -run 'TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation|TestEngineerFailingTestBlocksCommitTicketEvidenceAndDisposition|TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane'`
- F-006-S037: `go test ./internal/tools -run TestCTOFileWritePolicyAllowsTechnicalPlanningAndBlocksImplementation`
- F-006-S038: `go test ./internal/tools -run 'TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation|TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane'`
- F-006-S039: `go test ./internal/tools -run 'TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation|TestRecordSessionToolOutcomeEngineerTracksTestBuildRepairLane'`
- F-006-S040: `go test ./internal/agent -run TestRun_reviewEvidenceReminder`
- F-006-S041: `go test ./internal/tools -run TestEngineerFailingTestBlocksRuntimeProbeUntilSourceEditAndSameLaneValidation`
- F-006-S043: `go test ./internal/serve -run TestCleanStaleSQLitePreservesRecoverableSidecars`
- F-006-S044: `go test ./internal/serve -run TestHandleJobFailed_maxTurnsAfterTicketLifecyclePolicyBlockEnqueuesRepair` and `go test ./internal/tools -run TestRecordSessionToolOutcomeTracksHTTPProbeValidation`
- F-006-S045: `go test ./internal/serve -run 'TestHandleJobComplete_(openProductTicketRoutesBeforeRelease|uncoveredGeneratedFeatureScenarioRoutesCTOBeforeRelease)'`
- F-006-S046: `go test ./internal/serve -run 'TestHandleJobFailed_maxTurnsWithActiveProductTicketEnqueuesContinuation|TestHandleJobFailed_circleDetectedWithActiveProductTicketEnqueuesContinuation|TestHandleJobFailed_productContinuationDoesNotReenqueue|TestOrchestratorSurveyPausesTicketOwnerAfterRecentRuntimeFailure'`
