---
id: T-011
title: Measure and optimize factory pace
priority: high
complexity: large
work_type: intervention-debt
bdd_scenarios:
  - F-005-S003
  - F-008-S008
  - F-006-S015
  - F-006-S016
  - F-007-S010
  - F-007-S011
  - F-007-S012
  - F-006-S017
  - F-005-S010
  - F-005-S011
  - F-005-S015
  - F-005-S016
  - F-006-S019
  - F-006-S020
  - F-006-S021
  - F-006-S022
  - F-007-S041
end_to_end_evidence: not_applicable
evidence_links:
  - "go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries"
  - "go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive"
  - "go test ./internal/queue -run TestQueue_activeJobForRepoRole"
  - "go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'"
  - "go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|RejectsBarePortCommands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess|KillTrackedBackgroundPIDKillsDescendant)'"
  - "go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|BlocksDefaultGoBuildInsideRepoBeforeArtifact|BlocksDefaultGoBuildInShellCommandBeforeArtifact|AllowsGoBuildOutputOutsideRepo|NoopArgsNotMaskedByDirtyArtifact)'"
  - "go test ./internal/tools -run 'TestShellExecRejectsExternalTimeoutCommands|TestFileWriteBlocksNewRootValidationScript|TestFileWriteAllowsExistingRootValidationScriptUpdate'"
  - "go test ./internal/tools -run 'TestFileWriteBlocksNewRootScratchProbe|TestSecurityFileWriteBlocksProductRemediation|TestSecurityFileWriteAllowsSecurityReport'"
  - "go test ./internal/tools -run TestKillBackgroundProcsKillsEscapedChildProcess"
  - "go test ./internal/tools -run TestJobDispositionPolicyBlocksSuccessfulReviewWhenDocSyncFails"
  - "go test ./internal/tools -run 'TestReviewApprovalRequiresPassingValidationWhenTestsExist|TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation|TestShellExecPolicyAllowsEvidencedEnablerTicketDoneMove|TestShellExecPolicyBlocksEnablerTicketDoneMoveWithoutEvidence|TestShellExecArgvAllowsLiteralNewlineArgument|TestRecordSessionToolOutcomeTracksValidationCommands'"
  - "go test ./internal/tools -run TestFileWritePolicyBlocksScenarioIDsThatDoNotMatchFeatureContract"
  - "go test ./internal/tools -run 'TestEngineerPostValidationGateAllowsValidationWhileImplementationDirty|TestEngineerRepeatedNoopAfterValidationBlocksWithCommitGuidance|TestEngineerMustReopenDoneTicketBeforeProductMutation|TestEngineerCompletionCommitAllowsTicketMoveToDoneWithProductFiles'"
  - "go test ./internal/tools -run TestRecordSessionToolOutcomeTracksNoopFailures"
  - "go test ./internal/tools -run 'TestExternalValidationArtifactMustBeBuiltInSameSession|TestEngineerPostValidationAllowsFreshExternalValidationArtifact|TestRecordSessionToolOutcomeTracksValidationArtifactBuildAndRun'"
  - "go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksRuntimeValidationCommands|TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion'"
  - "go test ./internal/tools -run 'TestReviewApprovalRequiresPassingValidationWhenTestsExist|TestRecordSessionToolOutcomeTracks(RuntimeValidationCommands|ValidationArtifactBuildAndRun|ValidationCommands)'"
  - "go test ./internal/tools -run 'TestEngineerPostValidationCommitBlocksExploratoryShellUntilTicketDone|TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion'"
  - "go test ./internal/tools -run TestEngineerPostValidationDirtyNoopBlocksBeforeGenericNoop"
  - "go test ./internal/scanner -run 'TestInit_success|TestInitRolePromptsIncludeOperationalGuidance'"
  - "go test ./internal/docsync"
  - "go test ./internal/telemetry -run 'TestClassify|TestRetryable'"
  - "go test ./internal/scanner -run TestInit_success"
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-12-tool-argument-and-matrix-replay--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#non-static-matrix-replay-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-scheduler-skip-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-canonical-bootstrap-guidance-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-module-artifact-cleanup-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-generated-artifact-cleanup-hints-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-managed-background-validation-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-build-output-prevention-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-bare-port-rejection-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-scratch-validation-prevention-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-background-descendant-cleanup-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-default-go-build-preflight-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-tracked-background-kill-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-no-op-shell-guidance-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-docsync-disposition-gate-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-no-op-hard-error-and-docsync-write-preflight-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-claim-argv-normalization-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-claim-first-shell-policy-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-foreground-server-preflight-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-bounded-security-review-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#cli-matrix-replay-note-stats-cli--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#cli-matrix-replay-note-stats-cli-rework-bounds--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#cli-matrix-replay-note-stats-cli-contract-bounds--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#cli-matrix-replay-note-stats-cli-closure-bounds--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#cli-matrix-replay-dogfood-finding-handoff-bounds--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#cli-matrix-replay-review-validation-gates-and-remediation-closure
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#note-stats-cli-run-10-completion-commit-needed-a-rework-guard-exception
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#note-stats-cli-run-11-repeated-no-op-shell-calls-stopped-product-commit
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#note-stats-cli-run-12-qa-validation-surface-and-fresh-artifact-proof
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#note-stats-cli-run-13-direct-runtime-probe-classification
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#note-stats-cli-run-14-expected-runtime-error-probes-should-not-poison-qa-approval
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#note-stats-cli-run-15-post-validation-gate-needed-a-non-shell-next-step
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-62-slugify-cli-completed-full-local-lifecycle-after-rework-guidance
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-65-inventory-api-canary-reached-product-rework-exposed-post-validation-no-op-failure
  - docs/validation/baselines/2026-06-12-factory-pace-baseline.md
  - docs/validation/reports/2026-06-12-demo-11-pace-baseline.md#run-1-inventoryapi-canary-on-v0502--2026-06-12
  - docs/validation/reports/2026-06-11-demo-11-pace-baseline.md#run-1-inventoryapi-canary-on-v0501--2026-06-11
