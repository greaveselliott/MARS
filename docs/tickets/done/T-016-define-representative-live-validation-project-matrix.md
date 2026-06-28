---
id: T-016
title: Define representative live validation project matrix
priority: high
complexity: medium
work_type: docs
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - docs/design-docs/delivery-operating-model.md#AD-138-live-demo-improvement-loop
  - docs/exec-plans/active/current-operating-plan.md
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-12-tool-argument-and-matrix-replay--2026-05-20
verified_by: "go test ./internal/docsconsistency ./internal/docsync"
owner: "Codex"
last_attempt: "2026-05-20: operator feedback and run12 evidence updated AD-138 with a representative validation portfolio."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Use the matrix to select the smallest relevant replay subset for future lifecycle claims."
dedupe_key: "public-example"
source: live Run 12 and operator feedback on avoiding demo-123 overfitting
created: 2026-05-20
depends_on: []
---

# T-016: Define representative live validation project matrix

## Context
The live improvement loop has used demo-123 as the canonical first-run canary, which is useful but risks overfitting foundation behavior to one static Space Invaders project. MARS must generically build many software application shapes, so source lifecycle claims should be checked against a small representative project portfolio rather than only one demo.

## Requirements
- Define a foundation live validation matrix with at least static browser app, package-managed frontend, API/service, CLI/tooling, and existing-repo maintenance archetypes.
- Specify what each archetype is meant to catch, the minimum brief, expected lifecycle evidence, and when it must run.
- Keep demo-123 as one canary, not the whole acceptance suite.
- Update the active operating plan so future continuous-improvement loops choose the smallest relevant matrix subset instead of always optimizing for one game.

## Acceptance Criteria
- A durable operating-model artifact names the live validation matrix and selection rules.
- The active plan references matrix-based replay before broad lifecycle claims.
- The docs distinguish generic tool/runtime fixes from project-shape-specific prompt guidance.
- No v2 architecture or product implementation is chosen by the matrix work itself.

## Completion Evidence

AD-138 now treats `demo-123` as one canonical static-browser canary instead of
the full acceptance suite. It defines a validation portfolio covering static
browser apps, package-managed frontends, API/services, CLI/tooling projects,
and existing-repo maintenance, with selection rules for narrow versus broad
lifecycle claims.
