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
last_attempt: "2026-07-22: v1.2.2 probe compiled the exact APIs; Go 1.25.8 retained eight called stdlib findings while Go 1.25.12 produced zero called findings"
blocker: "Owner-approved T-067 must establish the exact Go 1.25.12 source compatibility floor before sigstore-go v1.2.2 can be admitted"
blocked_by: [T-067]
trace_id: "TBD"
next_action: "Wait for T-067 to establish and validate the exact Go 1.25.12 source floor, then restore T-066 to in-progress and rerun A1 admission for sigstore-go v1.2.2."
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

A1. Add the production offline Sigstore-bundle and strict canonical-checksum verification primitive plus bounded synthetic positive and hostile table tests. Commit and push before archive work.

A2. Add bounded gzip/tar inspection and extraction. Reject absolute/traversal names, backslashes, links, devices/FIFOs, sparse or duplicate members, missing/extra members, multiple gzip streams or trailing data, and member/file-count/compressed/expanded-size quota violations. The archive contains exactly mars, LICENSE, NOTICE, and THIRD_PARTY_NOTICES with fixed regular-file modes. Commit and push before updater integration.

B. Integrate mars update tool: fetch bounded metadata, signature bundle, checksums, and only the selected archive; verify completely; stage in a same-filesystem 0700 directory with non-final files 0600; serialize updates; atomically replace only after durability checks; retain/restore the prior binary on every failure and give an actionable recovery command. Source update modes remain unchanged. Commit and push.

C. Migrate scripts/install.sh to the same trusted verifier path. If bootstrap cannot invoke the same verifier without weakening trust, make binary installation explicitly unavailable and direct users to source installation; checksum-only compatibility is forbidden. Commit and push.

D. Migrate or retire release verify-assets, release audit, CLI/MarsCLI, current docs, DocSync routes, and generated target guidance. Remove legacy raw asset names mars-<os>-<arch> and mars-harness-<os>-<arch> and legacy checksum-only semantics; update binary/cli command aliases are not raw-asset aliases. Commit and push.

E. Run focused hostile/race tests, full source/vulnerability/DocSync/cross-build gates, an installed commit-bound binary, a fresh target, and an offline synthetic signed update/rollback lifecycle. Close ticket, F-018-S002, and plan evidence in a separate commit. T-068 retains two-build and clean macOS/Linux pipeline rehearsal.

## Checkpoint A1 dependency admission — blocked 2026-07-22

