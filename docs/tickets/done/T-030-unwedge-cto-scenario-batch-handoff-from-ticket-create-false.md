---
id: T-030
title: Unwedge CTO scenario-batch handoff from ticket_create false-duplicate title matching
priority: high
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - docs/validation/reports/2026-06-11-demo-11-pace-baseline.md#f1-foundation-owned-tool-policy-ticket_create-fuzzy-title-dedupe-falsely-rejects-distinct-endpoint-tickets-and-wedges-cto-against-the-scenario-coverage-handoff-gate
  - docs/validation/reports/2026-06-12-demo-11-t030-ticket-dedupe-canary.md
  - docs/validation/baselines/2026-06-12-factory-pace-baseline.md
verified_by: "go test ./internal/tools -run 'TestIsSubsetMatch|TestTicketCreate_allowsDistinct|TestTicketCreate_allowsSiblingEnabler'; demo-11 T-030 canary PASS 2026-06-12"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: isSubsetMatch requires true keyword subset (removed 80% fuzzy tolerance). demo-11 v3 canary PASS — cto-weekly 2m 1s, 3 ticket_create successes, 0 DUPLICATE, lifecycle to release-manager."
blocker: ""
blocked_by: []
trace_id: "demo-11-t030-ticket-dedupe-canary-v3"
next_action: ""
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "convergence"
  confidence: "high"
  origin: "demo-11 pace-baseline replay 2026-06-11"
  repo: "mars"
  role: "cto-weekly"
  severity: "high"
  target: "foundation harness"
source: demo-11 pace-baseline replay 2026-06-11 (T-011 measurement floor)
created: 2026-06-11
depends_on: []
---

# T-030: Unwedge CTO scenario-batch handoff from ticket_create false-duplicate title matching

Closed 2026-06-12. See validation report
`docs/validation/reports/2026-06-12-demo-11-t030-ticket-dedupe-canary.md`.

## Acceptance criteria

- [x] The demo-11 wedge title pair creates two distinct tickets (unit tests).
- [x] CTO can satisfy the 3-scenario handoff gate on an API/service target
  (canary: cto-weekly completed, Engineer dispatched).
- [x] True duplicates still rejected (unit tests).
- [x] Rerun Inventory/API canary per AD-284 and record pace delta (cto-weekly
  −41% wall vs baseline; no wedge recurrence).
