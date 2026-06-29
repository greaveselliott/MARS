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

## No Stale Documentation

All documentation is live. When code is written or materially changed, the code
file carries a top-of-file `MarsDocSync` comment block with a `docs:` array
listing the feature contracts, design docs, product specs, README surfaces,
ticket guidance, or other durable docs associated with that behavior. The listed
docs must be reviewed and updated in the same change, or the ticket, plan,
review, or commit evidence must state why they remain current.

The source-wide gate is:

```bash
mars docsync audit --repo .
mars tools run docsync_audit --repo . --args-json '{}'
```

The architecture and universal operating model for this process live in
[../design-docs/documentation-sync-architecture.md](../design-docs/documentation-sync-architecture.md).

CLI changes have an additional foundational operating model: keep the
`mars_cli` reference, repo-shortcut map, generated doctrine, and
affected skills synchronized using
[../design-docs/cli-tool-skill-sync.md](../design-docs/cli-tool-skill-sync.md).

## Contract Rules

- Feature contracts come after the active exec plan: the plan names the feature
  and scenario schedule before tickets or delivery begin.
- BDD defines the full feature before implementation.
- Business logic is documented step by step under the feature contract, not
  only in tickets, code comments, or release notes.
- Code files that implement or constrain behavior carry `MarsDocSync` metadata
  with a `docs:` array pointing at the docs that must stay current with that
  behavior.
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
| F-004 | [Target Harness Lifecycle](F-004-target-harness-lifecycle.md) | partially-passing | Init, upgrade, update check, doctor drift reporting, eject kill switch |
| F-005 | [Agent Execution Runtime](F-005-agent-execution-runtime.md) | partially-passing | Context assembly, tool calls, traces, budgets, run command |
| F-006 | [Queue And Orchestration](F-006-queue-and-orchestration.md) | partially-passing | Register, start, serve, scheduler, chains, dispatch, recovery |
| F-007 | [Guardrails And Safety](F-007-guardrails-and-safety.md) | partially-passing | Hard guardrails, secret scanning, sandbox, blast radius, emergency stop |
| F-008 | [Scoring Trust And Quality](F-008-scoring-trust-quality.md) | partially-passing | Role scores, trust levels, quality score export, intervention debt |
| F-009 | [Release And Update Lifecycle](F-009-release-update-lifecycle.md) | partially-passing | Versioning, changelog, historical backfills, GitHub releases, release assets, update tool |
| F-010 | [Dashboard And Control Plane](F-010-dashboard-control-plane.md) | partially-passing | Dashboard pages, status APIs, orchestration state, pause/resume/restart/scan/run-role controls |
| F-011 | [Optional GitHub Integration](F-011-optional-github-integration.md) | partially-passing | GitHub App setup, webhooks, statuses, comments |
| F-012 | [Self-Improvement Loop](F-012-self-improvement-loop.md) | partially-passing | Telemetry, intervention detection, skills, tool creation, bounded evolution |
| F-013 | [Board-Driven Integrations](F-013-board-driven-integrations.md) | active | Optional JIRA board intake, board prioritisation, Figma context, human-reviewed delivery, and traceability |
| F-014 | [MARS Rename](F-014-mars-rename.md) | active | Product identity, CLI, module path, compatibility aliases, release/update names, generated doctrine, and docs rewrite |
| F-015 | [Documentation Site Information Architecture](F-015-documentation-site-information-architecture.md) | active | Reader-first public docs, trust-building homepage, documentation map, governance guide, adoption lanes, canonical source labels |
| F-016 | [Foundation Orchestrator Planning Doctrine](F-016-foundation-provider-planning-doctrine.md) | passing | Source-only Orchestrator planning chain consumed by Claude, Codex, Copilot, Cursor, Windsurf, and other AI clients building the foundation harness |

## Historical Feature Audit

Historical release entries and semantic `feat:` commits map to the feature
contracts above. The audit keeps broad shipped capability surfaces under BDD
instead of letting old release notes become the only description of behavior.

| Historical Surface | Owning Contract | Coverage Note |
| --- | --- | --- |
| Local LLM client, model download, llama.cpp supervision, model routing, performance tuning, benchmark-backed model workflows | F-003 | Covered by setup, download, supervision, routing, missing-model, evaluation, and provider scenarios. |
| Agent loop, context assembly, tool execution, traces, budgets, run command, universal CLI/MCP tools | F-005 | Covered by role-scoped context, multi-turn tool loop, containment, traces, budgets, run, and tool-surface scenarios. |
| Queue, scheduler, trigger chains, start/serve, recovery, intervention-debt priority, native Orchestrator survey, dispatch-mode organization | F-006 | Covered by lifecycle, trigger, recovery, priority, survey, and dispatch scenarios. |
| Init, upgrade, update check, doctor target health, auto-scaffold, mirrored doctrine, target eject kill switch | F-004 | Eject coverage added as F-004-S008 during the historical release audit. |
| Dashboard pages, live event stream, controls, emergency stop, throughput, orchestration state | F-010 | Orchestration-state coverage added as F-010-S008; API decision-history evidence remains planned. |
| Semantic release notes, release assets, updater, update harness/check, detailed release narratives, historical backfills | F-009 | Historical backfill coverage added as F-009-S009 with release package and CLI evidence. |
| Scoring, trust, quality exports, intervention-debt creation and sparse-evidence handling | F-008 | Covered by scoring, trust, export, intervention-debt, release/quality classification, and evidence scenarios. |
| Telemetry triage, self-improvement, skills, tool creation, bounded evolution | F-012 | Covered by classification, triage, goal/ticket creation, intervention detection, bounded evolution, skills, and mirroring scenarios. |

## Scenario Evidence Discipline

Evidence entries should be executable where possible:

- `go test ./internal/... -run TestName`
- `mars <command> --dry-run`
- `mars doctor --repo <path> --json`
- dogfood run traces or release verification commands
- docs-consistency checks when the feature is about repository-owned contracts

When a scenario is not yet covered, record the missing evidence in the feature
contract and keep the status below `passing`.
