---
id: T-029
title: Close validation archetype baseline gaps
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md", "docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md", "docs/validation/baselines/2026-06-12-factory-pace-baseline.md"]
verified_by: "Live demo-12 (Vite/React habit tracker, fresh) and demo-13 (existing Phaser/Tetris repo with history and a known bug) replays on mars 0.50.2, balanced model set (Qwen3-Coder Q4_K_M reasoning/coding, Gemma-4-E4B Q5_K_M fast), scores exports committed to both targets"
owner: "foundation-maintainer"
last_attempt: >-
  2026-06-12: both archetype replays recorded per AD-285 with model identity.
  demo-12 (package-managed frontend) reached engineer initial scaffold then
  wedged deterministically on coding-tier context overflow (33281/32883
  tokens vs 32768 ctx; retry-stable) — foundation-owned, ticket T-032.
  demo-13 (existing-repo maintenance) reached a real engineer product commit
  (T-004 step 1 Tetris mechanics) then wedged on the same overflow (32923
  tokens), confirming the finding generalizes — recorded on T-032. Planning
  roles were clean on both archetypes. Neither replay completed its
  lifecycle; both reports record exact blocker, replay command, and
  ownership classification instead of improvement claims.
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Future replays of these archetypes compare against the lifecycle-reach bars recorded in the two 2026-06-12 reports; T-032 owns the engineer context-overflow fix."
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
- [x] Package-managed-frontend baseline replay recorded per AD-285 with pace snapshot. (docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md)
- [x] Existing-repo-maintenance baseline replay recorded per AD-285 with pace snapshot. (docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md)
- [x] Both archetype rows in the dated factory-pace baseline move from gap to recorded. (Archetype coverage table added to docs/validation/baselines/2026-06-12-factory-pace-baseline.md)

### Edge cases and negative paths
- [x] A replay that cannot complete records the exact blocker, replay command, and ownership classification instead of an improvement claim. (Both replays wedged on the T-032 coding-tier context overflow; reports record blocker, command, and foundation ownership with no improvement claims.)

### Non-goals
- Fixing convergence, tool policy, or role guidance issues surfaced by the replays (Phase 3 slices own those).

### Observability, docs, and regressions
- [x] Reports cross-referenced from this ticket's evidence_links and the baseline artifact.