verified_by: "partial: go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries; go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive; go test ./internal/queue -run TestQueue_activeJobForRepoRole; go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'; go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|RejectsBarePortCommands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess|KillTrackedBackgroundPIDKillsDescendant|NoopReturnsCompletionGuidance|NoopAfterBackgroundListsTrackedPID)'; go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|BlocksDefaultGoBuildInsideRepoBeforeArtifact|BlocksDefaultGoBuildInShellCommandBeforeArtifact|BlocksGoBuildOutputInShellCommandSegmentBeforeArtifact|AllowsGoBuildOutputOutsideRepo|NoopArgsNotMaskedByDirtyArtifact)'; go test ./internal/tools -run 'TestShellExecRejectsExternalTimeoutCommands|TestFileWriteBlocksNewRootValidationScript|TestFileWriteAllowsExistingRootValidationScriptUpdate|TestShellExecPolicyBlocksForegroundServerCommands|TestShellExecPolicyAllowsForegroundGoRunForNonServerCLI'; go test ./internal/tools -run TestKillBackgroundProcsKillsEscapedChildProcess; go test ./internal/tools -run TestJobDispositionPolicyBlocksSuccessfulReviewWhenDocSyncFails; go test ./internal/tools -run 'TestFileWritePolicyRequiresDocSyncForSourceFiles|TestFileWritePolicyRejectsSourceDocSyncMissingDoc|TestEngineerClaimPolicyRequiresInProgressBeforeProductMutation|TestFileWritePolicyBlocksScenarioIDsThatDoNotMatchFeatureContract'; go test ./internal/tools -run 'TestDogfoodUncommittedFindingBlocksFurtherValidationAndTickets|TestDogfoodFindingCreatedInRunRequiresDispositionBeforeFurtherValidation|TestReviewApprovalRequiresPassingValidationWhenTestsExist|TestShellExecPolicyAllowsEvidencedEnablerTicketDoneMove|TestShellExecPolicyBlocksEnablerTicketDoneMoveWithoutEvidence|TestShellExecArgvAllowsLiteralNewlineArgument|TestRecordSessionToolOutcomeTracksValidationCommands'; go test ./internal/orgstate -run TestDecodeDispositionNormalizesStringLists; go test ./internal/docsync; go test ./internal/telemetry -run 'TestClassify|TestRetryable'; run12, demo-api-run1 through demo-api-run20 scores exports and trace summaries captured live Factory Pace baselines"
owner: "Codex"
last_attempt: >-
  2026-05-21: first slice adds Factory Pace rows to QUALITY_SCORE.md by joining
  scoring outcomes to trace summaries; run12 export shows Engineer 92 turns/45
  tools and Dogfood 66 turns/32 tools. CLI and API canaries progressively fixed
  Security authority, scratch probes, contract drift, closure drift, Dogfood
  handoff, review validation evidence, newline argv, enabler closure metadata,
  review-ticket reopening, completion-commit classification, no-op loops, QA
  shell validation, temp artifact freshness, direct runtime validation, expected
  negative runtime probes, post-validation evidence update, startup retry
  persistence, failing test output guidance, missing Go module repair, raw
  `go get`, and assertion-test preservation. demo-inventory-api-run65 then
  confirmed product planning, ordinary ticketing, implementation, QA, Security,
  Dogfood finding creation, Orchestrator rework routing, and Engineer route
  repair validation on a distinct HTTP JSON API before exposing the next
  terminal gap: post-runtime-validation no-op placeholders with dirty rework.
  AD-218 now blocks the first such no-op with tracked-PID cleanup, commit,
  evidence, ticket closure, push, and qa_review guidance.
  2026-06-11: the demo-11 Inventory/API replay on v0.50.1 stopped at
  cto-weekly max_turns inside a deterministic ticket_create false-duplicate
  wedge (finding F1, ticket T-030), so the AD-218 Engineer confirmation could
  not execute; AD-218 remains unvalidated on this archetype until the Phase 3
  fix lands. The run did live-validate the T-027 Convergence And Guardrails
  export section. 2026-06-12: that run executed on the heavy quality-profile
  model (Qwen3-Coder Q8_0) which was maxing unified memory; the operator
  swapped the harness to the balanced model, so the heavy run is reclassified
  evidence-only and the Phase 3 pace baseline is re-captured on the balanced
  model (model identity is now part of the AD-285 measurement contract).
  Later on 2026-06-12: the balanced-model rerun (v0.50.2, Qwen3-Coder Q4_K_M
  reasoning ctx 131072 / coding ctx 32768, Gemma-4-E4B Q5_K_M fast) completed
  a full 51.6-minute lifecycle on a fresh demo-11 — T-001 done with three QA
  rework loops, two security audits, and a dogfood attempt — and the dated
  pace baseline is recorded in
  docs/validation/baselines/2026-06-12-factory-pace-baseline.md. AD-218 is
  now confirmed live on this archetype (0 no-op outcomes across 6 engineer
  jobs including rework). The heavy-model T-030 wedge did not recur. New
  foundation-owned findings: qa and dogfood circle_detected runtime failures
  stall lifecycle reach pending operator retry (report findings F1/F2).
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: >-
  Phase 3 pace optimization against the 2026-06-12 balanced-model baseline:
  engineer is the slowest role (75.2 avg turns, 244.7s avg wall, 2 limit
  stops). Harden terminal-role convergence so qa/dogfood circle_detected
  runtime failures (2026-06-12 report findings F1/F2) stop capping lifecycle
  reach, then calibrate max-turn limits once post-fix replays accumulate pace
  data against the baseline. All comparisons must run on the same model
  identity recorded in the baseline (AD-285).
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "factory_pace"
  confidence: "high"
  evidence: "User reported on 2026-05-19 that agents take many turns before solid outcomes, the upper turn limit is too low, and factory pace is not measured."
  origin: "user request 2026-05-19"
  repo: "mars-harness"
  role: "all factory agents"
  severity: "high"
  target: "foundation and deployed harnesses"
source: user request 2026-05-19
created: 2026-05-19
depends_on: []
---

# T-011: Measure and optimize factory pace

## Context

Factory pace is currently an implicit concern rather than a measured performance signal. The user reported that different agents take many turns before reaching a solid outcome while the upper turn limit is also too low for hard work. That combination creates two separate risks: wasted turns when agents drift, and premature failure when a valid task needs a longer but still productive run.

The factory needs a durable pace metric that shows how quickly roles reach useful outcomes, how many turns and tool calls are spent on avoidable loops, and whether runtime limits are calibrated by evidence instead of guesswork.

## Requirements

- Build a clear baseline view of current factory pace from available traces, queue/job outcomes, scoring evidence, and dogfood reports.
- Define pace metrics that operators and future automation can reason about, including turns-to-solid-outcome, tool calls per successful outcome, wall-clock duration, retry count, max-turn failure rate, and avoidable-loop indicators.
- Decide where pace belongs in the operating model: scoring, quality score export, dashboard/CLI visibility, trace summaries, telemetry triage, or some combination.
- Plan and implement targeted speed improvements only after the baseline identifies the largest turn sinks. Candidate fixes may include role prompt tightening, terminal-handoff enforcement, context routing changes, deterministic preflight/remediation shortcuts, model/profile routing, or max-turn calibration by role.
- Calibrate upper limits so productive work can continue while unproductive loops still stop quickly and produce actionable intervention debt.
- Add tests and repeatable evidence that pace improved or that the new metric exposes the bottleneck honestly when improvement is not yet proven.

## Affected Files

- internal/agent/
- internal/serve/
- internal/scoring/
- internal/qualityscore/
- internal/trace/
- internal/tools/
- internal/scanner/
- internal/ui/
- internal/dashboard/
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-008-scoring-trust-quality.md
- docs/design-docs/scoring-system.md
- docs/design-docs/dogfood-and-decisions.md
- docs/QUALITY_SCORE.md

## BDD Evidence

