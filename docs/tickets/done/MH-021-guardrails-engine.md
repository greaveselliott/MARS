---
id: MH-021
title: YAML guardrails engine with scope, advisory/hard modes, overrides, staleness
priority: medium
complexity: medium
source: delivery-schedule M7
created: 2026-04-11
---

# MH-021: Guardrails engine — parse, scope by role, advisory and hard validation

## Context

Guardrails translate org policy into mechanical checks. M7 separates “nudge the model” from “block the tool” and makes exceptions visible and time-bounded.

## Requirements

- Parse guardrail YAML: rules with `name`, `scope` (global vs role list), `mode` (`advisory` | `hard`), matchers (`regex`, `file_pattern`, optional `path_prefix`)
- Advisory: inject compact reminder into prompt context when matcher hits pre-tool-call
- Hard: reject tool I/O or file edits when matcher hits; return structured violation to agent loop
- Override mechanism: manifest-declared break-glass tokens or operator CLI flag with mandatory audit log line (who/when/rule_id)
- Staleness: rules unused for 90 days flagged in `doctor` / status output; optional warn-only mode

## Acceptance Criteria

### Functional (happy path)
- [x] Role-scoped rule applies only to named role; global applies to all
- [x] Advisory rule surfaces once per turn max to avoid prompt spam (dedupe policy tested)
- [x] Hard rule blocks disallowed path write with actionable error referencing rule name

### Edge cases and negative paths
- [x] Invalid regex in YAML fails bundle load with path and column
- [x] Override on hard rule logs rule id, principal, expires if TTL set (optional TTL field)
- [x] Competing rules: highest severity wins; tie-break deterministic by rule order in file

### Non-goals
- Full Semgrep or AST policy language
- Remote policy fetch from network

### Observability, docs, and regressions
- [x] Unit tests: regex edge cases, file_pattern glob semantics, override audit
- [x] Golden bundle negative tests in CI
- [x] Docs: examples for secrets and destructive command patterns
