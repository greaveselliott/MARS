---
id: MH-045
title: Complete intervention-debt signal ingestion
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["go test ./internal/telemetry ./internal/tools", "go test ./internal/serve -run 'Test(CreateInterventionDebt|InterventionDebt|RecordInterventionDebt|CheckEvolution|HandleJobFailed|BuildTicketIndex)'", "go test ./internal/qualityscore", "go test ./internal/scanner ./internal/docsconsistency ./internal/operatingmodel", "go test ./..."]
verified_by: "go test ./..."
dedupe_key: "public-example"
source: Mars parity workstream D
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "intervention-debt"
  category: "signal_ingestion_gap"
  severity: "high"
  confidence: "high"
---

# MH-045: Complete intervention-debt signal ingestion

## Context

`MH-029` added the intervention-debt ticket type and telemetry-driven ticket
creation. The Mars parity loop is broader: failed terminal agent outcomes,
guardrail blocks, repeated tool loops, human follow-up commits, reverts, stale
in-progress work, manual stops, and timeouts should also create or update
deduped intervention-debt tickets linked to traces and scores.

## Requirements

- Extend intervention-debt creation rules beyond telemetry triage.
- Create or update intervention-debt tickets for non-success terminal agent
  results, guardrail blocks, repeated tool loops, human follow-up commits,
  reverted agent commits, stale in-progress tickets, manual stops, and timeouts.
- Preserve dedupe by repo, role, target, category, and evidence window.
- Link created tickets to trace IDs, commits, role, repo, score events, and the
  originating failure where available.
- Feed eligible intervention-debt tickets into bounded evolution review.
- Keep intervention-debt priority ahead of ordinary backlog work.

## Affected Files

- `internal/serve/intervention_debt.go`
- `internal/serve/executor.go`
- `internal/evolution/`
- `internal/trace/`
- `internal/scoring/`
- `internal/tools/ticket_create.go`
- `docs/design-docs/self-reflective-telemetry.md`
- `docs/tickets/README.md`

## Acceptance Criteria

### Functional

- [x] Each configured failure signal can create or update one deduped
      intervention-debt ticket.
- [x] Tickets include role, repo, target, category, severity, confidence,
      evidence, origin, trace ID, and commit where available.
- [x] Eligible tickets are offered to bounded evolution with source evidence.
- [x] Planner and Engineer context still list intervention-debt ahead of ordinary
      backlog work.

### Edge cases and negative paths

- [x] The same repeated signal updates an existing open ticket instead of
      creating duplicates.
- [x] Missing optional GitHub data does not block local intervention-debt
      creation.
- [x] Low-confidence or unsafe evolution targets remain tickets only.

### Observability, docs, and regressions

- [x] Unit tests cover each new signal path and dedupe behavior.
- [x] Integration tests cover at least one terminal failure creating a ticket.
- [x] Docs explain which signals become intervention debt and how dedupe works.

## Completion Notes

- Added telemetry categories and triage mappings for guardrail/tool-policy
  blocks, human follow-up, reverted commits, stale in-progress tickets, manual
  stops, and terminal timeout/loop signals.
- Failed job handling now creates deduped intervention-debt tickets immediately
  with telemetry event, outcome, and trace ID evidence when available.
- Tool-policy blocks report intervention signals from inside running agent
  loops, and quality-score export creates deduped tickets for outcome and
  stale-ticket evidence without requiring optional GitHub metadata.
- Bounded evolution review now filters candidate files to allowlisted harness
  paths and leaves low-confidence or unsafe targets as tickets only.
