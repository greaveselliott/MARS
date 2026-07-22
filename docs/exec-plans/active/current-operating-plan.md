# Active P0 Exec Plan: Retire v0.93.0 And Adopt GoReleaser

**Status:** Active
**Priority:** P0
**Depends On:** T-064 through T-067 complete; T-066/F-018-S002 entry gate passed
**Blocks:** F-017-S003 public release readiness and every later cutover claim
**Related Tickets:** T-063 through T-069
**Current Ticket:** None; T-068/F-018-S003 passed and F-018-S004 remains gated by F-017
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-018-goreleaser-distribution.md
**Related Feature Contracts:** F-001, F-009, F-017, F-018
**Hypothesis:** Retiring the private `v0.93.0` experiment and using a pinned GoReleaser archive pipeline will reduce bespoke release code while retaining fail-closed consumer verification.
**Success Evidence:** `v0.93.0` is absent; protected work survives; GoReleaser archives are reproducible; exact archive/SBOM checksums pass; signed consumers reject hostile input before mutation and preserve the installed binary on failure; and no unsupported release is advertised.
**Falsification Evidence:** A lease drifts, protected work changes, v0.93 remains reachable through live refs/releases, legacy publisher behavior remains required, or a partial/tampered release can be accepted.
**Scenario Schedule:** T-064 retirement passed; T-065 GoReleaser producer passed; T-067 minimum-Go compatibility passed; T-066 installer/updater migration passed; T-069 source-compatibility portability passed; T-068 private rehearsal passed; F-018-S004 waits for the owner-approved F-017 cutover
**Current Failing Scenario:** F-018-S001 through F-018-S003 passed privately. F-018-S004 is deliberately not entered: fresh packaged bootstrap, notices, producer findings, and the independent F-017 audit, runtime, contribution, cutover, and canary gates remain incomplete.
**Walking Skeleton Slice:** As of 2026-07-22, F-018-S001 through F-018-S003 are complete: the pinned producer, fail-closed signed consumer, source/install transition, and private cross-platform rehearsal passed without publication authority. F-018-S004 remains deferred to F-017.
**Learning Or MVP Outcome:** Publication-disabled archive production plus fail-closed signed archive consumption, recovery, and private rehearsal are proven before any real MARS signing, publication, or visibility change.
**Created:** 2026-07-21
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager packets

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including anonymous signed install/update, fork-safe contribution, logged-out cutover smoke, and a clean 48-hour canary.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** Fresh packaged bootstrap, complete notices, recorded producer findings, and the independent F-017 audit, runtime, contribution, cutover, and canary gates remain incomplete.
- **Next Primary Action:** Return to F-017 through a bounded `ticket_create` handoff for the owner-operated standard-tool audit and disposition; do not enter F-018-S004 or publish.

## Locked Decisions

1. Remove `v0.93.0` Release, tag, release-note commit, and live evidence claims completely; never reuse `v0.93.0`.
2. Reset the retained release floor to `v0.68.49` and prepare `v0.69.0` as the first GoReleaser release, accepting that the version name was previously used privately and may survive in old caches.
3. Use a clean archive migration. Existing v0.93 clients receive no raw-asset compatibility bridge and require one manual reinstall.
4. GoReleaser is source-only. Generated target harnesses use project-owned release tooling rather than inheriting a Go-specific publisher.
5. Private validation uses snapshots only. The `v0.69.0` tag and keyless public signature wait for the separately approved cutover.
6. Keep the repository private and `primary_blocked` throughout this plan.

## Operator-Authorized Release Transition Exception

As of 2026-07-21, T-065 has retired the bespoke publisher and AGENTS rule 14 now points to the repository-owned producer. The operator-approved transition still defers the next source version, tag, signature, upload, and supported-release claim until the public cutover. Therefore T-064 through T-068 use this bounded source-only exception:

- commit and push each validated ticket normally, but do not create a private support release or legacy assets;
- keep `VERSION` and the changelog at `0.68.49` through T-068 while untagged source builds report `0.69.0-dev`;
- require T-068 release-note dry-run evidence to select exactly `0.69.0`, but leave the actual release-note commit, tag, and assets to the later cutover ticket;
- restore rules 12 and 14 to the GoReleaser steady-state workflow in the same migration before the exception expires at cutover.

