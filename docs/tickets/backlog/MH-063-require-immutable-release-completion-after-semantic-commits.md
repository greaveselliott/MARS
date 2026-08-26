---
id: MH-063
title: Require immutable release completion after semantic commits
priority: high
complexity: small
work_type: operating-model
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "TBD"
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
- [ ] F-009 and foundation release doctrine require release completion.
- [ ] Source guidance specifies tag, publication, and verification.
- [ ] The current repair is published as the next immutable release.
