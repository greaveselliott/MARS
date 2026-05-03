---
id: MH-043
title: Add checked role registry
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: TBD
dedupe_key: "public-example"
source: Mars parity workstream B
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "role-registry"
  category: "automation_inventory_gap"
  severity: "high"
  confidence: "high"
---

# MH-043: Add checked role registry

## Context

Mars has `docs/automations/BOTS.md` as a human-readable automation inventory.
Harness has manifests and design docs, but no single checked-in artifact that
answers which autonomous roles exist, what modes they run, and what tools,
guardrails, trust levels, triggers, and score signals apply.

## Requirements

- Add a repo-visible role registry under `docs/roles/ROLES.md` or
  `docs/automations/ROLES.md`.
- Include role, canonical domain, modes, trigger sources, schedules, tools,
  trust level, guardrails, model routing, scoring signals, and escalation
  behavior.
- Generate an equivalent target registry during `mars-harness init`.
- Add a consistency check that compares generated manifests with the registry.
- Surface registry health in `mars-harness doctor`.

## Affected Files

- `docs/roles/ROLES.md` or `docs/automations/ROLES.md`
- `internal/scanner/init.go`
- `internal/docsconsistency/`
- `internal/doctor/`
- `examples/sample-bundle/.harness/manifest.yaml`
- `docs/design-docs/index.md`

## Acceptance Criteria

### Functional

- [ ] A human or agent can find all default roles and their modes in one
      checked-in registry.
- [ ] `mars-harness init` emits the target registry.
- [ ] Docs-consistency or an equivalent check fails when a default manifest role
      is missing from the registry.
- [ ] `mars-harness doctor --repo <repo>` reports role-registry health.

### Edge cases and negative paths

- [ ] Custom target roles are represented without being mistaken for missing
      source defaults.
- [ ] Optional GitHub triggers are marked optional and are not treated as the
      default delivery model.
- [ ] Registry checks produce actionable remediation text.

### Observability, docs, and regressions

- [ ] Tests cover registry generation, missing-role detection, and doctor
      reporting.
- [ ] The registry links to relevant design docs instead of duplicating long
      architecture sections.
- [ ] Generated target docs remain strict-trunk native.
