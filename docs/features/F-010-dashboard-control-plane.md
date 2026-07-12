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
10. F-010-S010 - The next-generation dashboard is governed by the dashboard control-plane product spec.
11. F-010-S011 - Dashboard prerequisite checks require external Node `24.x` and `pnpm@11.1.1` without bundling or installing them.
12. F-010-S012 - Local-admin auth protects every dashboard route and mutation.
13. F-010-S013 - The TanStack Start sidecar is served behind the Go gateway.
14. F-010-S014 - Dashboard APIs remain concurrent and nonblocking while orchestrator work is active.
15. F-010-S015 - Overview shows active agents, work, duration, issues, telemetry, token usage, DORA, and model usage.
16. F-010-S016 - Active Work renders current ticket details tastefully while preserving repo artifacts as source of truth.
17. F-010-S017 - Active Work Preview adapts to web, mobile, API, cloud, distributed-system, library, and CLI work.
18. F-010-S018 - Feedback is captured in a next-turn mailbox with stable anchors when available.
19. F-010-S019 - Agent Roster lists prompts, metrics, models, tools, guardrails, and safe code-host proposal actions.
20. F-010-S020 - Available Models lists offline and cloud-hosted models and proposes additions or overrides with validation.
21. F-010-S021 - GitHub-derived DORA metrics are defined from configured deployment workflows and show typed unavailable states.
22. F-010-S022 - Legacy dashboard migration preserves operator clarity while the TanStack control plane rolls out.
23. F-010-S023 - Control and dashboard listeners default to loopback and reject non-loopback exposure until the authenticated remote gateway lands.

## Scenarios

### F-010-S001: Embedded Dashboard Pages

Given the server is running
When a user opens the dashboard routes
Then all pages return successfully and static assets are served without external frontend dependencies
And dashboard theme assets use the current tokenized operations palette rather than hard-coded legacy palette values

### F-010-S002: Status APIs

Given repos, roles, jobs, and quality artifacts exist
When dashboard API endpoints are called
Then they return structured state for status, repos, roles, and repo-visible quality score

### F-010-S003: Shared Control Actions

Given the orchestrator is running
When dashboard controls call pause, resume, restart, stop, scan, or run-role
Then the same server control path used by runtime controls is invoked and method errors are reported

Given dashboard stop is called over HTTP
When the stop request is accepted
Then the handler returns success before dashboard HTTP shutdown begins
And the server loop exits through the normal graceful shutdown path without leaving the `start` or `serve` process running
And any shutdown failure is reported through the command/log path rather than by deadlocking the dashboard request

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
And the current work line reports the active job phase with phase age, including `waiting for model response` during non-streaming LLM calls

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

### F-010-S009: Terminal Dashboard

Given `serve`, `start`, or `run` is attached to an interactive TTY
When the command runs without `--debug`
Then the terminal redraws a dashboard-style view with state, current work, recent events, blocker summaries, controls, dashboard URL when available, and command log path
And slow local model responses remain visibly active as `waiting for model response` rather than staying at `inference ready`

Given an operator passes `--debug`
When the command runs
Then verbose trace and slog output streams inline while durable command logs remain available

### F-010-S010: Product Spec Governance

Given next-generation dashboard work is planned
When an agent scopes, designs, implements, reviews, or validates that work
Then [dashboard-control-plane.md](../product-specs/dashboard-control-plane.md) is the governing product contract
And the existing embedded dashboard is described only as legacy/current implementation until migration evidence exists
And no document claims the TanStack dashboard is implemented before code and evidence exist

### F-010-S011: External Node And pnpm Prerequisites

Given the TanStack dashboard sidecar is requested
When Node.js is absent, is not `24.x`, pnpm is absent, or pnpm is not `11.1.1`
Then MARS reports the installed versions when known
And reports the required versions
And gives a concrete remediation command or install path
And does not bundle Node, download Node, install Node, or run a package-manager installer for the user
And the core orchestrator remains usable without the TanStack dashboard

### F-010-S012: Local-Admin Auth

Given the TanStack dashboard is enabled
When an unauthenticated request reaches any dashboard route, dashboard API, event stream, feedback endpoint, roster proposal action, model proposal action, or command endpoint
Then the Go gateway denies dashboard data and routes the user to local-admin login

