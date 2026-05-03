# Mars Observer Validation Profile

**Target:** `../mars`
**Mode:** observer
**Purpose:** supersession benchmark for proving Mars Harness can inspect the
legacy Mars repo without writing to it, committing to it, or claiming delivery
before evidence exists.

## Required Setup

Run from this repo with the target path resolved explicitly:

```bash
mars-harness doctor --repo ../mars --json
mars-harness update check --repo ../mars --skip-remote --json
mars-harness run engineer --repo ../mars --dry-run --trace
mars-harness tools run git_status --repo ../mars --trust observer
```

The first validation pass must not call `file_write`, `shell_exec`,
`mars_harness_cli`, `git_commit`, `git_push`, `record_decision`,
`ticket_create`, `tool_create`, or `release_orchestrate` unless the command is
run against a temporary copy of the target repo.

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
