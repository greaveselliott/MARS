---
id: MH-052
title: Add dashboard Node and pnpm prerequisite checks
priority: high
complexity: medium
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S011"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-051"]
trace_id: none
next_action: "Design and implement prerequisite checks for external Node 24.x and pnpm@11.1.1 with actionable remediation output."
dedupe_key: dashboard-control-plane:node-pnpm-prerequisites
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-051"]
---

# MH-052: Add dashboard Node and pnpm prerequisite checks

## Context

The TanStack dashboard must not bundle or install Node. MARS needs a
clear prerequisite check that detects whether the operator already has the right
external runtime and package manager before sidecar startup is treated as
supported.

## BDD Scenario IDs

- F-010-S011

## Affected Docs/Code Areas

- `cmd/mars/`
- `internal/dashboard/`
- `internal/doctor/`
- `internal/serve/`
- future `web/dashboard/`
- `docs/product-specs/dashboard-control-plane.md`
- `docs/features/F-010-dashboard-control-plane.md`

## Acceptance Criteria

- [ ] MARS detects missing Node and reports that Node.js `24.x` is required.
- [ ] MARS detects a non-`24.x` Node version and reports installed and required versions.
- [ ] MARS detects missing pnpm and reports that `pnpm@11.1.1` is required.
- [ ] MARS detects a wrong pnpm version and reports installed and required versions.
- [ ] Remediation output is concrete and does not run an installer for the user.
- [ ] Core `serve` and `start` orchestration remain usable when TanStack dashboard prerequisites are missing.
- [ ] The future dashboard package pins `pnpm@11.1.1` through `packageManager`.

## Non-Goals

- Installing Node.
- Installing pnpm.
- Creating the TanStack dashboard shell.
- Changing local model or inference prerequisites.

## Evidence Requirements

- Unit tests for missing Node, wrong Node, missing pnpm, wrong pnpm, and success.
- CLI or doctor evidence showing actionable remediation output.
- Docs update if the remediation command or prerequisite wording changes.

## Next Action

Choose the owning package for dashboard prerequisite checks and add tests before
wiring the check into dashboard startup or doctor output.
