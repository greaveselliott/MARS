# T-078 Conventional No-Publish Rehearsal — Pass

**Date:** 2026-08-24  
**Ticket:** T-078  
**Scenarios:** F-017-S003, F-018-S004  
**Source:** `d411cbe38128b184d10904c50126caad1cca03fc`  
**Status:** `checkpoint_passed`  
**Primary Status:** `primary_blocked`

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project through
the shortest credible conventional producer, attestation, and consumer path.

**Primary Pass Gate:** the repository is public; attested `v0.69.1` is the
supported release with attested `v0.69.0` retained only as its rollback bridge;
the anonymous lifecycle, contribution controls, public security surfaces, and
48-hour canary pass before announcement.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** the current hosted-state seal and a separately
authorized sanitation/future-only immutable-Release transaction are incomplete;
public visibility, real attestations, launch Releases, and canary remain later
gates.

**Next Primary Action:** reacquire and scan the complete current hosted surface,
freeze exact live IDs and consequences, and present the owner with the bounded
hosted transaction for separate approval.

**Supporting Evidence:** the two definitive distributions, exact tool and image
identities, independent verifier results, and affected normal/race gates below.

## Outcome

The conventional AD-315 producer and independent verifier passed twice from
distinct clean source roots. Each run produced exactly nine unsigned files:
four deterministic platform archives, four SPDX-JSON SBOMs, and one canonical
checksum file. The archives were byte-identical across runs. Raw SBOM bytes and
therefore raw checksum files differed only on the producer fields that the
repository verifier explicitly normalizes; the normalized SBOM semantics were
identical.

The standard public GitHub-attestation fixture also passed the consumer policy.
It remains format/policy regression evidence only. No real attestation bundle
was created because private no-publish rehearsal has no OIDC or publication
authority. The eventual public Release still requires the tenth uploaded asset,
`checksums.txt.sigstore.json`, from the pinned GitHub attestation job.

This completes T-078 Checkpoint C. It does not complete T-078 or authorize a
tag, attestation, upload, Release, visibility change, deletion, settings
mutation, or announcement. The next launch-critical work is the current hosted
state seal and a separately approved exact sanitation/immutable-Release
transaction.

## Boundaries

- The repository remained private and `VERSION` remained `0.68.49`.
- Dependency staging used only the public Go proxy and SumDB. Both producer
  executions used Docker `--network none`.
- Producer containers received no GitHub token, secret, OIDC variable, SSH
  agent, Cosign key, signing authority, or publication command.
- Source and module-cache mounts were read-only. Each run had a distinct clean
  source clone, module cache, build cache, Syft binary, and output root.
- No GitHub API mutation, tag, attestation, upload, Release, deployment,
  workflow mutation, settings change, or visibility change occurred.
- The official container image was used through the ordinary Docker CLI; no
  custom Engine API, ptrace, Landlock, executable-format, or bespoke SPDX
  infrastructure was added or invoked.

## Exact Inputs

| Input | Identity |
|---|---|
| Source | `d411cbe38128b184d10904c50126caad1cca03fc`, commit time `2026-08-24T22:29:51+01:00`, normalized producer time `2026-08-24T21:29:51Z` |
| Container | `golang:1.27.0-bookworm` manifest `sha256:484ef6066fa69acb059fdfeda7ba2b8f7391f2ef6abc6f9b8411e669ebd56466`, image ID `sha256:6efd57250189d8a71bff8534ea7861892200358b90934b3ee93bc5bfeb874aa5`, Linux arm64 |
| Go | `go1.27.0 linux/arm64`, binary SHA-256 `b51e8499a917e56a0b290e2ab3ba96f11715dc47ad9739d307e03708e630343a` |
| Syft | binary SHA-256 `4da6c2b575afeb1108b04f22b6abc4fb33d65d6fe54bfb3f3ba20cbb1e35296d`; Go `1.27.0`; CGO disabled; `github.com/anchore/syft v1.51.0`; SumDB `h1:0AZveyFLkCK96uk/ykoRo1ScJ7qJ+/SgNbv7XRmJcGU=` |

The two independently built Syft binaries were byte-identical. Both definitive
source clones remained clean after production, and their source and output
directory identities were distinct.

## Rehearsal-Found Corrections

The rehearsal found and closed two ordinary producer defects before the
definitive runs:

1. `checksums.txt` was sorted by digest rather than artifact name. Commit
   `3e164525eec80818dc3344330624f46a2c3588eb` changed the GNU sort key to the
   canonical filename field and added the owning contract assertion.
