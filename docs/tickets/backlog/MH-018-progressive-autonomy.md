---
id: MH-018
title: Trust levels, thresholds, trial mode, promotion/demotion, enforcement
priority: medium
complexity: medium
source: delivery-schedule M6
created: 2026-04-11
---

# MH-018: Progressive autonomy — observer / contributor / autonomous

## Context

Roles should gain capability only when evidence supports it. Trust levels gate PR creation, direct pushes, and emergency-stop overrides while keeping manual CLI escape hatches for operators.

## Requirements

- Trust enum per role: `observer`, `contributor`, `autonomous` with manifest defaults
- Configurable numeric thresholds tying to MH-017 (e.g. promote to `contributor` after score ≥ X for Y jobs)
- Trial mode: allow exactly N runs at `contributor` before auto-revert to `observer` unless promoted by metrics
- Automatic promotion/demotion evaluated after each completed job; persisted in SQLite
- Enforcement: `observer` cannot open or update PRs (read-only tools + comments allowed); violations blocked in tool layer with clear error
- CLI override: `mars-harness trust set <role> <level> --reason` audited

## Acceptance Criteria

### Functional (happy path)
- [ ] Role at `observer` completes a diagnostic job without creating a PR
- [ ] Scripted high scores promote `contributor` → `autonomous` per thresholds
- [ ] Trial mode expires and downgrades unless criteria met

### Edge cases and negative paths
- [ ] Manual CLI override immediately affects enforcement without waiting for next job
- [ ] Conflicting manifest trust vs DB: precedence rules documented and tested (recommend DB wins, manifest is default seed)
- [ ] Demotion mid-queue: new jobs pick up reduced trust; running job policy documented (finish vs cancel)

### Non-goals
- [ ] Per-user GitHub identity trust (installation-scoped only)
- [ ] Fine-grained OAuth scopes beyond MH-010

### Observability, docs, and regressions
- [ ] Audit table entries for every promotion/demotion/override
- [ ] Integration test covering blocked PR tool at observer
- [ ] User-facing table of capabilities × trust level in docs