Given a local admin is authenticated
When a mutating dashboard request is submitted
Then the request must include valid CSRF protection tied to the session
And the command or proposal records the authenticated local admin as the actor

### F-010-S013: TanStack Sidecar Behind Go Gateway

Given Node `24.x`, `pnpm@11.1.1`, and local-admin auth are available
When `mars serve` starts the planned dashboard generation
Then the Go gateway remains the public dashboard listener
And the TanStack Start sidecar listens only behind that gateway
And the gateway enforces auth before proxying dashboard routes or serving dashboard APIs
And the sidecar never owns queue, scoring, trust, safety, or orchestration state

### F-010-S014: Concurrent Nonblocking APIs

Given one or more agents are running long model calls, repo scans, release checks, or queue work
When an authenticated dashboard client requests Overview, Active Work, roster, model, telemetry, DORA, preview, or command-status data
Then the request uses cancellable read paths, snapshots, independent database access, or event streams
And the response is not blocked by orchestrator processing locks
And stale or partial data is reported as a typed unavailable state

Given a dashboard control action is submitted
When the action is accepted
Then the HTTP response returns an asynchronous command id
And command status is visible through the authenticated event stream or status endpoint

### F-010-S015: Overview

Given agents, tickets, telemetry, models, and quality signals exist
When an authenticated operator opens Overview
Then the page shows active agents, roles, domains, modes, current work, owner, repo, duration, queue age, run status, issues, telemetry warnings, token usage, DORA cards, quality score links, and models being used

Given any required data source is missing or unavailable
When Overview renders
Then the affected section explains the unavailable state without hiding the rest of the page

### F-010-S016: Active Work Renderer

Given an agent is working a ticket or task
When an authenticated operator opens Active Work
Then the dashboard renders title, context, BDD scenario ids, acceptance criteria, blockers, evidence, affected files, owner, and next action in a polished work-focused layout
And the original repo artifact remains linked and authoritative

### F-010-S017: Adaptive Active Work Preview

Given active work has a preview provider
When the provider is a web app, mobile app, API, cloud service, distributed system, library, or CLI flow
Then the dashboard chooses the matching preview mode and shows the best live or artifact-backed evidence available

Given a live preview cannot be started
When alternative evidence exists
Then screenshots, videos, route inventories, traces, logs, command transcripts, environment links, or build artifacts are shown with a clear unavailable state for the live preview

### F-010-S018: Feedback Mailbox

Given an authenticated operator submits feedback on active work
When a stable anchor exists
Then the feedback is stored against route, selector, file, line, log span, trace id, screenshot coordinate, API route, or environment link

Given no stable anchor exists
When feedback is submitted
Then the feedback is stored against the active work item and current run
And it is injected at the next safe agent boundary rather than silently mutating durable repo files

### F-010-S019: Agent Roster

Given a target repo has configured agents
When an authenticated operator opens Agent Roster
Then every role shows prompt source, domain, mode, tools, guardrails, triggers, trust, scores, recent outcomes, token usage, model tier, model override, and availability

Given an operator requests a prompt, roster, tool, guardrail, trigger, or model-routing change
When the change is submitted
Then MARS creates a draft code-host proposal or local patch preview with files, rationale, validation, and rollback notes
And it does not silently mutate role source files from the dashboard

### F-010-S020: Available Models

Given local registry models, cached model files, llama.cpp endpoints, Ollama models, OpenAI-compatible endpoints, or configured cloud providers exist
When an authenticated operator opens Available Models
Then each model shows provider, name, local/cloud kind, endpoint, health, context window when known, role or tier eligibility, benchmark evidence when available, current usage, and unavailable reason

Given an operator adds a model or changes model routing
When the change is submitted
Then MARS creates an explicit override or registry-change proposal with validation requirements
And a model is not promoted as a default solely because it is reachable

### F-010-S021: GitHub-Derived DORA

Given `.harness/dashboard.yaml` configures `dashboard.dora.window_days`, `dashboard.dora.deployment_workflows`, and `dashboard.dora.branch`
When GitHub workflow evidence is available
Then Deployment Frequency counts successful terminal configured workflow runs inside the selected window
And Lead Time For Changes measures earliest commit timestamp in the range from the prior successful deployment SHA to the current successful deployment SHA through workflow completion
And Change Failure Rate divides failed, cancelled, or timed-out configured workflow runs by all terminal configured workflow runs
And Mean Time To Restore measures from a failed configured workflow run to the next successful configured workflow run for the same workflow and branch

