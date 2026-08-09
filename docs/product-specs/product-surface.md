# Product Surface

**Status:** Accepted
**Updated:** 2026-06-28
**Owner:** MARS maintainers
**Sources:** [AGENTS.md](../../AGENTS.md), [quickstart](../quickstart.html), [design docs](../design-docs/index.md), [vision](vision.md)

## Current Product Shape

MARS is a local autonomous delivery runtime with four visible layers:

| Layer | User-facing surface | Product promise |
| --- | --- | --- |
| Install and setup | installed `mars` command, `mars auth github setup`, `mars setup`, `mars path setup`, config, model and binary cache | Run the harness from any directory, configure private-release auth and provider credential names without exposing secret values, configure the user's shell PATH automatically, detect hardware, install local inference, choose a sensible performance profile, and explain missing prerequisites. |
| Target harness | `mars init`, `upgrade`, generated `AGENTS.md`, `.harness/`, docs, tickets, references | Give every target repo a mirrored agent operating system from day one. |
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

Documentation site status: the repository GitHub Pages site is published from
`/docs`, and `docs/index.html` is the user-facing developer documentation front
door with a dark default theme, stable compact header navigation, and a
reader-first IA governed by
[documentation-site.md](documentation-site.md). The homepage is a trust-building
front door for MARS as a local AI product engineering team that can be
inspected, governed, and improved. It routes readers to safe first actions,
proof questions, security and governance material, and canonical system records
instead of carrying the full documentation catalog. The full map now lives at
`docs/documentation-map.html`; control-boundary evidence is routed through
`docs/security-governance-guide.html`; and safe trial, control review, pilot,
proof, and rollout decisions live at `docs/adoption-guide.html`. On mobile
viewports, the
site uses stable disclosure navigation, keeps page content ahead of the long
section index, and renders wide reference tables as labelled cards instead of
forcing sideways reading. It includes a beginner-safe command chooser that
separates read-only inspection, local machine writes, target-file writes, and
autonomous/agent-mediated actions. It documents
installation, setup, CLI workflows, target harness lifecycle, operations,
models, safety, telemetry, release management, extension, troubleshooting, and
links into repo-owned source docs. First-run onboarding has a dedicated static
guide at `docs/quickstart.html`, covering
requirements, source install, local inference setup, target initialization,
safe inspection, dashboard checks, update paths, file ownership, and
troubleshooting, including the distinction between the dashboard URL and the
webhook/control health URL. Install and setup now has a dedicated reference at
`docs/install-setup-reference.html`, covering source install, shell PATH setup,
`mars setup`, private-release auth, inference mode, local bundles, machine
state, automation, `doctor`, and first-run recovery. The site also includes
dedicated pages for the detailed CLI reference (`docs/cli-reference.html`),
task-oriented user workflows
(`docs/workflows.html`), command-by-command target repository lifecycle
(`docs/target-lifecycle-reference.html`), target harness
structure/customization (`docs/harness-guide.html`), and the exact
bundle/manifest contract (`docs/bundle-reference.html`). The site now also
includes deep user guides for
runtime operations (`docs/operations-guide.html`), models and inference
(`docs/models-guide.html`), and the universal tools/MCP bridge
(`docs/tools-mcp-guide.html`). Dashboard API reference coverage lives in
`docs/dashboard-api-reference.html`, documenting pages, local control
endpoints, request bodies, response shapes, SSE events, errors, and operator
recipes. The site also includes governance and release guides for safety,
trust, quality, telemetry (`docs/safety-quality-guide.html`), release/update
operations (`docs/release-update-guide.html`), and optional integration plus
validation workflows (`docs/integrations-validation-guide.html`). Validation
coverage includes the `mars validation agent-smoke` command surface, suite
selection, report fields, retained-run files, cleanup behavior, failure
classes, and fixture/live evidence boundaries. File and
state ownership is documented in `docs/files-state-reference.html`, covering
target harness files, local runtime state, databases, logs, traces, model
artifacts, secrets, release assets, upgrade preservation, and eject cleanup.
The site now also documents roles and agents (`docs/roles-guide.html`), target
and local configuration (`docs/configuration-reference.html`), and the no-stale-docs
documentation sync operating model (`docs/documentation-sync-guide.html`).
Configuration coverage includes aggregate telemetry YAML keys, environment
overrides, collector URL semantics, token handling, the preview-before-send
privacy boundary, and the fact that the mode named <code>anonymous</code> does
not make collector transport anonymous.
Observability and recovery coverage lives in
`docs/observability-guide.html` and `docs/troubleshooting-guide.html`, covering
dashboard state, terminal status, logs, traces, quality score, telemetry,
code-intelligence metrics, diagnostics, and symptom-driven recovery. The
dedicated code-intelligence reference at `docs/code-intel-reference.html`
documents graph context enablement, code-intel tools, metrics, benchmarks,
check recording, validation boundaries, local state, and evidence recipes. A
plain local-checks guide at `docs/checks-evidence-guide.html` documents when
to run `mars checks run`, what gets stored, how failures behave, and how check
results flow into the quality record. The
delivery operating path lives in `docs/planning-delivery-guide.html`, covering
the user-facing goal, active plan, BDD contract, ticket, role handoff, evidence,
review, and release chain. Guardrail behavior now has a dedicated site
reference at `docs/guardrails-reference.html`, covering YAML rules, hard and
advisory severity, scope, matching semantics, secret scanning, optional hooks,
stale-rule review, overrides, and recovery. The repo also includes a static adoption explainer
under `docs/harness-ecosystem/`; the explainer remains a linked page for the
value case, foundation/deployed boundary, operating model, DocSync, telemetry,
safety, and pilot adoption path for mixed leadership and engineering review.
These static pages remain self-contained with local assets only, so they can be
reviewed from the repo and deployed without a frontend build step.

