# Mars Harness — Agent Guide

> First file any agent reads. Fits in a single context window by design.

## What is Mars Harness?

A self-hosted autonomous AI delivery system written in Go. You provide a machine with a GPU, run `mars-harness setup`, and it autonomously manages your development pipeline: CI diagnosis and repair, code generation from tickets, trunk checks and review, release management, documentation maintenance. All inference runs locally on open models (Gemma 4, Qwen3-Coder-Next). No cloud API costs, no data exfiltration, no vendor lock-in.

**Lineage:** Mars Harness is an evolution of the [Mars](https://github.com/elliottgreaves/mars) monorepo's Cursor Automations pipeline. Mars proved the model works (11 autonomous roles). This product extracts that into a standalone, self-hosted system.

**Tech stack:** Go, SQLite, llama.cpp (managed as subprocess), htmx + Chart.js (embedded dashboard).

## Harness Glossary

These definitions are first-class harness context. They apply to this
foundation harness and to deployed harnesses generated into target projects.
Expand the glossary when repeated language, distinctions, or routing rules
would otherwise live only in chat.

- **mars-harness** — this repo: a software factory containing an AI harness, agent orchestration platform, CLI, local inference management, queue, telemetry, scoring, trust, dashboard, scanner, release tooling, and generated target harness defaults.
- **Harness** — extensive organized documentation for how an LLM should operate within the scope of a given directory.
- **Harness definitions** — individual pieces of documentation contained within the harness.
- **Foundation harness** — the harness consumed by `mars-harness` in this source repo.
- **Deployed harness** — the harness consumed by the target application being built by `mars-harness`.
- **Mirrored harness definitions** — harness definitions included in both the foundation harness and deployed harnesses.
- **Operating model** — the documented way a harness turns intent into shipped, verifiable work: goals, BDD contracts, active plans, ticket flow, quality evidence, release discipline, context routing, trust/autonomy behavior, and self-improvement loops.
- **BDD feature contract** — a Markdown feature artifact in `docs/features/` that defines feature completeness, business logic, step-by-step behavior, scenarios, and evidence.
- **Business logic** — product rules, workflow branches, state transitions, validations, permissions, scoring/trust behavior, routing rules, release classification, and user-visible outcomes; business logic is documented step by step in BDD feature contracts before or alongside implementation.
- **No stale documentation** — all durable docs are updated as behavior changes; code carries top-of-file `MarsDocSync` metadata with a `docs` array listing associated documentation so reviewers and automation know which docs must be checked.
- **MarsDocSync block** — a top-of-file code comment block beginning with `MarsDocSync:` and containing a `docs:` list of repo-relative documentation paths, usually feature contracts, design docs, product specs, ticket guidance, or README surfaces touched by that code.
- **Canonical operating domain** — one of the six stable role-memory groups: Planner, Engineer, Reviewer, Maintainer, End-to-End Tester, or Orchestrator.
- **Role mode** — a lower-kebab-case purpose inside a domain that explains why an explicit manifest role is running, such as `ticket-delivery`, `quality-review`, or `pipeline-repair`.
- **Role registry** — a checked inventory of manifest roles, domains, modes, triggers, tools, trust, guardrails, model routing, scoring signals, and escalation behavior.
- **Foundation operating model** — the operating model for `mars-harness` itself, governing how the software factory evolves, validates changes, versions releases, and mirrors doctrine into deployed harnesses.
- **Foundation maintainer role** — the source-only `foundation-maintainer` role for agents maintaining this software factory; it is manual/operator-invoked, consumes the foundation operating model, classifies foundation/deployed ownership before changes, and is not mirrored into deployed harnesses.
- **Deployed operating model** — the operating model inside a target application harness, governing how agents build that target while inheriting mirrored foundation doctrine unless local project policy deliberately overrides it.
- **Symbiotic operating-model change** — a change to operating doctrine that fits the existing closed loop without handoff gaps, duplicate sources of truth, or inconsistencies with adjacent workflows.
- **Live demo improvement loop** — the foundation stabilization loop for lifecycle work: run a clean representative target from the validation matrix, review findings, implement one or two bounded source actions, rerun, and claim improvement only after rerun evidence is confirmed, merged or fast-forwarded to trunk, and pushed to the remote.
- **Failure ownership classification** — the universal operating-model step that classifies every observed failure as foundation-owned, deployed-owned, or mixed/unclear before creating tickets or fixes; foundation-owned fixes belong in `mars-harness` source/runtime/generated doctrine and should benefit all applicable users, while deployed-owned fixes belong in the target repo and should improve that deployed harness or product.
- **Foundation-owned failure** — a failure caused by Mars Harness runtime, orchestration, role guidance, tool policy, generated defaults, model/provider behavior, telemetry, release/update, or mirrored doctrine; record it as foundation evidence or a source ticket instead of converting it into target product backlog.
- **Deployed-owned failure** — a failure caused by target product behavior, target architecture, local build/test setup, target docs, target skills, or project-specific policy; fix it inside the deployed harness or target project and mirror only reusable doctrine back to the foundation.
- **Conversation system record** — significant agent conversations are inputs that must become durable repo artifacts when they change plans, decisions, investigations, quality findings, or completed-work state; chat summaries cannot replace the owning artifact.
- **Tools** — capabilities of AI models to connect with external software, APIs, and systems to perform actions, retrieve current data, and execute complex, multi-step tasks.
- **Mirrored tools** — tools found in both the foundation harness and deployed harness. The mirrored built-in set includes `file_read`, `file_write`, `file_search`, `shell_exec`, `mars_harness_cli`, `grep`, `workspace_hygiene`, `github_auth_check`, `dependency_sync`, `record_decision`, `ticket_create`, `tool_create`, `persona_create`, `docsync_audit`, release/status/audit workflow tools, and git tools.
- **Universal tool surface** — the mirrored Mars Harness tool registry exposed through agent role allowlists, `mars-harness tools run`, and `mars-harness mcp serve` so any MCP-compatible client or local harness agent can use the same tools without depending on a model provider.
- **Formalized tool creation trigger** — repeated, risky, validation-heavy, or likely-to-recur processes should become first-class tools instead of staying as chat memory or ad hoc shell steps.
- **Tool creation path** — new built-in tools must originate through `tool_create`; bypassing it requires a prior `record_decision` entry and design-doc rationale.
- **Meta tool** — a tool that creates, updates, inventories, or validates other tools or tool definitions.
- **Skills** — compact reusable workflow instructions stored in `.harness/skills/<name>/SKILL.md` that teach agents how to perform recurring procedures; skills guide behavior but do not grant tool authority.
- **Universal skills** — skills intentionally mirrored between the foundation harness and deployed harnesses because they encode reusable Mars Harness operating doctrine.
- **Foundation skills** — skills used by agents operating on `mars-harness` itself to evolve, validate, release, or maintain the software factory.
- **Deployed skills** — skills stored in a target project's `.harness/skills/` directory and used by that deployed harness to capture project-specific reusable procedures.
- **Vendor-neutral foundation adapter** — a thin AI-client instruction file that points an external agent at `AGENTS.md` and `docs/roles/personas/foundation-maintainer.md` without becoming an independent source of doctrine.
- **CLI tool/skill sync** — foundational operating rule that any `mars-harness` CLI change must update the mirrored `mars_harness_cli` tool reference, repo-shortcut map, generated target guidance, and any skills that name the affected CLI workflow.
- **Tenets** — foundational rules both the foundation and deployed harness should follow at all times.
- **First-class harness definition** — context that should always be included in the top-level `AGENTS.md`.
- **Contextual harness definition** — situational context routed through the harness glossary with the form: `When doing X include this: <path to document.md>`.
- **Target project** — the repository that `mars-harness` is building, testing, and managing. When you run `mars-harness start --repo /path/to/wave-shooter`, `mars-harness` is the software factory and `wave-shooter` is the target project.

Full glossary: [docs/design-docs/harness-glossary.md](docs/design-docs/harness-glossary.md)
Tools glossary: [docs/design-docs/tools-glossary.md](docs/design-docs/tools-glossary.md)
Role model: [docs/design-docs/harness-operating-model.md](docs/design-docs/harness-operating-model.md)
Role registry: [docs/roles/ROLES.md](docs/roles/ROLES.md)
Documentation sync architecture: [docs/design-docs/documentation-sync-architecture.md](docs/design-docs/documentation-sync-architecture.md)
CLI tool/skill sync: [docs/design-docs/cli-tool-skill-sync.md](docs/design-docs/cli-tool-skill-sync.md)

Agents always operate on a target project. The harness is never the target of its own agents (no self-modification during runs).

## Foundation Mode For AI Clients

The foundation operating model must work with any capable AI coding client.
`AGENTS.md` and `docs/roles/personas/foundation-maintainer.md` are the source
of truth; vendor files are compatibility adapters only.

| Client | Instruction surface | Foundation mode entry |
| --- | --- | --- |
| Claude Code (recommended) | `CLAUDE.md` imports this file and the foundation role packet. | Read `CLAUDE.md`, then work as `foundation-maintainer`. |
| Cursor | Root `AGENTS.md` plus thin `.cursor/rules/*.mdc` adapters. | Read this file and the foundation role packet. |
| Gemini CLI | `GEMINI.md` points to this file and the foundation role packet. | Read `GEMINI.md`, then the referenced canonical docs. |
| Windsurf | Root `AGENTS.md`. | Use this file as the always-on workspace rule. |
| OpenCode | Root `AGENTS.md`. | Use this file as the project rules file. |
| GitHub Copilot | `.github/copilot-instructions.md` plus agent instructions where supported. | Read the Copilot adapter, then the canonical docs it names. |
| Kiro IDE & CLI | Root `AGENTS.md`. | Use this file as steering for the workspace. |
| Codex / Other Agents | Root `AGENTS.md`. | Read this file first, then the foundation role packet for source work. |

When changing the foundation harness, every client must classify findings as
foundation-owned, deployed-owned, mirrored doctrine, or evidence-only before
creating tickets or patches. Client-specific files must not carry independent
operating doctrine.

## The Nine Tenets

Every decision in this project is filtered through these tenets (priority order):

1. **Plug and Play** — zero to running in one command; extends to full lifecycle
2. **Self-Improving System** — evolves from human interventions and its own failures
3. **Accuracy and Value Scoring** — per-role health scores from real outcomes
4. **Customisable Guardrails** — user-defined rules enforced during execution
5. **Roadmap from Init** — tickets and backlog deployed on day one
6. **Blast Radius Containment** — never cause irreversible damage
7. **Execution Truth and Transparency** — auditable, attributable, everything in git
8. **Progressive Autonomy** — earn trust, graduate from observer to autonomous
9. **Context Efficiency** — minimal context assembly, retrieval over stuffing

Full text: [docs/design-docs/tenets.md](docs/design-docs/tenets.md)

## Directory Structure

```
mars-harness/
├── cmd/mars-harness/          CLI entry point (main.go)
├── internal/                   All internal packages
│   ├── agent/                  Agent runtime (conversation loop)
│   ├── bundle/                 Bundle reader and resolver
│   ├── context/                Context assembly engine
│   ├── dashboard/              Built-in web dashboard
│   ├── evolution/              Self-improvement (intervention detector, Reviewer)
│   ├── github/                 GitHub App client and webhook receiver
│   ├── guardrails/             Guardrails engine
│   ├── hardware/               GPU detection and hardware profiles
│   ├── inference/              llama.cpp server management
│   ├── llm/                    LLM client (OpenAI-compatible) and router
│   ├── models/                 Model registry and download
│   ├── queue/                  SQLite job queue
│   ├── release/                Semantic version and patch-note generation
│   ├── safety/                 Blast radius, emergency stop
│   ├── sandbox/                Process sandbox
│   ├── scanner/                Repo scanner for starter backlog
│   ├── scheduler/              Cron scheduler
│   ├── scoring/                Accuracy and value scoring
│   ├── tools/                  Tool registry and core tools
│   ├── trace/                  Execution trace recorder
│   ├── trust/                  Progressive autonomy manager
│   └── ui/                     Terminal trace output
├── docs/
│   ├── design-docs/            Architectural decisions (see index.md)
│   ├── exec-plans/             Active and completed plans, trackers
│   ├── tickets/                Work items (backlog, in-progress, done)
│   ├── product-specs/          Product vision and specifications
│   ├── references/             Research findings and external sources
│   └── generated/              Reproducible generated reference docs (catalog-only until generators exist)
├── examples/
│   └── sample-bundle/          Example .harness/ bundle
├── .cursor/rules/              Agent governance rules
├── Makefile                    Local build, check, and dogfood gates
├── AGENTS.md                   This file
├── ARCHITECTURE.md             System architecture
├── CONTRIBUTING.md             Contributor guide
├── README.md                   Project overview
└── LICENSE                     Apache 2.0
```

## Key Constraints

1. **Single binary distribution.** The Go binary must cross-compile without CGO. llama.cpp is managed as a subprocess, not embedded.
2. **No external dependencies at runtime.** SQLite is embedded. The dashboard is server-rendered HTML with embedded static assets. No Postgres, no Redis, no npm, no Grafana.
3. **Errors must be actionable.** Every error message states what went wrong and provides a concrete remediation command. No cryptic codes.
4. **Tests alongside code.** Every new function gets a test in the same commit. Minimum 70% coverage for non-trivial packages.
5. **Architecture and product changes are recorded.** Any non-trivial architecture change or product feature goes in `docs/design-docs/` or `docs/product-specs/` with the reason why, and design decisions are indexed in `docs/design-docs/index.md`.
6. **Commit after every step.** When executing the delivery schedule, commit after each completed task referencing the milestone and step number.
7. **Trunk-based development.** All commits go directly to `main`. Do not use branch-based review as the default delivery path. This keeps flows simple for autonomous agents. Use semantic (conventional) commit messages: `feat(scope): ...`, `fix(scope): ...`, `docs: ...`, `chore: ...`, `test: ...`, `refactor: ...`.
8. **The repo is the system of record.** Decisions, discoveries, investigations, quality evidence, plans, and completed-work state live in durable repo artifacts, not in chat threads.
9. **Start from remote trunk.** Before non-trivial work in a repo with `origin/main`, fetch `origin main` and ensure local `main` is at or fast-forwarded to `origin/main` before editing. If the worktree is dirty, history diverged, or the remote is unavailable, stop and record the blocker unless the user explicitly asks for offline/local-only work.
10. **Always commit and push both repos.** When making changes to the harness and/or the target project, document, commit, and push changes in both. The harness and target project are separate git repositories — neither should have dangling uncommitted work at the end of a task.
11. **Push ready work immediately.** After a semantic commit, release-note commit, or required tag is ready and validated, push it to `origin main` or the version tag before starting unrelated work. A rejected push means fetch, rebase or resolve deliberately, rerun relevant checks, and push before moving on.
12. **Version every source and target change.** Every non-release semantic commit in this repo must be followed by `mars-harness release notes --repo . --bump auto`, `mars-harness release backfill-notes --repo . --check`, then a `release: notes X.Y.Z` commit before the task is considered done. If the backfill check reports legacy entries, run `mars-harness release backfill-notes --repo .` and include those changelog corrections in the release-note commit. Initialized target repos receive the same rule through generated harness docs. Release-note commits are exempt and are ignored by the generator.
13. **Operating rules mirror to targets.** Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only. When adding or changing a rule, update the generated target guidance and tests in the same task.
14. **Publish versioned release assets locally first.** After a release-note commit is pushed, create or update tag `vX.Y.Z` at the release-note commit, run `mars-harness release publish-assets --repo . --version vX.Y.Z --upload auto`, and verify the local dist with `mars-harness release verify-assets --dist dist/releases --version vX.Y.Z`. Repos with authenticated GitHub release capability may mirror the same assets to GitHub Releases, but GitHub Actions are not required infrastructure. If GitHub mirroring or asset verification is unavailable, record the blocker explicitly instead of treating release work as complete.
15. **Foundation changes validate on a clean project.** Any foundation-owned runtime change (orchestration, queue, agent loop, tool policy, inference, scanner/bootstrap behavior, scoring, trust, guardrails, dashboard, release/update) must be verified by running the installed binary against a clean validation target per [foundation-operating-model.md](docs/design-docs/foundation-operating-model.md) and [validation-matrix-gating.md](docs/design-docs/validation-matrix-gating.md). `make check` and unit tests are necessary but not sufficient for lifecycle claims. Stop wedged replays once diagnosed; do not wait on drain.

## Database Isolation

Each repo gets its own SQLite database to prevent cross-project contamination:

- **`start` and `register`**: default DB path is `~/.mars-harness/db/{repo-name}/mars.db`
- **`serve`**: uses the shared legacy path `~/.mars-harness/db/mars.db` (designed for multi-repo orchestration)
- **`doctor --repo`**: checks the per-repo database for the specified repository
- **`--db` flag**: explicit override on any command, takes precedence over defaults

This ensures queue, telemetry, scheduling, and repo registry are physically isolated per project. See AD-029 in `docs/design-docs/dogfood-and-decisions.md`.

## Operations

Mars Harness is controllable by any AI agent via CLI commands. These are the core operations.

### 1. Setup

First-time install: private release auth, hardware detection, llama-server binary, model download.

```bash
mars-harness auth github setup
mars-harness setup
```

Flags: `--skip-download`, `--skip-github`, `--github`, `--test-mode`, `--dry-run`

Private release auth is part of Getting Started because `update tool` reads
private GitHub Release assets. The resolver tries `GH_TOKEN`, `GITHUB_TOKEN`,
GitHub CLI auth from `gh auth token`, then the optional local config token. Use
`mars-harness auth github check` or the `github_auth_check` tool before update,
release verification, install repair, or version-drift remediation. Never paste
token values into chat, docs, commits, traces, tickets, logs, or tool output.

### 2. Serve

Start the autonomous orchestrator (webhooks, cron, queue, workers).

```bash
mars-harness serve
```

Flags: `--addr :9091`, `--concurrency 2`, `--db <path>`

Health check: `curl http://localhost:9091/healthz` → `{"status":"healthy"}`

### 3. Register

Register a repository for autonomous management.

```bash
mars-harness register --repo /path/to/repo --remote owner/repo
```

If `.harness/manifest.yaml` is missing, `register`, `run`, `scan`, and `start` run the same scaffold as `mars-harness init` automatically (the repo must be a git checkout).

### 4. Status

Health check for GPU, models, config, and database.

```bash
mars-harness doctor
mars-harness doctor --repo /path/to/repo   # check per-repo database
mars-harness doctor --json
```

### 5. Upgrade

Fill in missing target harness defaults after upgrading `mars-harness`. Starter roles are configurable by the target repo owner, so upgrade preserves existing `manifest.yaml`, role prompts, knowledge routes, guardrails, target `AGENTS.md`, tickets, exec plans, design docs, and references.

```bash
mars-harness upgrade --repo /path/to/repo
```

This writes only missing default files. To adopt changed starter prompts, compare the new defaults from a fresh temporary `mars-harness init` and apply the parts you want deliberately.

### 6. Eject

Remove Mars Harness from a target repository with a dry-run kill switch. The
apply path removes `.harness/`, generated harness docs, tickets, feature
contracts, root generated guidance/version files, and the associated per-repo
SQLite database. It does not rewrite git history.

```bash
mars-harness eject --repo /path/to/repo
mars-harness eject --repo /path/to/repo --apply --confirm repo
```

Aliases: `kill-switch`, `uninstall`.

### 7. Run

Manually execute a single agent role against a repository.

```bash
mars-harness run <role> --repo /path/to/repo
mars-harness run engineer --repo . --dry-run   # preview system prompt
mars-harness run engineer --repo /path/to/legacy-repo --dry-run --no-init   # observer-safe missing-harness check
mars-harness run foundation-maintainer --repo . --dry-run --no-init   # preview source-only foundation context
```

Flags: `--model-endpoint`, `--trace`, `--dry-run`, `--no-init`, `--budget`, `--max-turns`

### 8. Universal Tools

Expose the same registered Mars Harness tool surface to operators and external
LLM clients.

```bash
mars-harness tools list --json
mars-harness tools run tool_creation_guard --repo . --args-json '{"tool_name":"tool_creation_guard"}'
mars-harness mcp serve --repo /path/to/repo --trust observer
```

Use `mcp serve` for any MCP-compatible client or local harness agent.
Use `--trust contributor` only when the client should be allowed to call
mutating tools such as `tool_create`, `file_write`, or `record_decision`.

### 9. Release Notes

Generate semantic-versioned patch notes from commits and update `VERSION` plus `CHANGELOG.md`.
Generated release notes must explain the complete `Impact`, `Why`, and
`What Changed` before listing semantic commit buckets.

```bash
mars-harness release notes --repo . --bump auto
mars-harness release notes --repo . --bump auto --dry-run
```

This same command is generated into target repo release guidance by `mars-harness init`.

Backfill historical changelog entries after release-note standards change:

```bash
mars-harness release backfill-notes --repo . --dry-run
mars-harness release backfill-notes --repo .
mars-harness release backfill-notes --repo . --check
```

For this source repo, run it automatically after every non-release semantic commit, commit the generated `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go` changes with `release: notes X.Y.Z`, then push `main`.

After pushing the release-note commit, create or update tag `vX.Y.Z` at that commit, run `mars-harness release publish-assets --repo . --version vX.Y.Z --upload auto`, and verify the local dist with `mars-harness release verify-assets --dist dist/releases --version vX.Y.Z`. If GitHub mirroring is configured, confirm `gh release view vX.Y.Z` succeeds after the upload. If publishing or asset verification is blocked by missing credentials, missing remote, local build failure, or API failure, record that blocker before ending the task.

### Interactive Controls

When `serve` or `start` is running, the terminal provides interactive key bindings and a status bar:

| Key | Action |
|-----|--------|
| `p` | Pause / Resume (toggle). Paused pool finishes running jobs but claims no new ones. |
| `r` | Warm restart — stops workers, reloads manifests and triggers, restarts workers. HTTP stays up. |
| `s` | Re-scan all registered repos for findings (generates tickets). |
| `q` | Graceful stop (same as Ctrl+C). |
| `h` | Print key binding help. |

The same controls are available via the web dashboard at `http://localhost:9090` (control bar in the sidebar) and as HTTP API endpoints:

```
POST /api/pause          POST /api/resume
POST /api/restart        POST /api/stop
POST /api/scan           POST /api/run-role
GET  /api/status         GET  /api/repos
GET  /api/repo-roles
```

See [docs/design-docs/dashboard.md](docs/design-docs/dashboard.md) AD-027 for full details.

## How to Build

```bash
go build ./cmd/mars-harness
```

For day-to-day source development, install the command into the Go bin directory and run it from anywhere. `make install` also configures the user's supported shell profile so new terminals can resolve `mars-harness` without manual PATH edits:

```bash
make install
mars-harness start --repo /path/to/target-repo
```

Upgrade or reinstall the installed command without changing into this repo:

```bash
mars-harness update check --repo /path/to/target-repo
mars-harness auth github check
mars-harness update tool
mars-harness update harness --repo /path/to/target-repo
```

If shell PATH setup needs to be repaired directly, run:

```bash
mars-harness path setup
```

Do not use `go build ./cmd/mars-harness; ./mars-harness ...` as the normal loop. The semicolon runs the old source-root binary when the build fails. Use `make install` or `go build -o build/mars-harness ./cmd/mars-harness` for one-off binaries.

## How to Test

```bash
go test ./...                    # All tests
go test ./internal/agent/...     # Specific package
go test -cover ./...             # With coverage
```

## How to Lint

```bash
golangci-lint run
```

## Working Discipline

1. **Refresh from remote trunk first.** Run `git fetch origin main` before non-trivial work and make sure local `main` is at or fast-forwarded to `origin/main` before editing. If local work, divergence, or remote failure prevents that, record the blocker and next action instead of building on stale state.
2. **Commit after every step.** Each completed task gets a commit referencing the milestone and step (e.g., `feat(agent): implement conversation loop (M1.3.1)`).
3. **Push ready commits immediately.** Push each validated semantic commit and its follow-up release-note commit to `origin main` before taking unrelated work. If push is rejected, fetch, rebase or resolve intentionally, rerun relevant checks, and push before proceeding.
4. **Version after non-release commits.** Immediately after a semantic change commit, run `mars-harness release notes --repo . --bump auto`, verify, run `mars-harness release backfill-notes --repo . --check`, apply any required backfill, commit the release-note update, and push. Do not generate another version for the `release: notes` commit itself.
5. **Publish release assets locally first.** Once the release-note commit is pushed, tag `vX.Y.Z` at that commit, run `mars-harness release publish-assets --repo . --version vX.Y.Z --upload auto`, and verify assets with `mars-harness release verify-assets --dist dist/releases --version vX.Y.Z`. Missing GitHub mirror capability or failed asset verification is a blocker to record, not something to silently ignore.
6. **Document every decision and feature.** Architecture changes and product features go in `docs/design-docs/` or `docs/product-specs/` with the reason why. Discoveries go in the relevant design doc's Discoveries section.
7. **No stale documentation.** All docs are live system artifacts. When writing or materially changing code, add or update the top-of-file `MarsDocSync` comment block with a `docs:` list of associated docs, run or satisfy the docsync audit where applicable, then update those docs in the same commit or record why no doc change was needed.
8. **BDD defines done.** Goals align the active plan, BDD feature contracts define feature completeness, and walking skeleton slices implement the next failing scenario through real E2E/integration evidence.
9. **Business logic is first-class BDD.** Every product rule, workflow branch, state transition, validation, permission, scoring/trust rule, routing rule, release classification, or user-visible outcome must be documented step by step in `docs/features/` before or alongside implementation.
10. **Source improvements use the live demo loop.** Read [foundation-operating-model.md](docs/design-docs/foundation-operating-model.md) first. When changing Mars Harness lifecycle, orchestration, generated target scaffolding, intervention-debt routing, model/provider behavior, dashboard/control-plane behavior, scoring, safety/guardrails, or update/release behavior, run the installed binary against a **clean validation project** from the AD-284 matrix, review findings, implement one or two bounded actions, rerun on a clean seed, and claim improvement only after rerun evidence confirms the fix and the work is merged or fast-forwarded to trunk and pushed to the remote. If the loop cannot run, record the exact blocker and replay command. Stop known-wedge replays immediately; do not monitor until timeout.
11. **Quality is not a phase.** Tests are written alongside code. Every milestone has a quality gate.
12. **Feed conversations back.** Significant conversations must update the owning repo artifact in the same direct commit to `main`: plans, tickets, design docs, product specs, investigation notes, quality evidence, or release evidence as applicable. Chat summaries cannot replace those artifacts.
13. **Avoid docs churn for trivial replies.** Simple command answers, restatements of existing docs, and explicitly throwaway experiments do not need new artifacts unless they later justify a decision, investigation, quality claim, or completion claim.
14. **Use the ticket lifecycle directories.** New tickets are created with `ticket_create` in `docs/tickets/backlog/`. Do not hand-write ticket markdown directly under `docs/tickets/`; ticket files belong only in `backlog/`, `in-progress/`, `in-review/`, or `done/`.
15. **Keep CLI tools and skills synchronized.** Whenever `cmd/mars-harness` changes a command, flag, output contract, repo behavior, or workflow, update the `mars_harness_cli` reference and repo-shortcut map, generated target doctrine, and any affected skills in the same change. Run the CLI sync tests named in `docs/design-docs/cli-tool-skill-sync.md`.

## Pointers

- **Current operating plan:** [docs/exec-plans/active/current-operating-plan.md](docs/exec-plans/active/current-operating-plan.md)
- **Tenets:** [docs/design-docs/tenets.md](docs/design-docs/tenets.md)
- **Architecture decisions:** [docs/design-docs/index.md](docs/design-docs/index.md)
- **Foundation operating model (clean-project validation):** [docs/design-docs/foundation-operating-model.md](docs/design-docs/foundation-operating-model.md)
- **Conversation record discipline:** [docs/design-docs/conversation-as-system-record.md](docs/design-docs/conversation-as-system-record.md)
- **Product vision:** [docs/product-specs/vision.md](docs/product-specs/vision.md)
- **Quality score:** [docs/QUALITY_SCORE.md](docs/QUALITY_SCORE.md)
- **Goals:** [docs/goals/active.md](docs/goals/active.md)
- **BDD feature contracts:** [docs/features/README.md](docs/features/README.md)
- **Role registry:** [docs/roles/ROLES.md](docs/roles/ROLES.md)
- **Model research:** [docs/references/model-landscape-may-2026.md](docs/references/model-landscape-may-2026.md)
- **CLI tool/skill sync:** [docs/design-docs/cli-tool-skill-sync.md](docs/design-docs/cli-tool-skill-sync.md)
- **Tech debt:** [docs/exec-plans/tech-debt.md](docs/exec-plans/tech-debt.md)
- **Tickets:** [docs/tickets/](docs/tickets/)