This exception cannot authorize a supported-release claim, visibility change, or unrelated unversioned source work.

## Ticket Sequence

| Ticket | Outcome | Entry gate | Exit gate |
| --- | --- | --- | --- |
| T-064 | Retire v0.93 and reset the release floor. | Live leases match the captured manifest. | Passed 2026-07-21: Release/tag are absent, Pages is disabled, protected work is re-anchored, `VERSION` is `0.68.49`, and the source fallback is `0.69.0-dev`. |
| T-065 / F-018-S001 | Passed 2026-07-21: replaced the bespoke producer with pinned GoReleaser. | T-064 complete. | Two-clean-root reproducibility plus a final `bb1b79b` snapshot, exact publishable-set contract, full source/cross-build gates, installed-binary and fresh-target checks, and persona reviews passed without publication authority. |
| T-066 / F-018-S002 | Passed 2026-07-22 through `7fe152c`: fail-closed signed archive consumers, retired legacy paths, isolated-prefix source, fresh-target, and preverified update/rollback evidence passed. | T-065 and T-067 complete; T-066 created through `ticket_create` with a frozen offline Sigstore trust contract. | Consumers reject tampering and unsafe archives; legacy checksum-only verification and raw aliases are absent; exact validation and persona gates passed without a real MARS signature or release. |
| T-067 | Passed 2026-07-22: raised the MARS source minimum to exact Go 1.25.12 without imposing an external Go requirement on packaged operation or generated targets. | Owner approved 2026-07-22. | Exact Go 1.25.12 and release Go 1.26.5 gates, real Go 1.25.11 rejection, patch-aware source doctor tests, four cross-builds, installed/fresh-target smoke, and QA/Security review passed without Sigstore or publication changes. |
| T-068 / F-018-S003 | Passed 2026-07-22: checkpoints A/B proved the private cross-platform pipeline and checkpoint C passed no-write release selection, exact cleanup, immutable-state reconciliation, and all persona sign-offs. | T-066 complete after T-067; T-069 passed. | Two-root producer proof, local native macOS unsigned-snapshot/source fixture, exact-SHA read-only Ubuntu native fixture, offline preverified consumer rollback, publication-authority absence, release-note dry-run, state invariants, and cleanup pass. |
| T-069 | Passed 2026-07-22 at `03008f7`: restored deterministic supported-source Linux CI without weakening runtime or filesystem policy. | Exact-head run `29896978096` failed identically in Go 1.25.12 and 1.26.5. | Exact-head run `29898672813` passed Go 1.25.12, Go 1.26.5, and Go 1.25.11 rejection. |
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
- Release ID `354800199` had exactly nine uploaded assets before deletion, returned HTTP 204 on deletion, and both numeric-ID and tag lookups returned 404 afterward. The exact `v0.93.0` tag lease at `bf3034863691443962ec251e62c8dec1ee5138fb` was then deleted. Final inventory is 301 tags; the REST collection enumerates 56 Releases while direct/GraphQL lookup additionally exposes intentionally retained zero-asset `v0.65.7` Release ID `344415010`, yielding 57 addressable retained Release objects.
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

## T-065 Checkpoint Evidence — 2026-07-21