## CLI Contract

| Command | Status | Product behavior |
| --- | --- | --- |
| `mars version`, `mars --version`, `mars -v` | Implemented | Prints the installed binary version, OS/architecture, commit, and build date through either the explicit command or root-level version flags. |
| `mars auth github check` / `setup` | Implemented | Checks or prepares private GitHub Release auth for `update tool` without printing token values. The resolver tries `GH_TOKEN`, `GITHUB_TOKEN`, GitHub CLI auth, then the optional local config token; setup stores a verified GitHub CLI token as an owner-only local fallback so future updates do not depend on keychain access. |
| `mars setup` | Implemented, still hardening | Creates `~/.mars/`, checks private-release auth unless `--skip-github` or `--test-mode` is used, configures supported shell profiles for the installed command, writes config, detects hardware, installs llama.cpp server artifacts, downloads pinned models, and keeps optional integration setup explicit. |
| `mars update check --repo <path>` | Implemented | Reports whether the installed CLI, deployed target `.harness/` metadata, or mirrored operating-model artifacts are behind, with JSON output for automation and unknown-but-nonfatal remote status when release lookup fails. |
| `mars update tool` | Implemented; packaged availability blocked pending cutover | Release mode resolves optional private access, acquires only the canonical signed archive contract, verifies the offline Sigstore bundle over the exact eight-entry checksum file, immutable tag/full commit, platform/build metadata, archive digest and structure, then durably replaces or restores the fixed installed command before configuring shell PATH. Source-development updates remain available with `--source` or `--version main`; no supported packaged update exists until an approved F-018 release supplies that contract. |
| `mars path setup --install-dir <path>` | Implemented | Detects Fish, Zsh, Bash, POSIX sh/Ksh, Csh, or Tcsh and writes an idempotent user-profile snippet so new terminals can resolve `mars`. User docs pair this with the Shell Integration Reference for profile targets, command-cache repair, and completions. |
| `mars update harness --repo <path>` | Implemented | Uses the same update verb to refresh the deployed target `.harness/` bundle without overwriting user-owned agent configuration. |
| `make install` from source checkout | Implemented for source development | Installs the dev binary into the Go bin directory and runs the installed binary's PATH setup so operators do not run stale source-root binaries or hand-edit shell config. |
| `make update-tool` from source checkout | Implemented for source development | Safely fast-forwards a clean source checkout from `origin/main`, installs the updated command with `go install`, runs shell PATH setup, and prints the installed version. Dirty, missing-origin, and diverged checkouts fail with local remediation. |
| `mars init --repo <path>` | Implemented | Scaffolds the target harness: manifest, roles, guardrails, knowledge routes, compact `AGENTS.md`, goals, BDD feature contracts, tickets, exec-plan docs, design-doc index, context glossary, quality score, and references. |
| `mars upgrade --repo <path>` | Implemented, still hardening | Fills missing target harness defaults while preserving user-owned manifest, role prompts, knowledge routes, guardrails, tickets, design docs, exec plans, references, and target `AGENTS.md`. |
| `mars eject --repo <path>` | Implemented | Provides the repo-level kill switch. Dry-run is the default; `--apply --confirm <repo-name>` removes `.harness/`, generated harness docs, tickets, feature contracts, `AGENTS.md`, `VERSION`, `CHANGELOG.md`, and the associated per-repo SQLite database without rewriting git history. Aliases: `kill-switch`, `uninstall`. |
| `mars scan --repo <path> --tickets` | Implemented | Finds repo gaps and writes deduplicated backlog tickets through the canonical ticket path. |
| `mars run <role> --repo <path>` | Implemented | Loads manifest, guardrails, knowledge routes, context, tools, local model endpoint, and runs one role with terminal-result truth. Execution defaults to observer and independently caps mutating tools; acknowledged host execution requires `--execution-profile host --acknowledge-host-execution`, while `isolated` fails before state because no enforceable adapter exists. Observer run never initializes a missing target; `--dry-run --no-init` retains the explicit no-write missing-harness preview, and `foundation-maintainer` retains source-only preview context. Interactive TTYs show a full-screen terminal dashboard by default; `--debug` or legacy `--trace` restores verbose inline trace output and `--log-file` controls the durable command log path. |
| `mars start --repo <path>` | Implemented | Registers an initialized repo, reconciles existing lifecycle state before seeding CEO, and runs the per-repo pipeline with per-repo database state and recovery-queue self-healing. It defaults to observer, which blocks target writes, direct server mutation lanes, missing-harness initialization, and `--force` repair while retaining owner-local bookkeeping and managed inference. Initialization or autonomous target mutation requires `--execution-profile host --acknowledge-host-execution`; `isolated` is unavailable. `--new-lifecycle` is required for intentional CEO reseeding over resumable work. Interactive TTYs use the same full-screen terminal dashboard and keep verbose logs in `~/.mars/traces/logs/` unless `--debug` is set. |
| `mars serve` | Implemented | Runs the orchestrator, dashboard, optional fail-closed GitHub webhook receiver, cron scheduler, workers, recovery-queue self-heal watchdog, native survey loop, and dispatch-mode organization layer against the configured database. Execution defaults to observer, preserving owner-local state and managed inference while blocking target tool mutation plus direct scanner, Jira, remediation, learning, hygiene, and intervention-ticket writers; acknowledged host mode enables only the authority already allowed by role trust and policy. Control/dashboard listeners are loopback-only. GitHub dispatch requires an >=32-byte env-first/owner-only-setup-fallback secret, trusted numeric actor IDs, and an exact registered repository/branch; missing policy leaves local health available and returns 503 on `/webhook`. |
| `mars register --repo <path>` | Implemented | Registers a repo and creates the per-repo database path when one is not supplied. |
| `mars doctor [--repo <path>] [--json]` | Implemented, expanding | Checks Go, config, model registry, models directory, database, llama-server, disk space, private-release auth readiness, guardrail/workflow health, mirrored operating-model health, active-plan hygiene, and optional integration configuration. |
| `mars scores [--repo <path>]` | Implemented | Shows trunk-native role scores from stored outcomes. |
| `mars scores export --repo <path>` | Implemented | Refreshes `docs/QUALITY_SCORE.md` from live score, telemetry, ticket, dogfood, guardrail, check, no-op, and human follow-up evidence while preserving manual notes. Low scores become improvement targets by default; deduped intervention-debt tickets are created only with `--create-intervention-debt` or clearly target-owned evidence. |
| `mars telemetry status\|preview\|export\|send` | Implemented | Keeps raw telemetry local, previews the exact allowlisted aggregate payload, writes sanitized reports to the local outbox, and sends only when the reporting mode named `anonymous` is explicitly enabled. The label does not make network transport anonymous. |
| `mars telemetry collect --storage sqlite` | Implemented | Runs a literal-loopback-only aggregate collector backed by SQLite. It defaults to `127.0.0.1:9092`, rejects DNS and non-loopback binds before database creation, and admits only bounded JSON requests for the exact local Host. Remote collection is unavailable until a separately authenticated design exists. |
| `mars telemetry triage-foundation` | Implemented | Reads collector aggregates and creates MARS source intervention-debt work only for repeated minimized foundation-owned patterns. |
| `mars trust [--repo <path>]` | Implemented | Shows role trust levels. |
| `mars trust set <role> <repo> <level> --reason <text>` | Implemented | Overrides trust with an audit reason. |
| `mars models list [--provider registry\|ollama]` | Implemented | Lists pinned medium-profile registry defaults or locally installed Ollama models. Ollama listing is a catalog/evaluation surface, not default promotion. |
| `mars models evaluate [--endpoint <url> --model <name>]` | Implemented | Prints the model-refresh plan or runs benchmark probes with tool-call JSON, strict triage JSON, and repo-backed ticket-completion JSON. Live reports include provider, model, endpoint, hardware profile, timing, token counts, failures, promotion status, and are persisted under `docs/generated/model-evaluations/` by default. `--provider ollama --model <name>` targets local Ollama's OpenAI-compatible endpoint. |
| `mars models override --repo <path> (--tier <tier>\|--role <role>) --provider <provider> --model <name>` | Implemented | Writes `.harness/model-overrides.yaml` so a repo can explicitly route one tier or role to an Ollama or OpenAI-compatible model without changing default registry entries. |
| `mars release notes --repo <path> --bump auto` | Implemented | Generates semantic-versioned patch notes from commits, updates `VERSION`, prepends `CHANGELOG.md`, and explains impact, why, and what changed before semantic commit buckets. |
| `mars release publish-assets` | Retired | T-065 removes the bespoke source publisher and GitHub upload path. MARS source now uses pinned, publication-disabled GoReleaser/Syft snapshots under F-018; target repositories choose their own producer. |
| Pinned GoReleaser snapshot workflow | Implemented (private only) | Builds four source archives plus per-archive SPDX SBOMs and an exact checksum set without tag, upload, signing, announcement, or publication authority. |
| Signed archive consumer | Implemented (private transition) | Authenticates the exact checksum bytes and release identity, inspects one bounded canonical archive, and fails closed before replacement on any signature, source, platform, metadata, checksum, archive, or durable-transaction mismatch. Standalone `release verify-assets` and `release audit` commands are retired; source cutover verification and target-repository verification remain repository-owned gates. |

