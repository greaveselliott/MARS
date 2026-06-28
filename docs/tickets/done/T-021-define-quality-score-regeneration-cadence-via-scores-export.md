---
id: T-021
title: Define QUALITY_SCORE regeneration cadence via scores export
priority: medium
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - "AD-278 in docs/design-docs/self-reflective-telemetry.md, indexed in docs/design-docs/index.md"
  - "mars scores export --repo . regeneration on 2026-06-11 (docs/QUALITY_SCORE.md Updated: 2026-06-11)"
  - "TD-006 in docs/exec-plans/tech-debt.md records the hook/schedule automation follow-up"
  - "go test ./internal/docsconsistency/... ./internal/docsync/..."
verified_by: "foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; cadence documented as AD-278, artifact regenerated, automation follow-up recorded as TD-006."
source: Foundation improvement plan Phase 1 WS-A (provisional T-021)
created: 2026-06-11
depends_on: []
---

# T-021: Define QUALITY_SCORE regeneration cadence via scores export

## Context

The 2026-06-11 foundation review found docs/QUALITY_SCORE.md was last generated 2026-05-19 with no defined regeneration cadence. The export surface already exists (mars scores export, F-008 behavior); what is missing is a documented trigger and what gates on grade regressions.

Scope is bounded per the foundation improvement plan: if full automation (post-run hook or scheduled survey wiring) is too large for this slice, land the documented cadence plus a fresh regeneration and record the automation follow-up explicitly.

Ownership classification: foundation-owned enabler; extends existing F-008 behavior, no new surface.

## Requirements

- Document when docs/QUALITY_SCORE.md must be regenerated and what gates on grade regressions, as an AD in docs/design-docs/self-reflective-telemetry.md indexed in docs/design-docs/index.md.
- Run mars scores export --repo . to refresh the artifact.
- Record the automation follow-up in docs/exec-plans/tech-debt.md if hook/schedule wiring is deferred.

## Affected Files

- docs/QUALITY_SCORE.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/index.md
- docs/exec-plans/tech-debt.md

## Acceptance Criteria

### Functional (happy path)
- [x] Regeneration cadence and regression gating are documented in an indexed AD (AD-278).
- [x] docs/QUALITY_SCORE.md is freshly regenerated (Updated: 2026-06-11).

### Edge cases and negative paths
- [x] Insufficient-evidence exports remain honest and do not fabricate grades (2026-06-11 export reports Insufficient evidence overall and an honest D for ticket flow).

### Non-goals
- New telemetry surfaces or runtime behavior changes in this slice.

### Observability, docs, and regressions
- [x] go test ./internal/docsconsistency/... ./internal/docsync/... passes (run 2026-06-11).
