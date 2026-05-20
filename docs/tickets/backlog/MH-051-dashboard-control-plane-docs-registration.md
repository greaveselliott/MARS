---
id: MH-051
title: Register the dashboard control-plane product spec and epic docs
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-010-S010"]
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "pending docs consistency evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: []
trace_id: none
next_action: "Review the dashboard product spec, F-010 expansion, design-doc update, product-surface update, exec-plan registration, and ticket map; attach docs consistency evidence before moving this ticket to done."
dedupe_key: dashboard-control-plane:docs-registration
source: user request 2026-05-20
created: 2026-05-20
depends_on: []
---

# MH-051: Register the dashboard control-plane product spec and epic docs

## Context

The dashboard work expanded from a shadcn-ui restyle into a new frontend and
control plane for Mars Harness. Before implementation starts, the repo needs a
durable product contract, expanded BDD scenarios, architecture decision update,
backlog plan, active-plan registration, and focused implementation tickets.

## BDD Scenario IDs

- F-010-S010

## Affected Docs/Code Areas

- `docs/product-specs/dashboard-control-plane.md`
- `docs/product-specs/index.md`
- `docs/product-specs/product-surface.md`
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/design-docs/dashboard.md`
- `docs/design-docs/index.md`
- `docs/exec-plans/backlog/tanstack-dashboard-control-plane.md`
- `docs/exec-plans/active/current-operating-plan.md`
- `docs/tickets/backlog/`

## Acceptance Criteria

- [ ] The dashboard control-plane product spec exists with required product-spec metadata.
- [ ] The product-spec index catalogs the new spec.
- [ ] Product surface docs distinguish the legacy/current dashboard from planned TanStack work.
- [ ] F-010 includes scenarios for auth, sidecar runtime, nonblocking APIs, Overview, Active Work, Preview, feedback, Agent Roster, Models, DORA, and migration.
- [ ] Dashboard architecture docs record the external Node prerequisite, pnpm pin, TanStack Start sidecar, Go gateway, local-admin auth, concurrent read model, and optional GitHub-derived metrics or proposal flows.
- [ ] The active operating plan registers the dashboard epic as backlog without claiming implementation has started.
- [ ] MH-051 through MH-061 exist in `docs/tickets/backlog/`.

## Non-Goals

- Runtime implementation.
- Frontend scaffolding.
- Dashboard route, auth, API, or model behavior changes.
- Version promotion or release publication.

## Evidence Requirements

- `go test ./internal/docsconsistency`
- Manual check that the product spec is indexed.
- Manual check that all ticket scenario IDs exist in F-010.
- Manual check that new ticket files live only in `docs/tickets/backlog/`.
- Manual check that docs avoid claims that the TanStack dashboard is already implemented.

## Next Action

Run the docs consistency suite, attach the command evidence, and move this
ticket to done only if no runtime implementation is claimed.
