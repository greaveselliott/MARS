---
id: T-040
title: Extract ticket-lifecycle policy domain into policy_ticket.go (AD-287 step 5)
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/policy-decomposition.md"]
verified_by: "go build ./..., full go test ./internal/tools, make check (coverage ratchet), docsync audit, docsconsistency, line-multiset pure-motion verification; rides the AD-287 checkpoint policy (intermediate slice, test suite is the oracle)"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: landed. policy_ticket.go (55 functions, 1,095 lines); policy.go 4,395 -> 3,316 lines; reservedHarnessPortInScript moved to policy_browser.go (T-038 deferral resolved, sole browser caller). No test motion: policy_ticket_test.go is the ticket-domain test home per AD-287's stated end-state. Pure motion verified by line-multiset equality (zero removed lines; additions are exactly the new-file header). go build, full go test ./internal/tools, make check (coverage ratchet + vuln + fuzz smoke), docsync audit (0 findings), and docsconsistency all green. No dedicated replay per AD-287 checkpoint policy (intermediate slice)."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Next AD-287 slice: policy_review.go (sequence step 6)."
dedupe_key: "public-example"
source: docs/design-docs/policy-decomposition.md (AD-287) extraction sequence step 5; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-039]
---

# T-040: Extract ticket-lifecycle policy domain into policy_ticket.go (AD-287 step 5)

## Context

AD-287 extraction sequence step 5: the ticket-lifecycle domain (ticket_create gates and planning order, lifecycle move parsers, done-evidence checks, claim-before-mutation family) moves to policy_ticket.go as pure same-package code motion, plus the feature-contract/scenario-coverage helper family assigned to this slice by caller majority in T-039's deferral record. Dispatchers and evaluation order in policy.go stay untouched.

## Requirements

- policy_ticket.go with MarsDocSync header (delivery-operating-model + guardrails + F-001/F-005/F-007 + internal/tools docsync minimum).
- Test placement decided deliberately given policy_ticket_test.go is the legacy all-domain test mirror; record the decision here and in AD-287 Discoveries.
- Pure motion only: no renames, no signature/logic/comment changes. Line-multiset equality verification like T-036..T-039.
- Shared/borderline functions stay in policy.go with deferrals recorded here.
- Settle T-037/T-038 open assignments where the caller graph is unambiguous (checkGitPushPolicy, reservedHarnessPortInScript).

## Functions moved (55 to policy_ticket.go + 1 to policy_browser.go)

Ticket-create gates and planning order: checkTicketCreatePolicy, checkEngineerTicketCreatePolicy, isInterventionDebtTicket, checkTicketCreatePlanningOrder, isDependencyTicketForEligibleWork, ticketMetadataMentions, checkDogfoodTicketCreatePolicy, dogfoodTicketGroup, joinTicketNames.

Dogfood finding-commit family (ticket-markdown centric): checkDogfoodFindingCommitPolicy, dogfoodToolAllowedWithUncommittedFindings, uncommittedTicketMarkdownPaths, ticketMarkdownLifecyclePath.