## Optional Board Integrations

Target repos may opt into board-driven intake by copying
`.harness/integrations.example.yaml` to `.harness/integrations.yaml` and setting
`flow_profile: board-driven`. JIRA ingestion is disabled unless explicitly
enabled. The default `rest` provider uses direct Atlassian REST search; the
preferred Example Target Project path can set `provider: atlassian_mcp` to use a job-scoped
Atlassian MCP read session, including an optional stdio proxy such as
`mcp-remote` for OAuth-backed sessions. Both providers require explicit
project-to-repo mapping and enforce configured workspace, board, and
required-label containment before local tickets are written. Tenant-specific
values such as JIRA site URLs, Atlassian cloud IDs, board IDs, ID-bearing
workspace URLs, and JIRA custom field IDs can be supplied through configured
environment variable names so they do not need to be committed to the repo.

## Generated Target Harness

`mars init` must produce a target repo that is immediately usable by Codex, Cursor, MARS roles, and humans.

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
is `mars eject --repo <path>`, which previews the repo-local harness
surface and associated per-repo database before deleting anything. The command
removes working-tree traces only; users who want git history rewritten must do
that deliberately outside MARS.

Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only. Any change to remote-trunk freshness, commit discipline, push timing, versioning, ticket flow, documentation rules, skill creation, guardrail policy, trust/scoring behavior, release behavior, or context-routing discipline must update generated target guidance and tests in the same task.

