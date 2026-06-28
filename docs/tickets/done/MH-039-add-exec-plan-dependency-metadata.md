---
id: MH-039
title: Add exec-plan dependency metadata
priority: high
complexity: small
kind: intervention-debt
dedupe_key: "public-example"
source: user operating-model clarification 2026-05-02
created: 2026-05-02
completed: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars"
  target: "exec-plans"
  category: "plan_hygiene"
  severity: "high"
---

# MH-039: Add exec-plan dependency metadata

## Context

The single-active-plan lifecycle needs more than priority. Exec plans should be
sequenced like tickets, including dependency and blocker metadata, so agents do
not promote a lower-readiness plan just because its priority looks high.

## Requirements

- Require active and backlog exec plans to declare dependencies.
- Require active and backlog exec plans to declare blockers.
- Require active and backlog exec plans to declare related tickets.
- Mirror the fields into generated target harness docs and current-plan seed.
- Add docs-consistency coverage so the metadata does not drift away.

## Affected Files

- `docs/exec-plans/`
- `docs/design-docs/self-improvement.md`
- `docs/design-docs/index.md`
- `docs/product-specs/product-surface.md`
- `internal/scanner/init.go`
- `internal/scanner/scanner_test.go`
- `internal/docsconsistency/exec_plans_test.go`

## Acceptance Criteria

### Functional

- [x] Active exec plan declares priority, dependencies, blockers, and related tickets.
- [x] Backlog exec plans declare priority, dependencies, blockers, and related tickets.
- [x] Generated target harnesses seed the same fields.
- [x] Docs-consistency tests enforce dependency metadata on active/backlog plans.

### Edge cases and negative paths

- [x] Plans with no dependencies can explicitly say `None`.
- [x] Plans with no blockers can explicitly say `Nothing`.
- [x] Superseded plans remain lineage-only and are not required to carry active/backlog metadata.

### Observability, docs, and regressions

- [x] AD-073 documents dependency metadata as part of the one-active-plan rule.
- [x] Product specs describe priority plus dependencies and blockers.
- [x] No unchecked acceptance criteria remain in this completed ticket.
