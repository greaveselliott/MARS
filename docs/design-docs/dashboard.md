# Dashboard Design

**Status:** Draft
**Date:** 2026-04-11
**Author:** Agent-assisted

## Context

The harness needs an operations centre accessible at `localhost:9090` during `mars-harness serve`. It must be self-contained (no external dependencies like Grafana or Prometheus) and provide real-time visibility into the autonomous pipeline.

## Key Design Decisions

### AD-011: htmx + Chart.js embedded in Go binary

The dashboard uses server-rendered HTML with Go templates, htmx (~14 KB) for dynamic partial updates without full page reloads, and Chart.js (~70 KB) for graphs and sparklines. All static assets are embedded in the Go binary via the `embed` package. SSE (Server-Sent Events) provides real-time updates.

No React, no npm, no build step, no external CDN requests. The dashboard works fully offline.

**Trade-off:** Limited interactivity compared to a full SPA. Complex visualisations (animated DAG graphs) are harder without a client-side rendering library. Mitigated by using simple layered SVG layout computed server-side for the pipeline flow graph.

### Pipeline flow graph: simple layered layout

V1 uses a server-side layered layout algorithm (roles grouped by trigger depth: scheduled roles in layer 1, event-triggered in layer 2, etc.) rendered as SVG. Not a full Sugiyama DAG layout. This is good enough for pipelines with 11-15 roles and implementable in a day.

Full auto-layout (dagre or similar) is a v2 improvement if custom role topologies need better rendering.

## Five Pages

1. **Pipeline Flow (home):** Live DAG of roles with state, scores, trust levels, next trigger. SSE updates node state on job start/complete.
2. **Role Health:** Per-role detail — outcome history, accuracy trend (30d chart with evolution annotations), context usage, guardrail violations.
3. **Throughput:** System metrics — job stats, inference performance (tokens/sec, GPU util, latency percentiles), GitHub API usage, pipeline output counts.
4. **Debug:** Job timeline (segmented bars), execution trace viewer (live and replay), webhook delivery log, error log.
5. **Evolution History:** Self-improvement timeline with before/after score analysis, guardrail inventory with staleness flags.

### AD-027: Interactive control surface (CLI + dashboard)

Operators need to pause, restart, scan, stop, and force-run individual roles without killing the process. Two control surfaces share identical backend methods on `Server`:

**CLI (terminal):** Raw terminal mode key listener (`internal/ui/keylistener.go`) plus a persistent ANSI status bar (`internal/ui/statusbar.go`). Keys: `p` (pause/resume), `r` (warm restart), `s` (re-scan), `q` (graceful stop), `h` (help). Activated automatically during `mars-harness serve` and `mars-harness start`.

**Dashboard (localhost:9090):** Control bar in the sidebar with buttons for pause/resume, restart, stop, scan, and run-role. Repo and role selectors populated from `/api/repos` and `/api/repo-roles`. State updates flow via SSE `status_change` events.

**API endpoints:**

| Method | Path | Action |
|--------|------|--------|
| POST | `/api/pause` | Pause worker pool (no new claims) |
| POST | `/api/resume` | Resume worker pool |
| POST | `/api/stop` | Graceful shutdown |
| POST | `/api/restart` | Warm restart (reload config, keep HTTP) |
| POST | `/api/scan` | Scan repo for findings |
| POST | `/api/run-role` | Enqueue a specific role |
| GET | `/api/status` | JSON status snapshot |
| GET | `/api/repos` | List registered repos |
| GET | `/api/repo-roles` | List roles for a repo |

**Design choices:**
- Pause stops claiming, not killing. Running jobs complete. Prevents data loss.
- Warm restart reloads manifests and triggers within the same process. Does not rebuild the binary.
- `q` key cancels the signal context, triggering the same graceful shutdown path as Ctrl+C.

## Discoveries

- The `pipeline.html` template uses its own full HTML structure (not `base.html`), so the control bar had to be added directly to the template sidebar rather than relying on template inheritance. Both templates should be reconciled eventually (tech debt).
- Raw terminal mode on macOS uses `TIOCGETA`/`TIOCSETA` from `golang.org/x/sys/unix`. The key listener gracefully degrades (logs a warning and disables itself) if the process is not attached to a TTY.
