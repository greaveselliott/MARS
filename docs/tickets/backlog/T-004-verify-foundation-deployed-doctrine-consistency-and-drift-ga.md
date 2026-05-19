---
id: T-004
title: Verify foundation deployed doctrine consistency and drift gates
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-001-S015", "F-004-S007"]
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: ["T-002", "T-003"]
trace_id: "TBD"
next_action: "Run a doctrine consistency review after the architecture doc and generated target route land."
dedupe_key: "public-example"
metadata:
  category: "drift_review"
  confidence: "high"
  repo_id: "mars-harness"
  role: "reviewer"
  severity: "high"
  target: "doctrine"
source: foundation/deployed architecture planning 2026-05-19
created: 2026-05-19
depends_on: [T-002, T-003]
---

# T-004: Verify foundation deployed doctrine consistency and drift gates

## Context
The failure class this architecture slice is trying to prevent is doctrine existing in one place but not where agents actually operate. After the source design doc and generated target route land, a focused drift review should verify that source, generated target, glossary, tool, skill, and release surfaces agree.

## Requirements
- Review AGENTS.md, docs/design-docs/harness-glossary.md, docs/design-docs/mirrored-harness-and-context-glossary.md, docs/design-docs/tools-glossary.md, docs/design-docs/release-versioning.md, generated target guidance, and scanner tests.
- Confirm glossary terms are used consistently.
- Confirm foundation-only behavior is explicitly source-only.
- Confirm mirrored rules are represented through generated target defaults or deliberately not mirrored with rationale.
- Confirm release publication doctrine distinguishes GitHub Release object publication from source binary asset verification.
- Record any remaining drift as a follow-up ticket rather than folding unbounded cleanup into this review.

## Affected Files
- AGENTS.md
- docs/design-docs/harness-glossary.md
- docs/design-docs/mirrored-harness-and-context-glossary.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/release-versioning.md
- internal/scanner/init.go
- internal/scanner/scanner_test.go

## BDD Evidence
- Scenario IDs: F-001-S015, F-004-S007
- Evidence links: go test ./internal/docsconsistency ./internal/docsync ./internal/scanner
- Verified by: command

## Acceptance Criteria

### Functional
- [ ] The review records whether source and generated target doctrine agree.
- [ ] Any mismatch has a fix or a follow-up ticket.
- [ ] Source-only rules are visibly marked source-only.

### Edge cases and negative paths
- [ ] The review does not silently broaden target doctrine with foundation-only mechanics.
- [ ] The review does not treat docs as current without checking generated target output.

### Non-goals
- Implementing every follow-up drift fix in this ticket.

### Observability, docs, and regressions
- [ ] docsconsistency, docsync, and scanner checks pass or blockers are recorded with exact commands.
