# Product Surface

**Status:** Accepted
**Updated:** 2026-05-20
**Owner:** Mars Harness maintainers
**Sources:** [AGENTS.md](../../AGENTS.md), [quickstart](../quickstart.md), [design docs](../design-docs/index.md), [vision](vision.md)

## Current Product Shape

Mars Harness is a local autonomous delivery runtime with four visible layers:

| Layer | User-facing surface | Product promise |
| --- | --- | --- |
| Install and setup | installed `mars-harness` command, `mars-harness auth github setup`, `mars-harness setup`, `mars-harness path setup`, config, model and binary cache | Run the harness from any directory, configure private-release auth, configure the user's shell PATH automatically, detect hardware, install local inference, choose a sensible performance profile, and explain missing prerequisites. |
| Target harness | `mars-harness init`, `upgrade`, generated `AGENTS.md`, `.harness/`, docs, tickets, references | Give every target repo a mirrored agent operating system from day one. |
| Execution | `run`, `start`, `serve`, queue, scheduler, tools, traces, dashboard | Execute roles against target repos with bounded tool access, strict trunk commits, visible run state, default dispatch orchestration, and narrow self-healing for stale recovery jobs. |
| Delivery model | `docs/goals/`, one active exec plan, `docs/features/`, BDD scenario evidence, ticket completion gates, org liveness state | Keep planning order explicit: active exec plan first, feature contracts second, tickets third, delivery fourth, with all business logic documented step by step under `docs/features/`, Orchestrator routing the next best role, work starting from `origin/main`, ready commits pushed to `origin main`, and shipped scenarios backed by real E2E/integration evidence. |
| Learning loop | `scores`, `trust`, `docs/QUALITY_SCORE.md`, telemetry triage, skills, guardrails, decisions, evolution reviews | Turn real outcomes into trust changes, intervention work, reusable workflow skills, prompt or process improvements, repo-visible grades, and safety controls. |
| Generated references | `docs/generated/` | Provide reproducible, cataloged source-harness maps when generator commands exist. |
| Release state | `VERSION`, `CHANGELOG.md`, `release notes` | Maintain semantic versions and generated patch notes for both the source harness and target repos. |

Dashboard surface status: the current web dashboard is implemented as the
legacy/current embedded Go, htmx, Chart.js, and SSE surface. The planned
TanStack control-plane dashboard is not implemented yet and is governed by
[dashboard-control-plane.md](dashboard-control-plane.md). It deliberately
changes the dashboard prerequisite model by requiring an external Node.js
`24.x` and `pnpm@11.1.1` sidecar while preserving Go as the authoritative
gateway, auth boundary, and runtime state owner.

## CLI Contract

