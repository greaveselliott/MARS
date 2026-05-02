# Mars Meta-Harness Relevance Audit

**Status:** Reference
**Created:** 2026-05-02
**Owner:** Mars Harness
**Source repo:** `../mars`
**Purpose:** Identify which Mars rules, docs, prompts, and operating decisions are relevant to Mars Harness as it works to supersede Mars.

## Coverage Note

This audit covers Mars's governing material: root and template `AGENTS.md`, all `.cursor/rules/*.mdc`, `docs/automations/`, active design docs, product specs, reference docs, standing trackers, quality score, ticket guidance, and the completed plans that explain major automation changes.

The full historical ticket set and completed-plan archive were inventoried and pattern-scanned. Individual old tickets are treated as evidence of recurring failure modes, not as governing rules to port wholesale.

## Relevance Levels

| Level | Meaning |
| --- | --- |
| Direct | Bring the concept into Mars Harness with only strict-trunk wording changes. |
| Translate | Keep the operating idea, but rebuild it in Harness-native primitives. |
| Reference | Useful background or pattern evidence; do not make it a default Harness feature yet. |
| Exclude | Mars-specific or incompatible with strict trunk. Do not port except as optional compatibility notes. |

## Executive Cut

The most relevant Mars material for Mars Harness is the meta-harness layer:

1. Six domain-shaped roles with payload modes.
2. Repo-visible automation registry.
3. Active-plan hygiene enforced by prompts and CI.
4. Intervention debt as the mandatory result of human rescue.
5. Quality score as a repo artifact and event source.
6. Deterministic remediation before LLM repair.
7. Orchestrator as a non-agent surveyor/remediator.
8. Dogfood matrix with dedupe, caps, and escalation semantics.
9. Generated project `AGENTS.md` and docs as the handoff interface.
10. Conversation-as-system-record discipline.

The least relevant Mars material is the PR and Cursor-specific delivery model:

- branch-only workflow
- PR creation and merge as the default delivery path
- Cursor prompt sync runbooks
- Cursor webhook secrets as product setup
- npm/changeset-specific publishing mechanics
- Mars SaaS generator rules as generic Harness defaults

Those may inform optional compatibility integrations, but they should not become Mars Harness defaults.

## Rules

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `.cursor/rules/documentation-discipline.mdc` | Direct | Port active-plan hygiene thresholds: no old TBD goals, close stale snapshots, close plans with only trailing verification, require movement within 30/60 days. Implement in `doctor`, CI, and Planner/Orchestrator behavior. |
| `.cursor/rules/intervention-to-automation.mdc` | Direct | Already aligned with Harness tenets. Strengthen into mechanical intervention-debt ticket creation from failed jobs, guardrail blocks, human follow-up, and stuck in-progress work. |
| `.cursor/rules/knowledge-base.mdc` | Translate | Keep the routing-table pattern. Replace Mars package/generator routes with Harness routes for agent runtime, inference, scoring, trust, guardrails, queue, dashboard, scanner, and docs consistency. Generate target repo knowledge routes during `init`. |
| `.cursor/rules/scaffold-testing.mdc` | Translate | Not directly about Harness internals, but the pattern is relevant: generated bundles must be self-contained, must declare their footprint, and must have golden/matrix tests. Apply to `.harness` bundle generation and target repo scaffolding. |
| `.cursor/rules/no-push-to-main.mdc` | Exclude | This conflicts with strict trunk. Preserve only as historical contrast. Harness default must say commit and push `main`. |
| `.cursor/rules/clean-branch-before-commit.mdc` | Translate | Convert into clean-main discipline: start from current `origin/main`, keep unrelated work out, commit one semantic step, push `main`, and ensure the tree is clean after completion. |

## Entrypoint Docs

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `../mars/AGENTS.md` | Translate | Good shape for a root agent guide: compact map, constraints, run/test commands, pointers, known setup issues. Do not port Mars SaaS constraints or PR workflow. |
| `../mars/template/AGENTS.md` | Direct | High-value pattern for Harness-generated target repos. `mars-harness init` should emit a richer generated `AGENTS.md` with architecture, run/test, local setup, ticket lifecycle, decisions, dogfood, and pointers. |
| `../mars/ARCHITECTURE.md`, `README.md`, `CONTRIBUTING.md`, `SECURITY.md` | Reference | Useful as examples of project orientation, but not central to Harness parity. |

