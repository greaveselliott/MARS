---
id: T-002
title: Document foundation and deployed harness architecture
priority: high
complexity: medium
work_type: docs
bdd_scenarios: ["F-001-S015"]
end_to_end_evidence: not_applicable
evidence_links: ["docs/design-docs/foundation-deployed-harness-architecture.md", "go test ./internal/docsconsistency ./internal/docsync"]
verified_by: "command"
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done; use T-003 for generated target mirroring."
dedupe_key: "public-example"
metadata:
  category: "architecture_doctrine"
  confidence: "high"
  repo_id: "mars"
  role: "planner"
  severity: "high"
  target: "operating-model"
source: foundation/deployed architecture planning 2026-05-19
created: 2026-05-19
depends_on: []
---

# T-002: Document foundation and deployed harness architecture

## Context
MARS now has two related harness contexts: the foundation harness used to evolve mars itself and deployed harnesses generated into target projects. Recent recursive improvement and release-publication work showed that agents need a clear architecture for what belongs to the foundation harness, what belongs to deployed harnesses, what mirrors, and what stays source-only.

## Requirements
- Add docs/design-docs/foundation-deployed-harness-architecture.md.
- Use existing glossary terms: foundation harness, deployed harness, target project, operating model, mirrored harness definitions, mirrored tools, universal tool surface, universal skills, foundation skills, and deployed skills.
- Explain why this is happening: the binary executes orchestration while the foundation harness owns evolving doctrine, generated target defaults, tools, skills, release discipline, and self-improvement loops.
- Define the foundation harness, runtime substrate, deployed harness, target project, and mirrored operating-model core.
- Include boundary tables for foundation-only, mirrored, and deployed-only behavior.
- Include diagrams or tables for the foundation/deployed architecture and doctrine flow.
- Update docs/design-docs/index.md.

## Affected Files
- docs/design-docs/foundation-deployed-harness-architecture.md
- docs/design-docs/index.md
- docs/features/F-001-delivery-operating-model.md
- docs/exec-plans/active/current-operating-plan.md

## BDD Evidence
- Scenario IDs: F-001-S015
- Evidence links: go test ./internal/docsconsistency ./internal/docsync
- Verified by: command

## Acceptance Criteria

### Functional
- [x] The architecture doc explains the foundation harness, runtime substrate, deployed harness, and target project using glossary terms.
- [x] The doc explains why the split exists and why recursive improvement is repo-owned doctrine maintenance, not uncontrolled self-modification.
- [x] Foundation-only, mirrored, and deployed-only responsibilities are separated.
- [x] The design-doc index links to the new doc.

### Edge cases and negative paths
- [x] The doc does not imply the harness is the target of its own agents during active runs.
- [x] Source-only binary release mechanics are not described as deployed-target requirements.

### Non-goals
- Adding a new CLI command, built-in tool, database table, or skill.

### Observability, docs, and regressions
- [x] docsconsistency and docsync checks pass.
- [x] The active plan and F-001 evidence point at the new architecture slice.