- Scenario IDs: F-008-S008, F-006-S015, F-006-S016, F-007-S010, F-007-S011, F-007-S012, F-007-S041, and F-005-S015.
- Evidence links: `go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries` covers the first quality-export pace slice.
- Evidence links: `go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive` and `go test ./internal/queue -run TestQueue_activeJobForRepoRole` cover the scheduled duplicate-work fix.
- Evidence links: `go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'` covers the bounded build-artifact cleanup exception.
- Evidence links: `go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'` covers managed long-running server validation tool behavior.
- Evidence links: `go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|AllowsGoBuildOutputOutsideRepo|NoopArgsNotMaskedByDirtyArtifact)'` covers repo-local validation binary prevention and dirty-artifact validation ordering.
- Evidence links: `go test ./internal/tools -run 'TestShellExecRejectsExternalTimeoutCommands|TestFileWriteBlocksNewRootValidationScript|TestFileWriteAllowsExistingRootValidationScriptUpdate'` covers scratch validation script prevention and portable timeout guidance.
- Evidence links: `go test ./internal/tools -run 'TestFileWriteBlocksNewRootScratchProbe|TestSecurityFileWriteBlocksProductRemediation|TestSecurityFileWriteAllowsSecurityReport'` covers root scratch-probe prevention and Security report-only file writes.
- Evidence links: `go test ./internal/tools -run TestKillBackgroundProcsKillsEscapedChildProcess` covers cleanup of wrapper-spawned background descendants.
- Evidence links: `go test ./internal/scanner -run TestInit_success` covers generated CEO/COO canonical feature-contract reuse.
- Evidence links: `demo-api-run18` confirmed claim-first shell policy and source DocSync write preflight in the live implementation path, then exposed foreground `go run main.go` server validation as the next timeout/no-op sink.
- Evidence links: `demo-api-run19` confirmed managed background API validation, ticket completion, QA approval, target quality-score export, and zero open target intervention-debt tickets, then exposed Security terminal-disposition turn budget as the next sink.
- Evidence links: `demo-api-run20` confirmed bounded Security terminal evidence, Dogfood validation, target release notes, quality-score export grade `A`, and zero open target intervention-debt tickets.
- Evidence links: `demo-cli-run1` confirmed the lifecycle reaches product-specific CLI planning, ticketing, implementation, and QA without intervention-debt tickets, then exposed Security product-mutation and root scratch-probe gaps.
- Evidence links: `demo-cli-run2` confirmed Security report-only writes and scratch-probe blocking in the live CLI path, then exposed Security false-positive remediation and unbounded Engineer review rework.
- Evidence links: `demo-cli-run3` confirmed patched target guidance, claim-first implementation start, root scratch-probe blocking, and zero target intervention-debt tickets, then exposed initial Engineer drift from the selected ticket and BDD contract.
- Evidence links: `demo-cli-run4` confirmed contract-first implementation honors selected ticket semantics and commits product code, then exposed post-success packaging/build-output exploration before ticket closure.
- Evidence links: `demo-cli-run5` confirmed closure-before-packaging reaches QA, bounded Security, and Dogfood without repo-local packaging artifacts, then exposed duplicate uncommitted Dogfood findings and max-turn failure before handoff.
- Evidence links: `demo-cli-run9` confirmed QA validation gates catch missing tests and route rework without target intervention-debt amplification, then exposed review rework against a done ticket.
- Evidence links: `demo-cli-run10` confirmed review-ticket reopening enforcement in the live CLI path, then exposed ordinary completion commits being misclassified as hidden rework when a ticket is actively moving from in-progress to done.
- Evidence links: `demo-cli-run12` confirmed Engineer reaches QA with a completed product ticket, then exposed QA's missing validation shell surface and stale `<validation-root>` evidence risk.
- Evidence links: `demo-cli-run13` confirmed product-first CLI planning, ticketing, and product code commit, then exposed direct runtime probes not being counted as validation evidence for convergence.
- Evidence links: `demo-cli-run14` confirmed direct runtime validation lets Engineer close the ticket and reach QA, then exposed expected non-zero runtime probes poisoning QA approval despite passing tests.
- Evidence links: `demo-cli-run15` confirmed the product path still reaches committed implementation and external runtime proof, then exposed post-validation shell retries instead of non-shell ticket evidence updates.
- Evidence links: `demo-cli-run16` confirmed Engineer's non-shell ticket-evidence recovery reaches QA/Security, then exposed unexpected runtime validation failure escaping QA because expected negative-path probes were not explicit.
- Evidence links: `go test ./internal/tools -run TestDogfoodUncommittedFindingBlocksFurtherValidationAndTickets` covers the new Dogfood finding commit/handoff policy.
- Evidence links: `go test ./internal/tools -run 'TestEngineerPostValidationGateAllowsValidationWhileImplementationDirty|TestEngineerMustReopenDoneTicketBeforeProductMutation|TestEngineerCompletionCommitAllowsTicketMoveToDoneWithProductFiles'` covers post-validation convergence, review-ticket reopening, and the completion-commit exception.
- Evidence links: `go test ./internal/tools -run 'TestExternalValidationArtifactMustBeBuiltInSameSession|TestEngineerPostValidationAllowsFreshExternalValidationArtifact|TestRecordSessionToolOutcomeTracksValidationArtifactBuildAndRun'` covers same-session external validation artifact freshness and rework proof.
- Evidence links: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksRuntimeValidationCommands|TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion'` covers direct runtime validation classification and post-commit no-op redirection.
- Evidence links: `go test ./internal/tools -run TestReviewApprovalRequiresPassingValidationWhenTestsExist` covers expected runtime error probes remaining review evidence while failed build/test commands still block approval.
- Evidence links: `go test ./internal/tools -run 'TestEngineerPostValidationCommitBlocksExploratoryShellUntilTicketDone|TestEngineerPostRuntimeValidationNoopRedirectsToTicketCompletion'` covers the post-validation non-shell evidence guidance before ticket lifecycle completion.
- Evidence links: `demo-inventory-api-run65` confirmed the generic HTTP API lifecycle reaches product rework and runtime validation, then exposed post-validation no-op placeholders before dirty rework commit; `go test ./internal/tools -run TestEngineerPostValidationDirtyNoopBlocksBeforeGenericNoop` covers the immediate convergence guard.
- Evidence links: `go test ./internal/tools -run 'TestRecordSessionToolOutcomeTracksExpectedRuntimeFailure|TestReviewApprovalRequiresPassingValidationWhenTestsExist|TestEngineerMustReopenDoneTicketBeforeRework'` covers explicit expected exit codes, unexpected runtime validation approval blocks, and external temp validation cleanup.
- Verified by: partial; live baseline export and before/after replay evidence remain open.

## Acceptance Criteria

### Functional (happy path)
- [x] Current factory pace is measured from durable evidence, with a dated baseline documented before optimization work starts. (docs/validation/baselines/2026-06-12-factory-pace-baseline.md, from the live balanced-model demo-11 replay on v0.50.2 with model identity recorded per AD-285; the 2026-06-11 heavy-model replay remains evidence-only.)
- [x] Pace is represented as a first-class metric with role/repo/job attribution rather than as ad hoc notes in chat. (Factory Pace and Convergence And Guardrails sections rendered from the live demo-11 DB on 2026-06-11.)
- [x] The implementation surfaces pace where operators already inspect factory health, such as scores export, trace summaries, dashboard, or CLI output. (`mars-harness scores export` live-validated against `~/.mars-harness/db/demo-11/mars.db`.)
- [ ] At least one evidence-backed resolution reduces avoidable turns or improves successful completion before max-turn failure.
- [ ] Max-turn and budget behavior is calibrated by role or work type so higher limits are available for productive work without allowing silent loops.

