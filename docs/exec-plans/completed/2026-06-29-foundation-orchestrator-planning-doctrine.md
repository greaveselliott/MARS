# Completed P0 Exec Plan: Foundation Orchestrator Planning Doctrine

**Status:** Completed
**Priority:** P0
**Depends On:** None
**Blocks:** durable foundation planning state, consistent foundation work across AI clients, external-tool consumption of the Orchestrator model
**Related Tickets:** T-054
**Goals:** G-FOUNDATION-PLANNING-001, G-001, G-002
**BDD Feature:** F-016-foundation-provider-planning-doctrine.md
**Related Feature Contracts:** F-001, F-016
**Hypothesis:** A source-only rule that names the MARS Orchestrator goal -> exec plan -> BDD feature -> tickets -> evidence chain will stop foundation feature work from drifting into chat-only or provider-specific plans.
**Success Evidence:** `AGENTS.md`, the foundation maintainer role packet, `docs/design-docs/foundation-operating-model.md`, `docs/features/F-016-foundation-provider-planning-doctrine.md`, `docs/goals/active.md`, and T-054 all require external tools to consume the same foundation planning chain when Claude, Codex, Copilot, Cursor, Windsurf, and other clients work on `mars`.
**Falsification Evidence:** A provider can plan or build a non-trivial foundation feature from only chat state, issue text, branch text, review checklist, or vendor-native task state; feature tickets can exist without an active goal, active exec plan, and BDD feature contract; vendor adapters carry independent doctrine; or deployed target harnesses are told to consume this source-only rule without a separate mirroring decision.
**Scenario Schedule:** F-016-S001, F-016-S002, F-016-S003, F-016-S004
**Current Failing Scenario:** None; F-016-S001 through F-016-S004 passed on 2026-06-29.
**Walking Skeleton Slice:** Record the doctrine in the active goal, active exec plan, BDD feature contract, foundation operating model, foundation maintainer role packet, and top-level AI-client guidance before creating T-054 completion evidence.
**Learning Or MVP Outcome:** Any capable AI coding provider can continue foundation feature planning from repo artifacts, not from a previous chat or client-specific task surface.
**Created:** 2026-06-29
**Completed:** 2026-06-29
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, QA, and Release Manager responsibilities
**Source:** Operator request to make external AI clients consume the existing MARS doctrine when planning or building foundation harness features.

## Primary Outcome

Make the foundational operating model explicit: the MARS Orchestrator manages
planning, building, validation, deployment, and release flow for foundation
harness work, and external clients such as Claude, Codex, Copilot, Cursor,
Windsurf, Gemini, OpenCode, and Kiro consume that model when they build `mars`.

## Primary Pass Gate

- Active goal `G-FOUNDATION-PLANNING-001` exists.
- The active exec plan scheduled F-016 and T-054.
- `docs/features/F-016-foundation-provider-planning-doctrine.md` defines the source-only external-client consumption behavior.
- `docs/design-docs/foundation-operating-model.md` records AD-308.
- `AGENTS.md` and the foundation maintainer role packet direct providers to goal -> exec plan -> BDD feature -> tickets -> evidence.
- T-054 records completion evidence.
- Docs checks and full Go tests pass.
- The foundation-maintainer dry run consumes the doctrine.
- Generated target surfaces do not contain the source-only AD-308/F-016 wording.

## Scenario Schedule

| Scenario | Ticket | Outcome | Status |
| --- | --- | --- | --- |
| F-016-S001 | T-054 | Planning state aligns the active goal, active exec plan, feature contract, and ticket. | Passed |
| F-016-S002 | T-054 | AI client guidance names the MARS Orchestrator planning chain for external providers working on `mars`. | Passed |
| F-016-S003 | T-054 | Foundation operating doctrine defines the source-only sequence before feature implementation. | Passed |
| F-016-S004 | T-054 | Trivial and blocked-work exceptions avoid parallel provider-specific doctrine. | Passed |

## Validation Gates And Evidence

- PASS: `git diff --check`.
- PASS: `mars docsync audit --repo .`.
- PASS: `go test ./internal/docsconsistency ./internal/docsync`.
- PASS: `go test ./...` with full test permissions.
- PASS: `mars run foundation-maintainer --repo . --dry-run --no-init`.
- PASS: generated-target leakage grep found the doctrine only in source foundation docs.
- T-054 was created through `ticket_create`, updated with evidence, and moved to `docs/tickets/done/`.

## Residual Risks

- This is source-only doctrine and does not change runtime enforcement or generated target scaffolding.
- Future provider adapters must remain thin pointers to canonical foundation guidance.
