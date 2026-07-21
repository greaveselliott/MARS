# Active P0 Exec Plan: Retire v0.93.0 And Adopt GoReleaser

**Status:** Active
**Priority:** P0
**Depends On:** Exact live `main`/tag/Release leases and operator-approved private rewrite authority
**Blocks:** F-017-S003 public release readiness and every later cutover claim
**Related Tickets:** T-063, T-064; scheduled T-065 through T-067
**Current Ticket:** T-065
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-018-goreleaser-distribution.md
**Related Feature Contracts:** F-001, F-009, F-017, F-018
**Hypothesis:** Retiring the private `v0.93.0` experiment and using a pinned GoReleaser archive pipeline will reduce bespoke release code while retaining fail-closed consumer verification.
**Success Evidence:** `v0.93.0` is absent; protected work survives; GoReleaser archives are reproducible; each snapshot's exact archive/SBOM checksum contract passes; and no unsupported release is advertised.
**Falsification Evidence:** A lease drifts, protected work changes, v0.93 remains reachable through live refs/releases, legacy publisher behavior remains required, or a partial/tampered release can be accepted.
**Scenario Schedule:** T-064 retirement; T-065 GoReleaser producer; T-066 installer/updater migration; T-067 private rehearsal; later owner-approved `v0.69.0` cutover
**Current Failing Scenario:** F-018-S001: the retained private baseline has no pinned, publication-disabled GoReleaser archive producer or verified snapshot contract.
**Walking Skeleton Slice:** Build four private snapshot archives with deterministic binary/archive metadata, exact notices, per-archive SPDX SBOMs, and an exact self-verifying checksum set without creating a tag, Release, signature, or supported-release claim.
**Learning Or MVP Outcome:** Establish a truthful private release floor and an explicit migration contract before changing release implementation or consumer behavior.
**Created:** 2026-07-21
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager packets

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including anonymous signed install/update, fork-safe contribution, logged-out cutover smoke, and a clean 48-hour canary.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** T-065 through T-067 and the independent F-017 audit, runtime, contribution, and cutover gates remain incomplete.
- **Next Primary Action:** Implement T-065 as the pinned source-only GoReleaser snapshot producer without creating a tag, Release, public signature, or supported-release claim.

## Locked Decisions

1. Remove `v0.93.0` Release, tag, release-note commit, and live evidence claims completely; never reuse `v0.93.0`.
2. Reset the retained release floor to `v0.68.49` and prepare `v0.69.0` as the first GoReleaser release, accepting that the version name was previously used privately and may survive in old caches.
3. Use a clean archive migration. Existing v0.93 clients receive no raw-asset compatibility bridge and require one manual reinstall.
4. GoReleaser is source-only. Generated target harnesses use project-owned release tooling rather than inheriting a Go-specific publisher.
5. Private validation uses snapshots only. The `v0.69.0` tag and keyless public signature wait for the separately approved cutover.
6. Keep the repository private and `primary_blocked` throughout this plan.

## Operator-Authorized Release Transition Exception

As of 2026-07-21, AGENTS rules 12 and 14 describe the steady-state release process that depends on `mars release publish-assets`. The operator has explicitly approved retiring that command and deferring the next tag until the public cutover. Therefore T-064 through T-067 use this bounded source-only exception:

- commit and push each validated ticket normally, but do not create a private support release or legacy assets;
- keep `VERSION` and the changelog at `0.68.49` through T-067 while untagged source builds report `0.69.0-dev`;
- require T-067 release-note dry-run evidence to select exactly `0.69.0`, but leave the actual release-note commit, tag, and assets to the later cutover ticket;
- restore rules 12 and 14 to the GoReleaser steady-state workflow in the same migration before the exception expires at cutover.

This exception cannot authorize a supported-release claim, visibility change, or unrelated unversioned source work.

## Ticket Sequence

