# F-016: Foundation Orchestrator Planning Doctrine

- Feature ID: F-016
- Goals: G-FOUNDATION-PLANNING-001, G-001, G-002
- Status: passing
- Owner: foundation-maintainer

## Business Logic

The MARS Orchestrator owns the operating model for planning, building,
validating, deploying, and releasing foundation harness work. Capable AI coding
clients working on MARS foundation source consume that model before they plan or
build a non-trivial feature. Claude, Codex, Copilot, Cursor, Windsurf, Gemini,
OpenCode, Kiro, and future providers may use their native planning surfaces as
scratch space, but durable work state must live in MARS repo artifacts.

The feature-work chain is:

1. Goal in `docs/goals/active.md`.
2. Active exec plan in `docs/exec-plans/active/current-operating-plan.md`.
3. BDD feature contract in `docs/features/F-NNN-*.md`.
4. Tickets created through `ticket_create`.
5. Implementation evidence recorded in the ticket, plan, and relevant docs.

Existing goals, plans, feature contracts, and tickets are updated when they
already own the work. New artifacts are created only when the requested
foundation feature does not map to an existing owner. This contract is
source-only and is not generated target doctrine.

## Step-By-Step Behavior

1. An operator asks any AI coding client to plan or build a non-trivial
   foundation harness feature.
2. The client reads `AGENTS.md`, the foundation maintainer role packet, the
   active plan, relevant feature contracts, and the foundation operating model.
3. The client classifies the work as foundation-owned, deployed-owned, mirrored
   doctrine, evidence-only, or mixed/unclear.
4. The client updates or confirms the active goal.
5. The client updates the active exec plan with the Primary Outcome Contract,
   hypothesis, evidence, scenario schedule, current failing scenario, walking
   skeleton slice, and validation gates.
6. The client creates or updates the BDD feature contract with business logic
   and step-by-step scenarios.
7. The client creates tickets through `ticket_create` for the current failing
   scenario or bounded scenario group.
8. The client implements only ticket-backed work and records validation
   evidence before claiming completion.
9. If trunk freshness, dirty state, unclear ownership, missing evidence, or
   unavailable tooling blocks the chain, the client records the blocker instead
   of creating a parallel plan.

## Scenario Schedule

1. F-016-S001 - Provider-neutral feature work has aligned goal, exec plan,
   feature contract, and ticket state.
2. F-016-S002 - AI client guidance names the MARS planning chain for Claude,
   Codex, Copilot, Cursor, Windsurf, and other providers.
3. F-016-S003 - Foundation operating doctrine defines the Orchestrator-managed
   goal -> exec plan -> feature -> tickets -> evidence sequence before feature
   implementation.
4. F-016-S004 - Exceptions for trivial or blocked work avoid creating parallel
   provider-specific doctrine.

## Scenarios

### F-016-S001: Planning State Alignment

Given a non-trivial foundation feature request reaches any capable AI coding client
When the client inspects planning state
Then `docs/goals/active.md`, `docs/exec-plans/active/current-operating-plan.md`,
the matching BDD feature contract, and the ticket state point to the same
feature outcome.

### F-016-S002: Provider-Neutral Client Guidance

Given Claude, Codex, Copilot, Cursor, Windsurf, Gemini, OpenCode, Kiro, or
another AI coding client reads the repo instructions
When it prepares foundation feature planning or feature delivery
Then it is directed to consume the canonical foundation Orchestrator doctrine
and not to create provider-specific independent planning rules.

### F-016-S003: Feature Implementation Follows Planning

Given a foundation feature requires implementation
When the client starts source changes
Then the active goal, active exec plan, BDD feature contract, and ticket for the
current scenario already exist or have been updated.

### F-016-S004: Exceptions Stay Explicit

Given the work is a tiny typo, simple command answer, throwaway experiment, or
blocked by dirty state, remote trunk, unclear ownership, or missing evidence
When the client does not create the full planning chain
Then it records why the chain is not needed or why the work is blocked, without
creating parallel provider-specific doctrine.

## Out of Scope

- Runtime orchestration changes.
- Generated target harness mirroring changes.
- New provider adapter files.
- New ticket tool behavior.

## Descoped Scenarios

None.

## Evidence

- F-016-S001: `docs/goals/active.md`,
  `docs/exec-plans/active/current-operating-plan.md`, this contract, and T-054.
- F-016-S002: `AGENTS.md` and
  `docs/roles/personas/foundation-maintainer.md`.
- F-016-S003: `docs/design-docs/foundation-operating-model.md` AD-308.
- F-016-S004: `docs/design-docs/foundation-operating-model.md` AD-308
  exception language and `AGENTS.md` Working Discipline.
