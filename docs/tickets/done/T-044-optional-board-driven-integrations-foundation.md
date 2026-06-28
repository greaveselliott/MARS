---
id: T-044
title: Add optional board-driven integrations foundation
priority: high
complexity: medium
work_type: feature
bdd_scenarios: [F-013-S001]
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-06-23-example-target-project-optionality-foundation.md
verified_by: "Codex foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-06-23"
blocker: none
blocked_by: []
trace_id: ""
next_action: "Plan 1 is complete; start Plan 2 from T-045 only."
dedupe_key: "public-example"
source: docs/exec-plans/active/current-operating-plan.md
created: 2026-06-23
depends_on: []
---

# T-044: Add Optional Board-Driven Integrations Foundation

## Context

The Example Target Project Ways Of Working program needed a default-off configuration substrate
before JIRA, prioritisation, gateway routing, Figma, or PR delivery could ship.
Current MARS behavior is CEO-led, schedule-driven, GitHub-ingress, and
strict-trunk by default; this ticket preserves that behavior when
`.harness/integrations.yaml` is absent.

## Requirements

- Add an optional `.harness/integrations.yaml` loader in `internal/integrations`.
- Missing config returns version `1`, effective `ceo-led`, and all sections disabled.
- Empty or unknown `flow_profile` fails safe to `ceo-led`; only exact `board-driven` enables board-driven gates.
- Add section structs for JIRA ingestion, Figma design sources, and delivery settings without implementing those integrations yet.
- Generate `.harness/integrations.example.yaml` during init/upgrade without writing `.harness/integrations.yaml`.
- Reload integrations config during startup and warm restart.
- Replace scheduler registrations on warm restart so stale schedules do not remain in memory.
- Under `board-driven`, suppress schedules only for `ceo`, `coo`, `head-of-strategy`, and `cto-weekly`.
- Compute effective role tool allowlists late in executor; no-config and `ceo-led` stay byte-identical.
- Surface active flow profile in logs and status/dashboard APIs.

## Acceptance Criteria

- [x] `go test ./internal/integrations` covers missing config, defaults, board-driven parsing, unknown profile fail-safe, unknown fields, and example parsing.
- [x] Serve/scheduler tests prove warm restart rebuilds schedules and board-driven suppresses only planning schedules.
- [x] Executor tests prove no-config effective tool lists are unchanged and future integration tools are appended only when registered and gated.
- [x] Scanner tests prove init/upgrade write `.harness/integrations.example.yaml` and preserve user-owned files.
- [x] No JIRA route, poller, prioritisation, Figma tool, PR tool, or model gateway behavior is implemented by this ticket.
- [x] A clean-project validation report records installed-binary evidence and replay commands.

## Evidence

- PASS: `git diff --check`
- PASS: `go test -count=1 ./cmd/mars -run 'TestStartCommand'`
- PASS: `go test -count=1 ./internal/serve`
- PASS: `go test -count=1 ./internal/codeintel ./internal/scoring ./internal/personas`
- PASS: `go test -count=1 ./internal/integrations ./internal/scheduler ./internal/docsync ./internal/docsconsistency`
- PASS: `GOCACHE=<validation-root> go test ./...`
- PASS: `GOCACHE=<validation-root> make check`
- PASS: `GOCACHE=<validation-root> go run ./cmd/mars release backfill-notes --repo . --check`
- PASS: `/path/to/local-redacted auth github check`
- PASS: `make install`
- PASS: installed-binary `validation agent-smoke` for `static-web-ticket` and `go-api-ticket` through `http://127.0.0.1:18654/v1`.
- PASS: installed-binary `start` replay for static browser and Go API targets generated `.harness/integrations.example.yaml`, did not write `.harness/integrations.yaml`, loaded `flow_profile="ceo-led"`, replaced 8 schedules, completed CEO -> COO -> CTO -> Engineer, and recorded product evidence.
- Report: `docs/validation/reports/2026-06-23-example-target-project-optionality-foundation.md`

## Completion Notes

Plan 1 validation blockers were foundation-owned validation infrastructure and
live-loop issues, not optionality regressions. The closure slice repaired model
preflight leakage in tests, deterministic fixture drift, release marker drift,
coverage ratchet gaps, the static no-package-manager ticket path, and the Go
product-source repair path. Plan 2 must start from T-045 and must not assume any
JIRA/Figma/PR/frontier behavior exists yet.
