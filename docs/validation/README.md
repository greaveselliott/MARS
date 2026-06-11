# Validation

This directory holds durable validation artifacts: target profiles, evidence
contracts, and run reports that prove Mars Harness behavior outside unit tests.

Use `dogfood` for the runtime role and mode. Use `docs/validation/` for the
repo-visible artifacts that describe what was validated, against which target,
and with which guardrails.

## Layout

- `profiles/` contains reusable target profiles, trust level, guardrails,
  command lists, and graduation criteria.
- `reports/` contains completed validation reports for source-owned dogfood
  or supersession trials. Create this directory when the first report lands.
- `baselines/` contains dated factory-pace baselines (T-011): the
  before-measurement that Phase 3 convergence and decomposition slices are
  judged against per AD-138 and AD-284/AD-285.
- `release-blockers.md` is the per-version release publication and
  asset-verification blocker ledger extracted from the active plan on
  2026-06-11 (T-022); new release blockers are appended there.
- Target-local `docs/reports/dogfood/` remains the right place for evidence
  written during a deployed target run.

Validation reports should cite the profile, exact target path or remote, trust
level, commands run, skipped optional paths, ticket or telemetry outputs, and
the decision about whether contributor-mode validation is allowed.

Report naming, required run fields, pass criteria, and the mapping from
source-change classes to minimum archetype replays are defined in
[docs/design-docs/validation-matrix-gating.md](../design-docs/validation-matrix-gating.md)
(AD-284, AD-285).
