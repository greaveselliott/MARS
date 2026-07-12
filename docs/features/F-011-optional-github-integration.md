# F-011: Optional GitHub Integration

- Feature ID: F-011
- Goals: G-001, G-003
- Status: partially-passing
- Owner: Release Manager

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-011-S001 - GitHub client supports PAT and App modes with actionable configuration validation.
2. F-011-S002 - App setup serves the manifest flow and exchanges setup codes.
3. F-011-S003 - Webhook receiver requires transport authenticity, explicit principal/repository/branch authorization, same-repository provenance, bounded durable replay protection, and supported event policy.
4. F-011-S004 - GitHub API operations create pull requests, check runs, status updates, and comments when configured.
5. F-011-S005 - Rate limits, server errors, and context cancellation are handled without corrupting local queue state.
6. F-011-S006 - Optional integration never blocks local-first operation when credentials are absent.
7. F-011-S007 - Remote-code-host signals feed telemetry and orchestration without replacing strict trunk commits.

## Scenarios

### F-011-S001: Client Configuration

Given GitHub integration configuration is supplied
When a client is created
Then PAT mode, App mode, default base URLs, and unknown modes are validated with actionable errors

### F-011-S002: App Setup Flow

Given a user runs GitHub setup
When the setup page, manifest, callback, and code exchange are exercised
Then the app credentials are preserved or populated without timing out silently

### F-011-S003: Verified Webhooks

Given GitHub sends webhook deliveries
When the receiver handles disabled, signed, tampered, missing-header,
oversized, replayed, unknown, unauthorized-actor, wrong-repository,
wrong-branch, fork-origin, issue-comment, or wrong-method requests
Then local operation remains healthy when GitHub ingress is disabled
And enabled ingress requires a secret of sufficient length, valid HMAC over the
exact body, a trusted numeric actor ID, an exact registered `owner/repo`, the
registered branch, and same-repository event provenance
And fork-derived events and issue comments never dispatch autonomous work
And rejected events cannot poison replay state
And a delivery/body identity can enqueue at most once across concurrency,
completion, and restart

### F-011-S004: Remote Reporting Operations

Given optional GitHub credentials are configured
When the harness posts coordination signals
Then it can create PRs, create and update check runs, and post comments through the GitHub API client

### F-011-S005: API Resilience

Given GitHub responds with rate limits, server errors, unexpected statuses, or cancellation
When the client sends a request
Then retry and error behavior is bounded and explicit

### F-011-S006: Local-First Without Credentials

Given GitHub credentials are not configured
When setup, run, start, serve, doctor, or release paths are used
Then core local harness operation continues and optional integration is reported as unavailable rather than complete
And process health remains available while `/webhook` reports service unavailable and dispatches nothing

### F-011-S007: Signals Around Strict Trunk

Given GitHub events, statuses, or comments are available
When they feed orchestration or telemetry
Then they augment local strict-trunk commits and do not turn branch-based review into the default delivery model

## Out of Scope

- Requiring GitHub for normal local operation.
- Claiming integration health before credentials, webhook delivery, and status/comment behavior are validated.
- Replacing repo-owned docs and commits with remote-only state.

## Descoped Scenarios

None.

## Evidence

- F-011-S001: `go test ./internal/github -run TestNewClient`
- F-011-S002: `go test ./internal/github -run 'TestRunSetup|TestAppManifest|TestExchangeManifestCode'`
- F-011-S003: current OSS-02 slice; pending HMAC, numeric actor, exact registered repository/branch, fork/comment policy, bounded replay, durable queue idempotency, and installed-binary evidence
- F-011-S004: `go test ./internal/github -run 'TestCreatePR|TestCreateCheckRun|TestUpdateCheckRun|TestPostComment'`
- F-011-S005: `go test ./internal/github -run 'TestClient_(rateLimit|serverError|context|unexpected)|TestRateLimitWait|TestBackoff'`
- F-011-S006: existing doctor/local command evidence plus pending OSS-02 proof that health/local operation continue while disabled webhook ingress returns unavailable and dispatches nothing
- F-011-S007: planned orchestration/telemetry E2E evidence for GitHub events around strict-trunk runs
