---
id: MH-063
title: Require immutable release completion after semantic commits
priority: high
complexity: small
work_type: operating-model
bdd_scenarios: ["F-009-S019"]
end_to_end_evidence: "v0.69.4 immutable GitHub Release"
evidence_links: ["https://github.com/greaveselliott/MARS/releases/tag/v0.69.4", "https://github.com/greaveselliott/MARS/actions/runs/32942457278"]
verified_by: "foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-08-26"
blocker: "none"
blocked_by: []
trace_id: "release-v0.69.4"
next_action: "Complete: v0.69.4 published with ten verified assets after the source release-completion rule landed."
kind: intervention-debt
dedupe_key: "foundation:semantic-commit-immutable-release"
source: 2026-08-26 owner release-completion directive
created: 2026-08-26
depends_on: []
---

# MH-063: Require immutable release completion after semantic commits

## Context
The owner requires each non-release semantic commit to complete as an immutable published release, not merely as a pushed release-note commit.

## Requirements
- Preserve the release-note commit exemption.
- Require source publication and verification after every semantic commit.
- Mirror the release-completion principle to generated targets through their repository-owned producer.

## Acceptance Criteria
- [x] F-009 and foundation release doctrine require release completion.
- [x] Source guidance specifies tag, publication, and verification.
- [x] The current repair is published as the next immutable release.

## Completion Evidence

`v0.69.4` is immutable, points at release-note commit
`35dbb75707dfd8ae406ce79f9d16677937c496e5`, and has exactly ten uploaded
release assets. GitHub Actions run `32942457278` passed production, keyless
attestation, independent verification, and publication.
