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
last_attempt: "2026-07-22: C pushed at f45d5d2; unsafe shell binary bootstrap now fails closed to an independently reviewed source checkout, with hostile-environment tests and Engineer/QA/Security GO"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Implement only checkpoint D: migrate or retire retained release verification/audit, CLI/MarsCLI, live docs, DocSync routes, and generated target guidance; remove legacy raw asset names and checksum-only semantics without adding a publisher or installer."
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

checksums.txt.sigstore.json is a Sigstore bundle JSON v0.3 over the exact raw bytes of checksums.txt. Verification is offline against a repository-pinned Sigstore trusted-root document and pinned SHA-256; runtime trust-root or identity overrides are forbidden. The accepted OIDC issuer is https://token.actions.githubusercontent.com. The certificate identity is exactly https://github.com/greaveselliott/MARS/.github/workflows/release.yml@refs/tags/v<semver>. In addition to issuer, repository, workflow, and tag-ref checks, the Fulcio/Sigstore GitHub workflow SHA certificate extension must equal the expected full release commit, and the archive binary must report that same full commit. The selected vulnerability-reviewed sigstore-go API must expose and verify this claim; if it cannot, implementation fails closed and records the blocker rather than adding custom certificate or cryptographic parsing. The bundle must also contain a valid transparency-log inclusion proof. Missing, malformed, expired-at-signing, wrong-root, wrong-issuer, wrong-workflow, wrong-repository, wrong-ref, wrong-commit, or unverifiable bundle evidence fails closed. Tests may use synthetic bundles or an immutable upstream public signature vector offline; neither constitutes a MARS signature or release.

The selected archive name is exact for the normalized release version and current GOOS/GOARCH. Before any candidate execution or destination mutation, verify the signature bundle, strict checksum grammar and membership, selected archive digest, exact archive structure, and the extracted binary's debug/buildinfo: GOOS, GOARCH, pinned Go toolchain policy, `CGO_ENABLED=0`, full vcs.revision, vcs.modified=false, and a valid vcs.time. T-065's producer proof owns the commit-derived-time guarantee; T-066 must not claim the signed checksum bundle authenticates a timestamp it does not contain.

## Checkpoint sequence

A1. Passed at `fcf7397`: production offline Sigstore-bundle and strict canonical-checksum verification plus bounded upstream-positive and hostile tests. Committed and pushed before archive work.

A2. Passed at `b824b91`: bounded in-memory gzip/tar inspection and extraction rejects unsafe or noncanonical archive content and verifies exact release build metadata before returning cloned binary bytes. Committed and pushed before updater integration.

B1. Passed at `f3ed495`: bounded credential-safe acquisition enforces exact immutable release/tag/commit and ten-asset reconciliation, anonymous-first GitHub access, manually constrained redirects, strict response quotas, A1 then A2 verification, a final remote drift recheck, and downgrade/replay rejection. It returns only a private cloned candidate and performs no filesystem writes.

B2. Passed at `92d7ddd`: the network-free local transaction enforces descriptor-bound destination admission, a persistent nonblocking lock, same-filesystem 0700 staging, 0600 candidate/backup files, file and directory durability, atomic replacement, post-replace verification, and verified compensation or explicit recovery-required evidence. Committed and pushed before updater wiring.

B3. Passed at `683daf8`: release-mode `Run` now binds implicit latest to the exact running executable and digest, invokes B1 then B2 before PATH repair, preserves source/main and authority-free dry-run behavior, and removes download URLs from live plans/output. Committed and pushed before installer work.

C. Passed at `f45d5d2`: because shell bootstrap cannot invoke an independently trusted verifier without circular trust, binary installation is explicitly unavailable and `scripts/install.sh` fails closed to an independently reviewed exact source checkout. Checksum-only compatibility is absent. Committed and pushed before broader consumer migration.

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
- No MARS dependency, verifier source, tag, Release, signature, upload, version, or visibility change occurred. At that blocked admission checkpoint, A2 and every later T-066 checkpoint were unstarted.

## Checkpoint A1 completion — 2026-07-22

