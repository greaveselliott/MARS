---
id: T-036
title: Extract browser-framework static-analysis policy domain into policy_browser.go (AD-287 slice 1)
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "foundation-maintainer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Perform the extraction, run gates, land commits, run the demo-12 checkpoint replay, record evidence, close."
source: docs/design-docs/policy-decomposition.md (AD-287) — extraction sequence step 1; foundation improvement plan WS-E
created: 2026-06-12
depends_on: []
---

# T-036: Extract browser-framework static-analysis policy domain into policy_browser.go (AD-287 slice 1)

## Context

AD-287 (docs/design-docs/policy-decomposition.md) decomposes internal/tools/policy.go (7,091 lines / 354 functions at v0.50.15) by ordered same-package file extraction. The extraction sequence names policy_browser.go as slice 1: the browser-framework static-analysis domain is the most self-contained large domain (mostly pure functions over file content and ticket text, minimal ToolState coupling, bounded *Browser*/*Phaser*/*CDN*/*Smoke* name space). This is pure code motion — behavior-preserving by construction. Dispatchers (preToolPolicy/postToolPolicy) stay in policy.go so literal call order (evaluation order) is untouched.

## Requirements

- Move the browser-framework domain (46 functions + the browserFrameworkInfo type) from policy.go into policy_browser.go: checkBrowserFrameworkTicketCreatePolicy, phaserTicketPrescribesCDNRuntime, engineerBrowserFrameworkEvidenceComplete, checkEngineerBrowserPostBuildSmokeOnlyPolicy, engineerPostCommitBrowserValidationAllowed, checkEngineerBrowserFrameworkImplementationShapePolicy, checkEngineerBrowserFrameworkPackageWritePolicy, phaserRuntimeScriptUsesStaticSourceServer, viteConfigImportsPhaserRuntime, viteConfigExternalizesPhaser, checkShellNodeCheckHTMLPolicy, shellExecNodeCheckHTML, shellExecRunsBrowserProductSmokeCommand, repoBrowserFrameworkInfo, browserFrameworkCompletionBlockers, engineerBrowserFrameworkCompletionBlockers, browserProductSmokeCommandGuidance, browserFrameworkTerminalDispositionGuidance, browserFrameworkRequiresProductSmoke, browserFrameworkSourceFindings, javascriptSourcePath, browserFrameworkValidationHelperPath, htmlSourcePath, repoHasPhaserScriptTag, phaserGoModuleFindings, phaserHTMLFindings, phaserSingleFileSourceFindings, frameworkListContains, phaserSceneReferencesIdentifier, jsDefinesOrImportsIdentifier, phaserUnboundSceneHelperFindings, phaserMissingImportFindings, phaserSceneContextFindings, phaserGameConstructionFindings, phaserNewGameInsideFunction, jsMatchingBrace, jsLocalNamedImportFindings, jsMissingLocalExportImportFindings, jsUsesIdentifier, jsNamedImportNames, resolveLocalJSModuleRel, jsExportedNames, htmlClassicScriptModuleFindings, resolveHTMLScriptRel, jsContainsModuleSyntax, checkEngineerBrowserFrameworkTicketDoneMovePolicy.
- Move the matching browser test blocks from policy_ticket_test.go into policy_browser_test.go in the SAME commit.
- MarsDocSync block on both new files listing only browser-domain docs plus the internal/tools docsync minimum (code-documentation-map.md, tools-glossary.md, F-005) plus guardrails.md, delivery-operating-model.md, F-007.
- No renames, no signature changes, no logic edits, no call-site reordering in dispatchers.

## Borderline functions deliberately NOT moved (record per AD-287)

- projectBriefMentionsFramework / projectBriefNamesGoBackend: caller majority is browser checks, but AD-287 assigns brief interpretation to policy_capability.go (slice 4); moving them now would pre-empt that seam.
- shellExecRunsHTTPProbe: shared with reviewer/validation shell checks — stays.
- testFilePath: called by repoHasTestFiles (validation domain) — stays.
- shellExecMovesInProgressTicketToDone / shellExecInProgressToDoneTicketID: ticket-lifecycle domain with callers across validation checks and executor.go — stays for slice covering policy_ticket.go.
- browserProductSmokeSuccessKey constant: shared ToolState key constants stay in policy.go per AD-287.

## Acceptance criteria

- go build ./... green; full go test ./internal/tools/... green (behavior-preservation oracle); make check passes (coverage ratchet for internal/tools does not regress); docsync audit and docsconsistency green.
- git diff shows only moves plus the new file headers (verify with git diff --color-moved=dimmed-zebra).
- One refactor(tools) semantic commit + release-note commit; trunk fast-forwarded, tagged, assets published and verified per the v0.50.13-15 pattern.
- AD-287 slice-1 checkpoint replay: demo-12 (package-managed frontend canary, AD-284 tool-policy row) reset to clean seed, full lifecycle on the patched binary, model identity recorded per AD-285. Expectation: zero rule-level behavior drift vs the v0.50.11 demo-12 Run 2 baseline (guardrail_block ~65, 4 max_turns, T-001 closed + T-002 in progress); lifecycle reach may legitimately exceed baseline due to AD-289 convergence retry routing which postdates the baseline. New/different guardrail_block patterns or policy errors = extraction drift: stop, investigate, fix or revert.
- AD-285 validation report appended to docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md or a new dated report; ticket closed with evidence links.
