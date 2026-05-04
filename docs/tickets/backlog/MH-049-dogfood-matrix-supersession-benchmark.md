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
- Add deterministic fake-LLM integration tests for the fast foundation
  containment gate.
- Keep the broader dogfood loop open for ticket creation, test, commit, push,
  scoring, quality export, and release evidence beyond the fast gate.
- Add a dogfood target profile for `../mars`.
- Run Harness against Mars as a target repo in observer mode first.
- Define graduation criteria for contributor mode.
- Document supersession results in a completed execution plan when the trial
  runs.

## Status Notes

- Completed in the fast gate: generated target baseline, real executor, fake
  OpenAI-compatible LLM endpoint, contributor trust, controlled mutation,
  generated scaffold commits across `init`, `start`, `run`, `register`, and
  `scan`, destructive shell preflight block, dirty-worktree containment,
  read-only shell inspection behavior, intervention-debt dedupe, ticket-gate
  fallout suppression, scoring, and bounded triage updates.
- Remaining in this ticket: broader matrix coverage, full dogfood loop evidence,
  `../mars` observer profile/trial, contributor-mode graduation criteria, and a
  completed validation report.

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

- [x] The dogfood matrix names product surfaces, evidence commands, expected
      artifacts, and failure routing.
- [x] Fast fake-LLM containment tests cover generated target execution,
      controlled edit, trust policy, dirty-worktree containment, scoring,
      telemetry, and intervention-debt routing.
- [x] `init`, `start`, `run`, `register`, and `scan` commit the generated
      scaffold baseline without staging pre-existing target work.
- [ ] Broader fake-LLM integration tests cover ticket creation, test, direct-main
      commit, push attempt, scoring, and quality export hooks.
- [ ] A `../mars` observer-mode target profile exists with guardrails and
      non-destructive behavior.
- [ ] Contributor-mode graduation criteria are explicit and evidence-based.

### Edge cases and negative paths

- [ ] Dogfood against `../mars` cannot write to Mars without contributor-mode
      trust and guardrail approval.
- [ ] Optional GitHub paths are skipped honestly when credentials are missing.
- [x] Failed fast-gate dogfood creates or updates deduped intervention-debt
      tickets without secondary ticket-gate amplification.
- [ ] Broader failed dogfood runs create or update deduped intervention-debt
      tickets.

### Observability, docs, and regressions

- [x] Tests prove the fast fake-LLM containment loop is deterministic in CI.
- [ ] Tests or dry-runs prove the broader fake-LLM loop is deterministic in CI.
- [ ] Design docs explain where dogfood evidence is stored.
- [ ] A completed exec-plan report records the first observer-mode trial results.
