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
