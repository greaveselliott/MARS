# Design Documents

Catalog of architectural decisions and design rationale for the Mars Harness project.

| Document | Status | Summary |
|----------|--------|---------|
| [tenets.md](tenets.md) | Accepted | The 9 founding tenets: plug and play, self-improving system, accuracy scoring, customisable guardrails, roadmap from init, blast radius containment, execution truth, progressive autonomy, context efficiency |
| [agent-runtime.md](agent-runtime.md) | Draft | Agent execution loop: multi-turn conversation, tool calling, error handling, budget enforcement. AD-004 (sync per-job), AD-005 (sequential tools), AD-006 (additive context) |
| [local-inference.md](local-inference.md) | Draft | Local model serving: llama.cpp as subprocess, hardware profiles, model registry, download management. AD-007 (no CGO), AD-008 (weights in ~/.mars-harness/), AD-031 (inference resilience), AD-032 (zero-config performance tuning), AD-063 (benchmark-backed model promotion), AD-064 (manifest-tier routing), AD-066 (Ollama catalog/swap provider) |
| [scoring-system.md](scoring-system.md) | Draft | Accuracy and value scoring: outcome tracking, rolling scores, progressive autonomy thresholds, noop detection |
| [self-improvement.md](self-improvement.md) | Draft | Intervention detection, Reviewer meta-role, bounded evolution commits, before/after tracking, safety rails. AD-071 (active plan drift), AD-073 (single active exec plan with dependencies) |
| [guardrails.md](guardrails.md) | Draft | Advisory vs hard guardrails, validation types, override mechanism, staleness detection. AD-012 (syntactic only in v1) |
| [pipeline-engine.md](pipeline-engine.md) | Draft | Job queue (SQLite), worker dispatcher, cron scheduler, sandbox. AD-009 (SQLite), AD-010 (repo_id from day one) |
| [dashboard.md](dashboard.md) | Draft | 5-page dashboard: pipeline flow, role health, throughput, debug, evolution history. AD-011 (htmx + Chart.js embedded) |
| [context-efficiency.md](context-efficiency.md) | Draft | Context assembly, budgets, knowledge routing, guardrail scoping |
| [harness-glossary.md](harness-glossary.md) | Accepted | First-class and contextual harness definitions mirrored between the foundation harness and deployed harnesses. |
| [trigger-orchestration.md](trigger-orchestration.md) | Draft | Trigger sources (webhook, schedule, chain), upstream chaining via `then`, custom cron, strict-trunk default roles. AD-016 through AD-020. |
| [dogfood-and-decisions.md](dogfood-and-decisions.md) | Accepted | Containerised E2E validation (Podman + native fallback), decision recording tool, strict-trunk pipeline for local use, ticket-drain discipline, recovery loop containment, queue self-healing, and tool-creation scaffolding. AD-021 through AD-030, AD-033, AD-060, AD-062, AD-077. |
| [mirrored-harness-and-context-glossary.md](mirrored-harness-and-context-glossary.md) | Accepted | Source and initialized harness parity, glossary-as-route context, repo-owned workflow contracts, operating-rule inheritance, and rationale-bearing architecture/product docs. AD-034 through AD-036, AD-058, AD-061. |
| [self-reflective-telemetry.md](self-reflective-telemetry.md) | Accepted | Harness grades itself and triages telemetry into prompt, skill, process, guardrail, context, inference, manifest, and tool-policy improvement targets. AD-037 through AD-039, AD-065, AD-072. |
| [product-spec-governance.md](product-spec-governance.md) | Accepted | Product specs as a living product contract with metadata, index coverage, and docs-consistency enforcement. AD-040 through AD-042. |
| [generated-docs-governance.md](generated-docs-governance.md) | Accepted | Generated docs as reproducible reference snapshots with catalog and docs-consistency checks. AD-043 through AD-045. |
| [role-customization.md](role-customization.md) | Accepted | Shipped roles as configurable starter agents; target manifests, prompts, knowledge routes, and guardrails are user-owned after init. AD-046 through AD-048. |
| [release-versioning.md](release-versioning.md) | Accepted | Semantic versioning, generated patch notes, mirrored source/target release behavior, automatic versioning after non-release commits, GitHub Release publication when configured, self-update, release assets, and shell PATH setup. AD-049 through AD-051, AD-056 through AD-057, AD-059, AD-068 through AD-070, AD-075, AD-078. |
| [skill-evolution.md](skill-evolution.md) | Accepted | Skills as a first-class self-improvement target and the decision matrix for prompt, skill, tool, guardrail, or knowledge-route changes. AD-052 through AD-055. |
| [delivery-operating-model.md](delivery-operating-model.md) | Accepted | BDD-led goal-driven walking-skeleton delivery as the source and generated target operating model. AD-074. |

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
| AD-034 | Source and initialized harnesses must mirror operating doctrine: compact AGENTS.md, strict trunk, ticket workflow, design decisions, references, and knowledge routing. | mirrored-harness-and-context-glossary.md | Generated targets |
| AD-035 | Context glossary is a routing layer, not a prompt-stuffed manual. Initialized repos get `docs/design-docs/context-glossary.md` and `.harness/knowledge/context-glossary.yaml`. | mirrored-harness-and-context-glossary.md | Context |
| AD-036 | Workflow contracts belong in repo-owned artifacts. Current v1 contract is AGENTS.md + manifest + ticket docs + exec-plan docs + knowledge routes; a first-class WORKFLOW.md remains a future option. | mirrored-harness-and-context-glossary.md | Orchestration |
| AD-037 | Telemetry must be triaged into explicit improvement targets: prompt, skill, process, guardrail, context, inference, manifest, tool policy, or unknown. | self-reflective-telemetry.md | Self-improvement |
| AD-038 | Scores are control signals, not vanity metrics. Low rolling scores with enough samples trigger triage and bounded evolution review. | self-reflective-telemetry.md | Scoring |
| AD-039 | Self-improvement is proactive but bounded. Direct edits stay inside allowlisted harness surfaces; process/product changes become tickets or plans unless trust and scope allow direct evolution. | self-reflective-telemetry.md | Safety |
| AD-040 | Product specs are a living product contract, updated with product-facing changes and carrying status, update date, and owner metadata. | product-spec-governance.md | Docs |
| AD-041 | Product specs link to design docs, exec plans, references, and tickets instead of duplicating all supporting context. | product-spec-governance.md | Context |
| AD-042 | Product spec freshness is mechanically checked through docs-consistency tests for metadata, index coverage, links, and strict trunk wording. | product-spec-governance.md | Quality |
| AD-043 | Generated docs are reproducible reference snapshots, not hand-written decisions, tickets, product specs, or plans. | generated-docs-governance.md | Docs |
| AD-044 | Empty generated docs are acceptable until a generator exists, as long as the README states that intentionally. | generated-docs-governance.md | Docs |
| AD-045 | Generated docs must be cataloged and mechanically checked for README metadata, catalog coverage, and valid links. | generated-docs-governance.md | Quality |
| AD-046 | Shipped roles are starter agents, not a claim of universal correctness. Users may edit, replace, remove, or add roles. | role-customization.md | Roles |
| AD-047 | Target role configuration is user-owned once generated: manifest, prompts, knowledge routes, and guardrails are preserved by default. | role-customization.md | Upgrade |
| AD-048 | Upgrade fills missing defaults and does not silently retune existing agents. New prompt wording must be adopted deliberately. | role-customization.md | Upgrade |
| AD-049 | VERSION is the semantic version source for Mars Harness and initialized target repos. | release-versioning.md | Release |
| AD-050 | CHANGELOG.md stores generated patch notes with release markers so future runs can find new commits. | release-versioning.md | Release |
| AD-051 | Source and target release behavior mirrors through generated VERSION, CHANGELOG.md, release guidance, knowledge routes, and release-manager prompts. | release-versioning.md | Generated targets |
| AD-052 | Skills are a first-class evolution target when repeated failures or interventions reveal missing reusable procedure. | skill-evolution.md | Self-improvement |
| AD-053 | Skills must stay compact, scoped, evidence-oriented, and separate from large manuals or role identity. | skill-evolution.md | Context |
| AD-054 | Evolution triage chooses between prompt, skill, tool, guardrail, knowledge route, inference/config, manifest/tool policy, or tickets by evidence type. | skill-evolution.md | Self-improvement |
| AD-055 | Initialized target harnesses receive skill-evolution guidance so source and target workflows reduce human intervention the same way. | skill-evolution.md | Generated targets |
| AD-056 | Every non-release semantic commit to this source repo is followed by generated release notes and a `release: notes X.Y.Z` commit before the task is done. | release-versioning.md | Release |
| AD-057 | Initialized target repos inherit the same automatic versioning workflow through generated AGENTS.md, release-versioning docs, and release-manager prompts. | release-versioning.md | Generated targets |
| AD-058 | Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only. | mirrored-harness-and-context-glossary.md | Generated targets |
| AD-059 | Versioned release notes are published as GitHub Releases named `vX.Y.Z` when authenticated GitHub release capability is configured. | release-versioning.md | Release |
| AD-060 | Failed auto-recovery jobs do not recursively enqueue more recovery jobs, and active recovery is idempotent by repo and role. | dogfood-and-decisions.md | Recovery |
| AD-061 | Architecture changes and product features must be documented with their rationale, and the rule mirrors into initialized target harnesses. | mirrored-harness-and-context-glossary.md | Generated targets |
| AD-062 | Active recovery queue storms are self-healed by failing stale recovery jobs and cancelling duplicate pending recovery jobs. | dogfood-and-decisions.md | Recovery |
| AD-063 | Default model registry changes require harness-specific benchmark evidence and immutable pinned artifacts. | local-inference.md | Inference |
| AD-064 | Inference routing honors manifest `role.model` tiers before role-name fallback mapping, and missing-model errors name the expected repair path. | local-inference.md | Inference |
| AD-065 | Telemetry triage creates or updates `kind: intervention-debt` tickets through the canonical ticket path, deduped by repo, role, target, category, and evidence window. | self-reflective-telemetry.md | Self-improvement |
| AD-066 | Ollama is a first-class catalog and explicit model-swap provider, while zero-config default promotion still requires benchmark evidence, immutable revision, and SHA256. | local-inference.md | Inference |
| AD-067 | Source development installs the command into the Go bin directory before operating target repos; root-level source binaries are treated as stale-binary traps. | dogfood-and-decisions.md | Operator workflow |
| AD-068 | The installed command can update itself with `mars-harness update tool`; packaged users get checksum-verified release assets, while source-development updates remain available through `--source` or `--version main`. | release-versioning.md | Release |
| AD-069 | `update` is the unified CLI verb for the installed tool and deployed target harness. | release-versioning.md | Release |
| AD-070 | `mars-harness update check` reports tool and target harness version drift, writes JSON for automation, and feeds doctor warnings without mutating state. | release-versioning.md | Release |
| AD-071 | Stale active exec plans are intervention debt; they must be corrected with a current operating plan and mechanical hygiene checks. | self-improvement.md | Self-improvement |
| AD-072 | A-F quality scores are repo artifacts; source and initialized target repos carry `docs/QUALITY_SCORE.md` until deterministic score export owns the refresh loop. | self-reflective-telemetry.md | Self-improvement |
| AD-073 | Exec plans follow a ticket-like lifecycle with exactly one active plan; waiting plans live in a prioritized backlog with dependency, blocker, and related-ticket metadata, while historical plans live in superseded lineage. | self-improvement.md | Planning |
| AD-074 | BDD feature contracts define feature completeness; walking skeleton is the implementation strategy; one active plan schedules failing scenarios from active goals through tickets and evidence. | delivery-operating-model.md | Operating model |
| AD-075 | Source install, setup, and update-tool configure the user's shell PATH idempotently across common shells so the installed CLI is zero-config. | release-versioning.md | Setup |
| AD-076 | The harness glossary is mirrored first-class context in source and generated target `AGENTS.md`, with contextual definitions routed through `docs/design-docs/harness-glossary.md`. | mirrored-harness-and-context-glossary.md | Generated targets |
| AD-077 | `tool_create` is a mirrored mutating tool that scaffolds built-in tool files and tests while requiring separate implementation, registration, trust-policy review, and allowlist exposure. | dogfood-and-decisions.md | Tools |
| AD-078 | Source harness release assets are built from pushed tags, verified before publication, and checked with `mars-harness release verify-assets`. | release-versioning.md | Release |
