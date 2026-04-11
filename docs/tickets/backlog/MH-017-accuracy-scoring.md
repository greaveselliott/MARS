---
id: MH-017
title: Outcome tracking and rolling accuracy score from GitHub signals
priority: high
complexity: medium
source: delivery-schedule M6
created: 2026-04-11
---

# MH-017: Accuracy scoring — PR outcomes, CI, reviews, noop detection, alerts

## Context

Trust and autonomy (M6) need grounded metrics, not vibes. Scores must derive from durable GitHub facts tied to harness-created PRs and jobs.

## Requirements

- Ingest GitHub events (merged, closed without merge, CI success/failure, required review approvals) keyed by `job_id` / PR number
- Scoring formula versioned in code (e.g. `v1`: merge +CI green +approval weights; penalties for CI red, closed, manual revert flags from MH-019 when available)
- Rolling window over last 20 completed harness jobs per repo (configurable)
- Noop detection: PR with zero non-comment events or only whitespace → score neutral bucket, not positive
- Value scoring: optional weight when user merges with edits (signals real value) vs straight merge
- Alert hooks: log + optional webhook when rolling score drops below threshold for N consecutive jobs

## Acceptance Criteria

### Functional (happy path)
- [ ] Synthetic fixture of 20 jobs produces expected numeric score for known outcomes
- [ ] Noop PR does not increase accuracy metric
- [ ] Alert fires when scripted drop crosses threshold

### Edge cases and negative paths
- [ ] Events arriving out-of-order reconcile to stable final state (merge after close ignored correctly)
- [ ] Missing optional signals (no reviews configured) do not NaN the score; documented defaults
- [ ] Fewer than 20 jobs: score computed on available sample with `sample_size` exposed in status

### Non-goals
- [ ] LLM-as-judge for code quality
- [ ] Cross-repo normalization across unrelated languages

### Observability, docs, and regressions
- [ ] Golden-file tests for scorer `v1`
- [ ] `/api/status` or dedicated `/api/metrics` exposes rolling score, window, formula version
- [ ] Design note: formula tuning process and backwards compatibility
