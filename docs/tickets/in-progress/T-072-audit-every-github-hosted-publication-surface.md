---
id: T-072
title: Audit every GitHub-hosted publication surface
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/tickets/done/T-071-restore-the-green-vulnerability-baseline-with-grpc-1-82-1.md"]
verified_by: "TBD"
owner: "security"
last_attempt: "2026-08-08: preflight confirmed private visibility and FileVault; hosted counts remain hypotheses and the 56-Release pagination result conflicts with an addressable zero-asset v0.65.7 Release"
blocker: "none"
blocked_by: []
trace_id: "github-hosted-audit:2026-08-08-preflight"
next_action: "Commit this ticket, wait for all Actions to become quiescent, freeze the exact hosted inventory, and acquire it read-only into the owner-only FileVault boundary."
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

## Non-Goals

No hosted deletion, settings change, source remediation, rights disposition, version/release work, visibility change, announcement, or publication.