### Edge cases and negative paths
- [ ] Sparse or missing trace data reports insufficient pace evidence instead of a healthy default.
- [ ] Long but productive runs are distinguished from circular runs using terminal outcomes, tool diversity, diff/evidence progress, or other auditable signals.
- [ ] Pace scoring does not reward rushed incomplete work or penalize necessary validation on complex tickets.
- [ ] Max-turn failures create actionable evidence or intervention debt with the observed turn sink.

### Non-goals
- Blindly raising every role max-turn limit without pace evidence.
- Treating wall-clock speed as more important than correct, tested, documented outcomes.
- Replacing accuracy/value scores; pace should complement them.

### Observability, docs, and regressions
- [ ] Feature contracts document any new pace metric, score effect, runtime limit rule, or user-visible output.
- [ ] Design docs explain the metric formula, thresholds, and how it avoids gaming.
- [ ] Tests cover metric computation, empty evidence, max-turn classification, and any CLI/dashboard/export output changes.
- [ ] A final evidence note records baseline pace, changes made, post-change pace, residual bottlenecks, and follow-up tickets if optimum is not reached.

## Notes

Treat optimum as an evidence-backed operating target, not a fixed magic number. The first implementation slice should make pace visible and identify the biggest bottleneck before changing runtime limits.

## 2026-05-20 Progress

The first implementation slice makes pace visible without changing runtime
limits. `mars-harness scores export` now joins terminal scoring outcomes to
trace summaries by `job_id` and renders `docs/QUALITY_SCORE.md` Factory Pace
rows grouped by repo and role. The rows include job count, average turns,
average tool invocations, average LLM calls, average wall time, limit-stop
count, and a pace signal. Missing trace rows render as missing pace evidence
instead of implying healthy speed.

Remaining work:

- Run quality export against a live repo DB with recent traces and record the
  dated baseline in the active plan or validation report. Done for
  `demo-123-run12` and the non-static `demo-api-run1` API canary.
- The non-static canary identified same-repo same-role scheduled duplication as
  the first generic optimization target. Scheduler fire now checks the queue for
  active same-role work before enqueueing another scheduled job.
- The scheduler-skip rerun identified repo-local compiled binaries as the next
  generic containment trap. `go build .` can create an untracked root binary
  named after the repo; shell cleanup now permits removing that exact generated
  artifact while preserving ordinary deletion blocks.
- The first rerun after the cleanup fix identified a prior planning ambiguity:
  CEO and COO still tried duplicate `F-001` paths or duplicate starter scenario
  IDs before CTO ticketing. Generated bootstrap guidance now instructs CEO to
  hand off the existing feature-contract path and COO to rewrite the starter
  contract in place.
- The rerun after canonical planning guidance confirmed CEO, COO, and CTO now
  reach Engineer on the Task Notes API path, but Engineer generated a root
  binary named after the Go module (`task-notes-api`) rather than the repo
  directory. The cleanup exception now also allows untracked, binary-looking
  root artifacts named after the root `go.mod` module basename.
- The rerun after module-named cleanup confirmed the exception is not
  discoverable enough from generic blast-radius wording. Engineer never tried
  `rm task-notes-api`, so blast-radius errors now append the exact cleanup
  command when the oversized file is an untracked, root-level, binary-looking
  repo/module artifact.
- The rerun after cleanup hints confirmed Engineer can discover and run
  `rm task-notes-api`, then exposed long-running API validation as the next
  generic turn sink. Foreground `go run` spent a timeout, and shell `&`
  process management left port `8080` occupied. `shell_exec` now rejects shell
  background operators, treats early `background:true` exits as boot failures,
  and generated Engineer guidance forbids `cmd & PID=$!` snippets.
- The rerun after managed-background hardening confirmed Engineer can validate
  the API with `background:true`, probe `/health`, and kill the managed PID.
  The next turn sink is now validation build-output prevention: `go build -o`
  inside the target repo created a root binary trap. `shell_exec` blocks that
  output path before execution and keeps malformed shell payload errors visible
  even when an existing artifact has already made the repo dirty.
- The rerun after build-output prevention confirmed no repo-root binary is
  created and the target can stay clean with committed product progress. The
  next turn sink is now malformed port-token recovery: bare `:8080` commands
  repeat after validation errors. `shell_exec` now rejects bare port tokens in
  argv and single-token shell_command mode before process execution and points
  roles back to real server commands plus curl probes.
- The rerun after bare-port rejection confirmed Engineer no longer loops on
  `:8080`, the full API lifecycle reaches local release notes, and
  intervention-debt remains at zero. The next turn sink is accidental scratch
  validation surface: Engineer committed a root `validate.sh` that depended on
  non-portable `timeout`. `file_write` now blocks new root validation scripts,
  and `shell_exec` rejects `timeout`/`gtimeout` in favor of tool
  `timeout_seconds` or managed `background:true` probes.
- The rerun after scratch-validation prevention exposed a lower-level cleanup
  leak before it could reach Dogfood: a prior `go run` background validation
  left its compiled child server on port `8080`. `shell_exec` cleanup now kills
  known descendants before the tracked process group so wrapper commands do not
  leak dev servers across jobs or canaries.
- The rerun after tracked-background kill interception confirmed same-job
  cleanup works: the external `/tmp` validation binary started after the role
  killed the tracked `go run` PID. The next turn sink is no-op shell calls after
  successful validation. `shell_exec` now returns completion guidance for empty
  `argv` and single `:` calls, names active background PIDs, and generated
  Engineer guidance forbids using no-op calls as wait commands.
- The rerun after no-op shell guidance confirmed Engineer completes the product
  ticket lifecycle, including background process cleanup and a terminal
  disposition. The next turn sink is no longer implementation liveness; it is
  quality gate integrity. DocSync findings were observed by QA/Security/Dogfood
  and still approved, while a policy-blocked external `timeout` command was
  misclassified as retryable. Successful implementation/review/validation/
  release dispositions now run DocSync mechanically, and tool-policy timeout
  blocks classify as guardrail evidence.
- Subsequent API replays confirmed source-write DocSync preflight, claim
  normalization, claim-first shell execution, managed background validation,
  bounded Security review, Dogfood validation, target release notes, quality
  grade `A`, and zero target intervention-debt tickets.
- The first CLI matrix replay (`demo-cli-run1`) confirmed product-first
  planning, ticketing, implementation, and QA for a command-line tool, but
  exposed root scratch-probe pollution (`debug.go`) and Security patching
  product code after finding a failing test. Security now writes only security
  reports and routes remediation to Engineer; root scratch probes are blocked
  before creation.