## Automation Docs

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `docs/automations/BOTS.md` | Direct | Create a Harness role registry equivalent. It should list roles, modes, triggers, model tier, tools, trust level, guardrails, score signals, schedules, and status. |
| `docs/automations/README.md` | Translate | Use as the model for operational documentation: pipeline diagram, trigger map, payload fields, orchestration signals, setup notes, failure semantics. Replace Cursor/GitHub Actions delivery with Harness queue/scheduler/executor behavior. |
| `docs/automations/SETUP.md` | Reference | Mostly Cursor UI setup and secrets. Relevant only as a warning: Harness should avoid this setup burden and make GitHub optional/validated. |
| `docs/automations/prompts/planner.md` | Direct | Very relevant. Import the two-mode pattern: strategic `vision` plus tactical `tickets`; active-plan hygiene; quality regression triage; intervention-debt auto-promotion; "This week only" ticket creation. Translate PR output to direct `main` commits. |
| `docs/automations/prompts/engineer.md` | Direct | Very relevant. Import in-progress drain, blocker handling, one-ticket focus, stale cleanup, CI/pipeline-fix recipes, intervention-debt subroutine, failure learning, and loop prevention. Translate PR work into strict-trunk commit/push with guardrail policy. |
| `docs/automations/prompts/reviewer.md` | Translate | Keep the split between functional, security, and dependency modes. In Harness, Reviewer should review commits/traces/tickets and optionally GitHub checks, not PRs by default. |
| `docs/automations/prompts/maintainer.md` | Direct | High relevance for Harness self-maintenance: docs index checks, generated-doc freshness, quality score accuracy, tech-debt reconciliation, release readiness, and no empty work. |
| `docs/automations/prompts/end-to-end-tester.md` | Direct | High relevance, especially the dogfood semantics: real user path, matrix coverage, new-vs-known failure dedupe, escalation after repeated failures, and one ticket per run maximum. |

## Core Design Docs

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `docs/design-docs/automation-team.md` | Direct | This is the main operating-model source. Port the founding principle, canonical decision registry, AI-owned delivery idea, intervention-debt loop, ticket lifecycle, dogfood confidence model, deterministic-before-LLM approach, and Orchestrator signals. Translate PR mechanics to strict trunk. |
| `docs/design-docs/agent-context-model.md` | Direct | This is the strongest role-topology candidate for Harness: six domains, payload modes, memory continuity, mode maps, rules for changing role count, and Orchestrator as non-agent surveyor. |
| `docs/design-docs/conversation-as-system-record.md` | Direct | Add a Harness version. The current Harness docs imply this, but Mars states it cleanly and ties artifacts to conversation types. |
| `docs/design-docs/self-correcting-ci.md` | Direct | Port the principle: deterministic fixes should be guarantees or recipes before model repair. Translate changeset/auto-merge specifics into Harness doctor/remediation/orchestrator recipes. |
| `docs/design-docs/core-beliefs.md` | Translate | Import agent-first principles: repo as system of record, actionable errors, boring tech, central enforcement, every conversation feeds back. Do not import Mars-specific Next.js architecture as generic Harness rules. |
| `docs/design-docs/dogfood-matrix-automation.md` | Translate | Valuable pattern for generated verification matrices: spec registry, tiered coverage, machine-readable freshness, merge-blocking failure semantics, and gradual migration from manual matrix. |
| `docs/design-docs/generated-apps-p0-policy.md` | Translate | Apply to Harness target repo confidence: generated bundles and target scaffolds should have a P0 "said vs done" dogfood bar. Do not port the specific SaaS feature rows globally. |
| `docs/design-docs/scaffold-testing-strategy.md` | Translate | Use the testing philosophy: strategic fixtures, tiered verification, generated output self-containment, and browser-level confidence for user-visible flows. |
| `docs/design-docs/local-first-development.md` | Reference | Relevant to Harness zero-config philosophy, local-first stubs, and config honesty. Most Next.js/generator details are Mars-specific. |
| `docs/design-docs/trusted-publishing.md` | Reference | Useful for release incident analysis and "single source of release intent." The PR/changset/npm specifics are not Harness defaults. |
| `docs/design-docs/platform-*.md` | Reference | Future hosted-product material. Not required for core Harness parity. |
| `docs/design-docs/adopt-composition-patterns.md`, `optional-peer-dep-imports.md`, `zod-4-migration.md`, `migrate-prisma-7.md`, `fumadocs-v14-migration.md`, `separate-dts-from-tsup.md`, `workspace-version-resolution.md`, `feature-catalog-parity.md`, `cli-version-sourcing.md`, `mars-stack-npm-scope.md` | Reference or Exclude | Mostly Mars product implementation decisions. They can inform target-specific generated docs, but they are not Harness operating-model requirements. |

