---
id: T-042
title: Carve job-disposition and CTO handoff gates from the monolith (AD-287 step 7)
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/policy-decomposition.md"]
verified_by: "go build ./..., full go test ./internal/tools, make check (coverage ratchet), docsync audit, docsconsistency, line-multiset pure-motion verification; rides the AD-287 checkpoint policy (intermediate slice, test suite is the oracle)"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: landed. policy_disposition.go (27 functions + featureScenarioSection type, 633 lines); policy.go 3,089 -> 2,434 lines; 34 disposition tests + 19 mover-only fixtures moved to policy_disposition_test.go and TestQAApprovalRequiresGoTestsForGoSource to policy_review_test.go (policy_ticket_test.go 5,376 -> 2,982 lines). Four caller-majority reassignments settled (pendingCTOHandoffRequiredScenarios, splitScenarioList, quoteStringArray -> policy_ticket.go; roleRequiresDocSyncForSuccessfulDisposition -> policy_review.go). Slice-6 orphaned doc comments for the three exported review functions reattached in policy_review.go; doc-comment audit of all slice 1-7 moves found no other orphan. Pure motion verified by line-multiset equality (zero removed lines). go build, full go test ./internal/tools, make check, docsync audit (0 findings), docsconsistency all green. No dedicated replay per AD-287 checkpoint policy (intermediate slice)."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Final AD-287 extraction slice: policy_validation.go (sequence step 8) plus the final two-archetype replay checkpoint, in a dedicated dispatch."
dedupe_key: "public-example"
source: docs/design-docs/policy-decomposition.md (AD-287) extraction sequence step 7; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-041]
---

# T-042: Carve job-disposition and CTO handoff gates from the monolith (AD-287 step 7)

## Context

AD-287 extraction sequence step 7: the disposition/handoff domain (job_disposition_record gates, CTO handoff batch checks, docsync-at-disposition checks) moves to policy_disposition.go as pure same-package code motion, plus the capability-coverage consumers deferred from T-039 (checkProductCapabilityScenarioCoverage, scenarioCoversProductCapability, productScenarioIDsForHandoff, earlyCTOHandoffRequiredScenarios) and the deferrals from T-040/T-041 whose callers are all disposition-domain. Dispatchers and evaluation order in policy.go stay untouched.

## Requirements

- policy_disposition.go with MarsDocSync header (delivery-operating-model + guardrails + F-001/F-005/F-007 + internal/tools docsync minimum).
- Disposition-domain tests (job-disposition gates, COO-completion coverage family, CTO-completion batch family) move from policy_ticket_test.go to policy_disposition_test.go in the SAME commit, with fixture helpers used only by movers.
- Pure motion only: no renames, no signature/logic/comment changes. Line-multiset equality verification like T-036..T-041.
- Settle remaining unambiguous caller-majority assignments (quoteStringArray, pendingCTOHandoffRequiredScenarios, splitScenarioList, roleRequiresDocSyncForSuccessfulDisposition).

## Functions moved (27 + 1 type to policy_disposition.go)

job_disposition_record gates: checkJobDispositionRecordPolicy, checkSuccessfulDispositionUnresolvedTicketCreation, checkEngineerDispositionTicketState, checkDogfoodDispositionValidationEvidence, checkPlanningDispositionFeatureSpecificity, checkSuccessfulDispositionDocSync, docSyncDispositionStatus, summarizeDocSyncFindings, successfulDispositionStatus, dispositionRequiresCleanTree, successfulReviewDispositionStatus (T-041 deferral; both remaining callers disposition).

CTO handoff batch checks: checkCTODispositionTicketBatch, recordCTOHandoffRequiredScenarios, ctoHandoffFeatureContractIDs, activePlanFeatureIDs, ctoImplementationHandoffTicketBatchSatisfied, planningRoleCanHandOffTicketCreation, earlyCTOHandoffRequiredScenarios, productScenarioIDsForHandoff.

