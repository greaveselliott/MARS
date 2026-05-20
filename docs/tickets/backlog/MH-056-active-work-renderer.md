---
id: MH-056
title: Render Active Work ticket details in the dashboard
priority: medium
complexity: medium
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S016"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-053", "MH-054", "MH-055"]
trace_id: none
next_action: "Build the Active Work data contract and Markdown renderer against representative ticket fixtures."
dedupe_key: dashboard-control-plane:active-work-renderer
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-053", "MH-054", "MH-055"]
---

# MH-056: Render Active Work ticket details in the dashboard

## Context

Operators need a tasteful view of the work in progress without losing the repo
artifact as the source of truth. Active Work should make tickets and tasks easy
to scan while keeping links to the underlying files, traces, evidence, and BDD
scenarios.

## BDD Scenario IDs

- F-010-S016

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/serve/`
- `docs/tickets/`
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/product-specs/dashboard-control-plane.md`

## Acceptance Criteria

- [ ] Active Work displays the current ticket or task title, context, BDD scenario ids, acceptance criteria, blockers, evidence, affected files, owner, and next action.
- [ ] Markdown is rendered safely and readably.
- [ ] The original ticket or task artifact remains linked and authoritative.
- [ ] Missing active work shows a polished empty state.
- [ ] Malformed ticket metadata produces an actionable unavailable state instead of a broken page.
- [ ] The view remains compact and readable on desktop and mobile.

## Non-Goals

- Editing ticket files directly in the dashboard.
- Replacing the ticket lifecycle.
- Implementing preview or feedback behavior.

## Evidence Requirements

- Renderer tests for complete, minimal, blocked, malformed, and absent tickets.
- Link tests for source artifacts and referenced BDD scenarios.
- Browser screenshots for desktop and mobile.
- Accessibility checks for headings, lists, focus order, and link labels.

## Next Action

Create representative ticket fixtures and wire the Active Work route to the
nonblocking dashboard data contract.
