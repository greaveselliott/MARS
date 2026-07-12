---
id: T-056
title: Inventory publication history and GitHub surfaces without exposing restricted evidence
priority: high
complexity: medium
work_type: research
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: not_applicable
evidence_links:
  - docs/validation/reports/2026-07-12-open-source-publication-surface-inventory.md
  - "PASS: QA completeness, schedule, snapshot, gap-binding, and forbidden-content review"
  - "PASS: Security restricted-environment, provenance, hostile-archive, evidence-boundary, and no-go review"
  - "PASS: full uncached Go suite, DocSync 332/0, docs consistency, diff check, and leak-pattern sweep"
  - "PASS: Dogfood classified AD-284 and installed-binary replay not applicable to the docs-only research diff"
verified_by: "Engineer; QA; Security; Dogfood; foundation-maintainer Orchestrator"
owner: "engineer"
last_attempt: "2026-07-12 bounded inventory and accepted-review corrections passed Engineer, QA, Security, Dogfood, and Orchestrator validation."
blocker: "none"
blocked_by: []
trace_id: "not_applicable"
next_action: "Complete; the repository owner must provision the restricted audit and appoint its Security operator while owner/legal review remains the F-017-S001 blocker."
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
- Every applicable GitHub surface is inventoried, absent, or recorded as ambiguous/inaccessible with the exact safe observation and bound to an owner-controlled restricted-audit input, owner, and next action; ambiguity does not block this bounded inventory or imply a clean surface.
- Tool readiness records exact pins and safe offline flags without using latest, provider verification, or SaaS uploads.
- The report contains no token values, authorization headers, authenticated URLs, personal email list, raw private content, candidate fragments, or external evidence-root path.
- The report states that pinned scans, manual privacy/IP/provenance review, legal authority, and name clearance remain pending.
- No GitHub mutation, scanner upload/verification, credential rotation, history rewrite, release deletion/upload, visibility change, or announcement occurs.
- Primary Status remains primary_blocked.
- git diff --check, DocSync, docs consistency, full tests, and foundation-maintainer dry-run pass.
- AD-284 live replay is not applicable because this ticket changes research evidence and planning only.

## Stop Conditions

Stop and record a partial or blocked report if a candidate credential appears, raw output would enter an agent transcript, an ambiguous/inaccessible surface cannot be safely bound to the owner-controlled restricted audit, a command would mutate Git or GitHub state, a tool is unpinned, an agent would make an ownership or trademark judgment, or local main ceases to match origin/main. An inaccessible API surface alone may complete this bounded inventory only when the report records it as unknown and assigns its restricted-audit input, owner, and next action.

## Engineer Evidence

- PASS: local `HEAD`, fetched `origin/main`, and advertised `main` matched at collection time.
- PASS: local Git integrity and ref/object/category counts were projected without paths, bodies, emails, or candidate values.
- PASS: read-only GitHub queries emitted only aggregate counts and non-sensitive status fields; inaccessible and ambiguous surfaces are recorded as unknown rather than inferred clean.
- PASS: no scanner, GitHub mutation, ref mutation, release operation, credential operation, or public action ran.
- PASS: the report defines the restricted evidence boundary and pins Gitleaks v8.30.1 and TruffleHog v3.95.9 without installing or executing them.
- PASS: every ambiguous/inaccessible outcome is treated as unknown and assigned to the repository owner, appointed Security audit operator, and owner/legal disposition path; T-056 itself has no blocker because its bounded inventory objective is fulfilled, while F-017-S001 remains blocked.
- PASS: QA and Security independently blocked the first draft, then passed the
  corrected schedule, snapshot attribution, gap binding, isolated scanner
  execution, trusted provenance, hostile-archive, and no-go contracts.
- PASS: the Orchestrator ran the full uncached Go suite, DocSync, docs
  consistency, forbidden-content pattern checks, and diff checks.
- PASS: Dogfood classified AD-284 and installed-binary clean-project replay as
  not applicable to this docs-only research/evidence diff.
- NOT APPLICABLE: AD-284 live replay because this ticket changes research evidence and planning only.
