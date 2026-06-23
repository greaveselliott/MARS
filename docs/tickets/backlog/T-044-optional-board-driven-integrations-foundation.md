---
id: T-044
title: Add optional board-driven integrations foundation
priority: high
complexity: medium
work_type: feature
bdd_scenarios: [F-013-S001]
end_to_end_evidence: required
evidence_links: []
verified_by: ""
owner: "foundation-maintainer"
last_attempt: "2026-06-23"
blocker: "Focused Plan 1 tests pass, but broad gates and clean-project installed-binary validation are blocked by local environment constraints; see docs/validation/reports/2026-06-23-example-target-project-optionality-foundation.md."
blocked_by: []
trace_id: ""
next_action: "Implement the optional integrations loader, generated example, flow-profile runtime gates, schedule rebuild/suppression, effective-tool hook, and profile visibility without enabling JIRA/Figma/PR behavior."
dedupe_key: "public-example"
source: docs/exec-plans/active/current-operating-plan.md
created: 2026-06-23
depends_on: []
---

# T-044: Add Optional Board-Driven Integrations Foundation

## Context

The Example Target Project Ways Of Working program needs a default-off configuration substrate before JIRA, prioritisation, gateway routing, Figma, or PR delivery can ship. Current Mars Harness behavior is CEO-led, schedule-driven, GitHub-ingress, and strict-trunk by default; this ticket must preserve that behavior byte-identically when `.harness/integrations.yaml` is absent.

## Requirements

- Add an optional `.harness/integrations.yaml` loader in `internal/integrations`.
- Missing config returns version `1`, effective `ceo-led`, and all sections disabled.
- Empty or unknown `flow_profile` fails safe to `ceo-led`; only exact `board-driven` enables board-driven gates.
- Add section structs for JIRA ingestion, Figma design sources, and delivery settings, but do not implement those integrations yet.
- Generate `.harness/integrations.example.yaml` during init/upgrade without writing `.harness/integrations.yaml`.
- Reload integrations config during startup and warm restart.
- Replace scheduler registrations on warm restart so stale schedules do not remain in memory.
- Under `board-driven`, suppress schedules only for `ceo`, `coo`, `head-of-strategy`, and `cto-weekly`.
- Compute effective role tool allowlists late in executor; no-config and `ceo-led` must stay byte-identical.
- Surface active flow profile in logs and status/dashboard APIs.

## Acceptance Criteria

- `go test ./internal/integrations` covers missing config, defaults, board-driven parsing, unknown profile fail-safe, unknown fields, and example parsing.
- Serve/scheduler tests prove warm restart rebuilds schedules and board-driven suppresses only planning schedules.
- Executor tests prove no-config effective tool lists are unchanged and future integration tools are appended only when registered and gated.
- Scanner tests prove init/upgrade write `.harness/integrations.example.yaml` and preserve user-owned files.
- No JIRA route, poller, prioritisation, Figma tool, PR tool, or model gateway behavior is implemented by this ticket.
- A clean-project validation report records installed-binary evidence or a concrete blocker/replay command.

## Evidence

- PASS: `git diff --check`
- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/integrations`
- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/scheduler`
- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/serve -run 'TestEffectiveToolAllowlist|TestServer_registerCronSchedules'`
- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/scanner -run 'TestInit_success|TestUpgrade_preservesUserConfiguredManifestAndPrompts'`
- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/docsync`
- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/docsconsistency`
- BLOCKED: `GOCACHE=<validation-root> go test ./...`
- BLOCKED: `GOCACHE=<validation-root> make check`
- PASS: `GOCACHE=<validation-root> go run ./cmd/mars-harness release notes --repo . --bump auto`
- BLOCKED: `GOCACHE=<validation-root> go run ./cmd/mars-harness release backfill-notes --repo . --check`
- PASS: `git push origin main`
- PASS: `git push origin v0.63.0`
- PASS: `GOCACHE=<validation-root> go run ./cmd/mars-harness release publish-assets --repo . --version v0.63.0 --upload auto`
- PASS: `GOCACHE=<validation-root> go run ./cmd/mars-harness release verify-assets --dist dist/releases --version v0.63.0`
- PASS: `gh release view v0.63.0 --repo greaveselliott/mars-harness --json tagName,name,url,isDraft,isPrerelease`
- Report: `docs/validation/reports/2026-06-23-example-target-project-optionality-foundation.md`
