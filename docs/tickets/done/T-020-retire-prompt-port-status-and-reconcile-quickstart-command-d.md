---
id: T-020
title: Retire prompt-port-status and reconcile quickstart command drift
priority: medium
complexity: small
work_type: docs
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - "AD-277 in docs/design-docs/self-improvement.md, indexed in docs/design-docs/index.md"
  - "mars run --help / setup --help / version output compared against docs/quickstart.md on 2026-06-11"
  - "go test ./internal/docsconsistency/... ./internal/docsync/..."
verified_by: "foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; prompt-port-status.md retired under AD-277 and quickstart drift fixed."
source: Foundation improvement plan Phase 1 WS-A (provisional T-020)
created: 2026-06-11
depends_on: []
---

# T-020: Retire prompt-port-status and reconcile quickstart command drift

## Context

The 2026-06-11 foundation review found docs/prompt-port-status.md is an MH-025-era snapshot: every checklist box is checked and the prompt port completed in 2026-04. The role inventory now lives in examples/roles/ and docs/roles/ROLES.md, so the snapshot is orphaned. docs/quickstart.md must also be reconciled against the actual current CLI command surface in cmd/mars.

Ownership classification: foundation-owned documentation hygiene.

## Requirements

- Retire docs/prompt-port-status.md; the completed MH-025 ticket remains the historical record.
- Compare docs/quickstart.md claims (commands, flags, defaults) against the current CLI and fix drift.
- Record the retirement decision as an AD indexed in docs/design-docs/index.md.

## Affected Files

- docs/prompt-port-status.md
- docs/quickstart.md
- docs/design-docs/index.md

## Acceptance Criteria

### Functional (happy path)
- [x] prompt-port-status.md is removed and no live doc links to it.
- [x] quickstart.md commands and flags match the current CLI help output (run/setup flags verified verbatim; version example corrected to the unprefixed `VERSION`-file form; install pin example made placeholder `vX.Y.Z`).

### Edge cases and negative paths
- [x] Done-ticket history that names the file is preserved unchanged (docs/tickets/done/MH-025-mars-prompt-port.md untouched).

### Non-goals
- Rewriting quickstart structure or adding new tutorial content.

### Observability, docs, and regressions
- [x] go test ./internal/docsconsistency/... ./internal/docsync/... passes (run 2026-06-11).
