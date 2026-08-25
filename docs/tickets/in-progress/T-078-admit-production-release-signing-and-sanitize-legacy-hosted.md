---
id: T-078
title: Adopt the standard release path and sanitize legacy hosted objects
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S003", "F-018-S004"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md#f-017-s003-anonymous-immutable-verifiable-release-lifecycle", "docs/features/F-018-goreleaser-distribution.md", "docs/validation/reports/2026-08-24-owner-launch-dispositions.md", "docs/validation/reports/2026-08-24-t078-release-production-admission-blocked.md", "docs/validation/reports/2026-08-24-t078-conventional-no-publish-rehearsal.md", "docs/validation/reports/2026-08-24-t078-hosted-state-revalidation.md"]
verified_by: "pending"
owner: "foundation-maintainer"
last_attempt: "2026-08-25: the owner approved the bounded source-floor remediation from Go 1.25.12 to 1.25.13 without a vulnerability exception; implementation and hosted proof are current"
blocker: "The revised cleanup/future-only immutable-Release transaction remains separately approval-gated after deployment-state drift; hosted source compatibility must pass the approved Go 1.25.13 floor and intentional Go 1.25.12 rejection."
blocked_by: []
trace_id: "launch-release-production:2026-08-24"
next_action: "Raise only the current source/bootstrap floor to Go 1.25.13, retain Go 1.27.0 release production, prove Go 1.25.12 is rejected, push once, and require hosted source compatibility to pass without broadening the vulnerability disposition."
dedupe_key: "open-source:release-production-and-legacy-sanitation"
metadata:
  baseline_tag_count: "301"
  classification: "foundation-owned-and-hosted-evidence"
  expected_post_t078_asset_count: "0"
  expected_post_t078_deployment_count: "0"
  expected_post_t078_release_count: "56"
  legacy_asset_count: "500"
  last_sealed_run_count: "474"
  live_deployment_count: "0"
  mutation_authority: "repository-source-tests-docs-only-until-separate-exact-hosted-cleanup-and-setting-approval"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  producer_admission_status: "dormant_source_and_no_publish_rehearsal_passed"
  supports: "F-017-S003,F-018-S004"
source: MARS Launch-Complete Open-Source Delivery Plan — T-078
created: 2026-08-24
depends_on: [T-077]
---

# T-078: Adopt The Standard Release Path And Sanitize Legacy Hosted Objects

## Primary Outcome

Replace the rejected and over-expanded release route with the shortest
supported launch path: conventional Go production, upstream Syft SBOMs,
GitHub's standard keyless attestation action, independent verification, and an
exact least-privilege dormant workflow. Separately revalidate and sanitize the
legacy hosted surface without changing visibility or publishing a release.

## Owner Re-Baseline — 2026-08-24

The owner stopped further bespoke Docker Engine/API, ptrace/Landlock,
executable-format, SPDX-parser, and recursive proof-platform work. The frozen
source/evidence checkpoint is preserved off the launch branch and is
non-authorizing. New reviewer ideas in those areas are post-launch backlog by
default unless an existing launch acceptance criterion demonstrably requires
them.

The owner also:

- accepted the unresolved `MARS` name risk without trademark registration or
  counsel clearance;
- attested authority to publish the current repository and retained material;
- funded the GitHub account; and
- removed account-wide GitHub App administration from the MARS launch scope.

The exact dispositions are recorded in
`docs/validation/reports/2026-08-24-owner-launch-dispositions.md`.

## Scope And Authority

T-078 may change repository-owned release configuration, workflows, consumer
verification, tests, notices, doctrine, and durable evidence. It may perform
read-only GitHub acquisition and reconciliation.

T-078 does not authorize a launch tag, real attestation, upload, supported
Release, visibility change, Pages change, announcement, GitHub App mutation,
hosted deletion, or hosted settings mutation. Destructive cleanup and enabling
future-only immutable Releases require separately recorded exact approval,
immediate live revalidation, and bounded postcondition checks.

## Checkpoint A — Conventional Dormant Release Workflow

**Source status:** Complete at pushed commit `6c044c1`. The four jobs remain
structurally dormant; no tag, OIDC request, attestation, upload, Release, or
other GitHub mutation ran while establishing this checkpoint.

- Use the supported Go toolchain to produce CGO-disabled Darwin/Linux AMD64 and
  arm64 binaries.
- Create deterministic archives and SHA-256 checksums with ordinary supported
  commands; do not introduce a new producer service or privileged runtime.
- Use upstream Syft to create one SPDX-JSON SBOM per archive. Verify required
  subject/digest relationships without reviving the deferred exhaustive SPDX
  parser.
- Pin every third-party action by immutable commit SHA.
- Keep the producer/SBOM job at `contents: read` with no OIDC, secrets,
  environment, attestation, or publication authority.
- Keep the attestation job separate with only the minimum documented
  `id-token: write` and `attestations: write` authority.
- Keep publication separate with `contents: write` only after production,
  attestation, and independent verification succeed.
- Keep the workflow structurally dormant until the later owning ticket and
  separately approved public cutover activate it.

