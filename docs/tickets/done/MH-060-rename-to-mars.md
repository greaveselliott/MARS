---
id: MH-060
title: Rename Mars Harness to MARS
priority: critical
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-014-S001", "F-014-S002", "F-014-S003", "F-014-S004", "F-014-S005", "F-014-S006", "F-014-S007", "F-014-S008"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-014-mars-rename.md"]
verified_by: "GOCACHE=/private/tmp/mars-go-cache make check; go build -o /private/tmp/mars-rename-smoke ./cmd/mars; /private/tmp/mars-rename-smoke init --repo /private/tmp/mars-rename-target; /private/tmp/mars-rename-smoke docsync audit --repo ."
owner: "foundation-maintainer"
last_attempt: 2026-06-28
completed: 2026-06-28
blocker: none
blocked_by: []
trace_id: none
next_action: "Done. Broader lifecycle matrix replay is a separate validation follow-up if a lifecycle claim is needed."
dedupe_key: foundation:mars-rename
source: operator request 2026-06-28
created: 2026-06-28
depends_on: []
---

# MH-060: Rename Mars Harness to MARS

## Context

The repository used to be called `mars-harness` and the product was described
as Mars Harness. The operator renamed it to MARS and requested an exhaustive
tracked-reference migration. The CLI is lowercase `mars`; the Go module path is
`github.com/greaveselliott/mars`.

## BDD Scenario IDs

- F-014-S001
- F-014-S002
- F-014-S003
- F-014-S004
- F-014-S005
- F-014-S006
- F-014-S007
- F-014-S008

## Affected Docs/Code Areas

- `go.mod`
- `cmd/mars/`
- `internal/tools/`
- `internal/config/`
- `internal/shellpath/`
- `internal/selfupdate/`
- `internal/release/`
- `internal/scanner/`
- `internal/operatingmodel/`
- `internal/codeintel/`
- `internal/setup/`
- `internal/doctor/`
- `internal/githubauth/`
- `scripts/`
- `.cursor/skills/`
- `.harness/skills/`
- `README.md`, `AGENTS.md`, `ARCHITECTURE.md`, `CONTRIBUTING.md`
- `docs/features/`, `docs/design-docs/`, `docs/product-specs/`, `docs/tickets/`, `docs/validation/`, `CHANGELOG.md`

## Acceptance Criteria

- [x] Source module path is `github.com/greaveselliott/mars`.
- [x] CLI entrypoint and build/install paths use `cmd/mars` and binary `mars`.
- [x] Canonical product prose says MARS.
- [x] Canonical tool is `mars_cli`, with tested `mars_harness_cli` alias.
- [x] Canonical env vars are `MARS_*`, with tested legacy `MARS_HARNESS_*` fallbacks.
- [x] New runtime defaults write under `~/.mars`, with tested old-state fallback where compatibility matters.
- [x] Release/update package paths, repo names, assets, installer commands, and markers use MARS names, with old marker parsing retained.
- [x] Generated target guidance uses MARS names and reads old metadata where required.
- [x] Historical tracked docs are rewritten except explicit allowlist cases.
- [x] Final old-name audit has no unclassified hits.

## Evidence Requirements

- Targeted package tests for CLI/tool/config/shellpath/release/update/scanner/operatingmodel/codeintel/setup/doctor: complete.
- Broad `go test ./...` and `make check`: complete.
- `go build ./cmd/mars`: complete with `/private/tmp/mars-rename-smoke`.
- Generated target smoke with `mars init --repo <clean-temp-target>`: complete.
- Old-name grep and old-path grep with allowlist: complete for working tree and staged index.
- Release notes/backfill/assets evidence: complete with `v0.66.0`, local verification, and GitHub release verification.
