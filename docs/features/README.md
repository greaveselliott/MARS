# BDD Feature Contracts

BDD feature contracts define feature completeness. Walking skeleton is the
implementation strategy used to make scenarios pass through the thinnest real
end-to-end path.

V1 uses Markdown Given/When/Then. Do not introduce a custom Gherkin parser until
there is evidence that Markdown plus Go integration/E2E tests is not enough.

## Business Logic Is First-Class BDD

All business logic belongs in `docs/features/`. A feature contract is not just a
completion checklist; it is the durable step-by-step description of how the
product behaves. Document product rules, workflow branches, state transitions,
validations, permissions, scoring or trust rules, queue routing, release
classification, and user-visible outcomes here before or alongside
implementation.

Tickets may scope a slice of work, and code may include local comments, but the
feature contract is the source of truth for business behavior. If an engineer
discovers that required behavior is missing or stale in `docs/features/`, the
correct fix is to update the feature contract or return to planning before
expanding implementation.

## Contract Rules

- Feature contracts come after the active exec plan: the plan names the feature
  and scenario schedule before tickets or delivery begin.
- BDD defines the full feature before implementation.
- Business logic is documented step by step under the feature contract, not
  only in tickets, code comments, or release notes.
- The schedule is the ordered list of failing BDD scenarios or scenario groups.
- Tickets implement only the current failing scenario or scenario group.
- No feature ships until in-scope scenarios pass or are explicitly descoped.
- Every feature has integration, E2E, dogfood, command, or docs-consistency
  evidence mapped to scenario IDs.
- Unit tests may support deterministic helpers, but they do not replace the
  product-level scenario evidence for user-visible behavior.
- Enabler work can complete without feature evidence, but it must not claim
  shipped feature value.
- A broad contract may be `partially-passing` when some scenarios have evidence
  and the remaining scenarios are still open product promises.

## Required Fields

- `Feature ID`
- `Goals`
- `Status`: draft, active, partially-passing, passing, superseded
- `Owner`
- `Business Logic`
- `Step-By-Step Behavior`
- `Scenario Schedule`
- `Out of Scope`
- `Descoped Scenarios`
- `Evidence`

## Status Semantics

- `draft`: contract exists for planned work, but implementation evidence is not
  yet expected.
- `active`: current or near-current work is executing against the scenario
  schedule.
- `partially-passing`: at least one scenario has evidence, and at least one
  scenario remains unproven or intentionally pending.
- `passing`: all in-scope scenarios have evidence or are explicitly descoped.
- `superseded`: retained for lineage; do not schedule new tickets from it.

## Feature Catalog

| ID | Contract | Status | Product Surface |
| --- | --- | --- | --- |
| F-001 | [Delivery Operating Model](F-001-delivery-operating-model.md) | passing | Goals, BDD, plans, tickets, evidence, target mirroring |
| F-002 | [Zero-Config Shell PATH](F-002-zero-config-shell-path.md) | passing | Source install, setup, self-update PATH setup |
| F-003 | [Local Inference Lifecycle](F-003-local-inference-lifecycle.md) | partially-passing | Hardware profiles, model downloads, llama.cpp supervision, model evaluation |
| F-004 | [Target Harness Lifecycle](F-004-target-harness-lifecycle.md) | partially-passing | Init, upgrade, update check, doctor drift reporting |
| F-005 | [Agent Execution Runtime](F-005-agent-execution-runtime.md) | partially-passing | Context assembly, tool calls, traces, budgets, run command |
| F-006 | [Queue And Orchestration](F-006-queue-and-orchestration.md) | partially-passing | Register, start, serve, scheduler, chains, recovery |
| F-007 | [Guardrails And Safety](F-007-guardrails-and-safety.md) | partially-passing | Hard guardrails, secret scanning, sandbox, blast radius, emergency stop |
| F-008 | [Scoring Trust And Quality](F-008-scoring-trust-quality.md) | partially-passing | Role scores, trust levels, quality score export, intervention debt |
| F-009 | [Release And Update Lifecycle](F-009-release-update-lifecycle.md) | partially-passing | Versioning, changelog, GitHub releases, release assets, update tool |
| F-010 | [Dashboard And Control Plane](F-010-dashboard-control-plane.md) | partially-passing | Dashboard pages, status APIs, pause/resume/restart/scan/run-role controls |
| F-011 | [Optional GitHub Integration](F-011-optional-github-integration.md) | partially-passing | GitHub App setup, webhooks, statuses, comments |
| F-012 | [Self-Improvement Loop](F-012-self-improvement-loop.md) | partially-passing | Telemetry, intervention detection, skills, tool creation, bounded evolution |

## Scenario Evidence Discipline

Evidence entries should be executable where possible:

- `go test ./internal/... -run TestName`
- `mars-harness <command> --dry-run`
- `mars-harness doctor --repo <path> --json`
- dogfood run traces or release verification commands
- docs-consistency checks when the feature is about repository-owned contracts

When a scenario is not yet covered, record the missing evidence in the feature
contract and keep the status below `passing`.
