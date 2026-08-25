# F-018: Supported Release Distribution

- Feature ID: F-018
- Goals: G-OSS-001, G-004
- Status: active
- Owner: Release Manager with foundation-maintainer

## Business Logic

MARS uses a repository-owned, source-only producer for supported binary
releases. Generated target harnesses do not inherit this Go-specific
implementation. A supported release exists only after deterministic output,
checksums, SBOMs, GitHub/Sigstore keyless provenance, a fresh-download
comparison, immutable GitHub publication, and consumer verification all pass.

Private snapshot evidence proves build behavior but is never a
supported-release claim. T-065 through T-068 retain their historical
GoReleaser evidence. AD-315 supersedes GoReleaser/Cosign as the launch
implementation: the supported Go toolchain builds the four targets, ordinary
deterministic archive/checksum commands package them, upstream Syft creates the
SBOMs, and GitHub `actions/attest` creates keyless provenance after visibility
changes under separate approval. The launch contract reserves attested
`v0.69.0` as a rollback bridge and attested `v0.69.1` as the supported latest
release.

## Step-By-Step Behavior

1. T-064 removes the unsupported private v0.93 lineage and restores the retained v0.68.49 release floor without publishing a replacement.
2. T-065 builds pinned GoReleaser and Syft modules with the pinned Go toolchain, defines the four archive targets, and removes the bespoke publication path only after the replacement producer passes.
3. T-066 migrates installation and updates to fail-closed archive, checksum, signature, identity, metadata, and extraction verification.
4. T-067 raised the MARS source floor to Go 1.25.12 under the owner's 2026-07-22 approval; historical private release evidence used Go 1.26.5, while AD-315 selects exact Go 1.27.0 for launch production. Packaged MARS operation does not require an externally installed Go toolchain, and generated targets choose their own toolchain.
5. T-068 runs reproducible private snapshots and clean installation fixtures without creating a tag, Release, signature, or supported-release claim.
6. T-071 restored the green application vulnerability baseline and T-077
   completed anonymous bootstrap. T-078's first GoReleaser/Cosign route failed
   producer admission and then expanded into bespoke security infrastructure;
   that work is preserved as a non-authorizing checkpoint.
7. T-078/T-079 replace the stale producer with the conventional AD-315 dormant
   workflow, compatible consumer, no-publish rehearsal, hosted sanitation, and
   private contribution controls.
8. After a separately approved public-visibility change, T-080 publishes and
   verifies attested `v0.69.0` as the rollback bridge and attested `v0.69.1` as
   latest. T-081 owns the 48-hour canary and announcement.

## Scenario Schedule

1. F-018-S001 - Produce deterministic, publication-disabled GoReleaser snapshot archives.
2. F-018-S002 - Install safely and migrate the binary updater to the archive contract.
3. F-018-S003 - Rehearse the complete pipeline privately without publishing.
4. F-018-S004 - Publish and verify an immutable public release after F-017 approval.

## Scenarios

### F-018-S001: Deterministic Release Production

Given an exact clean source commit and explicit non-release snapshot version
When pinned GoReleaser builds MARS twice with the pinned Go toolchain
Then Darwin and Linux AMD64 and ARM64 archives are byte-identical between runs
And each archive contains exactly the MARS binary, license, notice, and third-party notices
And all four binaries' build metadata binds the exact platform, Go toolchain, full commit, clean VCS state, and commit-derived build time
And the host-native binary reports the linked snapshot version, full commit, and commit-derived build date
And SPDX-JSON SBOMs and an exact SHA-256 checksum set covering all four archives and four SBOMs are produced
And the four archives and their checksum records are reproducible while each run verifies its own raw SBOM bytes and the two runs have equivalent normalized SPDX semantics
And the default producer contains no signing or publication authority and creates no public keyless signature
And raw binaries, legacy `mars-harness-*` aliases, stale files, and extra assets are absent.

### F-018-S002: Safe Archive Installation And Binary Updater

Given a supported release archive, checksum file, and signature bundle
When the binary updater `mars update tool` selects the current platform
Then the expected workflow identity, signature, checksum, tag, commit, and platform are verified before replacement
And extraction rejects absolute or traversal paths, links, devices, duplicates, missing entries, unexpected entries, and quota violations
And the installed binary is replaced atomically only after every verification passes
And a pre-commit failure preserves the previous binary, while an unprovable
post-commit compensation fails recovery-required with transaction-preservation
and trusted-source repair guidance

