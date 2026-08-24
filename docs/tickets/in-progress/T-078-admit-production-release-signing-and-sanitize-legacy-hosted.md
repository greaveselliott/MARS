---
id: T-078
title: Admit production release signing and sanitize legacy hosted objects
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S003", "F-018-S004"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md#f-017-s003-anonymous-immutable-verifiable-release-lifecycle", "docs/features/F-018-goreleaser-distribution.md", "docs/validation/reports/2026-08-08-github-hosted-publication-surface-audit.md", "docs/validation/reports/2026-08-24-t077-bootstrap-setup-closure.md"]
verified_by: "pending"
owner: "foundation-maintainer"
last_attempt: "2026-08-24: creation preflight found 462 live completed workflow runs and the final closeout recheck found 465 versus the sealed 401-run T-072 set; hosted Actions fail before step one on GitHub Billing & plans"
blocker: "Hosted workflow proof is blocked until the owner resolves GitHub Billing & plans. Destructive hosted cleanup and the immutable-Release setting mutation remain denied until separately recorded exact owner approval names each admitted transaction."
blocked_by: []
trace_id: "launch-release-production:2026-08-24"
next_action: "Admit the pinned producer and protected signing workflow locally, acquire/scan/reseal the currently observed 64-run delta plus every later run, and prepare the exact hosted cleanup transaction without tags, signing, deletion, or publication."
dedupe_key: "open-source:release-production-and-legacy-sanitation"
metadata:
  baseline_tag_count: "301"
  classification: "foundation-owned-and-hosted-evidence"
  expected_post_t078_asset_count: "0"
  expected_post_t078_deployment_count: "0"
  expected_post_t078_release_count: "56"
  legacy_asset_count: "500"
  live_run_count: "465"
  mutation_authority: "repository-source-tests-docs-only-until-separate-exact-hosted-cleanup-and-setting-approval"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  sealed_run_count: "401"
  supports: "F-017-S003,F-018-S004"
source: MARS Launch-Complete Open-Source Delivery Plan — T-078
created: 2026-08-24
depends_on: [T-077]
---

# T-078: Admit production release signing and sanitize legacy hosted objects

## Context

T-077 closed anonymous setup, explicit third-party download acknowledgement, and the exact Go/SumDB bootstrap into the existing signed updater. T-078 owns production admission for the repository-pinned GoReleaser/Syft/Cosign chain, the protected final release artifact contract, and sanitation of obsolete private hosted release surfaces before T-079. The repository remains private at VERSION=0.68.49 and source fallback 0.69.0-dev.

The T-072 sealed cleanup boundary contains 301 tags, 57 Release objects, 500 legacy assets, 401 completed workflow runs, and 77 deployments. The 2026-08-24 creation preflight found 462 live completed runs; the final closeout recheck found 465. The currently observed 64-run delta plus every later run must be acquired, scanned, reconciled, and resealed before any run deletion. Absent a separately authorized third launch tag, the later T-080 convergence is 303 tags, not 304: the 301-tag baseline plus v0.69.0 and v0.69.1.

GitHub-hosted jobs currently fail before step one with the Billing & plans annotation recorded in the T-077 closure report. Local and synthetic producer work may proceed; hosted workflow proof cannot pass until the owner resolves that external condition.

## Scope And Authority

Keep the repository private and retain the launch version freeze. T-078 may change repository-owned producer configuration, protected workflows, verification tests, release doctrine, and durable evidence. It may perform read-only hosted acquisition and reconciliation.

T-078 does not authorize v0.69.0/v0.69.1 tags, real launch signing, uploads, supported Releases, visibility changes, Pages, announcements, GitHub App mutations, or hosted settings changes. It also does not authorize deletion merely because the plan names cleanup. Before deleting any Release, asset, workflow run, deployment, or other hosted object, or enabling future-only immutable Releases, require a separately recorded owner approval naming the exact admitted object classes/counts and setting transaction; revalidate every live identifier and current setting against the refreshed sealed set immediately before mutation and stop on drift.

## Checkpoint A — Admit The Pinned Producer And Signer

- Admit exact GoReleaser v2.17.1, Syft v1.50.0, and Cosign v3.0.6 through their authoritative provenance/checksum/signature or exact-module/SumDB paths. Record exact identities and binary SHA-256 values; scan the admitted binaries before use.
- Make the protected producer workflow build with CGO disabled through the repository-pinned supported Go toolchain, generate four platform archives and one SPDX SBOM per archive, and emit checksums.
- Keep keyless Cosign signing/publishing in a separate maintainer-only job with the minimum permissions and environment. Untrusted pull requests and forks must never receive OIDC, signing, release-write, package-write, or secret authority.
- Prove a two-root verification boundary: producer provenance admits the tools, while independently implemented consumer checks validate the outputs. Do not accept a workflow's self-assertion as verification.
- Use only synthetic/local rehearsal identities in T-078. Real launch tags and signatures remain T-080.

