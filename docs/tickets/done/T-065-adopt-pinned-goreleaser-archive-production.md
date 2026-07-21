---
id: T-065
title: Adopt pinned GoReleaser archive production
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-018-S001"]
end_to_end_evidence: required
evidence_links: ["docs/features/F-018-goreleaser-distribution.md#f-018-s001-deterministic-release-production", "docs/exec-plans/active/current-operating-plan.md#t-065-checkpoint-evidence--2026-07-21", "git show bb1b79b7aa5787cea3355a2e592d9bfe4a0d2849"]
verified_by: "QA, Security, Dogfood, Release Manager, foundation-maintainer as Orchestrator"
owner: "engineer"
last_attempt: "2026-07-21: final bb1b79b snapshot, source gate, installed-binary, clean-target, and state-guard evidence passed"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "None; T-065 is complete. Create T-066 only through ticket_create."
dedupe_key: "release:goreleaser-snapshot-producer"
metadata:
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  supports: "F-018-S001"
source: Owner-approved GoReleaser migration after T-064
created: 2026-07-21
depends_on: [T-064]
---

# T-065: Adopt pinned GoReleaser archive production

## Context

T-064 retired the unsupported private v0.93 release and restored a truthful private release floor. MARS needs a conventional source-only producer before consumers or public cutover work can proceed.

## Requirements

Build exact GoReleaser and Syft modules under the pinned Go toolchain; add the four-platform CGO-disabled snapshot archive contract, provisional private notices, SPDX SBOMs, checksums, deterministic metadata, and fork-safe read-only snapshot workflow; keep signing and publication absent from the default producer; remove the bespoke upload/publish path only after replacement snapshot tests pass; retain no tag, GitHub Release, public keyless signature, or supported-release claim. Generated target repositories must not inherit the Go-specific producer. Support the 0.69.0-dev source fallback in the non-dry-run release-note writer before cutover.

## Interfaces And Blast Radius

Release configuration and workflows, release CLI producer surfaces, source-only release doctrine, generated-target boundaries, artifact validation scripts, notices, tests, and documentation synchronization. Installer and updater consumption remain T-066.

## Acceptance criteria

Pinned goreleaser check passes; two clean snapshot builds for Darwin/Linux AMD64/ARM64 have byte-identical archives and archive checksum records; each run's exact eight-entry checksum file verifies its four archives and four SPDX-JSON SBOMs; normalized SBOM semantics match after excluding only documented volatile SPDX creation/namespace fields; archives contain only mars, LICENSE, NOTICE, and THIRD_PARTY_NOTICES; binary metadata binds the explicit snapshot version, full commit, platform, toolchain, and commit-derived date; raw binaries are excluded from the publishable set and aliases, stale, missing, duplicate, and extra outputs fail; the default producer has no signing/publication authority; fork-safe CI has read-only permissions and no secrets; bespoke publication is unreachable; MARS source vulnerability, race, fuzz, DocSync, and cross-build gates pass; repository remains private and primary_blocked. Complete Go dependency notice text and the exact GoReleaser producer findings recorded below remain explicit pre-cutover blockers rather than being replaced by the SBOM or a snapshot pass.

## Third-party producer security disposition

The SumDB-built GoReleaser `v2.17.0` binary produced with Go `1.26.5` has two exact `govulncheck -mode=binary` findings: GO-2026-5970 through `golang.org/x/text@v0.38.0` (fixed upstream after the tag) and GO-2026-5932 through the compiled `golang.org/x/crypto/openpgp@v0.53.0` dependency chain used by the dormant `ko`/Sigstore/Rekor path (no fixed version). Syft `v1.49.0` has no called-symbol finding. T-065 does not fork, overlay, or silently repack GoReleaser: the exact official module remains restricted to credential-free, publication-disabled private snapshots with `ko`, signing, announcement, and publication explicitly skipped, while Syft update checks, enrichment, and remote license lookup are disabled. No OpenPGP input is parsed by this archive-only path. This expiring, risk-calibrated exception may prove the private archive producer contract, but it cannot satisfy the public cutover vulnerability gate or authorize `ko`. A later ticket must move to an upstream release/removal with an acceptable scan before any supported release or signing workflow is authorized.

