---
id: MH-034
title: Implement active-plan hygiene checker
priority: high
complexity: medium
kind: intervention-debt
dedupe_key: "public-example"
source: MH-033
created: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "exec-plans"
  category: "plan_hygiene"
  severity: "high"
---

# MH-034: Implement active-plan hygiene checker

## Context

Mars Harness relies on repo-owned plans as the system of record, but stale active
plans can quietly misdirect future agents. The harness needs a mechanical check
that detects plan drift before it becomes operational confusion.

## Requirements

- Add a docs consistency or doctor check for active-plan hygiene.
- Detect more than one active exec plan and require waiting plans to live in `docs/exec-plans/backlog/` with priority.
- Detect active plans that are marked superseded without a pointer to the current plan.
- Detect active plans with stale status claims that conflict with completed ticket locations where practical.
- Detect unresolved `TBD`, stale verification notes, and old "current" language in active plans.
- Report actionable remediation rather than only failing.
- Keep the check local and dependency-light.

## Affected Files

- `internal/docsconsistency/`
- `internal/doctor/`
- `docs/exec-plans/`
- `.github/workflows/`

## Acceptance Criteria

### Functional

- [ ] `go test ./internal/docsconsistency/...` fails on a fixture with stale active-plan status.
- [ ] Docs consistency fails when more than one active exec plan exists.
- [ ] Backlog exec plans without priority fail the check.
- [ ] `mars-harness doctor --repo .` reports active-plan hygiene warnings or ok status.
- [ ] Superseded plans with a current-plan pointer do not fail the check.

### Edge cases and negative paths

- [ ] Completed historical plans are not treated as active-plan failures.
- [ ] Empty `completed/.gitkeep` does not trigger a false positive.
- [ ] The check can run without network or external services.

### Observability, docs, and regressions

- [ ] Docs explain how to supersede, reconcile, or complete an exec plan.
- [ ] Tests cover stale active plans, valid active plans, and superseded plans.
