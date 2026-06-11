---
id: T-027
title: Promote convergence failures and guardrail block rates into scores export
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-008-S007", "F-008-S009"]
end_to_end_evidence: not_applicable
evidence_links: ["AD-283 in docs/design-docs/self-reflective-telemetry.md, indexed in docs/design-docs/index.md", "F-008-S009 in docs/features/F-008-scoring-trust-quality.md", "go test ./internal/qualityscore -run 'TestExportRendersConvergenceAndGuardrailSignals|TestSummarizeConvergenceCleanWindowReportsNoFailures'", "Regenerated docs/QUALITY_SCORE.md for this repo renders the Convergence And Guardrails section and clean-window signal", "No runtime telemetry recording changed: data joins existing trace summaries and scoring outcome counts, so no canary replay is required"]
verified_by: "command"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; convergence and guardrail evidence renders in scores export."
source: Foundation improvement plan Phase 2 WS-C (provisional T-027)
created: 2026-06-11
depends_on: []
---

# T-027: Promote convergence failures and guardrail block rates into scores export

## Context

The 2026-06-11 foundation review found that operator telemetry stops at Factory Pace averages: every unproductive stop collapses into one limit-stop count and guardrail blocks appear only as a raw total. Operators triaging convergence problems need the failure kinds separated (circle detection vs budget exhaustion vs no-op terminal outcomes) and guardrail intervention expressed as a rate over terminal outcomes.

Ownership classification: foundation-owned quality-export behavior. Extends T-011's Factory Pace surface rather than duplicating it.

**Replay-validation flag:** no runtime telemetry recording changed. The new section joins existing trace summary and scoring outcome tables, so no canary replay is required for this slice. The next phase's T-011 baseline replay will exercise the new section against live data as a side effect.

## Requirements

- `scores export` renders a Convergence And Guardrails section in docs/QUALITY_SCORE.md with per repo/role counts: circle-detected stops, max-turn/max-tool stops, other limit stops, no-op terminal outcomes, guardrail blocks, and guardrail block rate over terminal outcomes.
- Evidence Signals gains a Convergence failures roll-up row across all repo/role rows.
- A clean window reports that no failures were recorded; missing evidence still renders as missing rather than healthy (F-008-S007).
- Data comes from existing trace summaries (trace outcome strings) and scoring outcome counts; no new runtime recording.

## Affected Files

- internal/qualityscore/export.go (convergenceSummary, summarizeConvergence, render section + signal row)
- internal/qualityscore/export_test.go
- docs/design-docs/self-reflective-telemetry.md (AD-283), docs/design-docs/index.md
- docs/features/F-008-scoring-trust-quality.md (F-008-S009)
- docs/QUALITY_SCORE.md (regenerated)

## BDD Evidence

- Scenario IDs: F-008-S009 (documented in the feature contract; ticket filed as enabler because earlier F-008 scenarios predate ticket scenario coverage tracking and the create policy requires earliest-uncovered-scenario ordering)
- Evidence links: see frontmatter
- Verified by: command

## Acceptance Criteria

### Functional (happy path)
- [x] Circle-detected, max-turn/max-tool, other limit stops, no-op outcomes, guardrail blocks, and block rate render per repo/role.
- [x] The Evidence Signals roll-up totals match the per-row counts.

### Edge cases and negative paths
- [x] Clean windows report no failures recorded instead of implying missing evidence is healthy.
- [x] Roles with outcomes but no traces still render their outcome-derived counts.

### Non-goals
- New runtime telemetry recording or trace schema changes.
- Automatic intervention-debt creation from convergence rows (existing outcome-signal ticketing already covers guardrail/no-op categories).

### Observability, docs, and regressions
- [x] AD-283 recorded and indexed; F-008-S009 documented with evidence.
- [x] Package tests cover populated and clean windows.
