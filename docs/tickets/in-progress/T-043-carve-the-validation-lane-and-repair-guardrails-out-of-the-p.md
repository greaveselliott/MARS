---
id: T-043
title: Carve the validation lane and repair guardrails out of the policy monolith to close the AD-287 sequence
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/design-docs/policy-decomposition.md"]
verified_by: "go build ./..., full go test ./internal/tools, make check, docsync audit, docsconsistency, line-multiset pure-motion verification; final-sequence two-archetype replay checkpoint per AD-287/AD-284"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: extraction landed as 5cd4eb3 (v0.50.24, tagged, assets published+verified, trunk pushed). policy_validation.go (82 functions, 1,219 lines); policy.go 2,434 -> 1,070 lines / 51 functions; 30 validation-domain tests -> policy_validation_test.go (policy_ticket_test.go 2,982 -> 1,873 lines). Reassignments: packageBuildScriptNoop + packageBuildScriptOnlySyntaxCheck -> policy_browser.go; repoHasTestFiles + repoHasGoSourceFiles + testFilePath -> policy_review.go; capabilityStopWords/capabilityLabelKeepWords vars -> policy_capability.go (slice-4 residue). Settled: checkGitPushPolicy dispatcher-resident; checkFileWritePolicy children recorded as dispatcher-resident file-write surface. Pure motion verified (line-multiset, zero removed lines); dispatch order untouched. All gates green. Replay checkpoint: demo-12 Run 4 PASS (no extraction drift, 4 tickets closed, 0 operator retries, 0 context_overflow); demo-15 API-leg run preempted by operator pause before Engineer ran (evidence-only; one orphaned pending dogfood job, T-035 shape) — must be replayed from clean reset on resume."
blocker: "paused by operator 2026-06-12; API-leg replay (demo-15) outstanding"
blocked_by: []
trace_id: "TBD"
next_action: "Resume: reset demo-15 to 62861dc (git reset --hard + clean -fdx, force-push demo-15-origin.git, delete ~/.mars-harness/db/demo-15), rerun mars-harness start --repo .../demo-15 to natural drain, write the AD-285 run section, then close T-043 and record the AD-287 sequence completion in the active plan"
dedupe_key: "public-example"
source: docs/design-docs/policy-decomposition.md (AD-287) extraction sequence step 8; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-042]
---

# T-043: Carve the validation lane and repair guardrails out of the policy monolith to close the AD-287 sequence

AD-287 extraction sequence step 8 (final slice): the validation-lane/repair domain moves to policy_validation.go as pure same-package code motion — the engineer runtime/test-build validation families, repair-scope and watermark-key helpers, validation artifact freshness/build tracking, validation command classifiers, script-name helpers, checkEngineerPostValidationCompletionShellPolicy, checkEngineerMissingArgumentCorrectionOnly, expectedExitCodeCorrectsUnexpectedValidationFailure, and the shellExecRunsValidationCommandForSession family — plus the validation-domain tests out of policy_ticket_test.go. The slice also settles the remaining assignments deferred by T-036..T-042: the file-write sub-dispatcher families, checkGitPushPolicy, and the permanent policy.go residents. This slice gates on the AD-287 final-sequence replay checkpoint: two archetypes (demo-12 package-managed frontend reset replay, fresh demo-15 Inventory/API target) with AD-285 validation reports and rule-level drift criteria.

## Functions moved (82 to policy_validation.go)

Unresolved-validation gate families: checkEngineerPostValidationCompletionShellPolicy, engineerDirtyPostValidationGuidance, checkEngineerMissingArgumentCorrectionOnly, missingArgumentCorrectionAttempted, checkEngineerUnresolvedRuntimeValidationBeforeCompletion/BeforeCommit, checkEngineerUnresolvedTestBuildValidationBeforeCompletion/BeforeCommit/BeforeFileWrite, checkEngineerUnresolvedRuntimeValidationBeforeDoneFileWrite, engineerOutstandingRuntimeValidationFailures, engineerOutstandingTestBuildValidationFailures, the four unresolved*Error builders, testBuildValidationCorrectionGuidance, compactPolicyFailureOutput, runtimeValidationCorrectionGuidance, runtimeValidationExactCorrection.

Rework/repair lane: checkEngineerTestBuildValidationReworkPolicy, checkEngineerUnexpectedRuntimeValidationReworkPolicy, shellExecRunsMissingPackageConfigBootstrap, testBuildFailureLooksLikeMissingGoModule, repoPathExists, shellExecSameJobTestBuildRepairCleanup(+NoRoot), testBuildFailureAllowsSameJobTestCleanup, engineerTestBuildRepairRemovalPath, engineerTestBuildRepairWritePath(+InScope), pathLooksLikeFixtureOrTestdata, pathLooksLikeTestFile, testBuildRepairWritePathKey, testBuildValidationRepairScopes, goFlagLikelyConsumesValue, goValidationTargetRepairScope, uniqueNonEmptyStrings.

