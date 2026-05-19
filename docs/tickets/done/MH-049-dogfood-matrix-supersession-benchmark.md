---
id: MH-049
title: Define dogfood matrix supersession benchmark
priority: high
complexity: medium
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md", "docs/validation/reports/2026-05-19-mars-observer-validation.md"]
verified_by: "go test ./...; go run ./cmd/mars-harness doctor --repo /path/to/local-redacted --json; go run ./cmd/mars-harness update check --repo /path/to/local-redacted --skip-remote --json; go run ./cmd/mars-harness tools run git_status --repo /path/to/local-redacted --trust observer --json; observer-trust file_write block; run --dry-run on <validation-root>"
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
- Remaining in this ticket: `../mars` observer profile/trial and the observer
  validation report. The broader demo target lifecycle now has source-owned
  validation evidence.
- 2026-05-19: Claimed the next foundation build slice. Added broader fake-LLM
  dogfood coverage for dogfood report writing, bounded test execution,
  deduped ticket creation, direct-main commit, no-remote push attempt, scoring
  outcome, and `scores export` quality hook. Added observer-trust coverage that
  blocks mutating writes before Mars contributor-mode graduation.
- 2026-05-19: Ran a live `demo-123` Space Invaders target from a clean repo.
  The first replay found a completed COO `feature_contract` handoff that
  stopped dispatch before product tickets. The source fix routes completed
  same-role planning needs to the role's default forward owner. A fresh replay
  reached CEO, COO, CTO, Engineer, QA, Security, Dogfood, and Release Manager,
  created and closed ordinary product ticket `T-001`, and recorded the next
  source-owned blockers as `T-007` and `T-008`.
- 2026-05-19: Ran the first Mars observer validation against
  `/path/to/local-redacted` at
  `aa79b0039e7a2fb75c539fa427c02160ff2a33b9`. The real Mars target stayed
  clean, observer trust blocked mutating `file_write`, and the report recorded
  missing `.harness/`, operating-model drift, role-registry unavailability,
  active-plan hygiene warnings, and the need for `T-009` because `run
  --dry-run` auto-initializes uninitialized targets.

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
- [x] Broader fake-LLM integration tests cover ticket creation, test, direct-main
      commit, push attempt, scoring, and quality export hooks.
- [x] A `../mars` observer-mode target profile exists with guardrails and
      non-destructive behavior.
- [x] Contributor-mode graduation criteria are explicit and evidence-based.

### Edge cases and negative paths

- [x] Dogfood against `../mars` cannot write to Mars without contributor-mode
      trust and guardrail approval.
- [x] Optional GitHub paths are skipped honestly when credentials are missing.
- [x] Failed fast-gate dogfood creates or updates deduped intervention-debt
      tickets without secondary ticket-gate amplification.
- [x] Broader failed dogfood runs create or update deduped intervention-debt
      tickets.

### Observability, docs, and regressions

- [x] Tests prove the fast fake-LLM containment loop is deterministic in CI.
- [x] Tests or dry-runs prove the broader fake-LLM loop is deterministic in CI.
- [x] Design docs explain where dogfood evidence is stored.
- [x] A completed validation report records the live `demo-123` lifecycle
      results.
- [x] A completed validation report records the first observer-mode `../mars`
      trial results.
