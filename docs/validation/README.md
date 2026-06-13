# Validation

This directory holds durable validation artifacts: target profiles, evidence
contracts, and run reports that prove Mars Harness behavior outside unit tests.

Use `dogfood` for the runtime role and mode. Use `docs/validation/` for the
repo-visible artifacts that describe what was validated, against which target,
and with which guardrails.

## Layout

- `profiles/` contains reusable target profiles (frontmatter + brief body), trust
  level, guardrails, command lists, and graduation criteria. Foundation replays
  use these via `scripts/validation-target.mjs create --profile <slug>` (AD-293).
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
(AD-284, AD-285). The canonical **foundation** requirement to run those replays
on a **clean project** before treating runtime fixes as done is
[docs/design-docs/foundation-operating-model.md](../design-docs/foundation-operating-model.md)
(AD-291, AD-292, AD-293).

Closure reports that summarize AD-284 replay batches should be checked with:

```bash
mars-harness validation check-closure --report docs/validation/reports/<report>.md
```

The gate fails when a closure verdict claims confirmed or complete while a
required archetype row is blocked, failed, or pending. It is an honesty gate for
the report; it does not replace the fresh target runs themselves.

## Ephemeral validation runs (AD-293)

Foundation replays create a **new folder every time** under grouped storage:

| Item | Default |
| --- | --- |
| Parent directory | `../demo/validation-runs/` (override `MH_VALIDATION_ROOT`) |
| Create | `node scripts/validation-target.mjs create --profile <slug> [--label <purpose>]` |
| List | `node scripts/validation-target.mjs list` |
| Discard run + DB | `node scripts/validation-target.mjs discard <run-id>` |
| Retention | `node scripts/validation-target.mjs cleanup --keep 3` |
| Monitor | `node scripts/replay-progress.mjs --repo <run-folder-basename>` |

Discarded runs archive `.validation-run.json` under
`validation-runs/.discarded/` and delete the active folder and per-repo SQLite DB.
Legacy numbered demos remain historical evidence only.
