---
id: MH-036
title: Seed repo-visible quality score
priority: high
complexity: small
kind: intervention-debt
dedupe_key: "public-example"
source: Mars QUALITY_SCORE.md parity audit
created: 2026-05-02
completed: 2026-05-02
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "quality-score"
  category: "mars_parity_gap"
  severity: "high"
---

# MH-036: Seed repo-visible quality score

## Context

Mars has `docs/QUALITY_SCORE.md`, which grades harness functionality from A to
F and makes quality visible in the repo. Mars Harness referenced this pattern in
the parity plan, but did not yet have the source artifact or generated target
seed.

## Requirements

- Add source `docs/QUALITY_SCORE.md` with an honest current A-F scorecard.
- Generate starter `docs/QUALITY_SCORE.md` for initialized target repos.
- Update target `AGENTS.md` guidance so agents read the score before claiming quality or readiness.
- Record the self-reflective telemetry decision and product surface change.
- Keep score export automation as follow-up work instead of pretending the seed is live telemetry.

## Affected Files

- `docs/QUALITY_SCORE.md`
- `internal/scanner/init.go`
- `internal/scanner/scanner_test.go`
- `docs/design-docs/self-reflective-telemetry.md`
- `docs/design-docs/index.md`
- `docs/product-specs/product-surface.md`
- `docs/exec-plans/active/mars-parity-supersession-plan.md`
- `AGENTS.md`

## Acceptance Criteria

### Functional

- [x] Source harness has a repo-visible A-F quality score artifact.
- [x] Initialized target harnesses receive a starter quality score artifact.
- [x] Generated target guidance points agents at the scorecard before quality claims.
- [x] Automation follow-up remains open and visible.

### Edge cases and negative paths

- [x] The source scorecard does not claim live generation from telemetry yet.
- [x] The target scorecard is project-agnostic and asks future agents to replace seed evidence.

### Observability, docs, and regressions

- [x] The product surface documents the generated target scorecard.
- [x] The design decision is indexed.
- [x] The init test checks the generated target artifact.
- [x] No unchecked acceptance criteria remain in this completed ticket.
