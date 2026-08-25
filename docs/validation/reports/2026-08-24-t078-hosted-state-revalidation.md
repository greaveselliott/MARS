# T-078 Hosted-State Revalidation And Cleanup — Complete

**Date:** 2026-08-24  
**Ticket:** T-078  
**Repository:** `greaveselliott/MARS`  
**Source:** `628b4b5109e35df83d15cd8143a3433435a12593`  
**Visibility:** private  
**Mutation status:** owner-approved exact cleanup and future-only immutable
Releases completed on 2026-08-25
**Primary outcome:** the exact legacy hosted surface is sanitized, all named
preservation postconditions pass, and future Releases are immutable.

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project through
the shortest credible conventional producer, attestation, and consumer path.

**Primary Pass Gate:** the repository is public; attested `v0.69.1` is the
supported release with attested `v0.69.0` retained only as its rollback bridge;
the anonymous lifecycle, contribution controls, public security surfaces, and
48-hour canary pass before announcement.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** T-078 is complete. Separately approved public
visibility, public-only controls, launch attestations/Releases, and the canary
remain T-080/T-081 gates.

**Next Primary Action:** activate T-080, revalidate its public-cutover inputs,
and obtain its separate visibility/publication approval before mutation.

**Supporting Evidence:** the exact inventory, run-delta scan, owner-only
manifest digests, consequences, and proposed transaction below.

## Authority Boundary

This checkpoint used authenticated read-only GitHub REST acquisition. It did
not delete a Release asset, workflow run, deployment, tag, or Release; create a
deployment status; enable immutable Releases; change visibility; run a release
workflow; upload an artifact; request OIDC; publish; or announce.

Account-wide GitHub App scope remains outside the MARS launch boundary by owner
disposition. The repository stayed private, local `HEAD` and `origin/main`
remained equal, and the worktree was clean before acquisition.

## 2026-08-24 Hosted Inventory

| Surface | Current state | Exact identity evidence |
|---|---:|---|
| Tags | 301 | sorted-name SHA-256 `7fd680992183459f3b8991331abe64837b3ac87ec3b7939d75f0a7c88ff9301e` |
| Releases | 56, all published and mutable legacy Releases | sorted-ID SHA-256 `fc5fb6824d6c6688274f98cd00d8c9ddf226f219b8a7f10b7390d7ed620d122f` |
| Release assets | 500 uploaded assets; 11,828,329,001 bytes | sorted-ID SHA-256 `4cde2219cb027dd0814186781a4821ee5a45327cd3c8e548ece1bf1a6e58700b`; structured record SHA-256 `11e12d8978d96553ec03688517725d493d5b2d10e7b5756a7d8c642ae9d199aa` |
| Workflow runs | 473 completed; zero queued or in progress | sorted-ID SHA-256 `2dffa5f0a2947d156658de8941ba7e72986f71944ccd56506fbedcf1f4a9b7ec` |
| Deployments | 77 `github-pages` records | sorted-ID SHA-256 `8b74b6d5b356696e244efdac506ced8fba7c0c43554eb7791d01f5349406506e` |
| Actions artifacts | 0 | current REST inventory |
| Actions caches | 0 | current REST inventory |
| Addressable workflows | 3 active | sorted-ID SHA-256 `b7c5dd202d09498da2c8a846fc69a4916b69d4919f9b6dcf01454a38a780ee21` |
| Repository Pages state | disabled (`has_pages=false`) | current repository receipt |
| Immutable Releases | disabled; not owner-enforced | `{"enabled":false,"enforced_by_owner":false}` |

T-072 recorded 57 Releases. The current inventory already contains the intended
56 historical Releases, and every Release tag is still present in the 301-tag
set. Therefore this proposed transaction deletes no Release object and no tag.
The 500-asset identifier digest exactly reproduces T-072, as does the 77-
deployment identifier digest.

## Workflow-Run Reconciliation And Secret Scan

