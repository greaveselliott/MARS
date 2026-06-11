---
id: T-030
title: Unwedge CTO scenario-batch handoff from ticket_create false-duplicate title matching
priority: high
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-06-11-demo-11-pace-baseline.md#f1-foundation-owned-tool-policy-ticket_create-fuzzy-title-dedupe-falsely-rejects-distinct-endpoint-tickets-and-wedges-cto-against-the-scenario-coverage-handoff-gate", "docs/validation/baselines/2026-06-12-factory-pace-baseline.md"]
verified_by: "TBD"
owner: "foundation-maintainer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Phase 3 WS-D slice: make isSubsetMatch title dedupe scenario-aware or tighten its threshold so distinct same-suffix endpoint tickets are not rejected, keep the 3-scenario handoff gate satisfiable, add policy tests for the wedge pair, then rerun the Inventory/API canary per AD-284 (tool-policy class: static browser plus API/service)."
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "convergence"
  confidence: "high"
  origin: "demo-11 pace-baseline replay 2026-06-11"
  repo: "mars-harness"
  role: "cto-weekly"
  severity: "high"
  target: "foundation harness"
source: demo-11 pace-baseline replay 2026-06-11 (T-011 measurement floor)
created: 2026-06-11
depends_on: []
---

# T-030: Unwedge CTO scenario-batch handoff from ticket_create false-duplicate title matching

## Context

The 2026-06-11 demo-11 Inventory/API pace-baseline replay (v0.50.1) stopped at cto-weekly max_turns inside a deterministic policy wedge:

1. CTO created T-001 "Implement health endpoint for inventory API" (F-001-S001).
2. The disposition gate required a backlog batch covering 3 early scenarios before Engineer handoff.
3. Every ticket_create for the second ticket ("Implement list items endpoint for inventory API") was falsely rejected as DUPLICATE: isSubsetMatch matches >=80% shared normalized words for 5+ word titles, and natural API ticket titles share a common suffix (implement, endpoint, inventory, api = 4/5 words).
4. CTO looped job_disposition_record (~13 blocked retries) until max_turns, leaving the ticket file untracked (the turn-47 commit claimed ticket creation but committed only .harness/learnings.yaml).

The wedge is archetype-shaped: API/service targets naturally produce same-suffix ticket titles, static browser targets mostly do not. This regressed lifecycle reach for the Inventory/API archetype from Engineer rework (run65 baseline) to CTO.

Ownership classification: foundation-owned (tool policy, internal/tools/ticket_create.go) with a secondary evidence-integrity facet (commit claiming ticket creation without the ticket file).

## Requirements

- Distinct-scenario feature tickets must be creatable when titles legitimately share archetype suffix words; false DUPLICATE results must not be able to make a required handoff gate unsatisfiable.
- The blocked-disposition guidance and dedupe rejection must not point the role into an infinite retry loop; one of them must yield a satisfiable next action.
- Commit claims about ticket creation should include the ticket file or be blocked (evidence integrity facet, may split into its own slice).
- Policy tests cover the exact wedge pair of titles plus the handoff-gate interaction; AD recorded per the convergence state-machine work (WS-D).

## Acceptance Criteria

### Functional (happy path)
- [ ] The demo-11 wedge title pair creates two distinct tickets.
- [ ] CTO can satisfy the 3-scenario handoff gate on an API/service target.

### Edge cases and negative paths
- [ ] True duplicates (same scenario set or same normalized full title) are still rejected.
- [ ] A blocked disposition retried with identical arguments still converges to guidance, not max_turns.

### Non-goals
- Raising cto-weekly max-turn limits to mask the wedge (T-011 falsification clause).

### Observability, docs, and regressions
- [ ] Rerun Inventory/API canary per AD-284 tool-policy row and record the pace delta against the 2026-06-11 baseline.