## Checkpoint B — Refresh And Sanitize Hosted State

- Acquire every completed workflow run added after the T-072 401-run seal, including attempts, jobs, logs, artifacts, and applicable metadata. Run the admitted offline scanners with zero errors/skips and classify every candidate without publishing candidate data.
- Reconcile the refreshed exact run set to the live repository and seal its sorted identifier digest. Do not delete any run until the currently observed 64-run delta and every later delta are acquired and scan-clean.
- Prepare the exact transaction for all 500 legacy assets, 77 deployments, the one obsolete Release object, refreshed obsolete workflow runs, and any independently proven obsolete hosted object. Preserve all 301 historical tags and retained historical Release notes unless the separately approved transaction says otherwise.
- After exact owner approval, if supplied, revalidate live identifiers/counts and current settings, delete only the approved exact set, enable future-only immutable Releases only when the approval separately names that setting change, verify all postconditions, and record failure/rollback truth. Reconciliation must precede both mutations; do not claim that past provider storage or third-party caches were physically erased.
- The expected admitted post-T-078 active surface is 301 tags, 56 historical Release objects, zero assets, and zero deployments, plus only workflow runs deliberately retained by the approved transaction. Any different count is a stop condition, not a rounding adjustment.

## Checkpoint C — Prove The Final Artifact Contract

Each release must contain exactly ten assets: four platform archives, four SPDX SBOMs, checksums.txt, and checksums.txt.sigstore.json. The four supported platforms are Darwin AMD64, Darwin arm64, Linux AMD64, and Linux arm64.

A bounded rehearsal must independently verify archive naming, contents, executable mode, OS/architecture, Go build metadata, version binding, clean VCS state, checksum coverage, SBOM subject/digest binding, provenance, signature bundle structure, certificate/issuer/identity policy, and zero extra or missing assets. Verify from two clean roots and reject moved tags, mutable aliases, unsigned checksums, mismatched subjects, duplicate names, unexpected formats, and producer/consumer agreement that is not independently grounded.

Document the future T-080 convergence: two exact-ten launch Releases create 20 assets total, 58 Release objects total, and 303 tags total. v0.69.0 is only the rollback bridge; v0.69.1 is the supported release.

## Validation And Evidence

Run affected normal/race tests, vet, workflow/config validation, DocSync/docs consistency, secret scanning, four CGO-disabled builds, local protected-workflow rehearsal, and independent artifact verification. Once GitHub billing is repaired, rerun the exact current commit in all hosted source/provenance lanes and bind the run IDs. Record the private artifact manifest, tool provenance, refreshed cleanup digest/counts, separately approved mutation record if any, postconditions, and QA/Security/Dogfood/Release Manager/Orchestrator sign-off.

## Acceptance

- The exact pinned GoReleaser/Syft/Cosign binaries and protected producer/signing workflow pass provenance, scan, permission, and two-root verification gates.
- A synthetic rehearsal produces and independently verifies exactly ten contract assets with no mutable or unsigned path.
- The complete post-T-072 workflow delta is acquired and scan-clean before the cleanup set is refreshed.
- No hosted deletion or immutable-Release setting change occurs without separate exact owner authority naming that mutation; any authorized transaction is live-revalidated, exact, auditable, and reaches the expected postcondition.
- GitHub-hosted CI is green on the accepted source after the owner resolves Billing & plans.
- The repository remains private; VERSION stays 0.68.49; no launch tag, signature, upload, supported Release, visibility change, or announcement occurs.
- T-078 may close private production admission while F-017-S003 and F-018-S004 remain incomplete pending T-080/T-081 real release and public lifecycle proof.

## Stop Conditions

Stop on provenance ambiguity, unscanned producer binaries, self-verifying outputs, fork/OIDC/release-write authority, an unexpected artifact, mutable tag/ref, unsigned or mismatched checksum/SBOM/provenance, stale cleanup identifiers, scanner error/skip, unapproved deletion or immutable-Release setting mutation, hosted billing failure treated as green, changed baseline arithmetic, launch tag/signing/upload/publication, or any claim that T-080/T-081 passed.
