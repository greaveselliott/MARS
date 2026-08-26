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

**Current Primary Blocker:** land the verified workflow correction, publish
immutable `v0.69.1` through the corrected end-to-end workflow, and complete the
anonymous update/rollback lifecycle.

**Next Primary Action:** commit and push the verification-phase correction,
require hosted source gates to pass, generate the `0.69.1` release-note commit,
then create and monitor the exact `v0.69.1` tag.

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

## Remaining T-080 Work

1. Push the workflow correction and require the hosted source gates to pass.
2. Generate and push the `0.69.1` release-note commit.
3. Publish and independently verify immutable exact-ten `v0.69.1` as latest.
4. Verify anonymous download/install/update to `v0.69.1` and rollback to
   `v0.69.0` on the supported macOS/Linux lanes.
5. Record the final hosted-state receipt and hand off to T-081's public canary.

The Node 20 compatibility warning for the pinned upstream upload-artifact
action is post-launch pin maintenance because GitHub successfully ran the
action under Node 24 and it did not affect artifact bytes or authority.
