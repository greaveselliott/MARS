---
id: MH-023
title: mars-harness init scaffolding and repo scanner for starter backlog
priority: medium
complexity: medium
source: delivery-schedule M8
created: 2026-04-11
---

# MH-023: `mars-harness init` — .harness/, tickets, exec-plans, AGENTS.md + repo scanner

## Context

New repos need opinionated structure aligned with the harness mental model. A scanner proposes high-value starter tickets without spamming mature codebases.

## Requirements

- `init` writes `.harness/` skeleton: `manifest.yaml` at bundle root; generated scaffold includes `roles.yaml`, guardrails stub, knowledge routes placeholder
- Create `.harness/tickets/` with seed tickets if missing; mirror exec-plans folders; append or create `AGENTS.md` section for harness conventions (non-destructive merge)
- Scanner: detect missing tests (packages without `*_test.go` / test dirs), high-count TODO/FIXME, packages lacking type coverage signals (heuristic: build tags / empty interfaces) — tune heuristics per language with Go-first implementation
- Smart skipping: if coverage heuristics pass thresholds, emit summary “no backlog generated” plus reason codes
- All writes support `--dry-run` listing paths

## Acceptance Criteria

### Functional (happy path)
- [x] Empty repo `init` produces valid bundle passing MH-009 reader validation
- [x] Scanner on repo with untested package generates markdown tickets under `.harness/tickets/` with YAML frontmatter consistent with repo ticket format
- [x] Well-covered fixture repo generates zero or minimal tickets with documented skip reasons

### Edge cases and negative paths
- [x] Re-run on dirty tree: refuse or require `--force` with backup of overwritten files (policy documented)
- [x] Monorepo with many packages completes under time budget via concurrency limits
- [x] Non-git directory fails with actionable message

### Non-goals
- LLM-generated ticket prose beyond templates
- Automatic fixing of findings

### Observability, docs, and regressions
- [x] Snapshot tests for generated tree layout
- [x] Scanner unit tests on synthetic tree fixtures
- [x] Docs: how to customize scanner thresholds in manifest
