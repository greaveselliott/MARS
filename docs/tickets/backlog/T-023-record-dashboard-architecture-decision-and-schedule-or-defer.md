---
id: T-023
title: Record dashboard architecture decision and schedule-or-defer outcome
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "foundation-maintainer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Write the dashboard architecture AD resolving the T-010 single-binary versus TanStack sidecar conflict and record schedule-or-defer in the active plan."
source: Foundation improvement plan Phase 1 WS-B (provisional T-023)
created: 2026-06-11
depends_on: []
---

# T-023: Record dashboard architecture decision and schedule-or-defer outcome

## Context

The 2026-06-11 foundation review found the dashboard epic ambiguous: T-010 and MH-051 through MH-061 plan a TanStack Start sidecar requiring external Node 24.x and pnpm, while AGENTS.md key constraints state single-binary distribution with no npm and no external runtime dependencies. AD-156 records the sidecar design but the constraint conflict and the scheduling decision are not explicitly resolved, so T-010 still warns it must not be picked up until the trade-off is explicit.

Ownership classification: foundation-owned product-surface decision.

## Requirements

- Write a dashboard architecture AD in docs/design-docs/dashboard.md that explicitly reconciles the single-binary constraint with the optional sidecar (scope of the no-Node constraint, default embedded dashboard, sidecar as optional operator-installed surface).
- Index the AD in docs/design-docs/index.md.
- Record an explicit schedule-or-defer decision for the dashboard epic in the active plan, with rationale and a concrete start condition.
- Update T-010 next_action to reference the AD. MH-051 executes only if the decision is schedule.

## Affected Files

- docs/design-docs/dashboard.md
- docs/design-docs/index.md
- docs/exec-plans/active/current-operating-plan.md
- docs/tickets/backlog/T-010-replace-dashboard-ui-with-shadcn-ui-component-system.md

## Acceptance Criteria

### Functional (happy path)
- [ ] AD resolves the constraint conflict explicitly and is indexed.
- [ ] Active plan states schedule-or-defer with the AD reference and start condition.

### Edge cases and negative paths
- [ ] No dashboard ticket remains ambiguous about whether it may start.

### Non-goals
- Implementing any dashboard runtime code in this slice.

### Observability, docs, and regressions
- [ ] go test ./internal/docsconsistency/... ./internal/docsync/... passes.
