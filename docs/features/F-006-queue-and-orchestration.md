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
8. F-006-S008 - In-progress and high-priority intervention-debt ticket priority controls what engineers claim next.
9. F-006-S009 - Native Orchestrator surveys route unattended failure states into bounded jobs or intervention-debt tickets.

## Scenarios

### F-006-S001: Repo Registration Isolation

Given a repo is registered without an explicit database path
When registration completes
Then queue, telemetry, schedule, and repo state use the repo-specific database path rather than contaminating another repo

### F-006-S002: SQLite Job Lifecycle

Given jobs are enqueued for a repo
When workers claim, complete, cancel, expire, or prune them
Then SQLite state changes are durable, idempotent where required, and queryable for recent job history

### F-006-S003: Worker Controls

Given the worker pool is running
When pause, resume, or completion hooks are used
Then new claims stop while paused, resume claims pending work, and success hooks enqueue downstream work

### F-006-S004: Trigger Routing

Given manifests define webhook, schedule, or chain triggers
When events, cron ticks, or completed jobs occur
Then matching roles are enqueued and invalid manifests do not take down the router

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
Then the system creates at most one active recovery job per repo/role key for transient failures, avoids recursive recovery, cancels duplicates, and routes deterministic failures to intervention debt without same-role retry

### F-006-S008: Ticket Claim Priority

Given tickets exist in in-progress, high-priority intervention-debt, medium-priority intervention-debt, and ordinary backlog states
When engineer context is assembled
Then existing in-progress work is prioritized first, high-priority intervention debt can preempt ordinary backlog claims, and medium/low intervention debt remains visible without blocking ordinary product backlog progress

### F-006-S009: Native Orchestrator Survey

Given queue, ticket, score, telemetry, and recent outcome signals exist
When the Orchestrator survey runs
Then stale tickets, blocked tickets, failed checks, dogfood failures, no-op outcomes, telemetry patterns, and low scores create bounded queue work or deduped intervention-debt tickets with payload mode, concurrency group, and daily cap metadata

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