| Command | Status | Product behavior |
| --- | --- | --- |
| `mars-harness version`, `mars-harness --version`, `mars-harness -v` | Implemented | Prints the installed binary version, OS/architecture, commit, and build date through either the explicit command or root-level version flags. |
| `mars-harness auth github check` / `setup` | Implemented | Checks or prepares private GitHub Release auth for `update tool` without printing token values. The resolver tries `GH_TOKEN`, `GITHUB_TOKEN`, GitHub CLI auth, then the optional local config token. |
| `mars-harness setup` | Implemented, still hardening | Creates `~/.mars-harness/`, checks private-release auth unless `--skip-github` or `--test-mode` is used, configures supported shell profiles for the installed command, writes config, detects hardware, installs llama.cpp server artifacts, downloads pinned models, and keeps optional integration setup explicit. |
| `mars-harness update check --repo <path>` | Implemented | Reports whether the installed CLI, deployed target `.harness/` metadata, or mirrored operating-model artifacts are behind, with JSON output for automation and unknown-but-nonfatal remote status when release lookup fails. |
| `mars-harness update tool` | Implemented | Downloads the latest private GitHub Release platform asset, resolves auth through the getting-started auth model, verifies `checksums.txt`, atomically replaces the installed command, and configures shell PATH. Source-development updates remain available with `--source` or `--version main`. |
| `mars-harness path setup --install-dir <path>` | Implemented | Detects Fish, Zsh, Bash, POSIX sh/Ksh, Csh, or Tcsh and writes an idempotent user-profile snippet so new terminals can resolve `mars-harness`. |
| `mars-harness update harness --repo <path>` | Implemented | Uses the same update verb to refresh the deployed target `.harness/` bundle without overwriting user-owned agent configuration. |
| `make install` from source checkout | Implemented for source development | Installs the dev binary into the Go bin directory and runs the installed binary's PATH setup so operators do not run stale source-root binaries or hand-edit shell config. |
| `mars-harness init --repo <path>` | Implemented | Scaffolds the target harness: manifest, roles, guardrails, knowledge routes, compact `AGENTS.md`, goals, BDD feature contracts, tickets, exec-plan docs, design-doc index, context glossary, quality score, and references. |
| `mars-harness upgrade --repo <path>` | Implemented, still hardening | Fills missing target harness defaults while preserving user-owned manifest, role prompts, knowledge routes, guardrails, tickets, design docs, exec plans, references, and target `AGENTS.md`. |
| `mars-harness eject --repo <path>` | Implemented | Provides the repo-level kill switch. Dry-run is the default; `--apply --confirm <repo-name>` removes `.harness/`, generated harness docs, tickets, feature contracts, `AGENTS.md`, `VERSION`, `CHANGELOG.md`, and the associated per-repo SQLite database without rewriting git history. Aliases: `kill-switch`, `uninstall`. |
| `mars-harness scan --repo <path> --tickets` | Implemented | Finds repo gaps and writes deduplicated backlog tickets through the canonical ticket path. |
| `mars-harness run <role> --repo <path>` | Implemented | Loads manifest, guardrails, knowledge routes, context, tools, local model endpoint, and runs one role with terminal-result truth. Interactive TTYs show a full-screen terminal dashboard by default; `--debug` or legacy `--trace` restores verbose inline trace output, `--log-file` controls the durable command log path, and `--dry-run --no-init` provides observer-safe inspection when a target has no `.harness/` yet. |
| `mars-harness start --repo <path>` | Implemented | Initializes if needed, registers the repo, seeds the CEO role, and runs the per-repo autonomous pipeline with isolated database state and recovery-queue self-healing. Interactive TTYs use the same full-screen terminal dashboard and keep verbose logs in `~/.mars-harness/traces/logs/` unless `--debug` is set. |
| `mars-harness serve` | Implemented | Runs the orchestrator, dashboard, webhook receiver, cron scheduler, workers, recovery-queue self-heal watchdog, native survey loop for unattended ticket/check/telemetry/score/dogfood/no-op/stuck-work signals, and dispatch-mode organization layer against the configured database. Interactive TTYs use the terminal dashboard while the web dashboard and SSE APIs remain separate live surfaces. |
| `mars-harness register --repo <path>` | Implemented | Registers a repo and creates the per-repo database path when one is not supplied. |
| `mars-harness doctor [--repo <path>] [--json]` | Implemented, expanding | Checks Go, config, model registry, models directory, database, llama-server, disk space, private-release auth readiness, guardrail/workflow health, mirrored operating-model health, active-plan hygiene, and optional integration configuration. |
| `mars-harness scores [--repo <path>]` | Implemented | Shows trunk-native role scores from stored outcomes. |
| `mars-harness scores export --repo <path>` | Implemented | Refreshes `docs/QUALITY_SCORE.md` from live score, telemetry, ticket, dogfood, guardrail, check, no-op, and human follow-up evidence while preserving manual notes. Low scores become improvement targets by default; deduped intervention-debt tickets are created only with `--create-intervention-debt` or clearly target-owned evidence. |
| `mars-harness telemetry status\|preview\|export\|send` | Implemented | Keeps raw telemetry local, previews the exact anonymous aggregate payload, writes sanitized reports to the local outbox, and sends only when anonymous reporting is explicitly enabled. |
| `mars-harness telemetry collect --storage sqlite` | Implemented | Runs a local anonymous foundation telemetry collector backed by SQLite; the collector API is designed so hosted Postgres-compatible storage can be added later without changing deployed harnesses. |
| `mars-harness telemetry triage-foundation` | Implemented | Reads collector aggregates and creates Mars Harness source intervention-debt work only for repeated anonymous foundation-owned patterns. |
| `mars-harness trust [--repo <path>]` | Implemented | Shows role trust levels. |
| `mars-harness trust set <role> <repo> <level> --reason <text>` | Implemented | Overrides trust with an audit reason. |
| `mars-harness models list [--provider registry\|ollama]` | Implemented | Lists pinned medium-profile registry defaults or locally installed Ollama models. Ollama listing is a catalog/evaluation surface, not default promotion. |
| `mars-harness models evaluate [--endpoint <url> --model <name>]` | Implemented | Prints the model-refresh plan or runs benchmark probes with tool-call JSON, strict triage JSON, and repo-backed ticket-completion JSON. Live reports include provider, model, endpoint, hardware profile, timing, token counts, failures, promotion status, and are persisted under `docs/generated/model-evaluations/` by default. `--provider ollama --model <name>` targets local Ollama's OpenAI-compatible endpoint. |
| `mars-harness models override --repo <path> (--tier <tier>\|--role <role>) --provider <provider> --model <name>` | Implemented | Writes `.harness/model-overrides.yaml` so a repo can explicitly route one tier or role to an Ollama or OpenAI-compatible model without changing default registry entries. |
| `mars-harness release notes --repo <path> --bump auto` | Implemented | Generates semantic-versioned patch notes from commits, updates `VERSION`, prepends `CHANGELOG.md`, and explains impact, why, and what changed before semantic commit buckets. |
| `mars-harness release verify-assets [--version <tag>]` | Implemented | Fails a release check unless all four platform binaries and `checksums.txt` are attached to the GitHub Release. |