GitHub-hosted attestations run only after public visibility because GitHub
documents ordinary-plan support for public repositories while private/internal
support requires Enterprise Cloud. Private work uses a no-publish rehearsal of
the producer and verifier, not a fake attestation claim.

## Checkpoint B — Compatible Consumer And Exact Asset Contract

**Source status:** Complete at pushed commit `6c044c1`. The updater accepts the
standard GitHub `actions/attest` SLSA provenance envelope for the exact eight
checksummed subjects while retaining the older public GoReleaser fixture only
as an immutable regression vector.

Each launch Release contains exactly ten uploaded assets: four platform
archives, four SPDX SBOMs, `checksums.txt`, and
`checksums.txt.sigstore.json`. Provider-generated source archives and hosted
attestation records are separate GitHub surfaces and do not change that count.

The consumer independently verifies:

- the GitHub/Sigstore bundle and trusted root;
- expected repository, workflow, ref/tag, and immutable commit identity;
- the exact checksum subject set and every downloaded byte digest;
- bounded archive structure, executable mode, platform, Go BuildInfo, version,
  commit, and clean VCS metadata; and
- anonymous install, update, and rollback behavior.

The launch consumer must not depend on the deferred Docker/ptrace, exact
Mach-O/ELF layout, or transcript-pinned full-SPDX-grammar implementation.
Historical offline GoReleaser signature fixtures remain regression vectors and
must not be described as current MARS signing evidence.

## Checkpoint C — No-Publish Rehearsal

**Status:** Complete at exact pushed source
`d411cbe38128b184d10904c50126caad1cca03fc`.

Run the complete producer and independent verifier twice from clean source
roots without credentials, OIDC, uploads, tags, Releases, visibility changes,
or hosted attestation. Require the same intended subject digests and equivalent
SBOM semantics, then exercise the signed-consumer policy against a standard
public GitHub-attestation fixture or a repository-owned format fixture that is
explicitly non-release evidence.

Two distinct clean source roots and independent caches/tools produced the
exact nine unsigned files with networking disabled during production. All four
archives were byte-identical. Each run's raw SBOMs matched its own canonical
checksums, and normalized SBOM semantics matched across runs. Exact Go 1.27
verification, native `mars --version`, the standard public GitHub-attestation
fixture, and affected normal/race suites passed. No real bundle, tag, upload,
Release, visibility change, or hosted mutation occurred. Full evidence is in
`docs/validation/reports/2026-08-24-t078-conventional-no-publish-rehearsal.md`.

The implementation of Checkpoints A through C is timeboxed to one working day.
If it cannot pass within that bound, record the exact upstream action, private
attestation, or consumer-compatibility blocker and stop. Do not add another
security subsystem.

## Checkpoint D — Refresh And Sanitize Hosted State

The last read-only seal contained 301 tags, 57 Release objects, 500 legacy
assets, 466 completed workflow runs, 77 deployments, zero Actions artifacts,
and zero caches. These are historical counts, not mutation inputs.

Before any hosted mutation:

1. reacquire every current Release, asset, deployment, workflow run/attempt,
   artifact, and cache;
2. scan and classify every delta under the existing narrow disposition with
   zero errors, skips, or unresolved findings;
3. freeze the exact IDs/digests/counts and consequences;
4. obtain separate owner approval naming the object classes and immutable-
   Release setting transaction;
5. revalidate the exact live IDs and settings immediately before mutation; and
6. mutate only the approved set, then prove the exact postconditions.

Preserve all 301 historical tags and retained historical Release notes unless
the approved transaction explicitly says otherwise. The expected sanitation
target remains 301 tags, 56 historical Release objects, zero uploaded assets,
and zero deployments, plus only deliberately retained workflow runs.

## Validation

- affected normal/race tests and vet;
- workflow permission/event/ref/activation tests;
- four CGO-disabled builds and native `--version`/BuildInfo checks;
- no-publish two-root producer/consumer rehearsal;
- application vulnerability scan under the existing narrow disposition;
- third-party notice regeneration and validation;
- DocSync, docs consistency, secret/workflow checks, formatting, and diff
  checks; and
- hosted CI rerun on the accepted source, now that the owner reports funding is
  restored.

## Acceptance

- AD-315's dormant standard workflow and compatible consumer are committed and
  independently verified without launch authority.
- The no-publish rehearsal produces and verifies the exact nine unsigned
  subjects from two clean roots and proves the tenth bundle's consumer format
  against the standard public fixture; the real tenth asset remains T-080.
- Coverage and documentation gates pass; no sub-threshold floor is seeded.
- The hosted surface is fully current before any separately approved sanitation
  or immutable-Release mutation, and exact postconditions pass if that approval
  is supplied.
- The repository remains private and `VERSION` remains `0.68.49`; no real
  attestation, launch tag, upload, supported Release, visibility change, or
  announcement occurs in T-078.

## Stop Conditions

Stop on action/tool provenance ambiguity, an unhandled called vulnerability,
wrong-job authority, fork authority, mutable action refs, unexpected artifacts,
consumer/producer circularity, a failed no-publish rehearsal, stale hosted IDs,
scanner error/skip, unapproved hosted mutation, or pressure to revive the
deferred bespoke platform. Record a bounded blocker instead of expanding the
implementation.