Given GitHub auth, remote configuration, DORA config, matching workflow runs, history, permissions, or rate limit budget is missing
When DORA cards render
Then each affected metric shows the typed unavailable state and missing evidence

### F-010-S022: Legacy Dashboard Migration

Given the TanStack dashboard ships in slices
When legacy dashboard routes, APIs, or assets are retained, redirected, or removed
Then the migration state is documented
And operators receive clear behavior for old URLs
And tests prove the current dashboard implementation is not falsely described as the new control plane

### F-010-S023: Loopback Listener Boundary

Given `start`, `serve`, or the dashboard is created without an explicit safe
listener
When the control and dashboard servers bind
Then each listener resolves to an IPv4 or IPv6 loopback address
And wildcard, LAN, or arbitrary-host dashboard/control binds fail with
actionable authenticated-gateway guidance
And local health and control operation remain available when optional GitHub
webhook integration is disabled

## Out of Scope

- Hosted SaaS dashboard operation.
- Bundling Node or auto-installing Node or pnpm.
- Anonymous dashboard access for the TanStack control plane.
- A dashboard quality score that diverges from `docs/QUALITY_SCORE.md`.
- Browser-only controls that bypass server methods.
- Silent prompt, role, model, or guardrail mutations from the frontend.

## Descoped Scenarios

None.

## Evidence

- F-010-S001: `go test ./internal/dashboard -run 'TestDashboard_allPagesReturn200|TestDashboard_staticAssets|TestDashboard_themeAvoidsLegacyBluePalette'`
- F-010-S002: `go test ./internal/dashboard -run 'TestDashboard_statusEndpoint|TestDashboard_reposEndpoint'` and `go test ./internal/serve -run TestServer_qualityScoreAPIServesRepoArtifact`
- F-010-S003: `go test ./internal/dashboard -run 'TestDashboard_(pause|resume|restart|stop|scan|runRole)Endpoint'` and `go test ./internal/serve -run TestServer_dashboardStopEndpointStopsStart`
- F-010-S004: `go test ./internal/dashboard -run TestDashboard_sseConnection`
- F-010-S005: `go test ./internal/dashboard -run TestDashboard_emergencyStop`
- F-010-S006: `go test ./internal/dashboard -run TestDashboard_missingModuleEmptyState`
- F-010-S007: `go test ./internal/ui` and planned E2E evidence comparing CLI key listener and HTTP controls
- F-010-S008: `go test ./internal/dashboard -run TestDashboard_missingModuleEmptyState`; API decision-history evidence remains planned
- F-010-S009: `go test ./internal/ui` and `go test ./cmd/mars -run 'TestRunStartServeExposeDebugAndLogFileFlags|TestStartCommandInitializesRegistersSeedsAndStops'`
- F-010-S010: `go test ./internal/docsconsistency` verifies the product spec is indexed and docs do not claim implementation early
- F-010-S011: planned prerequisite tests for Node `24.x` and `pnpm@11.1.1` detection and remediation
- F-010-S012: planned auth, session, CSRF, and protected-route tests
- F-010-S013: planned gateway and sidecar integration tests
- F-010-S014: planned concurrent API tests while orchestrator work is active, plus async command-status tests
- F-010-S015: planned Overview route, data, unavailable-state, and browser visual tests
- F-010-S016: planned Active Work renderer tests against representative ticket markdown
- F-010-S017: planned preview provider tests for web, artifact, API, mobile, cloud, distributed-system, library, and CLI cases
- F-010-S018: planned feedback mailbox tests for anchored and unanchored feedback
- F-010-S019: planned Agent Roster route and proposal-generation tests
- F-010-S020: planned model catalog, health, unavailable-state, override, and proposal tests
- F-010-S021: planned DORA fixture tests for successful, failed, insufficient-history, missing-auth, no-remote, missing-config, rate-limit, and permission-denied states
- F-010-S022: planned migration tests for retained, redirected, and removed legacy dashboard routes
- F-010-S023: T-057 Engineer tests prove direct/default dashboard and control addresses use explicit loopback, reject wildcard/LAN/hostnames before bind, preserve loopback health without webhook policy, and retain loopback-only ephemeral fallback. Installed-binary and independent review evidence remain pending.
