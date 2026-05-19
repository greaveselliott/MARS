---
id: T-007
title: Fix deployed mars_harness_cli binary resolution during release review
priority: high
complexity: medium
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
  category: "deployed_cli_resolution"
  severity: "high"
source: 2026-05-19 demo-123 live lifecycle run
created: 2026-05-19
depends_on: []
---

# T-007: Fix deployed mars_harness_cli binary resolution during release review

## Context

A live demo-123 run reached Release Manager, but the deployed mars_harness_cli tool resolved /path/to/local-redacted at version 0.0.1-dev instead of the running harness binary. That binary lacked release and tools commands, so release notes failed inside the deployed lifecycle after product implementation, QA, security, and dogfood had passed.

## Requirements

- Ensure mars_harness_cli uses the active foundation binary or a configured current binary path during deployed harness runs.
- Keep operator-visible diagnostics when the resolved binary is stale or lacks required commands.
- Add deterministic coverage for Release Manager invoking release notes through mars_harness_cli in a deployed target.

## Acceptance Criteria

- [ ] A deployed target Release Manager run uses a current mars-harness binary with release notes support.
- [ ] A stale PATH binary fails with actionable guidance instead of repeated unknown-command calls.
- [ ] A live or fake deployed lifecycle reaches release-note generation without resolving 0.0.1-dev from PATH.
