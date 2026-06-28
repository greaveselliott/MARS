---
name: mars-status
description: >-
  Check mars health and status. Use when the user wants to check
  status, see running jobs, verify health, diagnose issues, or mentions
  mars doctor or mars status.
---

# Check Status

## System Health

Run the doctor to verify all subsystems:

```bash
mars doctor
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

- If doctor fails on models: run `mars setup`
- If doctor fails on llama-server: run `mars setup`
- If doctor fails on config: check `~/.mars/config.yaml`