- Pushed commit `fcf7397` admits exact `sigstore-go v1.2.2` from upstream commit `55aa6240784677449a564e66a0fca7a6a3605ecd` with the recorded SumDB hashes and no replacement. The main module remains `go 1.25.12` with release `toolchain go1.26.5`.
- The embedded public-good root is the 6,787-byte `targets/trusted_root.json` from `sigstore/root-signing` commit `c9bda74ad2221f938f7d2e0295ca3aad2da710a8`, Git blob `effb0a19e6a0b3f69b3f0a2c72b5c2a02a0ddeea`, SHA-256 `6494e21ea73fa7ee769f85f57d5a3e6a08725eae1e38c755fc3517c9e6bc0b66`.
- Production now requires exact bundle v0.3, one transparency inclusion proof, observer time, SCT, raw checksum bytes, one exact non-regex GitHub Actions identity, modern source/build digests plus the required GitHub workflow SHA/repository/ref extensions, and exactly eight sorted archive/SBOM checksum records. Inputs are bounded and snapshotted; failures return fixed redacted errors. The primitive has no path, network, TUF/live-root, archive, updater, installer, signing, or publication surface.
- Positive offline conformance uses the public GoReleaser `v2.17.0` `checksums.txt` (SHA-256 `6fad0b6d…c921f`) and signature bundle as a pinned upstream test vector bound to commit `770a4fc7a8fb2dca874b6c98cb739dd64fc931c0`; it is not synthetic and is not MARS release evidence. Tampered bytes, wrong commit/root, missing proof, unsupported version, noncanonical checksum grammar, and hostile error content are rejected.
- Exact Go 1.25.12 full-source testing passed apart from one DocSync metadata omission, which was corrected and whose failed package plus DocSync/selfupdate reruns passed. Vet, focused race, module verification, exact-Go whole-source `govulncheck`, corrected Go 1.26.5 `make vuln`, and four Go 1.26.5 CGO-disabled cross-builds passed with zero called vulnerabilities. Engineer, QA, Security, and Orchestrator are GO.
- At the A1 checkpoint, A2 through E were unstarted. Dependency notice completion remained a public-cutover blocker. No MARS tag, Release, signature, upload, version, visibility, or supported-release claim changed.

## Checkpoint A2 completion — 2026-07-22

- Pushed commit `b824b915bb6424636bb9e09fd91c9f9501f259ac` adds a byte-only, offline archive verifier. It requires the A1 checksum result's exact tag/full-commit identity, derives the canonical platform archive name, authenticates its SHA-256 before decompression, and exposes only a cloned verified binary.
- The archive contract is exact and bounded: four ordered USTAR regular files, fixed root ownership and modes, no links/devices/PAX/xattrs or extra data, and fixed compressed, expanded, document, and binary quotas. Binary build information must match Go 1.26.5, the canonical MARS module/command, requested platform and architecture level, `CGO_ENABLED=0`, trimpath, exact clean Git revision, and valid VCS time.
- Focused normal/race tests, vet, DocSync, four CGO-disabled cross-compiles, and vulnerability scanning passed with zero called vulnerabilities. QA and Security accepted the diff after the authenticated checksum result was bound to its exact tag and commit.
- A clean GoReleaser snapshot `0.69.0-dev.b824b91` passed both the producer verifier and the A2 Darwin/arm64 environment verifier without execution, installation, or destination mutation.
- Checkpoints B through E remain incomplete. The repository remains private at `VERSION=0.68.49`; no tag, Release, MARS signature, upload, visibility, or supported-release claim changed.

## Checkpoint B1 completion — 2026-07-22

- Pushed commit `f3ed495` performs anonymous-first, optional credential-safe acquisition of one exact immutable ten-asset Release; resolves and boundedly peels its exact tag; enforces latest/downgrade/replay policy; and runs A1 then A2 before returning a private cloned candidate without filesystem mutation.
- All eight signed archive/SBOM digests bind to the remote inventory. Selected bytes bind to declared size and digest, and final exact Release ID, latest selector when used, inventory, and ref chain must equal the opening snapshot. Quotas, timeouts, redirects, credential handling, and failures are bounded and redacted.
- `go test ./internal/selfupdate`, focused B1 race, `go vet ./internal/selfupdate`, docsconsistency and DocSync, four Go 1.26.5 CGO-disabled compile-only tests, and `govulncheck ./internal/selfupdate` passed with zero called vulnerabilities. Engineer, QA, Security, and Orchestrator are GO. B2 through E remain incomplete; no candidate was written, executed, or installed and no release or publication authority changed.

## Checkpoint B2 completion — 2026-07-22

- Pushed commit `92d7ddd` adds an unwired, network-free replacement primitive for one A1/A2/B1-verified candidate. It admits only the fixed `mars` leaf beneath a canonical owner-controlled directory, binds operations to no-follow descriptors, serializes with a persistent 0600 nonblocking lock, and uses a same-filesystem 0700 transaction directory with 0600 candidate and backup files.
- Existing and absent destinations use atomic rename or no-replace link semantics. Success requires file and directory durability plus exact post-replacement inode, mode, link-count, bytes, platform, toolchain, and commit verification. Pre-commit cancellation preserves the prior state; post-commit failures either prove compensation or retain explicit recovery-required state and a verified backup without overwriting an unknown replacement.
- Focused normal and race tests, full `internal/selfupdate` tests, vet, docsconsistency and DocSync, Darwin/Linux AMD64/ARM64 Go 1.26.5 CGO-disabled compile-only tests, and `govulncheck ./internal/selfupdate` passed with zero called vulnerabilities. Engineer, QA, Security, and Orchestrator are GO.
- B2 is not wired into `Run`; the legacy raw/checksum-only release updater remains reachable until B3. B3 through E remain incomplete. No candidate was executed or installed through a live command, and no version, tag, Release, signature, upload, visibility, or publication authority changed.