## Checkpoint B evidence — immutable clean-root proof

At pushed commit `6a68eccf30036ab2fa84474afb85f7ee113c6ed9`, two independent clean clones produced snapshot `0.69.0-dev.6a68ecc` with Go `1.26.5`. The committed environment verifier passed while comparing the two outputs. Each publishable set contained exactly four archives, four matching SPDX-JSON SBOMs, and `checksums.txt`; each checksum file had exactly eight verified entries. The archive SHA-256 values matched across both roots: Darwin/AMD64 `0583adeb071a167726b17d4edbeaaf26e3e4ac5bb36f6cea09cdc4069594cf71`, Darwin/ARM64 `b417d6e2667698d98a37ea55e8f097c823a2dc1d7dc10b4ab790ccce224f5cec`, Linux/AMD64 `3478e0a1f0c75a76701899eeb27312cadfe348c2445471629cb7232755910853`, and Linux/ARM64 `94c7ab7ba772d73dffb452d56f33eeb7a942063a363cc483dd4177ccbd1e8450`.

The verifier statically inspected build metadata for all four binaries and executed only the native Darwin/ARM64 binary for linked version, full-commit, and commit-date evidence. Raw SBOM and whole-checksum-file hashes differed only as permitted by Syft's generated SPDX timestamp and namespace; normalized SBOM semantics matched, and each run's checksum file bound its exact raw SBOM bytes. QA, Security, and Release Manager reviews are GO for this trusted, publication-disabled producer proof. No tag, Release, signature, upload, visibility change, or supported-release claim occurred. Checkpoint C must still make the bespoke producer unreachable and run the remaining T-065 gates before F-018-S001 can pass.

## Final checkpoint evidence

Pushed commit `bb1b79b7aa5787cea3355a2e592d9bfe4a0d2849` removes the bespoke producer and GitHub upload/reconciliation implementation while retaining the fail-closed verifier, audit, and updater consumers for T-066. The compiled root CLI rejects `mars release publish-assets` as unknown; source release authority remains publication-disabled; generated targets receive repository-owned producer guidance without MARS GoReleaser configuration. AD-059 and AD-140 remain verbatim historical evidence under dated supersession notes.

The exact final commit passed `make check`: CGO-disabled build, uncached race suite, 73.9% coverage with every ratchet met, fail-closed `govulncheck` with zero called vulnerabilities, four fuzz-smoke lanes, and vet fallback. DocSync checked 345 files with zero findings. Four explicit CGO-disabled Darwin/Linux AMD64/ARM64 builds passed. A distinct clean clone ran exact GoReleaser `v2.17.0` and Syft `v1.49.0` under Go `1.26.5`, produced snapshot `0.69.0-dev.bb1b79b`, and passed `TestVerifyGoReleaserSnapshotDistFromEnvironment`; this final-commit run composes with the accepted two-clean-root reproducibility proof above.

The installed commit-bound Darwin/ARM64 binary reports full commit `bb1b79b7aa5787cea3355a2e592d9bfe4a0d2849`, clean VCS metadata, CGO disabled, and SHA-256 `fd8367e06e84203a7d971b00ef7b6b97fbb486756f85eb9d7d33909a20173fe2`; its release help omits the retired command and direct invocation exits 1 as unknown. A fresh initialized target committed a clean scaffold with producer-neutral guidance and no `publish-assets` or source GoReleaser injection. The source release-note dry run selects exactly `0.69.0` without writing files. Repository state remains private, `VERSION` remains `0.68.49`, the source fallback remains `0.69.0-dev`, and `v0.69.0` has no live tag or Release. QA, Security, Dogfood, Release Manager, and Orchestrator classify F-018-S001 as passed private producer evidence only; provisional dependency notices and GO-2026-5970/GO-2026-5932 remain public-cutover no-go items.
