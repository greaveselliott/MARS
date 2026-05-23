# Dashboard Design

**Status:** Draft
**Date:** 2026-04-11
**Updated:** 2026-05-20
**Author:** Agent-assisted

## Context

The harness needs an operations centre accessible at `localhost:9090` during `mars-harness serve`. It must be self-contained (no external dependencies like Grafana or Prometheus) and provide real-time visibility into the autonomous pipeline.

The embedded htmx and Chart.js dashboard is the legacy/current implementation.
The next dashboard generation is planned as the TanStack control plane governed
by [../product-specs/dashboard-control-plane.md](../product-specs/dashboard-control-plane.md)
and [../features/F-010-dashboard-control-plane.md](../features/F-010-dashboard-control-plane.md).

## Key Design Decisions

### AD-011: htmx + Chart.js embedded in Go binary

The dashboard uses server-rendered HTML with Go templates, htmx (~14 KB) for dynamic partial updates without full page reloads, and Chart.js (~70 KB) for graphs and sparklines. All static assets are embedded in the Go binary via the `embed` package. SSE (Server-Sent Events) provides real-time updates.

No React, no npm, no build step, no external CDN requests. The dashboard works fully offline.

**Trade-off:** Limited interactivity compared to a full SPA. Complex visualisations (animated DAG graphs) are harder without a client-side rendering library. Mitigated by using simple layered SVG layout computed server-side for the pipeline flow graph.

**Status note:** AD-011 describes the legacy/current dashboard implementation.
It remains true for shipped code until the TanStack migration lands, but it no
longer governs the planned next-generation dashboard.

### AD-156: TanStack Start Sidecar Behind Go Dashboard Gateway

The next dashboard generation uses a TanStack Start sidecar for the frontend and
keeps the Go server as the public dashboard gateway. The sidecar may render and
hydrate the dashboard UI, but the Go gateway owns local-admin auth, session
validation, CSRF checks, API authorization, event streams, command enqueueing,
and access to queue, telemetry, scoring, trust, model, and trace state.

Node.js `24.x` and `pnpm@11.1.1` are explicit external prerequisites for the
sidecar. Mars Harness must not bundle Node, install Node, download Node, or run
package-manager installers. Missing or wrong versions produce actionable
remediation output and leave the core orchestrator usable.

The dashboard API model is read-concurrent by design. Routes serving Overview,
Active Work, Preview, Agent Roster, Models, DORA, telemetry, token usage, and
command status use cancellable contexts, independent read paths, snapshots, and
typed unavailable states. They must not wait on long-running model generation,
repo scans, release checks, queue worker critical sections, or orchestrator
processing locks. Mutations return asynchronous command ids or proposal ids and
publish status over the authenticated event stream.

Local-admin auth is mandatory for the TanStack control plane. Anonymous
dashboard operation is not supported. The gateway is the only trusted auth
boundary, and every dashboard route, API, event stream, feedback action, roster
proposal, model proposal, and command action is protected.

GitHub-derived metrics and mutation proposals are optional integrations. DORA is
computed only from configured deployment workflow runs and shows missing-auth,
missing-remote, missing-config, insufficient-history, permission, and rate-limit
states honestly. Prompt, roster, guardrail, schedule, and model changes flow
through draft code-host proposals or local patch previews instead of direct
frontend writes to source files.

### Pipeline flow graph: simple layered layout

V1 uses a server-side layered layout algorithm (roles grouped by trigger depth: scheduled roles in layer 1, event-triggered in layer 2, etc.) rendered as SVG. Not a full Sugiyama DAG layout. This is good enough for pipelines with 11-15 roles and implementable in a day.

Full auto-layout (dagre or similar) is a v2 improvement if custom role topologies need better rendering.

### Visual theme tokens

The web dashboard uses a neutral operations theme with semantic CSS tokens for background, surface, border, text, primary, accent, success, warning, and danger colors. Chart configuration reads those same tokens where possible, so refreshed dashboard styling is embedded consistently across HTML, CSS, and Chart.js surfaces instead of preserving hard-coded legacy palette values.

## Dashboard Pages

### Legacy/current Pages

1. **Pipeline Flow (home):** Live DAG of roles with state, scores, trust levels, next trigger. SSE updates node state on job start/complete.
2. **Orchestration:** Live orchestration mode, role topology, dispatch status, and recent Orchestrator decisions so dispatch-mode repos are not represented as a static linear pipeline.
3. **Role Health:** Per-role detail — outcome history, accuracy trend (30d chart with evolution annotations), context usage, guardrail violations.
4. **Throughput:** System metrics — job stats, inference performance (tokens/sec, GPU util, latency percentiles), GitHub API usage, pipeline output counts.
5. **Debug:** Job timeline (segmented bars), execution trace viewer (live and replay), webhook delivery log, error log.
6. **Evolution History:** Self-improvement timeline with before/after score analysis, guardrail inventory with staleness flags.

