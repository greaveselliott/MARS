---
id: MH-029
title: Create intervention-debt tickets from telemetry triage
priority: high
complexity: medium
source: self-reflective-telemetry.md AD-037 through AD-039
created: 2026-05-02
---

# MH-029: Create intervention-debt tickets from telemetry triage

## Context

Mars Harness now triages recurring telemetry patterns and low scores into typed improvement targets. The next step is to turn actionable proposals into durable work items so the harness proactively improves prompts, skills, process, guardrails, context routes, inference setup, manifests, and tool policy.

## Requirements

- Add a first-class `intervention-debt` ticket kind or metadata field.
- Create or update a ticket when telemetry triage identifies a recurring or high-severity improvement target.
- Dedupe by repo, role, target, category, and evidence window.
- Link each ticket to the originating telemetry event or score snapshot.
- Prefer in-progress intervention-debt tickets above ordinary backlog work.
- Keep direct evolution bounded by trust and allowlists; process/product changes become tickets by default.

## Affected Files

- `internal/telemetry/`
- `internal/serve/`
- `internal/tools/ticket_create.go`
- `docs/tickets/README.md`
- `docs/design-docs/self-reflective-telemetry.md`

## Acceptance Criteria

### Functional (happy path)

- [x] Recurring telemetry pattern creates one intervention-debt ticket with role, repo, target, category, severity, confidence, and evidence.
- [x] Low score snapshot with enough samples creates or updates an intervention-debt ticket.
- [x] Existing matching intervention-debt ticket is updated rather than duplicated.
- [x] Planner/Engineer prioritization sees intervention-debt ahead of ordinary backlog work.

### Edge cases and negative paths

- [x] Healthy or sparse scores do not create tickets.
- [x] Repeated identical telemetry patterns do not create ticket storms.
- [x] Unknown failures create investigation tickets without proposing unsafe direct edits.
- [x] Ticket creation failure is recorded in telemetry and does not crash the server.

### Non-goals

- Direct unbounded prompt or process edits.
- Cross-repo learning without explicit operator consent.

### Observability, docs, and regressions

- [x] `docs/tickets/README.md` documents the intervention-debt ticket type.
- [x] Tests cover dedupe, severity, and low-score ticket creation.
- [x] Dashboard or API exposes the latest triage-created ticket links.

## Completion Notes

- `ticket_create` now supports `kind`, `dedupe_key`, and metadata frontmatter. Matching intervention-debt tickets are updated in place instead of duplicated.
- Telemetry pattern and score triage now create intervention-debt tickets through the canonical ticket path and emit `intervention_debt_ticket` dashboard events with ticket paths.
- The generated target ticket README mirrors intervention-debt rules so initialized repos inherit the same operating doctrine.
