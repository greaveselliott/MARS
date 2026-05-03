---
id: MH-049
title: Define dogfood matrix supersession benchmark
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: TBD
dedupe_key: "public-example"
source: Mars parity workstream I
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "dogfood"
  category: "supersession_evidence_gap"
  severity: "high"
  confidence: "high"
---

# MH-049: Define dogfood matrix supersession benchmark

## Context

Harness should not claim Mars supersession from design intent alone. It needs a
repo-visible dogfood matrix, deterministic fake-LLM integration tests, and an
observer-mode trial against `../mars` before contributor-mode autonomy is
trusted.

## Requirements

- Add `docs/design-docs/dogfood-matrix.md`.
- Define a Harness product dogfood matrix covering setup, init, register, run,
  start, serve, scan, doctor, trust, scores, dashboard, local inference,
  optional GitHub helpers, and upgrade.
- Add deterministic fake-LLM integration tests for the full loop.
- Add a dogfood target profile for `../mars`.
- Run Harness against Mars as a target repo in observer mode first.
- Define graduation criteria for contributor mode.
- Document supersession results in a completed execution plan when the trial
  runs.

## Affected Files

- `docs/design-docs/dogfood-matrix.md`
- `docs/design-docs/index.md`
- `docs/exec-plans/`
- `internal/agent/`
- `internal/serve/`
- `internal/scanner/`
- `internal/trust/`
- `internal/scoring/`
- `.github/workflows/`

## Acceptance Criteria

### Functional

- [ ] The dogfood matrix names product surfaces, evidence commands, expected
      artifacts, and failure routing.
- [ ] Fake-LLM integration tests cover ticket creation, edit, test, direct-main
      commit, push attempt, scoring, and quality export hooks.
- [ ] A `../mars` observer-mode target profile exists with guardrails and
      non-destructive behavior.
- [ ] Contributor-mode graduation criteria are explicit and evidence-based.

### Edge cases and negative paths

- [ ] Dogfood against `../mars` cannot write to Mars without contributor-mode
      trust and guardrail approval.
- [ ] Optional GitHub paths are skipped honestly when credentials are missing.
- [ ] Failed dogfood creates or updates deduped intervention-debt tickets.

### Observability, docs, and regressions

- [ ] Tests or dry-runs prove the fake-LLM loop is deterministic in CI.
- [ ] Design docs explain where dogfood evidence is stored.
- [ ] A completed exec-plan report records the first observer-mode trial results.
