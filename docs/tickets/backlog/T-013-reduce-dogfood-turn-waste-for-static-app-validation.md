---
id: T-013
title: Reduce Dogfood turn waste for static app validation
priority: high
complexity: medium
work_type: bug
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Update generated Dogfood static-app validation guidance, add tests, and rerun demo-123."
dedupe_key: "public-example"
source: docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md
created: 2026-05-20
depends_on: []
---

# T-013: Reduce Dogfood turn waste for static app validation

## Context
The demo-123-run6 live replay reached Dogfood with product progress, but Dogfood spent many turns discovering how to serve a static src/ app, retried root servers, hit guardrails for broad find commands, and created T-002 on its final turn before max_turns. This kept the product finding from reaching a normal committed terminal disposition.

## Requirements
- Make generated Dogfood/static-app validation guidance prefer bounded static serving from the detected app root.
- Avoid repeated root-server retries, broad process scans, and broad find commands when validating simple static HTML/CSS/JS targets.
- Ensure target-owned Dogfood findings are created and committed early enough to record a terminal disposition before max_turns.
- Preserve foundation telemetry routing for guardrail/runtime failures.

## Acceptance Criteria
- Tests cover the generated Dogfood guidance for static serving and bounded evidence collection.
- A clean demo-123-style replay reaches Dogfood terminal disposition without dirty watchdog routing.
- Live evidence is recorded in the demo lifecycle report and active plan.
