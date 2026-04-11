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

## Discoveries

(Record during implementation.)
