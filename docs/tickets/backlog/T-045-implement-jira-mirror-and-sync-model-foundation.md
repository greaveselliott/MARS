---
id: T-045
title: Implement JIRA mirror and sync model foundation
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-013-S002"]
end_to_end_evidence: required
evidence_links: ["go test ./internal/jira ./internal/tickets ./internal/docsync", "docs/validation/reports/"]
verified_by: "TBD"
owner: "cto-weekly"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Scope and implement Plan 2 only: JIRA webhook/poll ingestion, jira_key materialisation, and safe reconciliation preserving harness-owned fields. Do not add board prioritisation, Figma, PR delivery, or frontier model routing."
dedupe_key: "public-example"
source: docs/exec-plans/active/current-operating-plan.md
created: 2026-06-23
depends_on: [T-044]
---

# T-045: Implement JIRA mirror and sync model foundation

## Context

Plan 1 established the default-off integrations substrate. Plan 2 adds the JIRA mirror and sync model for board-driven repos while keeping JIRA as the source of record for board-owned fields. This ticket must not enqueue LLM work per JIRA event and must not implement prioritisation, Figma, PR delivery, or frontier model routing.

## Requirements

- Add an internal JIRA ingestion package with webhook and polling entry points behind `.harness/integrations.yaml` gates.
- Require explicit project-to-repo mapping before any JIRA issue can materialize locally.
- Materialize tickets by stable `jira_key`; first sighting creates exactly one backlog Markdown ticket.
- Reconcile later pulls by updating JIRA-owned front matter and requirement/body sections in place.
- Preserve harness-owned lifecycle directory, evidence fields, scoped marker, and agent notes byte-for-byte.
- Drop unmapped or ambiguous JIRA projects with an operator-visible log instead of fan-out.
- Do not register `jira_issue.*` triggers and do not enqueue an LLM job directly from JIRA events.

## Acceptance Criteria

- Missing integrations config keeps all JIRA routes and pollers disabled.
- Webhook and poll ingestion are gated by `flow_profile: board-driven` plus `ingestion.jira.enabled: true`.
- First JIRA sighting creates a single backlog ticket carrying `jira_key` and JIRA-owned fields.
- Reconciliation updates JIRA-owned fields/body while preserving harness-owned lifecycle/evidence/agent notes byte-for-byte.
- Tests cover unmapped projects, ambiguous project mapping, repeated events, and no LLM job per JIRA event.
- Documentation and MarsDocSync references are updated for the new package and generated defaults.

## Non-Goals

- Board prioritisation and ready-work selection.
- Cost guards beyond no-LLM-per-event behavior.
- Figma context, PR delivery, or frontier model routing.
- JIRA write-back.
