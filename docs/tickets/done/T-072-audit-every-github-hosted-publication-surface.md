---
id: T-072
title: Audit every GitHub-hosted publication surface
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/validation/reports/2026-08-08-github-hosted-publication-surface-audit.md", "docs/tickets/done/T-071-restore-the-green-vulnerability-baseline-with-grpc-1-82-1.md"]
verified_by: "QA, Security, Release Manager, and Orchestrator — 2026-08-08"
owner: "security"
last_attempt: "2026-08-08: owner-authenticated UI resolved all four access gaps; the post-evidence workflow delta was acquired and scanned with zero findings or errors"
blocker: "none"
blocked_by: []
trace_id: "github-hosted-audit:2026-08-08-preflight"
next_action: "Commit and push closure, then create T-073 through ticket_create; retain the installed-App finding as a T-079/T-080 launch no-go."
dedupe_key: "open-source:github-hosted-publication-surface"
metadata:
  classification: "evidence-only"
  mutation_authority: "denied"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  supports: "F-017-S001"
source: MARS Launch-Complete Open-Source Delivery Plan — T-072
created: 2026-08-08
depends_on: [T-071]
---

# T-072: Audit every GitHub-hosted publication surface

## Context

T-070 passed only the advertised-Git surface. F-017-S001 still lacks complete coverage of retained GitHub-hosted records and payloads. Recorded counts—302 tags, 56 Releases, 500 assets, 392 workflow runs, 77 deployments, zero packages, one collaborator, and zero current artifacts/caches—are hypotheses to freeze and reconcile. REST pagination reported 56 Releases while direct lookup exposes an addressable zero-asset v0.65.7 Release; no Release count is accepted until REST, direct lookup, and addressable objects agree.

## Scope And Authority

This is read-only, evidence-only work. Keep the repository private. Do not change source, version, tags, Releases/assets, workflows, runs, deployments, Pages/Wiki, settings, visibility, security features, or announcement state. Use standard git, gh/API, Gitleaks, TruffleHog, archive, hash, and filesystem tools only. Do not create a MARS audit runtime, wrapper framework, VM, container, volume ceremony, or backup program.

Raw API bodies, logs, archives, attachments, scanner reports, candidate values, paths, names, URLs, and identifiers stay in an owner-home FileVault boundary with 0700 directories and 0600 files. Keep immutable inputs, provenance-checked tools, and outputs separate; never scan outputs or the boundary root. Repository evidence contains only snapshot/tool digests, coverage/error counts, broad classes/statuses, cleanup-manifest counts and digests, and opaque finding IDs.

## Checkpoints

1. Inventory and acquisition manifest: wait for Actions quiescence; freeze local/remote main, advertised refs, private visibility, repository timestamps, every hosted count/ID set, and feature/settings state. Collect every applicable surface and payload through fully paginated read-only APIs.
2. Offline scan and classification: admit exact Gitleaks v8.30.1 only after a non-stopword multi-detector synthetic canary; admit exact TruffleHog v3.95.9 from accepted provenance. Scan local acquired inputs without updates, verification, provider detectors, credentials, or network-backed targets.
3. Redacted report and cleanup manifests: reconcile every object/payload and scanner result; seal exact owner-only immutable-ID cleanup manifests for later T-078 deletion and bind only their counts and SHA-256 digests into the repository-safe report.
4. Closure: one bounded QA, Security, Release Manager, and Orchestrator review; move T-072 to done and hand directly to T-073.

Each checkpoint is committed and pushed before the next when it produces repository evidence. No repeated full source gate is required for evidence-only documentation diffs.

## Required Surfaces

- Repository identity/settings/features, advertised refs, branches/protection/rulesets, Actions permissions, and security configuration.
- Every Release record/note and asset payload by immutable ID.
- Workflow objects; all runs, attempts, logs, artifacts, caches, and stale workflow objects.
- Deployments/statuses, environments, Pages configuration/source/build/output, and Wiki state/history.
- Issues, pull requests, comments, reviews, reactions, Discussions/categories/comments, projects, attachments, collaborators, invitations, hooks, deploy keys, and App/install permissions.
- Repository/environment/Dependabot secret and variable names/counts only—never values.
- CodeQL/code-scanning, Dependabot, secret-scanning, push-protection, private-reporting configuration and accessible alert metadata.
- Package metadata/payload count for every applicable package type using existing authority only; zero requires a complete authoritative result.

## Freeze And Coverage Semantics

