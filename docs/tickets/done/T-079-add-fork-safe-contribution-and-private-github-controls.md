---
id: T-079
title: Add fork-safe contribution and private GitHub controls
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-017-S004"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md#f-017-s004-fork-safe-contribution-and-governance"]
verified_by: "local normal/race/vet/docs gates; hosted runs 32901437495 and 32903803262"
owner: "foundation-maintainer"
last_attempt: "2026-08-25"
blocker: "none"
blocked_by: []
trace_id: "launch-contribution-controls:2026-08-25"
next_action: "Complete; keep public-only controls and genuine hostile-fork smoke in T-080 after separately approved visibility."
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

## Evidence

- Source checkpoint `b807afa8de29756ccf0b6612135aad749e16ee07` is pushed and hosted run `32901437495` passes every source-compatibility lane.
- GitHub Discussions, Dependabot alerts/security updates, fork-PR approval/read-only/no-secret policy, GitHub-owned full-SHA Actions policy, the `dependencies` label, and exact ruleset `21491158` are enabled and re-read from GitHub.
- The complete pre/post receipt and no-mutation boundary are recorded in `docs/validation/reports/2026-08-25-t079-private-contribution-controls.md`.
- Visibility remains private; Pages, Apps, tags, Releases, assets, immutable Releases, public-only security controls, and T-078 cleanup were not changed.
- Final source `3502110854aafb561b9c90dec84259629c56cce8` passes hosted run `32903803262`. The narrow authenticated Dependabot exception accepts only GitHub's exact actor/author/committer/support-signoff identity; spoofed identity and trailer fixtures fail.
