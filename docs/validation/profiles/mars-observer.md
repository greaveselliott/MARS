# Mars Observer Validation Profile

**Target:** `../mars`
**Mode:** observer
**Purpose:** supersession benchmark for proving MARS can inspect the
legacy Mars repo without writing to it, committing to it, or claiming delivery
before evidence exists.

## Required Setup

Run from this repo with the target path resolved explicitly:

```bash
mars doctor --repo ../mars --json
mars update check --repo ../mars --skip-remote --json
mars run engineer --repo ../mars --dry-run --trace --no-init
mars tools run git_status --repo ../mars --trust observer
```

Use `--no-init` for the dry-run step when `.harness/manifest.yaml` is missing.
The normal `run` command intentionally auto-initializes missing harness
scaffolds for plug-and-play execution; observer validation must opt out so the
real target remains unchanged.

The first validation pass must not call `file_write`, `shell_exec`,
`mars_cli`, `git_commit`, `git_push`, `record_decision`,
`ticket_create`, `tool_create`, or `release_orchestrate` unless the command is
run against a temporary copy of the target repo.

If an agent attempts one of those mutating tools while trust is `observer`, the
correct result is a blocked disposition plus source-side evidence. Do not
retry with contributor trust inside the same observer report.

## Guardrails

- Trust level stays `observer`.
- Generated findings are recorded in this repo, not in `../mars`.
- Optional GitHub checks are skipped when credentials or remote visibility are
  absent, with the skip reason in the validation report.
- Local inference failures remain warnings until the setup/runtime evidence
  path is proven on the host.
- No target harness files are overwritten; drift becomes a migration ticket.

## Graduation Criteria

The target can move to contributor-mode dogfood only after the observer report
shows:

- no attempted writes, commits, pushes, or destructive shell commands
- guardrails and role routes were loaded or their absence was recorded
- all skipped optional paths have reasons
- follow-up tickets exist for every blocker that prevents a truthful BDD claim
- the report names the exact commit or worktree state inspected
- at least one source-side maintainer explicitly accepts the observer report as
  safe enough for a contributor-mode trial
