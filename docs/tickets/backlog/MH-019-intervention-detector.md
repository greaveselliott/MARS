---
id: MH-019
title: Monitor GitHub for human interventions on harness PRs
priority: medium
complexity: medium
source: delivery-schedule M7
created: 2026-04-11
---

# MH-019: Intervention detector — classify human actions, store in SQLite

## Context

Self-improvement loops need labels: when humans fix agent output, that signal should be structured, not inferred from chat. M7 focuses on GitHub-side evidence on harness-owned PRs.

## Requirements

- Track harness PRs by label or branch prefix + author bot id
- Classify interventions: `clear` (commits that edit agent-touched files, revert commits), `ambiguous` (merged without edits, closed without merge), `non_intervention` (comments only, review comments without code change)
- Persist rows: `pr_number`, `repo_id`, `classification`, `evidence_json`, `detected_at`, `job_id` link when resolvable
- Poll or webhook-driven incremental updates; idempotent upsert by `(repo_id, pr_number, evidence_hash)`

## Acceptance Criteria

### Functional (happy path)
- [ ] Fixture PR with human follow-up commit classified `clear` with file list in evidence
- [ ] Comment-only activity stored as `non_intervention` without duplicating rows on redelivery

### Edge cases and negative paths
- [ ] Squash-merge with edited message still detected as code change when diff non-empty
- [ ] Force-push rewriting history: latest state wins; audit trail notes rewrite
- [ ] Non-harness PRs ignored cheaply (prefix/label filter first)

### Non-goals
- [ ] IDE-local edits before push (invisible to GitHub)
- [ ] Sentiment analysis of review text

### Observability, docs, and regressions
- [ ] Table migration + unit tests with JSON fixtures from GitHub API payloads
- [ ] Dashboard hook optional; at minimum `/api/status` exposes last intervention summary per repo
- [ ] Privacy note: store minimal diff metadata, not full secret-bearing files
