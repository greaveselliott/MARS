---
id: T-037
title: Extract release-gate and diff/secrets policy domains into policy_release.go and policy_diff.go (AD-287 slice 2)
priority: high
complexity: small
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/policy-decomposition.md", "docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md#run-3-v05016-ad287-slice1-checkpoint"]
verified_by: "go build ./..., full go test ./internal/tools, make check (coverage ratchet), docsync audit 0 findings, line-multiset pure-motion verification; rides the slice-1 demo-12 Run 3 replay checkpoint per AD-287 checkpoint policy"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: landed in the same dispatch as T-036. policy_release.go (4 functions) + policy_diff.go (10 functions); policy.go 6,009 -> 5,650 lines; diffStats tests moved to policy_diff_test.go. Pure motion verified; all gates green; no dedicated replay needed per AD-287 (between checkpoints)."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Next AD-287 slice: policy_shell.go (sequence step 3)."
source: docs/design-docs/policy-decomposition.md (AD-287) — extraction sequence step 2; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-036]
---

# T-037: Extract release-gate and diff/secrets policy domains into policy_release.go and policy_diff.go (AD-287 slice 2)

## Context

AD-287 extraction sequence step 2: the release-gates and diff/secrets domains are tiny and self-contained, and ride as one slice. Pure same-package code motion; dispatchers stay in policy.go.

## Requirements

- policy_release.go (4 functions): checkShellReleaseTagPolicy, shellExecReleaseTagMutation, gitTagArgsListOnly, gitTagFlagConsumesNext. MarsDocSync lists release-versioning.md + F-009 plus the internal/tools docsync minimum.
- policy_diff.go (10 functions): validateRepoDiff, ValidateRepoDiff, buildArtifactCleanupHint, checkDiffForSecrets, diffStats, diffNameStatusAddedPaths, atoiDiffField, isTicketLifecycleMoveDeletion, ticketLifecycleCounterpartInCandidates, ticketLifecycleCounterpartExists. MarsDocSync lists guardrails.md + F-007 plus the docsync minimum.
- Test motion in the SAME commit: the four diffStats tests (TestDiffStatsAllowsTicketLifecycleMoveDeletion, TestDiffStatsAllowsStagedTicketLifecycleMoveDeletion, TestDiffStatsAllowsTicketLifecycleDuplicateCleanup, TestDiffStatsStillCountsArbitraryDeletion) move from policy_ticket_test.go to policy_diff_test.go. Release-tag policy tests already live in shell_exec_test.go (their domain-named home) — no test motion for the release domain; recorded here per AD-287.

## Borderline functions deliberately NOT moved

- changedFiles: shared across validation, disposition, claim, and browser checks — stays in policy.go until a majority domain emerges (likely the validation slice).
- ticketLifecyclePathIdentity: called by ticket/file-write checks across domains and by the diff lifecycle-move helpers — ticket-lifecycle domain; stays for the policy_ticket.go slice.
- checkGitPushPolicy: git-push gating spans ticket evidence and release ordering concerns; assignment deferred to a later slice rather than forcing it here.

## Checkpoint decision (record per AD-287)

AD-287's per-slice gate places replay checkpoints after slice 1 (done: demo-12 Run 3 on v0.50.16, PASS — docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md#run-3-v05016-ad287-slice1-checkpoint) and after the final slice of the sequence. Slice 2 is a pure same-package move with byte-identical bodies between checkpoints, so it relies on the test suite per the AD; no dedicated replay is required for this slice.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check (coverage ratchet) passes; docsync audit and docsconsistency green.
- Pure motion verified (line-multiset equality; git diff --color-moved shows only moves plus new file headers).
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
