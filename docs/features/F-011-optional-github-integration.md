# F-011: Optional GitHub Integration

- Feature ID: F-011
- Goals: G-001, G-003
- Status: partially-passing
- Owner: Release Manager

## Scenario Schedule

1. F-011-S001 - GitHub client supports PAT and App modes with actionable configuration validation.
2. F-011-S002 - App setup serves the manifest flow and exchanges setup codes.
3. F-011-S003 - Webhook receiver verifies signatures, deduplicates deliveries, rejects invalid requests, and accepts supported events.
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
When the receiver handles signed, tampered, missing-header, oversized, duplicate, unknown, or wrong-method requests
Then valid events are accepted, invalid requests are rejected, and duplicate deliveries are ignored

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
- F-011-S003: `go test ./internal/github -run TestWebhookHandler`
- F-011-S004: `go test ./internal/github -run 'TestCreatePR|TestCreateCheckRun|TestUpdateCheckRun|TestPostComment'`
- F-011-S005: `go test ./internal/github -run 'TestClient_(rateLimit|serverError|context|unexpected)|TestRateLimitWait|TestBackoff'`
- F-011-S006: `go test ./internal/doctor` plus local command tests that do not require GitHub credentials
- F-011-S007: planned orchestration/telemetry E2E evidence for GitHub events around strict-trunk runs
