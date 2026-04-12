# Design Documents

Catalog of architectural decisions and design rationale for the Mars Harness project.

| Document | Status | Summary |
|----------|--------|---------|
| [tenets.md](tenets.md) | Accepted | The 9 founding tenets: plug and play, self-improving system, accuracy scoring, customisable guardrails, roadmap from init, blast radius containment, execution truth, progressive autonomy, context efficiency |
| [agent-runtime.md](agent-runtime.md) | Draft | Agent execution loop: multi-turn conversation, tool calling, error handling, budget enforcement. AD-004 (sync per-job), AD-005 (sequential tools), AD-006 (additive context) |
| [local-inference.md](local-inference.md) | Draft | Local model serving: llama.cpp as subprocess, hardware profiles, model registry, download management. AD-007 (no CGO), AD-008 (weights in ~/.mars-harness/) |
| [scoring-system.md](scoring-system.md) | Draft | Accuracy and value scoring: outcome tracking, rolling scores, progressive autonomy thresholds, noop detection |
| [self-improvement.md](self-improvement.md) | Draft | Intervention detection, Reviewer meta-role, evolution PRs, before/after tracking, safety rails |
| [guardrails.md](guardrails.md) | Draft | Advisory vs hard guardrails, validation types, override mechanism, staleness detection. AD-012 (syntactic only in v1) |
| [pipeline-engine.md](pipeline-engine.md) | Draft | Job queue (SQLite), worker dispatcher, cron scheduler, sandbox. AD-009 (SQLite), AD-010 (repo_id from day one) |
| [dashboard.md](dashboard.md) | Draft | 5-page dashboard: pipeline flow, role health, throughput, debug, evolution history. AD-011 (htmx + Chart.js embedded) |
| [context-efficiency.md](context-efficiency.md) | Draft | Context assembly, budgets, knowledge routing, guardrail scoping |
| [trigger-orchestration.md](trigger-orchestration.md) | Draft | Trigger sources (webhook, schedule, chain), upstream chaining via `then`, custom cron, dual-mode roles. Complete Mars pipeline role registry. AD-016 through AD-020. |
| [dogfood-and-decisions.md](dogfood-and-decisions.md) | Accepted | Containerised E2E validation (Podman + native fallback), decision recording tool, lean pipeline for local use. AD-021 through AD-025. |

## Architecture Decision Log

| ID | Decision | Design Doc | Milestone |
|---|---|---|---|
| AD-001 | Go as implementation language. Single binary, no CGO. llama.cpp as subprocess. | local-inference.md | M0 |
| AD-002 | Apache 2.0 license | index.md | M0 |
| AD-003 | Repo governance follows Mars conventions (AGENTS.md, design docs, exec plans, commit discipline) | AGENTS.md | M0 |
| AD-004 | Synchronous single-threaded agent loop per job. Concurrency at job level. | agent-runtime.md | M1 |
| AD-005 | Sequential tool execution, not parallel. Simpler, safer, easier to trace. | agent-runtime.md | M1 |
| AD-006 | Additive context assembly (concatenation with section headers). No template engines. | agent-runtime.md | M1 |
| AD-007 | llama.cpp managed as subprocess, not embedded via CGO. Clean binary distribution. | local-inference.md | M2 |
| AD-008 | Model weights stored in ~/.mars-harness/models/, not in repo. SHA256 in bundle.lock.json. | local-inference.md | M2 |
| AD-009 | SQLite for all persistent state (jobs, scores, interventions, traces). Zero external dependency. | pipeline-engine.md | M5 |
| AD-010 | repo_id column in schema from day one. Multi-repo ready even if v1 is single-repo. | pipeline-engine.md | M5 |
| AD-011 | htmx + Chart.js embedded in Go binary for dashboard. No React, no npm, no external dependencies. | dashboard.md | M9 |
| AD-012 | Hard guardrails limited to syntactic checks in v1 (regex, file pattern, file existence). AST analysis is v2. | guardrails.md | M7 |
| AD-013 | `log/slog` from stdlib for all structured logging. JSON in production, text in development. | AGENTS.md | M0 |
| AD-014 | Domain-grouped packages under `internal/` (agent, llm, tools, github, etc.). Public types in `pkg/` if needed. | AGENTS.md | M0 |
| AD-015 | YAML config file (`~/.mars-harness/config.yaml`) merged with environment variable overrides (`MARS_HARNESS_` prefix). | AGENTS.md | M0 |
| AD-016 | Three trigger source types: webhook, schedule, chain. No expression parser in v1. | trigger-orchestration.md | Orchestrator |
| AD-017 | Upstream chaining via `then` field. Direct chains vs event-mediated chains. | trigger-orchestration.md | Orchestrator |
| AD-018 | Fire-and-forget `OnComplete` hook, not a full event bus. Extension point for v2. | trigger-orchestration.md | Orchestrator |
| AD-019 | Custom cron with named presets (`hourly`, `daily`, `weekly`, `monthly`) as aliases. Standard 5-field only. | trigger-orchestration.md | Orchestrator |
| AD-020 | Dual-mode roles (e.g., CTO PR merge vs CTO weekly) are separate manifest entries sharing a prompt file. | trigger-orchestration.md | Orchestrator |
| AD-021 | Dogfood tester uses Podman containers with graceful native fallback. Auto-generated Containerfile from conventions. | dogfood-and-decisions.md | Dogfood |
| AD-022 | `record_decision` is a first-class agent tool, not a file-write convention. Decisions persist in `.harness/learnings.yaml` and are injected into all future agent context. | dogfood-and-decisions.md | Dogfood |
| AD-023 | Agents push after every semantic commit. Work must never be trapped locally. | dogfood-and-decisions.md | Dogfood |
| AD-024 | Lean 7-role pipeline for local use (CEO, CTO, COO, Engineer, QA, Dogfood, Janitor). Event-driven roles are dormant until GitHub webhooks are connected. | dogfood-and-decisions.md | Dogfood |
| AD-025 | Dogfood chains from Engineer (not schedule-only) so builds are validated after every feature when harness isn't always-on. | dogfood-and-decisions.md | Dogfood |
| AD-026 | Bootability checks: scanner detects unbootable projects (missing scripts, root layout, Tailwind config, conflicting dirs); Engineer must build-verify before closing tickets; Dogfood runs pre-flight structural checks; QA validates structural integrity. | dogfood-and-decisions.md | Bootability |
