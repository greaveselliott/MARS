---
id: MH-055
title: Create the TanStack shell and authenticated Overview
priority: high
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S013", "F-010-S015"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-052", "MH-053", "MH-054"]
trace_id: none
next_action: "Scaffold the TanStack Start shell after prerequisites, auth, and nonblocking Overview APIs are ready."
dedupe_key: dashboard-control-plane:tanstack-shell-overview
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-052", "MH-053", "MH-054"]
---

# MH-055: Create the TanStack shell and authenticated Overview

## Context

The Overview is the first real dashboard surface. It should show all active
agents, who is working, duration, issues, telemetry, active work items, token
usage, DORA state, and models in use. It must feel polished and operational,
not decorative.

## BDD Scenario IDs

- F-010-S013
- F-010-S015

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/serve/`
- `internal/telemetry/`
- `internal/scoring/`
- `internal/models/`
- `docs/product-specs/dashboard-control-plane.md`
- `docs/features/F-010-dashboard-control-plane.md`

## Acceptance Criteria

- [ ] TanStack Start shell is scaffolded with TanStack Router, Query, Table, and Form conventions.
- [ ] shadcn/ui and lucide are used for the component and icon system.
- [ ] Overview is protected by local-admin auth.
- [ ] Overview shows active agents, role/domain/mode, current work, owner, repo, duration, queue age, status, issues, telemetry warnings, token usage, quality score link, DORA state, and model usage.
- [ ] Every Overview section has a typed unavailable state.
- [ ] Layout works across desktop and mobile without text overlap or dashboard-card clutter.

## Non-Goals

- Implementing every dashboard page.
- Starting preview, feedback, roster mutation, or model override flows.
- Replacing terminal dashboard controls.

## Evidence Requirements

- Route and data tests for Overview.
- Browser screenshots for desktop and mobile.
- Accessibility checks for keyboard navigation, focus, labels, and contrast.
- Unavailable-state fixtures for missing telemetry, missing DORA config, missing model data, and no active work.

## Next Action

Implement the smallest authenticated shell and Overview route that uses the
nonblocking API from MH-054, then verify it in a browser against fixture data.
