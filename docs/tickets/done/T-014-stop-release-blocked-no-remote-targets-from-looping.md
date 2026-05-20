---
id: T-014
title: Stop release-blocked no-remote targets from looping
priority: high
complexity: small
work_type: bug
bdd_scenarios:
  - F-006 release_blocked dispatch stop
  - F-009 release_blocked publication stop
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-11-release-blocked-terminal-replay--2026-05-20
verified_by: "go test ./internal/orchestration ./internal/serve; go test ./internal/docsconsistency ./internal/docsync; demo-123-run11 live replay reached Release Manager, recorded release_blocked, added no remote, and stopped dispatch"
owner: "Codex"
last_attempt: "2026-05-20: demo-123-run11 confirmed release_blocked stops dispatch with no Orchestrator or Dogfood loop."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Continue the live loop with tool argument normalization and static DocSync cleanup found in run 11."
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

## Completion Evidence

The source fix blocks release-publication blockers as a terminal dispatch state.
`demo-123-run11` reached CEO, COO, CTO, Engineer, QA, Security, Dogfood, and
Release Manager. Release Manager recorded `status=blocked` and
`next_need=release_blocked`, the target still had no git remote, pending/running
queue count was zero, and no Orchestrator or Dogfood job was enqueued after the
publication blocker.
