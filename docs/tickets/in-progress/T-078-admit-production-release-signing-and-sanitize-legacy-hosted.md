---
id: T-078
title: Admit production release signing and sanitize legacy hosted objects
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S003", "F-018-S004"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md#f-017-s003-anonymous-immutable-verifiable-release-lifecycle", "docs/features/F-018-goreleaser-distribution.md", "docs/validation/reports/2026-08-08-github-hosted-publication-surface-audit.md", "docs/validation/reports/2026-08-24-t077-bootstrap-setup-closure.md", "docs/validation/reports/2026-08-24-t078-release-production-admission-blocked.md"]
verified_by: "pending"
owner: "foundation-maintainer"
last_attempt: "2026-08-24: exact SumDB GoReleaser v2.17.1 built with Go 1.26.5 reported 12 called vulnerabilities before execution; the exact 65-run delta at 466 live runs was separately acquired and scan-clean"
blocker: "Producer admission is blocked until the plan selects a current supported Go patch and exact scan-clean GoReleaser release/dependency graph or records a separately authorized qualified vulnerability disposition. Hosted proof also remains blocked on GitHub Billing & plans. Destructive hosted cleanup and the immutable-Release setting mutation remain denied until separately recorded exact owner approval names each admitted transaction."
blocked_by: []
trace_id: "launch-release-production:2026-08-24"
next_action: "Replan the exact production toolchain and producer selection, then repeat provenance, BuildInfo, structured called-symbol scanning, and two-root gates before implementation or execution; acquire every run after the 466-run scan-clean snapshot before any separately approved cleanup."
dedupe_key: "open-source:release-production-and-legacy-sanitation"
metadata:
  baseline_tag_count: "301"
  classification: "foundation-owned-and-hosted-evidence"
  expected_post_t078_asset_count: "0"
  expected_post_t078_deployment_count: "0"
  expected_post_t078_release_count: "56"
  legacy_asset_count: "500"
  live_run_count: "466"
  mutation_authority: "repository-source-tests-docs-only-until-separate-exact-hosted-cleanup-and-setting-approval"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  producer_admission_status: "blocked_called_vulnerabilities"
  run_delta_count: "65"
  sealed_run_count: "401"
  supports: "F-017-S003,F-018-S004"
source: MARS Launch-Complete Open-Source Delivery Plan — T-078
created: 2026-08-24
depends_on: [T-077]
---

# T-078: Admit production release signing and sanitize legacy hosted objects

## Context

T-077 closed anonymous setup, explicit third-party download acknowledgement, and the exact Go/SumDB bootstrap into the existing signed updater. T-078 owns production admission for the repository-pinned GoReleaser/Syft/Cosign chain, the protected final release artifact contract, and sanitation of obsolete private hosted release surfaces before T-079. The repository remains private at VERSION=0.68.49 and source fallback 0.69.0-dev.

The T-072 sealed cleanup boundary contains 301 tags, 57 Release objects, 500 legacy assets, 401 completed workflow runs, and 77 deployments. The 2026-08-24 creation preflight found 462 live completed runs; T-077 closeout found 465; and the T-078 read-only acquisition froze 466. The exact 65-run delta at that snapshot is acquired and scan-clean, but every later run must also be acquired, scanned, reconciled, and resealed before any run deletion. Absent a separately authorized third launch tag, the later T-080 convergence is 303 tags, not 304: the 301-tag baseline plus v0.69.0 and v0.69.1.

GitHub-hosted jobs currently fail before step one with the Billing & plans annotation recorded in the T-077 closure report. Local producer implementation is additionally stopped: exact GoReleaser v2.17.1 built with the pinned Go 1.26.5 toolchain reported 12 called vulnerabilities before execution. Hosted workflow proof cannot pass until the owner resolves its external condition, and producer work cannot resume until the exact selection is replanned and scan-clean or a separately authorized qualified disposition exists.

