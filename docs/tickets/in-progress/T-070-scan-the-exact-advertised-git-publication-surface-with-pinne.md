---
id: T-070
title: Scan the exact advertised Git publication surface with pinned standard tools
priority: high
complexity: medium
work_type: research
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-07-22-advertised-git-publication-scan.md"]
verified_by: "TBD"
owner: "security"
last_attempt: "2026-07-22: owner classified at least one candidate as potentially real; both opaque groups remain unresolved because the affected group was not disclosed."
blocker: "plausible_secret_rotation_required: owner must locally identify all affected provider/accounts and rotate/revoke every potentially real credential through a separately approved action before T-070 or later audit slices resume."
blocked_by: []
trace_id: "not_applicable"
next_action: "Owner rotates/revokes every potentially real credential outside the agent boundary, then records only the affected opaque group ID or IDs plus rotation complete; rerun both pinned tools' Git-history and raw-object lanes afterward."
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
- Elliott classified at least one candidate as potentially real. Because the response did not identify an opaque group, `a2a292e31d652f22` and `e32927624f4a2cac` both remain unresolved without reproducing raw evidence or guessing which group is safe.
- T-070, F-017-S001, and Primary Status are blocked on separately approved owner rotation/revocation and a complete rescan. No GitHub or publication mutation occurred.
