---
id: MH-013
title: SQLite job queue with per-repo serialization and idempotency
priority: high
complexity: large
source: delivery-schedule M5a
created: 2026-04-11
---

# MH-013: SQLite job queue with repo_id from day one and worker dispatcher

## Context

M5a introduces durable work: webhook- and scheduler-driven jobs must not lose state on restart, must not stampede per repository, and must support idempotent replays after crashes.

## Requirements

- Schema includes `repo_id` (stable string key) on every job row from first migration
- Lifecycle states: `pending`, `claimed`, `running`, `completed`, `failed`, `cancelled` with valid transitions enforced in code
- Claim API: transactional `UPDATE … WHERE status='pending' AND repo_id=?` with lease (`claimed_until`) to prevent stuck workers
- Per-repo serialization: at most one `running` job per `repo_id` (configurable strict mode)
- Idempotency: unique index on `(repo_id, idempotency_key)`; second enqueue returns existing job id
- Worker pool: configurable concurrency; graceful shutdown waits for in-flight jobs or respects timeout
- Dispatcher integrates with normalized events from MH-012 (interface only if needed to avoid circular deps)
- Job TTL — failed/completed jobs retained for 30 days, then pruned

## Acceptance Criteria

### Functional (happy path)
- [ ] Enqueue → claim → run → complete persists across process restart (SQLite file)
- [ ] Two jobs for same `repo_id` run strictly one-after-another when strict serialization enabled
- [ ] Same `idempotency_key` re-enqueued returns same job without duplicate execution

### Edge cases and negative paths
- [ ] Worker death mid-job: lease expires, job becomes reclaimable or moves to `failed` per policy (documented)
- [ ] Cancel transitions a `pending` job to `cancelled`; `running` cancel requests cooperative stop flag
- [ ] SQLite busy: retry with backoff; no indefinite spin
- [ ] Completed/failed jobs older than 30 days are pruned automatically

### Non-goals
- [ ] Distributed multi-node queue (single process / single DB file)
- [ ] Priority inheritance across repos

### Observability, docs, and regressions
- [ ] Stress test: N workers, M repos, verifies serialization invariant
- [ ] Migration test from empty DB to v1 schema
- [ ] Metrics: queue depth per repo, claim latency, job duration histogram

> Note: Schedule uses queued/running/completed/failed; implementation uses pending/claimed/running/completed/failed/cancelled for lease pattern precision.
