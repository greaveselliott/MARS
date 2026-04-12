---
name: mars-harness-status
description: >-
  Check mars-harness health and status. Use when the user wants to check
  status, see running jobs, verify health, diagnose issues, or mentions
  mars-harness doctor or mars-harness status.
---

# Check Status

## System Health

Run the doctor to verify all subsystems:

```bash
mars-harness doctor
```

This checks: Go version, config, models, database, llama-server, disk space.

## Orchestrator Health

If the orchestrator is running, check its health endpoint:

```bash
curl http://localhost:9091/healthz
```

Expected: `{"status":"healthy"}`

## Doctor Flags

| Flag | Effect |
|---|---|
| `--config` | Custom config path |
| `--db` | Custom database path |
| `--skip-remote` | Skip network checks |
| `--json` | Output as JSON |

## Interpreting Results

- **PASS**: Check succeeded
- **WARN**: Non-critical issue, system can still operate
- **FAIL**: Critical issue with remediation command in output

## Troubleshooting

- If doctor fails on models: run `mars-harness setup`
- If doctor fails on llama-server: run `mars-harness setup`
- If doctor fails on config: check `~/.mars-harness/config.yaml`
