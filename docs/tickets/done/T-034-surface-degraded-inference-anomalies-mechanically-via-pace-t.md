---
id: T-034
title: Surface degraded-inference anomalies mechanically via pace telemetry and a doctor RAM-footprint check
priority: medium
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-06-11-demo-11-pace-baseline.md
  - docs/design-docs/local-inference.md
verified_by: "doctor profile-ram-footprint check; go test ./internal/hardware -run EstimatedProfile"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: doctor profile-ram-footprint warns when estimated profile RAMMinMiB sum >= 85% physical RAM."
blocker: ""
blocked_by: []
trace_id: ""
next_action: ""
kind: intervention-debt
source: intervention-to-automation: 2026-06-11 heavy-model RAM-pressure degradation
created: 2026-06-12
depends_on: []
---

# T-034: Surface degraded-inference anomalies

Closed 2026-06-12 (doctor slice). Telemetry anomaly rows vs baseline deferred to
T-011 pace export follow-on.

## Acceptance criteria

- [x] doctor warns when configured model footprint approaches physical RAM.
- [ ] Export flags per-role wall-time anomalies vs baseline (deferred — T-011).