- The second CLI matrix replay (`demo-cli-run2`) confirmed the Security
  authority boundary and root scratch-probe block in a live target. It produced
  target quality grade `B`, one done ticket, and zero intervention debt. The
  next turn sink is review accuracy and bounded rework: Security converted an
  already-safe/speculative CLI concern into `changes_requested`, and Engineer
  kept validating after a small committed patch until `max_turns`. Generated
  Security guidance now grounds remediation in current evidence, and generated
  Engineer guidance now stops review rework after the exact requested proof
  passes.
- The third CLI matrix replay (`demo-cli-run3`) confirmed patched target
  guidance, claim-first implementation start, and root scratch-probe blocking,
  with runtime failure containment and zero target intervention debt. It then
  exposed initial Engineer contract drift: the selected ticket required empty
  text to produce zero counts, but Engineer rewrote tests around a one-line
  empty-text interpretation and hit `max_turns`. Generated Engineer guidance
  now treats the selected ticket and BDD feature contract as the product
  contract before product writes.
- The fourth CLI matrix replay (`demo-cli-run4`) confirmed contract-first
  implementation fixed the empty-text semantic drift and produced a committed
  product slice. It then exposed post-success closure drift: Engineer continued
  into repo-local packaging/build-output exploration, created an untracked
  `bin/` artifact through a shell-wrapped `go build`, and hit `max_turns`
  before moving `T-001` to done. Generated Engineer guidance now closes the
  selected ticket before packaging/distribution exploration, and shell policy
  scans shell command segments for repo-local Go build outputs before execution.
- The fifth CLI matrix replay (`demo-cli-run5`) confirmed closure-before-
  packaging reaches QA, bounded Security, and Dogfood. It then exposed Dogfood
  finding-handoff drift: Dogfood created duplicate uncommitted tickets for the
  same missing-argument behavior and hit `max_turns` before committing or
  recording disposition. Tool policy now blocks further Dogfood shell validation
  and additional ticket creation while a finding ticket is uncommitted.
- The sixth CLI matrix replay (`demo-cli-run6`) confirmed Dogfood handoff now
  creates one committed finding and routes rework without intervention-debt
  amplification. It then exposed review approval without live validation
  evidence, Dogfood post-commit validation before disposition, literal-newline
  argv overblocking, and enabler ticket closure metadata churn. Tool policy now
  records shell validation outcomes, blocks QA/Security approval without
  passing evidence, freezes Dogfood after finding creation until disposition,
  allows literal newline argv arguments, and lets evidenced enabler tickets
  close without feature metadata.
- The seventh CLI matrix replay (`demo-cli-run7`) restarted from a fresh
  generic Note Stats target. CEO and COO stayed product-specific and committed
  goal/plan/feature updates with zero target intervention-debt tickets, but CTO
  stalled because the feature contract contained `F-002-SNNN` headings inside
  `docs/features/F-001-product-walking-skeleton.md`. Tool policy now rejects
  scenario headings whose feature ID does not match the contract path, and
  generated COO/CTO guidance explains the recovery path.
- The eighth CLI matrix replay (`demo-cli-run8`) confirmed scenario/file ID
  alignment reaches product ticket creation and a committed working CLI. It then
  exposed the next generic turn sink: after successful validation and an
  implementation commit, Engineer kept running exploratory shell probes instead
  of moving `T-001` to done and recording the QA handoff, eventually causing
  `context_overflow`. Tool policy now blocks post-validation exploratory
  `shell_exec` calls while an ordinary product ticket remains in progress, and
  context pruning now removes old assistant tool-call arguments in addition to
  old tool results.
- Later CLI replays through `demo-cli-run17` confirmed the loop is now measuring
  review-quality and handoff pace rather than bootstrap liveness. Runs 14-16
  reached QA after product-specific planning, ticketing, implementation,
  evidence updates, and ticket closure. Run 17 corrected the empty-text CLI
  behavior but found a generic review sink: QA ran a failing `go test ./...` at
  the turn-budget edge and ended as `max_turns` instead of structured
  `changes_requested`; Engineer also used `<validation-root>` instead of the
  freshness-tracked `<validation-root>` path. Tool policy now stops
  QA/Security shell validation after the first failing build/test/unexpected
  runtime command, gives dispatch jobs one final terminal-tool grace prompt at
  the turn-budget edge, and blocks external Go validation builds that do not use
  `<validation-root>`.
- `demo-cli-run18` confirmed the structured review handoff: QA recorded
  `changes_requested` after failed validation, Orchestrator routed back to
  Engineer, Engineer reopened the done ticket before rework, and same-session
  `<validation-root>` freshness blocked stale binaries. It exposed the next
  generic review-procedure edge: an expected-negative missing-argument probe run
  without `expected_exit_code` could not be corrected because the shell-stop
  rule blocked the exact rerun. AD-173 now allows one exact matching
  `expected_exit_code` correction for runtime probes while preserving hard
  failure handling for builds, tests, and uncorrected runtime failures.
- `demo-cli-run19` confirmed product-specific bootstrap and implementation
  still work on a fresh non-game Note Stats target, with no target
  intervention-debt tickets. It exposed the next evidence-integrity issue:
  Engineer observed an empty-text runtime failure, then marked the ticket
  complete and moved it to done anyway. The same run showed that the terminal
  grace turn could still execute non-terminal lifecycle cleanup before ending
  as `max_turns`. AD-174 now keeps unexpected runtime validation failures
  outstanding until the exact command passes or is corrected with matching
  `expected_exit_code`, blocks Engineer ticket completion while outstanding
  failures remain, and allows the budget-edge grace turn to execute only the
  configured terminal disposition tool.
- `demo-cli-run20` confirmed AD-174 blocks the bad completion path: Engineer
  observed the empty-text runtime failure and policy rejected the attempted
  ticket move to `docs/tickets/done/`. The target stayed with `T-001` in
  progress and no intervention-debt tickets were created. It exposed a narrower
  bypass loop: Engineer retried the failed acceptance path with
  `expected_exit_code: 1` until `circle_detected`. AD-175 makes the
  retroactive expected-exit correction review-only for QA/Security; Engineer
  must make a previously unexpected runtime command pass before completion.
- `demo-cli-run21` confirmed the expected-exit bypass is closed and target
  intervention-debt remains quarantined, then exposed a repeat-failure loop:
  Engineer reran runtime probes after the empty-text acceptance command failed
  instead of editing code. AD-176 now blocks further Engineer runtime probes
  until a post-failure implementation edit happens, then requires the exact
  failed command to pass.
- `demo-cli-run22` confirmed AD-176 changed live behavior: Engineer edited
  after the blocked empty-text rerun and made the exact command pass. It then
  exposed a missing-argument negative-path gap because Engineer omitted
  `expected_exit_code` on `<validation-root>`; AD-177 allows that
  exact obvious missing-argument probe to be corrected while keeping positive
  acceptance failures strict.
- `demo-cli-run23` confirmed the missing-argument correction path, then exposed
  a process-status blind spot: `<validation-root> --text ""` exited
  zero while printing `error:` and usage text to stderr. AD-178 treats
  error-shaped stderr from direct runtime validation as failed evidence until
  the exact command passes cleanly.
