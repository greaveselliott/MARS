# T-078 Hosted-State Revalidation — Mutation Pending

**Date:** 2026-08-24  
**Ticket:** T-078  
**Repository:** `greaveselliott/MARS`  
**Source:** `628b4b5109e35df83d15cd8143a3433435a12593`  
**Visibility:** private  
**Mutation status:** none performed  
**Primary outcome:** the current hosted surface and exact cleanup sets are
reconciled; destructive cleanup and the future-only immutable-Release setting
still require separate owner approval.

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project through
the shortest credible conventional producer, attestation, and consumer path.

**Primary Pass Gate:** the repository is public; attested `v0.69.1` is the
supported release with attested `v0.69.0` retained only as its rollback bridge;
the anonymous lifecycle, contribution controls, public security surfaces, and
48-hour canary pass before announcement.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** the exact hosted cleanup/future-only immutable-
Release transaction needs separate owner approval, and hosted source
compatibility is red on called vulnerabilities fixed in Go 1.25.13. Public
visibility, launch attestations, Releases, and canary remain later gates.

**Next Primary Action:** present the exact cleanup and setting transaction for
separate approval without broadening the vulnerability disposition or changing
GitHub state before an immediate live revalidation.

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

## Current Hosted Inventory

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

## Proposed Mutation Transaction — Not Yet Authorized

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

## Outcome

The read-only hosted-state refresh passes. Exact current cleanup inputs and
consequences are reviewable and no secret candidate remains. T-078 is still
blocked on separate mutation authority, successful bounded postconditions, and
an accepted-source green hosted run. No launch tag, upload, Release,
attestation, visibility change, publication, or announcement occurred.
