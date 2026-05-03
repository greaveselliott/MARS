# Product Surface

**Status:** Accepted
**Updated:** 2026-05-03
**Owner:** Mars Harness maintainers
**Sources:** [AGENTS.md](../../AGENTS.md), [quickstart](../quickstart.md), [design docs](../design-docs/index.md), [vision](vision.md)

## Current Product Shape

Mars Harness is a local autonomous delivery runtime with four visible layers:

| Layer | User-facing surface | Product promise |
| --- | --- | --- |
| Install and setup | installed `mars-harness` command, `mars-harness setup`, `mars-harness path setup`, config, model and binary cache | Run the harness from any directory, configure the user's shell PATH automatically, detect hardware, install local inference, choose a sensible performance profile, and explain missing prerequisites. |
| Target harness | `mars-harness init`, `upgrade`, generated `AGENTS.md`, `.harness/`, docs, tickets, references | Give every target repo a mirrored agent operating system from day one. |
| Execution | `run`, `start`, `serve`, queue, scheduler, tools, traces, dashboard | Execute roles against target repos with bounded tool access, strict trunk commits, visible run state, and narrow self-healing for stale recovery jobs. |
| Delivery model | `docs/goals/`, `docs/features/`, one active exec plan, BDD scenario evidence, ticket completion gates | Align work to goals, define feature completeness before implementation, and ship only scenarios with real E2E/integration evidence. |
| Learning loop | `scores`, `trust`, `docs/QUALITY_SCORE.md`, telemetry triage, skills, guardrails, decisions, evolution reviews | Turn real outcomes into trust changes, intervention work, reusable workflow skills, prompt or process improvements, repo-visible grades, and safety controls. |
| Generated references | `docs/generated/` | Provide reproducible, cataloged source-harness maps when generator commands exist. |
| Release state | `VERSION`, `CHANGELOG.md`, `release notes` | Maintain semantic versions and generated patch notes for both the source harness and target repos. |

## CLI Contract

| Command | Status | Product behavior |
| --- | --- | --- |
| `mars-harness setup` | Implemented, still hardening | Creates `~/.mars-harness/`, configures supported shell profiles for the installed command, writes config, detects hardware, installs llama.cpp server artifacts, downloads pinned models, and keeps optional integration setup explicit. |
| `mars-harness update check --repo <path>` | Implemented | Reports whether the installed CLI, deployed target `.harness/` metadata, or mirrored operating-model artifacts are behind, with JSON output for automation and unknown-but-nonfatal remote status when release lookup fails. |
| `mars-harness update tool` | Implemented | Downloads the latest platform release asset, verifies `checksums.txt`, atomically replaces the installed command, and configures shell PATH. Source-development updates remain available with `--source` or `--version main`. |
| `mars-harness path setup --install-dir <path>` | Implemented | Detects Fish, Zsh, Bash, POSIX sh/Ksh, Csh, or Tcsh and writes an idempotent user-profile snippet so new terminals can resolve `mars-harness`. |
| `mars-harness update harness --repo <path>` | Implemented | Uses the same update verb to refresh the deployed target `.harness/` bundle without overwriting user-owned agent configuration. |
| `make install` from source checkout | Implemented for source development | Installs the dev binary into the Go bin directory and runs the installed binary's PATH setup so operators do not run stale source-root binaries or hand-edit shell config. |
| `mars-harness init --repo <path>` | Implemented | Scaffolds the target harness: manifest, roles, guardrails, knowledge routes, compact `AGENTS.md`, goals, BDD feature contracts, tickets, exec-plan docs, design-doc index, context glossary, quality score, and references. |
| `mars-harness upgrade --repo <path>` | Implemented, still hardening | Fills missing target harness defaults while preserving user-owned manifest, role prompts, knowledge routes, guardrails, tickets, design docs, exec plans, references, and target `AGENTS.md`. |
| `mars-harness scan --repo <path> --tickets` | Implemented | Finds repo gaps and writes deduplicated backlog tickets through the canonical ticket path. |
| `mars-harness run <role> --repo <path>` | Implemented | Loads manifest, guardrails, knowledge routes, context, tools, local model endpoint, and runs one role with terminal-result truth. |
| `mars-harness start --repo <path>` | Implemented | Initializes if needed, registers the repo, seeds the CEO role, and runs the per-repo autonomous pipeline with isolated database state and recovery-queue self-healing. |
| `mars-harness serve` | Implemented, multi-repo legacy mode | Runs the orchestrator, dashboard, webhook receiver, cron scheduler, workers, and recovery-queue self-heal watchdog against the configured database. |
| `mars-harness register --repo <path>` | Implemented | Registers a repo and creates the per-repo database path when one is not supplied. |
| `mars-harness doctor [--repo <path>] [--json]` | Implemented, expanding | Checks Go, config, model registry, models directory, database, llama-server, disk space, guardrail/workflow health, mirrored operating-model health, active-plan hygiene, and optional integration configuration. |
| `mars-harness scores [--repo <path>]` | Implemented | Shows trunk-native role scores from stored outcomes. |
| `mars-harness scores export --repo <path>` | Implemented | Refreshes `docs/QUALITY_SCORE.md` from live score, telemetry, ticket, dogfood, guardrail, check, no-op, and human follow-up evidence while preserving manual notes and creating deduped low-score intervention-debt tickets. |
| `mars-harness trust [--repo <path>]` | Implemented | Shows role trust levels. |
| `mars-harness trust set <role> <repo> <level> --reason <text>` | Implemented | Overrides trust with an audit reason. |
| `mars-harness models evaluate [--endpoint <url> --model <name>]` | Initial implementation | Prints the current model-refresh plan and candidate shortlist, or runs mechanical OpenAI-compatible benchmark probes against a supplied model endpoint. Next slice adds Ollama catalog access, explicit tier/role swaps, persisted reports, and promotion decisions. |
| `mars-harness release notes --repo <path> --bump auto` | Implemented | Generates semantic-versioned patch notes from commits, updates `VERSION`, and prepends `CHANGELOG.md`. |
| `mars-harness release verify-assets [--version <tag>]` | Implemented | Fails a release check unless all four platform binaries and `checksums.txt` are attached to the GitHub Release. |

