# Mars Harness

A self-hosted autonomous AI delivery system. Run it on your own hardware with open models. No cloud API costs, no data exfiltration, no vendor lock-in.

You provide a machine with a GPU. Mars Harness autonomously manages your development pipeline: CI diagnosis and repair, code generation from tickets, trunk checks and review, release management, documentation maintenance. All inference runs locally.

**Status:** Under active development. See the [current operating plan](docs/exec-plans/active/current-operating-plan.md).

## The Nine Tenets

1. **Plug and Play** — zero to running in one command, extends to full lifecycle
2. **Self-Improving System** — evolves from human interventions and its own failures
3. **Accuracy and Value Scoring** — per-role health scores from real outcomes
4. **Customisable Guardrails** — user-defined rules enforced during execution
5. **Roadmap from Init** — tickets and backlog deployed on day one
6. **Blast Radius Containment** — never cause irreversible damage
7. **Execution Truth and Transparency** — auditable, attributable, everything in git
8. **Progressive Autonomy** — earn trust, graduate from observer to autonomous
9. **Context Efficiency** — minimal context assembly, retrieval over stuffing

Full text: [docs/design-docs/tenets.md](docs/design-docs/tenets.md)

## System Requirements

- macOS or Linux
- Git
- Go 1.22+ for source installs
- Network access for first setup downloads
- Disk space for multi-GB GGUF model files under `~/.mars-harness/models`
- Recommended GPU: Apple Silicon/Metal, NVIDIA CUDA, or AMD ROCm

CPU fallback exists for development and dry runs, but the normal autonomous
workflow is designed for a GPU-backed local model.

## Quick Start

```bash
# Clone and install from source
git clone https://github.com/greaveselliott/mars-harness.git
cd mars-harness
make install

# Download local inference assets and verify the machine
mars-harness setup --skip-github
mars-harness doctor

# Prepare a target repo and preview an agent run
mars-harness init --repo /path/to/your/repo
mars-harness run engineer --repo /path/to/your/repo --dry-run

# Start the full autonomous pipeline
mars-harness start --repo /path/to/your/repo
```

`make install` installs the current checkout exactly as-is. It uses `go install`
and then runs shell PATH setup through the installed binary so new terminals can
resolve `mars-harness`.

Avoid `go build ./cmd/mars-harness; ./mars-harness ...`: the semicolon can run
a stale old binary if the build fails, and the source-tree binary is easy to
confuse with the installed command.

## Local Models And Setup

`mars-harness setup --skip-github` is the source-checkout first-run path. It:

- creates `~/.mars-harness/`
- writes default config
- detects hardware
- installs the pinned `llama-server` binary
- downloads pinned GGUF model files into `~/.mars-harness/models`
- configures shell PATH for the installed command

Use `--skip-download` only when compatible model files are already present. Use
`--test-mode` for dry/local setup paths that intentionally avoid downloads and
external services.

## Updating From A Clone

Run this from the Mars Harness source checkout:

```bash
make update-tool
```

`make update-tool` fast-forwards the checkout from `origin/main` when the
worktree is clean, installs the updated command with `go install`, refreshes
shell PATH setup, and prints the installed version. If you have local changes,
commit or stash them first, or run `make install` to install the current checkout
without pulling.

Update generated harness files in a target repo with:

```bash
mars-harness update check --repo /path/to/target-repo
mars-harness update harness --repo /path/to/target-repo
```

## Optional Binary Releases

The source checkout path above is the supported path for anyone pulling this
repo today. Binary release assets and GitHub Release mirrors remain available
for compatibility, but they are not required for source onboarding.

If you use the binary updater, run `mars-harness auth github setup` once so
private release-asset workflows can reuse GitHub CLI or token auth. Source users
can ignore that step and use `make update-tool` instead.

Remove Mars Harness from a target repo with a dry-run kill switch:

```bash
mars-harness eject --repo /path/to/target-repo
mars-harness eject --repo /path/to/target-repo --apply --confirm sample-target
```

The apply path removes generated harness artifacts and the associated per-repo
SQLite database, but does not rewrite git history.

Generate semantic-versioned patch notes from commits:

```bash
mars-harness release notes --repo . --bump auto
```

For changes to this source repo and repos initialized by Mars Harness, that release command is part of the commit flow: every non-release semantic commit is followed by a generated `release: notes X.Y.Z` commit before `main` is pushed.
For source releases, push tag `vX.Y.Z` at the release-note commit, publish
local assets with `mars-harness release publish-assets --repo . --version
vX.Y.Z --upload auto`, then verify the local dist with `mars-harness release
verify-assets --dist dist/releases --version vX.Y.Z`. When GitHub release
credentials are configured, the same command may mirror those assets to GitHub
Releases as an optional distribution surface.

## Lineage

Mars Harness is an evolution of the [Mars](https://github.com/elliottgreaves/mars) monorepo's automation pipeline. Mars proved the model works — 11 autonomous roles running a full development lifecycle via Cursor Automations. This product extracts that into a standalone system that runs on your own hardware against any repo.

## Documentation

- [AGENTS.md](AGENTS.md) — project guide for AI agents working in this repo
- [ARCHITECTURE.md](ARCHITECTURE.md) — system architecture
- [docs/design-docs/](docs/design-docs/) — architectural decisions
- [docs/exec-plans/](docs/exec-plans/) — delivery plans and trackers
- [docs/product-specs/index.md](docs/product-specs/index.md) — living product specs
- [docs/references/](docs/references/) — research findings and external sources

## License

Apache 2.0 — see [LICENSE](LICENSE).
