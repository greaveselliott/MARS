---
id: T-041
title: Give review terminal gates a dedicated policy file (AD-287 step 6)
priority: high
complexity: small
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/policy-decomposition.md"]
verified_by: "go build ./..., full go test ./internal/tools, make check (coverage ratchet), docsync audit, docsconsistency, line-multiset pure-motion verification; rides the AD-287 checkpoint policy (intermediate slice, test suite is the oracle)"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: landed. policy_review.go (11 functions, 243 lines); policy.go 3,316 -> 3,089 lines; seventeen review-domain tests moved to policy_review_test.go (policy_ticket_test.go 5,790 -> 5,376 lines). Pure motion verified by line-multiset equality (zero removed lines; additions are exactly the two new-file headers). One recorded non-motion line change: trimmed the gofmt-reported trailing blank line slice 5 left at policy.go EOF. go build, full go test ./internal/tools, make check (coverage ratchet + vuln + fuzz smoke), docsync audit (0 findings), and docsconsistency all green. No dedicated replay per AD-287 checkpoint policy (intermediate slice)."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Next AD-287 slice: policy_disposition.go (sequence step 7)."
dedupe_key: "public-example"
source: docs/design-docs/policy-decomposition.md (AD-287) extraction sequence step 6; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-040]
---

# T-041: Give review terminal gates a dedicated policy file (AD-287 step 6)

## Context

AD-287 extraction sequence step 6: the review-gates domain (review terminal gates around the ReviewTerminalEvidenceSatisfied family, reviewer shell validation policy, changes-requested ownership checks) moves to policy_review.go as pure same-package code motion. Dispatchers and evaluation order in policy.go stay untouched.

## Requirements

- policy_review.go with MarsDocSync header (guardrails + F-005/F-007 + internal/tools docsync minimum).
- Review-domain tests move from policy_ticket_test.go to policy_review_test.go in the SAME commit (per the T-040 test-placement decision: non-ticket residue moves out of policy_ticket_test.go slice by slice).
- Pure motion only: no renames, no signature/logic/comment changes. Line-multiset equality verification like T-036..T-040.
- Shared/borderline functions stay in policy.go with deferrals recorded here.

## Functions moved (11)

Review terminal gates: checkReviewTerminalDispositionOnly, reviewTerminalDispositionGuidance, ReviewTerminalEvidenceSatisfied, MarkReviewTerminalDispositionRequired, ReviewTerminalDispositionGuidance.

Reviewer shell validation policy: checkReviewValidationFailureShellPolicy, checkReviewerShellExecValidationPolicy, shellExecRunsValidationCommand (sole caller is checkReviewerShellExecValidationPolicy; assigned review by caller majority).

Review approval evidence and changes-requested ownership: checkReviewDispositionValidationEvidence, checkReviewChangesRequestedFeedbackOwnership (both called only from checkJobDispositionRecordPolicy, but assigned review per the AD-287 seed table: QA/Security approval evidence and changes-requested ownership are review-domain doctrine).

Shared role predicate: reviewRoleRequiresValidationEvidence (six review-domain callers plus executor.go; review home is unambiguous).

## Test motion in the SAME commit (17 tests)

TestReviewApprovalRequiresPassingValidationWhenTestsExist, TestReviewValidationFailureBlocksFurtherShellBeforeDisposition, TestReviewValidationFailureAllowsExactExpectedExitCorrection, the eight TestReviewShellExecPolicy* tests, the three TestReviewTerminalEvidence* tests, TestReviewTerminalDispositionRequiredBlocksFurtherShellExec, and the two TestQAChangesRequested* ownership tests -> policy_review_test.go. TestReviewHTTPProbeBeforeServerStartIsProcedureFailure stays in policy_ticket_test.go: despite the Review name its subject is validation procedure-failure accounting through recordSessionToolOutcome (validation slice).

## Borderline functions deliberately NOT moved (deferrals)

- successfulReviewDispositionStatus: callers are checkReviewDispositionValidationEvidence (moved), checkDogfoodDispositionValidationEvidence and checkEngineerDispositionTicketState (both disposition slice) — caller majority disposition; waits for policy_disposition.go.
- expectedExitCodeCorrectsUnexpectedValidationFailure: 1-1 tie (checkEngineerUnexpectedRuntimeValidationReworkPolicy validation vs checkReviewValidationFailureShellPolicy review); validation semantics; waits for the validation slice.
- shellExecRunsValidationCommandForSession + shellExecCountsAsValidationEvidence: called only from executor.go outcome recording; validation accounting; wait for the validation slice.
- shellExecStopsTrackedBackgroundPID: three-domain shared (validation, review, browser) per T-038's record; stays in policy.go.

## Recorded non-motion change

Slice 5 (T-040) moved the last function out of policy.go EOF and left a trailing blank line that gofmt reports; this slice trims that one blank line. It is the only non-motion line change and nets to zero in the line-multiset check.

## Checkpoint decision (record per AD-287)

Intermediate slice between the slice-1 (demo-12 Run 3, PASS) and final-sequence replay checkpoints. Pure same-package motion with byte-identical bodies rides the full test suite; no dedicated replay in this dispatch.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check passes; docsync audit and docsconsistency green. [met 2026-06-12]
- Pure motion verified (line-multiset equality; zero removed lines). [met 2026-06-12]
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
