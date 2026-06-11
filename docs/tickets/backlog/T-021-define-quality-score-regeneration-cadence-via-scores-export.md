---
id: T-021
title: Define QUALITY_SCORE regeneration cadence via scores export
priority: medium
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "foundation-maintainer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Document the regeneration cadence for docs/QUALITY_SCORE.md, run a regeneration with scores export, and record any automation follow-up."
source: Foundation improvement plan Phase 1 WS-A (provisional T-021)
created: 2026-06-11
depends_on: []
---

# T-021: Define QUALITY_SCORE regeneration cadence via scores export

## Context

The 2026-06-11 foundation review found docs/QUALITY_SCORE.md was last generated 2026-05-19 with no defined regeneration cadence. The export surface already exists (mars-harness scores export, F-008 behavior); what is missing is a documented trigger and what gates on grade regressions.

Scope is bounded per the foundation improvement plan: if full automation (post-run hook or scheduled survey wiring) is too large for this slice, land the documented cadence plus a fresh regeneration and record the automation follow-up explicitly.

Ownership classification: foundation-owned enabler; extends existing F-008 behavior, no new surface.

## Requirements

- Document when docs/QUALITY_SCORE.md must be regenerated and what gates on grade regressions, as an AD in docs/design-docs/self-reflective-telemetry.md indexed in docs/design-docs/index.md.
- Run mars-harness scores export --repo . to refresh the artifact.
- Record the automation follow-up in docs/exec-plans/tech-debt.md if hook/schedule wiring is deferred.

## Affected Files

- docs/QUALITY_SCORE.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/index.md
- docs/exec-plans/tech-debt.md

## Acceptance Criteria

### Functional (happy path)
- [ ] Regeneration cadence and regression gating are documented in an indexed AD.
- [ ] docs/QUALITY_SCORE.md is freshly regenerated.

### Edge cases and negative paths
- [ ] Insufficient-evidence exports remain honest and do not fabricate grades.

### Non-goals
- New telemetry surfaces or runtime behavior changes in this slice.

### Observability, docs, and regressions
- [ ] go test ./internal/docsconsistency/... ./internal/docsync/... passes.