- Checkpoint A is pushed at `dc5685be81e49a0506979ac610ef59126e70c1a6`: exact GoReleaser `v2.17.0` and Syft `v1.49.0` source-built with Go `1.26.5`, publication-disabled configuration, provisional notices, and the `0.69.0-dev` release-note compatibility fix.
- Checkpoint B1 is pushed at `6a68eccf30036ab2fa84474afb85f7ee113c6ed9`: the test/CI-only contract checker requires exactly four archives, four SBOMs, and one checksum file; verifies the exact eight checksum records, bounded archive structure, four-platform build metadata, archive/SBOM binding, and one native runtime identity; and rejects same-root comparison evidence.
- Two clean clones at `6a68eccf30036ab2fa84474afb85f7ee113c6ed9` independently produced `0.69.0-dev.6a68ecc`. The committed cross-root verifier passed. All four archive hashes and archive checksum records are identical; each run's eight records verify its exact raw artifacts; all four raw SBOMs differ as expected from Syft's timestamp and namespace; and normalized SPDX semantics match after excluding only those fields. Both clones remained clean.
- QA, Security, Release Manager, and Orchestrator accept this as private producer evidence only. GoReleaser's recorded binary findings, provisional dependency notices, signatures, consumers, clean install/update, and public cutover remain unresolved. No tag, Release, upload, signature, or visibility change occurred.
- Checkpoint C is pushed at `bb1b79b7aa5787cea3355a2e592d9bfe4a0d2849`: the bespoke publisher, GitHub creation/upload/reconciliation path, and source command are unreachable; retained verifier/audit/updater consumers are explicitly handed to T-066; source and generated-target authority is synchronized without erasing historical AD-059, AD-140, or AD-312 evidence.
- The final pushed commit passed `make check` (CGO-disabled build, uncached race/coverage at 73.9% with all ratchets met, zero called source vulnerabilities, fuzz smoke, and vet), DocSync `345/0`, and explicit four-platform CGO-disabled builds. A distinct clean clone produced `0.69.0-dev.bb1b79b` with the exact pinned tools and passed the committed environment verifier.
- The installed commit-bound binary has SHA-256 `fd8367e06e84203a7d971b00ef7b6b97fbb486756f85eb9d7d33909a20173fe2`, identifies full clean commit `bb1b79b7aa5787cea3355a2e592d9bfe4a0d2849`, omits the retired command from release help, and rejects its direct invocation. A fresh initialized target committed a clean producer-neutral scaffold with no MARS GoReleaser or retired-command injection. Release-note dry-run selects exactly `0.69.0` without writing files.
- QA, Security, Dogfood, Release Manager, and Orchestrator accept F-018-S001 as passed private producer evidence only. The repository remains private at `VERSION=0.68.49`; `v0.69.0` has no live tag or Release; no signature, upload, announcement, visibility change, or supported-release claim occurred. Provisional notices and the two recorded GoReleaser binary findings remain public-cutover blockers.

## T-066 Planning Handoff — 2026-07-22

- `ticket_create` materialized T-066 for F-018-S002 through the primary `foundation-maintainer`/Orchestrator with T-065 as its dependency. COO, CTO-weekly, and Security packets froze six independently bounded planned checkpoints: signed-checksum verifier, archive extractor, updater transaction, installer boundary, consumer/DocSync retirement, and lifecycle evidence.
- The consumer contract adds `checksums.txt.sigstore.json`, an offline Sigstore bundle v0.3 over the exact canonical checksum bytes. Trust is compile-time pinned to the Sigstore root hash, GitHub Actions issuer, exact MARS release workflow/tag identity, and the Fulcio GitHub workflow SHA extension equal to the expected full commit; failure to verify any claim blocks rather than falling back or parsing certificates ad hoc.
- The first CTO-labelled call exposed a planning-policy limitation: completed-validation detection looks only for literal build-and-smoke tokens in ticket frontmatter. The Orchestrator reran unchanged `ticket_create` under its owning role, preserving feature-contract and scenario-order enforcement. No policy implementation is added to T-066.
- Repository state remains private and `primary_blocked`; `VERSION` is `0.68.49`; source fallback is `0.69.0-dev`; no tag, Release, MARS signature, upload, announcement, or visibility change is authorized.

## T-066 A1 Dependency Admission — 2026-07-22

