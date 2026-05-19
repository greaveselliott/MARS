---
id: T-008
title: Make dashboard stop exit the start process cleanly
priority: medium
complexity: small
work_type: intervention-debt
bdd_scenarios: ["F-010-S003", "F-010-S007"]
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md"]
verified_by: "go test ./internal/serve -run 'TestServer_(dashboardStopEndpointStopsStart|startStop)'; go test ./internal/dashboard -run 'TestDashboard_(stopEndpoint|controlEndpoints_methodNotAllowed|controlEndpoints_nilCallbacks)'; go test ./internal/docsconsistency ./internal/docsync; live stop replay at <validation-root> returned HTTP 200 {\"ok\":true} and the start process exited cleanly"
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done."
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "runtime_shutdown"
  severity: "medium"
source: 2026-05-19 demo-123 live lifecycle run
created: 2026-05-19
depends_on: []
---

# T-008: Make dashboard stop exit the start process cleanly

## Context

During bounded live demo-123 runs, POST /api/stop stopped workers, schedulers, inference servers, and sleep prevention, but the HTTP response returned dashboard shutdown: context deadline exceeded and the start process remained alive until killed manually.

## Requirements

- Make the stop endpoint complete cleanly after graceful shutdown.
- Ensure start exits once workers, scheduler, inference, and dashboard shutdown are complete.
- Preserve useful error reporting when shutdown cannot complete.

## Acceptance Criteria

- [x] POST /api/stop returns success for a clean idle start process.
- [x] The start process exits without manual kill after stop.
- [x] A regression test covers dashboard/webhook stop behavior without leaving a process running.

## Completion Notes

- `Server.RequestStop` now routes dashboard stop through the server loop instead
  of calling `Server.Stop` from inside the active dashboard request handler.
- `TestServer_dashboardStopEndpointStopsStart` boots real webhook and dashboard
  listeners on local ephemeral ports, posts to `/api/stop`, and verifies
  `Start` returns cleanly.
- The live stop replay against `demo-123-stop-check2` returned HTTP `200` with
  `{"ok":true}` and no manual kill was needed.
