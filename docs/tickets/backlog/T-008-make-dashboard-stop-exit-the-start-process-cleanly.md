---
id: T-008
title: Make dashboard stop exit the start process cleanly
priority: medium
complexity: small
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md"]
verified_by: "live demo-123 dogfood run"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "TBD"
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "runtime_shutdown"
  severity: "medium"
source: 2026-05-19 demo-123 live lifecycle run
created: 2026-05-19
depends_on: []
---

# T-008: Make dashboard stop exit the start process cleanly

## Context

During bounded live demo-123 runs, POST /api/stop stopped workers, schedulers, inference servers, and sleep prevention, but the HTTP response returned dashboard shutdown: context deadline exceeded and the start process remained alive until killed manually.

## Requirements

- Make the stop endpoint complete cleanly after graceful shutdown.
- Ensure start exits once workers, scheduler, inference, and dashboard shutdown are complete.
- Preserve useful error reporting when shutdown cannot complete.

## Acceptance Criteria

- [ ] POST /api/stop returns success for a clean idle start process.
- [ ] The start process exits without manual kill after stop.
- [ ] A regression test covers dashboard/webhook stop behavior without leaving a process running.
