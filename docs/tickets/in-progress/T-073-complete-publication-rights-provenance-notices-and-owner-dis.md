---
id: T-073
title: Complete publication rights, provenance, notices, and owner disposition
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md", "docs/validation/reports/2026-08-08-github-hosted-publication-surface-audit.md", "docs/tickets/done/T-072-audit-every-github-hosted-publication-surface.md"]
verified_by: "pending repository owner, QA, Security, Release Manager, and Orchestrator"
owner: "foundation-maintainer"
last_attempt: "2026-08-08: created through ticket_create after COO, CTO-weekly, and Security froze four bounded checkpoints"
blocker: "none"
blocked_by: []
trace_id: "launch-rights-provenance:2026-08-08"
next_action: "Complete checkpoint A: authority, predecessor/AI, media, and current official-register name-search evidence; stop for rename or replacement if any retained item is unresolved."
dedupe_key: "open-source:rights-provenance-owner-disposition"
metadata:
  classification: "mixed-unclear-until-disposition"
  mutation_authority: "repository-files-only"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  supports: "F-017-S001"
source: MARS Launch-Complete Open-Source Delivery Plan — T-073
created: 2026-08-08
depends_on: [T-072]
---

# T-073: Complete publication rights, provenance, notices, and owner disposition

## Context

T-070 and T-072 cleared the advertised Git and GitHub-hosted publication surfaces for secrets and inventory completeness. F-017-S001 remains blocked until every retained source, document, prompt, asset, model, binary, dependency, and project name is authorized, provenance-complete, notice-complete, and covered by a final owner disposition.

## Scope And Authority

Keep the repository private and retain VERSION 0.68.49 with source fallback 0.69.0-dev. This ticket may update repository source, metadata, notices, CI checks, and public-facing claims needed to resolve rights and provenance. It may not create or move tags, sign or publish artifacts, delete hosted assets, change GitHub settings or visibility, or announce the project. Raw search exports, personal identifiers, candidate material, and provider records stay owner-only; repository evidence contains only authoritative citations, immutable upstream identifiers and hashes, broad classifications, dates/jurisdictions, opaque finding IDs, and dispositions.

## Checkpoints

1. Authority, name, predecessor, AI, and media: map first-party source/docs/prompts/assets, employment or client conflicts, the 11 predecessor Mars prompt ports, AI-assisted commits and applicable terms, and every media/branding file. Each port maps its predecessor full commit, source path/blob, retained path/blob, and normalized similarity evidence. Perform current UKIPO, USPTO, EUIPO/TMview, and WIPO searches. A material name conflict stops for rename; unverifiable media is replaced or removed. Commit and push this independently reviewed checkpoint.
2. Model, quantizer, llama.cpp, and vendored provenance: record full immutable model and quantizer revisions, artifact SHA-256, original-model license and terms, quantizer source/license, llama.cpp tag resolved to its full source commit plus checksums/license/notices/platform mapping, and htmx/Chart.js provenance for every retained or downloaded artifact. Abbreviated revisions and family-level citations do not pass. Replace, remove, or fail closed on any incomplete chain. Commit and push this independently reviewed checkpoint.
3. Dependency notices and product claims: pin go-licenses v2.0.1, generate deterministic Go dependency notices, and add CI that rejects unknown, stale, GPL, AGPL, or SSPL classifications. Complete THIRD_PARTY_NOTICES and remove provisional wording. Correct every claim to local-first and explicitly document configured cloud/integration transmission, opt-in telemetry, and host execution with the current OS user's authority. Commit and push this independently reviewed checkpoint.
4. Owner disposition: create one repository-safe rights/provenance report and owner attestation covering first-party authority, employment/client conflicts, predecessor ports, AI terms, third-party obligations, models, llama.cpp, media, name-search disposition, accepted evidence gaps, and history strategy. Accepted gaps are limited to already recorded non-content/provider limitations and can never substitute for ownership, license, provenance, model terms, predecessor, media, or trademark evidence. Every finding must be resolved by retain_authorized, retain_with_notice, replace, exclude_from_release, or remove_current_and_history; permission_required, clean_snapshot_required, or no_go cannot remain at completion. exclude_from_release is terminal only for an external artifact proven absent from public source, retained history, and Release assets whose exact terms still permit linking or downloading. replace and remove_current_and_history close only after executed, mechanically verified remediation; required history removal stops T-073 for separately authorized destructive rewrite. Commit preserve_audited_history only if every retained publishable surface is complete. Obtain repository-owner, QA, Security, Release Manager, and Orchestrator sign-offs and close F-017-S001.

## Affected Interfaces And DocSync

Expected implementation surfaces are the model and setup registries, llama.cpp download metadata, THIRD_PARTY_NOTICES/NOTICE/LICENSE metadata, deterministic dependency-license configuration and CI, the unprovenanced media asset if unresolved, README/AGENTS/product and documentation claim surfaces, and one rights/provenance validation report. Update MarsDocSync routes and generated target guidance only where behavior or mirrored claims actually change.

## Validation

Run focused provenance/registry/license tests, deterministic two-run notice comparison, forbidden-or-unknown license negative fixtures, immutable revision/hash and platform coverage checks, media inventory/provenance checks, claim-string consistency and DocSync checks, link checks, uncached tests for changed packages, four CGO-disabled cross-builds when source metadata changes, and installed-binary smoke only if runtime behavior changes. QA and Security review each frozen implementation diff concurrently; Release Manager verifies the launch freeze and no remote publication authority.

## Acceptance

- The owner explicitly attests authority over first-party source, documentation, prompts, assets, predecessor material, and AI-assisted contributions, with no unresolved employment/client/assignment conflict.
- Every retained media, model, quantizer, llama.cpp binary, vendored browser asset, and Go dependency has an exact authorized provenance/license/terms/notices chain.
- Current UKIPO, USPTO, EUIPO/TMview, and WIPO searches have a repository-safe date/jurisdiction/risk disposition and no unresolved material conflict.
- Deterministic dependency notices are complete; CI rejects unknown, stale, GPL, AGPL, and SSPL classifications; THIRD_PARTY_NOTICES is final and non-provisional.
- Product claims accurately describe local-first operation, configured transmissions, opt-in telemetry, and host execution authority.
- No finding is deferred and no unverifiable material remains retained. Every replacement or removal is executed and mechanically verified; owner acceptance covers no missing rights evidence. The owner attests preserve_audited_history; QA, Security, Release Manager, repository owner, and Orchestrator sign off.
- F-017-S001 passes, while Primary Status remains primary_blocked on T-074 through T-081 and the installed-App no-go remains routed to T-079/T-080.

## No-Go

Any unresolved ownership, employment/client, predecessor, AI-term, trademark, media, model, binary, dependency, notice, or claim finding; any provisional notice; any unverifiable retained artifact; or any tag, release, GitHub-setting, visibility, or announcement mutation.