## Standing Trackers And Specs

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `docs/QUALITY_SCORE.md` | Direct | Harness needs a repo-visible quality score export sourced from scoring DB. It should drive Planner and Orchestrator triggers on regression. |
| `docs/exec-plans/tech-debt.md` | Direct | Import the standing tracker pattern and leading indicators table. Harness should track intervention debt, stuck in-progress tickets, score regressions, dogfood failures, model/setup drift, and guardrail blocks. |
| `docs/exec-plans/pipeline-learnings.md` | Direct | Harness should record structured failure learnings from deterministic recipes and pipeline fixes. Engineers should read these before work; Planner should promote repeated patterns. |
| `docs/tickets/README.md` | Translate | Keep repo-native markdown tickets and rich acceptance criteria. Replace Mars `needs-human/` with Harness strict-trunk blocked/intervention-debt semantics. Keep edge-case AC discipline. |
| `docs/product-specs/mission.md` | Translate | Useful north-star shape: product promise plus operating pillars. Harness already has tenets; use this to improve prioritization wording and score exports. |
| `docs/product-specs/feature-production-readiness.md` | Reference | Product-specific roll-up, but the pattern of "inventory, verdict, cross-cutting gates, refresh instructions" is relevant for Harness feature readiness. |
| `docs/COMPATIBILITY.md` | Reference | Mars package compatibility policy. Harness may need its own for OS, Go, llama.cpp, model revisions, and target repo assumptions. |

## Completed Plans With Reusable Lessons

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `completed/agent-context-model-migration.md` | Direct | Best migration playbook for reducing role sprawl. Reuse phased migration, donor-memory rationale translated to Harness traces/learnings, one-week observation gates, and registry update discipline. |
| `completed/needs-human-as-evolution-signal.md` | Direct | Best evidence for intervention-debt mechanics. Port the four-part fix: close stale cycles, use head-commit age not PR updated time, retry blocked work deliberately, and file intervention-debt tickets. |
| `completed/pipeline-self-correction-deep-analysis.md` | Direct | Provides root-cause taxonomy and layered solution model. Port the layered model: prevention, main/trunk hygiene, fixer reliability, merge/completion automation, observability, and careful expansion of fixable recipes. |
| `completed/dogfood-guard-rails-generated-apps-p0.md` | Translate | Use as the pattern for turning a one-time dogfood plan into a standing design policy. Harness should do this for target repo dogfood and `.harness` bundle confidence. |
| `completed/scaffold-template-sync-upgrade.md` | Translate | Strong model for `mars-harness upgrade`: manifest, path categories, dry-run, backups, excluded/generated/user-owned distinctions, and tests. |
| `completed/pipeline-failure-learning.md`, `pipeline-hygiene-changeset-and-dependabot.md`, `pipeline-learnings-foundation.md`, `pipeline-learnings-consumer-wiring.md` | Direct | Reinforce structured learnings and deterministic remediation before LLM repair. |
| Other completed feature tickets/plans | Reference | Mostly Mars app/product work. Use only when a similar Harness failure class appears. |

## Historical Tickets

The individual `MARS-*` tickets are mostly historical evidence. Relevant patterns:

