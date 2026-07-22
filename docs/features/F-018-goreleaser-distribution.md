# F-018: GoReleaser Distribution

- Feature ID: F-018
- Goals: G-OSS-001, G-004
- Status: active
- Owner: Release Manager with foundation-maintainer

## Business Logic

MARS uses pinned GoReleaser as the source-only producer for supported binary releases. Generated target harnesses do not inherit this Go-specific implementation. A supported release exists only after deterministic local output, signed checksums, SBOMs, a fresh-download comparison, immutable GitHub publication, and consumer verification all pass.

Private snapshot evidence proves build behavior but is never a supported-release claim. The first new-format version is reserved as `v0.69.0`; its tag and release remain deferred until the independent F-017 cutover gates and separate owner approval pass.

## Step-By-Step Behavior

1. T-064 removes the unsupported private v0.93 lineage and restores the retained v0.68.49 release floor without publishing a replacement.
2. T-065 builds pinned GoReleaser and Syft modules with the pinned Go toolchain, defines the four archive targets, and removes the bespoke publication path only after the replacement producer passes.
3. T-066 migrates installation and updates to fail-closed archive, checksum, signature, identity, metadata, and extraction verification.
4. T-067 raises the MARS source floor to Go 1.25.12 under the owner's 2026-07-22 approval; the release toolchain remains Go 1.26.5, packaged MARS operation does not require an externally installed Go toolchain, and generated targets choose their own toolchain.
5. T-068 runs reproducible private snapshots and clean installation fixtures without creating a tag, Release, signature, or supported-release claim.
6. A separately approved F-017 cutover publishes the first immutable supported release only after every independent publication gate passes.

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
And any failure preserves the previous binary and provides an actionable recovery command

Given a fresh installation before a non-circular packaged bootstrap exists
When the shell installer runs
Then it fails closed without network or mutation and directs the operator to an
independently reviewed source checkout while source install remains the supported route.

### F-018-S003: Private Rehearsal Without Publication

Given the repository remains private and the public cutover is not approved
When CI and Dogfood exercise the release pipeline
Then fork-safe CI runs `goreleaser check` and a clean snapshot without secrets or write authority
And two-build reproducibility and hostile artifact tests pass
And clean macOS/Linux source setup plus direct native execution of contract-verified unsigned snapshots pass as private fixture evidence
And offline synthetic/preverified consumer update and rollback pass without treating those snapshots as supported packaged installs
And fresh shell/package bootstrap remains fail-closed pending a non-circular trust mechanism
And offline synthetic fixtures or pinned upstream public test vectors may exercise consumer verification without representing a MARS release
And the read-only snapshot workflow succeeds with publication, signing, attestation, upload, draft, tag, and Release authority absent
And no supported tag, GitHub Release, public signature, announcement, or visibility change occurs.

### F-018-S004: Immutable Public Publication

Given F-018-S001 through F-018-S003 and every F-017 prerequisite pass
And the owner separately approves the public cutover
When the exact `v0.69.0` release workflow runs
Then GoReleaser uploads only to an unpublished draft
And fresh downloads match the local artifacts before publication
And GitHub attestations and the keyless checksum signature verify against the pinned workflow identity
And publication makes the release immutable
And logged-out archive download, install, update, rollback, `gh release verify`, and per-asset verification pass.

## Evidence

