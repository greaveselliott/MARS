---
id: T-080
title: Perform the approved public cutover and publish attested launch releases
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S003", "F-017-S004", "F-017-S005", "F-018-S004"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md", "docs/features/F-018-goreleaser-distribution.md", "docs/validation/reports/2026-08-24-t078-hosted-state-revalidation.md", "docs/validation/reports/2026-08-25-t079-private-contribution-controls.md", "docs/validation/reports/2026-08-26-t080-public-cutover-preflight.md"]
verified_by: "TBD"
owner: "foundation-maintainer"
last_attempt: "2026-08-26: owner approved the exact frozen transaction; activating the guarded tag-only workflow while the repository remains private and no launch tag exists"
blocker: "complete and verify the private workflow-activation and 0.69.0 version commits before the disposable public rehearsal and visibility cutover"
blocked_by: []
trace_id: "launch-public-cutover:2026-08-26"
next_action: "Validate, commit, and push the guarded workflow activation; then generate and validate the 0.69.0 version commit without creating its tag."
dedupe_key: "open-source:public-cutover-and-attested-launch-releases"
metadata:
  apps: "out-of-scope"
  classification: "foundation-owned-and-hosted-cutover"
  mutation_authority: "approved for the exact frozen T-080 transaction"
  name_risk: "owner-accepted"
  publication_authority: "approved for exact immutable v0.69.0 and v0.69.1 sequence after preceding gates"
  supports: "F-017-S003,F-017-S004,F-017-S005,F-018-S004"
  version_floor: "0.68.49"
source: MARS Launch-Complete Open-Source Delivery Plan — owner-approved Item 10
created: 2026-08-26
depends_on: [T-078, T-079]
---

# T-080: Perform the approved public cutover and publish attested launch releases

## Context

T-078 completed the standard dormant release workflow, exact hosted cleanup, and future-only immutable-Release setting. T-079 completed private contribution controls. The owner directed the shortest credible launch path and prohibited further bespoke release-security infrastructure unless an existing launch criterion demonstrably requires it.

## Requirements

- Perform a read-only exact preflight of repository visibility, main/refs, launch-tag and Release-name availability, hosted runs/artifacts/caches/deployments, immutable Releases, workflows, permissions, secrets/variables, rulesets, security controls, community surfaces, Discussions, and Pages.
- Verify the dormant release workflow remains full-SHA pinned, least privilege, tag/ref/SHA guarded, credential-free outside the GitHub token, and disabled before approval.
- Freeze the exact visibility, public-only-control, and two-release transaction together with irreversible consequences and rollback boundaries.
- Obtain separate explicit owner approval before changing visibility, public-only controls, tags, Releases, Pages, or publication state.
- After approval, make the repository public, enable and verify applicable public-only security controls, activate the conventional release workflow, publish and independently verify immutable attested v0.69.0, then publish and verify immutable attested v0.69.1 as latest with v0.69.0 retained only as rollback.
- Keep GitHub Apps out of scope, retain the accepted name risk, preserve the narrow vulnerability disposition, and do not revive custom Docker Engine/API, ptrace/Landlock, executable-format, or exhaustive SPDX-parser work.
- Leave announcement and the 48-hour canary to T-081.

## Acceptance

The exact pre-state and consequences are recorded before mutation; the owner separately approves the frozen transaction; the repository becomes public with applicable controls active; v0.69.0 and v0.69.1 are distinct immutable exact-ten attested Releases; v0.69.1 is latest; anonymous verification/update/rollback pass; no unrelated hosted surface changes; and a durable postcondition receipt hands off to T-081.
