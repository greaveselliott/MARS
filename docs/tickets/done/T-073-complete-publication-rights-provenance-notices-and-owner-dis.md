---
id: T-073
title: Complete publication rights, provenance, notices, and owner disposition
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S001"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md", "docs/validation/reports/2026-08-09-rights-media-and-name-review.md", "docs/validation/reports/2026-08-24-owner-launch-dispositions.md", "docs/tickets/done/T-072-audit-every-github-hosted-publication-surface.md"]
verified_by: "repository owner disposition plus retained QA/Security/Release Manager/Orchestrator machine evidence"
owner: "foundation-maintainer"
last_attempt: "2026-08-24: owner accepted the recorded name risk and attested publication authority; machine-verifiable checkpoints were already complete"
blocker: ""
blocked_by: []
trace_id: "launch-rights-provenance:2026-08-08"
next_action: "Complete T-078 through T-081; preserve the owner name-risk disposition without presenting it as trademark clearance."
dedupe_key: "open-source:rights-provenance-owner-disposition"
metadata:
  classification: "owner-disposed-and-machine-verified"
  mutation_authority: "repository-files-only"
  primary_status: "primary_blocked_on_later_launch_gates"
  publication_authority: "granted_subject_to_remaining_launch_gates"
  supports: "F-017-S001"
source: MARS Launch-Complete Open-Source Delivery Plan — T-073
created: 2026-08-08
depends_on: [T-072]
---

# T-073: Complete Publication Rights, Provenance, Notices, And Owner Disposition

## Outcome

T-073 is complete. The machine-verifiable provenance, browser-asset,
llama.cpp, model, dependency-notice, current-tree media, prompt-attribution,
and product-claim checkpoints passed in the retained 2026-08-09 evidence. On
2026-08-24 the repository owner supplied the two remaining owner-only
dispositions:

- the owner accepts the recorded unresolved `MARS` name-conflict risk, will not
  register the mark, and does not require qualified trademark counsel
  clearance for launch; and
- the owner attests authority to publish the current source, documentation,
  release artifacts, retained history, first-party material, and
  Cursor/automation-assisted material, subject to the remaining launch gates.

The exact statement and scope are recorded in
`docs/validation/reports/2026-08-24-owner-launch-dispositions.md`.

## Important Qualification

This closure is an owner risk/authority disposition, not a legal finding that
`MARS` is clear, registrable, or free of third-party claims. The earlier search
evidence remains accurate and is not erased. Launch and public support remain
blocked on T-078 through T-081.

## Retained Evidence

- `c7168c5` and `f3df0a5` bind embedded browser assets and llama.cpp
  provenance.
- `2ffde82` corrects public network/telemetry/host-authority claims.
- `79d524b` and `dc0dbe0` pin and generate deterministic dependency notices;
  GitHub run `31288019067` passed the owning lanes.
- `a8d448f` records verified prompt introduction/comparison facts without
  claiming unsupported textual lineage; `12faa47` removed the unprovenanced
  PNG from the current tree.
- `cf95b39` and `b8d9349` bind the six default GGUF artifacts and fail closed
  on incomplete download metadata; GitHub run `31289522986` passed.
- `docs/validation/reports/2026-08-09-rights-media-and-name-review.md` preserves
  the machine inventory and adverse name-search facts.

No tag, Release, signature, upload, hosted deletion, settings mutation,
visibility change, or announcement was authorized or performed by this ticket.