The sorted oldest 401 run IDs reproduce T-072's SHA-256
`cdfb0cc95e6dc8f4e2b9e910f3c897834fa17d2d18f3a9e454c2b04db74b938b`.
The next 65 reproduce the separately sealed T-078 SHA-256
`2cb0b770e3915f6fcacc6cd086ba9fe808e8279c6f50e7b6a7279b5da8d9e0fc`.
The exact seven-run delta has sorted-ID SHA-256
`5a846450bbc5fe21ead22659a240ea3a1f511c3676` and contains seven completed,
failed `source-compatibility` push runs, 28 jobs, zero Actions artifacts, and
seven log responses.

One log response is the canonical 22-byte empty ZIP, SHA-256
`8739c76e681f900923b900c9df0ef75cf421d39cabb54650c4b9ad19b6a76d85`;
its four job records contain no steps. The other six ZIPs passed integrity,
member-name, link, entry-count, and expanded-size checks before 132 regular log
files were extracted. Their sorted extracted-file digest aggregate is
`4d8dbd74c4080ff91e81fe44c66e6c4a6aea1ac3f294af1909798cd3024699eb`.

Exact Gitleaks v8.30.1 binary SHA-256
`ba52fb1bfabbcde42f032afad3d6e0b19dff8ed105229a16e7caa338bbc0e84f`
and exact TruffleHog v3.95.9 binary SHA-256
`8c6110728eca539ac188a149d8a1e0510e5e59e4d3e3f1ce9daa41fa4961814f`
scanned the complete delta acquisition without secret verification or update
network calls. Gitleaks returned zero findings. The authoritative host
TruffleHog run scanned 642 chunks and 5,106,884 bytes and returned zero
verified, unverified, or unknown findings and zero scanner errors. A prior
app-sandbox attempt reported only sandbox-denied process-cleanup diagnostics;
it is retained but is not the authoritative scanner receipt.

The current hosted CI result is independently red: the Go 1.25.12 compatibility
lane reports six called standard-library vulnerabilities fixed in Go 1.25.13
(`GO-2026-6218`, `GO-2026-6091`, `GO-2026-6090`, `GO-2026-6089`,
`GO-2026-5972`, and `GO-2026-5026`). No exception or disposition was broadened.
An accepted-source green hosted run remains a launch gate after this cleanup
checkpoint.

## Owner-Only Exact Manifests

The acquisition root is owner-only mode `0700`; retained files are mode `0600`
except the two scanner executables, which are mode `0700`. Its 202-file digest
manifest has SHA-256
`cff22933e090becbcc0779dbd78fe8a6ca8ccc03b49430b6d387eaff79302fdb`.
Exact live identifiers remain owner-only and are bound by these manifest
digests:

| Proposed operation | Count | Manifest SHA-256 |
|---|---:|---|
| Delete exact Release assets | 500 | `d1017a350956b5082fccc40ec08dedd8c2c2d9dbca849269e5bb5b52f4521427` |
| Inactivate when required, then delete exact deployments | 77 | `379df1d509909db7b6e130be7f6e7dca280439790164c15435ea313f594a8306` |
| Delete exact completed workflow runs | 473 | `74cc49fa17ed4e11d51add27dac2365cb333067d0f08239531382eaf1e873348` |
| Preserve Releases/tags and change only future immutability | 56 Releases / 301 tags | `403e1a0864599a7f1718e1597e3bf673b3e74cdc2e714a181d686dfee8bafd04` |

Of the 77 deployments, 74 are already inactive. Two have latest state
`failure` and one has latest state `success`; those three require an exact
`inactive` status before GitHub permits deletion. Because repository Pages is
disabled, the operation removes historical deployment records rather than a
current Pages site.

## Proposed Mutation Transaction At The 2026-08-24 Seal

If separately approved, the transaction is:

1. Re-fetch the live repository identity, visibility, immutable-Release state,
   all Release, asset, tag, workflow-run, artifact, cache, and deployment IDs.
   Stop before mutation unless every count and sorted digest exactly matches
   this checkpoint, all 473 runs remain completed, and no unexpected object or
   setting exists.
2. Delete only the 500 exact asset IDs. Preserve all 56 Release objects, their
   notes, and all 301 tags. Require the expected REST success for each ID and
   record the exact completed subset after every call.
