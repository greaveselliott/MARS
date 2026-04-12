---
name: mars-harness-serve
description: >-
  Start the mars-harness autonomous orchestrator. Use when the user wants to
  start the pipeline, begin autonomous operation, launch the orchestrator,
  run the server, or mentions mars-harness serve.
---

# Start the Orchestrator

## Prerequisites

Verify setup is complete:

```bash
mars-harness doctor
```

## Start

```bash
mars-harness serve
```

This starts the long-running orchestrator that:
- Receives GitHub webhooks and matches them to agent roles
- Fires cron schedules for periodic tasks
- Manages a concurrent worker pool executing agent jobs
- Auto-starts local inference via llama-server

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--addr` | From config (`:9091`) | Webhook listen address |
| `--concurrency` | `2` | Parallel agent workers |
| `--db` | `~/.mars-harness/db/mars.db` | SQLite database path |

## Verify

Check the health endpoint:

```bash
curl http://localhost:9091/healthz
```

Expected response: `{"status":"healthy"}`

## Stop

Send SIGINT (Ctrl+C). The orchestrator drains running jobs and shuts down gracefully.

## Troubleshooting

- **Port in use**: Change `webhook_port` in `~/.mars-harness/config.yaml` or use `--addr`
- **Models missing**: Run `mars-harness setup`
- **llama-server missing**: Run `mars-harness setup`
- **No repos registered**: Run `mars-harness register --repo /path/to/repo`
