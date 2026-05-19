---
id: T-012
title: Commit target tickets before handoff after dogfood findings
priority: high
complexity: medium
work_type: bug
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md"]
verified_by: "go test ./...; demo-123-run6 patched dirty-target survey replay"
owner: "codex"
last_attempt: "2026-05-20 demo-123-run6"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done; continue next live loop on Dogfood turn/tool waste and static serving evidence."
dedupe_key: "public-example"
source: docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md
created: 2026-05-19
depends_on: []
---

# T-012: Commit target tickets before handoff after dogfood findings

## Context
A clean demo-123 run on 2026-05-19 reached product planning, implementation, QA, security, and Dogfood. Dogfood created T-002 for missing core Space Invaders features, then recorded changes_requested without committing the created ticket. The next Engineer job received T-002 in context, but git mv could not claim it because the ticket file was untracked, causing repeated guardrail blocks and ticket-gate failure.

The 2026-05-20 rerun confirmed terminal clean-tree handoffs work for normal role completions: CEO was blocked from handoff until it committed the product vision, COO committed the product plan and feature contract, and CTO committed T-001 before Engineer. It also found a related failure path: Dogfood created T-002 on its final turn and hit max_turns before it could commit or record disposition. Direct runtime-failure dispatch was quarantined, but the later orchestrator survey routed Engineer for dogfood_failure while T-002 was still uncommitted.

## Requirements
- Terminal role disposition must not hand off while target `ticket_create` output or dogfood evidence changes are still uncommitted.
- Dogfood-created target-owned findings must be committed before Orchestrator or Engineer consumes them.
- Engineer claim flow must not get stuck on untracked tickets created by a previous role.
- Runtime failures and guardrail blocks must remain foundation telemetry, not target intervention-debt churn.
- Orchestrator survey/watchdog routing must not dispatch autonomous follow-up for a dirty target workspace containing uncommitted target-owned artifacts.
- Runtime-only `.harness/learnings.yaml` remains non-blocking for survey routing.

## Acceptance Criteria
- A test proves job_disposition_record rejects changes_requested or approved dispositions when ticket_create left an uncommitted backlog ticket.
- A test proves the role can recover by committing the ticket and then recording the disposition.
- A test proves dogfood_failure survey routing pauses while a target-owned backlog ticket is uncommitted.
- Generated role guidance tells Dogfood to commit target-owned findings before handoff.
- The live demo report and active plan record the demo-123 run evidence and blocker.

## Verification
- `go test ./internal/serve -run 'TestOrchestratorSurveyPausesDirtyTargetAfterDogfoodFailure|TestOrchestratorSurveyPausesTicketOwnerAfterRecentRuntimeFailure|TestOrchestratorSurveyRoutesFailedChecksAndNoops'`
- `go test ./internal/tools -run 'TestJobDispositionPolicyRequiresCleanTreeFor(ChangesRequestedHandoff|SuccessfulNonOrchestratorHandoff)|TestJobDispositionPolicyIgnoresRuntimeLearningsOnlyDirtyState'`
- `go test ./internal/serve ./internal/tools ./internal/scanner`
- `go test ./internal/docsconsistency ./internal/docsync`
- `go test ./...`
- Live `demo-123-run6` replay confirmed product progress through Dogfood, exposed the max-turn dirty-ticket watchdog gap, and a patched `serve` replay paused survey routing while uncommitted `T-002` was present.
