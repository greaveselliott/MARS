---
id: MH-011
title: GitHub App manifest flow with PAT detection and fallback
priority: high
complexity: medium
source: delivery-schedule M4
created: 2026-04-11
---

# MH-011: GitHub App manifest flow (browser wizard, callback server, credential storage)

## Context

Operators must install the harness GitHub App without hand-editing PEMs and webhook secrets. A guided flow reduces misconfiguration and unlocks M4 webhook and API features.

## Requirements

- Local HTTP callback server (fixed high port or ephemeral + printed URL) to receive manifest exchange `code`
- Open browser to GitHub “new manifest” URL with manifest payload for required permissions/events
- Exchange `code` for `id`, `pem`, `webhook_secret`; persist to local secrets store (file permissions 0600)
- Detect existing PAT-only configuration; offer “continue with PAT” vs “upgrade to App” without clobbering secrets silently
- Idempotent re-run: if App already registered with same `app_id`, validate files and exit success with summary

## Acceptance Criteria

### Functional (happy path)
- [ ] Fresh setup completes: manifest URL → browser → callback → credentials on disk → printed next steps (webhook URL, events)
- [ ] Stored credentials load into MH-010 client without manual path edits
- [ ] PAT-only env is detected and documented in CLI output (no false “App ready” state)

### Edge cases and negative paths
- [ ] User cancels browser flow: callback timeout with clear “run again” message; no partial secrets written
- [ ] Port already in use: automatic fallback port or explicit `--listen` flag with validation
- [ ] Exchange fails (invalid code): error includes GitHub error JSON message when present

### Non-goals
- [ ] Org-wide bulk installation across many orgs from one wizard
- [ ] Rotating webhook secret via API (manual rotate documented only)

### Observability, docs, and regressions
- [ ] Integration test using `httptest` simulating manifest callback and exchange
- [ ] Docs: prerequisites (org admin vs repo admin), required events list, troubleshooting TLS/localhost
- [ ] `doctor` (MH-028) hooks referenced as follow-up, not blocking this ticket