- `demo-cli-run24` confirmed product-first planning, feature contract, ticket
  creation, and intervention-debt quarantine, then exposed a pre-implementation
  Engineer no-op loop after ticket claim. AD-179 blocks repeated no-op shell
  calls in that phase and routes the role to reading the ticket/feature plus
  `file_write` implementation or a blocked disposition.
- `demo-cli-run25` confirmed the claimed-ticket no-op fix: Engineer wrote
  product files and repaired the empty-text positive path. It then exposed
  unclear unresolved-runtime guidance for a missing-argument negative probe
  first run without `expected_exit_code`; AD-180 names the exact
  `expected_exit_code` correction in blocker messages.
- `demo-cli-run26` confirmed CEO/COO product-first bootstrap still works, then
  exposed false ticket progress: CTO repeated a malformed `ticket_create`
  payload, tried to bypass it with direct ticket `file_write`, and recorded
  completion without a ticket. AD-181 keeps ticket-creation failures
  unresolved until a later successful `ticket_create` and blocks successful
  dispositions that would hand off non-existent backlog work.
- `demo-cli-run27` confirmed the ticket-creation fix: CTO created a valid
  product ticket and Engineer began implementation. It exposed that
  missing-argument runtime probes still need one exact next action after
  `expected_exit_code` is omitted; AD-182 now stores the exact correction and
  blocks unrelated Engineer mutations until it runs or the role records a
  blocked disposition.
- `demo-cli-run28` confirmed the path reached Engineer and product-specific Go
  source again, then exposed stale external validation artifact reuse after an
  empty-text acceptance failure. AD-183 now requires rebuilding
  `<validation-root>` artifacts after post-failure implementation edits before
  rerun evidence is trusted.
- `demo-cli-run29` confirmed fresh product-specific planning, feature-contract
  update, ticket creation, and Engineer implementation work still occur without
  target intervention-debt tickets. It exposed ticket evidence outrunning
  validation: Engineer wrote in-progress `evidence_links` and `verified_by`
  before any successful validation command. AD-184 now blocks Engineer
  in-progress ticket evidence writes until the same job has successful
  validation.
- `demo-cli-run30` confirmed AD-184: Engineer validated with `go test`,
  `<validation-root> --text ""`, `<validation-root> --text
  "hello world"`, and docsync before updating evidence and moving `T-001` to
  done. QA then stalled after the same-session `<validation-root>` freshness
  guard instead of rebuilding the binary in its own job. AD-185 now returns an
  exact `shell_exec argv` rebuild correction for stale external artifact
  blocks and mirrors that recovery path into QA/Security guidance.
- `demo-cli-run31` confirmed the product-first path through a real
  implementation and QA handoff, and confirmed missing-argument runtime
  correction state works live. It exposed two validation-quality gaps:
  Engineer accepted exit-zero runtime output even though empty text returned
  `lines:1` instead of the contracted `lines:0`, and QA guessed the wrong
  package target after a repo-local `go build ./cmd/note-stats` guardrail
  block. AD-186 now emits exact corrected `shell_exec argv` build commands
  that preserve package targets, and generated role guidance requires tests
  for exact expected-output examples. QA approval is also mechanically blocked
  for Go source changes when no `_test.go` files exist.
- `demo-cli-run32` confirmed the path reaches a committed product ticket and
  Engineer-written Go tests, and confirmed exact build-output correction is
  usable. It exposed a deterministic missing-input correction loop: once the
  exact `expected_exit_code` repro still panicked, policy kept blocking
  `file_write` and sent Engineer back to the same failing command. It also
  showed target/foundation naming drift through `cmd/mars-harness` and
  `module mars-harness` inside a Note Stats CLI target. AD-187 now allows
  implementation edits after a failed missing-input correction attempt while
  completion remains blocked, and generated CTO/Engineer guidance requires
  target-derived command, module, and binary names.
- `demo-cli-run33` confirmed target-derived `cmd/note-stats` and `module
  note-stats` behavior, plus implementation repair after runtime failure. It
  then exposed stale runtime-failure accounting: repeated failures of the same
  exact `--text ""` command left multiple outstanding counters, and one later
  exact success only cleared one. AD-188 now clears all unmatched failures for
  the same command fingerprint when the exact runtime rerun succeeds.
- `demo-cli-run34` confirmed AD-188 in a fresh target: Engineer reached tests,
  docsync, done ticket, and QA without target intervention-debt tickets. It
  exposed the next generic pace issues: QA used shell for package
  initialization, missed first-run `expected_exit_code` on an intentional
  omitted-flag probe, Orchestrator treated ticket README examples as live
  backlog, and product code was bundled into the ticket done move commit.
  AD-189 now makes reviewer shell validation-only, requires implementation
  commits before ticket done moves, and keeps Orchestrator routing tied to live
  lifecycle state.
- `demo-cli-run35` confirmed AD-189's traceability fix: Engineer created a
  separate `feat(cli)` implementation commit before a lifecycle-only ticket
  done commit. QA performed useful review validation, including docsync, fresh
  `<validation-root>` build, happy path, and empty-string probes, then looped
  on empty `shell_exec` placeholders instead of recording the required
  disposition. AD-190 now gives required terminal-tool jobs one circle-grace
  reminder and routes reviewer no-op placeholders after successful validation
  directly to `job_disposition_record`.
- `demo-cli-run36` confirmed product-first planning and ticketing still work
  without target intervention-debt tickets, but exposed that an unresolved
  empty-string acceptance failure could still be bypassed with shell-wrapper
  probes, unrelated validation, ticket evidence edits, and an implementation
  commit. AD-191 now freezes unrelated shell paths and product commits while
  Engineer has an outstanding runtime acceptance failure, while still allowing
  stale `<validation-root>` artifact rebuilds before the exact repaired rerun.
- `demo-temp-run37` used a different Temperature JSON CLI target and confirmed
  product-specific planning, ticket creation, implementation, tests, exact
  omitted-flag `expected_exit_code` correction, product commit, evidence
  update, docsync, and a lifecycle-only done-ticket commit. It then exposed
  stale ticket-creation state: a blocked Engineer pre-validation evidence
  write later prevented an otherwise valid successful disposition. AD-192 now
  ignores Engineer evidence-write failures for ticket-creation debt while
  preserving false-progress blocks for failed `ticket_create` and non-Engineer
  ticket-file bypass attempts.
- `demo-temp-run38` confirmed product-first delivery reaches implementation,
  evidence update, and done-ticket closure on the alternate CLI target. It
  exposed COO alternate ticket-creation attempts and QA no-op shell cycling
  after validation. AD-193 now makes non-ticket-owning planners hand off
  `ticket_breakdown` to CTO, makes review no-op recovery terminal-only, and
  routes Go source with no `_test.go` files to QA `changes_requested`.
- `demo-temp-run39` confirmed AD-193's planning handoff behavior on the
  alternate CLI target and confirmed Engineer attempted durable Go tests. It
  exposed a generic validation-integrity gap: after `go test` failed, runtime
  probes and a product commit still advanced before tests passed. AD-194 now
  blocks runtime side paths, ticket evidence, ticket completion, successful
  disposition, and product commits until source/test repair is followed by the
  exact passing test/build command.
