# TanStack Dashboard Control Plane

**Status:** Backlog
**Priority:** P2
**Depends On:** Completion or explicit suspension of the active P0 factory-pace slice; `dashboard-control-plane.md` accepted as the governing product spec; F-010 scenarios registered before implementation tickets start.
**Blocks:** Next-generation dashboard implementation claims, broad dashboard restyle work, dashboard auth claims, dashboard DORA claims, and dashboard source-change proposal claims.
**Related Tickets:** MH-051, MH-052, MH-053, MH-054, MH-055, MH-056, MH-057, MH-058, MH-059, MH-060, MH-061, T-010
**Goals:** G-001, G-003
**BDD Feature:** F-010
**Hypothesis:** A TanStack Start sidecar behind the Go gateway can provide a polished local control plane without blocking orchestrator work, while keeping local-admin auth, repo artifacts, score evidence, model routing, and guarded source-change proposals authoritative.
**Success Evidence:** F-010-S010 through F-010-S022 pass with docs, prerequisite, auth, nonblocking API, Overview, Active Work, Preview, Feedback, Agent Roster, Models, DORA, migration, accessibility, visual, and browser evidence. No dashboard route claims implementation before matching code and tests exist.
**Falsification Evidence:** The dashboard requires bundled Node, silently installs prerequisites, allows anonymous data access, blocks behind agent execution, mutates source files directly from the frontend, shows DORA without configured workflow evidence, or replaces repo artifacts as source of truth.
**Scenario Schedule:** F-010-S010, F-010-S011, F-010-S012, F-010-S013, F-010-S014, F-010-S015, F-010-S016, F-010-S017, F-010-S018, F-010-S019, F-010-S020, F-010-S021, F-010-S022
**Current Failing Scenario:** Waiting for active-plan promotion; no TanStack dashboard implementation scenario is current while this plan remains backlog.
**Walking Skeleton Slice:** Ship docs registration first, then prerequisite checks, local-admin gateway auth, concurrent read APIs, a minimal authenticated Overview, and one adaptive unavailable state before adding richer views.
**Learning Or MVP Outcome:** Establish the dashboard as a true MARS frontend and control plane without weakening strict local operation, source-of-truth docs, trust policy, or orchestrator responsiveness.
**Created:** 2026-05-20
**Updated:** 2026-05-20
**Owner:** MARS maintainers
**Source:** User dashboard control-plane expansion request on 2026-05-20.

## Purpose

This backlog plan turns the dashboard rewrite conversation into durable work
that Orchestrator can pick up later. It is intentionally documentation and
planning only until promoted. It does not claim runtime implementation has
started.

The plan expands the earlier `T-010` shadcn-ui restyle into a control-plane
epic for MARS itself: TanStack Start frontend, external Node
prerequisite, local-admin auth, concurrent APIs, Overview, Active Work, Preview,
Feedback, Agent Roster, Models, GitHub-derived DORA, and migration from the
legacy embedded dashboard.

## Slice Boundaries

- **Documentation slice (`MH-051`)**: create and index the product spec, expand
  F-010, update dashboard architecture docs, register this plan, and create the
  backlog ticket map. No runtime code.
- **Prerequisite slice (`MH-052`)**: detect Node `24.x` and `pnpm@11.1.1`,
  report remediation, and keep core orchestration usable without the sidecar.
- **Security and gateway slice (`MH-053`)**: local-admin auth, sessions, CSRF,
  gateway protection, and sidecar proxy boundaries.
- **Concurrency slice (`MH-054`)**: read snapshots, event stream, typed
  unavailable states, and async command-status contracts.
- **First UX slice (`MH-055`)**: TanStack shell and authenticated Overview.
- **Work surface slices (`MH-056`, `MH-057`)**: Active Work, adaptive Preview,
  annotations, and next-turn feedback.
- **Configuration slices (`MH-058`, `MH-059`)**: Agent Roster, model inventory,
  override proposals, and serve/start override support.
- **Delivery metrics slice (`MH-060`)**: GitHub-derived DORA with strict
  unavailable-state behavior.