- Official `sigstore-go v0.7.0` resolves to commit `9c466a8b8df6a1886292e0a82023bc217968da9e` with SumDB hashes `h1:bIGPc2IbnbxnzlqQcKlh1o96bxVJ4yRElpP1gHrOH48=` and `h1:4RrCK+i+jhx7lyOG2Vgef0/kFLbKlDI1hrioUYvkxxA=` for its module and go.mod.
- A disposable module running the exact Go 1.22.4 toolchain retained `go 1.22.4` after `go mod tidy -go=1.22.4`. The exact production wrapper compiled. A valid v0.3 fixture independently passed offline bundle, trusted-root, transparency-log, integrated-timestamp, and identity verification, while a focused extension test rejected a mismatched source digest; the exact-artifact-byte and SCT paths were compile-proven rather than accepted as release evidence.
- Exact called-path `govulncheck v1.3.0` reported 14 findings: `GO-2025-4192`, `GO-2026-4316`, `GO-2026-4348`, `GO-2026-4349`, `GO-2026-4354`, `GO-2026-4355`, `GO-2026-4377`, `GO-2026-4945`, `GO-2026-5547`, `GO-2026-5763`, `GO-2026-5778`, `GO-2026-5851`, `GO-2026-5932`, and `GO-2026-5952`. The decisive direct trace reaches the `GO-2026-5952` multi-log verification-threshold bypass through `SignedEntityVerifier.Verify`; upstream marks it fixed in `v1.2.0`.
- `sigstore-go v1.2.0` requires Go 1.25.0, while MARS's approved minimum remains Go 1.22.4. Raising that floor is a separate compatibility decision and is not authorized by T-066. Security and the Orchestrator therefore reject v0.7.0, any custom certificate/crypto parser, and any implicit Go-floor change.
- A second bounded probe evaluated the newest official fixed release, `sigstore-go v1.2.2` at commit `55aa6240784677449a564e66a0fca7a6a3605ecd`, with SumDB hashes `h1:xAJ8hxaoecC0HKBYVbrwUjkeAI+GJYu6vLqbxDlD2Q0=` and `h1:MIFwBxAHJD+/lKgZzt9n/4Zhq/3T2+EuGX8iGrIsZgU=`. Its declared compatibility floor is Go 1.25.8 and the exact T-066 wrapper compiles there.
- Go 1.25.8 is not an acceptable MARS security floor: exact called-path scanning found eight reachable standard-library findings (`GO-2026-5856`, `GO-2026-5039`, `GO-2026-5037`, `GO-2026-4971`, `GO-2026-4947`, `GO-2026-4946`, `GO-2026-4870`, and `GO-2026-4865`). The identical wrapper built with Go 1.25.12 produced zero called findings. Residual `GO-2026-5970` and `GO-2026-5932` were uncalled or module-only on this path and remain subject to the whole-source gate after dependency admission.
- The proposed compatibility ticket must set `go 1.25.12`, retain release `toolchain go1.26.5`, add an exact minimum-toolchain lane, reject Go 1.25.11 without auto-download, and make `doctor` patch-aware for explicit MARS source builds while keeping packaged-binary and target users healthy without Go. Generated target repositories do not inherit this source-only floor.
- The current public-good root candidate was independently identified at `sigstore/root-signing` commit `c9bda74ad2221f938f7d2e0295ca3aad2da710a8`, SHA-256 `6494e21ea73fa7ee769f85f57d5a3e6a08725eae1e38c755fc3517c9e6bc0b66`, and parsed fully offline with v0.7.0. It was not added because no verifier dependency or compatibility-floor change is authorized yet.
- No MARS dependency, verifier source, tag, Release, signature, upload, version, or visibility change occurred. A2 and every later T-066 checkpoint remain unstarted.

## Interfaces and blast radius

internal/selfupdate, internal/release retained consumer verification/audit, scripts/install.sh, update/release CLI output, MarsCLI/tool guidance, directly affected live docs and generated target doctrine, plus focused tests. Production verification must be shared rather than duplicated in shell. Network reads are HTTPS-only, size/time/redirect bounded, and credentials may be sent only to the exact approved GitHub API origin and must be stripped on redirects.

## Acceptance criteria

All authenticity, checksum, canonical-name, tag, commit, platform, toolchain, metadata, and archive checks occur before replacement. Negative lanes cover tampered/missing/extra/duplicate records or assets, unsigned/invalid/wrong-identity bundles, wrong tag/commit/platform, hostile archives, oversized/truncated responses, partial/cancelled downloads, redirect credential stripping, implicit/latest downgrade or tag/commit replay, concurrent updates, unsafe destination types, and injected failures before and after rename. An explicit older-version request may perform a rollback only when its signature, tag, commit, platform, and archive all verify. Every failure leaves or restores the prior binary byte-identical and reports an actionable recovery command. Verification errors never echo hostile URLs, member names, certificate fields, response bodies, tokens, or candidate bytes.

## Non-goals and authority boundary

Do not modify the GoReleaser producer, create a production signing workflow, fetch credentials/OIDC in tests, change VERSION/CHANGELOG, create or move a tag, create/upload/sign/publish a GitHub Release, enable Pages, change repository visibility, announce support, or claim a real release passed. Keep VERSION 0.68.49, source fallback 0.69.0-dev, repository private, and Primary Status primary_blocked. Two-build private rehearsal and clean macOS/Linux synthetic-fixture lifecycle remain T-068 work. Anonymous release access, actual keyless signing/publication, public canary, and every visibility action remain solely the separately approved F-018-S004/F-017 cutover.

## Planning tool note

The first `ticket_create` attempt under the CTO-weekly role reached its stricter first-proof heuristic because that heuristic recognizes build-and-smoke tokens only in ticket frontmatter rather than T-065's full body and plan evidence. The Orchestrator recreated this same bounded ticket through unchanged `ticket_create` as `foundation-maintainer`, retaining the normal feature-contract and earliest-uncovered-scenario gates. No ticket-policy source change is part of T-066; revisit the heuristic only if the limitation recurs.
