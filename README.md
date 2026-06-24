# Mars Harness

Mars Harness is a self-hosted autonomous AI delivery system. Install one Go
command, let it prepare local inference, point it at a separate git checkout,
and it gives that target repo a repo-owned agent operating system: roles,
guardrails, tickets, BDD feature contracts, execution plans, scoring, traces,
and release discipline.

The default path is local-first and strict-trunk. Open-weight models run on
your hardware through llama.cpp by default, optional integrations are explicit,
and the repository remains the system of record.

## Mental Model

There are three related things:

- **mars-harness**: this source repo and the installed `mars-harness` command.
- **Deployed harness**: the `.harness/`, `AGENTS.md`, docs, tickets, and release
  files that `mars-harness init`, `start`, or `upgrade` writes into a target
  repo.
- **Target project**: the app or service Mars Harness is building, testing, and
  managing. The harness operates on target projects; its agents do not treat
  this source repo as an ordinary target.

For the full product contract, start with the
[product vision](docs/product-specs/vision.md) and
[product surface](docs/product-specs/product-surface.md).

## Harness Explainer

For a high-level adoption walkthrough, use the interactive
[Mars Harness explainer](https://greaveselliott.github.io/software-factory/).
It explains the software-factory problem, the value case, the foundation vs.
deployed harness boundary, design decisions, operating model, DocSync,
telemetry, safety, and a pilot adoption path for mixed leadership and
engineering audiences.

The source lives in
[docs/harness-ecosystem/](docs/harness-ecosystem/). The
repository GitHub Pages site publishes from `/docs`, with
[docs/index.html](docs/index.html) redirecting the site root to the explainer.

## Current Status

Mars Harness is under active development. The supported first path today is a
source checkout install, local setup, and a scoped target run.

- **Ready default path**: install from source, run setup, run `doctor`, and use
  `mars-harness start --repo <target>` to scaffold, register, reconcile, and run
  one target repository.
- **Implemented and still hardening**: local inference lifecycle, generated
  target harness lifecycle, `run`/`start`/`serve`, embedded dashboard,
  release/update workflows, scoring, trust, telemetry, tools, and MCP.
- **Optional or planned surfaces**: GitHub release/auth workflows, board-driven
  JIRA intake, and the future TanStack dashboard control plane. The current
  dashboard is the embedded Go, htmx, Chart.js, and SSE surface.

The active source plan is tracked in
[docs/exec-plans/active/current-operating-plan.md](docs/exec-plans/active/current-operating-plan.md).

## Requirements

- macOS or Linux
- Git
- Go 1.22+ for source installs
- Network access for first setup downloads
- Disk space for multi-GB GGUF model files under `~/.mars-harness/models`
- Recommended GPU: Apple Silicon/Metal, NVIDIA CUDA, or AMD ROCm

CPU fallback exists for development and dry runs, but ordinary autonomous
operation is designed for a GPU-backed local model.

## Quick Start

```bash
# Install the command from this source checkout.
git clone https://github.com/greaveselliott/software-factory.git
cd software-factory
make install

# Download local inference assets and verify the machine.
mars-harness setup --skip-github
mars-harness doctor

# Bootstrap and run a target repository.
mars-harness start --repo /path/to/target-repo
```

`start` initializes a missing target harness, registers the repo, reconciles
existing lifecycle state, and runs the autonomous loop for that repo.

If you want to inspect before running agents:

```bash
mars-harness init --repo /path/to/target-repo
mars-harness run engineer --repo /path/to/target-repo --dry-run
```

See [docs/quickstart.md](docs/quickstart.md) for the detailed walkthrough,
common flags, PATH repair, binary updater notes, and release examples.

## What It Creates

In a target repo, Mars Harness creates or fills in a deployed harness:

- `AGENTS.md` as the compact first-read map for agents and humans.
- `.harness/manifest.yaml`, role prompts, guardrails, skills, and knowledge
  routes.
- `docs/goals/`, `docs/features/`, `docs/exec-plans/`, and `docs/tickets/` for
  strategy, BDD contracts, active plans, and work items.
- `docs/QUALITY_SCORE.md`, `VERSION`, `CHANGELOG.md`, design docs, references,
  and generated guidance needed by the operating model.

`upgrade` fills missing defaults without overwriting user-owned target
configuration. `eject` previews and then, only with explicit confirmation,
removes the deployed harness files and the associated per-repo SQLite database.

## First-Run Safety

Mars Harness is built around blast-radius containment:

- `run --dry-run` previews assembled role context without calling the model.
- `doctor --repo <path> --json` checks target health and drift.
- Each registered repo gets isolated SQLite state under
  `~/.mars-harness/db/{repo-name}/mars.db` unless `--db` overrides it.
- `start` and `serve` expose interactive controls for pause, resume, restart,
  scan, stop, and role runs; the default web dashboard is served locally.
- `eject` is dry-run by default:

```bash
mars-harness eject --repo /path/to/target-repo
mars-harness eject --repo /path/to/target-repo --apply --confirm target-repo
```

The apply path removes generated harness artifacts and the associated database;
it does not rewrite git history.

## Operating Model

Mars Harness turns product intent into small, verifiable trunk commits:

- Goals and one active execution plan define priority.
- BDD feature contracts in `docs/features/` define business behavior and done.
- Tickets scope the next walking-skeleton slice.
- Roles implement, review, validate, release, and maintain with explicit tools,
  guardrails, trust levels, and scores.
- Evidence lives in the repo: commits, tests, traces, quality score, validation
  reports, release notes, and design decisions.

For the deeper model, read
[docs/features/README.md](docs/features/README.md),
[docs/design-docs/delivery-operating-model.md](docs/design-docs/delivery-operating-model.md),
and [docs/roles/ROLES.md](docs/roles/ROLES.md).

## Common Commands

```bash
# Target lifecycle and execution
mars-harness start --repo /path/to/repo
mars-harness run engineer --repo /path/to/repo --dry-run
mars-harness upgrade --repo /path/to/repo
mars-harness eject --repo /path/to/repo

# Health, drift, and updates
mars-harness doctor --repo /path/to/repo --json
mars-harness update check --repo /path/to/repo --json
mars-harness update harness --repo /path/to/repo

# Tool and MCP surfaces for agents or external clients
mars-harness tools list --json
mars-harness tools run git_status --repo /path/to/repo --args-json '{}'
mars-harness mcp serve --repo /path/to/repo --trust observer

# Multi-repo daemon path
mars-harness register --repo /path/to/repo --remote owner/repo
mars-harness serve --addr :9091 --concurrency 2
```

Use `--trust contributor` for MCP only when the connected client should be able
to call mutating tools.

## Install, Update, And Release Notes

`make install` installs the current checkout exactly as-is. It uses `go install`
and then runs shell PATH setup through the installed binary so new terminals can
resolve `mars-harness`.

Avoid `go build ./cmd/mars-harness; ./mars-harness ...`: the semicolon can run
a stale old binary if the build fails, and the source-tree binary is easy to
confuse with the installed command.

Update a source checkout install from a clean worktree with:

```bash
make update-tool
```

Binary release assets and GitHub Release mirrors are optional compatibility
surfaces. If you use them, configure release auth once:

```bash
mars-harness auth github setup
mars-harness auth github check
mars-harness update tool
```

Mars Harness and initialized target repos use semantic versions and generated
patch notes:

```bash
mars-harness release notes --repo . --bump auto
mars-harness release backfill-notes --repo . --check
```

Source releases are tagged as `vX.Y.Z`, published locally with
`mars-harness release publish-assets`, and verified with
`mars-harness release verify-assets`. See
[docs/design-docs/release-versioning.md](docs/design-docs/release-versioning.md)
for the full contract.

## Optional Integrations

Mars Harness is useful without remote services. Optional integration surfaces
include:

- GitHub auth and release/update helpers.
- GitHub/webhook coordination where configured.
- Board-driven JIRA intake through `.harness/integrations.yaml`; this is
  default-off and currently under active staged delivery.
- MCP exposure for Codex, Cursor, Claude, and other compatible clients.

Board-driven details live in
[docs/design-docs/board-driven-integrations.md](docs/design-docs/board-driven-integrations.md)
and the [Atlassian MCP runbook](docs/runbooks/atlassian-mcp-jira-intake.md).

## Contributor Notes

When changing this source repo, start with [AGENTS.md](AGENTS.md) and the
[foundation-maintainer role packet](docs/roles/personas/foundation-maintainer.md).
Source work follows remote-trunk freshness, no-stale-docs, BDD evidence,
release notes, local asset publication, and source validation rules.

Docs-only README work does not require clean-project lifecycle validation, but
runtime, generated-target, dashboard, model/provider, release/update, scoring,
safety, or orchestration changes do. See
[docs/validation/README.md](docs/validation/README.md) and
[docs/design-docs/foundation-operating-model.md](docs/design-docs/foundation-operating-model.md).

## Documentation Map

- [docs/quickstart.md](docs/quickstart.md): detailed install and first run.
- [docs/harness-ecosystem/](docs/harness-ecosystem/):
  interactive adoption explainer published to GitHub Pages.
- [ARCHITECTURE.md](ARCHITECTURE.md): system architecture.
- [docs/product-specs/index.md](docs/product-specs/index.md): product contract.
- [docs/features/README.md](docs/features/README.md): BDD feature contracts.
- [docs/bundle-reference.md](docs/bundle-reference.md): target harness format.
- [docs/guardrails-guide.md](docs/guardrails-guide.md): safety rule authoring.
- [docs/design-docs/index.md](docs/design-docs/index.md): decisions and design
  rationale.
- [docs/QUALITY_SCORE.md](docs/QUALITY_SCORE.md): repo-visible quality evidence.

## Lineage

Mars Harness is an evolution of the
[Mars](https://github.com/elliottgreaves/mars) monorepo's automation pipeline.
Mars proved the model with 11 autonomous roles running a full development
lifecycle via Cursor Automations; Mars Harness extracts that into a standalone,
self-hosted system for any target repo.

## License

Apache 2.0. See [LICENSE](LICENSE).