Capture all freeze facts before and after acquisition. Any ref/OID, repository timestamp, Release/asset set, workflow/run/attempt set, active Actions state, artifact/cache set, deployment/status set, Pages/Wiki identity, settings state, or surface-count drift invalidates acquisition and requires a quiescent restart.

A surface is collected only when every paginated record reconciles and each retrievable payload is hashed. confirmed_empty requires a complete authorized zero result. not_applicable requires an authoritative disabled/no-parent condition plus correlated absence. A 401/403, ambiguous 404/204, pagination mismatch, REST/GraphQL disagreement, rate limit, redirect/download failure, expired content without definitive absence, archive rejection, scanner error, or drift is unresolved, never clean.

## Security Rules

Supply credentials only to gh; never mount or pass them to scanners. Redirect connected acquisition output directly into owner-only files. Treat all archives and attachments as hostile: reject absolute/traversal paths, links, devices, FIFOs/sockets, encryption, unsupported formats, case collisions, excessive nesting/count/expansion, timeout, or quota breach. Any rejected or partial input remains unresolved.

A plausible credential stops acquisition and scanning immediately without reproducing, hashing, online verification, deletion, or continued scanning. Rotation requires separate owner approval before resumption.

## Acceptance

- Every enumerated surface is collected, confirmed_empty, or not_applicable with exact pagination and count reconciliation.
- Start/end freeze facts match and the Release-count discrepancy is resolved.
- Exact Gitleaks v8.30.1 and TruffleHog v3.95.9 provenance/canaries pass; scanner errors, skips, timeouts, rejected inputs, and unresolved findings are zero.
- Exact owner-only legacy asset, workflow-run/artifact/cache, deployment, and Pages cleanup manifests are sealed and digest-bound for T-078.
- Repository-safe evidence passes redaction, DocSync, QA, Security, Release Manager, and Orchestrator review.
- F-017-S001 and Primary Status remain blocked on T-073 rights, provenance, notices, name review, and owner disposition.

## Checkpoint Evidence — 2026-08-08

Snapshot `T072-723d689d-3194-40c8-9d92-322b921149a3` froze and reconciled
305 advertised refs, 57 Release objects, 500 exact asset payloads, five
addressable workflows, 400 completed runs, 400 attempts, 1,728 jobs, zero
artifacts/caches, 77 deployments, one environment, zero issues, and one pull
request. Every Release asset matched its exact size and provider digest; all
Actions log responses were valid ZIPs. The 56-versus-57 Release discrepancy is
resolved by the direct-only zero-asset record.

Pinned offline Gitleaks and TruffleHog lanes completed with zero scan errors,
skips, timeouts, rejected inputs, or unresolved candidates. Provider-response,
commit-identifier, dependency-test-filename, and already approved synthetic Git
fixture matches were resolved mechanically without placing candidate values in
the repository. Exact owner-only cleanup identifier lists are sealed and bound
by count and digest in the redacted report.

At this checkpoint, four surfaces remained unresolved rather than clean:
GitHub Apps, Packages, Projects v2, and enabled-but-uninitialized Wiki state
required one consolidated owner-authenticated read-only confirmation. T-072
remained in progress and authorized no hosted mutation.

## Closure Evidence — 2026-08-08

The owner-authenticated repository UI confirmed two installed third-party
GitHub Apps, zero packages, zero open or closed linked projects, and an enabled
but never-initialized Wiki. Exact UI evidence remains owner-only and its
redacted confirmation digest is
`e28c14addb6a2c1af096ba7b9f415490a9b0765de2632f0c107c7bc3c904421a`.

Both installed Apps have all-repository write authority and one has a pending
permission update. Opaque finding
`T072-6a87eaed-e746-41c1-bf0e-1e519fc66705` is durably classified as a
T-079/T-080 launch no-go requiring minimum-scope reconfiguration or removal
and owner verification before cutover. T-072 performed no configuration
mutation.

The first redacted evidence commit created one additional successful workflow
run. Its one attempt, three jobs, 27 extracted log files, and zero-artifact
result were acquired and scanned; both admitted scanners returned zero
findings and zero errors. The final exact run cleanup set contains 401 IDs with
SHA-256 `cdfb0cc95e6dc8f4e2b9e910f3c897834fa17d2d18f3a9e454c2b04db74b938b`.
All enumerated surfaces are now collected, confirmed empty, or not applicable;
scanner errors, skips, timeouts, rejected inputs, and unresolved secret
candidates are zero. T-072 passes without authorizing hosted mutation,
publication, visibility, or announcement.

## Non-Goals

No hosted deletion, settings change, source remediation, rights disposition, version/release work, visibility change, announcement, or publication.
