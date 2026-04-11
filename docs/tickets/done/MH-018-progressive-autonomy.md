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
- **Runs-to-promote (schedule):** trust advances by accumulated successful evidence, not by a time-boxed trial that **expires and reverts**.
  - **`observer` → `contributor`:** after **N trial runs** (default **5**); each trial run is an audited job toward the promotion counter (runs-to-promote, not time-boxed expiry)
  - **`contributor` → `autonomous`:** after MH-017 accuracy **score ≥ configured threshold** sustained over **20+ terminal outcomes** in the scoring window
- Automatic promotion/demotion evaluated after each completed job; persisted in SQLite
- Enforcement: `observer` cannot open or update PRs (read-only tools + comments allowed); violations blocked in tool layer with clear error
- CLI override: `mars-harness trust set <role> <level> --reason` audited

## Acceptance Criteria

### Functional (happy path)
- [ ] Role at `observer` completes a diagnostic job without creating a PR
- [ ] Scripted high scores promote `contributor` → `autonomous` per threshold **over 20+ outcomes** (MH-017)
- [ ] **Runs-to-promote:** default **5** trial runs advance `observer` → `contributor`; no automatic downgrade solely because a trial “expired”

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
