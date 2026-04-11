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
- Scoring formula versioned in code: **`v1`** is a simple ratio (each terminal outcome in exactly one bucket); **`v2` (deferred):** weighted scoring (CI/review/value weights, explicit penalties, MH-019 revert flags when available) — not in v1.
  - **v1 accuracy** = `(merged + passed) / (merged + passed + closed + failed + noop)`
- **30-day rolling window** per repo (configurable anchor, e.g. calendar UTC); only completed harness outcomes whose terminal timestamp falls in the window count toward the ratio
- Noop detection: PR with zero non-comment events or only whitespace → **noop** bucket in the denominator; does not count as merged/passed positive signal
- Value signals (e.g. user merge with edits vs straight merge): tracked for analytics only in v1; **weighted value scoring is deferred to v2** alongside the weighted accuracy formula
- Alert hooks: log + optional webhook when rolling score drops below threshold for N consecutive evaluations in-window (configurable)

## Acceptance Criteria

### Functional (happy path)
- [ ] Synthetic fixture spanning the **30-day rolling window** produces the expected v1 ratio for known outcome mix
- [ ] Noop PR lands in **noop** and does not increase the numerator; denominator reflects noop correctly
- [ ] Alert fires when scripted drop crosses threshold

### Edge cases and negative paths
- [ ] Events arriving out-of-order reconcile to stable final state (merge after close ignored correctly)
- [ ] Missing optional signals (no reviews configured) do not NaN the score; documented defaults
- [ ] Sparse data in window: score computed on available terminal outcomes with `sample_size` and window bounds exposed in status

### Non-goals
- [ ] LLM-as-judge for code quality
- [ ] Cross-repo normalization across unrelated languages

### Observability, docs, and regressions
- [ ] Golden-file tests for scorer `v1` (ratio + window boundaries)
- [ ] `/api/metrics` exposes rolling score, window, formula version (see MH-016 `/healthz` for process health)
- [ ] Design note: formula tuning process and backwards compatibility (v2 weighted scoring when promoted from backlog)
