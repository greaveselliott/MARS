---
id: MH-058
title: Build Agent Roster with guarded source-change proposals
priority: medium
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S019"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-053", "MH-054", "MH-055"]
trace_id: none
next_action: "Inventory manifest, role prompt, metrics, trust, score, tool, guardrail, and model data needed by the roster."
dedupe_key: dashboard-control-plane:agent-roster-proposals
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-053", "MH-054", "MH-055"]
---

# MH-058: Build Agent Roster with guarded source-change proposals

## Context

Operators need to see which agents exist, what their prompts and metrics are,
which models they use, and how to safely propose changes. The dashboard must
not silently mutate role files; changes need a draft code-host proposal or local
patch preview with validation and rollback notes.

## BDD Scenario IDs

- F-010-S019

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/roles/`
- `internal/scoring/`
- `internal/trust/`
- `internal/models/`
- `.harness/manifest.yaml` handling
- `.harness/roles/` handling
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/product-specs/dashboard-control-plane.md`

## Acceptance Criteria

- [ ] Agent Roster lists role key, display name, domain, mode, prompt source, tools, guardrails, triggers, trust, scores, recent outcomes, token usage, model tier, model override, and availability.
- [ ] Roster data is protected by local-admin auth.
- [ ] Missing prompt, score, trust, metric, model, or manifest data shows a typed unavailable state.
- [ ] Prompt, roster, tool, guardrail, trigger, and model-routing changes produce a draft code-host proposal or local patch preview.
- [ ] Proposal output includes changed files, rationale, validation plan, and rollback notes.
- [ ] The dashboard does not directly write role source files.

## Non-Goals

- Replacing manifest ownership rules.
- Overwriting user-owned target role prompts during upgrade.
- Auto-approving source changes.
- Changing trust levels without audit evidence.

## Evidence Requirements

- Manifest and role prompt fixture tests.
- Score, trust, token, and model usage fixture tests.
- Proposal-generation tests for prompt, tool, guardrail, trigger, and model-routing changes.
- Guardrail tests proving no silent dashboard source mutation.
- Browser screenshots for roster table, detail view, unavailable states, and proposal preview.

## Next Action

Create a roster data contract from current manifest and role surfaces, then
design the proposal generator before building edit controls.
