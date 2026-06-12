---
id: T-039
title: Move capability and brief-parsing helpers out of the policy monolith (AD-287 step 4)
priority: high
complexity: small
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/policy-decomposition.md"]
verified_by: "go build ./..., full go test ./internal/tools, make check (coverage ratchet), docsync audit, docsconsistency, line-multiset pure-motion verification; rides the AD-287 checkpoint policy (intermediate slice, test suite is the oracle)"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: landed in the same dispatch as T-038. policy_capability.go (25 functions, 627 lines); policy.go 5,005 -> 4,395 lines; two pure capability unit tests moved to policy_capability_test.go (policy_ticket_test.go 5,822 -> 5,790 lines). Pure motion verified by line-multiset equality (zero removed lines; additions are exactly the two new-file headers). go build, full go test ./internal/tools, make check, docsync audit (0 findings), and docsconsistency all green. No dedicated replay per AD-287 checkpoint policy (intermediate slice)."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Next AD-287 slice: policy_ticket.go (sequence step 5) in a later dedicated dispatch."
dedupe_key: "public-example"
source: docs/design-docs/policy-decomposition.md (AD-287) extraction sequence step 4; foundation improvement plan WS-E
created: 2026-06-12
depends_on: [T-038]
---

# T-039: Move capability and brief-parsing helpers out of the policy monolith (AD-287 step 4)

## Context

AD-287 extraction sequence step 4: the capability/brief-parsing domain (projectBriefCapabilityPhrases and the capability surface/keyword family, plus the brief-interpretation helpers deferred from the T-036 browser slice: projectBriefMentionsFramework, projectBriefNamesGoBackend) moves to policy_capability.go as pure same-package code motion. This is the AD-286 interpretation layer; it resolves the fuzzy feature-contract boundary before the ticket extraction needs it.

## Requirements

- policy_capability.go with MarsDocSync header (guardrails.md + F-007 + internal/tools docsync minimum).
- Matching pure capability unit tests move from policy_ticket_test.go to policy_capability_test.go in the SAME commit.
- Pure motion only: no renames, no signature/logic/comment changes. Line-multiset equality verification like T-036.
- Shared/borderline functions stay in policy.go with deferrals recorded here.

## Functions moved (25)

Brief interpretation: projectBriefMentionsFramework and projectBriefNamesGoBackend (the T-036 browser-slice deferrals; callers are all in policy_browser.go but AD-287 and the T-036 record assign them to the capability interpretation layer), projectBriefHasConcreteProductIntent, projectBriefCapabilityPhrases, projectBriefLabelTokens, stripCapabilityLabelTokens, projectBriefSourceText, splitBriefSentences, splitCapabilitySegment, stripCapabilityCategoryPrefix, cleanCapabilityPhrase, isValidationEvidenceCapabilityPhrase, isValidationEvidenceTailPhrase.

Capability surface family: featureScenarioSurface, featureScenarioOutlineSurface, featureOutOfScopeSurface, outOfScopeSurfaceRequiresDescoping, outOfScopeLineIsExplanation, outOfScopeLineLeavesBasicCapabilityInScope, featureDescopedSurface.

Capability matching/keyword family: capabilityPhraseCovered, normalizeCapabilitySurface, capabilityKeywordSet, capabilityKeywords, capabilityKeyword.

Caller-majority records: projectBriefHasConcreteProductIntent has a single external caller in the planning-disposition check (checkPlanningDispositionFeatureSpecificity) but is pure brief interpretation named by AD-287's capability domain definition; normalizeCapabilitySurface has one out-of-set caller (scenarioLooksProductImplementation, staying) versus eight in-set callers.

Test motion in the SAME commit: TestCapabilityCoverageTreatsPiecesAsTetrominoesForLocking and TestCapabilityMatchingIgnoresIncludingAndDetectionGlue (the two pure capability unit tests) -> policy_capability_test.go.

## Borderline functions deliberately NOT moved (deferrals)

- The coverage/handoff consumers of this layer stay in policy.go for the disposition/ticket slices: checkProductCapabilityScenarioCoverage, productCapabilityCoverageFeatureContents, scenarioCoversProductCapability, scenarioLooksProductImplementation, productScenarioIDsForHandoff, earlyCTOHandoffRequiredScenarios.
- The feature-contract/scenario-coverage helper family (featureIDsFromScenarios, featureScenarioCoverage, featureContractIDs, orderedFeatureScenarioIDs, and friends): caller majority is the ticket_create planning-order gates, so per AD-287's fuzzy-boundary rule they are ticket-lifecycle domain and wait for the policy_ticket.go slice.
- The blended COO-completion tests (TestCOOCompletion*) stay in policy_ticket_test.go: their primary subject is the disposition coverage gate exercised through preToolPolicy, even where they also assert capability parsing directly. Only the two pure unit tests moved.

## Checkpoint decision (record per AD-287)

Intermediate slice between the slice-1 (demo-12 Run 3, PASS) and final-sequence replay checkpoints. Pure same-package motion with byte-identical bodies rides the full test suite; no dedicated replay in this dispatch.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools green; make check passes; docsync audit and docsconsistency green.
- Pure motion verified (line-multiset equality).
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified.
