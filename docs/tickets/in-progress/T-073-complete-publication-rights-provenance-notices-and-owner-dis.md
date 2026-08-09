---
id: T-073
title: Complete publication rights, provenance, notices, and owner disposition
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md", "docs/validation/reports/2026-08-08-github-hosted-publication-surface-audit.md", "docs/validation/reports/2026-08-09-rights-media-and-name-review.md", "docs/tickets/done/T-072-audit-every-github-hosted-publication-surface.md"]
verified_by: "pending repository owner, QA, Security, Release Manager, and Orchestrator"
owner: "foundation-maintainer"
last_attempt: "2026-08-09: browser/llama provenance, product claims, deterministic notices, truthful prompt lineage, and current-tree PNG removal are pushed and verified"
blocker: "Qualified trademark disposition, owner authority over first-party/AI/automation and retained-history PNG material, and exact original-model/quantizer chains remain required."
blocked_by: []
trace_id: "launch-rights-provenance:2026-08-08"
next_action: "Resolve the six default model artifacts' exact original-model and quantizer chains, then complete the bounded owner and trademark dispositions; do not close T-073 or publish early."
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

1. Authority, name, predecessor, AI, and media: map first-party source/docs/prompts/assets, employment or client conflicts, the 11 predecessor Mars prompt ports, AI-assisted commits and applicable terms, and every media/branding file. Each port maps its predecessor full commit, source path/blob, retained path/blob, and normalized similarity evidence. Perform current UKIPO, USPTO, EUIPO/TMview, and WIPO searches. A material name conflict is a launch no-go. The owner's 2026-08-09 decision to retain `MARS` requires qualified trademark counsel's written disposition; it does not waive the conflict. Unverifiable media is replaced or removed. Commit and push this independently reviewed checkpoint.
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

## Checkpoint A Evidence — 2026-08-09

The publication-root inventory is complete and recorded in
`docs/validation/reports/2026-08-09-rights-media-and-name-review.md`. It proves
the exact retained prompt, tool/automation-attribution, and media scope without treating Git
metadata as ownership evidence. The eleven role headers are symbolic rather
than immutable and the locally available predecessor snapshot does not support
a textual-port claim. The sole PNG has no embedded or commit-level provenance.

The official-register knockout search is materially adverse: live U.S.
registration 8092258 covers AI agent-based analysis and process automation in
Class 42. On 2026-08-09, the repository owner directed that `MARS` be retained.
Checkpoint A remains open pending qualified trademark counsel's written
disposition plus the bounded owner authority, prompt/AI/automation, and media
decisions. Checkpoints B and C may gather and remediate machine-verifiable
provenance and notice evidence while those owner-only inputs are pending, but
checkpoint D, publication, and visibility remain prohibited.
The repository remains private; checkpoints B and C do not authorize
checkpoint D, any later ticket, or any remote/publication action.

## Checkpoint B/C And Current-Tree Remediation Evidence — 2026-08-09

- `c7168c5f06b59c59d0a05f0cbd6966b897b0b3ee` binds the embedded htmx and
  Chart.js versions, source commits, upstream artifacts, committed bytes, and
  license files.
- `f3df0a5520cd18e9d542086b501b1eec81abe5d7` binds llama.cpp `b8833` to its
  full source commit, license/notice inputs, and four exact platform archives;
  unsupported Linux acquisition remains disabled for T-077.
- `2ffde8279327b1623784b66587330dc52e479a09` replaces universal local/no-
  exfiltration claims with the reviewed local-first, configured-transmission,
  telemetry-transport, and current-user host-execution boundaries.
- `79d524b0b9ba8a27f3c907bed84ae77df572d6b8` pins the notice policy and exact
  reviewed inputs. `dc0dbe087e93f1ee74eb6fa7d49f3f098ea6cd75`
  generates the four-platform dependency notice union, consumes exactly three
  reviewed overrides, and adds the non-mutating stale gate. The checked-in
  notice SHA-256 is
  `d18a021e0d32c342d733f1c3e59ad72da8893bbbc41ae5dda6dbcca980631739`;
  GitHub run `31288019067` passes the notice job, Go 1.25.12, Go 1.26.5, and
  the below-minimum rejection.
- `a8d448f12d7cd75b376fe40801d89fcf2e07869e` replaces the eleven unsupported
  symbolic prompt-source headers with verified introduction/comparison facts,
  `textual_port_evidence: not_established`, and `owner_disposition: pending`.
  `12faa47e8298d73fe492a47d6923b98cc6015c6e` removes the sole PNG and both
  live references from current `main` in favor of browser-verified semantic
  HTML/CSS.

Checkpoint C's machine work is complete. Checkpoint B remains open only for
the six default model/original/quantizer chains. Checkpoint A and D remain open
for the owner authority/AI/automation/history-PNG attestation and qualified
trademark disposition; none of the commits above clears retained history,
changes visibility, or authorizes publication.
