---
id: MH-050
title: Add persistent store upgrade fixtures
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: TBD
owner: TBD
last_attempt: TBD
blocker: none
blocked_by: []
trace_id: TBD
next_action: Add versioned legacy SQLite schema fixtures and CI coverage for every persistent store init path.
dedupe_key: intervention-debt:persistent-store-upgrade-fixtures
source: human-follow-up 2026-05-03 queue legacy schema regression
created: 2026-05-03
---

# MH-050: Add persistent store upgrade fixtures

## Context
`mars-harness start --repo /path/to/target-repo` failed on 2026-05-03 while opening an existing per-repo queue database:

```text
queue: init schema: SQL logic error: no such column: concurrency_group (1)
```

The queue store had a compatibility column backfill, but tests only exercised fresh databases. On old `jobs` tables, index creation ran before the `concurrency_group` column was added. The immediate regression is covered by `TestQueueOpenMigratesLegacyJobsTableBeforeCreatingIndexes`; this ticket tracks the broader quality gap.

## Requirements
- Add versioned legacy SQLite schema fixtures for all persistent stores with `initSchema` or equivalent schema creation paths.
- Exercise opening each legacy fixture with current code and assert required columns, indexes, and basic read/write operations still work.
- Include at least one fixture where a newly indexed column is missing before migration.
- Document the expected migration ordering rule: create base tables, backfill columns, then create indexes/triggers/views that depend on those columns.
- Make the fixture tests part of normal `go test ./...` CI.

## Affected Files
- `internal/queue/`
- `internal/telemetry/`
- `internal/trace/`
- `internal/scoring/`
- `internal/trust/`
- `internal/evolution/`
- `internal/orgstate/`
- Any other package that owns a persistent SQLite schema.

## BDD Evidence
- Scenario IDs: none
- Evidence links: `go test ./...`
- Verified by: command

## Acceptance Criteria

### Functional
- [ ] Every persistent SQLite store has at least one legacy-schema open test.
- [ ] Tests fail if a migration creates an index before its dependent column exists.
- [ ] Fixture coverage includes the queue `concurrency_group` class of failure.

### Edge cases and negative paths
- [ ] Tests cover missing columns with defaults.
- [ ] Tests cover existing old rows surviving migration.
- [ ] Tests cover indexes that depend on newly added columns.

### Non-goals
- Replacing SQLite with a standalone migration framework.
- Changing runtime database paths or per-repo database isolation.

### Observability, docs, and regressions
- [ ] Migration-ordering expectations are documented near the fixtures or store test helpers.
- [ ] CI runs the fixture tests by default.

## Notes
This is intervention debt because a human caught a fundamental upgrade-path regression that fresh-database tests could not see.
