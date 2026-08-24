# Release Publication

Use this source-only skill when Release Manager or the foundation maintainer
works on MARS release production. Generated target repositories do not inherit
this Go-specific producer; each target chooses and documents its own release
workflow.

## Current transition contract

1. Read the active exec plan, T-078, F-017, F-018, and AD-315 before changing
   release state.
2. Use the supported Go toolchain for four CGO-disabled builds, ordinary
   deterministic archive/checksum commands, upstream Syft for one SPDX-JSON
   SBOM per archive, and GitHub `actions/attest` for keyless provenance.
3. Pin every third-party action by full commit SHA. Keep build/SBOM production
   free of OIDC and write authority; keep attestation and publication in
   separate least-privilege jobs.
4. Require independent verification of the exact ten uploaded assets, subject
   digests, archive contents/platform metadata, workflow/ref/commit identity,
   and anonymous install/update/rollback behavior.
5. Complete a no-publish rehearsal and hosted sanitation while private. The
   real GitHub attestation run happens only after separately approved public
   visibility because ordinary-plan private repositories do not have the
   required hosted attestation service.
6. Record the immutable source commit, action SHAs, tool versions, artifact
   hashes, verifier result, and role sign-offs in the owning ticket and active
   plan.

Historical T-065 through T-068 GoReleaser evidence remains valid for the
archive and consumer invariants it proved. The preserved bespoke T-078
checkpoint is non-authorizing and is not a launch dependency.

## Safety

- Keep no-publish rehearsal jobs credential-free and deny OIDC, attestation,
  release-write, upload, announcement, and publication authority.
- Keep third-party notices and the application vulnerability gate current; an
  SBOM does not replace required notice text.
- Do not reintroduce the deferred custom Docker Engine, ptrace/Landlock,
  executable-layout, or full-SPDX-parser route to close an ordinary launch
  gate.
- Never paste token values, signing material, local evidence paths, or raw
  scanner output into chat, docs, commits, traces, tickets, logs, or tool output.
- Never treat a process exit, draft, upload, or partial remote list as release
  completion. Future publication must independently verify the immutable tag,
  exact artifacts, signatures, attestations, and fresh downloads.

## Stop conditions

Stop and record a blocker if the source commit is dirty, pins or hashes drift,
the artifact contract fails, authority reaches the wrong job, a hosted
mutation lacks exact revalidation/approval, or any required F-017/F-018 gate is
unresolved. Keep the repository private and `primary_blocked` until the
separately approved visibility transaction.
