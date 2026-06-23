---
id: T-045
title: Implement JIRA mirror and sync model foundation
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-013-S002"]
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-06-23-example-target-project-jira-mirror-sync.md
verified_by: "Codex foundation-maintainer with QA, Security, and Dogfood subagent review"
owner: "foundation-maintainer"
last_attempt: "2026-06-23"
blocker: "none"
blocked_by: []
trace_id: ""
next_action: "Plan 2 is complete; start Plan 3 from T-046 only."
dedupe_key: "public-example"
source: docs/exec-plans/active/current-operating-plan.md
created: 2026-06-23
depends_on: [T-044]
---

# T-045: Implement JIRA Mirror And Sync Model Foundation

## Context

Plan 1 established the default-off integrations substrate. Plan 2 adds the JIRA
mirror and sync model for board-driven repos while keeping JIRA as the source of
record for board-owned fields. This ticket does not enqueue LLM work per JIRA
event and does not implement prioritisation, Figma, PR delivery, or frontier
model routing.

## Requirements

- Add an internal JIRA ingestion package with webhook and polling entry points behind `.harness/integrations.yaml` gates.
- Require explicit project-to-repo mapping before any JIRA issue can materialize locally.
- Enforce config-owned workspace and required-label scope guards before any local ticket write when `ingestion.jira.scope` is configured.
- Materialize tickets by stable `jira_key`; first sighting creates exactly one backlog Markdown ticket.
- Reconcile later pulls by updating JIRA-owned front matter and requirement/body sections in place.
- Preserve harness-owned lifecycle directory, evidence fields, scoped marker, and agent notes byte-for-byte.
- Drop unmapped or ambiguous JIRA projects with an operator-visible log instead of fan-out.
- Do not register `jira_issue.*` triggers and do not enqueue an LLM job directly from JIRA events.

## Acceptance Criteria

- [x] Missing integrations config keeps all JIRA routes and pollers disabled.
- [x] Webhook and poll ingestion are gated by `flow_profile: board-driven` plus `ingestion.jira.enabled: true`.
- [x] Scoped ingestion drops issues outside configured workspaces or missing required labels such as the Example Target Project `example-required-label` label.
- [x] First JIRA sighting creates a single backlog ticket carrying `jira_key` and JIRA-owned fields.
- [x] Reconciliation updates JIRA-owned fields/body while preserving harness-owned lifecycle/evidence/agent notes byte-for-byte.
- [x] Tests cover unmapped projects, ambiguous project mapping, repeated events, and no LLM job per JIRA event.
- [x] Documentation and MarsDocSync references are updated for the new package and generated defaults.

## Non-Goals

- Board prioritisation and ready-work selection.
- Cost guards beyond no-LLM-per-event behavior.
- Figma context, PR delivery, or frontier model routing.
- JIRA write-back.

## Evidence

- PASS: `git diff --check`
- PASS: `go test -count=1 ./internal/jira ./internal/integrations ./internal/serve -run 'TestMirror|TestWebhook|TestPoll|TestServerJIRA|TestLoad_boardDrivenConfig'`
- PASS: `go test -count=1 ./internal/scanner ./internal/docsync ./internal/docsconsistency`
- PASS: `go test -cover ./internal/jira` with 75.1% coverage.
- PASS: `GOCACHE=<validation-root> go test ./...`
- PASS: `GOCACHE=<validation-root> make check`
- PASS: `GOCACHE=<validation-root> go vet ./...`
- PASS: `make install`
- PASS: installed-binary no-config route smoke returned 404 for `/webhooks/jira` and wrote only `.harness/integrations.example.yaml`.
- PASS: installed-binary board-driven smoke dropped a DEMO issue missing `example-required-label`.
- PASS: installed-binary board-driven smoke created one backlog ticket for a DEMO issue with the configured workspace and label.
- PASS: installed-binary board-driven smoke kept SQLite queue count unchanged across JIRA webhooks.
- Report: `docs/validation/reports/2026-06-23-example-target-project-jira-mirror-sync.md`

## Completion Notes

The JIRA mirror is intentionally pull-only and config-contained. The Example Target Project DEMO
board URL and `example-required-label` label are example/config values,
not Go constants. Plan 3 may now add board selection and paid-model cost guards;
Plan 2 did not start those behaviors.
