---
id: T-019
title: Retire pipeline-learnings standing tracker with recorded decision
priority: medium
complexity: small
work_type: docs
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "foundation-maintainer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Write the retirement AD, delete docs/exec-plans/pipeline-learnings.md, and update references that name the file."
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
- [ ] Retirement AD exists and is indexed.
- [ ] pipeline-learnings.md is removed and no live doc links to it.

### Edge cases and negative paths
- [ ] Historical reference docs that mention the file as Mars-monorepo evidence are preserved unchanged.

### Non-goals
- Backfilling historical run learnings into a new tracker.

### Observability, docs, and regressions
- [ ] go test ./internal/docsconsistency/... ./internal/docsync/... passes.