## Scope And Authority

Keep the repository private and retain the launch version freeze. T-078 may change repository-owned producer configuration, protected workflows, verification tests, release doctrine, and durable evidence. It may perform read-only hosted acquisition and reconciliation.

T-078 does not authorize v0.69.0/v0.69.1 tags, real launch signing, uploads, supported Releases, visibility changes, Pages, announcements, GitHub App mutations, or hosted settings changes. It also does not authorize deletion merely because the plan names cleanup. Before deleting any Release, asset, workflow run, deployment, or other hosted object, or enabling future-only immutable Releases, require a separately recorded owner approval naming the exact admitted object classes/counts and setting transaction; revalidate every live identifier and current setting against the refreshed sealed set immediately before mutation and stop on drift.

## Checkpoint A — Admit A Current Producer And Inert Signer Contract

- Replan and select an exact current supported Go patch plus exact GoReleaser, Syft, and Cosign versions whose dependency graphs can pass the production gate. Admit them through authoritative provenance/checksum/signature or exact-module/SumDB paths, record binary SHA-256 values, and parse structured scans for called symbols before any execution.
- Make the producer workflow build with CGO disabled through that admitted supported Go toolchain, generate four platform archives and one SPDX SBOM per archive, and emit checksums without OIDC or write authority.
- Keep keyless Cosign signing/publishing in a separate minimum-permission job that is structurally dormant. It may be described as protected or maintainer-only only after T-079 establishes and revalidates the live environment, ruleset, tag, and installed-App controls; T-080 alone activates it under separate release authority. Untrusted pull requests and forks must never receive OIDC, signing, release-write, package-write, or secret authority.
- Prove a two-root verification boundary: producer provenance admits the tools, while independently implemented consumer checks validate the outputs. Do not accept a workflow's self-assertion as verification.
- Use only synthetic/local rehearsal identities in T-078. Real launch tags and signatures remain T-080.

## Checkpoint B — Refresh And Sanitize Hosted State

- Acquire every completed workflow run added after the T-072 401-run seal, including attempts, jobs, logs, artifacts, and applicable metadata. Run the admitted offline scanners with zero errors/skips and classify every candidate without publishing candidate data.
- Reconcile the refreshed exact run set to the live repository and seal its sorted identifier digest. The 65-run delta frozen at 466 live completed runs on 2026-08-24 is acquired and scan-clean; do not delete any run until every later delta is also acquired and scan-clean.
- Prepare the exact transaction for all 500 legacy assets, 77 deployments, the one obsolete Release object, refreshed obsolete workflow runs, and any independently proven obsolete hosted object. Preserve all 301 historical tags and retained historical Release notes unless the separately approved transaction says otherwise.
- After exact owner approval, if supplied, revalidate live identifiers/counts and current settings, delete only the approved exact set, enable future-only immutable Releases only when the approval separately names that setting change, verify all postconditions, and record failure/rollback truth. Reconciliation must precede both mutations; do not claim that past provider storage or third-party caches were physically erased.
- The expected admitted post-T-078 active surface is 301 tags, 56 historical Release objects, zero assets, and zero deployments, plus only workflow runs deliberately retained by the approved transaction. Any different count is a stop condition, not a rounding adjustment.

## Checkpoint C — Prove The Final Artifact Contract

Each release must contain exactly ten uploaded Release assets: four platform archives, four SPDX SBOMs, checksums.txt, and checksums.txt.sigstore.json. GitHub-generated source archives and an immutable-Release attestation are additional provider surfaces, not uploaded assets and not part of the 10/20 arithmetic. The four supported platforms are Darwin AMD64, Darwin arm64, Linux AMD64, and Linux arm64.

