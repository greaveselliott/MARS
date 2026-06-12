# Policy God-File Decomposition

**Status:** Accepted
**Date:** 2026-06-12
**Author:** foundation-maintainer
**Tickets:** T-030 (WS-E), coordinates with T-028/T-029 (WS-D)

## Context

`internal/tools/policy.go` is the single largest source file in the harness
and the home of nearly every convergence guardrail. Measured shape at
v0.50.8 (verified for this AD):

- `policy.go`: **7,091 lines / 354 functions.**
- `policy_ticket_test.go`: **7,835 lines / 183 test functions** — despite
  the `_ticket_` name, it is the test file for *all* policy domains
  (shell, file-write, ticket, review, release, browser, disposition).

The cost is concrete: every guardrail change pays a whole-file
comprehension tax, unrelated domains churn in the same diff surface,
review and DocSync granularity is one giant blob, and the convergence
state-machine work (AD-286, WS-D) needs per-domain seams to thread
`DeliveryState` consultation through without touching 7,000 lines per
slice.

Structural facts that shape the split:

- **Evaluation order lives in two dispatchers.** `preToolPolicy` and
  `postToolPolicy` call every named check function in a fixed, explicit
  order (cross-cutting checks first, then a per-tool `switch`). The order
  is not registration-time magic; it is literal call order in those two
  functions. A same-package file move of the *called* functions cannot
  change evaluation order as long as the dispatchers stay intact.
- **Session-state keys are the only shared mutable surface.** The
  `validation:*`, `ticket:*`, `review:*`, and `shell:*` `ToolState` key
  constants at the top of `policy.go` are read across domains; they must
  move to a single shared location, not be duplicated per file.

## Decision (AD-287)

### Same-package file splits first; package splits only on measured need

`policy.go` is decomposed by **ordered same-package file extraction**
inside `internal/tools`. No new packages, no exported-surface changes, no
behavior changes in extraction slices. Target file set (names final unless
a slice discovers a better seam; record deviations here):

| File | Domain | Seed size (functions) |
| --- | --- | ---: |
| `policy.go` (remains) | Entry dispatchers (`preToolPolicy`, `postToolPolicy`), trust enforcement, shared `ToolState` key constants, `mutatingTools` | ~3 + constants |
| `policy_shell.go` | Shell safety: argv/shell-command validation, no-op loops, foreground/long-running, build-output, port/timeout, background cleanup | ~57 |
| `policy_validation.go` | Validation lane and repair: test/build failure lanes, runtime validation outcomes, artifact freshness, repair-scope writes, correction guidance | ~78 |
| `policy_ticket.go` | Ticket lifecycle: ticket_create gates, planning order, lifecycle moves, done-evidence, claim-before-mutation | ~62 |
| `policy_review.go` | Review terminal gates: QA/Security approval evidence, terminal-disposition-only, reviewer shell validation | ~19 |
| `policy_release.go` | Release gates: release-tag shell policy, release-note/tag ordering guards | ~5 |
| `policy_browser.go` | Browser-framework static analysis: framework/CDN/HTML/smoke heuristics and closure rules | ~52 |
| `policy_capability.go` | Capability/brief parsing: feature-contract and scenario coverage helpers, brief interpretation (the AD-286 "interpretation layer") | ~25 |
| `policy_disposition.go` | Disposition/handoff: job_disposition_record gates, dogfood finding handoff, docsync/claim checks | ~26 |
| `policy_diff.go` | Diff/secrets/blast-radius: repo-diff validation, secret and oversized-file checks | ~12 |

Seed counts come from a prior function-classification pass; this AD
verified the total shape (354 functions) and approximate domain
proportions by name-pattern sampling. Exact per-function assignment
happens at extraction time per slice; the known fuzzy boundary is
feature-contract/scenario helpers, which sit between ticket-lifecycle
(planning-order gates use them) and capability parsing — assign by
caller majority and record the choice in the slice commit.

Package splits (e.g. `internal/tools/policy/...`) are explicitly **not**
part of this decision. They require exported-surface design and a
measured need (compile time, import cycles, ownership boundaries) that
does not exist yet; if one emerges, it gets its own AD.

### Test splits ride the same seams in the same commits

Each extraction slice moves the corresponding tests out of
`policy_ticket_test.go` into the matching `policy_<domain>_test.go` in
the **same commit**, so the test file shrinks in lockstep and never
becomes the new god-file. At the end of the sequence
`policy_ticket_test.go` contains only ticket-domain tests, making its
name honest.

