---
id: T-070
title: Scan the exact advertised Git publication surface with pinned standard tools
priority: high
complexity: medium
work_type: research
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-07-22-advertised-git-publication-scan.md"]
verified_by: "Security, QA, Release Manager, and Orchestrator GO on 2026-08-08 advertised-Git closure"
owner: "security"
last_attempt: "2026-08-08: direct Git-plumbing reconciliation proved all 36 records in opaque group f3dc0e336620abc6 are child blob IDs from materialized reachable tree objects; all checks passed 36/36 with zero mismatches."
blocker: "none"
blocked_by: []
trace_id: "not_applicable"
next_action: "Completed; Orchestrator creates the next bounded F-017-S001 GitHub-hosted-surface ticket through ticket_create."
dedupe_key: "open-source:advertised-git-standard-scan"
metadata:
  audit_mode: "owner-standard-tools"
  classification: "evidence-only,mixed-unclear"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  scenario_slice: "advertised-git"
source: docs/exec-plans/active/current-operating-plan.md — F-017-S001 advertised Git slice
created: 2026-07-22
depends_on: [T-056, T-068]
---

# T-070: Scan the exact advertised Git publication surface with pinned standard tools

## Context

The reconstructed private publication candidate has not been scanned at its current head. T-056 inventoried the pre-rewrite surface without content scanning; earlier raw scan evidence predates later commits and cannot pass the current candidate. This ticket covers only advertised main and retained tags.

## Requirements

- Start and end with clean local main equal to fetched origin/main, private visibility, quiescent Actions, and an unchanged canonical publication-ref/OID manifest containing only `refs/heads/main` plus `refs/tags/*` from `git ls-remote --refs origin`. Treat symbolic `HEAD` and annotated-tag peeled targets as derived coverage records; any other advertised namespace is an unresolved scope change.
- Use one owner-only encrypted-home directory outside the repository and ordinary temporary directories with umask 077, directory mode 0700, and file mode 0600.
- Use stock git plus provenance-verified Gitleaks v8.30.1 and TruffleHog v3.95.9. No custom audit runtime, wrapper, VM, container, sparsebundle, service, or scanner framework.
- Mirror and scan every object reachable from the canonical `refs/heads/main` plus `refs/tags/*` roots. Preserve but exclude local stashes, reflogs, administrative refs, unreachable objects, and the unrelated Codex worktree.
- Disable scanner update, online verification, provider, and network-backed detector behavior; pass no GitHub/provider credentials or host configuration into scanner processes.
- Route raw output, candidates, and locators directly to owner-only files. Commit only snapshot/tool digests, aggregate coverage and error counts, broad finding classes/status, and random opaque finding IDs.
- Stop immediately on a plausible credential without reproducing, verifying online, hashing, deleting, or continuing. Rotation requires a separately approved owner action.

## Affected Files

- docs/validation/reports/2026-07-22-advertised-git-publication-scan.md
- docs/exec-plans/active/current-operating-plan.md
- docs/features/F-017-open-source-publication.md
- docs/goals/active.md
- this ticket

No product code, generated target content, refs, Releases, settings, visibility, credentials, signing, or publication changes are in scope.

## Acceptance Criteria

- Every in-scope publication ref and object reachable from it is reconciled as scanned by both exact pinned tools; scanner errors, skips, and unresolved findings are zero for this slice.
- Start/end HEAD, origin/main, canonical publication-ref manifest, and private visibility are identical; Actions is quiescent at both observations and T-070 initiates no GitHub mutation. Release, settings, and hosted-content reconciliation belongs to the next ticket.
- Repository-safe evidence contains no raw paths, filenames, emails, URLs, IDs, bodies, candidate fragments, secret-derived hashes, or scanner output.
- A plausible credential blocks completion pending separately approved rotation; missing coverage is unresolved, never clean.
- Security and QA approve coverage/redaction, Release Manager confirms no release authority was exercised, and Orchestrator records the next exact F-017-S001 slice.
- F-017-S001 and Primary Status remain blocked after this ticket because GitHub-hosted content, manual review, and owner disposition are separate later slices.
- git diff --check, documentation consistency, and DocSync pass; runtime/AD-284 replay is not applicable.

## Advertised Git Scan Checkpoint — 2026-07-22

- Frozen source `375a3a30140c9248f10c19eb4ff8a66ba83b7522` contains 302 canonical publication refs, 844 reachable commits, and 11,954 reachable objects. Pre/post ref manifests match and visibility remains private.
- Exact Gitleaks v8.30.1 and TruffleHog v3.95.9 completed Git-history and raw-object lanes with zero accepted-scan errors or skip events. The raw-object corpus reconciles every blob, tree, commit, and annotated-tag object.
- Gitleaks reported seven Git-history and 19 blob occurrences in one broad detector class. TruffleHog reported five Git-history and seven blob occurrences in one broad detector class, representing the same three distinct values across its two lanes. All Git-mode locations were mechanically test-like and no TruffleHog result was verified online.
- On 2026-08-08, after reviewing the owner-only Gitleaks context, Elliott classified `a2a292e31d652f22` as synthetic test stubs. No rotation is required for that group.
- `e32927624f4a2cac` was unresolved at this checkpoint. The 2026-08-08 checkpoint below supersedes that classification state; no GitHub or publication mutation occurred.

## Advertised Git Rescan Checkpoint — 2026-08-08

- Frozen source `d04f642e7b07e14e211c99f40e14bdd4bccef60e` contains 302 canonical publication refs, 847 reachable commits, and 12,002 reachable objects. Pre/post ref manifests match, visibility remains private, Actions remains quiescent, and no unexpected namespace appeared.
- Exact Gitleaks v8.30.1 and TruffleHog v3.95.9 completed all four Git-history/raw-object lanes with zero accepted-scan errors or skips. Gitleaks' seven Git-history and 19 raw-object locations reconcile to owner-confirmed synthetic group `a2a292e31d652f22`.
- TruffleHog's five Git-history URI occurrences and seven raw-object URI occurrences represent one distinct value and reconcile to local-MARS-confirmed synthetic group `e32927624f4a2cac`.
- The remaining 36 raw-object occurrences reduce mechanically to exactly four distinct values under opaque group `f3dc0e336620abc6`. Conservative isolated local-MARS review returned unknown, so the group remained fail-closed pending direct inspection.

## Direct Git-Object Disposition — 2026-08-08

- All 36 scanner source files are byte-identical materializations of reachable Git tree objects, and every alleged value has the exact 40-character lowercase SHA-1 form.
- For all 36 records, the value is the exact child-object field in the corresponding tree, resolves to an existing reachable blob, and equals the scanner raw field. Format, source-type, corpus, tree-entry, child-object, and raw-field mismatch counts are all zero.
- The four distinct values are content-addressed Git blob IDs, not CircleCI credentials. Group `f3dc0e336620abc6` is `resolved_scanner_false_positive_git_tree_object_ids`; no rotation is required.
- All finding groups are resolved, all four scanner lanes completed without errors/skips, no raw evidence escaped, and no GitHub or publication mutation occurred. T-070 passes; F-017-S001 and Primary Status remain blocked on later hosted-surface/manual-disposition gates.
