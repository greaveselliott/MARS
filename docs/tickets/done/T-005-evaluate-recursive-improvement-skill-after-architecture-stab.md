---
id: T-005
title: Evaluate recursive improvement skill after architecture stabilizes
priority: medium
complexity: small
work_type: research
bdd_scenarios: ["F-012-S006"]
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/skill-evolution.md#ad-140-recursive-improvement-loop-stays-doctrine-release-publication-becomes-a-foundation-skill", "docs/tickets/backlog/T-006-create-foundation-release-publication-skill.md", "go test ./internal/docsconsistency ./internal/docsync"]
verified_by: "command"
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: ["T-002", "T-004"]
trace_id: "TBD"
next_action: "Done; implement T-006 when ready."
dedupe_key: "public-example"
metadata:
  category: "skill_tool_decision"
  confidence: "medium"
  repo_id: "mars-harness"
  role: "planner"
  severity: "medium"
  target: "skill-evolution"
source: foundation/deployed architecture planning 2026-05-19
created: 2026-05-19
depends_on: [T-002, T-004]
---

# T-005: Evaluate recursive improvement skill after architecture stabilizes

## Context
After the foundation/deployed architecture is documented and drift-checked, the next question is whether the recursive improvement loop should remain design doctrine or become compact reusable skill guidance. The user selected a role-skill direction for release publication rather than immediately adding a deterministic release-publish CLI tool, but the architecture should stabilize first.

## Requirements
- Compare skill versus tool versus guardrail using docs/design-docs/skill-evolution.md and docs/design-docs/tools-glossary.md.
- Decide whether the recursive improvement loop should become a universal skill, foundation skill, deployed skill pattern, or remain design doctrine.
- Decide whether Release Manager needs a compact skill before any release-publish tool exists.
- Record the decision in the owning design doc or create a follow-up implementation ticket.

## Affected Files
- docs/design-docs/skill-evolution.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/foundation-deployed-harness-architecture.md
- .harness/skills/ or internal/scanner/init.go only if the decision is to create or mirror a skill in a later implementation slice

## BDD Evidence
- Scenario IDs: F-012-S006
- Evidence links: decision record or follow-up ticket; go test ./internal/docsconsistency if docs change
- Verified by: command or human review

## Acceptance Criteria

### Functional
- [x] The research records a clear skill/tool/guardrail/doctrine decision.
- [x] The decision distinguishes universal, foundation, and deployed skill scope.
- [x] Release Manager skill needs are evaluated separately from any future deterministic release-publish tool.

### Edge cases and negative paths
- [x] The research does not create a skill before the architecture and drift review are complete.
- [x] The research does not choose a tool for judgment-heavy procedure or a skill for deterministic enforcement.

### Non-goals
- Implementing the skill or tool in this ticket unless a later ticket explicitly claims that work.

### Observability, docs, and regressions
- [x] The decision is captured in a durable repo artifact, not only chat.
