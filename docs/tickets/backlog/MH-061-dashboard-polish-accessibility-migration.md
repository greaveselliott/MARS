---
id: MH-061
title: Complete dashboard visual polish, accessibility, responsive QA, and legacy migration
priority: medium
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S022"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-055", "MH-056", "MH-057", "MH-058", "MH-059", "MH-060"]
trace_id: none
next_action: "Define final legacy route behavior after the main TanStack dashboard views have implementation evidence."
dedupe_key: dashboard-control-plane:polish-accessibility-migration
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-055", "MH-056", "MH-057", "MH-058", "MH-059", "MH-060"]
---

# MH-061: Complete dashboard visual polish, accessibility, responsive QA, and legacy migration

## Context

The dashboard epic is not complete when routes merely render. It needs a
polished control-plane experience, accessible controls, responsive layouts, and
a truthful migration path for legacy dashboard URLs and assets.

## BDD Scenario IDs

- F-010-S022

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/serve/`
- `internal/ui/`
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/design-docs/dashboard.md`
- `docs/product-specs/dashboard-control-plane.md`
- generated target guidance if dashboard operator behavior changes

## Acceptance Criteria

- [ ] Desktop and mobile dashboard layouts have no incoherent overlap, clipped controls, or unreadable dense panels.
- [ ] Keyboard navigation, focus states, form labels, table semantics, contrast, and reduced-motion behavior are verified.
- [ ] Visual treatment stays operational and tastefully polished, with compact cards only for individual repeated items or framed tools.
- [ ] Legacy dashboard routes are either retained deliberately, redirected, or removed with documented operator behavior.
- [ ] Tests prove legacy/current dashboard docs and TanStack control-plane docs do not claim the wrong implementation state.
- [ ] Browser screenshots or equivalent visual evidence cover Overview, Active Work, Preview, Feedback, Agent Roster, Models, DORA unavailable states, and migration routes.
- [ ] Generated target guidance is updated if operator-facing dashboard behavior changes for target repos.

## Non-Goals

- Adding new dashboard feature areas beyond the epic scope.
- Reworking orchestrator business logic.
- Making GitHub-derived metrics mandatory.
- Removing terminal dashboard controls.

## Evidence Requirements

- Browser visual verification across desktop and mobile.
- Accessibility checks covering keyboard, focus, labels, contrast, table semantics, and reduced motion.
- Route tests for retained, redirected, or removed legacy dashboard URLs.
- Docs consistency evidence after migration wording changes.
- Manual QA notes for unavailable states and dense operational layouts.

## Next Action

After the main dashboard views have shipped, decide the legacy route behavior,
then run visual and accessibility QA before closing the epic.
