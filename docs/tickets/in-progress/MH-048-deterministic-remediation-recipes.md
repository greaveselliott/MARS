---
id: MH-048
title: Add deterministic remediation recipes
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: TBD
dedupe_key: "public-example"
source: Mars parity workstream H
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "remediation"
  category: "deterministic_repair_gap"
  severity: "high"
  confidence: "high"
---

# MH-048: Add deterministic remediation recipes

## Context

Mars learned to run deterministic maintenance and repair scripts before asking
an LLM to solve known failure classes. Harness should turn that lesson into
native remediation recipes recorded in traces and scores, so model turns are
reserved for judgment work.

## Requirements

- Add a remediation package for deterministic fixes.
- Start with recipes for dirty working tree before run, stale in-progress
  tickets, missing or invalid manifest, missing generated docs, failed doctor
  checks with known remediation, repeated scanner duplicate tickets, missing
  dependency setup, and model artifact checksum mismatch.
- Run safe recipes before LLM repair jobs.
- Record recipe attempts, outcomes, and skipped reasons in traces and scores.
- Promote repeated successful recipes into permanent doctor or setup checks.

## Progress Notes

- 2026-05-19: Claimed first bounded slice. Added `internal/remediation`
  registry and applicability planner with stable recipe IDs, safety
  classifications, candidate commands/files, skipped reasons, and next actions
  for the initial known recipe catalog. Next slice should wire safe recipes into
  `serve` before LLM repair jobs and record attempts in trace/score surfaces.

## Affected Files

- `internal/remediation/`
- `internal/serve/`
- `internal/doctor/`
- `internal/scanner/`
- `internal/setup/`
- `internal/models/`
- `internal/trace/`
- `internal/scoring/`
- `docs/design-docs/self-improvement.md`

## Acceptance Criteria

### Functional

- [x] A remediation registry can list known recipes and their applicability.
- [ ] Safe recipes run before LLM repair where configured.
- [ ] Recipe attempts are trace-linked and scored.
- [ ] Doctor or setup can adopt repeated successful recipes as permanent checks.

### Edge cases and negative paths

- [ ] Recipes never perform destructive git operations without explicit guardrail
      approval.
- [ ] Dirty working trees with user changes produce actionable blocker output
      instead of automatic reverts.
- [ ] Missing optional tools create remediation guidance, not false success.

### Observability, docs, and regressions

- [ ] Tests cover recipe selection, skipped unsafe recipes, trace recording, and
      known remediation output.
- [ ] Docs explain deterministic-first repair and how recipes graduate into
      checks.
- [ ] Quality export can include remediation attempt summaries once `MH-037`
      lands.
