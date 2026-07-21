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
last_attempt: "2026-07-21: ticket_create accepted the F-018-S001 implementation slice"
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

Pin and verify GoReleaser and its supporting supply-chain tools; add the four-platform CGO-disabled snapshot archive contract, licenses/notices, SPDX SBOMs, checksums, deterministic metadata, and fork-safe read-only snapshot workflow; structurally validate the deferred signing configuration; remove the bespoke upload/publish path only after replacement snapshot tests pass; retain no tag, GitHub Release, public keyless signature, or supported-release claim. Generated target repositories must not inherit the Go-specific producer. Support the 0.69.0-dev source fallback in the non-dry-run release-note writer before cutover.

## Interfaces And Blast Radius

Release configuration and workflows, release CLI producer surfaces, source-only release doctrine, generated-target boundaries, artifact validation scripts, notices, tests, and documentation synchronization. Installer and updater consumption remain T-066.

## Acceptance criteria

Pinned goreleaser check passes; two clean snapshot builds for Darwin/Linux AMD64/ARM64 are byte-reproducible; archives contain only mars, LICENSE, NOTICE, and THIRD_PARTY_NOTICES; each archive has an SPDX-JSON SBOM and checksums entry; binary metadata binds the explicit snapshot version, full commit, platform, toolchain, and commit-derived date; raw binaries, aliases, stale, missing, duplicate, and extra outputs fail; the signing configuration passes structural and synthetic-fixture tests without public signing; fork-safe CI has read-only permissions and no secrets; bespoke publication is unreachable; full source, race, vulnerability, fuzz, DocSync, and cross-build gates pass; repository remains private and primary_blocked.
