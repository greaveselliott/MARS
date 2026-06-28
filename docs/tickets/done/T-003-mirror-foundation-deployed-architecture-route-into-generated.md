---
id: T-003
title: Mirror foundation deployed architecture route into generated target doctrine
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-004-S007", "F-012-S007"]
end_to_end_evidence: not_applicable
evidence_links: ["go test ./internal/scanner -run TestInit_success", "go test ./internal/docsconsistency ./internal/docsync"]
verified_by: "command"
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: ["T-002"]
trace_id: "TBD"
next_action: "Done; use T-004 for drift review."
dedupe_key: "public-example"
metadata:
  category: "doctrine_mirroring"
  confidence: "high"
  repo_id: "mars"
  role: "planner"
  severity: "high"
  target: "generated-targets"
source: foundation/deployed architecture planning 2026-05-19
created: 2026-05-19
depends_on: [T-002]
---

# T-003: Mirror foundation deployed architecture route into generated target doctrine

## Context
The foundation/deployed architecture doc should not remain source-only if its reusable core changes how target agents understand mirrored doctrine. Generated targets need the route and core concept without inheriting foundation-only release asset or runtime implementation mechanics.

## Requirements
- Update docs/design-docs/harness-glossary.md with a contextual route for foundation/deployed architecture, mirrored operating doctrine, recursive improvement boundaries, and doctrine drift.
- Mirror the applicable route or core doctrine into internal/scanner/init.go.
- Keep foundation-only mechanics such as source binary asset verification clearly source-only.
- Update scanner tests proving initialized targets receive the route/core doctrine and do not import source-only release asset mechanics.

## Affected Files
- docs/design-docs/harness-glossary.md
- internal/scanner/init.go
- internal/scanner/scanner_test.go
- docs/design-docs/mirrored-harness-and-context-glossary.md

## BDD Evidence
- Scenario IDs: F-004-S007, F-012-S007
- Evidence links: go test ./internal/scanner -run TestInit_success; go test ./internal/docsconsistency
- Verified by: command

## Acceptance Criteria

### Functional
- [x] Generated target guidance includes the reusable foundation/deployed architecture route or core doctrine.
- [x] Source-only release asset mechanics remain marked source-only and are not copied as target requirements.
- [x] The mirrored-harness design doc points to the new architecture doc as the deeper view.

### Edge cases and negative paths
- [x] Existing target upgrades remain non-destructive.
- [x] Generated doctrine does not duplicate the full source architecture doc when a route is enough.

### Non-goals
- Adding a new tool or skill.
- Rewriting target prompts beyond the route/core doctrine needed for this slice.

### Observability, docs, and regressions
- [x] Scanner tests cover the generated target route/core doctrine.
- [x] docsconsistency remains green.
