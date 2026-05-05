# F-010: Dashboard And Control Plane

- Feature ID: F-010
- Goals: G-001, G-003
- Status: partially-passing
- Owner: COO

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-010-S001 - Dashboard pages and static assets are served from the embedded binary.
2. F-010-S002 - Health, status, repos, repo roles, and quality score APIs expose orchestrator state.
3. F-010-S003 - Pause, resume, restart, stop, scan, and run-role controls call shared server methods.
4. F-010-S004 - Server-sent events stream dashboard updates.
5. F-010-S005 - Emergency stop is available through the dashboard and reports callback errors.
6. F-010-S006 - Empty states explain missing modules or data without crashing the dashboard.
7. F-010-S007 - CLI interactive controls and dashboard controls remain behaviorally aligned.
8. F-010-S008 - The orchestration page and APIs show dispatch-mode routing state instead of a static linear pipeline.
9. F-010-S009 - Interactive CLI sessions use a full-screen terminal dashboard with durable debug logs.

## Scenarios

### F-010-S001: Embedded Dashboard Pages

Given the server is running
When a user opens the dashboard routes
Then all pages return successfully and static assets are served without external frontend dependencies

### F-010-S002: Status APIs

Given repos, roles, jobs, and quality artifacts exist
When dashboard API endpoints are called
Then they return structured state for status, repos, roles, and repo-visible quality score

### F-010-S003: Shared Control Actions

Given the orchestrator is running
When dashboard controls call pause, resume, restart, stop, scan, or run-role
Then the same server control path used by runtime controls is invoked and method errors are reported

### F-010-S004: Live Event Stream

Given dashboard clients subscribe to server-sent events
When runtime events are broadcast
Then the SSE connection stays open and receives update events

### F-010-S005: Emergency Stop Control

Given emergency stop callbacks are registered
When the dashboard emergency stop endpoint is called
Then all callbacks run and any callback failures are surfaced

### F-010-S006: Empty State

Given optional dashboard modules or backing data are unavailable
When the page is rendered
Then the dashboard shows an empty state instead of panicking or serving broken markup

### F-010-S007: CLI And Web Control Parity

Given serve or start is running with interactive controls
When an operator uses CLI keys or dashboard buttons
Then pause, resume, restart, scan, stop, and run-role semantics stay aligned

Given `serve`, `start`, or `run` is attached to an interactive TTY
When the command runs without `--debug`
Then the terminal redraws a dashboard-style view with state, current work, recent events, blocker summaries, controls, dashboard URL when available, and command log path

Given the same commands run without a TTY
When output is piped or captured
Then the command prints concise plain progress instead of entering alternate-screen mode

Given an operator passes `--debug`
When the command runs
Then verbose trace and slog output streams inline while all slog records still write to the command log file

### F-010-S008: Orchestration State View

Given registered repos use legacy chains or dispatch-mode orchestration
When an operator opens `/orchestration` or calls `/api/orchestration`
Then the dashboard reports each repo's orchestration mode, role topology, and dispatch status from live server state instead of showing a static or purely linear pipeline model

Given orchestration decisions have been recorded
When `/api/orchestration/decisions` is called or dispatch events are broadcast
Then the dashboard exposes recent decisions so operators can see why Orchestrator selected the next role or stopped dispatch

## Out of Scope

- React, npm, or externally hosted dashboard assets.
- A dashboard quality score that diverges from `docs/QUALITY_SCORE.md`.
- Browser-only controls that bypass server methods.

## Descoped Scenarios

None.

## Evidence

- F-010-S001: `go test ./internal/dashboard -run 'TestDashboard_allPagesReturn200|TestDashboard_staticAssets'`
- F-010-S002: `go test ./internal/dashboard -run 'TestDashboard_statusEndpoint|TestDashboard_reposEndpoint'` and `go test ./internal/serve -run TestServer_qualityScoreAPIServesRepoArtifact`
- F-010-S003: `go test ./internal/dashboard -run 'TestDashboard_(pause|resume|restart|stop|scan|runRole)Endpoint'`
- F-010-S004: `go test ./internal/dashboard -run TestDashboard_sseConnection`
- F-010-S005: `go test ./internal/dashboard -run TestDashboard_emergencyStop`
- F-010-S006: `go test ./internal/dashboard -run TestDashboard_missingModuleEmptyState`
- F-010-S007: `go test ./internal/ui` and planned E2E evidence comparing CLI key listener and HTTP controls
- F-010-S008: `go test ./internal/dashboard -run TestDashboard_missingModuleEmptyState`; API decision-history evidence remains planned
- F-010-S009: `go test ./internal/ui` and `go test ./cmd/mars-harness -run 'TestRunStartServeExposeDebugAndLogFileFlags|TestStartCommandInitializesRegistersSeedsAndStops'`