A bounded rehearsal must independently verify archive naming, contents, executable mode, OS/architecture, Go build metadata, version binding, clean VCS state, checksum coverage, SBOM subject/digest binding, provenance, signature bundle structure, certificate/issuer/identity policy, and zero extra or missing assets. Verify from two clean roots and reject moved tags, mutable aliases, unsigned checksums, mismatched subjects, duplicate names, unexpected formats, and producer/consumer agreement that is not independently grounded.

Document the future T-080 convergence: two exact-ten launch Releases create 20 uploaded Release assets total, 58 Release objects total, and 303 tags total. v0.69.0 is only the rollback bridge; v0.69.1 is the supported release.

## Blocked Admission Evidence — 2026-08-24

Exact SumDB GoReleaser v2.17.1 built with CGO disabled and Go 1.26.5 had canonical BuildInfo and zero replacements, but exact govulncheck v1.6.0 against the 2026-08-21 official database found 12 called vulnerability IDs and 104 terminal symbols before execution. Two records have no fixed dependency version, two require newer dependencies, and eight require a newer Go patch. Go 1.26.7 is already the current 1.26 patch. The structured JSON scan exited zero despite the called findings, so a future gate must parse and reject called traces rather than trust exit status. GoReleaser was never executed. Exact Syft v1.50.0 was built and its canonical BuildInfo inspected, but it was not scanned or executed after the shared producer stop; Cosign and the workflow/rehearsal were not attempted.

Independently, the oldest 401 live run IDs reproduced T-072's exact sealed digest. The exact 65-run delta at 466 completed and zero active runs was acquired as 65 attempts, 252 jobs, 65 valid log ZIPs, and zero artifacts. Both exact previously admitted offline scanners returned zero findings, errors, skips, timeouts, rejected inputs, or unresolved candidates. This is supporting scan-clean evidence only; the report commit and every later run remain in scope. Exact identities, hashes, findings, controls, counts, and cleanup status are recorded in `docs/validation/reports/2026-08-24-t078-release-production-admission-blocked.md`.

## Validation And Evidence

Run affected normal/race tests, vet, workflow/config validation, DocSync/docs consistency, secret scanning, four CGO-disabled builds, local protected-workflow rehearsal, and independent artifact verification. Once GitHub billing is repaired, rerun the exact current commit in all hosted source/provenance lanes and bind the run IDs. Record the private artifact manifest, tool provenance, refreshed cleanup digest/counts, separately approved mutation record if any, postconditions, and QA/Security/Dogfood/Release Manager/Orchestrator sign-off.

## Acceptance

- The exact admitted current GoReleaser/Syft/Cosign binaries and dormant producer/signing workflow pass provenance, structured called-symbol scan, permission, and two-root verification gates.
- A synthetic rehearsal produces and independently verifies exactly ten uploaded Release assets with no mutable or unsigned path.
- The complete post-T-072 workflow delta is acquired and scan-clean before the cleanup set is refreshed.
- No hosted deletion or immutable-Release setting change occurs without separate exact owner authority naming that mutation; any authorized transaction is live-revalidated, exact, auditable, and reaches the expected postcondition.
- GitHub-hosted CI is green on the accepted source after the owner resolves Billing & plans.
- The repository remains private; VERSION stays 0.68.49; no launch tag, signature, upload, supported Release, visibility change, or announcement occurs.
- T-078 may close private production admission while F-017-S003 and F-018-S004 remain incomplete pending T-080/T-081 real release and public lifecycle proof.

## Stop Conditions

Stop on provenance ambiguity, any called producer-tool vulnerability without an authorized qualified disposition, unparsed structured-scan findings, an unsupported/stale production toolchain, unscanned producer binaries, self-verifying outputs, fork/OIDC/release-write authority, an unexpected artifact, mutable tag/ref, unsigned or mismatched checksum/SBOM/provenance, stale cleanup identifiers, scanner error/skip, unapproved deletion or immutable-Release setting mutation, hosted billing failure treated as green, changed baseline arithmetic, launch tag/signing/upload/publication, or any claim that T-080/T-081 passed.