Any `mars` CLI command or flag change must also update the mirrored
`mars_cli` tool reference, repo-shortcut behavior, generated target
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

MARS and initialized target repos use the same release contract:

- `VERSION` contains `MAJOR.MINOR.PATCH`
- `CHANGELOG.md` contains generated patch notes
- generated release-note entries explain `Impact`, `Why`, and `What Changed`
  before listing semantic commit buckets
- `mars release notes --repo . --bump auto` infers the next version from semantic commits
- release-note commits themselves are ignored in the next generated entry
- generated entries include a marker so tags are useful but not required for the next diff
- in the source harness repo and initialized target repos, every non-release semantic commit is immediately followed by the generated version/patch-note commit before the task is done
- MARS source release production follows the pinned GoReleaser/Syft contract and active F-018 plan; generated target repositories choose their own producer and artifact contract

## Generated Source References

`docs/generated/` is reserved for reproducible reference snapshots generated from the source harness. It is intentionally catalog-only until generator commands exist.

Expected future artifacts include role registry, tool inventory, package map, model inventory, and bundle schema reference. Generated docs must name their generator, source inputs, and freshness signal so agents can trust them as context routes.

Model evaluation reports are generated evidence artifacts, not hand-written reference docs. `mars models evaluate` writes JSON under `docs/generated/model-evaluations/` for benchmark and promotion review; operators may commit selected reports when they support a default-model decision.

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
- Target-owned intervention-debt tickets are generated only from clearly target-owned repeated local telemetry failures, explicit operator requests, or opt-in score export; they do not outrank ordinary product backlog unless a product ticket explicitly names them in `blocked_by`. Foundation-owned failures stay local or flow through optional aggregate foundation telemetry; its minimized payload does not make transport anonymous.
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