## Checkpoint B3 completion — 2026-07-22

- Pushed commit `683daf8` wires release-mode `Run` through B1 acquisition, B2 durable replacement, then shell-PATH repair. Source/main and dry-run paths retain their prior behavior, and dry-run invokes no identity, network, credential, verifier, replacement, or filesystem authority.
- Implicit latest admits only the exact canonical running `mars` executable with clean full-commit build identity, hashes its descriptor-read bytes before acquisition, and requires B2 to recheck that digest under the destination lock before creating transaction state. Prior drift leaves the changed destination byte-identical and creates no transaction. Explicit-version rollback requests remain governed by their authenticated release identity.
- The returned plan is populated only from authenticated B1 evidence and exposes no raw download/checksum URLs. A post-commit replacement-identity mismatch returns that plan with recovery-required truth and does not invoke PATH repair.
- Full `internal/selfupdate` and isolated-home `cmd/mars` tests, focused race tests, vet, docsconsistency, DocSync, Darwin/Linux AMD64/ARM64 Go 1.26.5 CGO-disabled compile-only tests, `make vuln` with zero called vulnerabilities, fuzz smoke, and the lint fallback to whole-source vet passed. Engineer, QA, Security, and Orchestrator are GO.
- Legacy raw/checksum helpers remain dead and unreachable from `Run` for checkpoint D removal. Checkpoints C through E remain incomplete. No live update was run, and no version, tag, Release, signature, upload, visibility, or publication authority changed.

## Checkpoint C completion — 2026-07-22

- Pushed commit `f45d5d2` retires the legacy shell binary bootstrap. A shell cannot authenticate the first downloaded MARS verifier without circular trust, so the executable script now uses Bash builtins only, exits nonzero, and points to an independently reviewed exact source checkout with Go 1.25.12 or newer and `make install`.
- Latest and explicit-version hostile-environment tests prove the script invokes no network, credential resolver, external command, temporary file, privilege escalation, PATH mutation, or destination mutation. An existing `mars` canary remains byte-identical and the only destination entry; caller-controlled version, path, and token canaries remain absent from output.
- Focused and race installer/source-update tests, full `internal/selfupdate` tests, vet, Bash syntax, docsconsistency, DocSync, and diff checks passed. Engineer, QA, Security, and Orchestrator are GO.
- Fresh binary bootstrap remains unavailable and blocks public cutover until a non-circular trusted bootstrap is designed and validated. Checkpoints D and E remain incomplete. No live install was run, and no version, tag, Release, signature, upload, visibility, or publication authority changed.

## Interfaces and blast radius

internal/selfupdate, internal/release retained consumer verification/audit, scripts/install.sh, update/release CLI output, MarsCLI/tool guidance, directly affected live docs and generated target doctrine, plus focused tests. Production verification must be shared rather than duplicated in shell. Network reads are HTTPS-only, size/time/redirect bounded, and credentials may be sent only to the exact approved GitHub API origin and must be stripped on redirects.

## Acceptance criteria

All authenticity, checksum, canonical-name, tag, commit, platform, toolchain, metadata, and archive checks occur before replacement. Negative lanes cover tampered/missing/extra/duplicate records or assets, unsigned/invalid/wrong-identity bundles, wrong tag/commit/platform, hostile archives, oversized/truncated responses, partial/cancelled downloads, redirect credential stripping, implicit/latest downgrade or tag/commit replay, concurrent updates, unsafe destination types, and injected failures before and after rename. An explicit older-version request may perform a rollback only when its signature, tag, commit, platform, and archive all verify. Every failure leaves or restores the prior binary byte-identical and reports an actionable recovery command. Verification errors never echo hostile URLs, member names, certificate fields, response bodies, tokens, or candidate bytes.

## Non-goals and authority boundary

Do not modify the GoReleaser producer, create a production signing workflow, fetch credentials/OIDC in tests, change VERSION/CHANGELOG, create or move a tag, create/upload/sign/publish a GitHub Release, enable Pages, change repository visibility, announce support, or claim a real release passed. Keep VERSION 0.68.49, source fallback 0.69.0-dev, repository private, and Primary Status primary_blocked. Two-build private rehearsal and clean macOS/Linux synthetic-fixture lifecycle remain T-068 work. Anonymous release access, actual keyless signing/publication, public canary, and every visibility action remain solely the separately approved F-018-S004/F-017 cutover.

## Planning tool note

The first `ticket_create` attempt under the CTO-weekly role reached its stricter first-proof heuristic because that heuristic recognizes build-and-smoke tokens only in ticket frontmatter rather than T-065's full body and plan evidence. The Orchestrator recreated this same bounded ticket through unchanged `ticket_create` as `foundation-maintainer`, retaining the normal feature-contract and earliest-uncovered-scenario gates. No ticket-policy source change is part of T-066; revisit the heuristic only if the limitation recurs.
