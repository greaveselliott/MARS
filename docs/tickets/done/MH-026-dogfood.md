---
id: MH-026
title: Dogfood harness on mars repo with Pipeline Fixer and QA roles
priority: medium
complexity: medium
source: delivery-schedule M10
created: 2026-04-11
---

# MH-026: Dogfood — harness manages its own development

## Context

Shipping confidence requires the tool to run its own CI hygiene and QA loops against this repository under `.harness/` configuration.

## Requirements

- Commit `.harness/` for `mars` including roles for Pipeline Fixer and QA aligned with MH-025 tranche 1
- Wire GitHub Actions or self-hosted check to invoke `mars run` / `serve` paths as appropriate (documented non-secret env)
- Define allowed blast radius (MH-015) suitable for public OSS: branch prefix, max lines, no-delete default on
- Run book: how maintainers triage harness-opened PRs vs human PRs

## Acceptance Criteria

### Functional (happy path)
- [x] Pipeline Fixer opens a draft PR on a controlled fixture failure (or scheduled noop test branch) end-to-end
- [x] QA role executes on schedule or `workflow_dispatch` without manual token paste in logs
- [x] At least one merged dogfood PR authored by the harness with human review

### Edge cases and negative paths
- [x] Fork PRs from contributors do not auto-run privileged steps (guard documented)
- [x] Secret redaction verified in CI logs for tokens

### Non-goals
- Full removal of human reviewers
- Auto-merge to default branch without checks green

### Observability, docs, and regressions
- [x] Badge or docs section linking to example harness PRs
- [x] Post-incident checklist if dogfood loop wedges queue (MH-013/MH-016 ops)
- [x] MH-017 scoreboard includes dogfood repo metrics separately flagged
