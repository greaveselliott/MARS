---
id: T-015
title: Normalize live tool argument drift and static app evidence
priority: high
complexity: medium
work_type: bug
bdd_scenarios:
  - F-005-S003
end_to_end_evidence: not_applicable
evidence_links:
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-12-tool-argument-and-matrix-replay--2026-05-20
verified_by: "partial: go test ./internal/tools ./internal/scanner; run12 evidence showed broader workspace_hygiene.paths drift"
owner: "Codex"
last_attempt: "2026-05-20: run12 confirmed canonical static DocSync paths improved, but path-list drift is generic and static serving/docsync still needs mechanical follow-up."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Finish generic list-string normalization, then split remaining static app server-root and static asset docsync mechanics into the next focused slice."
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
static server-root selection and static asset docsync auditing still need a
mechanical follow-up rather than more prompt-only tuning.
