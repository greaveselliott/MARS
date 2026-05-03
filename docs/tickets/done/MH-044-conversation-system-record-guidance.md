---
id: MH-044
title: Add conversation system record guidance
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - go test ./internal/scanner ./internal/docsconsistency ./internal/operatingmodel ./internal/doctor ./internal/updatecheck
  - go test ./...
verified_by: command
dedupe_key: "public-example"
source: Mars parity workstream C
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "documentation-discipline"
  category: "conversation_record_gap"
  severity: "high"
  confidence: "high"
---

# MH-044: Add conversation system record guidance

## Context

Mars treats significant agent conversations as inputs that must become durable
repo artifacts. Harness has operating-model and active-plan hygiene checks, but
it still needs the strict-trunk conversation-as-system-record guidance and
generated target instructions that make chat-only decisions visible to future
agents.

## Requirements

- Add `docs/design-docs/conversation-as-system-record.md`, adapted for strict
  trunk delivery.
- Update root `AGENTS.md` guidance for plans, decisions, investigations,
  quality findings, and completed work.
- Update generated target `AGENTS.md` guidance through `internal/scanner/init.go`.
- Update `.cursor/rules/documentation-discipline.mdc` with the same doctrine
  where applicable.
- Link the design decision in `docs/design-docs/index.md`.
- Reference the existing active-plan hygiene checker delivered by `MH-034`
  rather than duplicating that work.

## Affected Files

- `AGENTS.md`
- `.cursor/rules/documentation-discipline.mdc`
- `docs/design-docs/conversation-as-system-record.md`
- `docs/design-docs/index.md`
- `internal/scanner/init.go`
- `internal/scanner/scanner_test.go`

## Acceptance Criteria

### Functional

- [x] The new design doc defines which conversations must create repo artifacts.
- [x] Root guidance and generated target guidance both require persistent
      artifacts for non-trivial decisions and investigations.
- [x] The guidance is framed for direct commits to `main`, not external handoff.
- [x] Existing active-plan hygiene checks are linked as enforcement evidence.

### Edge cases and negative paths

- [x] Trivial command responses are not forced into docs churn.
- [x] Chat summaries cannot replace tickets, design docs, quality notes, or
      exec plans when those artifacts are required.
- [x] Generated target guidance remains generic and does not import Mars-specific
      SaaS constraints.

### Observability, docs, and regressions

- [x] Scanner/init tests cover the generated guidance.
- [x] Docs-consistency tests still pass.
- [x] The design-doc index points to the new record-keeping decision.