Feature-contract/scenario-coverage helpers (per T-039's deferral record, ticket-gate caller majority): featureIDsFromScenarios, featureIDFromScenarioIDMust, featureIDFromScenarioID, featureContractExists, featureContractPath, firstUncoveredFeatureScenario, firstUncoveredFeatureScenarioFromCoverage, featureScenarioCoverage, featureContractIDs, orderedFeatureScenarioIDs, scenarioListContains.

Claim-before-mutation family: checkEngineerClaimBeforeProductMutation, checkEngineerShellExecBeforeTicketClaim, engineerCompletedTicketThisRun, engineerCompletedTicketID, engineerToolRequiresClaim, engineerToolIsTicketOnlyMutation, gitCommitTouchesOnlyWorkspaceNoise, engineerReworkTickets, engineerReworkTicketIDFromTrigger, worktreeHasInProgressToDoneTicketMove, ordinaryProductTickets.

Ticket file-write and done-evidence checks: checkTicketFileWritePolicy, checkTicketDoneContentPolicy, checkEngineerTicketEvidenceWriteRequiresValidation, checkShellTicketDoneEvidencePolicy, checkTicketDoneMoveHasOnlyTicketChanges.

Lifecycle move parsers and frontmatter/evidence helpers: ticketDoneCopySources, ticketDoneMoveSources, shellExecMovesTicketToInProgress, shellExecMovesInProgressTicketToDone, shellExecInProgressToDoneTicketID, ticketMoveOperands, ticketMoveTargetsInProgress, ticketMoveTargetsDone, parseTicketPolicyFrontmatter, missingFeatureTicketEvidence, ticketEvidencePopulated, ticketEvidenceFieldEmpty, ticketLifecycleDuplicatesOutsideState, ticketLifecyclePathIdentity, isTicketLifecycleDir.

T-038 deferral resolved: reservedHarnessPortInScript -> policy_browser.go (sole caller checkEngineerBrowserFrameworkPackageWritePolicy; unambiguous browser domain).

## Test-placement decision (recorded per dispatch)

policy_ticket_test.go IS the ticket-domain test home. AD-287 already names this end-state: "At the end of the sequence policy_ticket_test.go contains only ticket-domain tests, making its name honest... without a rename." So this slice moves NO tests; slices 6/7 and the validation slice continue moving non-ticket residue out until only ticket-domain tests remain. This avoids both a confusing rename and a pointless ticket-test shuffle into a second ticket-named file.

## Assignment resolutions

- ticketLifecyclePathIdentity: moved. Ticket caller majority (8 ticket-domain callers vs 4 in policy_diff.go vs 1 validation), confirming T-037's "ticket-lifecycle domain" classification.
- isTicketLifecycleDir: moved. 1-1 caller tie (checkTicketFileWritePolicy moving vs checkShellTicketPathPolicy in policy_shell.go) broken by ticket-lifecycle semantics; the dispatch scope explicitly includes lifecycle path parsing.
- reservedHarnessPortInScript: settled to policy_browser.go (unambiguous caller graph).
- checkGitPushPolicy: STILL DEFERRED to the validation slice. Caller graph gives no signal (dispatcher-only caller) and the body is pure trunk-branch enforcement ("strict trunk only allows pushing main") — neither ticket evidence nor release ordering as T-037 speculated. Ambiguous between shell git-safety and staying with the dispatchers.

## Borderline functions deliberately NOT moved (deferrals)

- countCoveredFeatureScenarios, firstUncoveredFeatureScenarios, featureContractSuperseded: deviate from T-039's "and friends" family grouping — their remaining callers are all disposition-domain (checkCTODispositionTicketBatch, ctoImplementationHandoffTicketBatchSatisfied, checkPlanningDispositionFeatureSpecificity, productCapabilityCoverageFeatureContents), so per AD-287's caller-majority rule they wait for the policy_disposition.go slice.
- checkEngineerPostValidationCompletionShellPolicy + engineerDirtyPostValidationGuidance: validation-lane majority of internal dependencies (validationCommandSuccessKey, shellExecRunsRecordedValidationArtifact, browser validation evidence helpers, dispositionBlockingFiles); wait for the validation slice.
- Feature-contract write-policy family (checkFeatureScenarioIDPolicy, featureScenarioContractMismatches, formatFeatureScenarioMismatches, duplicateFeatureScenarioHeadings, formatFeatureScenarioDuplicates, checkFeatureFileWritePolicy, roleMayWriteFeatureContracts, featureContractIDFromName) and planner write-path family (checkPlannerFileWritePolicy, cooPlanningWritePath, ctoTechnicalPlanningWritePath, ceoStrategyWritePath): stay with the checkFileWritePolicy sub-dispatcher in policy.go; the validation slice settles the file-write surface.
- repoFileExists: cross-domain shared (dependency_sync.go, workspace_hygiene.go, ticket planning order, validation artifact target); stays in policy.go, likely permanently.
- changedFiles, cleanRepoPath, checkFileWritePolicy: shared surface, stay in policy.go per prior-slice records.

## Checkpoint decision (record per AD-287)

Intermediate slice between the slice-1 (demo-12 Run 3, PASS) and final-sequence replay checkpoints. Pure same-package motion with byte-identical bodies rides the full test suite; no dedicated replay in this dispatch.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check passes; docsync audit and docsconsistency green. [met 2026-06-12]
- Pure motion verified (line-multiset equality; zero removed lines). [met 2026-06-12]
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