### Planned TanStack Control-Plane Pages

1. **Overview:** Active agents, owner, duration, active work, issues, telemetry,
   token usage, models, quality score links, and GitHub-derived DORA states.
2. **Active Work:** Current ticket or task details rendered from the repo
   artifact with BDD scenarios, acceptance criteria, evidence, blockers, files,
   owner, and next action.
3. **Preview:** Adaptive preview provider for web, mobile, API, cloud,
   distributed-system, library, and CLI work, with artifact fallback.
4. **Feedback:** Anchored or work-item-level next-turn feedback mailbox.
5. **Agent Roster:** Agent prompts, domains, modes, tools, guardrails, triggers,
   trust, metrics, model usage, and code-host proposal actions.
6. **Available Models:** Offline and cloud-hosted models, provider health,
   eligibility, benchmark evidence, usage, unavailable states, and override or
   registry-change proposals.

### AD-027: Interactive control surface (CLI + dashboard)

Operators need to pause, restart, scan, stop, and force-run individual roles without killing the process. Two control surfaces share identical backend methods on `Server`:

**CLI (terminal):** Interactive TTYs use a full-screen ANSI dashboard (`internal/ui/dashboard_tty.go`) plus the raw terminal key listener (`internal/ui/keylistener.go`). The dashboard redraws current state, repo, web dashboard URL, durable command log path, active jobs, current role/model, active phase with phase age, turn and tool counts, recent events, blocker summaries, and control hints. Non-streaming model calls appear as `waiting for model response` so a slow local first response is distinguishable from a stuck tool or completed idle state. Keys: `p` (pause/resume), `r` (warm restart), `s` (re-scan), `q` (graceful stop), `h` (help). Activated automatically during `mars-harness serve` and `mars-harness start`; `mars-harness run` uses the same dashboard without key bindings beyond Ctrl+C cancellation.

**CLI debug mode:** `run`, `start`, and `serve` accept `--debug` to restore verbose inline trace/log streaming. `run --trace` remains as a compatibility alias for debug-style trace detail. All three commands write slog output to `~/.mars-harness/traces/logs/YYYYMMDD-HHMMSS-<command>.log` unless `--log-file` is supplied. Non-TTY output falls back to concise plain progress and never enters alternate-screen mode.

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
| GET | `/api/orchestration` | Live orchestration mode and role topology per repo |
| GET | `/api/orchestration/decisions` | Recent dispatch decisions and stop reasons |

**Design choices:**
- Pause stops claiming, not killing. Running jobs complete. Prevents data loss.
- Warm restart reloads manifests and triggers within the same process. Does not rebuild the binary.
- Dashboard stop is accepted by the HTTP handler, then routed through the
  server's main loop before `Server.Stop` shuts down the dashboard HTTP server.
  This avoids deadlocking the stop request against its own active connection
  while preserving the normal `start`/`serve` shutdown path and command-level
  error reporting.
- `q` key cancels the signal context, triggering the same graceful shutdown path as Ctrl+C.

## Discoveries

- The `pipeline.html` template uses its own full HTML structure (not `base.html`), so the control bar had to be added directly to the template sidebar rather than relying on template inheritance. Both templates should be reconciled eventually (tech debt).
- Raw terminal mode on macOS uses `TIOCGETA`/`TIOCSETA` from `golang.org/x/sys/unix`. The key listener gracefully degrades (logs a warning and disables itself) if the process is not attached to a TTY.
- 2026-05-19 live `demo-123` dogfood showed `POST /api/stop` could stop
  workers, scheduler, inference, and sleep prevention but return `dashboard
  shutdown: context deadline exceeded` because the dashboard server was asked
  to shut down from inside the active dashboard request handler. Stop now uses
  a buffered stop request consumed by the server loop so the handler can return
  success before dashboard shutdown begins.
- 2026-05-23 live `demo-6` validation showed a `cto-weekly` job could look
  stalled while it was inside a non-streaming local model call: the terminal
  only reported `inference ready` with zero tools. The terminal dashboard now
  tracks a job phase and phase age, including `waiting for model response`, so
  operators can tell slow generation apart from an actual lack of runtime
  progress.
