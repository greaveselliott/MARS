# Active P0 Exec Plan: Foundation Orchestrator Planning Doctrine

**Status:** Active
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
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, QA, and Release Manager responsibilities
**Source:** Operator request to make external AI clients consume the existing MARS doctrine when planning or building foundation harness features.

## Primary Outcome

Make the foundational operating model explicit: the MARS Orchestrator manages
planning, building, validation, deployment, and release flow for foundation
harness work, and external clients such as Claude, Codex, Copilot, Cursor,
Windsurf, Gemini, OpenCode, and Kiro consume that model when they build `mars`.

## Primary Pass Gate

- Active goal `G-FOUNDATION-PLANNING-001` exists.
- This active exec plan schedules F-016 and T-054.
- `docs/features/F-016-foundation-provider-planning-doctrine.md` defines the
  source-only external-client consumption behavior.
- `docs/design-docs/foundation-operating-model.md` records AD-308.
- `AGENTS.md` and the foundation maintainer role packet direct providers to
  goal -> exec plan -> BDD feature -> tickets -> evidence.
- T-054 records completion evidence.
- Docs checks and full Go tests pass.
- The foundation-maintainer dry run consumes the doctrine.
- Generated target surfaces do not contain the source-only AD-308/F-016 wording.

## Scenario Schedule

| Scenario | Ticket | Outcome | Status |
| --- | --- | --- | --- |
| F-016-S001 | T-054 | Planning state aligns the active goal, active exec plan, feature contract, and ticket. | Passed |
| F-016-S002 | T-054 | AI client guidance names the MARS Orchestrator planning chain for Claude, Codex, Copilot, Cursor, Windsurf, and other providers working on `mars`. | Passed |
| F-016-S003 | T-054 | Foundation operating doctrine defines the source-only sequence before feature implementation. | Passed |
| F-016-S004 | T-054 | Trivial and blocked-work exceptions avoid parallel provider-specific doctrine. | Passed |

## File Changes

| File | Action | Purpose |
| --- | --- | --- |
| `docs/goals/active.md` | Update | Add `G-FOUNDATION-PLANNING-001`. |
| `docs/exec-plans/active/current-operating-plan.md` | Update | Make foundation Orchestrator planning doctrine the current active plan. |
| `docs/exec-plans/completed/2026-06-29-documentation-site-ia-rebuild.md` | New | Preserve the completed docs IA plan snapshot. |
| `docs/features/F-016-foundation-provider-planning-doctrine.md` | New | Define the source-only external-client consumption behavior. |
| `docs/tickets/done/T-054-add-provider-neutral-planning-doctrine.md` | New | Record delivered docs/doctrine work. |
| `docs/design-docs/foundation-operating-model.md` | Update | Add AD-308. |
| `docs/design-docs/index.md` | Update | Index AD-308 and the updated foundation operating model summary. |
| `docs/features/README.md` | Update | Add F-016 to the feature catalog. |
| `AGENTS.md` | Update | Add foundation Orchestrator planning chain to top-level client guidance and working discipline. |
| `docs/roles/personas/foundation-maintainer.md` | Update | Add AD-308 to foundation-maintainer responsibilities. |

## Validation Gates

- `git diff --check`
- `mars docsync audit --repo .`
- `go test ./internal/docsconsistency ./internal/docsync`
- `go test ./...`
- `mars run foundation-maintainer --repo . --dry-run --no-init`
- Generated-target leakage grep for AD-308/F-016 wording

## Current Evidence

- Remote trunk fetched and local `main` fast-forwarded to `origin/main`
  `e8dc152` on 2026-06-29 before commit.
- Completed docs IA active plan snapshot preserved at
  `docs/exec-plans/completed/2026-06-29-documentation-site-ia-rebuild.md`.
- AD-308 records the foundation Orchestrator feature-work chain.
- F-016 records the BDD behavior and evidence mapping.
- T-054 was created through `ticket_create`, updated with evidence, and moved
  to `docs/tickets/done/`.
- PASS: `git diff --check`.
- PASS: `mars docsync audit --repo .`.
- PASS: `go test ./internal/docsconsistency ./internal/docsync`.
- PASS: `go test ./...` with full test permissions.
- PASS: `mars run foundation-maintainer --repo . --dry-run --no-init`
  assembles a prompt that includes AD-308, the source-only foundation harness
  scope, and the goal -> exec plan -> BDD feature -> `ticket_create` sequence.
- PASS: `rg` found AD-308/F-016 foundation-planning wording in source
  foundation docs only, with no matches under `internal/scanner`, `examples`,
  or `docs/generated`.

## Residual Risks

- This change is source-only doctrine for building the foundation harness. It
  does not change runtime enforcement or generated target scaffolding.
- Vendor-specific adapters stay thin by design. If a future provider requires a
  dedicated adapter file, that adapter must still point back to `AGENTS.md` and
  the foundation maintainer role packet rather than carrying independent
  doctrine.
