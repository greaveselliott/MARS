---
id: T-071
title: Add configurable TypeScript monorepo DocSync coverage
priority: high
complexity: medium
work_type: feature
bdd_scenarios: ["F-019-S001"]
end_to_end_evidence: required
evidence_links: ["go test ./...", "go test -race ./internal/docsync ./internal/bundle", "go run ./cmd/mars docsync audit --repo .", "docs/validation/reports/2026-08-08-typescript-docsync-live-target.md"]
verified_by: "engineer plus deterministic full-suite, race, DocSync, and clean generated-target CLI/tool evidence"
owner: "engineer"
last_attempt: "2026-08-08"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Complete semantic commit and push codex/typescript-docsync for operator review."
dedupe_key: "docsync:typescript-monorepo-coverage"
source: operator request 2026-08-08 — Real-Time Chess MARS bootstrap prerequisite
created: 2026-08-08
depends_on: []
---

# T-071: Add configurable TypeScript monorepo DocSync coverage

## Context

The active plan requires MARS DocSync to govern modern TypeScript monorepos before the Real-Time Chess target is initialized. The existing auditor hard-codes Go/static roots and extensions, so TS/TSX workspaces escape the no-stale-documentation gate.

## Requirements

- Read optional docsync include_roots, include_extensions, and exclude_globs from .harness/manifest.yaml.
- Add safe built-in TypeScript/TSX monorepo defaults while preserving existing Go/static coverage.
- Validate configuration before traversal and keep CLI, mirrored tool, and disposition gates on one audit path.
- Mirror the defaults and doctrine into generated targets without overwriting existing manifests during upgrade.
- Update source and generated documentation, CLI reference, feature evidence, and MarsDocSync metadata.

## Affected Files

internal/docsync, internal/scanner/init.go, cmd/mars and mirrored tool tests, and owning operating-model docs.

## Acceptance Criteria

- [x] F-019-S001 passes for nested TS/TSX app, package, worker, and test fixtures.
- [x] Default generated/dependency/build paths are excluded and custom exclusions work.
- [x] Absolute paths, parent traversal, malformed extensions, and malformed globs fail before traversal.
- [x] CLI JSON/text and mirrored tool results agree.
- [x] Existing Go/static audits remain backward compatible.
- [x] Focused tests, full `go test ./...`, DocSync, docs consistency, and clean generated-target evidence pass.

## BDD Evidence

- Scenario: F-019-S001
- Focused packages: `go test ./internal/bundle ./internal/docsync ./internal/scanner ./cmd/mars ./internal/tools ./internal/docsconsistency ./internal/planhygiene`
- Full regression: `go test ./...`
- Race coverage: `go test -race ./internal/docsync ./internal/bundle`
- Source DocSync: `go run ./cmd/mars docsync audit --repo .`
- Live clean target: `docs/validation/reports/2026-08-08-typescript-docsync-live-target.md`
- Documentation reviewed as current: every `MarsDocSync` pointer on changed source plus public manifest/CLI references, generated target doctrine, F-019, active goal, plan, and ticket.

## Non-goals

Do not redesign MARS roles, orchestration, or the agent factory; do not add a Real-Time-Chess-specific auditor.
