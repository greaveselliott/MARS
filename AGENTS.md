# Mars Harness — Agent Guide

> First file any agent reads. Fits in a single context window by design.

## What is Mars Harness?

A self-hosted autonomous AI delivery system written in Go. You provide a machine with a GPU, run `mars-harness setup`, and it autonomously manages your development pipeline: CI diagnosis and repair, code generation from tickets, PR review, release management, documentation maintenance. All inference runs locally on open models (Gemma 4, Qwen3-Coder-Next). No cloud API costs, no data exfiltration, no vendor lock-in.

**Lineage:** Mars Harness is an evolution of the [Mars](https://github.com/elliottgreaves/mars) monorepo's Cursor Automations pipeline. Mars proved the model works (11 autonomous roles). This product extracts that into a standalone, self-hosted system.

**Tech stack:** Go, SQLite, llama.cpp (managed as subprocess), htmx + Chart.js (embedded dashboard).

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
│   └── generated/              Auto-generated docs
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
4. **Tests alongside code.** Every new function gets a test in the same PR. Minimum 70% coverage for non-trivial packages.
5. **Architecture decisions are recorded.** Any non-trivial decision goes in `docs/design-docs/` and is indexed in `docs/design-docs/index.md`.
6. **Commit after every step.** When executing the delivery schedule, commit after each completed task referencing the milestone and step number.
7. **Never push to main.** All changes go through a branch and PR.
8. **The repo is the system of record.** Decisions, discoveries, and plans live in docs, not in chat threads.

## Database Isolation

Each repo gets its own SQLite database to prevent cross-project contamination:

- **`start` and `register`**: default DB path is `~/.mars-harness/db/{repo-name}/mars.db`
- **`serve`**: uses the shared legacy path `~/.mars-harness/db/mars.db` (designed for multi-repo orchestration)
- **`doctor --repo`**: checks the per-repo database for the specified repository
- **`--db` flag**: explicit override on any command, takes precedence over defaults

This ensures queue, telemetry, scheduling, and repo registry are physically isolated per project. See AD-029 in `docs/design-docs/dogfood-and-decisions.md`.

## Operations

Mars Harness is controllable by any AI agent via CLI commands. These are the five core operations.

### 1. Setup

First-time install: hardware detection, llama-server binary, model download.

```bash
mars-harness setup
```

Flags: `--skip-download`, `--skip-github`, `--test-mode`, `--dry-run`

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

### 5. Run

Manually execute a single agent role against a repository.

```bash
mars-harness run <role> --repo /path/to/repo
mars-harness run engineer --repo . --dry-run   # preview system prompt
```

Flags: `--model-endpoint`, `--trace`, `--dry-run`, `--budget`, `--max-turns`

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
2. **Document every decision.** Architecture decisions go in `docs/design-docs/`. Discoveries go in the relevant design doc's Discoveries section.
3. **Quality is not a phase.** Tests are written alongside code. Every milestone has a quality gate.
4. **Feed conversations back.** Decisions that exist only in chat threads are invisible to future agents. See the delivery schedule for what to build next.

## Pointers

- **Delivery schedule:** [docs/exec-plans/active/delivery-schedule.md](docs/exec-plans/active/delivery-schedule.md)
- **Tenets:** [docs/design-docs/tenets.md](docs/design-docs/tenets.md)
- **Architecture decisions:** [docs/design-docs/index.md](docs/design-docs/index.md)
- **Product vision:** [docs/product-specs/vision.md](docs/product-specs/vision.md)
- **Model research:** [docs/references/model-landscape-april-2026.md](docs/references/model-landscape-april-2026.md)
- **Tech debt:** [docs/exec-plans/tech-debt.md](docs/exec-plans/tech-debt.md)
- **Tickets:** [docs/tickets/](docs/tickets/)
