---
id: T-066
title: Migrate release consumers to fail-closed signed archives
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-018-S002"]
end_to_end_evidence: required
evidence_links: ["docs/features/F-018-goreleaser-distribution.md#f-018-s002-safe-archive-installation-and-binary-updater", "docs/exec-plans/active/current-operating-plan.md"]
verified_by: "TBD"
owner: "engineer"
last_attempt: "2026-07-22: claimed for checkpoint A implementation after the planning handoff was pushed at 1257534"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Checkpoint A: implement the pure offline signature/checksum/archive verification primitive and hostile synthetic fixtures; commit and push before updater integration."
dedupe_key: "release:signed-archive-consumers"
metadata:
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  supports: "F-018-S002"
source: current-operating-plan.md — T-066 / F-018-S002
created: 2026-07-22
depends_on: [T-065]
---

# T-066: Migrate release consumers to fail-closed signed archives

## Context

T-065 froze the private GoReleaser producer contract and retired the bespoke publisher. Release-mode consumers still select legacy raw binaries, trust unsigned checksums, and install without authenticated archive, workflow-identity, metadata, or hostile-extraction checks. T-066 migrates those consumers while the repository remains private and no real MARS release is created.

## Locked consumer and trust contract

The supported artifact set is the T-065 nine-file set plus one signature bundle: four canonical mars_<version>_<os>_<arch>.tar.gz archives, four matching .sbom.json files, canonical checksums.txt with exactly the eight archive/SBOM records, and checksums.txt.sigstore.json.

checksums.txt.sigstore.json is a Sigstore bundle JSON v0.3 over the exact raw bytes of checksums.txt. Verification is offline against a repository-pinned Sigstore trusted-root document and pinned SHA-256; runtime trust-root or identity overrides are forbidden. The accepted OIDC issuer is https://token.actions.githubusercontent.com. The certificate identity is exactly https://github.com/greaveselliott/MARS/.github/workflows/release.yml@refs/tags/v<semver>. In addition to issuer, repository, workflow, and tag-ref checks, the Fulcio/Sigstore GitHub workflow SHA certificate extension must equal the expected full release commit, and the archive binary must report that same full commit. The selected vulnerability-reviewed sigstore-go API must expose and verify this claim; if it cannot, implementation fails closed and records the blocker rather than adding custom certificate or cryptographic parsing. The bundle must also contain a valid transparency-log inclusion proof. Missing, malformed, expired-at-signing, wrong-root, wrong-issuer, wrong-workflow, wrong-repository, wrong-ref, wrong-commit, or unverifiable bundle evidence fails closed. T-066 uses only ephemeral offline synthetic bundles; it does not sign MARS artifacts.

The selected archive name is exact for the normalized release version and current GOOS/GOARCH. Before any candidate execution or destination mutation, verify the signature bundle, strict checksum grammar and membership, selected archive digest, exact archive structure, and the extracted binary's debug/buildinfo: GOOS, GOARCH, pinned Go toolchain policy, `CGO_ENABLED=0`, full vcs.revision, vcs.modified=false, and a valid vcs.time. T-065's producer proof owns the commit-derived-time guarantee; T-066 must not claim the signed checksum bundle authenticates a timestamp it does not contain.

## Checkpoint sequence

A. Add one production offline verification/extraction primitive plus bounded synthetic positive and hostile table tests. Reject absolute/traversal names, backslashes, links, devices/FIFOs, sparse or duplicate members, missing/extra members, multiple gzip streams or trailing data, and member/file-count/compressed/expanded-size quota violations. The archive contains exactly mars, LICENSE, NOTICE, and THIRD_PARTY_NOTICES with fixed regular-file modes. Commit and push.

B. Integrate mars update tool: fetch bounded metadata, signature bundle, checksums, and only the selected archive; verify completely; stage in a same-filesystem 0700 directory with non-final files 0600; serialize updates; atomically replace only after durability checks; retain/restore the prior binary on every failure and give an actionable recovery command. Source update modes remain unchanged. Commit and push.

C. Migrate scripts/install.sh to the same trusted verifier path. If bootstrap cannot invoke the same verifier without weakening trust, make binary installation explicitly unavailable and direct users to source installation; checksum-only compatibility is forbidden. Commit and push.

D. Migrate or retire release verify-assets, release audit, CLI/MarsCLI, current docs, DocSync routes, and generated target guidance. Remove legacy raw asset names mars-<os>-<arch> and mars-harness-<os>-<arch> and legacy checksum-only semantics; update binary/cli command aliases are not raw-asset aliases. Commit and push.

E. Run focused hostile/race tests, full source/vulnerability/DocSync/cross-build gates, an installed commit-bound binary, a fresh target, and an offline synthetic signed update/rollback lifecycle. Close ticket, F-018-S002, and plan evidence in a separate commit. T-067 retains two-build and clean macOS/Linux pipeline rehearsal.

## Interfaces and blast radius

internal/selfupdate, internal/release retained consumer verification/audit, scripts/install.sh, update/release CLI output, MarsCLI/tool guidance, directly affected live docs and generated target doctrine, plus focused tests. Production verification must be shared rather than duplicated in shell. Network reads are HTTPS-only, size/time/redirect bounded, and credentials may be sent only to the exact approved GitHub API origin and must be stripped on redirects.

## Acceptance criteria

All authenticity, checksum, canonical-name, tag, commit, platform, toolchain, metadata, and archive checks occur before replacement. Negative lanes cover tampered/missing/extra/duplicate records or assets, unsigned/invalid/wrong-identity bundles, wrong tag/commit/platform, hostile archives, oversized/truncated responses, partial/cancelled downloads, redirect credential stripping, implicit/latest downgrade or tag/commit replay, concurrent updates, unsafe destination types, and injected failures before and after rename. An explicit older-version request may perform a rollback only when its signature, tag, commit, platform, and archive all verify. Every failure leaves or restores the prior binary byte-identical and reports an actionable recovery command. Verification errors never echo hostile URLs, member names, certificate fields, response bodies, tokens, or candidate bytes.

## Non-goals and authority boundary

Do not modify the GoReleaser producer, create a production signing workflow, fetch credentials/OIDC in tests, change VERSION/CHANGELOG, create or move a tag, create/upload/sign/publish a GitHub Release, enable Pages, change repository visibility, announce support, or claim a real release passed. Keep VERSION 0.68.49, source fallback 0.69.0-dev, repository private, and Primary Status primary_blocked. Two-build private rehearsal and clean macOS/Linux synthetic-fixture lifecycle remain T-067 work. Anonymous release access, actual keyless signing/publication, public canary, and every visibility action remain solely the separately approved F-018-S004/F-017 cutover.

## Planning tool note

The first `ticket_create` attempt under the CTO-weekly role reached its stricter first-proof heuristic because that heuristic recognizes build-and-smoke tokens only in ticket frontmatter rather than T-065's full body and plan evidence. The Orchestrator recreated this same bounded ticket through unchanged `ticket_create` as `foundation-maintainer`, retaining the normal feature-contract and earliest-uncovered-scenario gates. No ticket-policy source change is part of T-066; revisit the heuristic only if the limitation recurs.
