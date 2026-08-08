# Active P0 Exec Plan: Complete The Restricted Publication Audit

**Status:** Active
**Priority:** P0
**Depends On:** T-056, T-063, T-064, and T-068 complete
**Blocks:** F-017-S001 and every public cutover action
**Related Tickets:** T-056 and T-063 through T-070
**Current Ticket:** T-070 — scan the exact advertised Git publication surface with pinned standard tools
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-017-open-source-publication.md
**Related Feature Contracts:** F-001, F-017
**Hypothesis:** Scanning the exact advertised Git roots with pinned standard tools before collecting broader GitHub-hosted material will close the highest-reach secret-history risk without recreating an audit runtime.
**Success Evidence:** Both pinned scanners cover every object reachable from the frozen publication-ref manifest with zero errors, skips, unresolved findings, or raw-evidence leakage, while GitHub remains private and unchanged.
**Falsification Evidence:** A ref drifts, a scanner or object is skipped, a plausible credential appears, raw evidence escapes the owner-only boundary, or the ticket mutates GitHub state.
**Scenario Schedule:** T-070 advertised Git scan; then separately ticketed GitHub-hosted surface review; then manual privacy/IP/provenance/name review and owner disposition for F-017-S001.
**Current Failing Scenario:** F-017-S001 — the URI-detector group remains unclassified and the current advertised Git publication candidate has not passed the required newly frozen four-lane rescan.
**Walking Skeleton Slice:** Freeze the canonical publication refs, scan their reachable objects offline, and commit only redacted coverage and disposition status.
**Learning Or MVP Outcome:** Establish whether the retained Git history is technically eligible for later owner disposition before spending authority on broader hosted-surface acquisition.
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, Security, QA, Release Manager, and repository-owner disposition

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including anonymous signed install/update, fork-safe contribution, logged-out cutover smoke, and a clean 48-hour canary.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** Elliott classified the Gitleaks group `a2a292e31d652f22` as synthetic test stubs after owner-only review. The separate URI group `e32927624f4a2cac` remains unclassified; if it is real or uncertain, rotation or revocation is required before both pinned tools' Git-history and raw-object lanes rerun. GitHub-surface, manual disposition, runtime, contribution, cutover, and canary gates are also incomplete.
- **Next Primary Action:** Elliott reviews `e32927624f4a2cac` inside the owner-only boundary and reports only whether it is synthetic test stubs or real/unknown. If real/unknown, rotate or revoke it before the four-lane rescan. Do not publish, continue acquisition, or mutate GitHub state.

## Scope Decision

The completed F-018 GoReleaser plan is archived. F-017-S001 resumes through small owner-operated standard-tool slices rather than a repository-embedded audit runtime:

1. **T-070 — advertised Git surface:** freeze and hash the exact `refs/heads/main` plus `refs/tags/*` set returned by `git ls-remote --refs origin`, mirror those refs into an owner-only boundary, run exact pinned Gitleaks v8.30.1 and TruffleHog v3.95.9 offline, reconcile coverage/errors, and commit only a redacted report. Symbolic `HEAD` and annotated-tag peeled targets are derived coverage records, not independent scan roots; any other advertised namespace is an unresolved scope change.
2. **GitHub-hosted surfaces:** after T-070, create one bounded ticket for retained Releases/assets, Actions evidence, Wiki/Pages, repository metadata, and any proven applicable private surfaces. Inaccessible is unresolved, never clean.
3. **Manual disposition:** after technical collection, create one bounded ticket for privacy, IP, provenance, notices, models/media, name searches, accepted gaps, owner attestation, and the final history choice.

Only one ticket is current. Later tickets are created through `ticket_create` after the prior slice is durably closed.

## T-070 Contract

### In scope