## Generated Target Harness

`mars-harness init` must produce a target repo that is immediately usable by Codex, Cursor, Mars Harness roles, and humans.

Required generated surfaces:

- `AGENTS.md` as the compact first-read map
- `.harness/manifest.yaml` for roles, model tiers, tools, triggers, chains, guardrails, and knowledge routes
- `.harness/metadata.yaml` for generated-harness version drift checks
- `.harness/roles/*.md` for role prompts
- `.harness/skills/*/SKILL.md` for compact reusable workflows and self-improvement guidance
- `.harness/guardrails/*.yaml` for mechanical policy inputs
- `.harness/knowledge/*.yaml` for lightweight context routes
- `docs/goals/` for active goals, observations, and superseded goals
- `docs/features/` for Markdown BDD feature contracts and scenario schedules
- `docs/tickets/backlog/`, `docs/tickets/in-progress/`, and `docs/tickets/done/`
- `docs/tickets/README.md` for ticket lifecycle and completion rules
- `docs/exec-plans/README.md` and starter priority docs with a one-active-plan lifecycle
- `docs/QUALITY_SCORE.md` as the repo-visible A-F quality score artifact
- `docs/design-docs/index.md` and `context-glossary.md`
- `docs/references/README.md` and selected agent-first references
- `VERSION`, `CHANGELOG.md`, and `docs/design-docs/release-versioning.md`

Generated target docs must mirror source-harness doctrine while staying project-agnostic. Existing target harness files and user-owned docs are preserved by upgrades.

Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only. Any change to commit discipline, versioning, ticket flow, documentation rules, skill creation, guardrail policy, trust/scoring behavior, release behavior, or context-routing discipline must update generated target guidance and tests in the same task.

Architecture changes and product features must carry rationale in repo-owned docs: what changed, why it changed, and which behavior future agents should preserve. Source harness changes use design docs and product specs; initialized target repos inherit the same rule through generated `AGENTS.md`, using their design docs and, when present, product specs.

Exec plans mirror the ticket lifecycle. Exactly one plan may be active at a
time. Waiting plans live in `docs/exec-plans/backlog/` with explicit priority,
dependencies, blockers, related tickets, goals, BDD feature contracts,
hypotheses, success/falsification evidence, scenario schedules, current failing
scenarios, walking skeleton slices, and learning or MVP outcomes. Superseded
plans are lineage only.

BDD feature contracts are the source of truth for feature completeness.
Walking skeleton is the implementation strategy: agents implement the next
failing scenario through the thinnest real end-to-end path. Feature tickets must
carry BDD scenario evidence before done. Enabler work is allowed, but release
notes and quality score must separate enabler work from shipped feature
scenarios.

## Versioning And Patch Notes

Mars Harness and initialized target repos use the same release contract:

- `VERSION` contains `MAJOR.MINOR.PATCH`
- `CHANGELOG.md` contains generated patch notes
- `mars-harness release notes --repo . --bump auto` infers the next version from semantic commits
- release-note commits themselves are ignored in the next generated entry
- generated entries include a marker so tags are useful but not required for the next diff
- in the source harness repo and initialized target repos, every non-release semantic commit is immediately followed by the generated version/patch-note commit before the task is done
- when authenticated GitHub release capability is configured, the generated version is tagged as `vX.Y.Z`; the source harness Release workflow publishes or backfills the matching changelog entry plus checksum-verified binary assets

