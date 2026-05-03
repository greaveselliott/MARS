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
- **Tools** — capabilities of AI models to connect with external software, APIs, and systems to perform actions, retrieve current data, and execute complex, multi-step tasks.
- **Mirrored tools** — tools found in both the foundation harness and deployed harness. The mirrored built-in set includes `file_read`, `file_write`, `file_search`, `shell_exec`, `grep`, `record_decision`, `ticket_create`, `tool_create`, and git tools.
- **Meta tool** — a tool that creates, updates, inventories, or validates other tools or tool definitions.
- **Tenets** — foundational rules both the foundation and deployed harness should follow at all times.
- **First-class harness definition** — context that should always be included in the top-level `AGENTS.md`.
- **Contextual harness definition** — situational context routed through the harness glossary with the form: `When doing X include this: <path to document.md>`.
- **Target project** — the repository that `mars-harness` is building, testing, and managing. When you run `mars-harness start --repo /path/to/wave-shooter`, `mars-harness` is the software factory and `wave-shooter` is the target project.

Full glossary: [docs/design-docs/harness-glossary.md](docs/design-docs/harness-glossary.md)

Agents always operate on a target project. The harness is never the target of its own agents (no self-modification during runs).

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
├── .github/workflows/          CI workflows
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
8. **The repo is the system of record.** Decisions, discoveries, and plans live in docs, not in chat threads.
9. **Always commit and push both repos.** When making changes to the harness and/or the target project, document, commit, and push changes in both. The harness and target project are separate git repositories — neither should have dangling uncommitted work at the end of a task.
10. **Version every source and target change.** Every non-release semantic commit in this repo must be followed by `mars-harness release notes --repo . --bump auto`, then a `release: notes X.Y.Z` commit before the task is considered done. Initialized target repos receive the same rule through generated harness docs. Release-note commits are exempt and are ignored by the generator.
11. **Operating rules mirror to targets.** Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only. When adding or changing a rule, update the generated target guidance and tests in the same task.
12. **Publish versioned GitHub releases when configured.** After a release-note commit is pushed, repos with authenticated GitHub release capability must publish or update a GitHub Release named `vX.Y.Z` using the generated changelog entry. If GitHub release publication is unavailable, record the blocker explicitly instead of treating release work as complete.

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

First-time install: hardware detection, llama-server binary, model download.

```bash
mars-harness setup
```

Flags: `--skip-download`, `--github`, `--test-mode`, `--dry-run`

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

### 6. Run

Manually execute a single agent role against a repository.

```bash
mars-harness run <role> --repo /path/to/repo
mars-harness run engineer --repo . --dry-run   # preview system prompt
```

Flags: `--model-endpoint`, `--trace`, `--dry-run`, `--budget`, `--max-turns`

### 7. Release Notes

Generate semantic-versioned patch notes from commits and update `VERSION` plus `CHANGELOG.md`.

```bash
mars-harness release notes --repo . --bump auto
mars-harness release notes --repo . --bump auto --dry-run
```

This same command is generated into target repo release guidance by `mars-harness init`.

For this source repo, run it automatically after every non-release semantic commit, commit the generated `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go` changes with `release: notes X.Y.Z`, then push `main`.

After pushing the release-note commit, publish or update GitHub Release `vX.Y.Z` with the generated changelog entry whenever GitHub release credentials are configured. If publishing is blocked by missing credentials, missing remote, or API failure, record that blocker before ending the task.

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

1. **Commit after every step.** Each completed task gets a commit referencing the milestone and step (e.g., `feat(agent): implement conversation loop (M1.3.1)`).
2. **Version after non-release commits.** Immediately after a semantic change commit, run `mars-harness release notes --repo . --bump auto`, verify, commit the release-note update, and push. Do not generate another version for the `release: notes` commit itself.
3. **Publish GitHub release notes when configured.** Once the release-note commit is pushed, publish or update GitHub Release `vX.Y.Z` from the generated changelog entry. Missing GitHub release capability is a blocker to record, not something to silently ignore.
4. **Document every decision and feature.** Architecture changes and product features go in `docs/design-docs/` or `docs/product-specs/` with the reason why. Discoveries go in the relevant design doc's Discoveries section.
5. **BDD defines done.** Goals align the active plan, BDD feature contracts define feature completeness, and walking skeleton slices implement the next failing scenario through real E2E/integration evidence.
6. **Quality is not a phase.** Tests are written alongside code. Every milestone has a quality gate.
7. **Feed conversations back.** Decisions that exist only in chat threads are invisible to future agents. See the delivery schedule for what to build next.

## Pointers

- **Current operating plan:** [docs/exec-plans/active/current-operating-plan.md](docs/exec-plans/active/current-operating-plan.md)
- **Tenets:** [docs/design-docs/tenets.md](docs/design-docs/tenets.md)
- **Architecture decisions:** [docs/design-docs/index.md](docs/design-docs/index.md)
- **Product vision:** [docs/product-specs/vision.md](docs/product-specs/vision.md)
- **Quality score:** [docs/QUALITY_SCORE.md](docs/QUALITY_SCORE.md)
- **Goals:** [docs/goals/active.md](docs/goals/active.md)
- **BDD feature contracts:** [docs/features/README.md](docs/features/README.md)
- **Model research:** [docs/references/model-landscape-may-2026.md](docs/references/model-landscape-may-2026.md)
- **Tech debt:** [docs/exec-plans/tech-debt.md](docs/exec-plans/tech-debt.md)
- **Tickets:** [docs/tickets/](docs/tickets/)
