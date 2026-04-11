---
id: MH-004
title: Implement context assembly engine
priority: high
complexity: medium
source: delivery-schedule M1.4
created: 2026-04-11
---

# MH-004: Implement context assembly engine

## Context

The context assembler builds the system prompt for each job from: role prompt + scoped guardrails + knowledge routes + trigger context. It must respect context budgets (tenet 9) to keep local model inference efficient.

Reference: [docs/design-docs/context-efficiency.md](../../design-docs/context-efficiency.md), [docs/design-docs/agent-runtime.md](../../design-docs/agent-runtime.md) (AD-006)

## Requirements

Implement `internal/context/assembler.go`:

- Assemble system prompt from components in priority order:
  1. Role prompt (from `.harness/roles/<role>.md`)
  2. In-scope guardrails (filtered by `scope` field matching current role)
  3. Knowledge routes (from `.harness/knowledge-routes.yaml`, injected as "when X, read Y")
  4. Trigger context (ticket content, CI failure excerpt, PR diff — role-specific)
  5. Repo summary (directory tree)
- Token estimation for each section
- Context budget enforcement: if total exceeds budget, truncate lower-priority sections (repo summary first, then trigger context, then knowledge routes)
- Clear section headers between components (the model sees "## ROLE", "## GUARDRAILS", "## KNOWLEDGE ROUTES", etc.)

## Affected Files

- `internal/context/assembler.go`, `assembler_test.go`
- `internal/context/types.go` (assembly config, section priorities)

## Acceptance Criteria

### Functional (happy path)
- [ ] Assembler produces a system prompt containing role prompt, guardrails, knowledge routes, and trigger context
- [ ] Only in-scope guardrails are included (scope filtering works)
- [ ] Knowledge routes are formatted as "when working on X, read Y" guidance
- [ ] Section headers are present and clear

### Edge cases and negative paths
- [ ] Context budget exceeded → lower-priority sections truncated, role prompt never truncated
- [ ] Missing role prompt file → descriptive error
- [ ] No guardrails in scope → section omitted (not empty section header)
- [ ] No knowledge routes configured → section omitted
- [ ] Empty trigger context → section omitted

### Non-goals
- [ ] Guardrail YAML parsing (MH-014 in M7)
- [ ] Knowledge route YAML parsing (can use a simple struct for now)
- [ ] Token counting accuracy beyond estimation (exact counting is a future improvement)

### Observability, docs, and regressions
- [ ] Test verifies scoping: engineer role gets engineering guardrails, not security guardrails
- [ ] Test verifies budget enforcement: with a low budget, sections are truncated in correct order
- [ ] context-efficiency.md updated with discoveries

## Notes

AD-006: additive assembly with section headers, not template-based. Concatenate sections in order with clear markdown headers.
