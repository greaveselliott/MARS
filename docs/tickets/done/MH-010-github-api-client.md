---
id: MH-010
title: GitHub API client with App auth and PAT fallback
priority: high
complexity: large
source: delivery-schedule M4
created: 2026-04-11
---

# MH-010: GitHub API client with App auth (JWT → installation token), PAT fallback

## Context

M4 needs a single, well-tested GitHub REST client for PR lifecycle, checks, and comments. GitHub App JWT exchange to installation tokens is the primary path; PAT remains a supported fallback for bootstrap and local dev.

## Requirements

- JWT minting for the GitHub App (RS256) with configurable `iat`/`exp` skew
- Exchange JWT for short-lived installation access token; cache until expiry minus safety margin
- PAT auth path when `GITHUB_TOKEN` (or equivalent) is set without App credentials
- PR create, update (title/body/base), and issue-comment on PR threads
- Check runs: create, update, list for a commit SHA
- Rate limit handling: respect `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`; bounded exponential backoff; classify secondary vs primary limits in errors
- Structured errors (HTTP status, `message`, request id if present)

## Acceptance Criteria

### Functional (happy path)
- [x] Installation token is obtained from JWT and used for subsequent API calls until near expiry
- [x] PAT mode authenticates and completes a PR comment round-trip in a test repo
- [x] Create PR, push follow-up commit, update PR body/title via API
- [x] Create and transition a check run to success/failure on a known SHA

### Edge cases and negative paths
- [x] Expired installation token triggers transparent refresh without duplicate concurrent refreshes
- [x] 403 with SSO or suspended installation surfaces actionable error text
- [x] 403/429 responses obey `Retry-After` when provided; otherwise backoff using reset header
- [x] Missing App key or wrong `app_id` fails fast with configuration checklist in error

### Non-goals
- GraphQL API (REST only for this ticket)
- GitHub Enterprise Server quirks beyond configurable API base URL

### Observability, docs, and regressions
- [x] Unit tests with `httptest` for JWT exchange, token cache, rate-limit backoff, and PR/check helpers
- [x] Metrics or structured logs for token refresh and rate-limit waits (no secrets in logs)
- [x] Design doc or package README updated for auth modes and env vars
