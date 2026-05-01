---
id: MH-028
title: mars-harness doctor comprehensive health check with remediation hints
priority: medium
complexity: small
source: delivery-schedule M8
created: 2026-04-11
---

# MH-028: `mars-harness doctor` — GPU, models, inference, GitHub, webhook, DB, disk, scores

## Context

Support load drops when failures are self-explanatory. M8 closes the loop with a single diagnostic command that mirrors production checks and prints next commands.

## Requirements

- Subcommand `doctor` with `--json` for scripting
- Checks: GPU presence/driver (best-effort OS-specific), model files on disk vs manifest, inference ping (OpenAI-compatible), GitHub App/PAT auth probe (rate limit headers), webhook listener reachability (optional `--public-url`), SQLite schema version + migrator, free disk space thresholds, MH-017 rolling score snapshot, MH-019 intervention trend summary
- Exit code non-zero on any `fail` severity; `warn` does not fail by default (flag to promote warnings)
- Each finding includes `remediation` array of concrete shell commands (`mars-harness setup --step github`, etc.)

## Acceptance Criteria

### Functional (happy path)
- [x] Healthy dev machine prints all green with durations per check
- [x] `--json` emits stable schema version field for CI consumers

### Edge cases and negative paths
- [x] Partial configuration (PAT without App) reports `info` + clear mode explanation, not spurious failures
- [x] Inference timeout prints latency and suggests endpoint URL flag
- [x] DB missing runs migrator suggestion, does not auto-mutate unless `--fix` (optional flag)

### Non-goals
- Remote log collection from production fleet
- Editing config files automatically

### Observability, docs, and regressions
- [x] Table-driven unit tests per check with fakes
- [x] Docs section “Troubleshooting” references `doctor` as first step
- [x] Golden JSON snapshot for one representative failure bundle
