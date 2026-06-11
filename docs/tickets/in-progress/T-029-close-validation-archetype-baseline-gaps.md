---
id: T-029
title: Close validation archetype baseline gaps
priority: high
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
next_action: "Run one package-managed-frontend archetype replay and one existing-repo-maintenance archetype replay on v0.50.x, record both as AD-285 baseline evidence (reports plus pace snapshots), then close with evidence."
kind: enabler
source: Foundation improvement plan Phase 2 WS-F (provisional T-041); blocked_by T-028 doctrine now landed
created: 2026-06-11
depends_on: []
---

# T-029: Close validation archetype baseline gaps

## Context

AD-138 defines a five-archetype validation portfolio, and AD-284 (T-028) now gates source-change classes on minimum archetype replays. The 2026-06-11 foundation review found two archetypes have thin or no recent baseline coverage: package-managed frontend and existing-repo maintenance. Until both have dated baselines, WS-D/WS-E improvement claims against those archetypes cannot be judged.

Ownership classification: foundation-owned validation evidence work. Replay outputs are evidence-only artifacts under docs/validation/ per the foundation improvement plan.

## Requirements

- Run one package-managed-frontend archetype replay (fresh Vite/React-style target with package.json, build/dev scripts) against the current binary, to its natural lifecycle end or recorded blocker.
- Run one existing-repo-maintenance archetype replay (seeded repo with pre-existing files, git history, tickets, and a known bug or feature gap) against the current binary.
- Record both runs per the AD-285 evidence contract: reports under docs/validation/reports/ plus Factory Pace and Convergence And Guardrails snapshots in the dated baseline artifact.
- Classify any observed failures (foundation-owned / deployed-owned / mixed) before they become tickets or fixes; measurement only, no convergence/policy fixes in this ticket.

## Acceptance Criteria

### Functional (happy path)
- [ ] Package-managed-frontend baseline replay recorded per AD-285 with pace snapshot.
- [ ] Existing-repo-maintenance baseline replay recorded per AD-285 with pace snapshot.
- [ ] Both archetype rows in the dated factory-pace baseline move from gap to recorded.

### Edge cases and negative paths
- [ ] A replay that cannot complete records the exact blocker, replay command, and ownership classification instead of an improvement claim.

### Non-goals
- Fixing convergence, tool policy, or role guidance issues surfaced by the replays (Phase 3 slices own those).

### Observability, docs, and regressions
- [ ] Reports cross-referenced from this ticket's evidence_links and the baseline artifact.