### MarsDocSync per new file

Every new `policy_<domain>.go` file carries its own top-of-file
`MarsDocSync` block listing only the docs that domain actually
implements (e.g. `policy_release.go` lists `release-versioning.md`, not
the full nine-doc list the monolith carries today). This restores
per-domain DocSync granularity — one of the main payoffs of the split.

### Extraction sequence

Ordered by self-containment and risk, not by size:

1. **`policy_browser.go` first.** The browser-framework static-analysis
   domain is the most self-contained large domain: mostly pure functions
   over file content and ticket text, minimal `ToolState` coupling, and a
   clearly bounded name space (`*Browser*`, `*Phaser*`, `*CDN*`,
   `*Smoke*`). It removes ~50 functions in one low-risk move — the
   highest line-count payoff per unit of risk.
2. **`policy_release.go` + `policy_diff.go`** — tiny, self-contained;
   ride as one slice.
3. **`policy_shell.go`** — large but mechanically identifiable
   (shell-arg parsing and shell-only checks).
4. **`policy_capability.go`** — pure parsing helpers; resolves the fuzzy
   feature-contract boundary before the ticket extraction needs it.
5. **`policy_ticket.go`**, then **`policy_review.go`**, then
   **`policy_disposition.go`**.
6. **`policy_validation.go` last** — the largest and most
   session-state-coupled domain; by this point every neighbor has a
   settled home and the shared key constants have proven stable.

### Per-slice gate

Every slice is a `refactor(tools)` commit with:

- `go build ./...` and `go test ./internal/tools` green (the moved tests
  run from their new file in the same commit);
- the AD-284 **tool-policy** replay minimum — two archetypes (static
  browser app plus one of CLI/tooling or API/service) — once per
  extraction *sequence checkpoint* rather than per mechanical move:
  concretely, after slice 1 (first proof the seam technique is safe),
  and after the final slice. Pure same-package moves with byte-identical
  function bodies between checkpoints rely on the test suite; any slice
  that touches a function body for *any* reason immediately takes the
  full two-archetype replay before the claim. Blocked replays follow
  AD-284: the claim stays unconfirmed with the replay command recorded.

### Coordination with the convergence state machine (AD-286)

Two separate change classes, never mixed in one commit:

1. **Extraction slices (this AD, WS-E):** rules migrate into their domain
   files **as-is**. No signature changes, no `DeliveryState` references,
   no block-message rewording.
2. **State-machine slices (AD-286, WS-D):** once a domain lives in its own
   file, WS-D implementation slices thread `DeliveryState` consultation
   and the block-message contract through that domain as behavior-change
   commits with their own replay gates.

The extraction sequence above is also the natural WS-D enablement order:
`policy_validation.go` landing last matches WS-D's heaviest consumer
(repair-lane transitions) starting after the seams are proven.

## Consequences

- Guardrail changes get domain-scoped diffs, reviews, and DocSync lists.
- The dispatchers in `policy.go` become a readable table of contents for
  the whole policy surface — and the obvious future home for the AD-286
  transition table.
- Ten file moves cost roughly two replay checkpoints instead of ten
  full replays, while any body-touching slice still pays full price.
- Risk: a "pure move" that silently edits a function body. Mitigation:
  slice review compares moved bodies byte-for-byte (`git diff
  --color-moved=dimmed-zebra` makes non-identical moves visible), and
  the checkpoint replays bound the damage window.

## Discoveries