- The newest admissible-floor candidate, official `sigstore-go v0.7.0` at `9c466a8b8df6a1886292e0a82023bc217968da9e`, retained the main module's Go 1.22.4 directive after exact-toolchain tidy and compiled every required offline bundle, identity, commit-extension, log, timestamp, SCT, and artifact-byte API.
- Independent Engineer and Security probes both rejected it: exact called-path `govulncheck v1.3.0` found 14 called findings, decisively including the `GO-2026-5952` multi-log verification-threshold bypass in the verification path. Its upstream fix is `sigstore-go v1.2.0`, which requires Go 1.25.0.
- At that dependency-admission checkpoint, T-066 could not authorize a minimum-Go compatibility migration or custom cryptography. No dependency or source change had landed, the repository remained private at `VERSION=0.68.49`, and checkpoints A2 through E were unstarted pending a separately approved ticket or a secure Go-1.22-compatible upstream release.
- A bounded follow-up established an exact candidate route without changing MARS: newest official `sigstore-go v1.2.2` at `55aa6240784677449a564e66a0fca7a6a3605ecd` compiles the full offline wrapper. Its declared Go 1.25.8 floor still produced eight called standard-library findings, while the identical path on Go 1.25.12 produced zero called findings.
- The owner approved that bounded source-only compatibility route on 2026-07-22. T-067 now owns `go 1.25.12`, retained `toolchain go1.26.5`, exact minimum/release lanes, Go 1.25.11 rejection, patch-aware canonical-source checking, and no external Go requirement for packaged or generated-target operation; no Sigstore dependency or verifier work belongs in T-067.

## T-066 Checkpoint A1 Evidence — 2026-07-22

- Commit `fcf7397` is pushed. It admits exact `sigstore-go v1.2.2` from upstream commit `55aa6240784677449a564e66a0fca7a6a3605ecd`, retains `go 1.25.12` and `toolchain go1.26.5`, and verifies the recorded SumDB module and go.mod hashes with no replacement.
- The production primitive is byte-only and offline: it hash-pins the 6,787-byte public-good root at `6494e21e…0b66`, requires bundle v0.3, one inclusion proof, observer time, SCT, exact artifact bytes, and one exact GitHub Actions workflow/repository/tag/full-commit identity before accepting exactly eight sorted MARS archive/SBOM checksums. It exposes no path, network, root override, updater, installer, signing, or publication authority.
- The positive conformance vector is the public GoReleaser `v2.17.0` `checksums.txt` and signature bundle, pinned to source commit `770a4fc7a8fb2dca874b6c98cb739dd64fc931c0`; it is upstream release evidence used offline, not a synthetic or MARS signature. Tampered bytes, wrong commit/root, missing proof, unsupported bundle version, noncanonical checksum grammar, and hostile error content are rejected.
- Exact Go 1.25.12 full-source testing passed after the sole DocSync metadata finding was corrected and its package rerun passed. Vet, focused race, four Go 1.26.5 CGO-disabled cross-builds, module verification, exact-Go whole-source `govulncheck`, and the corrected Go 1.26.5 `make vuln` lane passed with zero called vulnerabilities. Engineer, QA, Security, and Orchestrator accepted A1.
- At the A1 checkpoint, A2 through E were unstarted. Dependency notices remained provisional and public cutover remained blocked; no MARS tag, Release, signature, upload, version, visibility, or support claim changed.

## T-066 Checkpoint A2 Evidence — 2026-07-22

- Commit `b824b915bb6424636bb9e09fd91c9f9501f259ac` is pushed. The byte-only verifier binds the authenticated checksum result to the exact tag and full commit, selects the canonical platform archive, authenticates its digest before decompression, and returns only a cloned verified binary.
- Inspection permits exactly the ordered `LICENSE`, `NOTICE`, `THIRD_PARTY_NOTICES`, and `mars` USTAR members with fixed ownership and modes; applies compressed, expanded, document, and binary limits; rejects unsafe types, metadata, missing/extra content, multiple streams, and trailing data; and requires exact Go 1.26.5, module, platform, architecture, CGO, trimpath, VCS revision, clean-state, and timestamp build metadata.
- Focused normal and race tests, vet, DocSync, four CGO-disabled cross-compiles, and vulnerability scanning passed with zero called vulnerabilities. QA and Security are GO after the authenticated checksum result was bound to its exact tag and commit.
- One clean GoReleaser snapshot, `0.69.0-dev.b824b91`, passed the producer verifier and the A2 Darwin/arm64 environment verifier without candidate execution or installation. Checkpoints B through E remain incomplete; the repository remains private and no version, tag, Release, MARS signature, upload, visibility, or support claim changed.

- COO, CTO-weekly, QA, and Security split checkpoint B into three independently green commits: B1 performs bounded credential-safe acquisition and A1/A2 verification without filesystem writes; B2 performs the serialized durable local transaction without network access; B3 wires `Run` and the minimum current-version identity while preserving source mode. Installer work remains C and broader CLI/audit/docs retirement remains D.

