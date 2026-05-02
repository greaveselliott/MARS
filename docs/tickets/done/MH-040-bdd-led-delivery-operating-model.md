---
id: MH-040
title: Implement BDD-led goal-driven walking-skeleton operating model
priority: high
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-001-S001", "F-001-S002", "F-001-S003", "F-001-S004", "F-001-S005", "F-001-S006"]
end_to_end_evidence: required
evidence_links: ["go test ./internal/docsconsistency", "go test ./internal/serve", "go test ./internal/scanner", "go test ./internal/updatecheck", "go test ./internal/doctor", "go test ./internal/release", "go test ./internal/telemetry", "go test ./..."]
verified_by: "Codex using go test"
source: AD-074 implementation plan from user chat
created: 2026-05-02
depends_on: []
---

# MH-040: Implement BDD-led goal-driven walking-skeleton operating model

## Context

The user requested AD-074 as the canonical operating model: goals drive BDD
feature contracts; the single active exec plan schedules failing scenarios;
tickets implement the next failing scenario group; walking skeleton delivery
makes scenarios pass through real evidence.

## Requirements

- Add AD-074 and index it.
- Add source goals and BDD feature-contract docs.
- Update the active plan and backlog plan metadata.
- Extend ticket metadata and completion evidence expectations.
- Mirror the model into generated target harness docs, prompts, knowledge
  routes, tickets, exec plans, quality score, and design docs.
- Report target operating-model drift through update check and doctor.
- Block feature ticket completion without BDD evidence.

## Affected Files

- `AGENTS.md`
- `docs/design-docs/`
- `docs/goals/`
- `docs/features/`
- `docs/exec-plans/`
- `docs/tickets/README.md`
- `docs/QUALITY_SCORE.md`
- `docs/product-specs/product-surface.md`
- `internal/scanner/init.go`
- `internal/tools/ticket_create.go`
- `internal/serve/ticket_gate.go`
- `internal/updatecheck/`
- `internal/telemetry/`
- `internal/doctor/`
- `internal/operatingmodel/`
- `internal/docsconsistency/`

## BDD Evidence

- Scenario IDs: F-001-S001, F-001-S002, F-001-S003, F-001-S004, F-001-S005, F-001-S006
- Evidence links:
  - `go test ./internal/docsconsistency`
  - `go test ./internal/serve`
  - `go test ./internal/scanner`
  - `go test ./internal/updatecheck`
  - `go test ./internal/doctor`
  - `go test ./internal/release`
  - `go test ./internal/telemetry`
  - `go test ./...`
- Verified by: Codex using `go test`

## Acceptance Criteria

### Functional

- [x] AD-074 is indexed and explains why the model exists.
- [x] Source harness contains goal, BDD, exec-plan, ticket, quality, and product-surface guidance.
- [x] Generated target harness contains mirrored goal, BDD, exec-plan, ticket, role, quality, and knowledge-route guidance.
- [x] Active exec plan references active goals and a BDD feature contract.
- [x] Feature ticket completion requires BDD scenario evidence before done.
- [x] Update check and doctor report stale or missing operating-model artifacts.
- [x] Telemetry proposals can create/update active goals or observations with dedupe keys.

### Edge cases and negative paths

- [x] Enabler tickets can complete without BDD evidence but do not claim shipped feature value.
- [x] Existing target adoption is non-destructive; update check and doctor report drift, while update writes missing defaults only.
- [x] Old tickets without `work_type` are not retroactively blocked unless they claim feature evidence requirements.

### Non-goals

- Custom Gherkin parser.
- Fully autonomous telemetry-derived goal creation in this slice.
- Overwriting user-owned target docs during update.

### Observability, docs, and regressions

- [x] Docs-consistency tests cover AD-074 artifacts, active goal/feature references, feature contract fields, and exec-plan metadata.
- [x] Scanner tests cover generated target parity.
- [x] Serve tests cover feature ticket evidence gates.