`docs/QUALITY_SCORE.md` is refreshed with `mars scores export --repo <path>`. The generated artifact is the quality source of truth for agents; dashboard quality views link to the same file and database-derived signals instead of becoming a separate grading surface. Missing SQLite evidence is explicitly graded as insufficient evidence.

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
- model default changes backed by `mars models evaluate` evidence rather than newest-model claims alone
- broad Ollama model access for evaluation and explicit operator-owned swaps, without treating every available model as a safe zero-config default

Manual tuning remains available, but the normal path should require none.

## Optional Integrations

MARS is complete without a remote-code-host integration. Optional integration exists for telemetry and coordination: webhooks, statuses, comments, and check-run style reporting.

GitHub webhooks are authenticated and authorized inputs, not a public command
surface. `mars start --remote owner/repo --branch <branch>` records the exact
repository boundary. Trusted actor IDs resolve from repeatable
`--webhook-actor-id`, `MARS_WEBHOOK_ALLOWED_ACTOR_IDS`, or
`webhook_allowed_actor_ids` with CLI-over-env-over-YAML precedence. Login names,
empty remotes, wrong branches, forks, and issue comments cannot dispatch work.
`MARS_WEBHOOK_SECRET` overrides the owner-only `0600` GitHub App credentials
fallback created by setup. Neither source is exposed through YAML/CLI, logs,
traces, HTTP responses, or returned setup results.

The product must never describe optional integration as complete unless credentials, webhook delivery, and status/comment behavior have actually been validated.

Local telemetry and dashboard evidence remain the default. Aggregate foundation
telemetry is a separate opt-in integration: deployed harnesses keep raw events
in their repo-specific SQLite database, derive sanitized aggregate reports into
a local outbox, and send only allowlisted fields to a configured collector. The
configuration mode remains named <code>anonymous</code>, but the collector and
network path can observe transport metadata. Local dogfood uses the built-in
SQLite collector; public operation can later host the same collector against a
Postgres-compatible backend such as Neon.

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
