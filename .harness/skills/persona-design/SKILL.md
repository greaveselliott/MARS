---
name: persona-design
scope: all
---

# Persona Design Skill

Use this when creating or revising an agent persona, role user manual, role
prompt manual, ownership boundary, feedback contract, or orchestrator handoff
expectation.

## Workflow

1. Read `docs/design-docs/harness-operating-model.md`, `docs/roles/ROLES.md`,
   and any existing `docs/roles/personas/<role>.md` manual.
2. Decide the persona scope: `universal`, `foundation`, or `deployed`.
3. Define the persona explicitly: modus operandi, priorities, owns, does not
   own, best feedback format, feedback I need, feedback I give, stop
   conditions, and orchestrator handoff.
4. Use `persona_create` for repo-local persona scaffolding when available.
5. For new foundation-default personas, add or update the canonical entry in
   `internal/personas` and regenerate/check docs and prompts.

## Stop Conditions

- Stop and route to CEO when the persona would change strategy ownership.
- Stop and route to Orchestrator when the persona would enter the default
  delivery loop without an explicit design decision.
- Stop and record a blocker when ownership, feedback, or stop conditions are
  not explicit.
