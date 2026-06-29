---
id: T-054
title: Add foundation Orchestrator planning doctrine
priority: high
complexity: small
work_type: docs
bdd_scenarios: ["F-016-S001", "F-016-S002", "F-016-S003", "F-016-S004"]
end_to_end_evidence: not_applicable
evidence_links:
  - "PASS: git diff --check"
  - "PASS: mars docsync audit --repo ."
  - "PASS: go test ./internal/docsconsistency ./internal/docsync"
  - "PASS: go test ./..."
  - "PASS: mars run foundation-maintainer --repo . --dry-run --no-init includes AD-308 foundation Orchestrator planning doctrine"
  - "PASS: AD-308/F-016 foundation-planning wording is absent from internal/scanner, examples, and docs/generated"
verified_by: "foundation-maintainer; DocSync; docs consistency; full Go suite; foundation-maintainer dry-run"
owner: "foundation-maintainer"
last_attempt: "2026-06-29"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Complete; monitor future provider work for planning-chain bypasses."
dedupe_key: "foundation-provider-neutral-planning-doctrine"
source: user_chat provider-neutral foundation planning doctrine 2026-06-29
created: 2026-06-29
depends_on: []
---

# T-054: Add Foundation Orchestrator Planning Doctrine

## Context
Foundation harness work across Claude, Codex, Copilot, Cursor, Windsurf, and other AI coding clients must consume the same MARS Orchestrator planning doctrine. The durable planning chain is goal -> active exec plan -> BDD feature contract -> tickets -> implementation evidence.

## Requirements
- Add or update active goal, active exec plan, BDD feature contract, and ticket state for the doctrine change.
- Update the foundation operating model with the source-only Orchestrator planning chain for external clients.
- Update top-level and foundation-maintainer guidance so providers do not use chat-only or provider-native plans as the system of record.
- Keep vendor adapters thin and avoid independent provider-specific doctrine.

## Affected Files
- AGENTS.md
- docs/roles/personas/foundation-maintainer.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/index.md
- docs/features/README.md
- docs/features/F-016-foundation-provider-planning-doctrine.md
- docs/goals/active.md
- docs/exec-plans/active/current-operating-plan.md
- docs/exec-plans/completed/2026-06-29-documentation-site-ia-rebuild.md

## BDD Evidence
- Scenario IDs: F-016-S001, F-016-S002, F-016-S003, F-016-S004
- Evidence links: `git diff --check`; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`; `mars run foundation-maintainer --repo . --dry-run --no-init`; no AD-308/F-016 generated-target leakage under `internal/scanner`, `examples`, or `docs/generated`
- Verified by: foundation-maintainer; DocSync; docs consistency; full Go suite; foundation-maintainer dry-run

## Acceptance Criteria

### Functional
- [x] Active goal, active plan, feature contract, and ticket state point to the same foundation Orchestrator planning outcome.
- [x] Provider-facing guidance names the MARS planning chain for Claude, Codex, Copilot, Cursor, Windsurf, and other clients building `mars`.
- [x] Foundation operating doctrine records the required source-only goal -> exec plan -> feature -> tickets -> evidence sequence.
- [x] Trivial and blocked-work exceptions are explicit.

### Non-goals
- Runtime enforcement changes.
- Generated target harness mirroring changes.
- New provider adapter files.

### Observability, docs, and regressions
- [x] git diff --check passes.
- [x] mars docsync audit --repo . passes.
- [x] go test ./internal/docsconsistency ./internal/docsync passes.
- [x] go test ./... passes.
- [x] foundation-maintainer dry-run consumes AD-308 and the source-only
  Orchestrator planning chain.
- [x] Generated-target sources do not contain AD-308/F-016 foundation-planning
  wording.