## T-066 Checkpoint B1 Evidence — 2026-07-22

- Commit `f3ed495` is pushed. It reconciles one exact immutable ten-asset GitHub Release, resolves and boundedly peels its exact tag to a full commit, applies default-selector/downgrade/replay policy, acquires only checksums, signature bundle, and the selected platform archive, then runs A1 and A2 before returning a private cloned candidate.
- Anonymous access is attempted first; optional credentials are resolved only after an exact GitHub API 401/403/404 and are never forwarded across the bounded HTTPS asset redirect. Metadata, evidence, archive, redirect, tag-chain, response-size, and overall acquisition limits fail closed with fixed redacted errors.
- All eight signed archive/SBOM digests must match the immutable Release inventory, selected bytes must match their declared size and digest, and a final exact Release ID, default selector when used, inventory, and ref-chain recheck must match the opening snapshot. The implementation performs no filesystem write, candidate execution, installation, signing, or publication.
- `go test ./internal/selfupdate`, focused B1 race, `go vet ./internal/selfupdate`, docsconsistency and DocSync, four Go 1.26.5 CGO-disabled compile-only tests, and `govulncheck ./internal/selfupdate` passed with zero called vulnerabilities. Engineer, QA, Security, and Orchestrator are GO. Checkpoints B2 through E remain incomplete; the repository remains private and no version, tag, Release, signature, upload, visibility, or support claim changed.

## T-066 Checkpoint B2 Evidence — 2026-07-22

- Commit `92d7ddd` is pushed. The unwired, network-free primitive admits only the fixed `mars` leaf under a canonical owner-controlled install directory, binds every mutation to no-follow descriptors, serializes with a persistent 0600 lock, and uses same-filesystem 0700 state with 0600 candidate and backup files.
- Existing and absent destinations use atomic rename or no-replace link semantics. Pre-commit failures and cancellation preserve the exact prior state; post-commit failures compensate only after revalidating the candidate, restore, and directory bindings, otherwise retaining explicit recovery-required evidence without overwriting an unknown entry. Success requires file and directory durability and exact post-cleanup identity, bytes, mode, link count, platform, toolchain, and commit proof.
- Focused normal/race and hostile-filesystem tests, full `internal/selfupdate` tests, vet, docsconsistency, DocSync, four Go 1.26.5 CGO-disabled compile-only targets, and `govulncheck ./internal/selfupdate` passed with zero called vulnerabilities. Engineer, QA, Security, and Orchestrator are GO.
- B2 remains deliberately unwired. B3 must remove the reachable legacy raw/checksum-only release branch before installer work. B3 through E remain incomplete; the repository remains private and `primary_blocked`, and no live installation, version, tag, Release, signature, upload, visibility, or support claim changed.

## T-066 Checkpoint B3 Evidence — 2026-07-22

- On 2026-07-22, commit `683daf8` was pushed. Release-mode `Run` performs B1 acquisition, B2 durable replacement, then shell-PATH repair; source/main and authority-free dry-run behavior remain unchanged.
- On 2026-07-22, implicit-latest updates admit only the exact canonical running `mars` executable with clean full-commit build identity, capture its digest before acquisition, and require B2 to recheck that digest under the destination lock before creating transaction state. Drift leaves the changed binary untouched and creates no transaction. Exact-version rollback requests remain bound to their authenticated requested release rather than the running-binary expectation.
- Plans and CLI output expose only the authenticated tag, commit, and archive identity. Raw download/checksum URLs are no longer produced by the live path. A post-commit identity mismatch returns the populated authenticated plan with recovery-required truth and skips PATH repair.
- Full `internal/selfupdate` and `cmd/mars` tests, focused race tests, vet, docsconsistency, DocSync, four Go 1.26.5 CGO-disabled compile-only targets, `make vuln` with zero called vulnerabilities, fuzz smoke, and the lint fallback to whole-source vet passed. Engineer, QA, Security, and Orchestrator are GO.
- Legacy raw/checksum helper code is unreachable from `Run` and remains only for checkpoint D removal. Checkpoints C through E remain incomplete; the repository remains private and `primary_blocked`, and no live update, version, tag, Release, signature, upload, visibility, or support claim changed.

