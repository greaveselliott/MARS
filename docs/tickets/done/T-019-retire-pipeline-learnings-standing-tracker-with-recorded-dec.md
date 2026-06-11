---
id: T-019
title: Retire pipeline-learnings standing tracker with recorded decision
priority: medium
complexity: small
work_type: docs
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - "AD-276 in docs/design-docs/self-improvement.md, indexed in docs/design-docs/index.md"
  - "go test ./internal/docsconsistency/... ./internal/docsync/..."
verified_by: "foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; pipeline-learnings.md removed, AD-276 indexed, exec-plans README updated."
source: Foundation improvement plan Phase 1 WS-A (provisional T-019)
created: 2026-06-11
depends_on: []
---

# T-019: Retire pipeline-learnings standing tracker with recorded decision

## Context

The 2026-06-11 foundation review found docs/exec-plans/pipeline-learnings.md is an empty standing tracker created from the Mars meta-harness relevance audit. The learnings loop it anticipated landed elsewhere: delivery-operating-model.md architecture decisions (AD-164 through AD-218 and later) plus docs/validation/ reports now record recurring failure patterns and fixes. Backfilling the tracker would duplicate existing evidence.

Ownership classification: foundation-owned documentation hygiene.

## Requirements

- Record an architecture decision in docs/design-docs/self-improvement.md explaining why the tracker is retired and where the learnings loop actually lives.
- Add the AD to docs/design-docs/index.md.
- Delete docs/exec-plans/pipeline-learnings.md.
- Update docs/exec-plans/README.md standing-tracker table; leave docs/references/mars-meta-harness-relevance-audit.md untouched as historical evidence.

## Affected Files

- docs/exec-plans/pipeline-learnings.md
- docs/exec-plans/README.md
- docs/design-docs/self-improvement.md
- docs/design-docs/index.md

## Acceptance Criteria

### Functional (happy path)
- [x] Retirement AD exists and is indexed (AD-276).
- [x] pipeline-learnings.md is removed and no live doc links to it.

### Edge cases and negative paths
- [x] Historical reference docs that mention the file as Mars-monorepo evidence are preserved unchanged (docs/references/mars-meta-harness-relevance-audit.md untouched).

### Non-goals
- Backfilling historical run learnings into a new tracker.

### Observability, docs, and regressions
- [x] go test ./internal/docsconsistency/... ./internal/docsync/... passes (run 2026-06-11).