- `demo-temp-run40` confirmed the early planning path again, then exposed CTO
  role-boundary drift: CTO created the ticket but also wrote `go.mod`,
  attempted source/test writes, updated README usage, and committed
  product-adjacent state. AD-195 now limits CTO file writes to technical
  planning artifacts and keeps package, README usage, source, test, build,
  config, and root product-file changes in ticket-backed Engineer delivery.
- `demo-temp-run41` confirmed AD-195 in a fresh alternate CLI target: CTO
  committed only the implementation ticket before Engineer claimed it. It then
  exposed that AD-194 was too exact-command-bound for test/build repair:
  focused same-lane `go test` commands were blocked after a package-pattern
  test failed, pushing the role toward workaround behavior. AD-196 now allows
  bounded source/test/fixture/build-config repair followed by same-lane
  focused validation, while runtime probes, helper scripts, evidence,
  completion, disposition, and commits stay blocked until validation passes.
- `demo-temp-run42` confirmed AD-196 still blocks bypasses, but exposed focused
  shell validation drift: `cd cmd/temperature-json-cli && go test -v .` was a
  legitimate same-lane test repair command and was blocked only because shell
  control syntax was unclassified. AD-197 now recognizes the narrow
  `cd <dir> && <test/build>` validation form while keeping arbitrary shell
  wrappers blocked.
- `demo-temp-run43` confirmed the alternate target can now reach Engineer
  implementation, expected negative runtime correction, test repair, product
  commit, ticket closure, and QA approval. It then exposed Security review
  completion latency: after clean read plus validation evidence, Security spent
  more than five minutes in the next LLM turn instead of recording disposition.
  AD-198 now makes clean QA/Security review evidence terminal-only and caps the
  required `job_disposition_record` grace response.
- `demo-temp-run44` confirmed product-first planning and Engineer delivery
  again, then exposed a false QA rework path: a reviewer command used
  `cmd/temperature-json-cli` instead of `./cmd/temperature-json-cli` for a Go
  build target. AD-199 now classifies obvious QA/Security validation-procedure
  command-target mistakes separately from product validation failures so
  corrected review validation can continue.
- `demo-temp-run45` confirmed planning and ticket creation again, then exposed
  a local-model tool-call formatting trap: Engineer tried focused nested-module
  validation as `argv:["cd","cmd/temperature-json-cli","&&","go","test",
  "./..."]`, which argv mode rejected before a root `go test ./cmd/...`
  command poisoned the repair lane. AD-200 now normalizes only the safe
  validation-only `cd <dir> && <test/build>` argv shape into `shell_command`.
- `demo-temp-run46` confirmed the external validation-binary correction and
  positive runtime evidence, then exposed a narrower missing-input validation
  loop: Engineer received the expected `--celsius flag is required` error from
  a no-argument CLI probe but could not complete because the command omitted
  `expected_exit_code`. AD-201 now treats clear missing-input CLI probes with
  required/usage output and no crash markers as expected negative-path
  evidence immediately.
- `demo-temp-run47` confirmed the missing-input part of AD-201, then exposed
  the sibling invalid-input edge: Engineer received the expected
  `Must be a number` error for `invalid`, but the runtime policy still opened
  an unexpected-failure blocker. AD-201 now includes deliberate invalid-input
  probes while preserving failures for valid positive inputs rejected as
  invalid.
- `demo-temp-run48` confirmed AD-201 through Engineer implementation,
  positive runtime checks, missing-input and invalid-input checks, product
  commit, ticket closure, and QA handoff. QA then requested Go test coverage,
  and the rework Engineer hit a missing `./` package-target procedure failure
  that incorrectly opened the product repair lane. AD-202 now classifies
  obvious Engineer Go validation-procedure mistakes separately so corrected
  validation can continue without a meaningless source edit.
- `demo-temp-run49` confirmed the alternate CLI target through planning,
  ticket creation, Engineer claim, source write with DocSync metadata,
  external validation build, positive runtime validation, and missing-input
  validation. It then exposed that surplus-argument CLI validation was still
  too narrow: `<validation-root> 25 30` correctly returned
  `error: too many arguments provided`, but the guardrail opened an unexpected
  runtime failure. AD-203 now treats clear surplus-argument CLI probes as
  expected negative-path evidence when output names too many or surplus
  arguments and no crash markers are present.
- `demo-temp-run50` confirmed another product-first path through Engineer
  implementation, external validation, positive runtime checks, and explicit
  expected-exit negative checks. It then exposed that the test/build repair
  lane blocks the only available deletion path for bad same-job test files:
  after duplicate test helpers caused a real `go test` compile failure,
  `rm cmd/..._test.go` stayed blocked and the role created more duplicate
  files. AD-204 now allows non-recursive removal of test-like files written by
  the same Engineer job after the test/build failure began, while source
  files, unmarked tests, and recursive cleanup remain blocked.
- `demo-temp-run51` confirmed full product-first delivery through Engineer:
  plan, ticket, implementation, tests, runtime evidence, ticket closure, and
  QA handoff. QA corrected the familiar Go package-target procedure mistake
  and then had sufficient review evidence, but missed the required terminal
  `job_disposition_record` call and ended as `circle_detected`. AD-205 now
  rejects the first non-terminal response after clean review-evidence guidance
  without executing it and gives one stronger terminal-only correction before
  repeated misses fail.
- `demo-temp-run52` exercised a different repair path. After a failing
  `go test ./cmd/temperature-json-cli/...` command, the test/build repair lane
  blocked runtime probes, destructive cleanup, commits, and ticket moves, but
  still allowed Engineer to create root `main.go` and `main_test.go`, causing
  validation to drift toward a parallel implementation. AD-206 now records
  narrow Go package repair scopes and blocks source/test/fixture writes
  outside that scope until the lane is repaired.
- `demo-temp-run53` confirmed AD-206 in a clean live target: Engineer repaired
  the failing `cmd/temperature-json-cli` package test in scope, QA approved,
  and the lifecycle reached Security, Dogfood, and Release Manager. Release
  Manager generated local release notes but tagged `v0.2.0` before committing
  the release-note files, leaving the tag on the pre-release-note Dogfood
  commit after the dirty-worktree disposition guard forced `release: notes
  0.2.0`. AD-207 now blocks release tag creation until `HEAD` is the clean
  release-note commit and makes `git_release_guard` report stale local version
  tags.
- `demo-temp-run54` validated AD-207: Release Manager generated notes,
  committed `release: notes 0.2.0`, tagged `v0.2.0` at that release-note
  commit, and recorded a clean release blocker only when the throwaway target
  lacked `origin`. The same run reproduced temporal evidence drift: Dogfood
  wrote a `2024-05-21` report path during a 2026-05-21 run after shell `date`
  was unavailable. AD-208 now injects non-droppable run metadata into server
  job context so dated evidence uses the real execution date.
