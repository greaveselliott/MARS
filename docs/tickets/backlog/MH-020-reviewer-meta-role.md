---
id: MH-020
title: Reviewer meta-role for failure analysis and evolution PRs to .harness
priority: medium
complexity: large
source: delivery-schedule M7
created: 2026-04-11
---

# MH-020: Reviewer meta-role — trace analysis, root-cause taxonomy, evolution PRs

## Context

When jobs fail, a dedicated meta-role should turn execution traces into concrete improvements: prompts, guardrails, triggers, policy, context packing, or model limits. It must be powerful but tightly safety-bounded.

## Requirements

- Reviewer consumes MH-005 traces + tool errors + GitHub CI logs (links)
- Classify root cause into fixed taxonomy: `prompt`, `guardrail`, `trigger`, `policy`, `context`, `model_gap`, `unknown`
- Open PRs only under `.harness/` (manifest, prompts, guardrails, knowledge routes) with clear description template
- Model policy: strongest configured model + highest context budget for this role only
- Safety: Reviewer cannot modify its own system prompt files; changes go through normal PR review by humans
- Rate limit: at most one evolution PR per role per day; **auto-disable if the last 3 Reviewer evolutions each worsened MH-017 rolling score** (thresholds configurable)

## Acceptance Criteria

### Functional (happy path)
- [ ] Failing fixture job produces Reviewer plan + single `.harness/` PR with expected file touched
- [ ] Classification JSON stored alongside job outcome for analytics

### Edge cases and negative paths
- [ ] Attempt to edit Reviewer prompt path is rejected at tool layer with explicit reason
- [ ] Second Reviewer run same day returns no-op with “budget exhausted” audit entry
- [ ] If the **last 3 evolutions each worsened** MH-017 rolling score, Reviewer auto-disable flag prevents new runs until CLI clear

### Non-goals
- [ ] Auto-merge of Reviewer PRs
- [ ] Editing application source outside `.harness/`

### Observability, docs, and regressions
- [ ] End-to-end test with stub LLM returning canned classification + patch
- [ ] Metrics: reviewer_runs, prs_opened, blocked_attempts
- [ ] Runbook: how humans should triage Reviewer PRs quickly
