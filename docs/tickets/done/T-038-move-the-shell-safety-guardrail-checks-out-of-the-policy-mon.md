---
id: T-038
title: Move the shell-safety guardrail checks out of the policy monolith (AD-287 step 3)
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/policy-decomposition.md"]
verified_by: "go build ./..., full go test ./internal/tools, make check (coverage ratchet), docsync audit, docsconsistency, line-multiset pure-motion verification; rides the AD-287 checkpoint policy (intermediate slice, test suite is the oracle)"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: landed. policy_shell.go (33 functions, 666 lines); policy.go 5,650 -> 5,005 lines; four shell-domain tests moved to policy_shell_test.go (policy_ticket_test.go 5,929 -> 5,822 lines). Pure motion verified by line-multiset equality (zero removed lines; additions are exactly the two new-file headers). go build, full go test ./internal/tools, make check (coverage ratchet + vuln + fuzz smoke), docsync audit (0 findings), and docsconsistency all green. No dedicated replay per AD-287 checkpoint policy (intermediate slice)."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Next AD-287 slice: policy_capability.go (sequence step 4)."
dedupe_key: "public-example"
source: docs/design-docs/policy-decomposition.md (AD-287) extraction sequence step 3; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-037]
---

# T-038: Move the shell-safety guardrail checks out of the policy monolith (AD-287 step 3)

## Context

AD-287 extraction sequence step 3: the shell-safety domain (argv/shell-command validation, destructive-operation detection, git safety, path containment, build-output guards, foreground/long-running detection, no-op loops) moves to policy_shell.go as pure same-package code motion. Dispatchers and evaluation order in policy.go stay untouched. This domain guards destructive operations: the test suite is the oracle; any internal/tools failure after motion means the motion is wrong, never the test.

## Requirements

- policy_shell.go with MarsDocSync header (guardrails.md + F-007 + internal/tools docsync minimum).
- Matching shell-domain test blocks move from policy_ticket_test.go to policy_shell_test.go in the SAME commit.
- Pure motion only: no renames, no signature/logic/comment changes. Line-multiset equality verification like T-036.
- Shared/borderline functions stay in policy.go with deferrals recorded here.
- Shared session-key constants stay in policy.go per AD-287 shared-key rule.

## Functions moved (33)

Entry checks: checkShellPolicy, checkShellMarsHarnessCLIPolicy, validateShellExecPolicyArgs, checkShellBuildOutputPolicy, checkForegroundLongRunningShellPolicy, checkEngineerRepeatedNoopPolicy, checkShellTicketPathPolicy, shellExecGeneratedArtifactCleanup.

Helpers with shell-only callers: shellExecMarsHarnessArgs, shellEnvAssignment, goBuildValidationCorrection, goBuildCommandFields, validationBinaryOutputSuggestion, cleanShellDisplayToken, dependencyShellOperation, nextTokenIs, broadGeneratedTraversal, hasGeneratedExcludeToken, likelyForegroundLongRunningCommand, serverScriptName, goRunLikelyStartsServer, goRunTargets, goRunCandidateFiles, sourceContainsServerMarker.

Destructive-operation/git-safety family: forbiddenShellOperation, hasGitSubcommand, hasToken, hasGitForcePushFlag, hasGitCleanForceDelete, hasGitBranchDelete, hasRootRemoval, hasShellRemoval, hasFindDelete.

hasToken is the one caller-majority assignment: two moving shell callers (broadGeneratedTraversal, forbiddenShellOperation) versus one in policy_release.go (shellExecReleaseTagMutation); assigned shell per AD-287's caller-majority rule.

Test motion in the SAME commit: TestShellExecPolicyBlocksTicketRootMarkdown, TestShellExecPolicyAllowsTicketLifecycleMove, TestEngineerRepeatedNoopAfterValidationBlocksWithCommitGuidance, TestEngineerRepeatedNoopBeforeImplementationRedirectsToFileWrite -> policy_shell_test.go. The bulk of shell-safety unit tests already live in shell_exec_test.go (their domain-named home), mirroring the T-037 release-domain precedent.

## Borderline functions deliberately NOT moved (deferrals)

- Core tokenizer/normalizer family (shared by validation, ticket, browser, release domains and executor.go): shellFields, shellFieldsPreserveCase, normalizedShellExecFields, simpleCDShellCommandTrailingFields, shellCommandHasControlSyntax, cleanShellPathToken, filepathBase, shellCommandFields, shellControlToken. Stay in policy.go until a majority domain emerges; several are likely permanent shared helpers.
- Read-only shell classifier family: shellExecReadOnly (called by both dispatchers, checkReviewerShellExecValidationPolicy, engineerToolRequiresClaim), shellTokensReadOnly, sedReadOnly, findReadOnly, gitShellReadOnly, gitShellSubcommand (also called from policy_release.go), hasGitShellOutputFlag. Shared across review/ticket/release; stays as a cluster.
- Build-output/path-containment helpers shared with validation and diff domains: pathResolvesInsideRepo, shellRemovalPathOperands, goBuildOutputPath, goBuildOutputPathFromFields, goBuildDefaultOutputName, isUntrackedRootBuildArtifact, isAllowedRootBuildArtifactName, goModuleBinaryName, fileLooksBinary, lineListContains. All have validation-lane or policy_diff.go callers; per the shared rule they stay for the validation-lane slice to reconsider.
- shellExecStopsTrackedBackgroundPID: AD-287's seed listed background cleanup under shell, but its callers are browser, validation, and review checks; shared, stays.
- reservedHarnessPortInScript: sole caller is checkEngineerBrowserFrameworkPackageWritePolicy in policy_browser.go; left in policy.go and recorded as a browser-domain reassignment candidate.
- checkShellTicketDoneEvidencePolicy, checkTicketDoneMoveHasOnlyTicketChanges, and the ticket-move parser family: ticket domain per T-036; stay for the policy_ticket.go slice.
- checkGitPushPolicy: assignment still deferred (T-037).

## Checkpoint decision (record per AD-287)

Intermediate slice between the slice-1 (demo-12 Run 3, PASS) and final-sequence replay checkpoints. Pure same-package motion with byte-identical bodies rides the full test suite; no dedicated replay in this dispatch.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check passes; docsync audit and docsconsistency green.
- Pure motion verified (line-multiset equality; git diff --color-moved shows only moves plus new file headers).
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
