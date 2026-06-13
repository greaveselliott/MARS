---
id: T-044
title: Earn closure assurity for API ephemeral replay
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md", "go test ./...", "make check", "mars-harness validation check-closure --report docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md", "mars-harness run foundation-maintainer --repo . --dry-run --no-init --log-file <validation-root>", "node scripts/validation-target.mjs create --profile depot-supplies-api --label closure-assurity-init --root <validation-root>"]
verified_by: "foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-06-13: honest docs and Go-native closure gate landed locally; README seed init smoke confirmed Depot Supplies API brief reaches generated active plan and F-001. Fresh static-browser closure-assurity replay reached CEO->COO->CTO->Engineer->QA and passed the closure threshold. Fresh depot-supplies-api closure-assurity replay advanced past the original COO max-turn blocker, but blocked before Engineer when CTO handoff coverage reported 0/1 early product scenarios despite committed API tickets with bdd_scenarios metadata."
blocker: "Fresh API closure replay is blocked at CTO->Engineer handoff coverage reconciliation; static-browser closure threshold passed."
blocked_by: []
trace_id: "run-20260613-124152-depot-supplies-api-closure-assurity"
next_action: "Fix or explain the CTO handoff coverage mismatch for committed API scenario tickets, then rerun a fresh depot-supplies-api closure target until Engineer and QA are reached without operator run-role calls."
dedupe_key: "public-example"
source: docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md
created: 2026-06-13
depends_on: []
---

# T-044: Earn closure assurity for API ephemeral replay

## Context

The 2026-06-13 WS-D closure replay left strict AD-284 validation unconfirmed: static-browser passed, but the fresh depot-supplies-api ephemeral replay blocked in COO planning before CTO, Engineer, or QA. The closure report also overclaimed confirmed status while a required row was blocked.

Ownership classification: foundation-owned validation and bootstrap/planning convergence. The failed target is evidence only; fixes belong in mars-harness source, validation tooling, generated defaults, or role guidance if evidence supports them.

## Requirements

- Correct closure docs so blocked rows cannot be presented as confirmed closure.
- Add a Go-native mechanical closure-report gate exposed through mars-harness.
- Record Run 2 forensics, including COO failure count, guardrail signal, seed state, stop reason, and limits of the preserved trace evidence.
- Implement the smallest evidence-backed source fix without blindly raising max_turns.
- Rerun fresh static-browser and api-service closure targets per AD-284/AD-285.
- Investigate the post-seed API blocker where CTO sees committed tickets for
  `F-001-S001`..`F-001-S003` but `job_disposition_record` still reports `0/1`
  early product scenario coverage before Engineer handoff.

## Affected Files

- docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md
- docs/exec-plans/active/current-operating-plan.md
- docs/exec-plans/completed/foundation-improvement-and-operating-model.md
- cmd/mars-harness/main.go
- internal validation/checking code
- scripts/validation-target.mjs if forensics supports bootstrap seed repair

## Acceptance Criteria

- Closure report verdict remains UNCONFIRMED while any required archetype row is blocked, failed, or pending.
- mars-harness validation check-closure fails on confirmed-plus-blocked fixtures and passes the current honest report.
- make check includes the closure gate without adding Node to core gates.
- The installed mars-harness binary validates the gate and bootstrap behavior.
- Fresh static-browser and depot-supplies-api closure replays pass or leave explicit recorded blockers and unconfirmed status.
- The current API blocker is reduced to a source fix, a documented policy
  correction, or a deliberately accepted limitation with replay evidence.