| Ticket | Outcome | Entry gate | Exit gate |
| --- | --- | --- | --- |
| T-064 | Retire v0.93 and reset the release floor. | Live leases match the captured manifest. | Passed 2026-07-21: Release/tag are absent, Pages is disabled, protected work is re-anchored, `VERSION` is `0.68.49`, and the source fallback is `0.69.0-dev`. |
| T-065 / F-018-S001 | Replace the bespoke producer with pinned GoReleaser. | T-064 complete. | Four deterministic archives, provisional private notices, SPDX SBOMs, exact run-local checksums, and a publication-disabled workflow pass snapshot tests. |
| T-066 / F-018-S002 | Migrate installer, updater, audit, CLI/docs, and generated guidance. | T-065 complete and contract frozen. | Consumers reject tampering and unsafe archives; legacy publish/verify commands and raw aliases are absent. |
| T-067 / F-018-S003 | Rehearse privately and prepare `0.69.0`. | T-066 complete. | Two-build reproducibility, clean macOS/Linux installs, workflow draft-failure behavior, and release-note preparation pass. |
| Cutover | Publish immutable `v0.69.0`. | Every F-017 prerequisite and separate owner approval pass. | Logged-out verification passes and the 48-hour canary starts. |

## T-064 Exact Retirement Transaction

1. Freeze writes; require private visibility, sole branch `main`, no active Actions run, and exact `bf3034863691443962ec251e62c8dec1ee5138fb` branch/tag leases.
2. Construct the replacement from `6d326ffa82e236570509b3783711b87911e5500d`; reset the release floor in `VERSION` to `0.68.49`, set the untagged source fallback to `0.69.0-dev`, remove v0.93 claims, and preserve T-061 source until T-065 replaces it.
3. Validate the replacement commit, disable the Pages site observed public on 2026-07-21, and verify logged-out Pages access is unavailable before exposing rewritten content.
4. Force-push only `main` with the captured branch lease; keep Pages disabled until the separately approved public cutover.
5. Delete numeric Release ID `354800199` and independently require tag lookup and Release ID lookup to return absent; then delete only remote `v0.93.0` with its exact OID lease.
6. Delete the retired Pages run, expired artifact, and deletable deployment without enabling a replacement deployment.
7. Re-anchor both protected stashes on rewritten `main`, preserving messages, file hashes, and staged/unstaged/untracked classifications. Leave the unrelated dashboard worktree untouched.
8. Replace the installed v0.93 binary with a verified source build before deleting exact local v0.93 artifacts.
9. Record actual counts and residual SHA-only hosting metadata without claiming physical erasure from caches or backups.

## T-064 Retirement Evidence — 2026-07-21

- Replacement commit `cf62513ea9a2e83e60e3bd74085191a2e977d74f` was built from `6d326ffa82e236570509b3783711b87911e5500d`, passed `make check`, DocSync, release-note dry-run, and four CGO-disabled target builds, then replaced remote `main` with the exact `bf3034863691443962ec251e62c8dec1ee5138fb` branch lease.
- GitHub remained private with sole branch `main`; Pages configuration changed from public/HTTP 200 to API 404 and logged-out HTTP 404 before the branch rewrite and remained unavailable afterward.
- Release ID `354800199` had exactly nine uploaded assets before deletion, returned HTTP 204 on deletion, and both numeric-ID and tag lookups returned 404 afterward. The exact `v0.93.0` tag lease at `bf3034863691443962ec251e62c8dec1ee5138fb` was then deleted; final counts are 301 tags and 56 Releases.
- Retired Pages artifact `8360983372`, run `29460583182`, and deployment `5465865045` were deleted and independently returned 404. Deployment status `15767000543` recorded `inactive` before deletion.
- Re-anchored stash `1592dd2628c7a7bb62f17be36e43712e0e27c0d0` preserves the T-060 tracked diff hash `a22bb0d367a0dc975a2dbdc101205438997a0f090a2375f037394b19cdab4434` and two untracked file hashes. Re-anchored stash `6ebc8f38320753b00787658c6a41c30bc54e30d4` preserves cognition diff hash `b596088de5d6b9cca5a8c39bf0f915ce3f5b70d24a2e3c712066de0ec4c9ff38`; both use `cf62513` as parent with their original messages and classifications.
- The unrelated dashboard worktree remains at `404c047e3cdfc39af29c3d630b558ad5f82b8709` with unchanged status fingerprint `96fd2019657d2c7a9e3b50c9f20e526df160c1e95cff1a6161543b5005f381a9`.
- As of 2026-07-21, installed MARS reports `0.69.0-dev.cf62513` with full commit metadata and matches the reviewed binary SHA-256 `0ae11dc259073c9cab48f5d4b32c0dd7f56c197ada32c0370d633ea3fd9a94e5`. This is a development identity, not a supported release.
- GitHub may retain provider-side SHA metadata, backups, logs, or caches without a deletion API; this evidence records logical absence and does not claim physical erasure.

