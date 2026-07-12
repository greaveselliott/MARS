---
id: T-056
title: Inventory publication history and GitHub surfaces without exposing restricted evidence
priority: high
complexity: medium
work_type: research
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "engineer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "not_applicable"
next_action: "Collect a read-only inventory, record access gaps and offline scan prerequisites, and commit only a redacted publication-surface report."
dedupe_key: "open-source:publication-history-github-inventory"
metadata:
  classification: "evidence-only,mixed-unclear"
  primary_status: "primary_blocked"
  technical_lane: "authorized"
source: docs/exec-plans/active/current-operating-plan.md — F-017-S001 OSS-00 publication-surface inventory
created: 2026-07-12
depends_on: [T-055]
---

# T-056: Inventory publication history and GitHub surfaces without exposing restricted evidence

## Context

Before MARS history, releases, or repository settings can become public, the foundation maintainer needs a complete read-only inventory and redacted access-gap record. This research ticket supplies inputs for an owner-controlled offline audit and legal review; it does not establish publication authority or pass F-017-S001.

## Requirements

- Inventory local and advertised Git refs, object/commit/tag counts, notes, replace refs, stash and local-only unreachable-object categories, LFS, submodules, and redacted remote identity.
- Inventory applicable GitHub metadata through read-only GET or GraphQL operations: repository features and visibility, branches/rulesets, releases/assets, Issues, PRs, Discussions, wiki/Pages, Actions runs/artifacts/caches, packages/deployments/environments, security features, collaborators/access counts, deploy keys, webhooks, Apps, and secret or variable names/counts only.
- Classify every surface as inventoried, absent, inaccessible, or requiring later restricted scanning. Permission gaps remain unknown and cannot be inferred clean.
- Record pinned offline scanner prerequisites for Gitleaks v8.30.1 and TruffleHog v3.95.9. Do not install or execute broad scanners in this inventory ticket.
- Define the owner-controlled restricted-evidence contract: an operator-approved encrypted MARS_OSS_AUDIT_ROOT outside the repository and normal temporary directories; umask 077; directories 0700; files 0600; no network verification; no raw output through agents, chat, CI, traces, screenshots, or repository files.
- Never print, transmit, or commit candidate secrets, personal emails, private bodies, authenticated URLs, exact sensitive paths, or raw scanner JSON. Use aggregate counts and opaque random finding IDs only.
- If a plausible real credential appears, stop without reproducing it. Rotation is separately approved incident work and must precede removal or history rewriting.
- Produce a technical recommendation of preserve_audited_history, clean_public_snapshot, or undecided. This is not the owner or legal decision.
- Keep publication authority, name clearance, visibility, history rewriting, release deletion, and announcement blocked.

## Affected Files

- docs/validation/reports/2026-07-12-open-source-publication-surface-inventory.md
- docs/exec-plans/active/current-operating-plan.md
- docs/features/F-017-open-source-publication.md
- docs/goals/active.md only if goal evidence changes
- this ticket

No product code, generated target files, refs, releases, or GitHub settings are changed.

## Evidence Boundary

Raw command output, GitHub bodies, attachment names/content, author/email inventories, scanner candidates, logs, assets, and evidence manifests stay outside the repository under owner-only protection. The committed report may contain source/ref manifest hashes, tool versions, counts, surface coverage, access gaps, classifications, opaque finding IDs, redacted dispositions, blockers, and exact next actions.

## Acceptance Criteria

- Start and end main are clean and equal to freshly fetched origin/main except for the scoped report/plan/feature/ticket edits.
- Local Git integrity passes; reachable, advertised, local-only, and unreachable object surfaces are distinguished.
- Every applicable GitHub surface is inventoried, absent, inaccessible with an exact permission blocker, or routed to the later restricted scan.
- Tool readiness records exact pins and safe offline flags without using latest, provider verification, or SaaS uploads.
- The report contains no token values, authorization headers, authenticated URLs, personal email list, raw private content, candidate fragments, or external evidence-root path.
- The report states that pinned scans, manual privacy/IP/provenance review, legal authority, and name clearance remain pending.
- No GitHub mutation, scanner upload/verification, credential rotation, history rewrite, release deletion/upload, visibility change, or announcement occurs.
- Primary Status remains primary_blocked.
- git diff --check, DocSync, docs consistency, full tests, and foundation-maintainer dry-run pass.
- AD-284 live replay is not applicable because this ticket changes research evidence and planning only.

## Stop Conditions

Stop and record a partial or blocked report if a candidate credential appears, raw output would enter an agent transcript, a required API surface is inaccessible, a command would mutate Git or GitHub state, a tool is unpinned, ownership or trademark judgment is required, or local main ceases to match origin/main.
