---
id: T-007
title: Fix deployed mars_harness_cli binary resolution during release review
priority: high
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md"]
verified_by: "go test ./internal/tools -run TestMarsHarnessCLI; PATH=<validation-root> <validation-root> tools run mars_harness_cli --repo . --trust contributor --args-json '{\"mode\":\"run\",\"args\":[\"release\",\"notes\",\"--repo\",\".\",\"--bump\",\"auto\",\"--dry-run\"],\"timeout_seconds\":10}'"
owner: "codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done"
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

## Progress

- 2026-05-19: Claimed from the live `demo-123` replay. Updated
  `mars_harness_cli` resolution to prefer `MARS_HARNESS_CLI_BIN`, then the
  active running harness executable, then `PATH`, with stale unknown-command
  guidance when a PATH binary is still used.
- 2026-05-19: Verified focused behavior with a freshly built harness binary and
  a deliberately stale `mars-harness` first on `PATH`. The tool reached the
  current `release notes` command through the active executable; it did not
  resolve the stale `0.0.1-dev`-style PATH binary.

## Acceptance Criteria

- [x] A deployed target Release Manager run uses a current mars-harness binary with release notes support.
- [x] A stale PATH binary fails with actionable guidance instead of repeated unknown-command calls.
- [x] A live or fake deployed lifecycle reaches release-note generation without resolving 0.0.1-dev from PATH.
