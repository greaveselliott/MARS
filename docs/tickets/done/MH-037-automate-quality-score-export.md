---
id: MH-037
title: Automate quality score export
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["go test ./internal/qualityscore ./internal/scoring ./internal/scanner ./internal/serve ./internal/dashboard", "go run ./cmd/mars scores export --repo . --no-ticket", "go test ./..."]
verified_by: command
dedupe_key: "public-example"
source: Mars parity workstream F
created: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars"
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

- Add `mars scores export --repo <path>` or an equivalent subcommand.
- Refresh `docs/QUALITY_SCORE.md` from role scores, recent outcomes, stuck tickets, failed dogfood, guardrail blocks, intervention debt, check results, no-op runs, human follow-up, and top telemetry triage targets.
- Preserve a small manual notes area if maintainers need to add context around generated grades.
- Detect quality regressions and create or update intervention-debt tickets.
- Link dashboard quality views to the same source data without making the dashboard the source of truth.
- Add generated target behavior so initialized repos can refresh their own quality score.

## Affected Files

- `cmd/mars`
- `internal/scoring`
- `internal/telemetry`
- `internal/serve`
- `internal/scanner/init.go`
- `docs/QUALITY_SCORE.md`
- `docs/design-docs/self-reflective-telemetry.md`
- `docs/product-specs/product-surface.md`
- `docs/exec-plans/backlog/mars-parity-supersession-plan.md`

## Acceptance Criteria

### Functional

- [x] A CLI command refreshes `docs/QUALITY_SCORE.md` from repository evidence.
- [x] The generated score includes role health, terminal outcomes, stuck tickets, dogfood failures, guardrail blocks, intervention debt, checks, no-op runs, human follow-up, and top improvement targets.
- [x] Source and initialized target repos use the same quality export contract.
- [x] Manual notes are preserved across export runs.

### Edge cases and negative paths

- [x] Missing SQLite data produces an honest "insufficient evidence" grade instead of fabricated confidence.
- [x] Export does not fail when optional GitHub telemetry is unavailable.
- [x] Regression detection dedupes intervention-debt tickets.

### Observability, docs, and regressions

- [x] Unit tests cover export rendering, missing data, and manual-note preservation.
- [x] Integration coverage verifies regression-triggered intervention debt.
- [x] Product and design docs explain the generated quality score contract.

## Completion Notes

- Added `mars scores export --repo <path>` with manual-note preservation,
  missing-database honesty, evidence signal rendering, and low-score
  intervention-debt ticket creation.
- Added generated target quality-score guidance and dashboard linking through
  the repo-owned `docs/QUALITY_SCORE.md` artifact.
- Verification includes targeted package tests, full `go test ./...`, and a
  source repo export run with `--no-ticket` because this checkout has no
  per-repo SQLite scoring database yet.