- **2026-06-12 — Slice 8 landed (T-043, `policy_validation.go`) — extraction
  sequence complete:** 82 functions moved (seed estimate was ~78): the
  engineer runtime/test-build validation gate families (unresolved-failure
  blocks before commit/completion/file-write, rework policies, the
  missing-argument correction gate), repair-scope and watermark-key helpers,
  validation artifact freshness/build tracking (the
  `validationArtifactPath`/`shellExecRunsRecordedValidationArtifact` family),
  the validation command classifiers
  (`shellExecRunsTestCommand`/`BuildCommand`/`RuntimeValidationCommand`/`HTTPProbe`
  and the `shellFieldsRun*` cores), script-name helpers
  (`testScriptName`/`buildScriptName`/`runtimeScriptName`),
  `checkEngineerPostValidationCompletionShellPolicy` +
  `engineerDirtyPostValidationGuidance` (T-040 deferral),
  `expectedExitCodeCorrectsUnexpectedValidationFailure` and the
  `shellExecRunsValidationCommandForSession` family (T-041 deferrals), the
  T-038-deferred build-output/path helpers (`goBuildOutputPath` family,
  `pathResolvesInsideRepo`, `shellRemovalPathOperands`), and the
  root-scratch-validation write family (validation-probe hygiene by
  semantics). `sourceFileRequiresDocSync` moved by 3-vs-1 caller majority
  (three validation repair-path callers vs one dispatcher-resident docsync
  write gate). Five more caller-majority reassignments settled in the same
  pure-motion commit: `packageBuildScriptNoop` +
  `packageBuildScriptOnlySyntaxCheck` → `policy_browser.go` (browser-only
  callers); `repoHasTestFiles`, `repoHasGoSourceFiles`, `testFilePath` →
  `policy_review.go` (review-only callers — a recorded deviation from the
  "repo-shape helpers are validation" seed; T-036's validation caller of
  `testFilePath` moved into review-called code in later slices). Slice-4
  motion residue fixed: the `capabilityStopWords`/`capabilityLabelKeepWords`
  vars had only `policy_capability.go` consumers and move there. Test motion:
  30 validation-domain tests (the runtime-failure rework family, the
  failing-test/build repair-lane family, the external-validation-artifact
  family, post-validation completion gates, and the two
  `recordSessionToolOutcome` procedure-failure accounting tests including the
  T-041-deferred `TestReviewHTTPProbeBeforeServerStartIsProcedureFailure`) →
  `policy_validation_test.go`; no mover-only fixtures (all fixtures shared
  with staying ticket tests). **Final assignment resolutions (closing the
  open items from T-036..T-042):** `checkGitPushPolicy` is settled
  dispatcher-resident — its body is pure trunk-branch enforcement with a
  dispatcher-only caller and no domain signal; it stays with the
  trust/dispatch core, recorded here as accepted. The `checkFileWritePolicy`
  sub-dispatcher and its role/artifact write children
  (`checkPlannerFileWritePolicy`, `checkSecurityFileWritePolicy`,
  `checkDogfoodFileWritePolicy`, `checkSourceFileDocSyncWritePolicy`, the
  feature-contract write family `checkFeatureFileWritePolicy`/
  `checkFeatureScenarioIDPolicy` + helpers, and their path predicates) are
  recorded as a **dispatcher-resident file-write surface**: every child's
  only caller is the sub-dispatcher, the children gate role/artifact write
  permissions at the `file_write` boundary rather than any one extracted
  domain, and no extracted file has caller majority. Their tests remain in
  `policy_ticket_test.go` as accepted residue alongside the ticket-domain
  tests that share the same fixtures. Permanent `policy.go` residents
  confirmed: dispatchers + `enforceTrust` + `planningRoleCannotMutateWithShell`,
  `mutatingTools`, the shared `ToolState` key constants (including the two
  executor-shared edit-watermark keys), the shared tokenizer family
  (`shellFields`, `shellFieldsPreserveCase`, `normalizedShellExecFields` +
  `simpleCDShellCommandTrailingFields`, `cleanShellPathToken`,
  `filepathBase`, `shellCommandFields`, `shellControlToken`), the read-only
  classifier family (`shellExecReadOnly` and friends), the
  `toolsFeature*Pattern` regexes (ticket+disposition shared), and the
  cross-domain shared helpers (`changedFiles`, `dispositionBlockingFiles`,
  `summarizeChangedFiles`, `cleanRepoPath`, `repoFileExists`,
  `shellExecStopsTrackedBackgroundPID` (3-domain),
  `shellExecCommandDisplay` (browser/review 1–1 tie), and the
  `isUntrackedRootBuildArtifact` family (diff/shell 1–1 tie)). **End state:**
  `policy.go` 2,434 → **1,070 lines / 51 functions** — under the ~1,500-line
  target and matching the explicitly accepted dispatcher-core shape
  (dispatchers, trust, shared keys, shared tokenizers/classifiers, the
  dispatcher-resident file-write surface, `checkGitPushPolicy`);
  `policy_ticket_test.go` 2,982 → 1,873 lines. Final file set:
  `policy.go` 1,070; `policy_browser.go` 1,176; `policy_capability.go` 655;
  `policy_diff.go` 264; `policy_disposition.go` 633; `policy_release.go` 132;
  `policy_review.go` 336; `policy_shell.go` 666; `policy_ticket.go` 1,127;
  `policy_validation.go` 1,219 — no policy file above ~1,500 lines except
  none; the largest is the new validation file at 1,219. Pure motion verified
  by line-multiset equality (zero removed lines; the only additions are the
  two new-file MarsDocSync headers, package clauses, and import scaffolding);
  dispatch order untouched (`preToolPolicy`/`postToolPolicy` absent from the
  diff). One process finding recorded as T-030 evidence: `ticket_create`
  false-duplicated this slice's first ticket title against T-040's
  ("Extract … policy domain into policy_*.go (AD-287 step N)" shape); a
  reworded title created T-043 cleanly. This slice takes the AD-287
  **final-sequence replay checkpoint** (two archetypes per AD-284 tool-policy
  class); evidence in the T-043 ticket and `docs/validation/reports/`.
