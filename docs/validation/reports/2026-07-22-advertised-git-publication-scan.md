# Advertised Git Publication Scan

- Date: 2026-07-22
- Latest checkpoint: 2026-08-08
- Ticket: T-070
- Scenario: F-017-S001
- Source commit: `d04f642e7b07e14e211c99f40e14bdd4bccef60e`
- Status: `plausible_secret_rotation_required`
- Primary Status: `primary_blocked`
- Publication authority: denied

## Primary Outcome Contract

**Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.

**Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including approved history, secure runtime defaults, anonymous signed release access, fork-safe contribution, logged-out cutover smoke, and a clean 48-hour canary.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** Two finding groups are resolved as synthetic test stubs. Opaque group `f3dc0e336620abc6` contains 36 occurrences mechanically reduced to four distinct values; bounded local-MARS review could not establish that any of the four is synthetic, so all four require owner rotation or revocation.

**Next Primary Action:** Elliott rotates or revokes all four potentially real values in `f3dc0e336620abc6` outside the agent boundary, then reports only `rotation complete: f3dc0e336620abc6`. Candidate values and locations must remain outside chat, traces, and repository evidence.

**Supporting Evidence:** The newly frozen 302-ref, 12,002-object publication surface completed both pinned tools' Git-history and raw-object lanes without accepted-scan errors or skips; the redacted coverage below does not claim that the unresolved group is clean.

## Outcome

The exact advertised Git publication surface was frozen and scanned with pinned standard tools. Raw reports and candidate data remain in an owner-only FileVault-protected boundary. Bounded candidate context was read only by isolated local MARS with credentials absent, offline controls enabled, outbound proxies directed to failing loopback endpoints, and `file_read` as its sole allowed tool; no candidate data entered chat, CI, normal traces, or repository output. Accepted scans reported zero execution errors and zero skip events.

The scan is not yet clean. The 2026-08-08 rescan reconciled the prior Gitleaks and URI groups to synthetic fixtures without exposing their values. A separate broad group contains four distinct values that local MARS could not establish as synthetic. T-070 therefore remains stopped for owner rotation or revocation before any rescan or later audit slice, and F-017-S001 remains blocked.

## Frozen Surface

| Evidence | Result |
| --- | --- |
| Canonical roots | one `refs/heads/main` plus 301 retained tag refs |
| Canonical ref count | 302 |
| Ref-manifest SHA-256 | `75a56fc716912416f189ba5aef07c49baada2976d2493df22fe14abc50be3720` |
| Reachable commits | 847 |
| Reachable objects | 12,002 |
| Reachable-object manifest SHA-256 | `c2e081aacf509362d18e2598b70e2b812dd126bb47c8f3bfa78f8ce5915e6059` |
| Object types | 5,961 blobs; 5,164 trees; 847 commits; 30 annotated tags |
| Other advertised namespaces | 0 |
| Local-only inputs | stashes, reflogs, administrative refs, unreachable objects, and the unrelated worktree were preserved and excluded |

A mirror and a standard-layout no-checkout clone resolved to the same reachable-object manifest. Stock Git then materialized every reachable object under its opaque object identifier. The corpus contained exactly 12,002 regular files, zero links, and zero special nodes.

## Tool Admission

| Tool | Pin and provenance | Accepted binary SHA-256 |
| --- | --- | --- |
| Gitleaks | v8.30.1; two independent exact-module/SumDB builds produced byte-identical binaries | `03eefdd1eaba674c37fde4971f9c005b90e21b0c216c8beb0e4613ca893d6d11` |
| TruffleHog | v3.95.9; publisher certificate/signature and the selected archive checksum verified before extraction | `8c6110728eca539ac188a149d8a1e0510e5e59e4d3e3f1ce9daa41fa4961814f` |
| Cosign | v3.0.6; exact-module/SumDB build used only for connected TruffleHog staging | `2ca1d7a73488cc73c3c0b9031fc0aed5ff745a4b6b5712272d76b20809c883a3` |

Accepted scanner processes used an empty home, sanitized environment, no GitHub or provider credentials, disabled update and online-verification behavior, and failing local proxy endpoints. Scanner stdout, stderr, and reports were written directly into the owner-only boundary.

## Coverage And Findings

| Scanner lane | Coverage | Accepted outcome |
| --- | --- | --- |
| Gitleaks Git history | all canonical roots with all-ref history traversal | exit 1 for seven findings; zero scan errors/skips; locators reconcile to resolved synthetic group `a2a292e31d652f22` |
| Gitleaks raw objects | all 12,002 materialized objects, including all 847 commit objects | exit 1 for 19 occurrences in the same broad class; zero scan errors/skips; locators reconcile to the same resolved group |
| TruffleHog Git history | standard-layout clone with the exact frozen reachable-object set | exit 183 for five unverified URI occurrences representing one distinct value; zero verified findings and zero scan errors/skips |
| TruffleHog raw objects | all 12,002 materialized objects | exit 183 for 43 occurrences: seven URI occurrences representing the same resolved value plus 36 occurrences representing four distinct unresolved values; zero verified findings, scan errors, or skips |

Every commit object, tag object, tree, and blob was present in the sealed corpus and scanned. The report does not infer that a nonzero scanner exit was an execution failure; the accepted nonzero exits are the tools' configured finding outcomes.

| Opaque finding ID | Broad class | Status |
| --- | --- | --- |
| `a2a292e31d652f22` | generic API-key detector results | `resolved_owner_confirmed_synthetic_test_stubs` |
| `e32927624f4a2cac` | unverified URI detector results | `resolved_local_mars_confirmed_synthetic_test_stubs` |
| `f3dc0e336620abc6` | unverified broad detector results; 36 occurrences, four distinct values | `real_or_unknown_rotation_required` |

No candidate value, fragment, location, filename, email, URL, body, raw scanner record, or candidate-derived hash is included here. The URI group was reviewed by an isolated local MARS run limited to owner-only context and `file_read`, with credentials absent, offline controls enabled, and outbound proxies directed to failing loopback endpoints. The four distinct values in the remaining group were also reviewed locally, but every classification remained unknown; no synthetic disposition is inferred from test-like context.

## Rejected Attempts

Across the two checkpoints, three TruffleHog setup invocations failed before scanning. Two used a bare mirror without the working-repository layout and index expected by the source adapter; the 2026-08-08 rescan's first Git-lane invocation used the accepted clone layout but lacked its local index. A no-checkout clone of the sealed object set plus an index derived from frozen `HEAD` corrected the interface mismatch. None of the rejected attempts produced accepted coverage; only the later zero-error runs above are evidence.

## Postconditions

- Local `HEAD`, fetched `origin/main`, and the frozen source commit remained equal.
- The canonical publication-ref manifest was byte-identical after scanning.
- Repository visibility remained private.
- Active or queued Actions runs were zero at the accepted postcondition.
- No unexpected advertised namespace appeared.
- T-070 initiated no ref, Release, asset, Pages, settings, credential, signing, visibility, or publication mutation.

## Remaining Gate

Owner rotation or revocation of all four potentially real values in `f3dc0e336620abc6` is required before T-070 can resume. The owner reports only `rotation complete: f3dc0e336620abc6`; values and locations remain outside chat, normal traces, and repository evidence. After rotation, both pinned tools' Git-history and raw-object lanes must rerun against a newly frozen publication manifest, and every retained fixture must reconcile to its explicit disposition without exposure. GitHub-hosted content, manual privacy/IP/provenance/name review, and final owner disposition remain blocked behind this stop.
