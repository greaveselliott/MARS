---
id: MH-057
title: Add adaptive preview and next-turn feedback mailbox
priority: medium
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S017", "F-010-S018"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-053", "MH-054", "MH-056"]
trace_id: none
next_action: "Design preview provider types and feedback mailbox storage before implementing the first web preview path."
dedupe_key: dashboard-control-plane:adaptive-preview-feedback
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-053", "MH-054", "MH-056"]
---

# MH-057: Add adaptive preview and next-turn feedback mailbox

## Context

Frontend work can often show a live preview, but mobile apps, APIs, cloud work,
distributed systems, libraries, and CLI tools need different evidence. The
dashboard should adapt to the project type and still let an operator provide
feedback in real time for the next safe agent turn.

## BDD Scenario IDs

- F-010-S017
- F-010-S018

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/serve/`
- `internal/trace/`
- `internal/queue/`
- `internal/context/`
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/product-specs/dashboard-control-plane.md`

## Acceptance Criteria

- [ ] Preview providers cover web URL, screenshot/video artifact, mobile metadata, API evidence, cloud or distributed-system evidence, library evidence, and CLI transcript evidence.
- [ ] Live preview unavailable states identify the attempted provider and available fallback evidence.
- [ ] Feedback can attach to route, selector, file, line, log span, trace id, screenshot coordinate, API route, environment link, active work item, or current run.
- [ ] Feedback appears immediately in the dashboard after submission.
- [ ] Feedback is injected at the next safe agent boundary rather than silently mutating durable repo files.
- [ ] Feedback status is available through the authenticated event stream or status API.

## Non-Goals

- Streaming remote desktops.
- Building mobile simulators.
- Deploying cloud environments.
- Direct prompt rewrites from feedback.

## Evidence Requirements

- Provider fixture tests for web, artifact, mobile, API, cloud, distributed-system, library, and CLI cases.
- Feedback storage tests for anchored and unanchored feedback.
- Agent-context injection tests for the next safe boundary.
- Browser verification for preview fallback states and feedback submission.

## Next Action

Define provider and feedback data models, then implement one web preview plus
one artifact fallback before broadening provider coverage.