- **2026-06-12 — Slice 7 landed (T-042, `policy_disposition.go`):** 27
  functions plus the `featureScenarioSection` type moved (seed estimate
  was ~26): the `job_disposition_record` gate family, CTO handoff batch
  checks, docsync-at-disposition checks, the T-039-deferred
  capability-coverage consumers (`checkProductCapabilityScenarioCoverage`,
  `scenarioCoversProductCapability`, `productScenarioIDsForHandoff`,
  `earlyCTOHandoffRequiredScenarios` and friends), and the T-040/T-041
  deferrals whose callers are all disposition-domain
  (`countCoveredFeatureScenarios`, `firstUncoveredFeatureScenarios`,
  `featureContractSuperseded`, `successfulReviewDispositionStatus`)
  (`policy.go` 3,089 → 2,434 lines). Four more caller-majority
  reassignments settled in the same pure-motion commit:
  `pendingCTOHandoffRequiredScenarios`, `splitScenarioList`, and
  `quoteStringArray` → `policy_ticket.go` (ticket-side callers only;
  the CTO-handoff session key stays in `policy.go` per the shared-key
  rule, keeping the producer/consumer pair discoverable);
  `roleRequiresDocSyncForSuccessfulDisposition` → `policy_review.go`
  (three review callers vs one disposition). Test motion: 34
  disposition-domain tests (job-disposition gates, the COO-completion
  coverage family per T-039's blended-test record, the CTO-completion
  batch family) plus 19 mover-only Tetris fixture helpers →
  `policy_disposition_test.go`; `TestQAApprovalRequiresGoTestsForGoSource`
  → `policy_review_test.go` as review residue
  (`policy_ticket_test.go` 5,376 → 2,982 lines). Shared fixtures
  (`writeDetailedTetrisBrief`, `writeTetrisFeatureWithFullScenarioSchedule`,
  `writeTetrisFeatureWithGenericScenarios`, `writePolicyPlan`,
  `writePolicyFeature`, `setupPolicyTicketRepo`, `writePolicyTicket`) stay
  in `policy_ticket_test.go` with both ticket and disposition consumers.
  **Slice-6 motion defect found and fixed:** the slice-B extraction left
  the doc comments of the three exported review functions orphaned in
  `policy.go` (the extraction tooling started spans at the `func` line);
  this slice reattaches them to their declarations in `policy_review.go`
  and the doc-comment audit of every function moved in slices 1–7 found
  no other orphan. Deferred to the validation slice and recorded in
  T-042: `dispositionBlockingFiles` and `summarizeChangedFiles`
  (cross-domain shared), `planningRoleCannotMutateWithShell`
  (dispatcher-adjacent trust enforcement), `checkGitPushPolicy` (still no
  caller-graph signal). Pure motion verified by line-multiset equality;
  intermediate slice per the checkpoint policy, rides the test suite.
