---
id: MH-038
title: Enforce one active exec plan
priority: high
complexity: small
kind: intervention-debt
dedupe_key: "public-example"
source: user operating-model change 2026-05-02
created: 2026-05-02
completed: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "exec-plans"
  category: "plan_hygiene"
  severity: "high"
---

# MH-038: Enforce one active exec plan

## Context

The source repo had multiple files under `docs/exec-plans/active/`, including
more than one plan claiming active status. That made exec plans behave like a
second in-progress queue and could mislead agents about which work to execute.

## Requirements

- Introduce a ticket-like exec-plan lifecycle: backlog, active, completed, and superseded.
- Keep exactly one active plan in `docs/exec-plans/active/`.
- Move waiting plans to `docs/exec-plans/backlog/` with explicit priority.
- Move historical baseline plans to `docs/exec-plans/superseded/`.
- Mirror the rule into generated target harnesses.
- Add docs-consistency coverage for the single-active-plan rule.

## Affected Files

- `docs/exec-plans/`
- `docs/design-docs/self-improvement.md`
- `docs/design-docs/index.md`
- `docs/product-specs/product-surface.md`
- `internal/scanner/init.go`
- `internal/scanner/scanner_test.go`
- `internal/docsconsistency/`

## Acceptance Criteria

### Functional

- [x] Only `docs/exec-plans/active/current-operating-plan.md` remains active.
- [x] Mars parity and model evaluation plans live in the plan backlog with priority.
- [x] Master execution plan and delivery schedule live in superseded lineage.
- [x] Initialized target harnesses receive the same lifecycle and current-plan seed.

### Edge cases and negative paths

- [x] Docs consistency fails if more than one active plan exists.
- [x] Backlog plans without priority fail docs consistency.
- [x] QA, security, and dependency reports no longer write into the active exec-plan directory.

### Observability, docs, and regressions

- [x] The design decision is indexed as AD-073.
- [x] Product specs describe the one-active-plan lifecycle.
- [x] The current operating plan records the plan queue.
- [x] No unchecked acceptance criteria remain in this completed ticket.