- **Polish and migration slice (`MH-061`)**: accessibility, responsive visual
  QA, legacy route behavior, and removal or fallback decisions.

## Ticket Map

| Ticket | Scenario focus | Purpose |
| --- | --- | --- |
| MH-051 | F-010-S010 | Product spec, F-010 expansion, dashboard design doc, product-surface update, and exec-plan registration. |
| MH-052 | F-010-S011 | Node `24.x` and `pnpm@11.1.1` prerequisite checks and remediation output. |
| MH-053 | F-010-S012, F-010-S013 | Local-admin auth, sessions, CSRF, and protected Go gateway. |
| MH-054 | F-010-S014 | Concurrent read model, event stream, typed unavailable states, and async command status. |
| MH-055 | F-010-S013, F-010-S015 | TanStack shell and authenticated Overview. |
| MH-056 | F-010-S016 | Active Work renderer. |
| MH-057 | F-010-S017, F-010-S018 | Adaptive Preview and next-turn feedback mailbox. |
| MH-058 | F-010-S019 | Agent Roster and draft code-host proposal workflow. |
| MH-059 | F-010-S020 | Available Models, override proposals, and serve/start override support. |
| MH-060 | F-010-S021 | GitHub-derived DORA metrics. |
| MH-061 | F-010-S022 | Visual polish, accessibility, responsive QA, and legacy dashboard migration or removal. |

## Dependencies

- Dashboard product spec and F-010 scenarios must land before runtime work.
- Node `24.x` and `pnpm@11.1.1` checks must land before TanStack sidecar
  startup is treated as supported.
- Local-admin auth must protect every dashboard route before sensitive
  Overview, roster, model, feedback, or DORA data ships.
- Nonblocking read APIs must exist before Overview becomes the primary
  operator surface.
- Agent Roster mutation actions depend on the guarded code-host proposal or
  local patch-preview path.
- DORA depends on `.harness/dashboard.yaml`, GitHub auth, a remote, configured
  deployment workflow names, and workflow run history.

## Validation Matrix

| Area | Required evidence |
| --- | --- |
| Docs registration | `go test ./internal/docsconsistency`; product spec indexed; active plan references backlog without claiming implementation. |
| Prerequisites | Unit tests for missing Node, wrong Node, missing pnpm, wrong pnpm, and actionable remediation. |
| Auth | Unit and integration tests for login, session expiry, CSRF failure, protected routes, protected event stream, and actor attribution. |
| Concurrency | Tests proving dashboard reads and command-status requests complete while orchestrator work is active or simulated as blocked. |
| Overview | Route tests, data fixtures, unavailable-state fixtures, browser screenshots, keyboard flow, and responsive checks. |
| Active Work | Markdown rendering tests, malformed ticket handling, empty work state, source artifact link checks, and visual verification. |
| Preview and feedback | Provider fixtures for web, artifact, mobile, API, cloud, distributed-system, library, and CLI work; anchored and unanchored feedback tests. |
| Agent Roster | Manifest fixtures, metrics fixtures, proposal preview tests, and guardrail checks proving no silent source mutation. |
| Models | Registry, cache, endpoint, Ollama, cloud-provider, health, override, unavailable-state, and validation-plan tests. |
| DORA | GitHub workflow fixtures for success, failure, cancellation, timeout, missing config, missing auth, no remote, insufficient history, permission error, and rate limit. |
| Migration | Legacy route redirect/fallback/removal tests and operator-facing behavior notes. |

## Non-Goals

- Runtime implementation inside `MH-051`.
- Bundling Node, installing Node, or installing pnpm for the user.
- Hosted dashboard service.
- Replacing repo artifacts with dashboard-only state.
- Treating optional GitHub-derived metrics as mandatory for local harness use.
- Promoting a model default solely because it is visible in the dashboard.

## Pickup Instructions

When this plan is promoted, start with the lowest-numbered open `MH-05x` ticket
whose dependencies are satisfied. Implement one scenario group at a time,
update F-010 evidence before completion, and keep the current embedded dashboard
truthful until migration evidence proves the new behavior.
