# Advertised Git Publication Scan

- Date: 2026-07-22
- Ticket: T-070
- Scenario: F-017-S001
- Source commit: `375a3a30140c9248f10c19eb4ff8a66ba83b7522`
- Status: `owner_classification_required`
- Primary Status: `primary_blocked`
- Publication authority: denied

## Primary Outcome Contract

**Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.

**Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including approved history, secure runtime defaults, anonymous signed release access, fork-safe contribution, logged-out cutover smoke, and a clean 48-hour canary.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** After owner-only review, Elliott classified the Gitleaks group as synthetic test stubs. The separate URI-detector group remains unclassified; if real or uncertain, it must be rotated or revoked before both tools' Git-history and raw-object lanes are rerun.

**Next Primary Action:** Elliott reviews opaque URI group `e32927624f4a2cac` locally and records only whether it is synthetic test stubs or real/unknown. If real/unknown, rotate or revoke it outside the agent boundary before the four-lane rescan. Candidate values and locations must remain outside chat and repository evidence.

**Supporting Evidence:** The frozen 302-ref, 11,954-object publication surface completed both pinned tools' Git-history and raw-object lanes without accepted-scan errors or skips; the redacted coverage below does not claim that unresolved findings are clean.

## Outcome

The exact advertised Git publication surface was frozen and scanned with pinned standard tools. Raw reports and candidate data remain in an owner-only FileVault-protected boundary and were not read into agent, chat, CI, trace, or repository output. Accepted scans reported zero execution errors and zero skip events.

The scan is not yet clean. On 2026-08-08, after reviewing the owner-only Gitleaks context, Elliott classified that group as synthetic test stubs. The separate URI-detector group was not part of that review and remains unclassified without reproducing or inferring its candidate values. T-070 therefore remains stopped before further scanning or audit slices, and F-017-S001 remains blocked.

## Frozen Surface

| Evidence | Result |
| --- | --- |
| Canonical roots | one `refs/heads/main` plus 301 retained tag refs |
| Canonical ref count | 302 |
| Ref-manifest SHA-256 | `41c901cb37df993fe19f85ed5dc31d46d1e19b1e204143f06e4beab79e2369c8` |
| Reachable commits | 844 |
| Reachable objects | 11,954 |
| Reachable-object manifest SHA-256 | `b0bfd34f40029aee79fbccd235262f4a9cc78fae412c7ce46e60eb75f65c0d89` |
| Object types | 5,946 blobs; 5,134 trees; 844 commits; 30 annotated tags |
| Other advertised namespaces | 0 |
| Local-only inputs | stashes, reflogs, administrative refs, unreachable objects, and the unrelated worktree were preserved and excluded |

A mirror and a standard-layout no-checkout clone resolved to the same reachable-object manifest. Stock Git then materialized every reachable object under its opaque object identifier. The corpus contained exactly 11,954 regular files, zero links, and zero special nodes.

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
| Gitleaks Git history | all canonical roots with all-ref history traversal; scanner reported 838 commit patches | exit 1 for seven `generic-api-key` findings; zero scan errors/skips; all seven locations mechanically test-like |
| Gitleaks raw objects | all 11,954 materialized objects, including all 844 commit objects | exit 1 for 19 blob occurrences in the same broad class; zero scan errors/skips |
| TruffleHog Git history | standard-layout clone with the exact frozen reachable-object set | exit 183 for five unverified `URI` occurrences representing three distinct candidate values; zero verified findings and zero scan errors/skips; all five locations mechanically test-like |
| TruffleHog raw objects | all 11,954 materialized objects | exit 183 for seven blob occurrences representing the same three distinct candidate values; zero object-only candidates, verified findings, scan errors, or skips |

The difference between 844 reachable commit objects and 838 Gitleaks-reported commit patches is closed by the independent raw-object lane: every commit object, tag object, tree, and blob was present in the sealed corpus and scanned. The report does not infer that a nonzero scanner exit was an execution failure; the accepted nonzero exits are the tools' configured finding outcomes.

| Opaque finding ID | Broad class | Status |
| --- | --- | --- |
| `a2a292e31d652f22` | generic API-key detector results | `resolved_owner_confirmed_synthetic_test_stubs` |
| `e32927624f4a2cac` | unverified URI detector results | `unresolved_owner_classification` |

No candidate value, fragment, location, filename, email, URL, body, raw scanner record, or candidate-derived hash is included here. The owner classification resolves only the Gitleaks group because that was the context reviewed. The URI group remains unresolved until its separate owner-only classification; real or uncertain content requires separately approved rotation/revocation.

## Rejected Attempts

Two TruffleHog setup invocations failed before scanning because a bare mirror did not provide the working-repository layout and index expected by that source adapter. A no-checkout clone of the same sealed object set plus a local index corrected the interface mismatch. Neither rejected attempt produced findings or accepted coverage; only the later zero-error runs above are evidence.

## Postconditions

- Local `HEAD`, fetched `origin/main`, and the frozen source commit remained equal.
- The canonical publication-ref manifest was byte-identical after scanning.
- Repository visibility remained private.
- Active or queued Actions runs were zero at the accepted postcondition.
- No unexpected advertised namespace appeared.
- T-070 initiated no ref, Release, asset, Pages, settings, credential, signing, visibility, or publication mutation.

## Remaining Gate

Owner classification of `e32927624f4a2cac` is required before T-070 can resume. If it is real or uncertain, rotate or revoke it first. After every group is resolved, both pinned tools' Git-history and raw-object lanes must rerun against a newly frozen publication manifest; any retained synthetic fixture must reconcile to its explicit owner classification without exposing it. GitHub-hosted content, manual privacy/IP/provenance/name review, and final owner disposition remain blocked behind this stop.
