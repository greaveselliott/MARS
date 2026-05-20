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
