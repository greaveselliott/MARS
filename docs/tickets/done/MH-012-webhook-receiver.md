---
id: MH-012
title: Webhook HTTP server with HMAC validation and event normalization
priority: high
complexity: medium
source: delivery-schedule M4
created: 2026-04-11
---

# MH-012: Webhook HTTP server with HMAC signature validation, event normalization, deduplication

## Context

The harness must react to GitHub asynchronously. A small HTTP server validates authenticity, normalizes payloads into internal events, and drops duplicate deliveries so downstream work stays idempotent.

## Requirements

- HTTP server with configurable bind address; health route separate from webhook path
- HMAC-SHA256 validation using `X-Hub-Signature-256` (constant-time compare); reject missing/invalid signature before body parse on large payloads
- Normalize to internal envelope: `event_type`, `action`, `repo`, `installation_id`, `delivery_id`, `payload` subset needed by dispatcher
- Supported GitHub types: `push`, `pull_request`, `check_suite`, `workflow_run`, `merge_group`, `issue_comment`
- Deduplication: key `(X-GitHub-Delivery, app_id)` persisted (SQLite or in-memory LRU with SQLite backing for restart safety); duplicates return 200 without enqueue; reject replays within 1-hour window

## Acceptance Criteria

### Functional (happy path)
- [ ] Valid signed POST for each supported type produces exactly one normalized event record for a unique `X-GitHub-Delivery`
- [ ] Duplicate delivery with same delivery id is acknowledged and not re-processed
- [ ] Health endpoint returns 200 when server and DB connectivity (if used) are OK

### Edge cases and negative paths
- [ ] Wrong secret → 401/403 with no body logged containing secrets
- [ ] Oversized body rejected with 413 before HMAC (configurable max)
- [ ] Unknown `X-GitHub-Event` header → 202 or 204 with structured log “ignored” (no crash)
- [ ] Clock skew does not break signature path (no time-based MAC)

### Non-goals
- [ ] Full GitHub Apps “org” event catalog
- [ ] Automatic TLS termination (assume reverse proxy or local dev HTTP)

### Observability, docs, and regressions
- [ ] Tests with golden payloads per event type; property test that duplicate deliveries hit dedupe path
- [ ] Metrics: deliveries accepted, ignored, rejected, deduped
- [ ] Runbook snippet: `smee.io` or ngrok for local webhook testing