3. For only the three exact non-inactive deployments, create an `inactive`
   deployment status. Delete only the 77 exact deployment IDs. Stop on any
   identity, state, or response mismatch.
4. Delete only the 473 exact completed workflow-run IDs. There are no Actions
   artifacts or caches to delete. Do not delete any run created after this
   seal.
5. Re-fetch and require 301 tags, 56 Releases, zero Release assets, zero
   deployments, zero pre-seal workflow runs, zero Actions artifacts, zero
   Actions caches, unchanged private visibility, and unchanged source refs.
6. Only after destructive cleanup passes, enable repository immutable Releases
   through GitHub's documented repository setting, then require
   `enabled=true` and `enforced_by_owner=false`.
7. Re-fetch all preserved Releases and tags. Require the existing 56 Releases
   to remain legacy-mutable, because GitHub documents that the setting applies
   only to future Releases. Record the final hosted receipt before deleting
   local owner-only acquisition data.

Deleting assets, workflow runs/logs, and deployment history is irreversible.
Enabling immutability affects future Releases only: when a future draft is
published, its assets and associated tag cannot be modified or deleted in the
ordinary flow, and GitHub creates a release attestation. GitHub recommends
assembling all assets on a draft before publication.

Official references:

- [Immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases)
- [Preventing changes to your releases](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/prevent-release-changes)
- [Repository immutable-Releases REST endpoint](https://docs.github.com/en/rest/repos/repos?apiVersion=2026-03-10#enable-immutable-releases)

## Outcome At The 2026-08-24 Seal

The read-only hosted-state refresh passed at this seal. Exact cleanup inputs
and consequences were reviewable and no secret candidate remained. T-078 was
then blocked on separate mutation authority, successful bounded
postconditions, and an accepted-source green hosted run. No launch tag,
upload, Release, attestation, visibility change, publication, or announcement
occurred during this checkpoint.

## 2026-08-25 Approval Revalidation And Source-Floor Remediation

The owner approved the originally sealed asset/run/deployment cleanup and
future-only immutable-Release transaction. Immediate fail-closed revalidation
then found that the 77 deployments were already absent. MARS stopped before
the first destructive call. Read-only reconciliation proved 56 Releases, 301
tags, the same 500 assets, 474 completed workflow runs, zero deployments, zero
artifacts/caches/active runs, disabled immutable Releases, private visibility,
and `has_pages=false`. The exact 474-run manifest SHA-256 is
`ab542a1921cf82ae4cb97a5576f470eb2294a15158d6348cc129b9667d05597a`;
the asset projection SHA-256 is
`14a6126ffeaa4a5e10eb59a90b77b4594f79258d03bd4483283d50cc11adcb93`;
the preserved 56-Release/301-tag manifest SHA-256 is
`141252fa879542d991ef98258626274dbf6c25b14958845aa61b4bacc7f4961e`.
No GitHub mutation occurred. A revised 500-asset/474-run cleanup plus
future-only immutable-Release transaction therefore remains separately
approval-gated.

The owner separately approved the shortest safe hosted-CI remediation: raise
the current MARS source and one-time bootstrap floor from Go 1.25.12 to Go
1.25.13, move the intentional below-floor lane to exact Go 1.25.12, retain
exact Go 1.27.0 release production, and add no vulnerability exception.

Before push, official Go 1.25.13 was selected exactly and passed `go mod tidy
-go=1.25.13`, the complete `go test ./...` suite, full `go vet ./...`, and a
CGO-disabled `./cmd/mars` build. Exact Go 1.25.12 with `GOTOOLCHAIN=local`
failed with `go.mod requires go >= 1.25.13`, as required. The affected
doctor/selfupdate/release race suite passed. SumDB-built `govulncheck v1.7.0`
under exact Go 1.25.13 and DB timestamp `2026-08-21T20:38:00Z` reported zero
called vulnerabilities; three required-module findings remain uncalled. The
dependency notices, DocSync audit with zero findings, formatting, and diff
checks passed.

Exact pushed commit `a4bbf81ccd29d4e8502a64cf649750bf3cb65c70`
then passed hosted source-compatibility run `32848968969`: exact Go 1.25.13
source compatibility completed in 4m21s, exact Go 1.27.0 completed in 4m55s,
dependency notices completed in 1m13s, and the intentional exact Go 1.25.12
module-floor rejection completed in 18s. Every job concluded successfully.
The workflow installed `govulncheck v1.7.0` inside both supported lanes and
therefore closes the hosted called-vulnerability blocker without an exception.

## 2026-08-25 Approved Transaction And Final Receipt

The owner approved the exact transaction only after a new read-only
revalidation. That revalidation found 507 completed and zero active workflow
runs: the previously reviewed oldest 474 remained the deletion set, while 33
newer source, contribution-policy, and Dependabot runs were explicitly
preserved. The original 77 Pages deployment records were visible again, and
their sorted-ID SHA-256 exactly reproduced the earlier seal. This was
reconciled as live-state drift before mutation rather than silently widening
the transaction.

Immediate fail-closed preflight proved:

| Surface | Approved pre-state | Identity receipt |
|---|---:|---|
| Repository | ID `1279592869`; private; default `main`; Pages disabled | `main` `ed6b46ad90cfd47cffb079f7ded587fbb0759bc7` |
| Releases | 56 published legacy Releases | sorted-ID SHA-256 `fc5fb6824d6c6688274f98cd00d8c9ddf226f219b8a7f10b7390d7ed620d122f` |
| Tags | 301 | sorted-name SHA-256 `7fd680992183459f3b8991331abe64837b3ac87ec3b7939d75f0a7c88ff9301e` |
| Release assets | 500; 11,828,329,001 bytes | sorted-ID SHA-256 `4cde2219cb027dd0814186781a4821ee5a45327cd3c8e548ece1bf1a6e58700b` |
| Deployments | 77 | sorted-ID SHA-256 `8b74b6d5b356696e244efdac506ced8fba7c0c43554eb7791d01f5349406506e` |
| Sealed workflow runs | 474 completed | sorted-ID SHA-256 `1b0119895c3b29031412f141ca828d4361c3051f38ef82153c7fbc12cfd38cfc` |
| Preserved workflow runs | 33 completed | sorted-ID SHA-256 `e424fdfe4a43e3fd51547098c0f6bc1ac672765032673d05c493b5ff728f6675` |
| Actions artifacts / caches | 0 / 0 | complete paginated REST inventories |
| Immutable Releases | disabled; not owner-enforced | `enabled=false`, `enforced_by_owner=false` |

The deployment status recheck found 74 already inactive, failures
`5188192046` and `5188234300`, and success `5212835398`. The transaction added
`inactive` only to those three exact non-inactive records, then deleted the
exact 77 deployment IDs.

The bounded mutation completed in four fail-closed phases:

1. delete the exact 500 asset IDs, then prove 56 Releases, zero assets, and
   all 301 tags;
2. inactivate only the three listed deployments, delete the exact 77 IDs, and
   prove zero deployments;
3. delete only the exact 474 sealed completed runs, then prove every sealed ID
   absent and every one of the 33 newer IDs present; and
4. prove the complete cleanup postcondition before enabling immutable
   Releases through GitHub's repository endpoint.

The final hosted receipt is 56 Releases, zero uploaded Release assets, 301
tags, zero deployments, the exact 33 preserved workflow runs, zero Actions
artifacts, zero caches, private visibility, disabled Pages, and unchanged
`main`. Immutable Releases re-fetches as `enabled=true` and
`enforced_by_owner=false`; all 56 historical Releases still report legacy
mutable, as GitHub documents for a future-only repository setting.

Deletion of the assets, workflow runs/logs, and deployment history is
irreversible. No Release object, tag, source ref, visibility, Pages, App,
publication, or announcement mutation occurred. The approval was consumed by
this exact transaction and does not authorize T-080's visibility or release
operations.

The synchronized completion record was pushed at exact source
`2d5c05e994c6794d8d14300a6c69bbe1f58c588d`. Hosted
source-compatibility run `32907186559` passed: Go 1.27.0 in 4m42s, Go 1.25.13
in 4m25s, dependency notices in 1m6s, and the intentional exact Go 1.25.12
module-floor rejection in 18s.
