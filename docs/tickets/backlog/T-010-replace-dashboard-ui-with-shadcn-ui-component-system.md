---
id: T-010
title: Replace dashboard UI with shadcn-ui component system
priority: medium
complexity: large
work_type: feature
bdd_scenarios: ["F-010-S001", "F-010-S003", "F-010-S006", "F-010-S007", "F-010-S008"]
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Update dashboard architecture docs and feature contract to authorize the shadcn-ui/ui frontend path before implementation begins."
dedupe_key: "public-example"
source: user request 2026-05-19
created: 2026-05-19
depends_on: []
---

# T-010: Replace dashboard UI with shadcn-ui component system

## Context

The current dashboard is an embedded Go-rendered operations UI using htmx, Chart.js, SSE, and static assets bundled into the single binary. The user requested a ticket to replace that dashboard using shadcn-ui/ui: https://github.com/shadcn-ui/ui.

This is a product and architecture change because existing dashboard doctrine explicitly rules out React, npm, build steps, external frontend dependencies, and externally hosted assets. The implementation must either preserve the single-binary/offline runtime constraints with a generated embedded frontend artifact, or record and approve a deliberate change to those constraints before code moves.

## Requirements

- Replace the existing dashboard visual and interaction layer with a shadcn-ui/ui based component system.
- Preserve the operator workflows currently exposed by the dashboard: status, repos, repo roles, orchestration state, quality score, pause, resume, restart, stop, scan, run-role, emergency stop, and SSE updates.
- Keep dashboard controls behaviorally aligned with CLI interactive controls and shared server methods.
- Preserve or deliberately revise the single-binary, offline, no-runtime-external-dependency distribution model before introducing any frontend build chain.
- Ensure the generated dashboard assets are embedded, versioned, reproducible, and covered by tests if a React/Tailwind build is introduced.
- Update dashboard docs, feature contract out-of-scope language, generated target guidance if affected, and any CLI/tool references that describe dashboard behavior.

## Affected Files

- `internal/dashboard/`
- `internal/serve/`
- `internal/ui/`
- `cmd/mars-harness/`
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/design-docs/dashboard.md`
- `docs/product-specs/product-surface.md`
- generated target harness guidance if dashboard setup or runtime requirements change

## Design Guidance

- Treat shadcn-ui/ui as source components, not as a runtime CDN dependency.
- Prefer a reproducible build that emits static assets embedded by Go, with no network calls at runtime.
- Keep cards compact and operational; the dashboard is a control plane, not a marketing surface.
- Keep server APIs as the behavioral source of truth. The frontend must call existing shared endpoints rather than duplicating control semantics.
- Add a migration note for operators if dashboard development now requires Node tooling while production remains a single binary.

## BDD Evidence

- Scenario IDs: F-010-S001, F-010-S003, F-010-S006, F-010-S007, F-010-S008
- Evidence links: expected `go test ./internal/dashboard`, `go test ./internal/serve`, `go test ./internal/ui`, and browser render verification of the rebuilt dashboard once implemented
- Verified by: TBD

## Acceptance Criteria

### Functional (happy path)
- [ ] Dashboard routes render the shadcn-ui/ui based experience from embedded assets.
- [ ] Existing dashboard APIs and controls continue to work with the same server-side behavior.
- [ ] Orchestration, quality, role health, status, and debug surfaces remain available or have documented replacement navigation.
- [ ] SSE or equivalent live update behavior remains visible in the new UI.

### Edge cases and negative paths
- [ ] Missing optional modules or data render polished empty states instead of broken markup.
- [ ] Control action failures surface actionable errors to the operator.
- [ ] Production runtime does not depend on npm, Vite, a dev server, CDN assets, or network access unless the architecture decision explicitly changes that constraint.
- [ ] Asset build failures produce actionable remediation commands.

### Non-goals
- Full authentication redesign.
- Changing orchestrator, queue, scoring, or trust business logic beyond dashboard presentation.
- Introducing hosted SaaS telemetry or cloud dashboard services.

### Observability, docs, and regressions
- [ ] Dashboard design docs and F-010 feature contract are updated before or alongside implementation.
- [ ] Tests cover routes, static assets, API control behavior, and empty states.
- [ ] Browser screenshots or equivalent visual verification demonstrate desktop and mobile dashboard layouts.
- [ ] Any new code files include `MarsDocSync` metadata that points to the reviewed dashboard docs.

## Notes

Current docs make React/npm/outside assets out of scope. This ticket should not be picked up as a pure UI restyle until that architecture trade-off is made explicit.