Capability-coverage consumers (T-039 deferral): checkProductCapabilityScenarioCoverage, productCapabilityCoverageFeatureContents, scenarioCoversProductCapability, scenarioLooksProductImplementation, orderedFeatureScenarioSections + the featureScenarioSection type (both consumers moved).

T-040 deferrals with disposition-only callers: countCoveredFeatureScenarios, firstUncoveredFeatureScenarios, featureContractSuperseded.

## Caller-majority reassignments settled in this slice

- pendingCTOHandoffRequiredScenarios + splitScenarioList -> policy_ticket.go: callers are checkTicketCreatePlanningOrder and ticket_create.go only. The producer (recordCTOHandoffRequiredScenarios) lives in policy_disposition.go; the shared session key constant stays in policy.go per AD-287's shared-key rule, which keeps the producer/consumer pair discoverable.
- quoteStringArray -> policy_ticket.go: sole caller checkTicketCreatePlanningOrder.
- roleRequiresDocSyncForSuccessfulDisposition -> policy_review.go: three review callers vs one disposition caller.

## Slice-6 motion defect fixed

The T-041 extraction left the doc comments of ReviewTerminalEvidenceSatisfied, MarkReviewTerminalDispositionRequired, and ReviewTerminalDispositionGuidance orphaned in policy.go (the extraction tooling started function spans at the func line, leaving preceding comment blocks behind, stacked above shellExecStopsTrackedBackgroundPID). This slice reattaches all three to their declarations in policy_review.go. A doc-comment audit of every function moved in slices 1-7 (98 names against the v0.50.20 file) found no other orphaned comment.

## Test motion in the SAME commit

34 tests -> policy_disposition_test.go: the nine job-disposition gate tests (TestEngineerDispositionPolicyRequiresTicketDoneBeforeSuccess, TestSuccessfulDispositionBlocksUnresolvedTicketCreationFailure, TestCTODispositionAllowsCoveredBatchAfterDuplicateTicketCreateFailure, TestPlanningRoleCanHandOffUnownedTicketCreationFailure, the five TestJobDispositionPolicy* tests), the twenty TestCOOCompletion* coverage-gate tests (per T-039's blended-test record their primary subject is the disposition coverage gate), and the five TestCTOCompletion* batch tests. Nineteen Tetris fixture helpers used only by movers move along; shared fixtures (writeDetailedTetrisBrief, writeTetrisFeatureWithFullScenarioSchedule, writeTetrisFeatureWithGenericScenarios, writePolicyPlan, writePolicyFeature, setupPolicyTicketRepo, writePolicyTicket) stay in policy_ticket_test.go because ticket-domain tests (TestCTOTicketCreate*) also consume them. TestQAApprovalRequiresGoTestsForGoSource -> policy_review_test.go as review residue per the T-040 test-placement decision.

## Borderline functions deliberately NOT moved (deferrals to the validation slice)

- dispositionBlockingFiles: callers span disposition, validation, browser, shell, and ticket domains — cross-domain shared, stays in policy.go.
- summarizeChangedFiles: callers span disposition, release, and ticket domains — stays.
- planningRoleCannotMutateWithShell: dispatcher-only caller, trust-enforcement adjacent; stays with the dispatchers.
- checkGitPushPolicy: still no caller-graph signal (T-040 record); validation slice settles it.

## Checkpoint decision (record per AD-287)

Intermediate slice between the slice-1 (demo-12 Run 3, PASS) and final-sequence replay checkpoints. Pure same-package motion with byte-identical bodies rides the full test suite; no dedicated replay in this dispatch. The final two-archetype replay follows the validation-lane slice (sequence step 8) in its own dispatch.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check passes; docsync audit and docsconsistency green. [met 2026-06-12]
- Pure motion verified (line-multiset equality; zero removed lines). [met 2026-06-12]
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
