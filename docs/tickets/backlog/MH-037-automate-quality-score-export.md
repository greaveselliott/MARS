---
id: MH-037
title: Automate quality score export
priority: high
complexity: medium
kind: intervention-debt
dedupe_key: "public-example"
source: Mars parity workstream F
created: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "quality-score"
  category: "repo_visible_scoring"
  severity: "high"
---

# MH-037: Automate quality score export

## Context

`docs/QUALITY_SCORE.md` now exists as a seeded audit artifact, but the Mars
parity target is stronger: quality should be generated from live harness
evidence and used as a control signal. Future agents should not hand-maintain
grades once scoring, telemetry, dogfood, and ticket-state data are available.

## Requirements

- Add `mars-harness scores export --repo <path>` or an equivalent subcommand.
- Refresh `docs/QUALITY_SCORE.md` from role scores, recent outcomes, stuck tickets, failed dogfood, guardrail blocks, intervention debt, check results, no-op runs, human follow-up, and top telemetry triage targets.
- Preserve a small manual notes area if maintainers need to add context around generated grades.
- Detect quality regressions and create or update intervention-debt tickets.
- Link dashboard quality views to the same source data without making the dashboard the source of truth.
- Add generated target behavior so initialized repos can refresh their own quality score.

## Affected Files

- `cmd/mars-harness`
- `internal/scoring`
- `internal/telemetry`
- `internal/serve`
- `internal/scanner/init.go`
- `docs/QUALITY_SCORE.md`
- `docs/design-docs/self-reflective-telemetry.md`
- `docs/product-specs/product-surface.md`
- `docs/exec-plans/active/mars-parity-supersession-plan.md`

## Acceptance Criteria

### Functional

- [ ] A CLI command refreshes `docs/QUALITY_SCORE.md` from repository evidence.
- [ ] The generated score includes role health, terminal outcomes, stuck tickets, dogfood failures, guardrail blocks, intervention debt, checks, no-op runs, human follow-up, and top improvement targets.
- [ ] Source and initialized target repos use the same quality export contract.
- [ ] Manual notes are preserved across export runs.

### Edge cases and negative paths

- [ ] Missing SQLite data produces an honest "insufficient evidence" grade instead of fabricated confidence.
- [ ] Export does not fail when optional GitHub telemetry is unavailable.
- [ ] Regression detection dedupes intervention-debt tickets.

### Observability, docs, and regressions

- [ ] Unit tests cover export rendering, missing data, and manual-note preservation.
- [ ] Integration coverage verifies regression-triggered intervention debt.
- [ ] Product and design docs explain the generated quality score contract.
