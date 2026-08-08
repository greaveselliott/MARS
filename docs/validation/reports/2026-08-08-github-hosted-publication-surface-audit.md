# GitHub-Hosted Publication Surface Audit

**Date:** 2026-08-08
**Ticket:** T-072
**Scenario:** F-017-S001
**Snapshot:** `T072-723d689d-3194-40c8-9d92-322b921149a3`
**Status:** `blocked_owner_confirmation`
**Primary Status:** `primary_blocked`

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project without
exposing confidential material, weakening controls, or distributing unsafe or
unverifiable binaries.
**Primary Pass Gate:** the repository is public; signed `v0.69.1` is supported
with signed `v0.69.0` retained only as a rollback bridge; all F-017 scenarios,
logged-out macOS/Linux lifecycle checks, fork controls, GitHub security and
community surfaces, and the 48-hour canary pass.
**Primary Status:** `primary_blocked`
**Current Primary Blocker:** T-072 requires one owner-authenticated read-only
confirmation of GitHub Apps, Packages, Projects v2, and Wiki state before the
remaining T-073 through T-081 launch gates may proceed.
**Next Primary Action:** commit this redacted checkpoint, complete the four
read-only confirmations, and acquire and scan any discovered content.
**Supporting Evidence:** the immutable snapshot, exact reconciled counts,
offline scanner classifications, and cleanup-set digests recorded below.

## Scope And Boundary

This checkpoint records a read-only snapshot of the private GitHub-hosted
publication surface. Connected acquisition used standard Git, GitHub API, and
download tools. Raw records, payloads, scanner output, candidates, and exact
cleanup identifiers remain in the owner-only FileVault boundary. No repository
setting, hosted object, ref, release, asset, deployment, visibility state, or
publication state changed.

Start and end freeze facts matched across protected refs, repository state,
Releases and assets, workflows and runs, artifacts and caches, deployments and
environments, issues and pull requests. Actions remained quiescent and the
repository remained private. The snapshot contains 305 advertised refs: two
branches, 301 tags, and the head and merge refs for one pull request. The extra
branch and pull-request refs were preserved and scanned.

## Reconciled Coverage

| Surface | Result | Evidence |
|---|---|---|
| Releases | collected | 56 paginated records plus one direct-only, zero-asset record reconcile to 57 immutable Release objects. |
| Release assets | collected | 500 of 500 payloads downloaded; all sizes and provider SHA-256 digests match; total logical bytes 11,828,329,001. |
| Asset formats | collected | 111 each of Darwin amd64, Darwin arm64, Linux amd64, and Linux arm64 executables, plus 56 text checksum files; zero other or archive formats. |
| Workflows and runs | collected | Four current and five addressable workflow objects; 400 completed runs, 400 attempts, and 1,728 jobs. |
| Actions logs | collected | 126 non-empty and 274 empty valid ZIP responses; zero invalid, rejected, skipped, or partial logs. |
| Actions artifacts and caches | confirmed_empty | Repository-wide and per-run artifact counts are zero; cache count, usage, and bytes are zero. |
| Deployments and environments | collected | 77 deployments with status histories and one environment with its policy surfaces. |
| Issues, pull requests, and attachments | collected | Zero issues, one pull request, and zero GitHub-hosted attachment references across acquired bodies. |
| Pages | not_applicable | Pages is disabled. |
| Discussions | not_applicable | Discussions is disabled. |
| Access and hooks | collected | One collaborator; zero invitations, hooks, deploy keys, self-hosted runners, rulesets, or protected branches. |
| Security and secret-name metadata | collected | Applicable configuration/status responses and names-only secret/variable inventories were acquired; values were never persisted. |
| GitHub Apps | unresolved | Current API authority cannot enumerate user installations; owner UI confirmation is required. |
| Packages | unresolved | Six package-type inventory calls are inaccessible with current authority; owner UI or names-only read confirmation is required. |
| Projects v2 | unresolved | Current API authority lacks project-read access; owner UI confirmation is required. |
| Wiki | unresolved | Wiki is enabled but its authenticated Git remote is absent; owner UI must confirm that it was never initialized or provide its history. |

## Offline Secret Scan

The admitted tools were Gitleaks v8.30.1
(`ba52fb1bfabbcde42f032afad3d6e0b19dff8ed105229a16e7caa338bbc0e84f`)
and TruffleHog v3.95.9
(`8c6110728eca539ac188a149d8a1e0510e5e59e4d3e3f1ce9daa41fa4961814f`).
Their offline multi-detector canary found both generated detector classes and
kept the generated candidate values out of normal output. TruffleHog release
evidence was verified with Cosign v3.0.6
(`5fadd012ae6381a6a29ff86a7d39aa873878852f1073fc90b15995961ecfb084`).

The acquired-filesystem lanes produced these mechanically resolved classes:

- two Gitleaks hits came from one transient provider-generated clone-response
  field. The immutable raw responses remain owner-only; a separate projection
  removed only that field, and the projection rescanned with zero findings;
- four Gitleaks hits reduced to two Git-commit identifiers used by checkout and
  build metadata, not credentials;
- twelve unverified TruffleHog hits were filename-only matches inside cleanup
  output from one pinned dependency test-data tree: non-credential filename
  fragments rather than live credentials.

The Git-ref lanes produced seven Gitleaks and five unverified TruffleHog hits.
Their commits, detector classes, and distinct values exactly match the T-070
owner-approved synthetic-fixture baseline; the additional branch and pull
request contribute zero findings. Across all lanes there are zero verified
credentials, zero unknown candidates, zero unresolved findings, and zero scan
errors, skips, timeouts, or rejected inputs.

## Immutable Cleanup Sets

Exact identifiers remain owner-only. These counts and sorted identifier-list
digests bind the later T-078 cleanup transaction:

| Object class | Count | Identifier-list SHA-256 |
|---|---:|---|
| Release objects | 57 | `a2b7e8a3aada4ee7bbee1ee86f0c768e4a46887d01e9d7554975d3f04e7b63b5` |
| Release assets | 500 | `4cde2219cb027dd0814186781a4821ee5a45327cd3c8e548ece1bf1a6e58700b` |
| Workflow runs | 400 | `d30f8bc235e6fb736a29afb41f7175adf676819834e3c0c130f467e3d86192bc` |
| Deployments | 77 | `8b74b6d5b356696e244efdac506ced8fba7c0c43554eb7791d01f5349406506e` |
| Addressable workflows | 5 | `de6719c9da5f25baf5550778d5355364eede71714ca8baf8ea42b0c789145e48` |

T-078 must revalidate each live object against the frozen exact set before any
deletion. This report does not authorize deletion or settings changes.

## Remaining Gate

T-072 remains in progress pending one owner-authenticated, read-only review of
GitHub Apps, Packages, Projects v2, and Wiki state. Any content found must be
acquired and scanned; an inaccessible or ambiguous surface remains unresolved.
No other acquisition or scanner blocker remains.
