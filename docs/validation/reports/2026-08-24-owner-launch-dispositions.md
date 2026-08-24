# Owner Launch Dispositions — 2026-08-24

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project through
the shortest credible launch-critical path while preserving the already
completed security work as a non-authorizing checkpoint.

**Primary Pass Gate:** the repository is public; attested `v0.69.1` is the
supported release with attested `v0.69.0` retained only as its rollback bridge;
the anonymous lifecycle, contribution controls, public security surfaces, and
48-hour canary pass before announcement.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** AD-315's standard workflow and compatible
consumer, the no-publish rehearsal, hosted sanitation and immutable-Release
transaction, private contribution controls, approved visibility change, two
attested releases, and canary are incomplete.

**Next Primary Action:** commit and verify the timeboxed dormant AD-315
producer/attestation/consumer path without reviving the bespoke platform.

**Supporting Evidence:** the owner decisions below close the name-risk,
publication-authority, account-funding, App-scope, and producer-selection
planning questions without authorizing any hosted mutation or release.

## Owner Decisions

The repository owner made the following launch decisions in the active launch
task on 2026-08-24:

1. **Name risk accepted.** The owner will not register `MARS` as a trademark
   and accepts the recorded unresolved name-conflict risk. Qualified trademark
   counsel clearance is removed from the launch acceptance criteria. This is a
   risk disposition, not a finding that the name is clear or registrable.
2. **Publication authority attested.** The owner stated:

   > I control greaveselliott/MARS and am authorized to publish its current
   > source, documentation, release artifacts, retained history, first-party
   > material, and Cursor/automation-assisted material. I authorize the private
   > signed releases and subsequent public publication, subject to the remaining
   > launch gates.

3. **Billing prerequisite fulfilled.** The owner funded the GitHub account.
   Hosted jobs still need an ordinary successful rerun, but funding is no
   longer a planning blocker.
4. **GitHub App administration is not a MARS launch gate.** Account-wide App
   installation scope is unrelated to this repository unless a workflow or
   launch control actually depends on that App. No further account-wide App
   mutation is authorized by the launch plan.
5. **Bespoke T-078 route stopped.** Custom Docker Engine/API orchestration,
   ptrace/Landlock enforcement, executable-format parsing, transcript-pinned
   SPDX parsing, and recursively added proof machinery are preserved as a
   reviewable local checkpoint and classified as post-launch work by default.
6. **Conventional release path approved.** The launch producer will use the
   supported Go toolchain and ordinary archive/checksum commands, upstream Syft
   for SBOM generation, and GitHub's standard `actions/attest` action for
   keyless build provenance. Release automation must pin third-party actions by
   immutable commit, split build/attestation/publication authority, and keep
   untrusted forks authority-free.

## Approved Sequence

1. Preserve the bespoke T-078 work without merging it into the launch diff.
2. Replace the stale producer contract with a conventional, least-privilege,
   dormant workflow and compatible release consumer.
3. Run a no-publish local/hosted rehearsal and independent artifact checks.
4. Revalidate and, only under separately recorded exact authority, sanitize
   the legacy hosted objects and enable future-only immutable Releases.
5. Complete private contribution/governance gates and final pre-cutover checks.
6. Under separate visibility approval, make the repository public, immediately
   enable and verify the public-only security controls, then run the keyless
   public attestation path for `v0.69.0` and `v0.69.1`.
7. Verify anonymous install/update/rollback and hold the 48-hour canary before
   announcement.

GitHub-hosted attestations are deliberately scheduled after visibility because
GitHub documents artifact attestations for public repositories on current
plans, while private/internal repository support requires GitHub Enterprise
Cloud. The implementation must pin and validate the exact action and bundle
contract before activation.

## Safety Boundaries Retained

- This report authorizes repository planning/source changes only. It does not
  authorize a tag, Release, signature, upload, visibility change, deletion,
  immutable-Release setting mutation, Pages change, announcement, or GitHub App
  mutation.
- Hosted deletion and settings changes still require exact live revalidation
  and separately recorded approval naming the affected object classes and
  consequences.
- No vulnerability exception is broadened. The previously recorded narrow
  dispositions remain historical evidence and do not authorize execution in
  the new path.
- `VERSION` remains `0.68.49` until the launch release tickets deliberately end
  the freeze.

## References

- GitHub artifact attestations:
  <https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations>
- GitHub `actions/attest`:
  <https://github.com/actions/attest>
- Offline attestation verification:
  <https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/verify-attestations-offline>