## Generated Target Harness

`mars-harness init` must produce a target repo that is immediately usable by Codex, Cursor, Mars Harness roles, and humans.

Required generated surfaces:

- `AGENTS.md` as the compact first-read map
- `.harness/manifest.yaml` for explicit roles, canonical domain/mode metadata, model tiers, tools, triggers, chains, guardrails, and knowledge routes
- `.harness/metadata.yaml` for generated-harness version drift checks
- `.harness/roles/*.md` for role prompts
- `.harness/skills/*/SKILL.md` for compact reusable workflows and self-improvement guidance
- `.harness/guardrails/*.yaml` for mechanical policy inputs
- `.harness/knowledge/*.yaml` for lightweight context routes
- `docs/goals/` for active goals, observations, and superseded goals
- `docs/features/` for Markdown BDD feature contracts, step-by-step business logic, and scenario schedules
- `docs/tickets/backlog/`, `docs/tickets/in-progress/`, `docs/tickets/in-review/`, and `docs/tickets/done/`
- `docs/tickets/README.md` for ticket lifecycle and completion rules
- `docs/exec-plans/README.md` and starter priority docs with a one-active-plan lifecycle
- `docs/QUALITY_SCORE.md` as the repo-visible A-F quality score artifact
- `docs/design-docs/index.md` and `context-glossary.md`
- `docs/references/README.md` and selected agent-first references
- `VERSION`, `CHANGELOG.md`, and `docs/design-docs/release-versioning.md`

Generated target docs must mirror source-harness doctrine while staying project-agnostic. Existing target harness files and user-owned docs are preserved by upgrades.

Generated target harnesses must also be removable. The supported removal path
is `mars-harness eject --repo <path>`, which previews the repo-local harness
surface and associated per-repo database before deleting anything. The command
removes working-tree traces only; users who want git history rewritten must do
that deliberately outside Mars Harness.

Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only. Any change to remote-trunk freshness, commit discipline, push timing, versioning, ticket flow, documentation rules, skill creation, guardrail policy, trust/scoring behavior, release behavior, or context-routing discipline must update generated target guidance and tests in the same task.

Any `mars-harness` CLI command or flag change must also update the mirrored
`mars_harness_cli` tool reference, repo-shortcut behavior, generated target
guidance, and any affected skills per
[../design-docs/cli-tool-skill-sync.md](../design-docs/cli-tool-skill-sync.md).

Architecture changes and product features must carry rationale in repo-owned docs: what changed, why it changed, and which behavior future agents should preserve. Source harness changes use design docs and product specs; initialized target repos inherit the same rule through generated `AGENTS.md`, using their design docs and, when present, product specs.

Documentation must not go stale. Code written or materially changed by agents
uses a top-of-file `MarsDocSync` metadata block to list the docs that describe
or constrain the behavior, and those docs are updated in the same change or
explicitly checked as still current.

Exec plans mirror the ticket lifecycle. Exactly one plan may be active at a
time. Waiting plans live in `docs/exec-plans/backlog/` with explicit priority,
dependencies, blockers, related tickets, goals, BDD feature contracts,
hypotheses, success/falsification evidence, scenario schedules, current failing
scenarios, walking skeleton slices, and learning or MVP outcomes. Superseded
plans are lineage only.

