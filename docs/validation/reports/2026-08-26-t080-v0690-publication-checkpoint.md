# T-080 v0.69.0 Publication Checkpoint

**Date:** 2026-08-26
**Ticket:** T-080
**Repository:** `greaveselliott/MARS`
**Release:** [`v0.69.0`](https://github.com/greaveselliott/MARS/releases/tag/v0.69.0)
**Release commit:** `8db7b82ea4013b7a9cf7f760129ee2815ca89103`

## Primary Outcome Contract

**Primary Outcome:** publish the immutable attested `v0.69.0` rollback bridge
without weakening artifact or attestation verification, then repair the narrow
workflow phase mismatch before `v0.69.1`.

**Primary Pass Gate:** `v0.69.0` is public and immutable with the exact ten
locally and remotely digest-matched assets; its tag resolves to the intended
commit; the verification mismatch is corrected and regression-tested before
the next tag.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** the anonymous first-install lifecycle proved that
GNU tar record padding in the immutable `v0.69.0` and `v0.69.1` archives is
rejected by the signed updater's canonical archive contract. Corrected
`v0.69.2` is public and passes first install; `v0.69.3`, followed by the
anonymous update and rollback replay, is still required before a supported
launch-pair claim.

**Next Primary Action:** generate and validate `v0.69.3`, publish it using the
standard corrected publisher, independently verify it, then replay anonymous
update to `v0.69.3` and rollback to `v0.69.2`.

**Supporting Evidence:** release run `32915168629`, locally downloaded attested
artifact `mars-release-attested-8db7b82ea4013b7a9cf7f760129ee2815ca89103`,
the exact hosted Release receipt, and the focused workflow regression gate.

## Public Cutover

The owner authenticated GitHub sudo mode and approved the frozen T-080
transaction. Repository `greaveselliott/MARS`, immutable ID `1279592869`, is
public. The disposable public rehearsal repository was deleted after its
Actions, attestation, security-control, and Pages checks passed.

The live repository was revalidated with:

- secret scanning and push protection enabled;
- private vulnerability reporting enabled;
- CodeQL default setup configured;
- Pages built from `main:/docs` at
  `https://greaveselliott.github.io/MARS/` with HTTPS;
- Actions limited to GitHub-owned actions, full-SHA pinning, a read-only
  default token, and approval for all external contributors;
- active ruleset `21491158`; and
- Dependabot security updates enabled.

GitHub Apps remain outside launch scope under the owner's recorded
disposition. The existing narrow vulnerability disposition was not expanded.
One open Dependabot alert in the notices-only tool module was not dismissed;
the source and hosted called-path vulnerability gates remain green.

## v0.69.0 Result

Annotated tag `v0.69.0` resolves to exact commit
`8db7b82ea4013b7a9cf7f760129ee2815ca89103`. Release workflow run
[`32915168629`](https://github.com/greaveselliott/MARS/actions/runs/32915168629)
completed production and GitHub keyless attestation, then stopped before its
publisher because the read-only verification job passed the ten-file attested
directory to the deliberately exact nine-file unsigned verifier. The verifier
correctly rejected the extra `checksums.txt.sigstore.json`; no Release existed
at that failure boundary.

The exact attested Actions artifact was downloaded to a fresh owner-only local
directory. With the bundle held aside, the nine-file distribution passed
`TestVerifyReleaseDistFromEnvironment` for version `0.69.0`, Go `1.27.0`, the
exact commit, and commit time. The restored ten-file directory then passed
`TestVerifyMARSReleaseAttestationFromEnvironment` for exact tag and commit.

A draft Release was created from the existing remote tag, the same ten verified
files were uploaded, and every hosted `sha256:` digest was compared with the
local byte digest before publication. The Release is now public, non-draft,
non-prerelease, and immutable with exactly ten assets. GitHub Release ID is
`376795141`; publication time is `2026-08-26T00:37:32Z`.

No tag, asset, or immutable Release object was replaced or rewritten. The
workflow correction holds the Sigstore bundle outside the unsigned verifier's
directory, restores it, and then runs the attestation verifier. The regression
test binds that order. This is the minimum launch-critical correction; it does
not revive the deferred bespoke release-security platform.

## v0.69.2 Corrected Rollback Result

The owner approved the corrected immutable `v0.69.2`/`v0.69.3` sequence.
Annotated tag `v0.69.2` resolves to
`28d31a5c9a6efaf05ebf507933d030499ec351d6`. Release workflow run
[`32918743269`](https://github.com/greaveselliott/MARS/actions/runs/32918743269)
passed production, GitHub attestation, and independent verification. Its old
publisher created one private draft, GitHub Release ID `376825184`, with the
exact ten verified assets, then failed because GitHub's tag lookup endpoint
does not resolve that draft. No artifact was changed. The draft's asset
digests were independently matched to the workflow output, the repository's
unsigned distribution verifier passed with the Sigstore bundle held outside
the nine-file directory, and all four archive consumers plus the attestation
consumer passed before that exact draft ID was published.

The bounded publisher correction at `f48eb8b` creates/uploads the draft in one
operation, locates the sole matching draft by list result, verifies its exact
asset set and digests, and publishes it by immutable release ID. Hosted source
compatibility, CodeQL, and Pages checks for that correction all passed. Release
`v0.69.2` is public, non-draft, non-prerelease, immutable, and latest only
temporarily pending `v0.69.3`.

A fresh credential-free, exact-tag clone under a cleared environment installed
`v0.69.2` into a private non-symlinked owner directory. The installed binary
reported `mars 0.69.2 darwin/arm64`, commit
`28d31a5c9a6efaf05ebf507933d030499ec351d6`, built
`2026-08-26T01:16:54Z`. This establishes the first-install part of the corrected
lifecycle; it does not yet establish update or rollback.

## Remaining T-080 Work

`v0.69.1` was subsequently produced, attested, and independently verified by
run `32916602734`, then published manually through the same draft, digest, and
immutable sequence after the checkout-free publisher could not use
`--notes-from-tag`. It is immutable and exact-ten, but the first real anonymous
bootstrap correctly rejected its archive before installation because GNU tar's
default record size added zero padding beyond the consumer's canonical two
terminator blocks. `v0.69.0` has the same structural mismatch. Both remain
authentic historical launch artifacts, but neither may be advertised as a
supported install or rollback target.

Remaining work is now:

1. Generate `v0.69.3` from the validated correction, publish it through the
   least-privilege draft-by-ID workflow, and independently verify all ten
   assets and its attestation.
2. From the retained anonymous `v0.69.2` install, update to `v0.69.3` and
   rollback to `v0.69.2`; preserve the immutable older releases untouched.
3. Record the final hosted-state receipt and hand off to T-081's public canary.

The Node 20 compatibility warning for the pinned upstream upload-artifact
action is post-launch pin maintenance because GitHub successfully ran the
action under Node 24 and it did not affect artifact bytes or authority.