Given a fresh installation has stable Go 1.25.13 or newer, an existing
owner-controlled final install directory, and an independently reviewed exact
release-tag checkout
When the shell installer is executed directly and receives one exact stable
semantic tag
Then it uses owner-only temporary staging and builds only the canonical MARS
command and module at that exact tag through the public Go proxy and SumDB,
with direct, private/no-sum, workspace, replacement, and floating-version paths
unavailable, without a network-fetched script-to-shell route
And its privileged Bash shebang suppresses inherited functions and `BASH_ENV`,
an explicit shell-interpreter invocation fails closed, and a clean environment
preserves only `PATH`, `HOME`, and `TMPDIR`; optional GitHub tokens cross over
dedicated descriptors, remain absent from Go, and are supplied only to the
staged signed updater
And the exact absolute Go executable runs with Go auth and CGO disabled,
compiler/tool controls neutralized and `-modcacherw` enabled, resolved
temporary-root ancestry restricted to private current-user or safe root-owned
directories, and private staging `TMPDIR`/`GOTMPDIR`
And the staged command's running `runtime/debug.BuildInfo` must confirm the
canonical command/module, exact module version, canonical SHA-256 `h1` sum, and
absence of replacements before bootstrap admission
And the staged command delegates the same exact tag and final directory to the
existing signed updater, which remains the sole archive/signature verifier and
durable replacement authority, without ordinary shell PATH mutation
And pre-commit rejection leaves an existing final binary unchanged; a
recovery-required result instead preserves transaction evidence and requires
trusted-source repair before retry
And successful script exit requires verified private-staging removal; an
ordinary failure preserves its original error plus a fixed path-free warning
when cleanup is incomplete, while post-install cleanup failure reports that the
binary was installed but staging cleanup remains incomplete.

### F-018-S003: Private Rehearsal Without Publication

Given the repository remains private and the public cutover is not approved
When CI and Dogfood exercise the release pipeline
Then fork-safe CI runs `goreleaser check` and a clean snapshot without secrets or write authority
And two-build reproducibility and hostile artifact tests pass
And clean macOS/Linux source setup plus direct native execution of contract-verified unsigned snapshots pass as private fixture evidence
And offline synthetic/preverified consumer update and rollback pass without treating those snapshots as supported packaged installs
And the exact-version Go/SumDB bootstrap is exercised only through bounded
private fixtures until a real official tag and signed Release exist
And offline synthetic fixtures or pinned upstream public test vectors may exercise consumer verification without representing a MARS release
And the read-only snapshot workflow succeeds with publication, signing, attestation, upload, draft, tag, and Release authority absent
And no supported tag, GitHub Release, public signature, announcement, or visibility change occurs.

### F-018-S004: Immutable Public Publication

Given F-018-S001 through F-018-S003 and every private F-017 prerequisite pass
And the owner separately approves public visibility
When the exact `v0.69.0` and `v0.69.1` release workflows run through T-080
Then the unprivileged Go build/archive/Syft producer job has no OIDC or write authority
And only the protected attestation job receives `id-token: write` and `attestations: write`
And only the subsequent publisher job receives `contents: write`
And each same-tag, same-commit Release converges on exactly ten uploaded Release assets: four archives, four SPDX SBOMs, `checksums.txt`, and `checksums.txt.sigstore.json`, excluding provider-generated source archives and attestations from that uploaded-asset count
And fresh downloads match the verified local artifacts before success
And the GitHub/Sigstore bundle verifies the exact checksum subject set, pinned workflow identity, tag, and commit
And `v0.69.0` remains only as the rollback bridge while `v0.69.1` is marked latest
And logged-out archive download, install, update, rollback, attestation, and per-asset verification pass.

## Evidence