- **2026-06-12 — Slice 6 landed (T-041, `policy_review.go`):** 11 functions
  moved (seed estimate was ~19; the delta is deferrals recorded in T-041):
  the review terminal-gate family (`checkReviewTerminalDispositionOnly`,
  `reviewTerminalDispositionGuidance`, `ReviewTerminalEvidenceSatisfied`,
  `MarkReviewTerminalDispositionRequired`, `ReviewTerminalDispositionGuidance`),
  reviewer shell validation policy (`checkReviewValidationFailureShellPolicy`,
  `checkReviewerShellExecValidationPolicy`, `shellExecRunsValidationCommand`
  by sole-caller majority), review approval evidence
  (`checkReviewDispositionValidationEvidence`), changes-requested ownership
  (`checkReviewChangesRequestedFeedbackOwnership`), and the shared role
  predicate `reviewRoleRequiresValidationEvidence` (`policy.go` 3,316 →
  3,089 lines). Seventeen review-domain tests moved to
  `policy_review_test.go` per the T-040 test-placement decision
  (`policy_ticket_test.go` 5,790 → 5,376 lines);
  `TestReviewHTTPProbeBeforeServerStartIsProcedureFailure` stays because its
  subject is validation procedure-failure accounting via
  `recordSessionToolOutcome`. Deferrals by caller majority:
  `successfulReviewDispositionStatus` (2 disposition callers vs 1 review)
  waits for the disposition slice;
  `expectedExitCodeCorrectsUnexpectedValidationFailure` (1–1 review/validation
  tie, validation semantics) and `shellExecRunsValidationCommandForSession` +
  `shellExecCountsAsValidationEvidence` (executor-called, validation
  accounting) wait for the validation slice; `shellExecStopsTrackedBackgroundPID`
  stays shared per T-038. One pre-existing wart fixed: slice 5 left a
  gofmt-reported trailing blank line at `policy.go` EOF; trimmed here (the
  only non-motion line change in the slice, recorded in T-041). Pure motion
  otherwise verified by line-multiset equality; intermediate slice per the
  checkpoint policy, rides the test suite.
