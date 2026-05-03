# F-010: Dashboard And Control Plane

- Feature ID: F-010
- Goals: G-001, G-003
- Status: partially-passing
- Owner: COO

## Scenario Schedule

1. F-010-S001 - Dashboard pages and static assets are served from the embedded binary.
2. F-010-S002 - Health, status, repos, repo roles, and quality score APIs expose orchestrator state.
3. F-010-S003 - Pause, resume, restart, stop, scan, and run-role controls call shared server methods.
4. F-010-S004 - Server-sent events stream dashboard updates.
5. F-010-S005 - Emergency stop is available through the dashboard and reports callback errors.
6. F-010-S006 - Empty states explain missing modules or data without crashing the dashboard.
7. F-010-S007 - CLI interactive controls and dashboard controls remain behaviorally aligned.

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
- F-010-S007: planned E2E evidence comparing CLI key listener and HTTP controls
