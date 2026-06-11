---
id: T-028
title: Define matrix-gating doctrine for source-change classes and validation evidence
priority: high
complexity: small
work_type: docs
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["AD-284 and AD-285 in docs/design-docs/validation-matrix-gating.md, indexed in docs/design-docs/index.md", "docs/validation/README.md points to the gating doctrine", "go test ./internal/docsconsistency ./internal/docsync", "Mirroring classification: source-only foundation validation doctrine; no generated target guidance change required (AGENTS.md rule 13)"]
verified_by: "command"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; AD-284/AD-285 recorded and indexed. T-041 archetype-gap baseline replays consume this table next phase."
source: Foundation improvement plan Phase 2 WS-F (provisional T-040)
created: 2026-06-11
depends_on: []
---

# T-028: Define matrix-gating doctrine for source-change classes and validation evidence

## Context

AD-138 defines the live validation portfolio (static browser, package-managed frontend, API/service, CLI/tooling, existing-repo maintenance) with informal selection rules. The 2026-06-11 foundation review found agents can under-select replays for the change classes most likely to regress across project shapes, and replay evidence is recorded inconsistently. This ticket (provisional T-040 in the foundation improvement plan; final ID T-028 from ticket_create) makes the class-to-replay mapping mechanical doctrine and fixes the evidence-recording contract.

Ownership classification: foundation-owned validation doctrine, docs-only (no runtime behavior change, so per the new table itself no archetype replay is required for this slice).

Mirroring classification (AGENTS.md rule 13): source-only. Generated targets inherit the generic product evidence loop from AD-138; the gating table governs foundation source validation and is not mirrored.

## Requirements

- Design-doc table mapping source-change classes (tool policy, role guidance, orchestration, release flow, scanner/generated doctrine, model behavior, export rendering, docs-only) to minimum AD-138 archetype replays (AD-284).
- Evidence-recording contract: docs/validation/reports/YYYY-MM-DD-<target>-<purpose>.md naming, required run fields, explicit pass criteria, ticket/report cross-references (AD-285).
- Index both ADs in docs/design-docs/index.md; point docs/validation/README.md at the doctrine.
- Record the mirrored-guidance classification.

## Affected Files

- docs/design-docs/validation-matrix-gating.md (new)
- docs/design-docs/index.md
- docs/validation/README.md

## Acceptance Criteria

### Functional (happy path)
- [x] Every source-change class in the table names its minimum archetype replays and the union rule for multi-class changes.
- [x] The evidence contract defines report naming, required fields, and what counts as a pass.

### Edge cases and negative paths
- [x] Blocked replays leave claims unconfirmed with a recorded blocker and exact replay command.
- [x] Docs-only and render-only changes are explicitly exempt so the table does not impose replay tax on non-runtime work.

### Non-goals
- Running the T-041 archetype-gap baseline replays (next phase; requires live factory runs).
- Mechanical enforcement tooling for the gating table (doctrine first; enforcement is future work if drift is observed).

### Observability, docs, and regressions
- [x] AD-284/AD-285 indexed; validation README cross-references the doctrine.
- [x] Docs gates pass (docsync, docsconsistency).
