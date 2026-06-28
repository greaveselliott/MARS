# Compartmentalised Agent Smoke Testing

`mars validation agent-smoke` runs role-local smoke cases against
fresh, one-use target repositories. The checked-in files in this directory are
generator recipes and matrix metadata only; generated target projects are
ephemeral and live under `../demo/validation-runs/agent-smoke/` by default.
The primary purpose is live per-agent execution: selected cases run through the
same server job executor as autonomous jobs and can run in parallel across
isolated repos and databases.
Local-model runs default to `--single-server` with `--single-server-tier coding`
so parallel selected cases share one llama-server process instead of quietly
starting separate tier servers.

## Contract

- Test projects are generated fresh for each run and discarded by default after
  successful report generation.
- Failed runs are retained unless the operator passes `--discard-failed`.
- Seed data is created through foundation harness surfaces: target harness
  scaffold generation, `file_write`, `ticket_create`, `record_decision`,
  `git_status`, `git_commit`, `workspace_hygiene`, and related built-in tools.
- Every generated target includes
  `docs/validation/agent-smoke/current-case.md`, a target-local case contract
  derived from the checked-in matrix. Agents read that file during live smoke
  runs; generated targets do not contain the foundation matrix itself.
- Each non-source-only selected role is executed through the server job path
  with the generated target repo, isolated DB, trust/org-state stores, trace
  persistence, structured trigger context, and role tool policy.
- Live execution defaults to `--max-turns 32`, because the smoke contract may
  include ticket claim, implementation, focused validation, ticket closure,
  push-or-skip behavior, and terminal disposition in one role run.
- Follow-on dispatch is suppressed by not invoking the job-completion routing
  callback; reports record the would-be next role and terminal disposition.
- Reports record inference topology. A validation report for the primary lane
  must show `single_server: true` or the Markdown single-server topology line
  when claiming single-server parallel coverage.
- Long-lived snapshot repos and checked-in generated target projects are not
  allowed.
- Full clean-project `mars start` sweeps still own end-to-end lifecycle
  confidence; agent smoke is the fast compartmentalized lane.
- `--model-endpoint` is only for a real OpenAI-compatible model endpoint.
  Fake, stub, mock, canned, or scripted endpoints are test plumbing only and
  must not be counted as validation evidence.

## Suites

- `fast` rotates one case per role using `--cycle`.
- `default` runs representative cases across core project shapes.
- `held-out` runs anti-overfit cases.
- `full` runs the complete matrix.

## Example

```bash
mars validation agent-smoke --suite fast --json
mars validation agent-smoke --role engineer --project-type go-api --suite fast --keep-runs
mars validation agent-smoke --suite held-out --parallel 2 --single-server --single-server-tier coding --timeout 10m
mars validation agent-smoke --cleanup-only
```

Use `--fixture-only` only to debug generator recipes and fixture linting; it is
not valid evidence that an agent role executed.
