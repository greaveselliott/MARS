# Active P0 Exec Plan: Retire v0.93.0 And Adopt GoReleaser

**Status:** Active
**Priority:** P0
**Depends On:** Exact live `main`/tag/Release leases and operator-approved private rewrite authority
**Blocks:** F-017-S003 public release readiness and every later cutover claim
**Related Tickets:** T-063, T-064; scheduled T-065 through T-067
**Current Ticket:** T-064
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-018-goreleaser-distribution.md
**Related Feature Contracts:** F-001, F-009, F-017, F-018
**Hypothesis:** Retiring the private `v0.93.0` experiment and using a pinned GoReleaser archive pipeline will reduce bespoke release code while retaining fail-closed consumer verification.
**Success Evidence:** `v0.93.0` is absent; protected work survives; GoReleaser snapshots are reproducible; the archive/checksum/SBOM/signature contract passes; and no unsupported release is advertised.
**Falsification Evidence:** A lease drifts, protected work changes, v0.93 remains reachable through live refs/releases, legacy publisher behavior remains required, or a partial/tampered release can be accepted.
**Scenario Schedule:** T-064 retirement; T-065 GoReleaser producer; T-066 installer/updater migration; T-067 private rehearsal; later owner-approved `v0.69.0` cutover
**Current Failing Scenario:** F-017-S003: v0.93 is still the live private release and the supported archive/signature lifecycle is not implemented
**Walking Skeleton Slice:** Retire the unsupported private v0.93 lineage with exact leases and preserve a private, fail-closed v0.68.49 baseline before replacing release production or consumption.
**Learning Or MVP Outcome:** Establish a truthful private release floor and an explicit migration contract before changing release implementation or consumer behavior.
**Created:** 2026-07-21
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager packets

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including anonymous signed install/update, fork-safe contribution, logged-out cutover smoke, and a clean 48-hour canary.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** T-064 through T-067 and the independent F-017 audit, runtime, contribution, and cutover gates remain incomplete.
- **Next Primary Action:** Retire v0.93 with exact leases, then implement and privately rehearse the source-only GoReleaser contract without changing visibility.

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
| T-064 | Retire v0.93 and reset the release floor. | Live leases match the captured manifest. | Release/tag/commit are absent, Pages is disabled, protected work is re-anchored, `VERSION` is `0.68.49`, and the source fallback is `0.69.0-dev`. |
| T-065 / F-018-S001 | Replace the bespoke producer with pinned GoReleaser. | T-064 complete. | Four deterministic archives, notices, SPDX SBOMs, checksums, and signing configuration pass snapshot tests. |
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

## GoReleaser Contract

- Pin GoReleaser OSS `v2.17.0`, Go `1.26.5`, Syft `v1.44.0`, Cosign `v3.0.6`, and every GitHub Action by full commit SHA.
- Build `./cmd/mars` with `CGO_ENABLED=0`, `-trimpath`, full commit, tag version, commit date, and commit-derived archive timestamps for Darwin/Linux AMD64/ARM64.
- Produce four `mars_<version>_<os>_<arch>.tar.gz` archives. Each contains exactly `mars`, `LICENSE`, `NOTICE`, and `THIRD_PARTY_NOTICES`.
- Produce one SPDX-JSON SBOM per archive, `checksums.txt`, and a keyless `checksums.txt.sigstore.json` bundle.
- Release into a draft, fresh-download and compare every asset, then publish only after checksum/signature/attestation verification. A failure remains an unpublished draft.
- Enable GitHub release immutability before the first supported release; never move or clobber a published tag or asset.

## Validation Gates

- `git diff --check`, uncached tests, race, vet, coverage ratchets, fuzz smoke, fail-closed `govulncheck`, DocSync, docs/link checks, and four-platform CGO-disabled builds.
- `goreleaser check` and two clean snapshot builds with identical archive and checksum hashes.
- Positive and negative checksum, signature, identity, tag, commit, platform, archive traversal/link/device/duplicate/extra/missing/size tests.
- Fork-safe snapshot workflow with read-only token and no secrets; release workflow limited to contents, OIDC, and attestation permissions.
- Installed clean-project macOS and Linux lifecycle; v0.93 manual-reinstall path; no reachable custom publisher or legacy raw asset alias.
- Security, QA, Dogfood, Release Manager, and Orchestrator sign-off before advancing each ticket.

## No-Go Rules

- Stop on lease, visibility, branch, workflow, stash, worktree, candidate, or required-tool drift.
- Never blind-force, wildcard-delete, reuse `v0.93.0`, or silently select another version if GitHub rejects reuse of `v0.69.0`.
- Never describe an upload exit code, draft, checksum-only install, inaccessible surface, or snapshot as a supported release.
- Do not create the `v0.69.0` tag, expose keyless private metadata, change visibility, or announce MARS during this plan.