BDD feature contracts are the source of truth for feature completeness and
business behavior. Product rules, workflow branches, state transitions,
validations, permissions, scoring/trust behavior, routing behavior, release
classification, and user-visible outcomes must be documented step by step under
`docs/features/`, not only in tickets or code comments. Walking skeleton is the
implementation strategy: agents implement the next failing scenario through the
thinnest real end-to-end path. Feature tickets must carry BDD scenario evidence
before done. Enabler work is allowed, but release notes and quality score must
separate enabler work from shipped feature scenarios.

## Versioning And Patch Notes

Mars Harness and initialized target repos use the same release contract:

- `VERSION` contains `MAJOR.MINOR.PATCH`
- `CHANGELOG.md` contains generated patch notes
- generated release-note entries explain `Impact`, `Why`, and `What Changed`
  before listing semantic commit buckets
- `mars-harness release notes --repo . --bump auto` infers the next version from semantic commits
- release-note commits themselves are ignored in the next generated entry
- generated entries include a marker so tags are useful but not required for the next diff
- in the source harness repo and initialized target repos, every non-release semantic commit is immediately followed by the generated version/patch-note commit before the task is done
- when authenticated GitHub release capability is configured, the generated version is tagged as `vX.Y.Z`; the source harness Release workflow publishes or backfills the matching changelog entry plus checksum-verified binary assets

## Generated Source References

`docs/generated/` is reserved for reproducible reference snapshots generated from the source harness. It is intentionally catalog-only until generator commands exist.

Expected future artifacts include role registry, tool inventory, package map, model inventory, and bundle schema reference. Generated docs must name their generator, source inputs, and freshness signal so agents can trust them as context routes.

Model evaluation reports are generated evidence artifacts, not hand-written reference docs. `mars-harness models evaluate` writes JSON under `docs/generated/model-evaluations/` for benchmark and promotion review; operators may commit selected reports when they support a default-model decision.

## Role And Work Semantics

Default roles are configurable starter agents, not perfect built-ins. The target repo owner can edit prompts, add or remove roles, change schedules and chains, restrict tools, attach guardrails, and route context differently through `.harness/manifest.yaml`.

The product contract is:

- Six canonical operating domains describe role memory and routing vocabulary: Planner, Engineer, Reviewer, Maintainer, End-to-End Tester, and Orchestrator.
- Explicit manifest role keys remain the executable units; optional `domain` and `mode` metadata classify why the role runs without changing trust, scoring, tool, or guardrail policy.
- `orchestration_mode: dispatch` is the generated default: completed jobs record dispositions, deterministic handoffs route directly, and Orchestrator handles ambiguous or governance-heavy follow-up while keeping the queue, tickets, BDD evidence, scoring, traces, and trust policy as the runtime backbone.
- `orchestration_mode: legacy` remains supported for repos that deliberately preserve manifest `then` and `idle_then` chains.
- Planner roles have explicit ownership boundaries.
- CEO owns vision, active goals, tradeoffs, and final strategy/scope decisions.
- COO owns active exec plans, BDD feature contracts, scenario schedules, and current failing scenarios.
- CTO owns architecture fit, technical decomposition, and implementation tickets for the current failing BDD scenario or scenario group.
- Engineer roles complete one ticket per run.
- Engineer roles provide scenario evidence before closing feature tickets.
- In-progress tickets are highest priority.
- Target-owned intervention-debt tickets are generated only from clearly target-owned repeated local telemetry failures, explicit operator requests, or opt-in score export; they do not outrank ordinary product backlog unless a product ticket explicitly names them in `blocked_by`. Foundation-owned failures stay local or flow through optional anonymous foundation telemetry.
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

When scores or telemetry show repeated workflow confusion, the harness records improvement targets first, then chooses the bounded improvement surface. Prefer a scoped skill over bloating a role prompt. Use guardrails for non-negotiable enforcement and tools for deterministic actions. Intervention-debt tickets require target ownership or explicit operator opt-in.

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

Local telemetry and dashboard evidence remain the default. Anonymous foundation
telemetry is a separate opt-in integration: deployed harnesses keep raw events
in their repo-specific SQLite database, derive sanitized aggregate reports into a
local outbox, and send only allowlisted fields to a configured collector. Local
dogfood uses the built-in SQLite collector; public operation can later host the
same collector against a Postgres-compatible backend such as Neon.

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
- planned TanStack dashboard control plane with local-admin auth, nonblocking
  APIs, adaptive previews, feedback, roster/model proposals, and
  GitHub-derived DORA metrics
