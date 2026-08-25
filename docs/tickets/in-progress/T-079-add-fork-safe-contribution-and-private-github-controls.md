---
id: T-079
title: Add fork-safe contribution and private GitHub controls
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-017-S004"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md#f-017-s004-fork-safe-contribution-and-governance"]
verified_by: "pending"
owner: "foundation-maintainer"
last_attempt: "2026-08-25"
blocker: "none"
blocked_by: []
trace_id: "launch-contribution-controls:2026-08-25"
next_action: "Commit and validate conventional contribution files and fork-safe CI, then configure exact private repository settings and ruleset without changing visibility or release state."
dedupe_key: "open-source:fork-safe-contribution-controls"
metadata:
  classification: "foundation-owned-and-hosted-settings"
  mutation_authority: "source plus exact private Issues/Discussions/Dependabot/ruleset controls only"
  public_only_controls: "deferred-to-T-080"
  publication_authority: "denied"
source: MARS Launch-Complete Open-Source Delivery Plan — owner-approved Item 9
created: 2026-08-25
depends_on: []
---

# T-079: Add fork-safe contribution and private GitHub controls

## Context

F-017-S004 requires a conventional contributor experience and private-safe GitHub controls before public cutover. The owner directed Item 9 to proceed independently while T-078's destructive hosted cleanup remains parked on separate approval.

## Requirements

- Add community, conduct, governance, support, security-reporting, issue, pull-request, DCO, and CODEOWNERS surfaces.
- Keep pull-request automation fork-safe with read-only permissions, no secrets, no OIDC, no publication authority, and no privileged runner.
- Add bounded Dependabot configuration for both Go modules and GitHub Actions.
- Revalidate and configure only Issues, Discussions, Dependabot alerts/updates, and one exact main-branch ruleset with maintainer bypass and contributor PR/review/status protections.
- Keep visibility, Apps, Pages, public-only CodeQL/secret-scanning/push-protection/private-reporting, tags, Releases, assets, immutable Releases, and T-078 cleanup unchanged.

## Validation

- Static workflow/permission/event tests and executable DCO positive/negative tests.
- Normal/race/vet and documentation consistency/DocSync gates.
- Hosted CI green on the pushed source.
- Exact read-only pre/post GitHub receipts for repository settings, ruleset, Actions permissions, secrets/variables names/counts, and security controls.
- Public-only hostile-fork and security-control smoke stays assigned to T-080 after separately approved visibility.

## Acceptance

The private repository has conventional community and fork-safe contribution files, source CI grants read-only authority to fork PRs, Dependabot is configured, Issues and Discussions are enabled, and an exact active ruleset protects main while preserving the documented maintainer trunk workflow through an explicit admin bypass. No public-only, release, cleanup, App, or visibility mutation occurs.