## GoReleaser Contract

- Build GoReleaser OSS `v2.17.0` and Syft `v1.49.0` from exact SumDB-verified modules with Go `1.26.5`; pin every GitHub Action by full commit SHA. Exact binary scanning records GO-2026-5970 and no-fix GO-2026-5932 through GoReleaser's dormant `ko`/Sigstore/Rekor dependency chain, while Syft has no called-symbol finding. The exact, unmodified GoReleaser module may run only credential-free, publication-disabled private snapshots with `ko`, signing, announcement, and publication explicitly skipped; it is not cleared for public cutover, and T-065 must not hide the findings through an unpublished dependency overlay. Cosign `v3.1.2` is reserved for the later approved signing/cutover work and is not executed by the default snapshot producer.
- Build `./cmd/mars` with `CGO_ENABLED=0`, `-trimpath`, full commit, tag version, commit date, and commit-derived archive timestamps for Darwin/Linux AMD64/ARM64.
- Produce four `mars_<version>_<os>_<arch>.tar.gz` archives. Each contains exactly `mars`, `LICENSE`, `NOTICE`, and `THIRD_PARTY_NOTICES`.
- Produce one SPDX-JSON SBOM per archive and `checksums.txt` containing exactly four archive and four SBOM entries. T-065 creates no signature bundle.
- Keep the default producer publication-disabled. Draft creation, fresh-download comparison, signing, attestation, and immutable publication remain later cutover work.
- Enable GitHub release immutability before the first supported release; never move or clobber a published tag or asset.

## Validation Gates

- `git diff --check`, uncached tests, race, vet, coverage ratchets, fuzz smoke, fail-closed MARS-source `govulncheck`, DocSync, docs/link checks, and four-platform CGO-disabled builds. Third-party producer binary findings must be recorded and remain a public-cutover no-go until an acceptable upstream tool release/removal exists.
- `goreleaser check` and two clean snapshot builds with identical archive hashes, identical archive checksum records, exact eight-entry run-local checksum verification, and normalized SPDX semantic equality. Raw SPDX creation time and document namespace are explicitly volatile and are not claimed byte-reproducible.
- Positive and negative checksum, identity, commit, platform, archive traversal/link/device/duplicate/extra/missing/size tests. Signature and tag verification belong to T-066 and the cutover gate.
- Fork-safe snapshot workflow with read-only token and no secrets; release workflow limited to contents, OIDC, and attestation permissions.
- Installed clean-project macOS and Linux lifecycle; v0.93 manual-reinstall path; no reachable custom publisher or legacy raw asset alias.
- Security, QA, Dogfood, Release Manager, and Orchestrator sign-off before advancing each ticket.

## No-Go Rules

- Stop on lease, visibility, branch, workflow, stash, worktree, candidate, or required-tool drift.
- Never blind-force, wildcard-delete, reuse `v0.93.0`, or silently select another version if GitHub rejects reuse of `v0.69.0`.
- Never describe an upload exit code, draft, checksum-only install, inaccessible surface, or snapshot as a supported release.
- Do not create the `v0.69.0` tag, expose keyless private metadata, change visibility, or announce MARS during this plan.