Artifact freshness/build tracking: checkExternalValidationArtifactFreshness, shellExecRebuildsStaleValidationArtifact, validationArtifactBuildCorrection, validationArtifactBuildTarget, firstCmdMain, recordSuccessfulValidationArtifactBuild, shellExecRunsRecordedValidationArtifact, shellExecValidationArtifactInvocation, validationArtifactPath, pathIsInTempDir, validationArtifactSessionKey, validationArtifactBuildEditWatermarkKey, validationArtifactStaleAfterRuntimeEdit.

Watermark/fingerprint keys: unexpectedRuntimeValidationFailureKey(+FingerprintKey), expectedRuntimeValidationCorrectionKey, runtimeValidationRepairKey, runtimeValidationFailureEditWatermarkKey, testBuildValidationFailureFingerprintKey, testBuildValidationRepairKey, testBuildValidationFailureEditWatermarkKey, shellExecCommandFingerprint, expectedExitCodeCorrectsUnexpectedValidationFailure, shellExecLooksLikeMissingArgumentRuntimeProbe.

Validation command classifiers and script names: shellExecRunsValidationCommandForSession, shellExecCountsAsValidationEvidence, shellExecRunsTestCommand, shellFieldsRunTestCommand, shellExecRunsBuildCommand, shellFieldsRunBuildCommand, shellExecRunsRuntimeValidationCommand, shellExecRunsHTTPProbe, testScriptName, buildScriptName, runtimeScriptName.

Build-output/path helpers (T-038 deferrals): goBuildOutputPath, goBuildOutputPathFromFields, goBuildDefaultOutputName, pathResolvesInsideRepo, shellRemovalPathOperands.

Validation-probe hygiene + docsync predicate: checkRootScratchValidationWritePolicy, rootScratchValidationName, rootScratchValidationExt, sourceFileRequiresDocSync (3-vs-1 validation caller majority).

## Caller-majority reassignments settled in this slice

- packageBuildScriptNoop + packageBuildScriptOnlySyntaxCheck -> policy_browser.go (browser-only callers).
- repoHasTestFiles, repoHasGoSourceFiles, testFilePath -> policy_review.go (review-only callers; recorded deviation from the validation seed grouping).
- capabilityStopWords + capabilityLabelKeepWords vars -> policy_capability.go (slice-4 motion residue; capability-only consumers).

## Final assignments recorded in AD-287

- checkGitPushPolicy: dispatcher-resident with the trust/dispatch core (pure trunk-branch enforcement, dispatcher-only caller).
- checkFileWritePolicy sub-dispatcher + role/artifact write children (planner/security/dogfood write paths, docsync write gate, feature-contract write family): dispatcher-resident file-write surface; every child's only caller is the sub-dispatcher and no extracted domain has caller majority.
- Permanent policy.go residents: dispatchers, trust enforcement, shared ToolState keys (including the two executor-shared edit-watermark consts), shared tokenizer and read-only classifier families, toolsFeature*Pattern regexes, cross-domain shared helpers (changedFiles, dispositionBlockingFiles, summarizeChangedFiles, cleanRepoPath, repoFileExists, shellExecStopsTrackedBackgroundPID, shellExecCommandDisplay, isUntrackedRootBuildArtifact family).

## Test motion in the SAME commit

30 validation-domain tests -> policy_validation_test.go: the runtime-failure rework family, the failing-test/build repair-lane family, the external-validation-artifact family, the post-validation completion gates, and the two recordSessionToolOutcome procedure-failure accounting tests (including the T-041-deferred TestReviewHTTPProbeBeforeServerStartIsProcedureFailure). No mover-only fixtures; all shared fixtures stay in policy_ticket_test.go.

## Process finding (T-030 evidence)

ticket_create false-duplicated this slice's first title against T-040 ("Extract ... policy domain into policy_*.go (AD-287 step N)" shape); a reworded title created T-043 cleanly.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check passes; docsync audit and docsconsistency green. [met 2026-06-12]
- Pure motion verified (line-multiset equality; zero removed lines); dispatch order untouched. [met 2026-06-12]
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
- Final-sequence replay checkpoint: demo-12 and fresh demo-15 archetypes replayed with AD-285 reports and no rule-level drift.