2. The producer validated but did not normalize an RFC3339 commit time before
   embedding it. A `+01:00` CLI identity therefore disagreed with the
   authenticated UTC VCS time. Commit
   `d411cbe38128b184d10904c50126caad1cca03fc` normalizes once to UTC before
   building and archiving and adds the owning contract assertion.

An initial Syft link also exhausted Docker's writable layer. Moving only Go's
temporary linker workspace onto the lane-local cache volume resolved the host
capacity issue. The versioned static tool was then installed through the
supported `go install ...@v1.51.0` path so BuildInfo retained the exact module
version. These were staging corrections, not new release infrastructure.

## Definitive Artifact Evidence

The four archive digests were identical in both runs:

| Archive | SHA-256 |
|---|---|
| `mars_0.68.49_darwin_amd64.tar.gz` | `9991c568f68e630d604423100c5b33d17c2b5fb835af05ef814b8901317af397` |
| `mars_0.68.49_darwin_arm64.tar.gz` | `0a225b68c6a3a1ff87a2373709161ddea85483dc3cc0aec2b3d0aad05299376f` |
| `mars_0.68.49_linux_amd64.tar.gz` | `9455fc565d4c516ecd704ab269eab2a0daaa44d73a08e1af8e2cba44760330af` |
| `mars_0.68.49_linux_arm64.tar.gz` | `0393a0040ac4535215672c037421450d1d587fa854b186381a873059076739bd` |

Raw per-run evidence, retained under the owner-only temporary rehearsal root:

| File | Lane A SHA-256 | Lane B SHA-256 |
|---|---|---|
| `checksums.txt` | `7f08fc8691fd6f54abb36a95d6ba3c339cd6c6671ad1de14620b7ebe74cfd45b` | `5627a40bba77b85bdcb6be82aae26da0b11f62b00a1754ee9a42025cb22c18cb` |
| `darwin_amd64.tar.gz.sbom.json` | `971636cec253889ea2d1ab835cc50c07cbb6fd83e6a8c2f9d002fbd2ea885c18` | `09c87681001596b53803833cd241ea653517ffb2e2af0a950a56984a4806e283` |
| `darwin_arm64.tar.gz.sbom.json` | `ce55a11ebe09722908045ab87ff8df7e21387c5a10f7dcf66a855e89a20e08f8` | `411e12b4df5a4868a14ba4d6474d802294594796345ae45c5bcf5b46418c775e` |
| `linux_amd64.tar.gz.sbom.json` | `a5e0cc8d7a80bcc85e3ab8686e00f40686c3f07342bcd839c81c3f9d8320cbb0` | `f95a8738a52c7e8a9d683df04c2cbd56a56ba4ad891cc7974c33de72adb922b1` |
| `linux_arm64.tar.gz.sbom.json` | `6610c70e8576d0720f6cf5942936bba4180cbfc12eb0d4ba9c7360eb6b2f7fef` | `fa96cd700c43008b4d4b5c3b2962eb3cbc501086efaf69314d889b0ec55888cd` |

Raw SBOM byte inequality is expected because Syft emits an approved fresh
namespace and creation time. The independent verifier required every local raw
digest to match its own checksum file, then compared normalized semantics
without treating those volatile fields as release authority.

## Independent Gates

- Exact retained Go 1.27.0:
  `TestVerifyReleaseDistFromEnvironment` — PASS for both distinct roots,
  including the native Darwin arm64 `mars --version` execution.
- Exact retained Go 1.27.0:
  `TestVerifySigstoreProvenanceEvidenceRealOfficialFixture` — PASS; explicitly
  non-release public fixture evidence.
- Exact retained Go 1.27.0:
  `go test -count=1 ./internal/release ./internal/selfupdate` — PASS outside
  the app sandbox for the self-update suite's intentional owner-local
  replacement fixtures.
- Exact retained Go 1.27.0:
  `go test -count=1 -race ./internal/release ./internal/selfupdate` — PASS
  under the same owner-local test boundary.
- `bash -n scripts/release-produce.sh`, focused producer contract tests, and
  `git diff --check` — PASS before both corrections were committed and pushed.

## Next Action

Reacquire and scan the complete current hosted surface, freeze exact live IDs
and consequences, and present the owner with the separately authorized
sanitation and future-only immutable-Release transaction. Do not expand the
producer or activate the dormant release workflow during that work.
