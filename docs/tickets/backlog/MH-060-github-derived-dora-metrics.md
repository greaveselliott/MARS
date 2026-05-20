---
id: MH-060
title: Implement GitHub-derived DORA metrics for the dashboard
priority: medium
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S021"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-053", "MH-054", "MH-055"]
trace_id: none
next_action: "Design the .harness/dashboard.yaml DORA config reader and GitHub workflow fixture set."
dedupe_key: dashboard-control-plane:github-derived-dora
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-053", "MH-054", "MH-055"]
---

# MH-060: Implement GitHub-derived DORA metrics for the dashboard

## Context

The Overview should show DORA metrics, but "GitHub-backed DORA" needs a precise
meaning. This ticket implements GitHub-derived metrics from configured workflow
runs and explicit unavailable states when the evidence is missing.

## BDD Scenario IDs

- F-010-S021

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/github/`
- `internal/config/`
- `.harness/dashboard.yaml` handling
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/product-specs/dashboard-control-plane.md`

## Acceptance Criteria

- [ ] `.harness/dashboard.yaml` supports `dashboard.dora.window_days`, `dashboard.dora.deployment_workflows`, and `dashboard.dora.branch`.
- [ ] Deployment Frequency counts successful terminal configured workflow runs inside the selected window.
- [ ] Lead Time For Changes compares prior successful deployment SHA to current successful deployment SHA and measures earliest commit timestamp through workflow completion.
- [ ] Change Failure Rate divides failed, cancelled, or timed-out configured workflow runs by all terminal configured workflow runs.
- [ ] Mean Time To Restore measures from a failed configured workflow run to the next successful configured workflow run for the same workflow and branch.
- [ ] Missing GitHub auth, no remote, missing config, no matching runs, insufficient history, permission errors, API errors, and rate limits show typed unavailable states.
- [ ] Merged code-host proposals and tags do not count as deployments unless the configured workflow succeeds.

## Non-Goals

- Inferring deployments from ticket movement alone.
- Treating every workflow as a deployment workflow.
- Requiring GitHub metrics for local-only harness operation.
- Showing estimated DORA values as if they were measured.

## Evidence Requirements

- GitHub workflow fixture tests for success, failure, cancellation, timeout, missing auth, no remote, missing config, no matching runs, insufficient history, permission error, API error, and rate limit.
- Metric calculation tests with multiple workflow names and selected windows.
- Browser verification for Overview DORA cards and unavailable states.
- Docs update if config keys or metric definitions change.

## Next Action

Build fixture data for workflow runs and compare ranges, then implement the
DORA calculator before adding Overview cards.