## Generated Source References

`docs/generated/` is reserved for reproducible reference snapshots generated from the source harness. It is intentionally catalog-only until generator commands exist.

Expected future artifacts include role registry, tool inventory, package map, model inventory, and bundle schema reference. Generated docs must name their generator, source inputs, and freshness signal so agents can trust them as context routes.

## Role And Work Semantics

Default roles are configurable starter agents, not perfect built-ins. The target repo owner can edit prompts, add or remove roles, change schedules and chains, restrict tools, attach guardrails, and route context differently through `.harness/manifest.yaml`.

The product contract is:

- Planner roles create scoped, deduplicated work.
- CEO owns goals, BDD feature contracts, scenario schedule, tradeoffs, and the active exec plan.
- CTO validates the hypothesis, architecture fit, and whether the walking skeleton is real.
- COO creates tickets only from the current failing BDD scenario or scenario group.
- Engineer roles complete one ticket per run.
- Engineer roles provide scenario evidence before closing feature tickets.
- In-progress tickets are highest priority.
- Intervention-debt tickets are generated from repeated telemetry failures or low score snapshots and outrank ordinary backlog work.
- Blocked work is documented and proactively unblocked when the fix is in scope.
- Dogfood and QA roles produce reproducible evidence.
- Janitor and orchestrator roles keep ticket state truthful.
- Evolution roles improve the harness only inside trust and safety limits.
- Repeated human recovery steps become compact scoped skills when they describe reusable procedure.

The harness should never reward a role for handing off incomplete work as if it were complete.

## Trust And Scoring

Trust levels are:

- `observer`: read and report only
- `contributor`: human-triggered or ticket-bound edit, test, commit, and push to `main`
- `autonomous`: may self-schedule, chain work, edit, test, commit, push to `main`, and perform bounded evolution

Scores are based on real outcomes: completed work, commits, checks, guardrail blocks, timeouts, noops, human follow-up, and reverts. Scores must drive behavior, not merely appear in a dashboard.

`docs/QUALITY_SCORE.md` is refreshed with `mars-harness scores export --repo <path>`. The generated artifact is the quality source of truth for agents; dashboard quality views link to the same file and database-derived signals instead of becoming a separate grading surface. Missing SQLite evidence is explicitly graded as insufficient evidence.

When scores or telemetry show repeated workflow confusion, the harness creates or updates intervention-debt tickets first, then chooses the bounded improvement surface. Prefer a scoped skill over bloating a role prompt. Use guardrails for non-negotiable enforcement and tools for deterministic actions.

## Guardrails And Safety

The product must enforce hard rules at tool execution, not only in prompts. Mutating tools are checked through session context containing role, job, repo, trust, guardrails, and safety limits.

Required protections include:

- repo-relative file writes
- observer mutation blocks
- hard guardrail checks
- secret scanning
- destructive shell and git operation blocks
- blast-radius limits before commit and push
- emergency stop for active workers

## Local Inference And Performance

Setup should choose the best practical local profile automatically. On Apple Silicon and other unified-memory machines, high RAM use and low CPU use can still indicate GPU/Metal-bound inference. The product should optimize for actual tokens per second, not CPU utilization alone.

Current defaults favor:

- `performance_profile: auto`
- one parallel generation slot for strict-trunk single-agent throughput
- hardware-based model tier selection, with manifest `role.model` as the source of truth for role routing
- pinned model revisions and SHA256 verification
- pinned llama.cpp artifacts with checksums
- doctor checks for missing or stale model state
- model default changes backed by `mars-harness models evaluate` evidence rather than newest-model claims alone
- broad Ollama model access for evaluation and explicit operator-owned swaps, without treating every available model as a safe zero-config default

Manual tuning remains available, but the normal path should require none.

## Optional Integrations

Mars Harness is complete without a remote-code-host integration. Optional integration exists for telemetry and coordination: webhooks, statuses, comments, and check-run style reporting.

The product must never describe optional integration as complete unless credentials, webhook delivery, and status/comment behavior have actually been validated.

## Known Hardening Areas

The single active execution plan tracks the remaining work and pulls from prioritized plan backlog items. The highest-value hardening areas are:

- richer repo-visible role registry
- richer repo-visible skill registry and skill-evolution proposals
- stronger generated target guidance
- deterministic remediation recipes before LLM work
- richer dashboard views for improvement targets backed by the generated quality score
- doctor checks for mirrored harness freshness
- generated role, tool, package, model, and score reference artifacts
- automatic intervention-debt ticket creation from telemetry triage
- broader dogfood matrices and generated-app validation
- safer upgrade previews and backups for target harness files