## T-066 Checkpoint C Evidence — 2026-07-22

- On 2026-07-22, commit `f45d5d2` was pushed. Because a shell cannot authenticate its first downloaded MARS verifier without circular trust, `scripts/install.sh` exits nonzero using Bash builtins only and directs operators to an independently reviewed exact source checkout with Go 1.25.12 or newer and `make install`.
- On 2026-07-22, latest and explicit-version hostile-environment tests proved no external command, network, credential, temporary-file, privilege, PATH, or destination authority is exercised; a prior `mars` binary remains byte-identical and alone. The script remains executable and contains no legacy release URL, raw asset, or checksum bootstrap path.
- Focused and race script/source-update tests, full `internal/selfupdate` tests, vet, Bash syntax, docsconsistency, DocSync, and diff checks passed. Engineer, QA, Security, and Orchestrator are GO.
- Fresh binary bootstrap remains unavailable and is a public-cutover blocker, not a completed install claim. Checkpoints D and E remain incomplete; the repository remains private and `primary_blocked`, and no live install, version, tag, Release, signature, upload, visibility, or support claim changed.

## T-066 Checkpoint D Evidence — 2026-07-22

- `9d2d8c9` retires the standalone verifier/audit commands and implementations plus dead legacy raw/checksum-only updater helpers; the retained version-only remote lookup is bounded, credential-scoped, redirect-closed, and redacted. `bb4b620` removes the retired commands from MarsCLI, formal workflows, and generated target doctrine while preserving repository-owned producer/verifier responsibility and exact remote convergence.
- `527646b` synchronizes current architecture, product, BDD, release-versioning, source/target, dogfood, README, and quickstart contracts. `64c7d3b` synchronizes the six live operator guides and adds a guard that rejects either retired command in executable HTML recipes or command-only cells. Original superseded AD bodies, tickets, reports, changelog, and dated review evidence remain unchanged.
- Focused Go 1.26.5 release/selfupdate/CLI, tools/scanner, positive Sigstore/archive/acquisition/replacement, docsconsistency, and DocSync gates passed. CTO-weekly, QA, Security, and Orchestrator are GO. Checkpoint E alone remains; the repository stays private and `primary_blocked`, packaged bootstrap remains unavailable, and no live update, version, tag, Release, signature, upload, visibility, or support claim changed.

## T-066 Checkpoint E Evidence — 2026-07-22

- Pushed `7fe152c` proves a mismatched candidate commit fails B2 admission before lock or transaction creation, then proves a newer preverified candidate and explicit older preverified rollback use the production durable replacement transaction in one `0700` directory with exact digest transitions, final bytes/mode/link count, persistent `0600` lock, and no transaction residue.
- The focused 46-test T-066 normal/race set passed with exact Go 1.26.5 and offline dependency resolution. It composes the B2 lifecycle with the existing explicit authenticated older-version acquisition test and immutable upstream offline Sigstore vector; no real MARS signature, download, or update was claimed.
- Preserving the already-green uncached full-source/race/coverage evidence, the outstanding tail passed: exact SumDB-backed `govulncheck v1.6.0` found zero called vulnerabilities; four default ten-second fuzz targets passed; the documented whole-source vet lint fallback passed; docsconsistency and DocSync passed; and all four Go 1.26.5 CGO-disabled trimpath builds bound the exact clean `7fe152c3f28af252d1eb11436298617e0cfad9de` revision and platform metadata.
- The isolated-prefix native source candidate reported `0.69.0-dev.7fe152c`, exact full commit/date and SHA-256 `9e4e419d4b65bacd907b3841646025c053fd4dd1c5dac4af55495cfa5459e96a`. With Go absent from its scrubbed runtime PATH it passed source DocSync, exposed no retired release commands or release URL in authority-free update dry run, initialized and committed a clean fresh target, passed target DocSync, emitted producer-neutral release doctrine without raw aliases, and assembled the Engineer dry-run prompt without changing either worktree or the binary. Only native Darwin/arm64 was executed; foreign cross builds were metadata-inspected.
- Engineer, QA, Security, Dogfood, Release Manager, and Orchestrator are GO. F-018-S002 and T-066 are complete. T-068, fresh packaged bootstrap, actual MARS signing/publication, notices, and all independent F-017 gates remain blocked; the repository stays private and `primary_blocked` at `VERSION=0.68.49`.