- Start from a clean private `main` equal to freshly fetched `origin/main`, with no active ref, Release, settings, or Actions mutation.
- Use an owner-only encrypted-home directory outside the repository and ordinary temporary directories, with `umask 077`, directory mode `0700`, and file mode `0600`.
- Use only stock `git` plus exact provenance-verified Gitleaks v8.30.1 and TruffleHog v3.95.9. No custom wrapper, MARS audit tool, VM, container, sparsebundle, service, or scanner framework.
- Freeze and hash the canonical `refs/heads/main` plus `refs/tags/*` OID manifest returned by `git ls-remote --refs origin`, verify object integrity and exact ref equality, and scan every object reachable from those roots offline with scanner update, verification, provider, and network-backed behavior disabled. Record symbolic `HEAD` and annotated-tag peeled targets as derived coverage only; do not admit another namespace silently.
- Route scanner stdout, stderr, reports, candidate values, and locators directly to owner-only files. Repository evidence may contain only snapshot/tool digests, aggregate coverage/error counts, broad finding classes, statuses, and random opaque finding IDs.
- Preserve but exclude local stashes, reflogs, administrative refs, unreachable objects, and the unrelated Codex worktree because they are not advertised publication refs.

### Stop and no-go rules

- A plausible credential stops T-070 immediately without reproducing, verifying online, hashing, deleting, or continuing other scans. Rotation is a separately approved owner action before resumption.
- Missing refs, scanner errors, skipped objects, incomplete coverage, unverified tools, or raw-output leakage remain unresolved and cannot be called clean.
- No visibility change, history rewrite, force-push, ref/tag/Release/asset deletion or creation, Pages/settings mutation, signing, publication, credential-value output, or announcement is authorized.
- Primary Status remains `primary_blocked` even if T-070 passes; F-017-S001 also remains pending until GitHub-hosted surfaces, manual review, and owner disposition pass.

## Evidence And Acceptance

- Start and end `HEAD`, `origin/main`, the canonical publication-ref manifest, and private visibility are identical; Actions is quiescent at both observations and T-070 initiates no GitHub mutation. Release, settings, and hosted-content reconciliation belongs to the next ticket.
- Every in-scope publication ref and object reachable from it is reconciled as scanned; scanner errors, skips, and unresolved findings are zero for this slice.
- The committed report contains no raw paths, filenames, emails, URLs, IDs, bodies, candidate fragments, secret-derived hashes, or scanner output.
- Security and QA approve coverage and redaction; Release Manager confirms no release authority was exercised; Orchestrator records the next exact slice.
- `git diff --check`, documentation consistency, and DocSync pass. Runtime/AD-284 replay is not applicable to this evidence-only ticket.

## Source Versioning Exception

T-070 changes only planning and redacted audit evidence while public release state remains deliberately frozen at `VERSION=0.68.49` with untagged source fallback `0.69.0-dev`. The owner-authorized F-018 transition exception therefore extends narrowly through T-070 documentation/evidence commits: commit and push each bounded checkpoint, but do not generate release notes, change VERSION/CHANGELOG/buildinfo, tag, sign, upload, or publish. Product remediation, if a finding requires it, must use a separately created ticket and the then-current versioning policy.

## Handoff State

- T-056 supplied the pre-rewrite inventory contract but did not scan content.
- T-063/T-064 reconstructed the private publication candidate and retired the audit-era refs/releases.
- T-065 through T-069 proved the private GoReleaser producer/consumer/rehearsal path without publication.
- The prior pre-destruction scan reduced transaction risk but predates later commits and its deleted raw evidence cannot pass the current publication surface.
- Repository visibility is private; publication authority is denied; no supported public release exists.

## T-070 Scan Checkpoint — 2026-07-22

- Source `375a3a30140c9248f10c19eb4ff8a66ba83b7522` froze 302 canonical publication refs, 844 commits, and 11,954 objects. The post-scan manifest matched, visibility stayed private, Actions was quiescent, and no unexpected namespace appeared.
- Exact Gitleaks v8.30.1 and signed TruffleHog v3.95.9 completed Git-history plus raw-object lanes with zero accepted-scan errors or skip events.
- After owner-only review, Elliott classified Gitleaks group `a2a292e31d652f22` as synthetic test stubs. No agent inspected or reproduced raw evidence.
- URI group `e32927624f4a2cac` remains unclassified. T-070 and F-017-S001 are stopped on that classification, any required owner rotation/revocation, and a complete four-lane rescan; the next GitHub-hosted slice cannot start.