- **F-018-S001:** Passed 2026-07-21 under T-065. Producer/config checkpoint `dc5685b`, checker checkpoint `6a68ecc`, evidence checkpoint `3b4f7c8`, and bespoke-producer retirement checkpoint `bb1b79b` are pushed. Two clean clones at `6a68eccf30036ab2fa84474afb85f7ee113c6ed9` passed the exact nine-file publishable-set, eight-checksum, byte-identical archive, four-platform build-metadata, one-native-runtime, archive/SBOM binding, and normalized-SPDX comparison contract. A separate clean final-commit snapshot `0.69.0-dev.bb1b79b` passed the same committed environment verifier after producer retirement; full source, cross-build, installed-binary, clean-target, DocSync, QA, Security, Dogfood, Release Manager, and Orchestrator gates passed. No tag, Release, signature, upload, visibility change, or supported-release claim occurred. Private snapshot notices remain provisional until the complete Go dependency notice review passes before cutover. Exact GoReleaser `v2.17.0` binary findings GO-2026-5970 and GO-2026-5932 are accepted only for credential-free, publication-disabled private snapshot evidence with `ko`, signing, announcement, and publication explicitly skipped; they remain a public-cutover no-go pending an acceptable upstream release/removal.
- **F-018-S002:** Passed 2026-07-22 under T-066. A1 `fcf7397` verifies the exact offline Sigstore/checksum trust contract; A2 `b824b91` verifies canonical bounded archives and binary build metadata; B1 `f3ed495` acquires one immutable replay/drift-checked candidate without mutation; B2 `92d7ddd` performs descriptor-bound durable replacement or provable compensation; B3 `683daf8` wires release-mode updates through those stages before PATH repair; C `f45d5d2` retires circular checksum-only shell bootstrap to a fail-closed reviewed-source route; and D through `64c7d3b` retires weaker verifier/audit consumers and synchronizes source/target doctrine. E at `7fe152c` combines the immutable upstream offline Sigstore vector and explicit authenticated older-version acquisition policy with a production-B2 preverified-candidate update/rollback lifecycle, including pre-mutation identity mismatch rejection, exact digest transitions, final mode/link/lock checks, and no transaction residue. The exact Go 1.26.5 focused normal/race set, preserved full-source/race/coverage evidence, exact `govulncheck v1.6.0`, four ten-second fuzz targets, whole-source vet lint fallback, docsconsistency, DocSync, and four-platform CGO-disabled metadata-inspected builds passed. An isolated-prefix native source candidate then passed source DocSync, retired-command and authority-free dry-run checks, fresh-target initialization/commit/DocSync, producer-neutral generated guidance, and Engineer prompt assembly with Go absent from runtime PATH. Engineer, QA, Security, Dogfood, Release Manager, and Orchestrator are GO. This is not a packaged install, real MARS-signed update, model-backed lifecycle, or executed Linux rehearsal; fresh packaged bootstrap and T-068 remain blocked before cutover. The repository remains private and `primary_blocked`, with no version, tag, Release, signature, upload, visibility, or supported-release claim.
- **F-018-S003:** Passed 2026-07-22 under T-068. Checkpoint A passed exact private GitHub Actions run `29894376197` at `aa4a16b` with the read-only Ubuntu two-root producer contract, supported source and unsigned-native fixtures, offline consumer/update/rollback tests, cleanup, and zero uploaded artifacts. Checkpoint B passed the corresponding macOS `26.3.1`/arm64 source and unsigned-snapshot split with exact artifact/target hashes; its functional marker passed before a truthfully recorded best-effort cleanup-status false negative, and independent verification proved no residue. Checkpoint C selected exactly `0.69.0` without writes, removed the original 22 exact roots, and failed closed on the pre-existing Linux source gate. T-069 corrected that bounded portability issue; runs `29898672813` at `03008f7` and `29899168382` at final evidence head `2ef9d27` passed both supported Go lanes plus the expected below-minimum rejection, with zero artifacts. Final private-state, no-`v0.69.0`, Pages-disabled, stash/worktree, cleanup, QA, Security, Dogfood, Release Manager, and Orchestrator gates passed. This remains private source, unsigned-snapshot, and offline-preverified fixture evidence—not fresh packaged bootstrap or a supported release. The repository remains private and `primary_blocked`; F-018-S004 and all public-cutover prerequisites remain pending.
- **F-018-S004:** T-078's rejected GoReleaser/Cosign and bespoke-containment
  attempts remain preserved in
  `docs/validation/reports/2026-08-24-t078-release-production-admission-blocked.md`
  and the local checkpoint named by the owner disposition. AD-315 is now the
  only launch implementation: conventional Go production, upstream Syft, and
  GitHub `actions/attest`. The owner has funded the account, accepted the
  recorded name risk, attested publication authority, and removed account-wide
  App scope from the launch. The conventional workflow, compatible consumer,
  no-publish rehearsal and contribution controls pass. The owner approved the
  exact T-080 transaction, so the workflow is now active only for guarded
  repository tag pushes; no launch tag exists yet. The separately
  approved hosted transaction deleted the exact 500 legacy assets, 77
  deployments, and 474 sealed runs while preserving 56 Releases, 301 tags,
  and 33 newer runs; future-only immutable Releases are enabled. Separately
  approved visibility change, public-only controls, real
  `v0.69.0`/`v0.69.1` publication, and anonymous lifecycle remain pending.

## Out of Scope

- Publishing raw binaries or `mars-harness-*` compatibility aliases.
- Imposing the MARS source producer on generated target repositories.
- Homebrew, container images, Windows packages, notarization, or public announcement in the first release.
- Reusing `v0.93.0` or silently selecting replacements if GitHub rejects either owner-selected launch version.

## Descoped Scenarios

- Legacy v0.93 raw-asset compatibility is deliberately not retained; affected private installations require one manual reinstall.
- Homebrew, container, Windows, and notarized macOS distribution may be planned only after the initial supported archive lifecycle passes.