## T-068 Checkpoint A Evidence — 2026-07-22

- Exact-head private GitHub Actions run `29894376197` passed commit `aa4a16bb5d26bcb766851dec375149b906fa6ce8`; every job step, including cleanup, was green.
- The read-only Ubuntu rehearsal passed the exact pinned producer, two-root artifact contract, source and unsigned-native target fixtures, authority-free update dry-run, and focused offline consumer/update/rollback lanes. The run uploaded zero artifacts.
- Repository reconciliation retained private visibility, exact `origin/main`, `VERSION=0.68.49`, source fallback `0.69.0-dev`, and absent `v0.69.0` tag and Release.
- Checkpoint A is accepted at this evidence boundary. Checkpoint B is accepted separately below; checkpoint C, F-018-S003, fresh packaged bootstrap, notices, recorded producer findings, signing, publication, and all independent F-017 gates remain blocked.

## T-068 Checkpoint B Evidence — 2026-07-22

- Immutable commit `aa4a16bb5d26bcb766851dec375149b906fa6ce8` passed the owner-host macOS `26.3.1` (Darwin `25.3.0`)/arm64 rehearsal with exact Go `1.26.5` and snapshot `0.69.0-dev.aa4a16b`: pinned production tooling, committed verification, supported source install, direct unsigned native execution, equivalent clean targets, authority-free update dry-run, and the exact five offline consumer/install/update/rollback tests all passed.
- Accepted identities are archive SHA-256 `b9f73f666d66ed7100a6c126b677541f5fea33ac8e7ffa36edcfe2b42a9ad460`, native binary SHA-256 `8530158a8f9df3799627efee3df51dafb3e71c578737cef13f24f572ada34bf9`, source binary SHA-256 `ba227b4d4eb98d6e190ab43fb57b1a42f3b8380db065d5a8762bc6714ce054e7`, and target tree `e3f3e0ee21b751bfa7a183717be48a76a9134a4b`.
- The functional run emitted `checkpoint_b=pass`; its aggregate shell then exited `1` solely because best-effort cleanup `chmod` warned on read-only module-cache entries. Independent verification proved the exact root absent and host worktree clean. Engineer, QA, and Security accepted the combined functional and postcondition evidence without claiming that the wrapper exited zero or rerunning the producer.
- Checkpoint C, F-018-S003, fresh packaged bootstrap, notices, recorded producer findings, signing, publication, and every independent F-017 gate remain blocked. No version, tag, Release, signature, upload, visibility, or support claim changed.

## T-068 Checkpoint C Evidence — 2026-07-22

- At exact evidence head `2ef9d277f60baca2123ecea7c460ffdb49d018eb`, source-compatibility run `29899168382` passed Go `1.25.12`, Go `1.26.5`, and the expected Go `1.25.11` rejection with zero Actions artifacts. Release-note dry-run selected exactly `0.69.0` without changing HEAD, tree, `VERSION`, `CHANGELOG.md`, or source fallback.
- Final reconciliation retained private visibility, one remote `main`, 301 tags, no `v0.69.0` tag or Release, disabled Pages, both protected stashes, and the unrelated Codex worktree. The REST/GraphQL Release enumeration discrepancy is classified above; the additional addressable object is retained zero-asset `v0.65.7`, not an unexpected release.
- The original 22 manifest-listed rehearsal roots were removed. Final Security discovery then found two additional T-068 reviewer roots—one omitted pre-existing root and one later dry-run cache—plus ten T-069 ticket caches; all were removed by exact path, and final bounded discovery found no rehearsal residue. QA, Security, Dogfood, Release Manager, and Orchestrator are GO for T-068/F-018-S003 only.
- F-018-S003 and T-068 are complete. The repository remains private and `primary_blocked`; fresh packaged bootstrap, notices, producer findings, every independent F-017 gate, F-018-S004, signing, publication, visibility, announcement, and support claims remain blocked.

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
