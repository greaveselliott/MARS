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
2. T-065 pins GoReleaser and the supporting supply-chain tools, defines the four archive targets, and removes the bespoke publication path.
3. T-066 migrates installation and updates to fail-closed archive, checksum, signature, identity, metadata, and extraction verification.
4. T-067 runs reproducible private snapshots and clean installation fixtures without creating a tag, Release, signature, or supported-release claim.
5. A separately approved F-017 cutover publishes the first immutable supported release only after every independent publication gate passes.

## Scenario Schedule

1. F-018-S001 - Produce deterministic GoReleaser snapshot archives and validate the signing configuration.
2. F-018-S002 - Install safely and migrate the binary updater to the archive contract.
3. F-018-S003 - Rehearse the complete pipeline privately without publishing.
4. F-018-S004 - Publish and verify an immutable public release after F-017 approval.

## Scenarios

### F-018-S001: Deterministic Release Production

Given an exact clean source commit and explicit non-release snapshot version
When pinned GoReleaser builds MARS twice with the pinned Go toolchain
Then Darwin and Linux AMD64 and ARM64 archives are byte-identical between runs
And each archive contains exactly the MARS binary, license, notice, and third-party notices
And each binary reports the exact version, full commit, platform, and commit-derived build date
And SPDX-JSON SBOMs and SHA-256 checksums are produced
And the checksum-signing configuration passes structural and synthetic-fixture tests without creating a public keyless signature
And raw binaries, legacy `mars-harness-*` aliases, stale files, and extra assets are absent.

### F-018-S002: Safe Archive Installation And Binary Updater

Given a supported release archive, checksum file, and signature bundle
When the installer or binary updater `mars update tool` selects the current platform
Then the expected workflow identity, signature, checksum, tag, commit, and platform are verified before replacement
And extraction rejects absolute or traversal paths, links, devices, duplicates, missing entries, unexpected entries, and quota violations
And the installed binary is replaced atomically only after every verification passes
And any failure preserves the previous binary and provides an actionable recovery command.

### F-018-S003: Private Rehearsal Without Publication

Given the repository remains private and the public cutover is not approved
When CI and Dogfood exercise the release pipeline
Then fork-safe CI runs `goreleaser check` and a clean snapshot without secrets or write authority
And two-build reproducibility, hostile artifact tests, clean macOS/Linux installation, and rollback pass
And ephemeral offline synthetic signed fixtures may exercise consumer verification without representing a MARS release
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

- **F-018-S001:** Pending T-065.
- **F-018-S002:** Pending T-066.
- **F-018-S003:** Pending T-067.
- **F-018-S004:** Deferred to the F-017 cutover; no public release is authorized by this feature alone.

## Out of Scope

- Publishing raw binaries or `mars-harness-*` compatibility aliases.
- Imposing GoReleaser on generated target repositories.
- Homebrew, container images, Windows packages, notarization, or public announcement in the first release.
- Reusing `v0.93.0` or silently selecting a replacement if GitHub rejects the owner-selected `v0.69.0` name.

## Descoped Scenarios

- Legacy v0.93 raw-asset compatibility is deliberately not retained; affected private installations require one manual reinstall.
- Homebrew, container, Windows, and notarized macOS distribution may be planned only after the initial supported archive lifecycle passes.