- **F-018-S001:** Passed 2026-07-21 under T-065. Producer/config checkpoint `dc5685b`, checker checkpoint `6a68ecc`, evidence checkpoint `3b4f7c8`, and bespoke-producer retirement checkpoint `bb1b79b` are pushed. Two clean clones at `6a68eccf30036ab2fa84474afb85f7ee113c6ed9` passed the exact nine-file publishable-set, eight-checksum, byte-identical archive, four-platform build-metadata, one-native-runtime, archive/SBOM binding, and normalized-SPDX comparison contract. A separate clean final-commit snapshot `0.69.0-dev.bb1b79b` passed the same committed environment verifier after producer retirement; full source, cross-build, installed-binary, clean-target, DocSync, QA, Security, Dogfood, Release Manager, and Orchestrator gates passed. No tag, Release, signature, upload, visibility change, or supported-release claim occurred. Private snapshot notices remain provisional until the complete Go dependency notice review passes before cutover. Exact GoReleaser `v2.17.0` binary findings GO-2026-5970 and GO-2026-5932 are accepted only for credential-free, publication-disabled private snapshot evidence with `ko`, signing, announcement, and publication explicitly skipped; they remain a public-cutover no-go pending an acceptable upstream release/removal.
- **F-018-S002:** Passed 2026-07-22 under T-066. A1 `fcf7397` verifies the exact offline Sigstore/checksum trust contract; A2 `b824b91` verifies canonical bounded archives and binary build metadata; B1 `f3ed495` acquires one immutable replay/drift-checked candidate without mutation; B2 `92d7ddd` performs descriptor-bound durable replacement or provable compensation; B3 `683daf8` wires release-mode updates through those stages before PATH repair; C `f45d5d2` retires circular checksum-only shell bootstrap to a fail-closed reviewed-source route; and D through `64c7d3b` retires weaker verifier/audit consumers and synchronizes source/target doctrine. E at `7fe152c` combines the immutable upstream offline Sigstore vector and explicit authenticated older-version acquisition policy with a production-B2 preverified-candidate update/rollback lifecycle, including pre-mutation identity mismatch rejection, exact digest transitions, final mode/link/lock checks, and no transaction residue. The exact Go 1.26.5 focused normal/race set, preserved full-source/race/coverage evidence, exact `govulncheck v1.6.0`, four ten-second fuzz targets, whole-source vet lint fallback, docsconsistency, DocSync, and four-platform CGO-disabled metadata-inspected builds passed. An isolated-prefix native source candidate then passed source DocSync, retired-command and authority-free dry-run checks, fresh-target initialization/commit/DocSync, producer-neutral generated guidance, and Engineer prompt assembly with Go absent from runtime PATH. Engineer, QA, Security, Dogfood, Release Manager, and Orchestrator are GO. This is not a packaged install, real MARS-signed update, model-backed lifecycle, or executed Linux rehearsal; fresh packaged bootstrap and T-068 remain blocked before cutover. The repository remains private and `primary_blocked`, with no version, tag, Release, signature, upload, visibility, or supported-release claim.
- **F-018-S003:** Checkpoint A passed 2026-07-22 at exact commit `aa4a16bb5d26bcb766851dec375149b906fa6ce8` in private GitHub Actions run `29894376197`. The read-only Ubuntu job passed its pinned-tool preflight, two-clean-root producer/contract proof, supported source fixture, direct unsigned native-snapshot fixture, clean equivalent targets, authority-free update dry-run, focused offline consumer/update/rollback tests, and explicit cleanup. It uploaded zero artifacts; the repository remained private and no `v0.69.0` tag or Release existed. Checkpoint B then passed on macOS `26.3.1` (Darwin `25.3.0`)/arm64 at the same immutable commit with Go `1.26.5`: the pinned snapshot and committed verifier, supported source install, unsigned native archive, equivalent clean targets, authority-free dry-run, and exact five offline consumer/install/update/rollback tests passed. The aggregate wrapper exited `1` only because best-effort cleanup `chmod` warned on read-only module-cache entries; an independent postcondition proved the exact root absent and the repository clean, so Engineer, QA, and Security accepted the combined evidence without claiming a zero wrapper exit. Rejected harness attempts and exact accepted hashes are classified in T-068. Checkpoint C selected `0.69.0` without repository writes and removed 22 exact T-068 temporary roots. T-069 then restored the supported Linux gate at `03008f7`: exact-head run `29898672813` passed Go `1.25.12`, Go `1.26.5`, and the expected Go `1.25.11` rejection. Checkpoint C final state/sign-offs, fresh packaged bootstrap, and public-cutover prerequisites remain pending, so F-018-S003 is not yet passed.
- **F-018-S004:** Deferred to the F-017 cutover; no public release is authorized by this feature alone.

## Out of Scope

- Publishing raw binaries or `mars-harness-*` compatibility aliases.
- Imposing GoReleaser on generated target repositories.
- Homebrew, container images, Windows packages, notarization, or public announcement in the first release.
- Reusing `v0.93.0` or silently selecting a replacement if GitHub rejects the owner-selected `v0.69.0` name.

## Descoped Scenarios

- Legacy v0.93 raw-asset compatibility is deliberately not retained; affected private installations require one manual reinstall.
- Homebrew, container, Windows, and notarized macOS distribution may be planned only after the initial supported archive lifecycle passes.
