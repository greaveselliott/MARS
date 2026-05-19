---
id: T-006
title: Create foundation release publication skill
priority: high
complexity: small
work_type: docs
bdd_scenarios: ["F-012-S006", "F-009-S013"]
end_to_end_evidence: not_applicable
evidence_links: [".harness/skills/release-publication/SKILL.md", "docs/design-docs/skill-evolution.md", "docs/design-docs/release-versioning.md", "docs/design-docs/index.md"]
verified_by: "go test ./internal/docsconsistency ./internal/docsync"
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: ["T-005"]
trace_id: "TBD"
next_action: "Done."
dedupe_key: "public-example"
metadata:
  category: "skill_evolution"
  confidence: "high"
  repo_id: "mars-harness"
  role: "release-manager"
  severity: "medium"
  target: "foundation-skill"
source: T-005 recursive improvement skill evaluation 2026-05-19
created: 2026-05-19
depends_on: [T-005]
---

# T-006: Create foundation release publication skill

## Context
The recursive improvement loop should remain operating doctrine, but the release-publication ritual has become a repeated, judgment-heavy Release Manager procedure: generate release notes, push release commits and tags, verify the GitHub Release object, create a notes-only fallback release when workflows are blocked, run asset verification, and record blockers without claiming completion.

## Requirements
- Add a compact foundation skill under .harness/skills/ for Release Manager or source release work.
- Cover GitHub release object publication, notes-only fallback from CHANGELOG.md, asset verification, missing-asset blocker recording, and token safety.
- Keep the skill procedural; do not grant new authority beyond existing tools and role allowlists.
- Update skill-evolution or release-versioning docs if the skill changes operating doctrine.
- Decide in the same implementation whether generated targets need a generic mirror or should keep existing release docs only.

## Affected Files
- .harness/skills/
- docs/design-docs/skill-evolution.md
- docs/design-docs/release-versioning.md
- internal/scanner/init.go, only if a target mirror is chosen

## Acceptance Criteria

### Functional
- [x] The foundation skill names when Release Manager should use it.
- [x] The skill gives a short ordered workflow from release-note commit through GitHub release verification and asset blocker recording.
- [x] The skill distinguishes release object publication from binary asset verification.
- [x] The skill states token safety rules and stop conditions.

### Edge cases and negative paths
- [x] The skill does not replace deterministic release tools or git commands.
- [x] The skill does not imply notes-only releases are complete.
- [x] Target mirroring is either implemented generically or explicitly deferred with rationale.

### Non-goals
- Adding a new release-publish CLI command or built-in tool.

### Observability, docs, and regressions
- [x] docsconsistency and relevant release or scanner checks pass.

## Completion Notes

- Added `.harness/skills/release-publication/SKILL.md` as a source-only
  foundation Release Manager skill.
- Updated AD-140 and AD-141 to record the doctrine boundary: recursive
  improvement remains operating model, the repeated release-publication ritual
  becomes a compact skill, and generated targets keep generic release guidance
  until target publication modes have a stable contract.