- `demo-temp-run55` confirmed run metadata reaches live server roles and
  product planning still precedes target intervention debt. Engineer created
  real Temperature JSON CLI product files, but duplicate/placeholder tests
  caused `go test ./cmd/temperature-json-cli -run TestTemperatureCLI` to fail.
  The repair lane correctly blocked runtime probes, commits, ticket evidence,
  and false completion, but `rm -f cmd/temperature-json-cli/main_test.go`
  stayed blocked because that test file was written before the failing command.
  AD-209 now records every successful Engineer `file_write` path so
  non-recursive cleanup can remove same-job duplicate generated tests during
  repair while protecting pre-existing tests and source files.
- `demo-temp-run56` validated that the lifecycle now moves past the prior
  Engineer blocker: product implementation, QA-requested test rework, focused
  Go tests, ticket completion, and second QA handoff all occurred. The next
  blocker is review terminal convergence firing before QA can run required
  `docsync_audit`; after a successful `go test ./cmd/temperature-json-cli/`,
  the attempted docsync audit was rejected and QA ended with
  `circle_detected`. AD-210 now waits for docsync evidence before forcing the
  terminal `job_disposition_record` boundary.
- `demo-temp-run57` validated AD-210 by allowing QA to run `docsync_audit`,
  but exposed build-only terminal convergence: after an external
  `go build -o <validation-root>
  ./cmd/temperature-json-cli`, the runtime forced terminal disposition before
  QA ran tests despite `_test.go` files being present. AD-211 now waits for a
  successful review test command when tests exist.
- `demo-temp-run58` validated the direct AD-211 convergence check but exposed
  no-op recovery as a stale terminal trigger: a blocked empty review
  `shell_exec` after build evidence still forced approval guidance before
  tests had passed. AD-212 now keeps no-op recovery aligned with the same
  evidence gates as approval and points reviewers to missing tests or docsync
  before terminal approval guidance.
- `demo-temp-run59` validated the review recovery fixes through a full
  product-first target lifecycle: product planning, ordinary ticketing,
  Engineer implementation, QA approval, Security approval, Dogfood approval,
  local `release: notes 0.2.0`, and tag `v0.2.0`. The next bottleneck was
  Release Manager using `shell_exec mars-harness release notes`, which resolved
  a stale installed binary and triggered a liveness retry before a second
  release pass recovered. AD-213 now blocks direct `mars-harness` shell
  invocations in agent jobs and routes those workflows through
  `mars_harness_cli`.
- `demo-temp-run60` broadened the canary matrix with a Word Count JSON CLI and
  validated AD-213 in the live release path: Release Manager used
  `mars_harness_cli`, committed `release: notes 0.2.0`, tagged `v0.2.0`, and
  stopped only on the real missing-remote publication blocker. The new
  bottleneck is retry persistence: a bind-failed start wrote repo/CEO bootstrap
  state into SQLite WAL files, then automatic cleanup deleted the sidecars
  before retry. AD-214 now preserves SQLite sidecars and lets SQLite recover or
  checkpoint them instead of deleting queue or repo registry state.
- `demo-slug-run61` validated AD-214 in a Slugify JSON CLI target: retry after
  bind failure reused the same repo ID and CEO bootstrap job without deleting
  SQLite sidecars. The next bottleneck is test-rework repair guidance. QA
  correctly requested missing-test rework, Orchestrator routed it to Engineer,
  and Engineer added failing tests that exposed a product mismatch, but
  guardrails repeated only the unresolved command while the role churned for
  9m44s. AD-215 now repeats the latest failing test/build output and tells
  Engineer to edit implementation when the failing assertion matches the
  contract.
- `demo-slug-run62` validated AD-215 against a fresh Slugify JSON CLI target:
  the lifecycle completed through CEO, COO, CTO-weekly, Engineer, QA,
  Security, Dogfood, and Release Manager; produced product-specific planning,
  one ordinary product ticket, product code, Go tests, release notes, and tag
  `v0.2.0`; kept guardrail blocks as telemetry; and created no target
  intervention-debt tickets. The only terminal blocker was expected missing
  remote publication in the temporary target.
- `demo-notes-api-run63` broadened the matrix with a Go HTTP JSON API and a
  local bare `origin`. CEO, COO, and CTO-weekly again produced product-first
  planning and one ordinary ticket, Engineer claimed and pushed the ticket, and
  intervention signals stayed foundation telemetry. The next blocker was a
  bootstrap repair trap: after `go test ./internal/note` failed because no
  `go.mod` existed, the repair lane blocked the direct `go mod init` fix and
  the role drifted into test deletion and placeholder attempts. AD-216 now
  allows `go mod init` only when missing-module output and an absent `go.mod`
  make it the direct package-config repair.
- `demo-notes-api-run64` avoided the module trap by writing `go.mod` early,
  but exposed two generic evidence holes: raw `go get` was not classified as
  dependency mutation, and same-job test cleanup could delete assertion
  evidence after a focused test failure. AD-217 now blocks raw `go get` with
  dependency-sync guidance and limits same-job test cleanup to
  duplicate/generated-test shaped failures.
- Define calibrated thresholds after the baseline, not before it.
- Broaden the canary matrix to remote-backed release validation and non-CLI
  application shapes before claiming the factory loop is generic rather than
  optimized around compact command-line products.
- Rerun the CLI canary to confirm unresolved runtime failures block ticket
  completion, cannot be bypassed by Engineer retroactive expected-exit calls,
  cause implementation edits before runtime probes repeat, and allow missing
  required-argument probes to be corrected with exact `expected_exit_code`,
  treat zero-exit error stderr as failed runtime evidence, turn claimed-ticket
  no-op loops into product implementation, name missing-argument
  `expected_exit_code` correction in blocker text, prevent false successful
  handoff after failed ticket creation, make that missing-argument correction
  the only next mutating action, require fresh external validation artifacts
  after runtime-failure edits, block ticket evidence until validation, make
  reviewer stale-artifact recovery exact, preserve exact Go build package
  targets in guardrail corrections, require automated assertions for expected
  output examples, keep review shell validation-only, require implementation
  commits before ticket lifecycle closure, route Orchestrator from live ticket
  state instead of examples, give reviewers a terminal-disposition off-ramp
  after repeated no-op shell placeholders, block product commits while runtime
  acceptance failures remain unresolved, avoid poisoning Engineer handoff after
  evidence-ordering guardrail recovery, keep planning ticket handoff on CTO,
  force review no-op recovery to terminal disposition, route missing Go tests
  to QA rework, keep CTO product writes behind Engineer, allow same-lane
  test/build repair without helper-script workarounds, classify simple
  `cd <dir> && <test/build>` validation as same-lane repair, allow same-job
  duplicate generated test cleanup, let QA run docsync before terminal
  disposition, require QA test evidence before terminal convergence when test
  files exist, keep review no-op recovery aligned with the same evidence gates,
  keep Release Manager on `mars_harness_cli` instead of stale shell binaries,
  preserve SQLite WAL sidecars across retry-after-bind-failure startup,
  repeat failing test/build output inside unresolved repair guidance,
  then continue through Security, Dogfood, release, and the multi-archetype
  validation matrix in clean targets.
- Continue the representative validation matrix before making broad optimization
  claims.
