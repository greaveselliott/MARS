# Design Documents

Catalog of architectural decisions and design rationale for the Mars Harness project.

| Document | Status | Summary |
|----------|--------|---------|
| [tenets.md](tenets.md) | Accepted | The 9 founding tenets: plug and play, self-improving system, accuracy scoring, customisable guardrails, roadmap from init, blast radius containment, execution truth, progressive autonomy, context efficiency |
| [agent-runtime.md](agent-runtime.md) | Draft | Agent execution loop: multi-turn conversation, tool calling, error handling, budget enforcement. AD-004 (sync per-job), AD-005 (sequential tools), AD-006 (additive context) |
| [local-inference.md](local-inference.md) | Draft | Local model serving: llama.cpp as subprocess, hardware profiles, model registry, download management. AD-007 (no CGO), AD-008 (weights in ~/.mars-harness/), AD-031 (inference resilience), AD-032 (zero-config performance tuning) |
| [scoring-system.md](scoring-system.md) | Draft | Accuracy and value scoring: outcome tracking, rolling scores, progressive autonomy thresholds, noop detection |
| [self-improvement.md](self-improvement.md) | Draft | Intervention detection, Reviewer meta-role, bounded evolution commits, before/after tracking, safety rails |
| [guardrails.md](guardrails.md) | Draft | Advisory vs hard guardrails, validation types, override mechanism, staleness detection. AD-012 (syntactic only in v1) |
| [pipeline-engine.md](pipeline-engine.md) | Draft | Job queue (SQLite), worker dispatcher, cron scheduler, sandbox. AD-009 (SQLite), AD-010 (repo_id from day one) |
| [dashboard.md](dashboard.md) | Draft | 5-page dashboard: pipeline flow, role health, throughput, debug, evolution history. AD-011 (htmx + Chart.js embedded) |
| [context-efficiency.md](context-efficiency.md) | Draft | Context assembly, budgets, knowledge routing, guardrail scoping |
| [trigger-orchestration.md](trigger-orchestration.md) | Draft | Trigger sources (webhook, schedule, chain), upstream chaining via `then`, custom cron, strict-trunk default roles. AD-016 through AD-020. |
| [dogfood-and-decisions.md](dogfood-and-decisions.md) | Accepted | Containerised E2E validation (Podman + native fallback), decision recording tool, strict-trunk pipeline for local use. AD-021 through AD-030, AD-033. |

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
| AD-020 | Strict trunk keeps default roles schedule- and chain-driven; compatibility event handlers must be explicit. | trigger-orchestration.md | Orchestrator |
| AD-021 | Dogfood tester uses Podman containers with graceful native fallback. Auto-generated Containerfile from conventions. | dogfood-and-decisions.md | Dogfood |
| AD-022 | `record_decision` is a first-class agent tool, not a file-write convention. Decisions persist in `.harness/learnings.yaml` and are injected into all future agent context. | dogfood-and-decisions.md | Dogfood |
| AD-023 | Agents push after every semantic commit. Work must never be trapped locally. | dogfood-and-decisions.md | Dogfood |
| AD-024 | Strict-trunk default pipeline for local use. Optional webhook integrations add telemetry and repair triggers without replacing direct commits to `main`. | dogfood-and-decisions.md | Dogfood |
| AD-025 | Dogfood chains from Engineer (not schedule-only) so builds are validated after every feature when harness isn't always-on. | dogfood-and-decisions.md | Dogfood |
| AD-026 | Bootability checks: scanner detects unbootable projects (missing scripts, root layout, Tailwind config, conflicting dirs); Engineer must build-verify before closing tickets; Dogfood runs pre-flight structural checks; QA validates structural integrity. | dogfood-and-decisions.md | Bootability |
| AD-027 | Interactive control surface: CLI key listener (p/r/s/q/h) + dashboard control bar. Shared Server methods, warm restart, pause/resume workers, scan repos, run specific roles. | dashboard.md | Controls |
| AD-028 | Git tools (`git_status`, `git_diff`, `git_commit`, `git_push`) added to all write-capable role manifests. Commit gates in prompts enforce clean working tree before run ends. Scanner detects missing `.gitignore`. | dogfood-and-decisions.md | Git discipline |
| AD-029 | Per-repo database isolation. Default DB path changed from `~/.mars-harness/db/mars.db` to `~/.mars-harness/db/{repo-slug}/mars.db`. `RepoScope` filter in `serve.Config` as defense-in-depth. `doctorCmd` gains `--repo` flag. Migration warning for legacy DB. | dogfood-and-decisions.md | Isolation |
| AD-030 | Mechanical ticket deduplication. `ticket_create` tool with built-in duplicate detection (normalized title subset matching). Ticket index injected into context assembly for COO/engineer/janitor. COO prompt rewritten to use tool-level enforcement. Scaffold template cleaned of Mars-specific boilerplate. | dogfood-and-decisions.md | Ticket dedup |
| AD-031 | Inference resilience. HTTP timeout 60s→5min for local inference. Fast-tier context 8192→16384 (Gemma 4 supports 128k). Active health spot-check in router before returning endpoint (catches stale healthy state from crashed servers). Non-retryable error for context-exceeded. Retry backoff increased across client and agent loop. | local-inference.md | Resilience |
| AD-032 | Zero-config local inference performance profile and llama tuning. Auto profile selects smaller Q4/Q5 defaults on memory-constrained unified-memory machines while keeping overrides available. Default parallel slots are capped at 1 for strict-trunk single-agent throughput. | local-inference.md | Performance |
| AD-033 | In-progress tickets are drained before backlog work. Engineer gates allow completing one pre-existing in-progress ticket while others remain queued, but block new claims, stale returns to backlog, and no-progress handoffs. | dogfood-and-decisions.md | Ticket completion |
