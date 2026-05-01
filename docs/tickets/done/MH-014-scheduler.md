---
id: MH-014
title: Cron scheduler from manifest with timezone and catch-up
priority: medium
complexity: small
source: delivery-schedule M5a
created: 2026-04-11
---

# MH-014: Cron scheduler parsing from manifest, timezone-aware, missed schedule handling

## Context

Roles like nightly QA need manifest-declared schedules without an external cron daemon. Parsing must be deterministic and safe across DST and process downtime.

## Requirements

- Schedule sources: config file entries and per-repo `.harness/schedules.yaml`. Parse cron expressions and timezone (IANA name, e.g. `Europe/London`) from those sources
- Evaluate next run times using a well-vetted cron library or documented subset (minute/hour/day/month/dow)
- On startup, detect missed windows since last persisted `last_fire_at`; configurable policy: `fire_once` catch-up vs `skip_missed`
- Emit enqueue calls to MH-013 with stable idempotency keys derived from `(repo_id, schedule_name, window_start)`

## Acceptance Criteria

### Functional (happy path)
- [x] Daily schedule fires at local wall time in configured timezone across a DST boundary (test with frozen clock)
- [x] After 3 days downtime, `fire_once` enqueues at most one catch-up per missed logical window per policy
- [x] Scheduler registers cleanly with dispatcher lifecycle (start/stop)

### Edge cases and negative paths
- [x] Invalid cron string fails bundle validation at load with file/line context
- [x] Unknown timezone fails fast with list of example valid names
- [x] Clock jump backward does not double-fire same window (idempotency keys prevent duplicate jobs)

### Non-goals
- Human natural language schedules (“every Tuesday afternoon”)
- Multi-repo fan-out from one manifest (single repo per process assumption)

### Observability, docs, and regressions
- [x] Unit tests with `github.com/benbjohnson/clock` or equivalent fake timer
- [x] Log line per evaluation: next fire time, last fire time, decision (fire/skip)
- [x] Manifest schema example added to docs
