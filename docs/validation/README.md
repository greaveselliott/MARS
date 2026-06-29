# Validation

This directory holds durable validation artifacts: target profiles, evidence
contracts, and run reports that prove MARS behavior outside unit tests.

Use `dogfood` for the runtime role and mode. Use `docs/validation/` for the
repo-visible artifacts that describe what was validated, against which target,
and with which guardrails.

## Layout

- `profiles/` contains reusable target profiles, trust level, guardrails,
  command lists, and graduation criteria.
- `agent-smoke/` contains the checked-in matrix and generator recipes for
  Compartmentalised Agent Smoke testing. Generated targets are ephemeral and
  are not stored in this repo.
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

Matrix run reports are required whenever a validation matrix is run or
attempted. They must name the selected matrix or suite, all selected cases or
archetypes, exact command, source ref or installed binary, model identity,
target/run paths, DB/log/trace paths, per-case status, failure class, cleanup
status, and any exact blocker or rerun command. A failed setup still gets a
matrix run report; do not leave the result only in chat or terminal scrollback.

Reports created on or after 2026-06-16 must lead with a Primary Outcome Contract:
`Primary Outcome`, `Primary Pass Gate`, `Primary Status`,
`Current Primary Blocker`, `Next Primary Action`, and `Supporting Evidence`.
Allowed `Primary Status` values are `primary_passed`, `primary_failed`,
`primary_blocked`, and `supporting_only`. If `Primary Status` is not
`primary_passed`, supporting checks remain evidence only and cannot be framed as
completion of the core validation claim.

Fake, stub, mock, canned, or scripted LLM endpoints are not validation
evidence. Runs backed by those endpoints can be recorded only as deterministic
plumbing checks and must be excluded from live role/model/lifecycle pass
claims.

Report naming, required run fields, pass criteria, and the mapping from
source-change classes to minimum archetype replays are defined in
[docs/design-docs/validation-matrix-gating.md](../design-docs/validation-matrix-gating.md)
(AD-284, AD-285). The canonical **foundation** requirement to run those replays
on a **clean project** before treating runtime fixes as done is
[docs/design-docs/foundation-operating-model.md](../design-docs/foundation-operating-model.md)
(AD-291, AD-292).

## Compartmentalised agent smoke

Use `mars validation agent-smoke` when a source change should be
checked against many role-stage target states faster than a full lifecycle
sweep. The runner creates fresh one-use targets under
`../demo/validation-runs/agent-smoke/` by default, seeds them through foundation
scaffold/tool surfaces, executes selected roles through the server job path,
writes JSON or Markdown evidence, and discards successful runs unless
`--keep-runs` is set. `--parallel` runs independent ephemeral repos and DBs at
the same time, and the runner suppresses follow-on dispatch after the target
role while recording the would-be next role and terminal disposition.
Run it from the MARS source checkout; there is no `--repo` flag because the
matrix is loaded from the current foundation repo. Use `--root` only to move
ephemeral run storage.
Live runs default to a 32-turn role budget so ticket claim, implementation,
validation evidence, ticket closure, and terminal disposition can complete.
Each generated target contains `docs/validation/agent-smoke/current-case.md`;
agents read that target-local contract instead of trying to inspect the
foundation matrix from inside the ephemeral repo.

Smoke examples:

```bash
mars validation agent-smoke --suite fast --json
mars validation agent-smoke --case static-web-ticket --role engineer --project-type static-web --suite fast --keep-runs
mars validation agent-smoke --role engineer --project-type go-api --suite default --keep-runs
mars validation agent-smoke --suite held-out --parallel 2 --single-server --single-server-tier coding --timeout 10m
mars validation agent-smoke --cleanup-only
```

Successful run directories are deleted unless `--keep-runs` is set. Failed
runs are retained unless `--discard-failed` is set. Cleanup removes every
`run-*` directory under the selected root and exits before matrix loading or
Markdown report writing, so use a dedicated root for agent-smoke output.

Agent smoke complements full clean-project sweeps. It does not prove
cross-agent handoff quality by itself, and it does not remove the AD-284/AD-291
requirement for broad lifecycle or release claims.

Current completion evidence for the full live matrix is
[2026-06-15-agent-smoke-full-matrix.md](reports/2026-06-15-agent-smoke-full-matrix.md):
`74 passed`, `0 failed`, `74 selected` with `--parallel 2` against the local
model router.