- **2026-06-12 — Slice 5 landed (T-040, `policy_ticket.go`):** 55 functions
  moved (seed estimate was ~62; the delta is recorded deferrals in T-040):
  the ticket_create gate/planning-order family, the dogfood finding-commit
  family, the claim-before-mutation family, ticket file-write and
  done-evidence checks, the lifecycle move-parser family, and the
  scenario-coverage helpers assigned by T-039's deferral record
  (`policy.go` 4,395 → 3,316 lines). One additional pure move resolves a
  T-038 deferral: `reservedHarnessPortInScript` → `policy_browser.go`
  (sole caller is a browser package-write check; unambiguous).
  **Test-placement decision:** `policy_ticket_test.go` IS the ticket-domain
  test home — exactly the end-state this AD already names ("contains only
  ticket-domain tests, making its name honest... without a rename"), so
  slice 5 moves no tests; slices 6/7 and the validation slice keep moving
  non-ticket residue out. Assignment resolutions recorded in T-040:
  `ticketLifecyclePathIdentity` moved (ticket caller majority 8-vs-4-diff,
  confirming T-037's classification); `isTicketLifecycleDir` moved (1–1
  caller tie broken by ticket-lifecycle semantics);
  `countCoveredFeatureScenarios`, `firstUncoveredFeatureScenarios`, and
  `featureContractSuperseded` deviate from T-039's "and friends" family
  grouping because their remaining callers are all disposition-domain —
  they wait for the disposition slice; `checkGitPushPolicy` stays deferred
  (dispatcher-only caller, body is pure trunk-branch enforcement — caller
  graph gives no domain signal; validation slice settles it);
  `checkEngineerPostValidationCompletionShellPolicy` +
  `engineerDirtyPostValidationGuidance` stay for the validation slice
  (validation-evidence keys and browser-validation helpers dominate the
  body); the feature-contract write-policy family
  (`checkFeatureScenarioIDPolicy`, `checkFeatureFileWritePolicy`, and
  helpers) and the planner write-path family stay with the
  `checkFileWritePolicy` sub-dispatcher for the validation slice;
  `repoFileExists` is cross-domain shared (dependency-sync, hygiene,
  validation callers) and stays in `policy.go`, likely permanently. Pure
  motion verified by line-multiset equality; intermediate slice per the
  checkpoint policy, rides the test suite.
- **2026-06-12 — Slice 4 landed (T-039, `policy_capability.go`):** 25
  functions moved, exactly matching the seed estimate (`policy.go` 5,005 →
  4,395 lines): the brief-interpretation family (including the T-036
  deferrals `projectBriefMentionsFramework` and `projectBriefNamesGoBackend`,
  whose callers are all browser checks but whose home is the interpretation
  layer), the capability surface family, and the capability matching/keyword
  family. The two pure capability unit tests moved to
  `policy_capability_test.go`; the blended COO-completion tests stay in
  `policy_ticket_test.go` because their primary subject is the disposition
  coverage gate. The fuzzy feature-contract boundary resolved as predicted:
  the scenario-coverage helper family (`featureIDsFromScenarios`,
  `featureScenarioCoverage`, and friends) has ticket-gate caller majority and
  waits for the `policy_ticket.go` slice, while the coverage/handoff
  consumers (`checkProductCapabilityScenarioCoverage`,
  `scenarioCoversProductCapability`, `productScenarioIDsForHandoff`) stay for
  the disposition slice. Pure motion verified by line-multiset equality;
  intermediate slice per the checkpoint policy, rides the test suite.
- **2026-06-12 — Slice 3 landed (T-038, `policy_shell.go`):** 33 functions
  moved (seed estimate was ~57; the delta is the shared parsing core, recorded
  as deferrals in T-038): the shell entry checks, the foreground/long-running
  family, the no-op loop check, and the destructive-operation/git-safety
  family (`policy.go` 5,650 → 5,005 lines). Four shell-domain tests moved to
  `policy_shell_test.go` (`policy_ticket_test.go` 5,929 → 5,822 lines); the
  bulk of shell-safety unit tests already live in `shell_exec_test.go`, their
  domain-named home. Key boundary findings: the core tokenizers
  (`shellFields`, `normalizedShellExecFields`, `cleanShellPathToken`,
  `filepathBase`) and the read-only classifier family (`shellExecReadOnly`
  and friends) are cross-domain shared surface and stay in `policy.go` —
  likely permanently, alongside the dispatchers and key constants; the
  build-output/path helpers (`goBuildOutputPath`, `pathResolvesInsideRepo`,
  `shellRemovalPathOperands`, `isUntrackedRootBuildArtifact` family) have
  validation-lane callers and wait for the validation slice;
  `shellExecStopsTrackedBackgroundPID` (seeded as shell "background cleanup")
  is actually browser/validation/review shared; `reservedHarnessPortInScript`
  (seeded as shell "port") has only a browser-domain caller. `hasToken`
  assigned shell by caller majority (2 moving callers vs 1 in
  `policy_release.go`). Pure motion verified by line-multiset equality;
  intermediate slice per the checkpoint policy, rides the test suite.
- **2026-06-12 — Slice 2 landed (T-037, `policy_release.go` + `policy_diff.go`):**
  4 release-gate functions and 10 diff/secrets functions moved
  (`policy.go` 6,009 → 5,650 lines); the four `diffStats` tests moved to
  `policy_diff_test.go`. Release-tag policy tests already live in
  `shell_exec_test.go`, so the release domain needed no test motion.
  Borderline functions recorded in T-037: `changedFiles` (shared across
  four domains) and `ticketLifecyclePathIdentity` (ticket domain) stay in
  `policy.go`; `checkGitPushPolicy` assignment deferred. Per the
  checkpoint policy, this pure-motion slice rides the slice-1 replay
  checkpoint (demo-12 Run 3, PASS) and the test suite; no dedicated
  replay.
- **2026-06-12 — Slice 1 landed (T-036, `policy_browser.go`):** 46 functions
  plus the `browserFrameworkInfo` type moved (seed estimate was ~52);
  `policy.go` 7,091 → 6,009 lines. The delta against the seed count is
  deliberate boundary discipline, recorded in T-036: brief-interpretation
  helpers (`projectBriefMentionsFramework`, `projectBriefNamesGoBackend`)
  stay for the `policy_capability.go` slice despite browser-majority
  callers; `shellExecRunsHTTPProbe` (reviewer/validation shared),
  `testFilePath` (validation caller), and the in-progress→done ticket-move
  parsers (ticket domain, also called from `executor.go`) stay put. Browser
  tests and the four Phaser fixture helpers moved to
  `policy_browser_test.go` in the same commit
  (`policy_ticket_test.go` 7,835 → 6,086 lines).

- `policy_ticket_test.go` is misnamed: it holds all 183 policy test
  functions across every domain, not just ticket tests. The split
  restores the name's honesty without a rename.
- Verified measured shape at v0.50.8: 7,091 lines / 354 functions
  (slightly above the 7,033 / ~352 estimate from the prior inventory
  pass — the file grew during Phase 2).
