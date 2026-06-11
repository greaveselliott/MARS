---
id: T-026
title: Make the release pipeline self-verifying with a release audit command
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-009-S017"]
end_to_end_evidence: not_applicable
evidence_links: ["AD-282 in docs/design-docs/release-versioning.md, indexed in docs/design-docs/index.md", "go test ./internal/release -run TestAudit", "go test ./cmd/mars-harness -run TestPrintReleaseAuditResult", "CLI sync gates: go test ./cmd/mars-harness ./internal/tools -run TestMarsHarnessCLI; go test ./internal/scanner -run TestInit_success; go test ./internal/docsconsistency/...", "First live run 2026-06-11 found a real defect: v0.45.1 is missing_release (tag with no GitHub Release object); recorded in docs/validation/release-blockers.md"]
verified_by: "command"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; release audit shipped, CLI mirror and release-publication skill synced, AD-282 recorded."
source: Foundation improvement plan Phase 2 WS-C (provisional T-026)
created: 2026-06-11
depends_on: []
---

# T-026: Make the release pipeline self-verifying with a release audit command

## Context

The 2026-06-11 foundation review found that release verification stops at the most recent version: `release verify-assets` checks one tag at a time, so notes-only releases (tag and changelog published, binary assets never mirrored) accumulate invisibly. The recorded GitHub Actions billing blocker makes this drift class likely, and the original Phase 2 scope (a scheduled CI workflow) no longer fits because trunk retired GitHub Actions workflows in favor of local delivery gates and local-first asset publication (AD-059, AD-078).

Ownership classification: foundation-owned release-flow behavior. The audit is a read-only mirror check; no factory runtime behavior changes, so no canary replay is required.

**Scope adaptation:** drafted as a Release workflow extension plus a scheduled notes-only-detection workflow; lands instead as a `mars-harness release audit` CLI command wired into the release-publication skill, which is where post-publication verification now lives.

## Requirements

- `mars-harness release audit --repo <path> [--github-repo owner/name] [--limit n] [--json]` audits the newest local vX.Y.Z tags (default 10) against GitHub Releases.
- Classify findings: `missing_release` (tag with no release object) and `notes_only` (release object missing required platform binaries or checksums.txt), reusing the selfupdate asset expectations.
- Every finding names the exact `release publish-assets --repo . --version vX.Y.Z --upload github` backfill command; findings exit non-zero for scripted callers.
- Graceful degradation: missing tags or an unreachable GitHub API report a skip reason and exit zero (the mirror is optional infrastructure); the operator records the blocker.
- CLI tool/skill sync: mars_harness_cli reference + repo-shortcut map updated; release-publication skill runs the audit after every publication.

## Affected Files

- internal/release/audit.go (new)
- internal/release/audit_test.go (new)
- internal/selfupdate/check.go (exported SetGitHubAPIHeaders)
- cmd/mars-harness/main.go (release audit subcommand)
- cmd/mars-harness/main_test.go
- internal/tools/mars_harness_cli.go (reference + repo-shortcut map)
- .harness/skills/release-publication/SKILL.md
- docs/design-docs/release-versioning.md (AD-282), docs/design-docs/index.md
- docs/features/F-009-release-update-lifecycle.md (F-009-S017)

## BDD Evidence

- Scenario IDs: F-009-S017 (documented in the feature contract; ticket filed as enabler because earlier F-009 scenarios predate ticket scenario coverage tracking and the create policy requires earliest-uncovered-scenario ordering)
- Evidence links: see frontmatter
- Verified by: command

## Acceptance Criteria

### Functional (happy path)
- [x] Audit of complete mirrors reports checked tags and exits zero.
- [x] Notes-only and missing releases are classified with missing asset names and exact backfill commands.

### Edge cases and negative paths
- [x] Findings exit non-zero; skips (no tags, API unavailable) exit zero with the recorded reason.
- [x] Non-semver tags are ignored; --limit bounds the audit newest-first.

### Non-goals
- Scheduled/automated execution infrastructure (GitHub Actions retired); the release-publication skill is the recurring trigger.
- Automatic backfill execution; the audit reports the command, the operator or agent runs it.

### Observability, docs, and regressions
- [x] AD-282 recorded and indexed; F-009-S017 documented with evidence.
- [x] CLI sync tests pass (cmd, tools, scanner, docsconsistency).
