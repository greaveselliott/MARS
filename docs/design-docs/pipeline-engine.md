# Pipeline Engine

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

Infrastructure for durable jobs: queueing, workers, scheduling, sandboxing, and operational controls (blast radius, emergency stop). Aligns with execution truth and progressive autonomy in [tenets.md](tenets.md).

## Context

The harness must run reliably on a single machine or small deployment **without** Redis, Postgres, or other mandatory external services for v1. State for jobs, traces, scores, and interventions must survive restarts and support future multi-repo scale-out.

The engine exposes a **single logical queue** per deployment in early milestones; fairness and isolation rules evolve as multi-tenant pressure appears.

## Key Design Decisions

### AD-009: SQLite for all persistent state

**SQLite** is the system of record for jobs, scores, interventions, traces, and related metadata. **Zero** mandatory external database dependency keeps install friction low and backup story simple (files).

Migrations are versioned; schema changes ship with the binary and apply on startup with clear logging.

### AD-010: `repo_id` from day one

Schema includes a **`repo_id`** column (and associated invariants) from the first migration—even if the product initially targets a single repo—so multi-repo installs do not require a painful migration later.

Foreign keys and indexes should be designed assuming **many repos** even when the UI hides that for v1.

### AD-214: Startup cleanup preserves SQLite WAL state

`start` and `serve` must not delete SQLite `-wal` or `-shm` sidecar files during automatic startup cleanup. SQLite WAL files can contain committed queue, repo registry, trace, and telemetry state that has not yet been checkpointed into the main database file. Deleting those sidecars after a failed bind or interrupted process can silently discard the previous run's bootstrap record and make a retry look like a fresh repo registration.

Startup cleanup now treats SQLite sidecars as recoverable database state. When a database file and sidecars are present, the harness opens the database with a busy timeout and asks SQLite to run a passive WAL checkpoint. If recovery/checkpointing fails, the sidecars are left in place and the operator sees a warning rather than losing state. Explicit destructive cleanup remains limited to operator-controlled flows such as `eject --apply`.

### Open topics

- **Job queue:** states (queued, running, succeeded, failed, cancelled), idempotency keys, lease/timeout semantics, dead-letter handling for poison jobs.
- **Worker dispatcher:** concurrency limits, fairness across repos, back-pressure when inference is saturated; integration with [local-inference.md](local-inference.md) health.
- **Cron scheduler:** manifest-driven schedules, jitter, missed tick behavior; clock skew assumptions documented.
- **Sandbox:** Linux namespaces/cgroups where available; **macOS fallback** with reduced isolation clearly documented and surfaced in UI.
- **Blast radius containment:** per-repo caps, directory allowlists, resource limits cohering with guardrails and Git allowed operations.
- **Emergency stop:** global and per-repo kill switch, draining in-flight jobs safely; operator CLI and optional file flag for air-gapped use.

## Discoveries

- 2026-05-21: `demo-temp-run60` reproduced a retry-after-bind-failure edge:
  the first sandboxed start registered the target and enqueued a CEO bootstrap
  job, then the escalated retry logged removal of `demo-temp-run60.db-wal` and
  `demo-temp-run60.db-shm` before registering the same path under a new repo
  ID. The final DB did not contain duplicate jobs because the WAL state had
  been discarded, which is worse than duplicate bootstrap work. Cleanup now
  preserves sidecars and lets SQLite recover/checkpoint them.
- 2026-05-20: A non-static `demo-api-run1` lifecycle replay showed the minute
  scheduler enqueueing a second Engineer while the first Engineer was still
  running. The queue already serialized claims per repo, but duplicate pending
  scheduled work still restarted the same implementation path after a runtime
  `max_turns` failure. Scheduler fire now checks for active pending, claimed, or
  running work with the same repo and role and skips instead of stacking another
  product worker.
- 2026-05-19: Live `demo-123` validation showed that operator shutdown during an active Dogfood job could cancel the worker context before the queue wrote the terminal failure state. The TUI showed the job as blocked, but SQLite still reported it as `running`. Worker finalization now uses a short fresh context for `completed`/`failed` state writes while leaving shutdown-canceled callbacks unable to enqueue more work.
