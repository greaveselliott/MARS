---
id: MH-053
title: Protect the TanStack dashboard with local-admin auth and Go gateway checks
priority: high
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S012", "F-010-S013"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-051", "MH-052"]
trace_id: none
next_action: "Design the local-admin credential store, session model, CSRF model, and gateway protection boundary before sidecar proxy work."
dedupe_key: dashboard-control-plane:local-admin-auth-gateway
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-051", "MH-052"]
---

# MH-053: Protect the TanStack dashboard with local-admin auth and Go gateway checks

## Context

The dashboard will expose sensitive runtime state, control actions, feedback,
role metadata, model routing, and optional GitHub-derived metrics. The TanStack
control plane therefore requires local-admin authentication for every dashboard
surface. The Go gateway must remain the only trusted boundary.

## BDD Scenario IDs

- F-010-S012
- F-010-S013

## Affected Docs/Code Areas

- `internal/dashboard/`
- `internal/serve/`
- `internal/config/`
- `internal/trace/`
- `cmd/mars-harness/`
- future `web/dashboard/`
- `docs/product-specs/dashboard-control-plane.md`
- `docs/design-docs/dashboard.md`
- `docs/features/F-010-dashboard-control-plane.md`

## Acceptance Criteria

- [ ] Local-admin credential storage is designed and implemented with password hashing, created timestamp, and disabled timestamp.
- [ ] Session cookies are HttpOnly, same-site strict, expiring, and server-invalidated.
- [ ] Mutating dashboard requests require CSRF protection tied to the session.
- [ ] Unauthenticated requests cannot read dashboard routes, APIs, event streams, feedback endpoints, roster proposal actions, model proposal actions, or command endpoints.
- [ ] The Go gateway enforces auth before serving dashboard data or proxying to the TanStack sidecar.
- [ ] Authenticated mutating actions record the local admin actor.
- [ ] Login, logout, expiry, denied-route, and CSRF failure states are user-visible and test-covered.

## Non-Goals

- Hosted multi-user identity.
- Remote single sign-on.
- Anonymous read-only dashboard mode.
- Changing queue, scoring, trust, or guardrail business logic.

## Evidence Requirements

- Unit tests for credential storage and password verification.
- Integration tests for protected routes, protected APIs, protected event streams, login, logout, expiry, and CSRF failure.
- Browser verification that unauthenticated dashboard access shows login rather than sensitive data.
- Trace or command-status evidence that mutating actions record actor attribution.

## Next Action

Write the auth design in the dashboard design doc if storage or session choices
need extra detail, then implement the smallest protected route and denied-state
test.
