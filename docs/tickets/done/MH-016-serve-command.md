---
id: MH-016
title: Wire subsystems into mars-harness serve with health and graceful shutdown
priority: high
complexity: medium
source: delivery-schedule M5b
created: 2026-04-11
---

# MH-016: `mars-harness serve` — startup sequence, graceful shutdown, `/healthz`

## Context

Operators need one command that brings inference, GitHub integration, webhooks, scheduling, and queue workers online in a predictable order with observable health.

## Requirements

- `serve` subcommand: flags for bind addresses, bundle path, database path, concurrency, shutdown timeout
- Startup sequence: hardware/model inference check → internal health → webhook listener (MH-012) → scheduler (MH-014) → worker dispatcher (MH-013)
- Graceful shutdown on SIGINT/SIGTERM: stop accepting webhooks, drain queue to configured depth or timeout, close DB
- HTTP `/healthz` JSON: versions, uptime, inference OK, GitHub auth mode, webhook last delivery time, queue depth per repo, scheduler next fires

## Acceptance Criteria

### Functional (happy path)
- [ ] Cold start completes with all green checks when dependencies configured
- [ ] `/healthz` reflects live metrics after synthetic webhook in dev
- [ ] SIGTERM triggers shutdown sequence logged step-by-step; in-flight job finishes or times out per flag

### Edge cases and negative paths
- [ ] Inference unavailable → serve refuses to start OR starts “degraded” with explicit `degraded_reasons[]` (behaviour chosen and documented)
- [ ] Port collision on webhook bind → actionable error suggesting `--webhook-addr`
- [ ] Double start on same DB file surfaces SQLite lock error with remediation

### Non-goals
- [ ] Dashboard UI (M9 / MH-024)
- [ ] TLS certificate management

### Observability, docs, and regressions
- [ ] Smoke test script: curl `/healthz` in CI with mocked GitHub
- [ ] Structured startup logs with correlation id per subsystem init
- [ ] README “day 2 operations” section updated