- Intervention-debt tickets `MARS-093` to `MARS-098` prove the `needs-human` loop became a ticketed pay-down system.
- Dogfood and matrix tickets `MARS-041`, `MARS-042`, `MARS-062`, `MARS-099` to `MARS-102` show how manual verification matured into CI-backed coverage.
- Pipeline tickets `MARS-021` to `MARS-030`, `MARS-049`, `MARS-050`, `MARS-054`, `MARS-055`, `MARS-057`, and `MARS-063` show the recurring theme: when a human notices a failure, add a deterministic check, recipe, or orchestrator signal.

Do not import ticket IDs into Harness defaults. Import the failure classes and closure patterns.

## Reference Docs

| Mars file | Relevance | Harness use |
| --- | --- | --- |
| `docs/references/harness-engineering-agent-first.md` | Direct | This is foundational for Harness and has been carried into `docs/references/`: AGENTS as map, docs as system of record, plans as artifacts, rule-to-code promotion, entropy management, and agent-first development. |
| `docs/references/ui-design-brain.md`, `vercel-composition-patterns.md`, `vercel-react-best-practices.md` | Reference | These have been carried into `docs/references/` as target-project and future skill/bundle references. They are useful for generated frontend repos, but not core Harness operating-model parity. |

## What To Port Into Mars Harness First

1. **Harness operating model doc** based on `automation-team.md` and `agent-context-model.md`.
2. **Role registry** based on `BOTS.md`, but native to Harness manifests, trust, tools, guardrails, and scoring.
3. **Documentation hygiene checks** based on `documentation-discipline.mdc`.
4. **Conversation-as-system-record doc** and generated target guidance.
5. **Intervention-debt schema and automatic ticket creation** based on `needs-human` evolution.
6. **In-progress ticket drain and blocker repair** based on Engineer prompt and user-observed Harness dogfood failures.
7. **Quality score export** based on `QUALITY_SCORE.md`, wired to scores/trust DB.
8. **Pipeline learnings and deterministic remediation recipes** based on `pipeline-learnings.md` and `self-correcting-ci.md`.
9. **Native Orchestrator survey loop** based on Mars Orchestrator signals, without depending on GitHub Actions.
10. **Generated target `AGENTS.md` parity** based on `template/AGENTS.md`.

## Translation Rules For Strict Trunk

| Mars concept | Harness translation |
| --- | --- |
| Open PR | Commit semantic step to `main`, push `main`, optionally publish status/check/comment. |
| Auto-merge | Commit/push gate plus post-commit checks; no manual integration wait. |
| PR review comment | Reviewer finding, trace annotation, ticket comment, GitHub check/comment if configured. |
| Stuck PR | Stuck job, stuck in-progress ticket, failed check, or blocked trace. |
| `needs-human` label | Intervention-debt ticket plus blocked outcome, with retry/recovery policy. |
| Cursor memory | Harness traces, learnings, scoring events, role registry, and repo artifacts. |
| GitHub Actions schedule/router | Harness scheduler, queue, trigger router, and Orchestrator. |
| Prompt sync discipline | Generated prompt/manifest upgrade checks; no manual Cursor UI sync. |
| Branch cleanliness | Clean-main and clean-tree discipline before and after each strict-trunk step. |

## Deliberate Non-Imports

- Default PR/branch workflow.
- "Never push main" policy.
- Cursor UI as execution source of truth.
- Prompt re-paste runbooks as a product requirement.
- Mars SaaS generator-specific constraints as universal target repo rules.
- npm changesets as a Harness release requirement.
- Dependabot and Version Packages PR machinery as default orchestration.
- Mars-specific feature flags, Next.js auth rules, CSRF implementation, and UI component policies as generic Harness behavior.

## Impact On The Active Supersession Plan

The audit supports these workstreams in [mars-parity-supersession-plan.md](../exec-plans/backlog/mars-parity-supersession-plan.md):

- A and B: role model and registry.
- C: conversation and documentation discipline.
- D and E: intervention debt, active ticket drain, and blocker repair.
- F: quality score as repo artifact.
- G and H: Orchestrator and deterministic remediation.
- I and J: dogfood matrix and generated target parity.
- K and L: optional GitHub integration and setup truth.

The strongest near-term move is to build the role registry and operating-model doc first. Those become the anchor for prompt generation, manifest validation, doctor checks, scoring, and the eventual `../mars` observer-mode supersession trial.
