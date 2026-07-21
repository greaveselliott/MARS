# Release Publication

Use this source-only skill when Release Manager or the foundation maintainer
works on MARS release production. Generated target repositories do not inherit
this Go-specific producer; each target chooses and documents its own release
workflow.

## Current transition contract

1. Read the active exec plan, T-065 through T-067, and F-018 before changing
   release state.
2. Build exact GoReleaser and Syft pins with the required Go toolchain and
   verify their recorded provenance and vulnerability disposition.
3. Run only the publication-disabled snapshot producer defined by
   `.goreleaser.yaml` and `.github/workflows/release-snapshot.yml`.
4. Require the committed T-065 contract checker to accept the exact publishable
   set, archive structure, checksums, binary metadata, and normalized SBOM
   comparison from distinct clean roots.
5. Record the immutable source commit, tool versions, artifact hashes, verifier
   result, and persona sign-offs in the owning ticket and active plan.

The current producer must not create a tag, GitHub Release, upload, signature,
attestation, visibility change, or supported-release claim. Consumer and
signature verification belong to T-066; private end-to-end rehearsal belongs to
T-067; public publication requires the separate F-017 cutover approval.

## Safety

- Run GoReleaser without credentials and explicitly skip `ko`, signing,
  announcement, and publication while the recorded producer findings remain.
- Keep provisional third-party notices and the GoReleaser binary findings as
  public-cutover blockers; an SBOM does not replace required notice text.
- Never paste token values, signing material, local evidence paths, or raw
  scanner output into chat, docs, commits, traces, tickets, logs, or tool output.
- Never treat a process exit, draft, upload, or partial remote list as release
  completion. Future publication must independently verify the immutable tag,
  exact artifacts, signatures, attestations, and fresh downloads.

## Stop conditions

Stop and record a blocker if the source commit is dirty, pins or hashes drift,
the clean-root contract fails, a credential reaches the snapshot producer, a
tag or remote mutation is attempted, or any required F-017/F-018 gate is
unresolved. Keep the repository private and `primary_blocked`.
