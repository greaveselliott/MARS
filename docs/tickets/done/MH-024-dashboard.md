---
id: MH-024
title: Built-in dashboard at 9090 with htmx Chart.js and SSE
priority: medium
complexity: large
source: delivery-schedule M9
created: 2026-04-11
---

# MH-024: Built-in dashboard — embedded UI in binary, five pages

## Context

Operators need visibility without standing up a separate frontend repo. M9 ships a small embedded server using htmx + Chart.js + SSE, assets compiled into the Go binary.

## Requirements

- Listen on `:9090` by default (configurable); authn via localhost-only bind + optional shared token header for remote bind
- Stack: server-rendered HTML fragments, htmx swaps, Chart.js charts fed by JSON endpoints, SSE stream for live job/webhook events
- Five pages: (1) pipeline flow diagram (layered SVG), (2) role health cards, (3) throughput metrics (jobs/hour, latency percentiles), (4) debug: job timeline + trace viewer + webhook log tail, (5) evolution history (Reviewer PRs, guardrail changes)
- Read-only against SQLite + in-memory ring buffers; no arbitrary code execution from UI

## Acceptance Criteria

### Functional (happy path)
- [x] Each page loads with seed data in dev mode; charts render in headless fetch tests (HTML contains expected canvas/svg markers)
- [x] SSE pushes new job state within one second of DB update in local benchmark
- [x] Trace viewer paginates large traces without OOM (streaming read)

### Edge cases and negative paths
- [x] Invalid token when required → 401 on all routes including static assets
- [x] DB locked briefly: UI shows degraded banner, retries with backoff
- [x] Missing optional modules (e.g. no Reviewer yet) page shows empty state, not 500

### Non-goals
- OAuth login for multi-user teams
- Full distributed tracing backend

### Observability, docs, and regressions
- [x] `go test` HTTP handler tests with `httptest` for each route
- [x] Bundle size budget documented; `//go:embed` asset checksum test
- [x] Screenshot or ascii wireframe in docs for each page
