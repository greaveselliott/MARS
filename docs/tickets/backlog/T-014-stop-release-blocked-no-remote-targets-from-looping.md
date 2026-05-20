---
id: T-014
title: Stop release-blocked no-remote targets from looping
priority: high
complexity: small
work_type: bug
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "2026-05-20: demo-123-run10 release-manager invented a fake origin and Orchestrator routed release_blocked back to Dogfood."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Prevent Release Manager from mutating remotes and prevent release_blocked publication failures from routing to Dogfood."
dedupe_key: "public-example"
source: docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-10-engineer-terminal-replay--2026-05-20
created: 2026-05-20
depends_on: []
---

# T-014: Stop release-blocked no-remote targets from looping

## Context
Run 10 proved the product lifecycle can reach Release Manager, but a throwaway target with no remote caused Release Manager to add a guessed origin and retry pushes. The release-blocked disposition then routed back to Dogfood, creating an unnecessary loop after product validation had already passed.

## Requirements
- Release Manager must never add, rewrite, guess, remove, or rename git remotes.
- No-remote targets should stop after local release notes/tag checks and record release_blocked.
- Orchestrator should not route release-blocked publication failures back to Dogfood unless product validation evidence changed.
- The live demo report and active plan should record the behavior and fix.

## Acceptance Criteria
- Tests block git remote mutation through shell_exec.
- Generated Release Manager guidance tells agents not to invent remotes.
- A clean demo replay or focused fake-LLM test proves release_blocked no-remote state does not route to Dogfood.
