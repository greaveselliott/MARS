---
id: T-065
title: Adopt pinned GoReleaser archive production
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-018-S001"]
end_to_end_evidence: required
evidence_links: ["docs/features/F-018-goreleaser-distribution.md#f-018-s001-deterministic-release-production"]
verified_by: "QA, Security, Release Manager, foundation-maintainer"
owner: "engineer"
last_attempt: "2026-07-21: checkpoint-A reviews accepted the bounded producer after Syft/config corrections; exact GoReleaser v2.17.0 binary findings remain a public-cutover blocker"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Verify upstream pins, implement the bounded snapshot producer, and remove bespoke publication only after replacement tests pass."
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
