---
id: T-015
title: Normalize live tool argument drift and static app evidence
priority: high
complexity: medium
work_type: bug
bdd_scenarios:
  - F-005-S003
  - F-001-S009
end_to_end_evidence: not_applicable
evidence_links:
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-12-tool-argument-and-matrix-replay--2026-05-20
verified_by: "go test ./internal/tools ./internal/scanner ./internal/docsync; go test ./internal/docsconsistency ./internal/docsync; go test ./...; focused run-12 docsync replay checked 3 src files instead of 0"
owner: "Codex"
last_attempt: "2026-05-20: completed generic list-string normalization plus mechanical deployed app-root DocSync auditing; static server-root turn waste remains for T-013."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Continue factory pace work through T-013/T-011 and the representative validation matrix; avoid treating the static Space Invaders canary as the only product archetype."
dedupe_key: "public-example"
source: docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-11-release-blocked-terminal-replay--2026-05-20
created: 2026-05-20
depends_on: []
---

# T-015: Normalize live tool argument drift and static app evidence

## Context
Run 11 reached terminal release-blocked dispatch, but the live trace showed tool payload drift and static app evidence waste still consuming turns. Release Manager emitted mars_harness_cli args as a string list, shell_exec argv as a Python-style list string, Engineer left static assets pointing at a non-existent duplicate F-001 feature path, and Dogfood spent turns discovering src/index.html.

## Requirements
- mars_harness_cli accepts list-shaped args emitted as strings.
- shell_exec accepts Python-style list strings for argv without shell fallback.
- Generated static-app guidance reuses existing feature contracts for MarsDocSync pointers.
- Dogfood starts static evidence from the detected app directory instead of repo-root retries.

## Acceptance Criteria
- Unit tests cover mars_harness_cli args strings and shell_exec Python-list argv strings.
- Unit tests cover path-list string normalization for workspace_hygiene, git_diff, and git_commit.
- Generated target guidance tells agents not to invent duplicate F-001 feature files for DocSync metadata.
- Generated Dogfood guidance prefers src/ when src/index.html is the static entrypoint.
- A clean demo replay or focused evidence confirms release tooling no longer falls back to stale PATH because of malformed args.

## Run 12 Update

Run 12 showed this should not become a Space Invaders-only optimization. The
useful generic fix is list-string normalization across built-in tool fields:
`mars_harness_cli.args`, `shell_exec.argv`, `workspace_hygiene.paths`, and git
path filters. Static app guidance improved canonical `MarsDocSync` paths, but
static server-root selection and static asset docsync auditing needed a
mechanical follow-up rather than more prompt-only tuning. This ticket completed
the static asset docsync follow-up; static server-root selection remains in
T-013.

## Completion Evidence

- Generic list-shaped argument normalization was completed for
  `mars_harness_cli.args`, `shell_exec.argv`, `workspace_hygiene.paths`,
  `git_diff.paths`, and `git_commit.paths`.
- `internal/docsync` now audits common deployed app roots (`src/`, `app/`,
  `pages/`, `public/`, `web/`, and `static/`) and parses compact inline static
  metadata such as
  `/* MarsDocSync: ["docs/features/F-001-product-walking-skeleton.md"] */`.
- Focused run-12 replay against
  `<validation-root>` changed
  `docsync_audit` from `checked 0 files` to `checked 3 files, findings 1`; the
  remaining `src/index.html` missing-metadata finding is legitimate target
  evidence rather than an invisible-audit failure.
- Generated target docs now explain compact inline static metadata and deployed
  app-root auditing, and scanner tests assert the mirrored doctrine.
- Remaining static server-root turn waste is intentionally left to T-013 so the
  next optimization is judged against the representative project matrix rather
  than only the Space Invaders static canary.
