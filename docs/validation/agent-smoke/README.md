# Compartmentalised Agent Smoke Testing

`mars-harness validation agent-smoke` runs role-local smoke cases against
fresh, one-use target repositories. The checked-in files in this directory are
generator recipes and matrix metadata only; generated target projects are
ephemeral and live under `../demo/validation-runs/agent-smoke/` by default.

## Contract

- Test projects are generated fresh for each run and discarded by default after
  successful report generation.
- Failed runs are retained unless the operator passes `--discard-failed`.
- Seed data is created through foundation harness surfaces: target harness
  scaffold generation, `file_write`, `ticket_create`, `record_decision`,
  `git_status`, `git_commit`, `workspace_hygiene`, and related built-in tools.
- Long-lived snapshot repos and checked-in generated target projects are not
  allowed.
- Full clean-project `mars-harness start` sweeps still own end-to-end lifecycle
  confidence; agent smoke is the fast compartmentalized lane.

## Suites

- `fast` rotates one case per role using `--cycle`.
- `default` runs representative cases across core project shapes.
- `held-out` runs anti-overfit cases.
- `full` runs the complete matrix.

## Example

```bash
mars-harness validation agent-smoke --suite fast --json
mars-harness validation agent-smoke --role engineer --project-type go-api --suite fast --keep-runs
mars-harness validation agent-smoke --suite held-out --parallel 2 --timeout 10m
mars-harness validation agent-smoke --cleanup-only
```
