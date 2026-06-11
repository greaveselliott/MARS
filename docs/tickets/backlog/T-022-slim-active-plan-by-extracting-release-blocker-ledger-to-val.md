---
id: T-022
title: Slim active plan by extracting release-blocker ledger to validation evidence
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
next_action: "Move the v0.21.0 through v0.42.20 per-version release-blocker ledger from the active plan to docs/validation/release-blockers.md and leave a one-line pointer."
source: Foundation improvement plan Phase 1 WS-A (provisional T-022)
created: 2026-06-11
depends_on: []
---

# T-022: Slim active plan by extracting release-blocker ledger to validation evidence

## Context

The 2026-06-11 foundation review found docs/exec-plans/active/current-operating-plan.md carries a roughly 300-line per-version release-blocker ledger (v0.21.0 through v0.42.20) inside Current Truth. That violates Context Efficiency (tenet 9): every agent reading the active plan pays for historical release evidence that belongs in durable evidence files.

Ownership classification: foundation-owned documentation hygiene; the ledger content itself stays evidence-only.

## Requirements

- Create docs/validation/release-blockers.md holding the per-version ledger verbatim with provenance metadata.
- Replace the ledger in the active plan with a one-line pointer.
- Keep active-plan hygiene checks green (no TBD, no undated relative language).
- Record a docsconsistency length/staleness guard as tech-debt follow-up if not landed here.

## Affected Files

- docs/exec-plans/active/current-operating-plan.md
- docs/validation/release-blockers.md
- docs/validation/README.md
- docs/exec-plans/tech-debt.md

## Acceptance Criteria

### Functional (happy path)
- [ ] Ledger lives under docs/validation/ with a one-line pointer in the active plan.
- [ ] Active plan Current Truth is readable in one screen-class read.

### Edge cases and negative paths
- [ ] No ledger entry is lost or reworded during the move.

### Non-goals
- Re-verifying or backfilling the historical release assets themselves.

### Observability, docs, and regressions
- [ ] go test ./internal/docsconsistency/... ./internal/docsync/... passes.
