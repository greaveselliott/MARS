---
id: MH-042
title: Create canonical harness operating model
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - go test ./internal/docsconsistency ./internal/scanner
  - go test ./internal/bundle
  - go test ./...
verified_by: go test ./...
dedupe_key: "public-example"
source: Mars parity workstream A
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "role-model"
  category: "operating_model_gap"
  severity: "high"
  confidence: "high"
---

# MH-042: Create canonical harness operating model

## Context

The Mars parity audit found that Mars has a clearer six-domain automation model
than the current Harness role docs. Harness needs a strict-trunk-native design
doc that defines the canonical domains and maps the current default roles into
domain modes without breaking existing bundles.

## Requirements

- Add `docs/design-docs/harness-operating-model.md`.
- Define Planner, Engineer, Reviewer, Maintainer, End-to-End Tester, and
  Orchestrator as canonical operating domains.
- Map current default roles to those domains and explicit modes.
- Decide whether manifests should expose six canonical roles with modes or
  retain explicit roles while declaring the six-domain model as canonical.
- Update `docs/design-docs/trigger-orchestration.md` where the role model
  affects trigger routing.
- Index the new design doc in `docs/design-docs/index.md`.

## Affected Files

- `docs/design-docs/harness-operating-model.md`
- `docs/design-docs/trigger-orchestration.md`
- `docs/design-docs/index.md`
- `internal/scanner/init.go`
- `examples/sample-bundle/.harness/manifest.yaml`

## Acceptance Criteria

### Functional

- [x] The six canonical domains are defined with responsibilities and mode
      boundaries.
- [x] Every current default role maps to one canonical domain and mode.
- [x] The design explains the migration path for existing 11-role bundles.
- [x] Trigger orchestration docs use the same domain and mode vocabulary.

### Edge cases and negative paths

- [x] Existing manifests are not invalidated by the new canonical model.
- [x] The doc explicitly rejects PR/branch delivery as the default workflow.
- [x] Domain modes do not hide guardrail, trust, or scoring responsibilities.

### Observability, docs, and regressions

- [x] Design-doc index includes the new decision.
- [x] Target-generation implications are recorded for a follow-up ticket when
      not implemented in this slice.
- [x] `go test ./internal/docsconsistency ./internal/scanner` passes.

## Completion Evidence

- `go test ./internal/docsconsistency ./internal/scanner`
- `go test ./internal/bundle`
- `go test ./...`
